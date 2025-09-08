package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sirupsen/logrus"

	"github.com/HaiFongPan/r2s3-cli/internal/config"
	"github.com/HaiFongPan/r2s3-cli/internal/r2"
	"github.com/HaiFongPan/r2s3-cli/internal/utils"
)

// FileItem represents a file in the browser
type FileItem struct {
	Key          string
	Size         int64
	LastModified time.Time
	ContentType  string
	Category     string
}

// KeyMap defines keybindings for the file browser
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding
	Refresh  key.Binding
	Delete   key.Binding
	Download key.Binding
	Preview  key.Binding
	Help     key.Binding
	Quit     key.Binding
	Confirm  key.Binding
	Cancel   key.Binding
}

// DefaultKeyMap returns default keybindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("home/g", "go to start"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("end/G", "go to end"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r", "f5"),
			key.WithHelp("r/f5", "refresh"),
		),
		Delete: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "delete"),
		),
		Download: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "download"),
		),
		Preview: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "preview URL"),
		),
		Help: key.NewBinding(
			key.WithKeys("?", "h"),
			key.WithHelp("?/h", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q/esc", "quit"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "yes"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "no"),
		),
	}
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown},
		{k.Home, k.End, k.Refresh},
		{k.Download, k.Preview, k.Delete},
		{k.Help, k.Quit},
	}
}

// FileBrowserModel represents the file browser TUI model
type FileBrowserModel struct {
	files            []FileItem
	cursor           int
	viewport         int
	viewportHeight   int
	loading          bool
	error            error
	client           *r2.Client
	config           *config.Config
	bucketName       string
	prefix           string
	showHelp         bool
	confirmDelete    bool
	deleteTarget     string
	infoMessage      string
	windowWidth      int
	windowHeight     int
	previewURL       string
	urlGenerator     *utils.URLGenerator
	fileDownloader   *utils.FileDownloader
	downloading      bool
	downloadingFile  string
	downloadProgress progress.Model
	downloadCancel   context.CancelFunc
	fileTable        table.Model
	keyMap           KeyMap
	help             help.Model
	spinner          spinner.Model
	helpViewport     viewport.Model
	program          *tea.Program
}

// NewFileBrowserModel creates a new file browser model
func NewFileBrowserModel(client *r2.Client, cfg *config.Config, bucketName, prefix string) *FileBrowserModel {
	urlGenerator := utils.NewURLGenerator(client.GetS3Client(), &cfg.R2, bucketName)
	fileDownloader := utils.NewFileDownloader(client.GetS3Client(), bucketName)

	// Initialize table with proper column configuration
	columns := []table.Column{
		{Title: "📄 NAME", Width: 45},
		{Title: "📊 SIZE", Width: 10},
		{Title: "🏷️ TYPE", Width: 8},
		{Title: "🕒 MODIFIED", Width: 16},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithHeight(20),
		table.WithFocused(true),
		table.WithStyles(table.Styles{
			Header: lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#00FFFF")).
				BorderBottom(true).
				Bold(true).
				Foreground(lipgloss.Color("#00FFFF")).
				Background(lipgloss.Color("#1a1a1a")),
			Selected: lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#4A90E2")).
				Bold(true),
			Cell: lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")),
		}),
	)

	// Initialize spinner for loading states
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00"))

	// Initialize help
	h := help.New()
	h.ShowAll = false

	// Initialize help viewport
	vp := viewport.New(60, 15)
	vp.Style = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FFEB3B")).
		Padding(1, 2)

	return &FileBrowserModel{
		client:         client,
		config:         cfg,
		bucketName:     bucketName,
		prefix:         prefix,
		viewportHeight: 20,
		loading:        true,
		windowWidth:    80,
		windowHeight:   24,
		urlGenerator:   urlGenerator,
		fileDownloader: fileDownloader,
		fileTable:    t,
		keyMap:       DefaultKeyMap(),
		help:         h,
		spinner:      s,
		helpViewport: vp,
	}
}

// SetProgram sets the tea.Program reference for direct message sending
func (m *FileBrowserModel) SetProgram(p *tea.Program) {
	m.program = p
}

// Init implements the bubbletea.Model interface
func (m *FileBrowserModel) Init() tea.Cmd {
	return tea.Batch(m.loadFiles(), m.spinner.Tick)
}

