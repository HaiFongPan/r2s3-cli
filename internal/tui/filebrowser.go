package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sirupsen/logrus"

	"github.com/HaiFongPan/r2s3-cli/internal/config"
	"github.com/HaiFongPan/r2s3-cli/internal/r2"
	tuiconfig "github.com/HaiFongPan/r2s3-cli/internal/tui/config"
	"github.com/HaiFongPan/r2s3-cli/internal/tui/image"
	"github.com/HaiFongPan/r2s3-cli/internal/tui/messaging"
	"github.com/HaiFongPan/r2s3-cli/internal/tui/theme"
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
	Up           key.Binding
	Down         key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	Home         key.Binding
	End          key.Binding
	Refresh      key.Binding
	Delete       key.Binding
	Download     key.Binding
	Preview      key.Binding
	Search       key.Binding
	Upload       key.Binding
	ClearSearch  key.Binding
	ChangeBucket key.Binding
	NextPage     key.Binding
	PrevPage     key.Binding
	ToggleImage  key.Binding
	ForcePreview key.Binding
	Help         key.Binding
	Quit         key.Binding
	Confirm      key.Binding
	Cancel       key.Binding
	CopyCustom   key.Binding
	CopyPresign  key.Binding
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
			key.WithKeys("v"),
			key.WithHelp("v", "preview URL"),
		),
		Search: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "search"),
		),
		Upload: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "upload"),
		),
		ClearSearch: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "clear search"),
		),
		ChangeBucket: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "change bucket"),
		),
		NextPage: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next page"),
		),
		PrevPage: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "prev page"),
		),
		ToggleImage: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "preview image"),
		),
		ForcePreview: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "force preview"),
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
		CopyCustom: key.NewBinding(
			key.WithKeys("ctrl+o"),
			key.WithHelp("ctrl+o", "copy custom URL"),
		),
		CopyPresign: key.NewBinding(
			key.WithKeys("ctrl+y"),
			key.WithHelp("ctrl+y", "copy presigned URL"),
		),
	}
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Search, k.Upload, k.ChangeBucket, k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown},
		{k.Home, k.End, k.Refresh},
		{k.Download, k.Preview, k.Delete},
		{k.Search, k.Upload, k.ClearSearch},
		{k.CopyCustom, k.CopyPresign},
		{k.ChangeBucket},
		{k.NextPage, k.PrevPage, k.ToggleImage, k.ForcePreview},
		{k.Help, k.Quit},
	}
}

// InputMode represents different input modes
type InputMode int

const (
	InputModeNone InputMode = iota
	InputModeSearch
	InputModeUpload
)

// InputComponentMode represents different input component types
type InputComponentMode int

const (
	InputComponentText InputComponentMode = iota
	InputComponentFilePicker
)

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

	// Bucket selector overlay
	showingBucketSelector bool
	bucketSelector        *BucketSelectorModel

	// Pagination
	currentPage         int
	hasNextPage         bool
	continuationToken   string
	paginationLoading   bool // Loading state for pagination (different from initial loading)
	estimatedTotalPages int  // Estimated total pages (updated as we navigate)

	// Input states
	showInput          bool
	inputMode          InputMode
	inputComponentMode InputComponentMode
	textInput          textinput.Model
	filePicker         filepicker.Model
	inputPrompt        string

	// Message system
	messageManager messaging.StatusManager

	// Search state
	searchQuery  string
	isSearchMode bool

	// Upload state
	uploadProgress progress.Model
	uploading      bool
	uploadingFile  string
	fileUploader   utils.FileUploader

	// Delete state
	deleting     bool
	deletingFile string

	// Image preview state
	imageManager        *image.ImageManager
	imagePreview        *image.ImagePreview
	isImagePreviewing   bool
	imageSpinner        spinner.Model
	imageLoadingFile    string
	currentPreviewForce bool
	imageAreaCols       int
	imageAreaRows       int

	// Preview modal state
	showingPreview   bool
	previewModal     *ImagePreviewModel
	lastPreviewFile  *FileItem
	lastPreviewForce bool
}

// createFilePicker creates a properly configured file picker
func createFilePicker() filepicker.Model {
	fp := filepicker.New()

	// Set the current directory to a sensible default
	if cwd, err := os.Getwd(); err == nil {
		fp.CurrentDirectory = cwd
	} else {
		// Fallback to home directory
		if homeDir, err := os.UserHomeDir(); err == nil {
			fp.CurrentDirectory = homeDir
		} else {
			fp.CurrentDirectory = "/"
		}
	}

	// Allow all file types
	fp.AllowedTypes = []string{}
	fp.ShowHidden = false
	fp.Height = 15 // Set height to show more files

	// Enable directory navigation features
	fp.DirAllowed = true
	fp.FileAllowed = true

	return fp
}

// NewFileBrowserModel creates a new file browser model
func NewFileBrowserModel(client *r2.Client, cfg *config.Config, bucketName, prefix string) *FileBrowserModel {
	urlGenerator := utils.NewURLGenerator(client.GetS3Client().(*s3.Client), cfg, bucketName)
	fileDownloader := utils.NewFileDownloader(client.GetS3Client().(*s3.Client), bucketName)
	fileUploader := utils.NewFileUploader(client, cfg, bucketName)

	// Initialize table with proper column configuration
	columns := []table.Column{
		{Title: "NAME", Width: tuiconfig.DefaultColumnNameWidth},
		{Title: "SIZE", Width: tuiconfig.DefaultColumnSizeWidth},
		{Title: "TYPE", Width: tuiconfig.DefaultColumnTypeWidth},
		{Title: "MODIFIED", Width: tuiconfig.DefaultColumnModifiedWidth},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithHeight(tuiconfig.DefaultTableHeight),
		table.WithFocused(true),
		table.WithStyles(table.Styles{
			Header: lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(lipgloss.Color(theme.ColorBrightBlue)).
				Foreground(lipgloss.Color(theme.ColorBrightCyan)).
				Bold(true),
			Selected: lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.ColorText)).
				Reverse(true).
				Bold(true),
			Cell: lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.ColorText)),
		}),
	)

	// Initialize spinner for loading states
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorBrightYellow))

	// Initialize help
	h := help.New()
	h.ShowAll = false

	// Initialize help viewport
	vp := viewport.New(60, 15)
	vp.Style = lipgloss.NewStyle().
		Border(theme.BorderStyleUnified).
		BorderForeground(lipgloss.Color(theme.ColorBrightBlue)).
		Padding(1, 2)

	m := &FileBrowserModel{
		client:              client,
		config:              cfg,
		bucketName:          bucketName,
		prefix:              prefix,
		viewportHeight:      20,
		loading:             true,
		windowWidth:         80,
		windowHeight:        24,
		urlGenerator:        urlGenerator,
		fileDownloader:      fileDownloader,
		fileUploader:        fileUploader,
		fileTable:           t,
		keyMap:              DefaultKeyMap(),
		help:                h,
		spinner:             s,
		helpViewport:        vp,
		currentPage:         1,
		hasNextPage:         false,
		estimatedTotalPages: 1,

		// Input states
		showInput:          false,
		inputMode:          InputModeNone,
		inputComponentMode: InputComponentText,
		textInput:          textinput.New(),
		filePicker:         createFilePicker(),

		// Message system
		messageManager: messaging.NewStatusManager(),

		// Search state
		searchQuery:  "",
		isSearchMode: false,

		// Upload state
		uploadProgress: progress.New(progress.WithDefaultGradient()),
		uploading:      false,
		uploadingFile:  "",

		// Delete state
		deleting:     false,
		deletingFile: "",

		// Image preview state
		imageManager:        image.NewImageManager("/tmp/r2s3-cli-cache", 100*1024*1024), // 100MB cache
		imagePreview:        nil,
		isImagePreviewing:   false,
		imageSpinner:        spinner.New(),
		imageLoadingFile:    "",
		currentPreviewForce: false,
		lastPreviewFile:     nil,
		lastPreviewForce:    false,
	}

	// Configure text input
	m.textInput.Placeholder = "Type here..."
	m.textInput.Focus()
	m.textInput.CharLimit = 200
	m.textInput.Width = 40

	// Configure file picker
	m.filePicker.AllowedTypes = []string{} // Allow all file types
	m.filePicker.ShowPermissions = false
	m.filePicker.ShowSize = true
	m.filePicker.ShowHidden = false

	// Configure image spinner
	m.imageSpinner.Spinner = spinner.Dot
	m.imageSpinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorBrightYellow))

	// Configure image manager with S3 client
	if s3Client, ok := client.GetS3Client().(*s3.Client); ok {
		m.imageManager.SetDownloaderClient(s3Client)
		m.imageManager.SetBucketName(bucketName)
	}
	// 在 TUI 中启用安全的文本模式渲染，避免控制序列破坏 UI
	m.imageManager.SetUseTextRender(true)
	// Set current directory to user's home directory
	if homeDir, err := os.UserHomeDir(); err == nil {
		m.filePicker.CurrentDirectory = homeDir
	} else {
		// Fallback to current working directory
		if cwd, err := os.Getwd(); err == nil {
			m.filePicker.CurrentDirectory = cwd
		}
	}

	return m
}

