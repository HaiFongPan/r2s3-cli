package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sirupsen/logrus"

	"github.com/HaiFongPan/r2s3-cli/internal/config"
	"github.com/HaiFongPan/r2s3-cli/internal/r2"
	"github.com/HaiFongPan/r2s3-cli/internal/tui/theme"
)

// BucketItem represents a bucket in the selector
type BucketItem struct {
	Name      string
	IsMain    bool
	IsCurrent bool
	Error     string
}

// BucketSelectorModel represents the bucket selector TUI model
type BucketSelectorModel struct {
	buckets       []BucketItem
	selectedIndex int
	loading       bool
	showHelp      bool
	message       string
	messageTimer  time.Time
	client        *r2.Client
	config        *config.Config
	keyMap        BucketSelectorKeyMap
	help          help.Model
	windowWidth   int
	windowHeight  int
}

// BucketSelectorKeyMap defines keybindings for bucket selector
type BucketSelectorKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Select  key.Binding
	SetMain key.Binding
	Help    key.Binding
	Quit    key.Binding
	Refresh key.Binding
}

// DefaultBucketSelectorKeyMap returns default keybindings
func DefaultBucketSelectorKeyMap() BucketSelectorKeyMap {
	return BucketSelectorKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "switch to bucket"),
		),
		SetMain: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "set as main bucket"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "esc"),
			key.WithHelp("q/esc", "quit"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
	}
}

// ShortHelp returns the short help view
func (k BucketSelectorKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.SetMain, k.Help, k.Quit}
}

// FullHelp returns the full help view
func (k BucketSelectorKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Select, k.SetMain},
		{k.Refresh, k.Help, k.Quit},
	}
}

// NewBucketSelectorModel creates a new bucket selector model
func NewBucketSelectorModel(client *r2.Client, cfg *config.Config) *BucketSelectorModel {
	return &BucketSelectorModel{
		client:       client,
		config:       cfg,
		keyMap:       DefaultBucketSelectorKeyMap(),
		help:         help.New(),
		loading:      true,
		windowWidth:  80,
		windowHeight: 24,
	}
}

// Init initializes the bucket selector
func (m *BucketSelectorModel) Init() tea.Cmd {
	return m.loadBuckets()
}

// Update handles messages in the bucket selector
func (m *BucketSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		m.help.Width = msg.Width
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case bucketsLoadedMsg:
		m.loading = false
		m.buckets = msg.buckets
		if len(m.buckets) > 0 && m.selectedIndex >= len(m.buckets) {
			m.selectedIndex = 0
		}
		return m, nil

	case bucketErrorMsg:
		m.loading = false
		m.setMessage(fmt.Sprintf("Error loading buckets: %v", msg.err))
		return m, nil

	case bucketSwitchedMsg:
		m.setMessage(fmt.Sprintf("Switched to bucket: %s", msg.bucket))
		return m, nil

	case mainBucketSetMsg:
		if msg.err != nil {
			m.setMessage(fmt.Sprintf("Error setting main bucket: %v", msg.err))
		} else {
			m.setMessage(fmt.Sprintf("Set main bucket: %s", msg.bucket))
			// Update local state
			for i := range m.buckets {
				m.buckets[i].IsMain = m.buckets[i].Name == msg.bucket
			}
		}
		return m, nil

	default:
		return m, nil
	}
}

// handleKeyPress processes keyboard input
func (m *BucketSelectorModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.loading {
		if key.Matches(msg, m.keyMap.Quit) {
			return m, tea.Quit
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keyMap.Up):
		if m.selectedIndex > 0 {
			m.selectedIndex--
		}
		return m, nil

	case key.Matches(msg, m.keyMap.Down):
		if m.selectedIndex < len(m.buckets)-1 {
			m.selectedIndex++
		}
		return m, nil

	case key.Matches(msg, m.keyMap.Select):
		if len(m.buckets) > 0 {
			bucket := m.buckets[m.selectedIndex].Name
			return m, m.switchToBucket(bucket)
		}
		return m, nil

	case key.Matches(msg, m.keyMap.SetMain):
		if len(m.buckets) > 0 {
			bucket := m.buckets[m.selectedIndex].Name
			return m, m.setMainBucket(bucket)
		}
		return m, nil

	case key.Matches(msg, m.keyMap.Help):
		m.showHelp = !m.showHelp
		return m, nil

	case key.Matches(msg, m.keyMap.Refresh):
		m.loading = true
		return m, m.loadBuckets()

	case key.Matches(msg, m.keyMap.Quit):
		return m, tea.Quit

	default:
		return m, nil
	}
}

// View renders the bucket selector
func (m *BucketSelectorModel) View() string {
	if m.loading {
		return m.renderLoading()
	}

	if len(m.buckets) == 0 {
		return m.renderEmpty()
	}

	if m.showHelp {
		return m.renderHelp()
	}

	return m.renderBucketList()
}

// renderLoading renders the loading state
func (m *BucketSelectorModel) renderLoading() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFEB3B")).
		Render("🗂️  R2 Bucket Selector")

	loading := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Render("Loading buckets...")

	content := lipgloss.JoinVertical(lipgloss.Left, title, "", loading)

	return lipgloss.Place(
		m.windowWidth, m.windowHeight,
		lipgloss.Center, lipgloss.Center,
		lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FFEB3B")).
			Padding(2, 4).
			Render(content),
	)
}