// Update implements the bubbletea.Model interface
func (m *FileBrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirmDelete {
			return m.handleDeleteConfirmation(msg)
		}
		return m.handleNavigation(msg)

	case filesLoadedMsg:
		m.loading = false
		m.files = msg.files
		m.error = msg.err
		if m.error == nil {
			m.updateTable()
		}
		return m, nil

	case deleteCompletedMsg:
		m.confirmDelete = false
		if msg.err != nil {
			m.error = msg.err
		} else {
			// Reload files after successful deletion
			m.loading = true
			return m, m.loadFiles()
		}
		return m, nil

	case previewURLGeneratedMsg:
		if msg.err != nil {
			m.infoMessage = fmt.Sprintf("Failed to generate preview URL: %v", msg.err)
		} else {
			m.previewURL = msg.url
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		m.viewportHeight = msg.Height - 10 // Reserve space for header/footer

		// Update table size and column widths
		leftPanelWidth := int(float64(msg.Width)*0.6) - 2 // 60% minus separator
		m.updateTableSize(leftPanelWidth, m.viewportHeight)

		// Update help viewport size
		m.helpViewport.Width = min(60, msg.Width-10)
		m.helpViewport.Height = min(15, msg.Height-10)
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case utils.DownloadStartedMsg:
		logrus.Infof("Update: handling direct DownloadStartedMsg for file: %s", msg.Filename)
		m.downloading = true
		m.downloadingFile = msg.Filename
		// Initialize progress bar for new download
		m.downloadProgress = progress.New(progress.WithDefaultGradient())
		// Don't call Init() here to avoid double rendering
		return m, nil

	case utils.DownloadProgressMsg:
		cmd := m.downloadProgress.SetPercent(msg.Progress)
		return m, cmd

	case utils.DownloadCompletedMsg:
		logrus.Info("Update: handling direct DownloadCompletedMsg")
		m.downloading = false
		m.downloadingFile = ""
		m.downloadCancel = nil
		if msg.Err != nil {
			logrus.Errorf("Update: download completed with error: %v", msg.Err)
			m.infoMessage = fmt.Sprintf("Download failed: %v", msg.Err)
		} else {
			logrus.Info("Update: download completed successfully")
			m.infoMessage = "File downloaded successfully"
		}
		return m, nil

	default:
		// Handle progress bar internal messages
		var cmd tea.Cmd
		progressModel, progressCmd := m.downloadProgress.Update(msg)
		if progressModel != nil {
			if pm, ok := progressModel.(progress.Model); ok {
				m.downloadProgress = pm
			}
		}
		cmd = progressCmd

		return m, cmd
	}
}