// SetProgram sets the tea.Program reference for direct message sending
func (m *FileBrowserModel) SetProgram(p *tea.Program) {
	m.program = p
}

// Init implements the bubbletea.Model interface
func (m *FileBrowserModel) Init() tea.Cmd {
	return tea.Batch(m.loadFiles(), m.spinner.Tick, m.imageSpinner.Tick)
}

// Update implements the bubbletea.Model interface
func (m *FileBrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If bucket selector is showing, forward non-key messages to it first
	if m.showingBucketSelector && m.bucketSelector != nil {
		if _, isKeyMsg := msg.(tea.KeyMsg); !isKeyMsg {
			selectorModel, cmd := m.bucketSelector.Update(msg)
			if selectorModel != nil {
				m.bucketSelector = selectorModel.(*BucketSelectorModel)
			}
			// If bucket selector has a command, execute it and also process the message in file browser
			if cmd != nil {
				return m, cmd
			}
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If preview modal is showing, route keys to it first
		if m.showingPreview && m.previewModal != nil {
			// Let modal handle closure keys
			newModal, cmd := m.previewModal.Update(msg)
			if im, ok := newModal.(*ImagePreviewModel); ok {
				m.previewModal = im
			}
			return m, cmd
		}
		// If bucket selector is showing, handle its keys
		if m.showingBucketSelector && m.bucketSelector != nil {
			// Check for ESC key to close bucket selector
			if msg.String() == "esc" || msg.String() == "q" {
				return m, func() tea.Msg { return bucketSelectorClosedMsg{} }
			}
			// Forward other keys to bucket selector
			selectorModel, cmd := m.bucketSelector.Update(msg)
			if selectorModel != nil {
				m.bucketSelector = selectorModel.(*BucketSelectorModel)
			}
			return m, cmd
		}

		if m.confirmDelete {
			return m.handleDeleteConfirmation(msg)
		}

		// Handle input popup
		if m.showInput {
			return m.handleInputPopup(msg)
		}

		return m.handleNavigation(msg)

	case filesLoadedMsg:
		m.loading = false
		m.paginationLoading = false
		m.files = msg.files
		m.hasNextPage = msg.hasNext
		m.continuationToken = msg.nextToken
		m.error = msg.err
		m.clearInlinePreview()

		// Update estimated total pages
		if msg.hasNext {
			// If there are more pages, estimate at least currentPage + 1
			if m.currentPage >= m.estimatedTotalPages {
				m.estimatedTotalPages = m.currentPage + 1
			}
		} else {
			// If this is the last page, we know the exact total
			m.estimatedTotalPages = m.currentPage
		}

		if m.error == nil {
			m.updateTable()
		}
		return m, nil

	case deleteCompletedMsg:
		m.confirmDelete = false
		m.deleting = false
		m.deletingFile = ""
		if msg.err != nil {
			m.setMessage(theme.FormatErrorMessage("Delete", msg.err), messaging.MessageError)
		} else {
			m.setMessage(theme.FormatSuccessMessage("deleted", m.deleteTarget), messaging.MessageSuccess)
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

	case imagePreviewCompletedMsg:
		// Image preview finished (success or error)
		m.imageLoadingFile = ""
		if msg.err != nil {
			m.imagePreview = nil
			m.isImagePreviewing = false
			m.currentPreviewForce = false
			m.setMessage(fmt.Sprintf("Image preview failed: %v", msg.err), messaging.MessageError)
		} else {
			m.imagePreview = msg.preview
			if msg.preview != nil {
				m.isImagePreviewing = true
			}
		}
		return m, nil

	case imagePreviewClearedMsg:
		// No-op placeholder to trigger rerender
		return m, nil

	case bucketSelectorClosedMsg:
		// Handle bucket selector being closed without selection
		m.showingBucketSelector = false
		m.bucketSelector = nil
		return m, nil

	case bucketSwitchedMsg:
		// Handle bucket switch from bucket selector
		m.bucketName = msg.bucket

		// Update URLGenerator and FileDownloader with new bucket
		m.urlGenerator.SetBucketName(msg.bucket)
		m.fileDownloader.SetBucketName(msg.bucket)

		m.showingBucketSelector = false
		m.bucketSelector = nil
		m.loading = true
		return m, m.loadFiles()

	case mainBucketSetMsg:
		// Handle main bucket being set from bucket selector
		if msg.err == nil {
			// Switch to the main bucket and reload files
			m.bucketName = msg.bucket

			// Update URLGenerator and FileDownloader with new bucket
			m.urlGenerator.SetBucketName(msg.bucket)
			m.fileDownloader.SetBucketName(msg.bucket)

			m.showingBucketSelector = false
			m.bucketSelector = nil
			m.loading = true
			return m, m.loadFiles()
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
		// Update image preview area size (right panel interior)
		rightPanelWidth := msg.Width - leftPanelWidth - 2
		infoRows := 10
		if infoRows > m.viewportHeight-3 {
			infoRows = max(3, m.viewportHeight/3)
		}
		m.imageAreaCols = max(1, rightPanelWidth-4)
		m.imageAreaRows = max(1, m.viewportHeight-infoRows-2)
		// 文本模式按字符网格渲染
		m.imageManager.SetCellSize(m.imageAreaCols, m.imageAreaRows)
		// Update modal size if open
		if m.showingPreview && m.previewModal != nil {
			m.previewModal.width = msg.Width
			m.previewModal.height = msg.Height
		}
		return m, nil

	case spinner.TickMsg:
		// Route spinner ticks to preview modal first if open
		if m.showingPreview && m.previewModal != nil {
			newModal, cmd := m.previewModal.Update(msg)
			if im, ok := newModal.(*ImagePreviewModel); ok {
				m.previewModal = im
			}
			return m, cmd
		}
		var cmds []tea.Cmd
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		m.imageSpinner, cmd = m.imageSpinner.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

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

	case uploadProgressMsg:
		// Update progress bar
		if m.uploading {
			cmd := m.uploadProgress.SetPercent(msg.percentage / 100.0)
			return m, cmd
		}
		return m, nil

	case uploadCompletedMsg:
		m.uploading = false
		m.uploadingFile = ""
		if msg.err != nil {
			m.setMessage(theme.FormatErrorMessage("Upload", msg.err), messaging.MessageError)
		} else {
			m.setMessage(theme.FormatSuccessMessage("uploaded", msg.file), messaging.MessageSuccess)
			// Reset upload state to prevent cached paths
			m.resetUploadState()
			// Refresh files to show the new upload
			return m, m.loadFiles()
		}
		// Reset upload state even on failure to clear cached paths
		m.resetUploadState()
		return m, nil

	case modalClosedMsg:
		m.showingPreview = false
		m.previewModal = nil
		m.imageManager.SetUseTextRender(true)
		m.imageManager.SetCellSize(m.imageAreaCols, m.imageAreaRows)
		m.clearInlinePreview()
		m.lastPreviewFile = nil
		m.lastPreviewForce = false
		return m, nil

	default:
		// Handle component internal messages
		var cmds []tea.Cmd

		// Handle progress bar internal messages
		progressModel, progressCmd := m.downloadProgress.Update(msg)
		if progressModel != nil {
			if pm, ok := progressModel.(progress.Model); ok {
				m.downloadProgress = pm
			}
		}
		if progressCmd != nil {
			cmds = append(cmds, progressCmd)
		}

		// Handle file picker internal messages if showing input with file picker
		if m.showInput && m.inputComponentMode == InputComponentFilePicker && m.inputMode == InputModeUpload {
			filePickerModel, filePickerCmd := m.filePicker.Update(msg)

			// Check if a file was selected
			if filePickerModel.Path != "" && filePickerModel.Path != m.filePicker.Path {
				// File was selected, process it
				m.showInput = false
				m.inputMode = InputModeNone
				m.inputComponentMode = InputComponentText
				selectedPath := filePickerModel.Path
				m.filePicker = filePickerModel
				return m.processUploadWithPath(selectedPath)
			}

			m.filePicker = filePickerModel
			if filePickerCmd != nil {
				cmds = append(cmds, filePickerCmd)
			}
		}

		return m, tea.Batch(cmds...)
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
		if m.downloading || m.deleting {
			return m, nil // Block navigation during download/delete
		}
		var cmd tea.Cmd
		m.fileTable, cmd = m.fileTable.Update(msg)
		m.cursor = m.fileTable.Cursor()
		m.infoMessage = "" // Clear info message on navigation
		m.clearMessage()   // Clear status message on navigation
		m.clearInlinePreview()
		m.updateRightPanel() // Update right panel on navigation
		return m, cmd

	case key.Matches(msg, m.keyMap.Down):
		if m.downloading || m.deleting {
			return m, nil // Block navigation during download/delete
		}
		var cmd tea.Cmd
		m.fileTable, cmd = m.fileTable.Update(msg)
		m.cursor = m.fileTable.Cursor()
		m.infoMessage = "" // Clear info message on navigation
		m.clearMessage()   // Clear status message on navigation
		m.clearInlinePreview()
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
		m.clearInlinePreview()
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
		m.clearInlinePreview()
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
		m.clearInlinePreview()
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
		m.clearInlinePreview()
		m.updateRightPanel()
		return m, cmd

	case key.Matches(msg, m.keyMap.Download):
		if m.downloading || m.deleting {
			return m, nil // Block new download during current download/delete
		}
		if len(m.files) > 0 && m.cursor < len(m.files) {
			file := m.files[m.cursor]
			return m, m.downloadFileWithProgress(file.Key)
		}

	case key.Matches(msg, m.keyMap.Preview):
		if m.downloading || m.deleting {
			return m, nil
		}
		if len(m.files) > 0 && m.cursor < len(m.files) {
			file := m.files[m.cursor]
			return m, m.generatePreviewURL(file.Key)
		}

	case key.Matches(msg, m.keyMap.Search):
		if m.downloading || m.uploading || m.deleting {
			return m, nil
		}
		m.showInput = true
		m.inputMode = InputModeSearch
		m.inputComponentMode = InputComponentText
		m.inputPrompt = "Search objects:"
		m.textInput.SetValue("")
		m.textInput.Placeholder = "Enter search query..."
		m.textInput.Focus()
		return m, nil

	case key.Matches(msg, m.keyMap.Upload):
		if m.downloading || m.uploading || m.deleting {
			return m, nil
		}

		// Ensure clean upload state before starting new upload
		m.resetUploadState()

		// Set up upload dialog
		m.showInput = true
		m.inputMode = InputModeUpload
		m.inputComponentMode = InputComponentText // Default to text input
		m.inputPrompt = "Upload file path:"
		m.textInput.SetValue("")
		m.textInput.Placeholder = "Enter file path... (Tab: file picker)"
		m.textInput.Focus()

		logrus.Info("Starting new upload dialog with clean state")
		return m, nil

	case key.Matches(msg, m.keyMap.ClearSearch):
		if m.isSearchMode {
			m.clearSearch()
			m.setMessage("Search cleared", messaging.MessageInfo)
			return m, m.loadFiles()
		}
		return m, nil

	case key.Matches(msg, m.keyMap.Delete):
		if m.downloading || m.deleting {
			return m, nil
		}
		if len(m.files) > 0 && m.cursor < len(m.files) {
			m.confirmDelete = true
			m.deleteTarget = m.files[m.cursor].Key
		}

	case key.Matches(msg, m.keyMap.ChangeBucket):
		if m.downloading || m.deleting {
			return m, nil
		}
		return m, m.openBucketSelector()

	case key.Matches(msg, m.keyMap.NextPage):
		if m.downloading || m.deleting || m.paginationLoading {
			return m, nil
		}
		if m.hasNextPage {
			m.paginationLoading = true
			m.cursor = 0
			m.currentPage++
			m.clearInlinePreview()
			return m, m.loadFiles()
		}

	case key.Matches(msg, m.keyMap.PrevPage):
		if m.downloading || m.deleting || m.paginationLoading {
			return m, nil
		}
		if m.currentPage > 1 {
			m.paginationLoading = true
			m.cursor = 0
			m.currentPage--
			m.continuationToken = "" // Reset to load from beginning
			m.clearInlinePreview()
			// For previous page, we need to implement page stack or reload from start
			// For simplicity, let's just reload from the first page and navigate
			return m, m.loadFromPage(m.currentPage)
		}

	case key.Matches(msg, m.keyMap.ToggleImage):
		return m.startPreviewModal(false)

	case key.Matches(msg, m.keyMap.ForcePreview):
		return m.startPreviewModal(true)

	case key.Matches(msg, m.keyMap.Refresh):
		if m.downloading || m.deleting {
			return m, nil
		}
		m.loading = true
		m.error = nil
		m.currentPage = 1
		m.continuationToken = ""
		m.estimatedTotalPages = 1
		m.infoMessage = "" // Clear info message on refresh
		return m, m.loadFiles()

	case key.Matches(msg, m.keyMap.CopyCustom):
		if len(m.files) > 0 && m.fileTable.Cursor() < len(m.files) {
			file := m.files[m.fileTable.Cursor()]
			customURL := m.urlGenerator.GenerateCustomDomainURL(file.Key)
			if customURL != "" {
				utils.CopyToClipboard(customURL)
				m.setMessage("Custom URL copied to clipboard", messaging.MessageInfo)
			} else {
				m.setMessage("No custom domain configured for this bucket", messaging.MessageError)
			}
		}

	case key.Matches(msg, m.keyMap.CopyPresign):
		if len(m.files) > 0 && m.fileTable.Cursor() < len(m.files) {
			file := m.files[m.fileTable.Cursor()]
			_, presignedURL, err := m.urlGenerator.GenerateFileURL(file.Key)
			if err != nil {
				m.setMessage(fmt.Sprintf("Failed to generate presigned URL: %s", err), messaging.MessageError)
			} else {
				utils.CopyToClipboard(presignedURL)
				m.setMessage("Presigned URL copied to clipboard", messaging.MessageInfo)
			}
		}

	case key.Matches(msg, m.keyMap.Help):
		logrus.Debugf("Help key pressed - before: ShowAll=%v, showHelp=%v", m.help.ShowAll, m.showHelp)
		m.showHelp = !m.showHelp
		logrus.Debugf("Help key pressed - after: ShowAll=%v, showHelp=%v", m.help.ShowAll, m.showHelp)
		if m.showHelp {
			m.setupHelpViewport()
		}

	}

	return m, nil
}

func (m *FileBrowserModel) startPreviewModal(force bool) (tea.Model, tea.Cmd) {
	if m.downloading || m.deleting {
		return m, nil
	}
	if len(m.files) == 0 || m.cursor >= len(m.files) {
		return m, nil
	}
	file := m.files[m.cursor]
	if !m.imageManager.IsImageFile(file.ContentType) {
		m.setMessage("Not an image file", messaging.MessageInfo)
		return m, nil
	}

	fileCopy := file
	m.lastPreviewFile = &fileCopy
	m.lastPreviewForce = force
	m.isImagePreviewing = false
	m.imagePreview = nil
	m.imageLoadingFile = ""
	m.currentPreviewForce = force

	logrus.WithFields(logrus.Fields{
		"file":  file.Key,
		"force": force,
	}).Info("opening image preview")

	m.showingPreview = true
	m.previewModal = NewImagePreviewModel(m.imageManager, file, m.windowWidth, m.windowHeight, force)

	if force {
		m.setMessage("Refreshing preview from R2", messaging.MessageInfo)
	}

	return m, m.previewModal.Init()
}

func (m *FileBrowserModel) clearInlinePreview() {
	m.isImagePreviewing = false
	m.imagePreview = nil
	m.imageLoadingFile = ""
	m.currentPreviewForce = false
}

// handleDeleteConfirmation handles delete confirmation dialog
func (m *FileBrowserModel) handleDeleteConfirmation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Confirm):
		m.confirmDelete = false
		m.deleting = true
		m.deletingFile = filepath.Base(m.deleteTarget)
		m.setMessage(theme.FormatProgressMessage("Deleting", m.deletingFile, -1), messaging.MessageWarning)
		return m, m.deleteFile(m.deleteTarget)

	case key.Matches(msg, m.keyMap.Cancel) || key.Matches(msg, m.keyMap.Quit):
		m.confirmDelete = false
		m.deleteTarget = ""
	}

	return m, nil
}