// renderEmpty renders the empty state
func (m *BucketSelectorModel) renderEmpty() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFEB3B")).
		Render("🗂️  R2 Bucket Selector")

	empty := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Render("No buckets found")

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Render("Press 'r' to refresh or 'q' to quit")

	content := lipgloss.JoinVertical(lipgloss.Left, title, "", empty, "", help)

	return lipgloss.Place(
		m.windowWidth, m.windowHeight,
		lipgloss.Center, lipgloss.Center,
		lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FFEB3B")).
			Padding(2, 4).
			Render(content),
	)
}

// renderBucketList renders the bucket list
func (m *BucketSelectorModel) renderBucketList() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFEB3B")).
		Render("🗂️  R2 Bucket Selector")

	var bucketItems []string
	for i, bucket := range m.buckets {
		var prefix string
		var style lipgloss.Style

		if i == m.selectedIndex {
			prefix = "▶ "
			style = lipgloss.NewStyle().
				Background(lipgloss.Color(theme.ColorBrightBlue)).
				Foreground(lipgloss.Color(theme.ColorWhite)).
				Bold(true).
				Padding(0, 1)
		} else {
			prefix = "  "
			style = lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.ColorText)).
				Padding(0, 1)
		}

		// Add main bucket indicator
		if bucket.IsMain {
			prefix += "* "
		} else {
			prefix += "  "
		}

		bucketLine := prefix + bucket.Name
		if bucket.Error != "" {
			bucketLine += lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.ColorBrightRed)).
				Italic(true).
				Render(fmt.Sprintf(" (%s)", bucket.Error))
		}

		bucketItems = append(bucketItems, style.Render(bucketLine))
	}

	bucketList := strings.Join(bucketItems, "\n")

	// Show current effective bucket
	effectiveBucket := m.config.GetEffectiveBucket()
	currentInfo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Render(fmt.Sprintf("Current: %s", effectiveBucket))

	// Show message if any
	var messageView string
	if m.message != "" && time.Since(m.messageTimer) < 3*time.Second {
		messageView = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50FA7B")).
			Render(m.message)
	}

	// Help line
	helpLine := m.help.ShortHelpView(m.keyMap.ShortHelp())

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		currentInfo,
		"",
		bucketList,
		"",
	)

	if messageView != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, content, messageView, "")
	}

	content = lipgloss.JoinVertical(lipgloss.Left, content, helpLine)

	return lipgloss.Place(
		m.windowWidth, m.windowHeight,
		lipgloss.Center, lipgloss.Center,
		lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FFEB3B")).
			Padding(2, 4).
			Width(60).
			Render(content),
	)
}

// renderHelp renders the help view
func (m *BucketSelectorModel) renderHelp() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFEB3B")).
		Render("🗂️  Bucket Selector Help")

	helpContent := m.help.FullHelpView(m.keyMap.FullHelp())

	content := lipgloss.JoinVertical(lipgloss.Left, title, "", helpContent)

	return lipgloss.Place(
		m.windowWidth, m.windowHeight,
		lipgloss.Center, lipgloss.Center,
		lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FFEB3B")).
			Padding(2, 4).
			Width(60).
			Render(content),
	)
}

// setMessage sets a temporary message
func (m *BucketSelectorModel) setMessage(message string) {
	m.message = message
	m.messageTimer = time.Now()
}

// Messages for bucket selector

type bucketsLoadedMsg struct {
	buckets []BucketItem
}

type bucketErrorMsg struct {
	err error
}

type bucketSwitchedMsg struct {
	bucket string
}

type mainBucketSetMsg struct {
	bucket string
	err    error
}

// loadBuckets loads available buckets
func (m *BucketSelectorModel) loadBuckets() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		logrus.Info("BucketSelector: loading buckets")

		// Get all buckets
		buckets, err := m.client.ListBuckets(ctx)
		if err != nil {
			logrus.Errorf("Failed to load buckets: %v", err)
			return bucketErrorMsg{err: err}
		}

		var bucketItems []BucketItem
		mainBucket := m.config.GetMainBucket()
		currentBucket := m.config.GetEffectiveBucket()

		for _, bucket := range buckets {
			bucketName := ""
			if bucket.Name != nil {
				bucketName = *bucket.Name
			}

			item := BucketItem{
				Name:      bucketName,
				IsMain:    bucketName == mainBucket,
				IsCurrent: bucketName == currentBucket,
			}

			bucketItems = append(bucketItems, item)
		}

		logrus.Infof("BucketSelector: loaded %d buckets", len(bucketItems))

		// Log each loaded bucket for debugging
		for i, item := range bucketItems {
			status := ""
			if item.IsMain {
				status += " [MAIN]"
			}
			if item.IsCurrent {
				status += " [CURRENT]"
			}
			logrus.Infof("BucketSelector: bucket[%d]: %s%s", i, item.Name, status)
		}

		return bucketsLoadedMsg{buckets: bucketItems}
	}
}

// switchToBucket switches to the selected bucket temporarily
func (m *BucketSelectorModel) switchToBucket(bucket string) tea.Cmd {
	return func() tea.Msg {
		logrus.Infof("BucketSelector: switching to bucket: %s", bucket)
		m.config.SetTempBucket(bucket)
		return bucketSwitchedMsg{bucket: bucket}
	}
}

// setMainBucket sets the selected bucket as main bucket
func (m *BucketSelectorModel) setMainBucket(bucket string) tea.Cmd {
	return func() tea.Msg {
		logrus.Infof("BucketSelector: setting main bucket: %s", bucket)
		err := m.config.SetMainBucket(bucket)
		return mainBucketSetMsg{bucket: bucket, err: err}
	}
}