// handleNavigation handles keyboard navigation
func (m *FileBrowserModel) handleNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Quit):
		if m.downloading {
			// Cancel download
			if m.downloadCancel != nil {
				m.downloadCancel()
			}
			m.downloading = false
			m.downloadingFile = ""
			m.downloadCancel = nil
			m.infoMessage = "Download cancelled"
			return m, nil
		}
		return m, tea.Quit

	case key.Matches(msg, m.keyMap.Up):
		if m.downloading {
			return m, nil // Block navigation during download
		}
		var cmd tea.Cmd
		m.fileTable, cmd = m.fileTable.Update(msg)
		m.cursor = m.fileTable.Cursor()
		m.infoMessage = ""   // Clear info message on navigation
		m.updateRightPanel() // Update right panel on navigation
		return m, cmd

	case key.Matches(msg, m.keyMap.Down):
		if m.downloading {
			return m, nil // Block navigation during download
		}
		var cmd tea.Cmd
		m.fileTable, cmd = m.fileTable.Update(msg)
		m.cursor = m.fileTable.Cursor()
		m.infoMessage = ""   // Clear info message on navigation
		m.updateRightPanel() // Update right panel on navigation
		return m, cmd

	case key.Matches(msg, m.keyMap.PageUp):
		if m.downloading {
			return m, nil
		}
		var cmd tea.Cmd
		m.fileTable, cmd = m.fileTable.Update(msg)
		m.cursor = m.fileTable.Cursor()
		m.infoMessage = "" // Clear info message on navigation
		m.updateRightPanel()
		return m, cmd

	case key.Matches(msg, m.keyMap.PageDown):
		if m.downloading {
			return m, nil
		}
		var cmd tea.Cmd
		m.fileTable, cmd = m.fileTable.Update(msg)
		m.cursor = m.fileTable.Cursor()
		m.infoMessage = "" // Clear info message on navigation
		m.updateRightPanel()
		return m, cmd

	case key.Matches(msg, m.keyMap.Home):
		if m.downloading {
			return m, nil
		}
		var cmd tea.Cmd
		m.fileTable, cmd = m.fileTable.Update(msg)
		m.cursor = m.fileTable.Cursor()
		m.infoMessage = "" // Clear info message on navigation
		m.updateRightPanel()
		return m, cmd

	case key.Matches(msg, m.keyMap.End):
		if m.downloading {
			return m, nil
		}
		var cmd tea.Cmd
		m.fileTable, cmd = m.fileTable.Update(msg)
		m.cursor = m.fileTable.Cursor()
		m.infoMessage = "" // Clear info message on navigation
		m.updateRightPanel()
		return m, cmd

	case key.Matches(msg, m.keyMap.Download):
		if m.downloading {
			return m, nil // Block new download during current download
		}
		if len(m.files) > 0 && m.cursor < len(m.files) {
			file := m.files[m.cursor]
			return m, m.downloadFileWithProgress(file.Key)
		}

	case key.Matches(msg, m.keyMap.Preview):
		if m.downloading {
			return m, nil
		}
		if len(m.files) > 0 && m.cursor < len(m.files) {
			file := m.files[m.cursor]
			return m, m.generatePreviewURL(file.Key)
		}

	case key.Matches(msg, m.keyMap.Delete):
		if m.downloading {
			return m, nil
		}
		if len(m.files) > 0 && m.cursor < len(m.files) {
			m.confirmDelete = true
			m.deleteTarget = m.files[m.cursor].Key
		}

	case key.Matches(msg, m.keyMap.Refresh):
		if m.downloading {
			return m, nil
		}
		m.loading = true
		m.error = nil
		m.infoMessage = "" // Clear info message on refresh
		return m, m.loadFiles()

	case key.Matches(msg, m.keyMap.Help):
		m.help.ShowAll = !m.help.ShowAll
		m.showHelp = !m.showHelp
		if m.showHelp {
			m.setupHelpViewport()
		}

	}

	return m, nil
}

// handleDeleteConfirmation handles delete confirmation dialog
func (m *FileBrowserModel) handleDeleteConfirmation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Confirm):
		m.confirmDelete = false
		m.infoMessage = fmt.Sprintf("Deleting %s...", m.deleteTarget)
		return m, m.deleteFile(m.deleteTarget)

	case key.Matches(msg, m.keyMap.Cancel) || key.Matches(msg, m.keyMap.Quit):
		m.confirmDelete = false
		m.deleteTarget = ""
	}

	return m, nil
}

// adjustViewport adjusts the viewport to show the cursor
func (m *FileBrowserModel) adjustViewport() {
	if m.cursor < m.viewport {
		m.viewport = m.cursor
	} else if m.cursor >= m.viewport+m.viewportHeight {
		m.viewport = m.cursor - m.viewportHeight + 1
	}
}

// setupHelpViewport sets up the help viewport with content
func (m *FileBrowserModel) setupHelpViewport() {
	helpContent := m.help.View(m.keyMap)
	m.helpViewport.SetContent(helpContent)
}

// View implements the bubbletea.Model interface
func (m *FileBrowserModel) View() string {
	// Calculate panel widths
	leftPanelWidth := int(float64(m.windowWidth) * 0.6)   // 60% for left panel
	rightPanelWidth := m.windowWidth - leftPanelWidth - 2 // Remaining width minus separator

	// Render header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FF80")).
		MarginBottom(1)

	header := fmt.Sprintf("R2 File Browser - %s", m.bucketName)
	if m.prefix != "" {
		header += fmt.Sprintf("/%s", m.prefix)
	}
	headerLine := headerStyle.Render(header)

	// Show loading state with spinner
	if m.loading {
		loadingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00"))
		return headerLine + "\n" + loadingStyle.Render(fmt.Sprintf("%s Loading files...", m.spinner.View()))
	}

	// Show error if any
	if m.error != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
		return headerLine + "\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.error))
	}

	// Render left panel (file list)
	leftPanel := m.renderLeftPanel(leftPanelWidth)

	// Render right panel (file info)
	rightPanel := m.renderRightPanel(rightPanelWidth)

	// Combine panels side by side
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPanel,
		lipgloss.NewStyle().Width(2).Render("  "), // Separator
		rightPanel,
	)

	// Footer with help
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		MarginTop(1)

	// Use bubbles help component for footer
	footerLine := footerStyle.Render(m.help.ShortHelpView(m.keyMap.ShortHelp()))

	baseView := headerLine + "\n" + content + "\n" + footerLine

	// Render floating dialogs on top of base view
	if m.downloading {
		return m.renderFloatingDialog(baseView, m.renderDownloadProgress())
	}

	if m.confirmDelete {
		return m.renderFloatingDialog(baseView, m.renderDeleteConfirmation())
	}

	if m.showHelp {
		return m.renderFloatingDialog(baseView, m.renderHelpDialog())
	}

	return baseView
}