// setupHelpViewport sets up the help viewport with content
func (m *FileBrowserModel) setupHelpViewport() {
	helpContent := m.help.View(m.keyMap)
	m.helpViewport.SetContent(helpContent)
}

// View implements the bubbletea.Model interface
func (m *FileBrowserModel) View() string {
	// Calculate panel widths
	leftPanelWidth := int(float64(m.windowWidth) * tuiconfig.LeftPanelWidthRatio) // 60% for left panel
	rightPanelWidth := m.windowWidth - leftPanelWidth - 2                         // Remaining width minus separator

	// Render header with consistent styling and left alignment with panel
	headerStyle := theme.CreateHeaderStyle()

	header := fmt.Sprintf("R2 File Browser - %s", m.bucketName)
	if m.prefix != "" {
		header += fmt.Sprintf("/%s", m.prefix)
	}
	if m.isSearchMode && m.searchQuery != "" {
		header += fmt.Sprintf(" [Search: '%s'] (l: clear)", m.searchQuery)
	}
	headerLine := headerStyle.Render(header)

	// Show loading state with spinner
	if m.loading {
		loadingStyle := theme.CreateLoadingStyle()
		return headerLine + "\n" + loadingStyle.Render(fmt.Sprintf("%s Loading files...", m.spinner.View()))
	}

	// Show error if any
	if m.error != nil {
		errorStyle := theme.CreateErrorStyle()
		return headerLine + "\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.error))
	}

	// Render left panel (file list)
	leftPanel := m.renderLeftPanel(leftPanelWidth)

	// Render right panel (file info)
	rightPanel := m.renderRightPanel(rightPanelWidth)

	// Create elegant separator between panels - let lipgloss handle alignment automatically
	// separator := lipgloss.NewStyle().
	// 	Width(3).
	// 	Foreground(lipgloss.Color(ColorBrightBlue)).
	// 	Align(lipgloss.Center).
	// 	Render("│")

	// Combine panels side by side with improved separator
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPanel,
		// separator,
		rightPanel,
	)

	// Footer with help and status messages
	footerStyle := theme.CreateFooterStyle()

	// Use bubbles help component for footer (always show short help)
	helpLine := m.help.ShortHelpView(m.keyMap.ShortHelp())
	logrus.Debugf("Help footer debug - ShowAll: %v, helpLine length: %d, content: '%s'",
		m.help.ShowAll, len(helpLine), helpLine)

	// Add status message if present
	var footerContent string
	if m.messageManager.HasMessage() {
		messageLine := m.messageManager.RenderMessage()
		footerContent = lipgloss.JoinVertical(lipgloss.Left, helpLine, messageLine)
		logrus.Debugf("Footer content debug - showing help + message, final length: %d", len(footerContent))
	} else {
		footerContent = helpLine
		logrus.Debugf("Footer content debug - showing help only, final length: %d, content: '%.100s'", len(footerContent), footerContent)
	}

	footerLine := footerStyle.Render(footerContent)
	logrus.Debugf("Final footer line length: %d", len(footerLine))

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

	if m.showingPreview && m.previewModal != nil {
		// Show modal as full-screen content to avoid extra centering
		return m.previewModal.View()
	}

	if m.showingBucketSelector && m.bucketSelector != nil {
		return m.renderFloatingDialog(baseView, m.bucketSelector.View())
	}

	if m.showInput {
		return m.renderFloatingDialog(baseView, m.renderInputPopup())
	}

	return baseView
}