// renderLeftPanel renders the left panel with file list using table component
func (m *FileBrowserModel) renderLeftPanel(width int) string {
	// Handle empty file list
	if len(m.files) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			Width(width).
			Height(m.viewportHeight).
			Align(lipgloss.Center).
			AlignVertical(lipgloss.Center)
		return emptyStyle.Render("No files found")
	}

	// Update table width if needed
	m.updateTableSize(width, m.viewportHeight)

	// Render the table
	tableView := m.fileTable.View()

	// Add file count info at the bottom if needed
	if len(m.files) > 0 {
		countStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			MarginTop(1)

		countInfo := fmt.Sprintf("Total: %d files", len(m.files))
		tableView += "\n" + countStyle.Render(countInfo)
	}

	return tableView
}

// renderRightPanel renders the right panel with file info
func (m *FileBrowserModel) renderRightPanel(width int) string {
	var b strings.Builder

	// Panel container style
	panelStyle := lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder(), false, false, false, true)

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FFFF")).
		MarginBottom(1)

	b.WriteString(titleStyle.Render("File Information"))
	b.WriteString("\n")

	// Show file info if a file is selected
	if len(m.files) > 0 && m.cursor < len(m.files) {
		file := m.files[m.cursor]

		// Basic file information
		infoStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

		// File details with emojis
		b.WriteString(infoStyle.Render(fmt.Sprintf("📄 Name: %s", file.Key)))
		b.WriteString("\n")
		b.WriteString(infoStyle.Render(fmt.Sprintf("📊 Size: %s", formatFileSize(file.Size))))
		b.WriteString("\n")
		b.WriteString(infoStyle.Render(fmt.Sprintf("%s Type: %s", m.getCategoryEmoji(file.Category), file.Category)))
		b.WriteString("\n")
		b.WriteString(infoStyle.Render(fmt.Sprintf("🏷️ Content-Type: %s", file.ContentType)))
		b.WriteString("\n")
		b.WriteString(infoStyle.Render(fmt.Sprintf("🕒 Modified: %s", file.LastModified.Format("2006-01-02 15:04:05"))))
		b.WriteString("\n\n")

		// Custom domain URL if configured
		if len(m.config.R2.CustomDomains) > 0 {
			customURL := m.urlGenerator.GenerateCustomDomainURL(file.Key)
			urlStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00FF00"))

			b.WriteString(urlStyle.Render("🔗 Custom URL:"))
			b.WriteString("\n")

			urlBoxStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00BFFF")).
				Background(lipgloss.Color("#333333")).
				Padding(0, 1).
				Margin(0, 1).
				Underline(true)

			// Make URL clickable with OSC 8 escape sequence
			clickableURL := m.makeClickableURL(customURL, customURL)

			// Wrap long URLs
			if len(customURL) > width-6 {
				for i := 0; i < len(customURL); i += width - 6 {
					end := i + width - 6
					if end > len(customURL) {
						end = len(customURL)
					}
					urlPart := customURL[i:end]
					b.WriteString(urlBoxStyle.Render(m.makeClickableURL(urlPart, customURL)))
					b.WriteString("\n")
				}
			} else {
				b.WriteString(urlBoxStyle.Render(clickableURL))
				b.WriteString("\n")
			}

			hintStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#808080")).
				Italic(true)
			b.WriteString(hintStyle.Render("💡 Tip: Click to open or select and copy"))
			b.WriteString("\n\n")
		}

		// Preview URL if generated
		if m.previewURL != "" {
			previewStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFF00"))

			b.WriteString(previewStyle.Render("⏱️ Presigned URL:"))
			b.WriteString("\n")

			urlBoxStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00BFFF")).
				Background(lipgloss.Color("#333333")).
				Padding(0, 1).
				Margin(0, 1).
				Underline(true)

			// Make URL clickable
			clickablePreviewURL := m.makeClickableURL(m.previewURL, m.previewURL)

			// Wrap long URLs
			if len(m.previewURL) > width-6 {
				for i := 0; i < len(m.previewURL); i += width - 6 {
					end := i + width - 6
					if end > len(m.previewURL) {
						end = len(m.previewURL)
					}
					urlPart := m.previewURL[i:end]
					b.WriteString(urlBoxStyle.Render(m.makeClickableURL(urlPart, m.previewURL)))
					b.WriteString("\n")
				}
			} else {
				b.WriteString(urlBoxStyle.Render(clickablePreviewURL))
				b.WriteString("\n")
			}

			hintStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#808080")).
				Italic(true)
			b.WriteString(hintStyle.Render("⏰ Valid for 1 hour • Click to open or copy"))
			b.WriteString("\n")
		}

	} else {
		// No file selected
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			Align(lipgloss.Center)

		b.WriteString(emptyStyle.Render("Select a file to view details"))
		b.WriteString("\n")
	}

	return panelStyle.Render(b.String())
}

// renderFloatingDialog renders a dialog floating over the base view while keeping base visible
func (m *FileBrowserModel) renderFloatingDialog(baseView, dialog string) string {
	// Split base view into lines to modify background
	baseLines := strings.Split(baseView, "\n")
	dimmedLines := make([]string, len(baseLines))

	// Dim the background slightly by applying a darker style
	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")) // Dim the text

	for i, line := range baseLines {
		dimmedLines[i] = dimStyle.Render(line)
	}

	dimmedBase := strings.Join(dimmedLines, "\n")

	// For now, just use simple overlay - lipgloss Place doesn't support layering easily
	// So we'll just show the dimmed background with centered dialog
	_ = dimmedBase // Mark as used

	return lipgloss.Place(
		m.windowWidth,
		m.windowHeight,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#222222")),
	)
}

// renderDownloadProgress renders the download progress dialog
func (m *FileBrowserModel) renderDownloadProgress() string {
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00FFFF")).
		Padding(2, 3).
		Width(50).
		Align(lipgloss.Center).
		Background(lipgloss.Color("#1a1a1a")).
		Foreground(lipgloss.Color("#FFFFFF"))

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FFFF")).
		Align(lipgloss.Center).
		MarginBottom(1)

	var b strings.Builder
	b.WriteString(titleStyle.Render("📥 Downloading File"))
	b.WriteString("\n\n")

	// Show filename
	filenameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Align(lipgloss.Center)
	b.WriteString(filenameStyle.Render(fmt.Sprintf("File: %s", m.downloadingFile)))
	b.WriteString("\n\n")

	// Show progress bar
	progressView := m.downloadProgress.View()
	b.WriteString(progressView)
	b.WriteString("\n\n")

	// Show instructions
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Italic(true).
		Align(lipgloss.Center)
	b.WriteString(instructionStyle.Render("Press ESC to cancel download"))

	return dialogStyle.Render(b.String())
}

// renderDeleteConfirmation renders the delete confirmation dialog
func (m *FileBrowserModel) renderDeleteConfirmation() string {
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF6B6B")).
		Padding(2, 3).
		Width(50).
		Align(lipgloss.Center).
		Background(lipgloss.Color("#1a1a1a")).
		Foreground(lipgloss.Color("#FFFFFF"))

	content := fmt.Sprintf("Delete file: %s\n\nThis action cannot be undone!\n\nPress 'y' to confirm, 'n' to cancel",
		m.deleteTarget)

	return dialogStyle.Render(content)
}