// renderLeftPanel renders the left panel with file list using table component
func (m *FileBrowserModel) renderLeftPanel(width int) string {
	// Create unified panel style
	panelWidth := width - tuiconfig.DefaultViewportPadding // Account for border
	panelHeight := m.viewportHeight

	// Handle empty file list
	if len(m.files) == 0 {
		emptyContent := theme.CreateInfoTextStyle().
			Align(lipgloss.Center).
			Width(panelWidth - tuiconfig.DefaultViewportPadding).
			Height(panelHeight - 4).
			AlignVertical(lipgloss.Center).
			Render("No files found")

		return theme.CreateUnifiedPanelStyle(panelWidth, panelHeight).Render(emptyContent)
	}

	// Update table width for content area (minus padding)
	m.updateTableSize(panelWidth-tuiconfig.DefaultViewportPadding, panelHeight-4)

	// Render the table
	tableView := m.fileTable.View()

	// Add file count info at the bottom if needed
	if len(m.files) > 0 {
		countStyle := theme.CreateSecondaryTextStyle().MarginTop(tuiconfig.DefaultMarginSize)

		countInfo := fmt.Sprintf("Total: %d files", len(m.files))

		// Add pagination info
		var pageInfo string
		if m.hasNextPage {
			pageInfo = fmt.Sprintf(" | Page %d/%d+", m.currentPage, m.estimatedTotalPages)
		} else {
			pageInfo = fmt.Sprintf(" | Page %d/%d", m.currentPage, m.estimatedTotalPages)
		}

		if m.currentPage > 1 {
			pageInfo += " (b: prev)"
		}
		if m.hasNextPage {
			pageInfo += " (n: next)"
		}

		// Add loading spinner for pagination
		if m.paginationLoading {
			pageInfo += fmt.Sprintf(" %s", m.spinner.View())
		}

		countInfo += pageInfo
		tableView += "\n" + countStyle.Render(countInfo)
	}

	return theme.CreateUnifiedPanelStyle(panelWidth, panelHeight).Render(tableView)
}