// renderHelpDialog renders the help dialog using bubbles components
func (m *FileBrowserModel) renderHelpDialog() string {
	// Create title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFEB3B")).
		Align(lipgloss.Center).
		MarginBottom(1)

	title := titleStyle.Render("🚀 R2 File Browser - Help")

	// Get help content from bubbles help component
	helpContent := m.help.FullHelpView(m.keyMap.FullHelp())

	// Create content with title and help
	content := lipgloss.JoinVertical(lipgloss.Left, title, helpContent)

	// Update viewport content
	m.helpViewport.SetContent(content)

	// Style the dialog container
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FFEB3B")).
		Padding(1).
		Width(min(70, m.windowWidth-10)).
		Background(lipgloss.Color("#1a1a1a")).
		Foreground(lipgloss.Color("#FFFFFF"))

	// Add instructions at the bottom
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Italic(true).
		Align(lipgloss.Center).
		MarginTop(1)

	instructions := instructionStyle.Render("Press ? or h to close help • Use ↑↓ to scroll")

	// Combine viewport with instructions
	dialogContent := lipgloss.JoinVertical(
		lipgloss.Left,
		m.helpViewport.View(),
		instructions,
	)

	return dialogStyle.Render(dialogContent)
}

// Message types for tea.Cmd communication
type filesLoadedMsg struct {
	files []FileItem
	err   error
}

type deleteCompletedMsg struct {
	err error
}

type previewURLGeneratedMsg struct {
	url string
	err error
}

// loadFiles loads files from R2
func (m *FileBrowserModel) loadFiles() tea.Cmd {
	return func() tea.Msg {
		files, err := m.fetchFiles()
		return filesLoadedMsg{files: files, err: err}
	}
}

// fetchFiles fetches files from R2 bucket
func (m *FileBrowserModel) fetchFiles() ([]FileItem, error) {
	s3Client := m.client.GetS3Client()

	// List objects
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(m.bucketName),
		MaxKeys: aws.Int32(1000),
	}

	if m.prefix != "" {
		input.Prefix = aws.String(m.prefix)
	}

	result, err := s3Client.ListObjectsV2(context.TODO(), input)
	if err != nil {
		return nil, err
	}

	files := make([]FileItem, 0, len(result.Contents))
	for _, obj := range result.Contents {
		contentType, err := utils.DetectContentType(aws.ToString(obj.Key), nil)
		if err != nil {
			logrus.Warnf("Failed to detect content type for %s: %v", aws.ToString(obj.Key), err)
			contentType = "application/octet-stream"
		}

		files = append(files, FileItem{
			Key:          aws.ToString(obj.Key),
			Size:         aws.ToInt64(obj.Size),
			LastModified: aws.ToTime(obj.LastModified),
			ContentType:  contentType,
			Category:     utils.GetFileCategory(contentType),
		})
	}

	return files, nil
}

// deleteFile deletes a file from R2
func (m *FileBrowserModel) deleteFile(key string) tea.Cmd {
	return func() tea.Msg {
		s3Client := m.client.GetS3Client()
		_, err := s3Client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
			Bucket: aws.String(m.bucketName),
			Key:    aws.String(key),
		})
		return deleteCompletedMsg{err: err}
	}
}

// Utility functions
func (m *FileBrowserModel) getFileColor(category string) string {
	switch category {
	case "image":
		return "#0080FF" // Blue
	case "document":
		return "#00FF00" // Green
	case "archive":
		return "#FFFF00" // Yellow
	case "video":
		return "#FF0000" // Red
	case "audio":
		return "#FF00FF" // Magenta
	case "text":
		return "#00FFFF" // Cyan
	default:
		return "#FFFFFF" // White
	}
}

func (m *FileBrowserModel) getCategoryEmoji(category string) string {
	switch category {
	case "image":
		return "🖼️"
	case "document":
		return "📝"
	case "archive":
		return "📦"
	case "video":
		return "🎬"
	case "audio":
		return "🎵"
	case "text":
		return "📄"
	default:
		return "📁"
	}
}

// makeClickableURL creates a clickable URL using OSC 8 escape sequences
// This is supported by modern terminals like iTerm2, Windows Terminal, Ghostty, etc.
func (m *FileBrowserModel) makeClickableURL(displayText, url string) string {
	// Check if we should disable OSC 8 (for debugging or compatibility)
	if m.shouldDisableOSC8() {
		return displayText // Return plain text if OSC 8 is disabled
	}

	// OSC 8 sequence: \033]8;;URL\033\\DISPLAY_TEXT\033]8;;\033\\
	// Use \007 (BEL) instead of \033\\ as terminator for better compatibility
	return fmt.Sprintf("\033]8;;%s\007%s\033]8;;\007", url, displayText)
}

// shouldDisableOSC8 checks if OSC 8 support should be disabled
func (m *FileBrowserModel) shouldDisableOSC8() bool {
	// For now, disable OSC 8 to avoid display issues
	// Can be made configurable later or add terminal detection
	return true
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// updateRightPanel updates right panel information for current file
func (m *FileBrowserModel) updateRightPanel() {
	// Clear previous preview URL when navigating
	m.previewURL = ""
}

// updateTable updates table data from files slice
func (m *FileBrowserModel) updateTable() {
	rows := make([]table.Row, len(m.files))
	for i, file := range m.files {
		name := file.Key
		if len(name) > 37 { // Leave space for "..."
			name = name[:37] + "..."
		}

		// Add emoji and color coding based on file type
		emoji := m.getCategoryEmoji(file.Category)
		coloredName := m.colorizeFileName(name, file.Category)
		nameWithEmoji := fmt.Sprintf("%s %s", emoji, coloredName)

		size := formatFileSize(file.Size)
		category := strings.ToUpper(file.Category)
		if len(category) > 6 {
			category = category[:6]
		}

		modified := file.LastModified.Format("01-02 15:04")

		rows[i] = table.Row{nameWithEmoji, size, category, modified}
	}
	m.fileTable.SetRows(rows)
}

// colorizeFileName applies color to filename based on file category
func (m *FileBrowserModel) colorizeFileName(name, category string) string {
	color := m.getFileColor(category)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	return style.Render(name)
}

// updateTableSize updates table dimensions and column widths
func (m *FileBrowserModel) updateTableSize(width, height int) {
	// Calculate optimal column widths based on available width
	totalWidth := width - 8 // Reserve space for borders and padding

	// Fixed column widths with priority to essential information
	sizeWidth := 10
	typeWidth := 8
	modifiedWidth := 16

	// NAME column gets remaining width, accounting for emoji (2 chars + space)
	nameWidth := totalWidth - sizeWidth - typeWidth - modifiedWidth
	if nameWidth < 25 { // Minimum for emoji + reasonable filename
		nameWidth = 25
	} else if nameWidth > 60 { // Maximum to prevent overly long names
		nameWidth = 60
	}

	// Update table columns
	columns := []table.Column{
		{Title: "📄 NAME", Width: nameWidth},
		{Title: "📊 SIZE", Width: sizeWidth},
		{Title: "🏷️ TYPE", Width: typeWidth},
		{Title: "🕒 MODIFIED", Width: modifiedWidth},
	}

	m.fileTable.SetColumns(columns)
	m.fileTable.SetHeight(height)
}

// generatePreviewURL generates preview URL for a file
func (m *FileBrowserModel) generatePreviewURL(key string) tea.Cmd {
	return func() tea.Msg {
		_, presignedURL, err := m.urlGenerator.GenerateFileURL(key)
		return previewURLGeneratedMsg{url: presignedURL, err: err}
	}
}

// downloadFile downloads a file to the local system
func (m *FileBrowserModel) downloadFile(key string) tea.Cmd {
	return func() tea.Msg {
		err := m.fileDownloader.DownloadFile(key)
		return utils.DownloadCompletedMsg{Err: err}
	}
}

// downloadFileWithProgress downloads a file with progress updates
func (m *FileBrowserModel) downloadFileWithProgress(key string) tea.Cmd {
	return func() tea.Msg {
		// Create context for cancellation
		ctx, cancel := context.WithCancel(context.Background())
		m.downloadCancel = cancel

		// Start download with direct message sending
		go func() {
			logrus.Info("downloadFileWithProgress: starting download with direct messaging")

			// Start download with progress callback
			err := m.fileDownloader.DownloadFileWithProgressCallback(ctx, key, func(msg tea.Msg) {
				if m.program != nil {
					m.program.Send(msg)
				}
			})

			// Send completion message
			if m.program != nil {
				m.program.Send(utils.DownloadCompletedMsg{Err: err})
			}

			logrus.Info("downloadFileWithProgress: download finished")
		}()

		// Return started message immediately for synchronous handling
		return utils.DownloadStartedMsg{Filename: filepath.Base(key)}
	}
}