// renderRightPanel renders the right panel with file info
func (m *FileBrowserModel) renderRightPanel(width int) string {
	var content strings.Builder

	// Panel dimensions
	panelWidth := width - tuiconfig.DefaultViewportPadding // Account for border
	panelHeight := m.viewportHeight

	// Title
	titleStyle := theme.CreateSectionHeaderStyle()
	content.WriteString(titleStyle.Render("📋 File Information"))
	content.WriteString("\n")

	// Show file info if a file is selected
	if len(m.files) > 0 && m.cursor < len(m.files) {
		file := m.files[m.cursor]

		// Basic file information section
		infoStyle := theme.CreateInfoTextStyle()

		// File details with emojis
		content.WriteString(infoStyle.Render(fmt.Sprintf("📄 Name: %s", file.Key)))
		content.WriteString("\n")
		content.WriteString(infoStyle.Render(fmt.Sprintf("📊 Size: %s", formatFileSize(file.Size))))
		content.WriteString("\n")
		content.WriteString(infoStyle.Render(fmt.Sprintf("%s Type: %s", m.getCategoryEmoji(file.Category), file.Category)))
		content.WriteString("\n")
		content.WriteString(infoStyle.Render(fmt.Sprintf("🏷️ Content-Type: %s", file.ContentType)))
		content.WriteString("\n")
		content.WriteString(infoStyle.Render(fmt.Sprintf("🕒 Modified: %s", file.LastModified.Format("2006-01-02 15:04:05"))))
		content.WriteString("\n\n")

		// Custom domain URL section if configured
		customURL := m.urlGenerator.GenerateCustomDomainURL(file.Key)
		if customURL != "" {
			urlSectionStyle := theme.CreateURLSectionStyle()

			content.WriteString(urlSectionStyle.Render("🔗 Custom URL:"))
			content.WriteString("\n")

			// Make URL clickable with OSC 8 escape sequence
			clickableURL := m.makeClickableURL(customURL, customURL)

			// Display URL with terminal-compatible colors
			coloredURL := theme.FormatClickableURL(clickableURL, customURL)
			content.WriteString(coloredURL)
			content.WriteString("\n")

			hintStyle := theme.CreateHintStyle()
			content.WriteString(hintStyle.Render("💡 Click to open or use Ctrl+O to copy, use v to generate Presigned URL"))
			content.WriteString("\n\n")
		}

		// Preview URL section if generated
		if m.previewURL != "" {
			previewSectionStyle := theme.CreatePreviewURLSectionStyle()

			content.WriteString(previewSectionStyle.Render("⏱️ Presigned URL:"))
			content.WriteString("\n")

			// Use hyperlink format for long URLs - display filename as clickable link
			filename := filepath.Base(file.Key)
			linkText := fmt.Sprintf("📥 [Click to download %s]", filename)
			clickablePreviewURL := m.makeClickableURL(linkText, m.previewURL)

			// Display hyperlink with terminal-compatible colors
			coloredURL := theme.FormatClickableURL(clickablePreviewURL, m.previewURL)
			content.WriteString(coloredURL)
			content.WriteString("\n")

			hintStyle := theme.CreateHintStyle()
			content.WriteString(hintStyle.Render("⏰ Valid for 1 hour • Click to open or use Ctrl+Y to copy"))
			content.WriteString("\n")
		}

	} else {
		// No file selected
		emptyStyle := theme.CreateInfoTextStyle().
			Align(lipgloss.Center)

		content.WriteString(emptyStyle.Render("Select a file to view details"))
		content.WriteString("\n")
	}

	// 附加预览（在同一个带边框的面板内容里），保证整体高度受控
	if m.isImagePreviewing && len(m.files) > 0 && m.cursor < len(m.files) {
		file := m.files[m.cursor]
		content.WriteString("\n")
		content.WriteString(titleStyle.Render("🖼 Preview"))
		content.WriteString("\n")
		if m.imageLoadingFile == file.Key && m.imagePreview == nil {
			loadingStyle := theme.CreateLoadingStyle()
			content.WriteString(loadingStyle.Render(fmt.Sprintf("%s Loading image preview...", m.imageSpinner.View())))
			content.WriteString("\n")
		}
		if m.imagePreview != nil && m.imagePreview.FileKey == file.Key {
			// 显示来源提示
			hintStyle := theme.CreateHintStyle()
			source := "Preview downloaded from R2"
			if m.imagePreview.CacheHit {
				source = "Preview served from cache"
			} else if m.currentPreviewForce {
				source = "Preview refreshed from R2"
			}
			content.WriteString(hintStyle.Render(source))
			content.WriteString("\n")
			// 预览是 ANSI 文本，直接放入内容中
			content.WriteString(m.imagePreview.RenderedData)
			content.WriteString("\x1b[0m")
		}
	}

	return theme.CreateUnifiedPanelStyle(panelWidth, panelHeight).Render(content.String())
}

// renderFloatingDialog renders a dialog floating over the base view while keeping base visible
func (m *FileBrowserModel) renderFloatingDialog(baseView, dialog string) string {
	// Split base view into lines to modify background
	baseLines := strings.Split(baseView, "\n")
	dimmedLines := make([]string, len(baseLines))

	// Dim the background slightly by applying a darker style
	dimStyle := theme.CreateOverlayStyle()

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
	dialogStyle := theme.CreateDialogStyle(tuiconfig.DialogDefaultWidth, theme.ColorBrightCyan)

	titleStyle := theme.CreateSectionHeaderStyle().
		Align(lipgloss.Center).
		MarginBottom(1)

	var b strings.Builder
	b.WriteString(titleStyle.Render("📥 Downloading File"))
	b.WriteString("\n\n")

	// Show filename
	filenameStyle := theme.CreateInfoTextStyle().Align(lipgloss.Center)
	b.WriteString(filenameStyle.Render(fmt.Sprintf("File: %s", m.downloadingFile)))
	b.WriteString("\n\n")

	// Show progress bar
	progressView := m.downloadProgress.View()
	b.WriteString(progressView)
	b.WriteString("\n\n")

	// Show instructions
	instructionStyle := theme.CreateSecondaryTextStyle().Align(lipgloss.Center)
	b.WriteString(instructionStyle.Render("Press ESC to cancel download"))

	return dialogStyle.Render(b.String())
}

// renderDeleteConfirmation renders the delete confirmation dialog
func (m *FileBrowserModel) renderDeleteConfirmation() string {
	dialogStyle := theme.CreateDialogStyle(tuiconfig.DialogDefaultWidth, theme.ColorBrightRed)

	content := fmt.Sprintf("Delete file: %s\n\nThis action cannot be undone!\n\nPress 'y' to confirm, 'n' to cancel",
		m.deleteTarget)

	return dialogStyle.Render(content)
}

// renderHelpDialog renders the help dialog using bubbles components
func (m *FileBrowserModel) renderHelpDialog() string {
	// Create title
	titleStyle := theme.CreateSectionHeaderStyle().
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
	dialogWidth := min(tuiconfig.DialogLargeWidth, m.windowWidth-10)
	dialogStyle := theme.CreateDialogStyle(dialogWidth, theme.ColorBrightCyan).
		Padding(1)

	// Add instructions at the bottom
	instructionStyle := theme.CreateSecondaryTextStyle().
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

// renderInputPopup renders the input popup dialog
func (m *FileBrowserModel) renderInputPopup() string {
	// Create title based on input mode
	titleStyle := theme.CreateSectionHeaderStyle().
		Align(lipgloss.Center).
		MarginBottom(1)

	var title string
	switch m.inputMode {
	case InputModeSearch:
		title = titleStyle.Render("🔍 Search Objects")
	case InputModeUpload:
		if m.inputComponentMode == InputComponentFilePicker {
			title = titleStyle.Render("📤 Upload File (File Picker)")
		} else {
			title = titleStyle.Render("📤 Upload File (Text Input)")
		}
	default:
		title = titleStyle.Render("Input")
	}

	// Create prompt
	promptStyle := theme.CreatePromptStyle().
		MarginBottom(1)

	prompt := promptStyle.Render(m.inputPrompt)

	// Create input field based on component mode
	var inputField string
	if m.inputComponentMode == InputComponentFilePicker && m.inputMode == InputModeUpload {
		// Render file picker
		inputField = m.filePicker.View()
	} else {
		// Render text input
		inputField = m.textInput.View()
	}

	// Create instructions
	instructionStyle := theme.CreateSecondaryTextStyle().
		Align(lipgloss.Center).
		MarginTop(1)

	var instructions string
	if m.inputMode == InputModeUpload {
		if m.inputComponentMode == InputComponentFilePicker {
			instructions = instructionStyle.Render("[Enter] Select • [Tab] Text Input • [Esc] Cancel")
		} else {
			instructions = instructionStyle.Render("[Enter] Confirm • [Tab] File Picker • [Esc] Cancel")
		}
	} else {
		instructions = instructionStyle.Render("[Enter] Confirm • [Esc] Cancel")
	}

	// Combine all elements
	dialogContent := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		prompt,
		inputField,
		instructions,
	)

	// Style the dialog container with appropriate size
	dialogWidth := min(60, m.windowWidth-10)
	dialogHeight := 0 // Let it auto-size by default

	if m.inputComponentMode == InputComponentFilePicker && m.inputMode == InputModeUpload {
		// Make dialog larger for file picker
		dialogWidth = min(tuiconfig.FilePickerDialogWidth, m.windowWidth-5)
		dialogHeight = min(20, m.windowHeight-10) // Set explicit height for file picker
	}

	dialogStyle := theme.CreateDialogStyle(dialogWidth, theme.ColorBrightCyan).
		Padding(1)

	if dialogHeight > 0 {
		dialogStyle = dialogStyle.Height(dialogHeight)
	}

	return dialogStyle.Render(dialogContent)
}

// Message types for tea.Cmd communication
type filesLoadedMsg struct {
	files     []FileItem
	hasNext   bool
	nextToken string
	err       error
}

type deleteCompletedMsg struct {
	err error
}

type previewURLGeneratedMsg struct {
	url string
	err error
}

type bucketSelectorClosedMsg struct{}

// Image preview related messages
type imagePreviewStartedMsg struct {
	fileKey string
}

type imagePreviewCompletedMsg struct {
	preview *image.ImagePreview
	err     error
}

type imagePreviewClearedMsg struct{}

// previewImageCmd triggers image preview generation via ImageManager
func (m *FileBrowserModel) previewImageCmd(file FileItem, force bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		// Map to image.FileItem for the image manager
		item := image.FileItem{
			Key:          file.Key,
			Size:         file.Size,
			LastModified: file.LastModified,
			ContentType:  file.ContentType,
			Category:     file.Category,
		}
		var (
			preview *image.ImagePreview
			err     error
		)
		if force {
			preview, err = m.imageManager.PreviewImageForce(ctx, file.Key, item)
		} else {
			preview, err = m.imageManager.PreviewImage(ctx, file.Key, item)
		}
		return imagePreviewCompletedMsg{preview: preview, err: err}
	}
}

// loadFiles loads files from R2
func (m *FileBrowserModel) loadFiles() tea.Cmd {
	return func() tea.Msg {
		files, hasNext, nextToken, err := m.fetchFiles(m.continuationToken)
		return filesLoadedMsg{files: files, hasNext: hasNext, nextToken: nextToken, err: err}
	}
}

// loadFromPage loads files from a specific page (used for previous page navigation)
func (m *FileBrowserModel) loadFromPage(targetPage int) tea.Cmd {
	return func() tea.Msg {
		// For previous page navigation, we need to reload from the beginning
		// and skip to the target page. This is less efficient but simpler to implement
		var continuationToken string
		var currentPage int = 1

		for currentPage < targetPage {
			files, hasNext, nextToken, err := m.fetchFiles(continuationToken)
			if err != nil {
				return filesLoadedMsg{files: nil, hasNext: false, nextToken: "", err: err}
			}

			if !hasNext {
				// We've reached the end before our target page
				return filesLoadedMsg{files: files, hasNext: hasNext, nextToken: nextToken, err: nil}
			}

			continuationToken = nextToken
			currentPage++
		}

		// Now load the target page
		files, hasNext, nextToken, err := m.fetchFiles(continuationToken)
		return filesLoadedMsg{files: files, hasNext: hasNext, nextToken: nextToken, err: err}
	}
}

// fetchFiles fetches files from R2 bucket
func (m *FileBrowserModel) fetchFiles(continuationToken string) ([]FileItem, bool, string, error) {
	return m.fetchFilesWithQuery(continuationToken, m.searchQuery)
}

func (m *FileBrowserModel) fetchFilesWithQuery(continuationToken, searchQuery string) ([]FileItem, bool, string, error) {
	s3Client := m.client.GetS3Client().(*s3.Client)

	// Use configured page size
	pageSize := int32(m.config.UI.PageSize)

	// List objects
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(m.bucketName),
		MaxKeys: aws.Int32(pageSize),
	}

	// Combine original prefix with search query
	var prefix string
	if m.prefix != "" && searchQuery != "" {
		// Combine original prefix with search query
		prefix = m.prefix + searchQuery
	} else if m.prefix != "" {
		prefix = m.prefix
	} else if searchQuery != "" {
		prefix = searchQuery
	}

	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	if continuationToken != "" {
		input.ContinuationToken = aws.String(continuationToken)
	}

	result, err := s3Client.ListObjectsV2(context.TODO(), input)
	if err != nil {
		return nil, false, "", err
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

	// Check if there are more results
	hasNext := aws.ToBool(result.IsTruncated)
	nextToken := aws.ToString(result.NextContinuationToken)

	return files, hasNext, nextToken, nil
}

// deleteFile deletes a file from R2
func (m *FileBrowserModel) deleteFile(key string) tea.Cmd {
	return func() tea.Msg {
		s3Client := m.client.GetS3Client().(*s3.Client)
		_, err := s3Client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
			Bucket: aws.String(m.bucketName),
			Key:    aws.String(key),
		})
		return deleteCompletedMsg{err: err}
	}
}

// Utility functions

func (m *FileBrowserModel) getCategoryEmoji(category string) string {
	switch category {
	case "image":
		return "🖼️"
	case "document":
		return "📝"
	case "spreadsheet":
		return "📊"
	case "presentation":
		return "📽️"
	case "archive":
		return "📦"
	case "video":
		return "🎬"
	case "audio":
		return "🎵"
	case "text":
		return "📄"
	case "code":
		return "💻"
	case "data":
		return "🗂️"
	case "font":
		return "🔤"
	default:
		return "📁"
	}
}

func (m *FileBrowserModel) getFormattedCategory(category string) string {
	switch category {
	case "image":
		return "Image"
	case "document":
		return "Document"
	case "spreadsheet":
		return "Spreadsheet"
	case "presentation":
		return "Slides"
	case "archive":
		return "Archive"
	case "video":
		return "Video"
	case "audio":
		return "Audio"
	case "text":
		return "Text"
	case "code":
		return "Code"
	case "data":
		return "Data"
	case "font":
		return "Font"
	default:
		return "Other"
	}
}

// makeClickableURL creates a clickable URL using OSC 8 escape sequences
// This is supported by modern terminals like iTerm2, Windows Terminal, Ghostty, etc.
func (m *FileBrowserModel) makeClickableURL(displayText, url string) string {
	// Check if we should disable OSC 8 (for debugging or compatibility)
	if m.shouldDisableOSC8() {
		return displayText // Return plain text if OSC 8 is disabled
	}

	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, displayText)
}

// shouldDisableOSC8 checks if OSC 8 support should be disabled
func (m *FileBrowserModel) shouldDisableOSC8() bool {
	return false
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
		if len(name) > tuiconfig.FileNameTruncateLength { // Leave space for "..."
			name = name[:tuiconfig.FileNameTruncateLength] + "..."
		}

		// Apply color to filename only (remove emoji to fix encoding issues)
		color := theme.GetFileColor(file.Category)
		coloredName := lipgloss.NewStyle().
			Foreground(lipgloss.Color(color)).
			Render(name)

		size := formatFileSize(file.Size)
		// Better formatted category names
		category := m.getFormattedCategory(file.Category)
		if len(category) > tuiconfig.CategoryTruncateLength {
			category = category[:tuiconfig.CategoryTruncateLength]
		}

		modified := file.LastModified.Format("01-02 15:04")

		rows[i] = table.Row{coloredName, size, category, modified}
	}
	m.fileTable.SetRows(rows)
}

// updateTableSize updates table dimensions and column widths
func (m *FileBrowserModel) updateTableSize(width, height int) {
	// Calculate optimal column widths based on available width
	totalWidth := width - 8 // Reserve space for borders and padding

	// Fixed column widths with priority to essential information
	sizeWidth := tuiconfig.DefaultColumnSizeWidth
	typeWidth := tuiconfig.DefaultColumnTypeWidth
	modifiedWidth := tuiconfig.DefaultColumnModifiedWidth

	// NAME column gets remaining width
	nameWidth := totalWidth - sizeWidth - typeWidth - modifiedWidth
	const minNameWidth = 25 // Minimum for reasonable filename
	const maxNameWidth = 60 // Maximum to prevent overly long names
	if nameWidth < minNameWidth {
		nameWidth = minNameWidth
	} else if nameWidth > maxNameWidth {
		nameWidth = maxNameWidth
	}

	// Update table columns
	columns := []table.Column{
		{Title: "NAME", Width: nameWidth},
		{Title: "SIZE", Width: sizeWidth},
		{Title: "TYPE", Width: typeWidth},
		{Title: "MODIFIED", Width: modifiedWidth},
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

// openBucketSelector opens the bucket selector as an overlay
func (m *FileBrowserModel) openBucketSelector() tea.Cmd {
	// Create bucket selector model
	m.bucketSelector = NewBucketSelectorModel(m.client, m.config)
	m.showingBucketSelector = true

	// Send window size to bucket selector and start loading
	windowSizeMsg := tea.WindowSizeMsg{Width: m.windowWidth, Height: m.windowHeight}
	m.bucketSelector.Update(windowSizeMsg)

	// Return command to load buckets
	return m.bucketSelector.Init()
}

// handleInputPopup handles input popup interactions
func (m *FileBrowserModel) handleInputPopup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		// Cancel input and reset upload state
		logrus.Info("Upload dialog cancelled, resetting state")
		m.resetUploadState()
		m.clearMessage()
		return m, nil

	case "tab":
		// Switch between text input and file picker (for upload mode only)
		if m.inputMode == InputModeUpload {
			if m.inputComponentMode == InputComponentText {
				// Before switching, update file picker directory based on current text input
				m.updateFilePickerFromTextInputSync()
				logrus.Infof("Switching to file picker, current directory: '%s'", m.filePicker.CurrentDirectory)

				m.inputComponentMode = InputComponentFilePicker
				m.textInput.Blur()

				// Reinitialize file picker to refresh its contents without recreating it
				return m, m.filePicker.Init()
			} else {
				logrus.Infof("Switching to text input, file picker path: '%s'", m.filePicker.Path)
				m.inputComponentMode = InputComponentText
				m.textInput.Focus()
				// If a file was selected in picker, update text input with that path
				if m.filePicker.Path != "" {
					m.textInput.SetValue(m.filePicker.Path)
				}
			}
		}
		return m, nil

	case "enter":
		// Handle different component modes
		if m.inputComponentMode == InputComponentFilePicker && m.inputMode == InputModeUpload {
			// Handle file picker selection
			oldPath := m.filePicker.Path
			updatedFilePicker, cmd := m.filePicker.Update(msg)
			m.filePicker = updatedFilePicker

			logrus.Infof("FilePicker enter key: oldPath='%s', newPath='%s'", oldPath, m.filePicker.Path)

			// Check if a file was selected (path changed or is now set)
			if m.filePicker.Path != "" && m.filePicker.Path != oldPath {
				selectedPath := m.filePicker.Path
				logrus.Infof("File selected in picker: '%s'", selectedPath)
				m.showInput = false
				m.inputMode = InputModeNone
				m.inputComponentMode = InputComponentText
				return m.processUploadWithPath(selectedPath)
			}

			// Also check if this was a file selection by checking if we have a non-empty path
			if m.filePicker.Path != "" {
				selectedPath := m.filePicker.Path
				logrus.Infof("File path available in picker: '%s'", selectedPath)
				m.showInput = false
				m.inputMode = InputModeNone
				m.inputComponentMode = InputComponentText
				return m.processUploadWithPath(selectedPath)
			}

			return m, cmd
		} else {
			// Handle text input
			switch m.inputMode {
			case InputModeSearch:
				return m.processSearchInput()
			case InputModeUpload:
				return m.processUploadInput()
			}
		}
		m.showInput = false
		m.inputMode = InputModeNone
		m.inputComponentMode = InputComponentText
		return m, nil
	}

	// Handle component-specific inputs
	var cmd tea.Cmd
	if m.inputComponentMode == InputComponentFilePicker && m.inputMode == InputModeUpload {
		// Update file picker
		oldDir := m.filePicker.CurrentDirectory
		oldPath := m.filePicker.Path
		m.filePicker, cmd = m.filePicker.Update(msg)

		// Log changes for debugging
		if m.filePicker.CurrentDirectory != oldDir {
			logrus.Infof("FilePicker directory changed: '%s' -> '%s'", oldDir, m.filePicker.CurrentDirectory)
		}
		if m.filePicker.Path != oldPath {
			logrus.Infof("FilePicker path changed: '%s' -> '%s'", oldPath, m.filePicker.Path)
		}
	} else {
		// Update text input
		m.textInput, cmd = m.textInput.Update(msg)

		// Smart folder navigation: update file picker directory based on text input
		if m.inputMode == InputModeUpload {
			cmd = tea.Batch(cmd, m.updateFilePickerFromTextInput())
		}
	}

	return m, cmd
}

// processSearchInput processes search input
func (m *FileBrowserModel) processSearchInput() (tea.Model, tea.Cmd) {
	m.showInput = false
	m.inputMode = InputModeNone

	// If input is empty, clear search and restore original files
	if strings.TrimSpace(m.textInput.Value()) == "" {
		m.clearSearch()
		m.setMessage("Search cleared", messaging.MessageInfo)
		return m, m.loadFiles()
	}

	m.searchQuery = strings.TrimSpace(m.textInput.Value())
	m.isSearchMode = true
	m.textInput.SetValue("")
	m.textInput.Blur()

	// Reset pagination for new search
	m.currentPage = 1
	m.continuationToken = ""
	m.estimatedTotalPages = 1
	m.cursor = 0

	// Start loading with search query
	m.loading = true
	m.setMessage(fmt.Sprintf("Searching for '%s'...", m.searchQuery), messaging.MessageInfo)
	return m, m.loadFiles()
}

// processUploadInput processes upload input
func (m *FileBrowserModel) processUploadInput() (tea.Model, tea.Cmd) {
	m.showInput = false
	m.inputMode = InputModeNone

	filePath := strings.TrimSpace(m.textInput.Value())
	m.textInput.SetValue("")
	m.textInput.Blur()

	if filePath == "" {
		m.setMessage("No file path provided", messaging.MessageError)
		return m, nil
	}

	// Debug: Log the received file path for troubleshooting
	logrus.Infof("Upload input received file path: '%s' (length: %d)", filePath, len(filePath))

	// Expand tilde to home directory if present
	if strings.HasPrefix(filePath, "~/") {
		if homeDir, err := os.UserHomeDir(); err == nil {
			originalPath := filePath
			filePath = filepath.Join(homeDir, filePath[2:])
			logrus.Infof("Expanded tilde path: '%s' -> '%s'", originalPath, filePath)
		}
	}

	// Clean the file path (handles . and .. and extra slashes)
	originalPath := filePath
	filePath = filepath.Clean(filePath)
	if originalPath != filePath {
		logrus.Infof("Cleaned file path: '%s' -> '%s'", originalPath, filePath)
	}

	// Check if file exists (filepath.Clean handles spaces correctly)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		m.setMessage(fmt.Sprintf("File not found: '%s'", filePath), messaging.MessageError)
		return m, nil
	} else if err != nil {
		m.setMessage(fmt.Sprintf("Error accessing file: %v", err), messaging.MessageError)
		return m, nil
	}

	// Start upload process
	m.uploading = true
	m.uploadingFile = filepath.Base(filePath)
	m.setMessage(fmt.Sprintf("Uploading %s...", m.uploadingFile), messaging.MessageWarning)

	// Return command to start upload
	return m, m.uploadFile(filePath)
}

// processUploadWithPath processes upload with file picker selected path
func (m *FileBrowserModel) processUploadWithPath(filePath string) (tea.Model, tea.Cmd) {
	if filePath == "" {
		m.setMessage("No file selected", messaging.MessageError)
		return m, nil
	}

	logrus.Infof("File picker selected path: '%s' (length: %d)", filePath, len(filePath))

	// Clean the file path (handles . and .. and extra slashes, and spaces)
	originalPath := filePath
	filePath = filepath.Clean(filePath)
	if originalPath != filePath {
		logrus.Infof("Cleaned picker path: '%s' -> '%s'", originalPath, filePath)
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		m.setMessage(fmt.Sprintf("File not found: '%s'", filePath), messaging.MessageError)
		return m, nil
	} else if err != nil {
		m.setMessage(fmt.Sprintf("Error accessing file: %v", err), messaging.MessageError)
		return m, nil
	}

	// Start upload process
	m.uploading = true
	m.uploadingFile = filepath.Base(filePath)
	m.setMessage(fmt.Sprintf("Uploading %s...", m.uploadingFile), messaging.MessageWarning)

	// Return command to start upload
	return m, m.uploadFile(filePath)
}

// clearSearch clears search mode and reloads files without search
func (m *FileBrowserModel) clearSearch() {
	m.isSearchMode = false
	m.searchQuery = ""
	m.currentPage = 1
	m.continuationToken = ""
	m.estimatedTotalPages = 1
	m.cursor = 0
	m.loading = true
}

// setMessage sets a status message with type
func (m *FileBrowserModel) setMessage(message string, msgType messaging.MessageType) {
	m.messageManager.SetMessage(message, msgType)
}

// clearMessage clears the status message
func (m *FileBrowserModel) clearMessage() {
	m.messageManager.ClearMessage()
}

// uploadFile uploads a file using the FileUploader
func (m *FileBrowserModel) uploadFile(filePath string) tea.Cmd {
	return func() tea.Msg {
		if m.fileUploader == nil {
			return uploadCompletedMsg{
				file: filepath.Base(filePath),
				err:  errors.New("file uploader not initialized"),
			}
		}

		// Determine remote path (use filename)
		remotePath := filepath.Base(filePath)
		if m.prefix != "" {
			remotePath = m.prefix + "/" + remotePath
		}

		// Create upload options from config
		options := &utils.UploadOptions{
			Overwrite:    m.config.Upload.DefaultOverwrite,
			PublicAccess: m.config.Upload.DefaultPublic,
			ContentType:  "", // Auto-detect
		}

		// Create progress callback
		progressCallback := func(uploaded, total int64, percentage float64) {
			if m.program != nil {
				m.program.Send(uploadProgressMsg{
					uploaded:   uploaded,
					total:      total,
					percentage: percentage,
				})
			}
		}

		// Perform upload with progress
		ctx := context.Background()
		err := m.fileUploader.UploadFileWithProgress(ctx, filePath, remotePath, options, progressCallback)

		return uploadCompletedMsg{
			file: filepath.Base(filePath),
			err:  err,
		}
	}
}

// updateFilePickerFromTextInput updates file picker directory based on text input path
func (m *FileBrowserModel) updateFilePickerFromTextInput() tea.Cmd {
	m.updateFilePickerFromTextInputSync()
	return nil
}

// updateFilePickerFromTextInputSync synchronously updates file picker directory based on text input path
func (m *FileBrowserModel) updateFilePickerFromTextInputSync() {
	inputText := strings.TrimSpace(m.textInput.Value())
	if inputText == "" {
		logrus.Debug("No text input, skipping file picker directory update")
		return
	}

	logrus.Infof("Updating file picker directory from text input: '%s'", inputText)
	oldDir := m.filePicker.CurrentDirectory

	// Expand tilde to home directory if present
	dirPath := inputText
	if strings.HasPrefix(dirPath, "~/") {
		if homeDir, err := os.UserHomeDir(); err == nil {
			dirPath = filepath.Join(homeDir, dirPath[2:])
			logrus.Infof("Expanded tilde for directory: '%s' -> '%s'", inputText, dirPath)
		}
	}

	// If the input contains a filename, get the directory part
	// Check if it's a directory or extract directory from file path
	if stat, err := os.Stat(dirPath); err == nil {
		if stat.IsDir() {
			// Input is a valid directory, update file picker to this directory
			cleanDir := filepath.Clean(dirPath)
			if m.filePicker.CurrentDirectory != cleanDir {
				logrus.Infof("Setting file picker directory to: '%s'", cleanDir)
				m.filePicker.CurrentDirectory = cleanDir
			}
		} else {
			// Input is a file, update file picker to the file's directory
			dir := filepath.Dir(dirPath)
			if stat, err := os.Stat(dir); err == nil && stat.IsDir() {
				cleanDir := filepath.Clean(dir)
				if m.filePicker.CurrentDirectory != cleanDir {
					logrus.Infof("Setting file picker directory to parent: '%s'", cleanDir)
					m.filePicker.CurrentDirectory = cleanDir
				}
			}
		}
	} else {
		// Path doesn't exist yet, try to get the directory part
		dir := filepath.Dir(dirPath)
		if stat, err := os.Stat(dir); err == nil && stat.IsDir() {
			cleanDir := filepath.Clean(dir)
			if m.filePicker.CurrentDirectory != cleanDir {
				logrus.Infof("Setting file picker directory to existing parent: '%s'", cleanDir)
				m.filePicker.CurrentDirectory = cleanDir
			}
		} else {
			// Directory doesn't exist - provide user feedback
			logrus.Warnf("Directory does not exist: '%s'", dir)
			m.setMessage(fmt.Sprintf("Directory does not exist: '%s'", dir), messaging.MessageError)
		}
	}

	// Log the final result
	if m.filePicker.CurrentDirectory != oldDir {
		logrus.Infof("File picker directory updated: '%s' -> '%s'", oldDir, m.filePicker.CurrentDirectory)
	} else {
		logrus.Debug("File picker directory unchanged")
	}
}

// resetUploadState completely resets all upload-related state
func (m *FileBrowserModel) resetUploadState() {
	logrus.Info("Resetting upload state to clean defaults")

	// Clear text input
	m.textInput.SetValue("")
	m.textInput.Blur()

	// Reset file picker to default directory (user home)
	if homeDir, err := os.UserHomeDir(); err == nil {
		if m.filePicker.CurrentDirectory != homeDir {
			logrus.Infof("Resetting file picker directory from '%s' to home: '%s'", m.filePicker.CurrentDirectory, homeDir)
			m.filePicker.CurrentDirectory = homeDir
		}
	} else {
		// Fallback to current working directory
		if cwd, err := os.Getwd(); err == nil {
			if m.filePicker.CurrentDirectory != cwd {
				logrus.Infof("Resetting file picker directory from '%s' to cwd: '%s'", m.filePicker.CurrentDirectory, cwd)
				m.filePicker.CurrentDirectory = cwd
			}
		}
	}

	// Clear file picker selected path
	m.filePicker.Path = ""

	// Reset input states
	m.showInput = false
	m.inputMode = InputModeNone
	m.inputComponentMode = InputComponentText
	m.inputPrompt = ""
}

// uploadCompletedMsg represents upload completion
type uploadCompletedMsg struct {
	file string
	err  error
}

type uploadProgressMsg struct {
	uploaded   int64
	total      int64
	percentage float64
}
