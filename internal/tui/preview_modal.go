package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	img "github.com/HaiFongPan/r2s3-cli/internal/tui/image"
	"github.com/HaiFongPan/r2s3-cli/internal/tui/theme"
)

// ImagePreviewModel is a fullscreen modal to show image previews safely
type ImagePreviewModel struct {
	width       int
	height      int
	file        FileItem
	manager     *img.ImageManager
	forceReload bool

	loading  bool
	rendered string
	err      error

	// metadata from preview
	imgPixelW int
	imgPixelH int
	cacheHit  bool

	// spinner for loading line
	spin spinner.Model

	// terminal cell footprint
	imgCols int
	imgRows int
}

type (
	previewReadyMsg struct{}
	modalClosedMsg  struct{}
)

func NewImagePreviewModel(manager *img.ImageManager, file FileItem, width, height int, force bool) *ImagePreviewModel {
	return &ImagePreviewModel{
		width:       width,
		height:      height,
		file:        file,
		manager:     manager,
		forceReload: force,
		loading:     true,
	}
}

func (m *ImagePreviewModel) Init() tea.Cmd {
	// Use crisp graphics mode if available
	m.manager.SetUseTextRender(false)
	// Reserve 2 lines (title+status) on top and 1 line hint bottom
	usableCols := max(1, m.width-4)
	usableRows := max(1, m.height-4)
	// Approximate pixel size from cells (8x16)
	m.manager.SetDisplaySize(usableCols*8, usableRows*16)

	m.spin = spinner.New()
	m.spin.Spinner = spinner.Line
	m.spin.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorBrightYellow))

	load := func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		item := img.FileItem{
			Key:          m.file.Key,
			Size:         m.file.Size,
			LastModified: m.file.LastModified,
			ContentType:  m.file.ContentType,
			Category:     m.file.Category,
		}

		var (
			preview *img.ImagePreview
			err     error
		)

		if m.forceReload {
			preview, err = m.manager.PreviewImageForce(ctx, m.file.Key, item)
		} else {
			preview, err = m.manager.PreviewImage(ctx, m.file.Key, item)
		}
		if err != nil {
			m.err = err
		} else if preview != nil {
			m.cacheHit = preview.CacheHit
			cols := preview.RenderCols
			if cols < 1 {
				cols = 1
			}
			if cols > m.width {
				cols = m.width
			}
			gap := m.width - cols
			col := 1 + gap/2
			if col < 1 {
				col = 1
			}
			if col > m.width {
				col = m.width
			}
			// Calculate row position dynamically: name + status + hint + cache + separator + margin
			row := 4 // base: name, status, hint lines
			if m.forceReload || !m.forceReload { // cacheInfo will be shown
				row++
			}
			row++ // separator line
			row++ // extra margin for better spacing

			if m.forceReload {
				preview, err = m.manager.PreviewImageAtForce(ctx, m.file.Key, item, col, row)
			} else {
				preview, err = m.manager.PreviewImageAt(ctx, m.file.Key, item, col, row)
			}
			if err != nil {
				m.err = err
			} else if preview != nil {
				m.rendered = preview.RenderedData
				m.imgPixelW = preview.DisplaySize.Width
				m.imgPixelH = preview.DisplaySize.Height
				m.imgCols = preview.RenderCols
				m.imgRows = preview.RenderRows
				m.cacheHit = preview.CacheHit
			}
		}
		m.loading = false
		return previewReadyMsg{}
	}

	return tea.Batch(load, m.spin.Tick)
}

func (m *ImagePreviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "p", "P":
			// Clear graphics images only (no full clear)
			_ = m.manager.ClearPreview()
			return m, func() tea.Msg { return modalClosedMsg{} }
		}
	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spin, cmd = m.spin.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m *ImagePreviewModel) View() string {
	// First line: filename (centered)
	filename := filepath.Base(m.file.Key)
	nameLine := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Bold(true).
		Foreground(lipgloss.Color(theme.ColorBrightCyan)).
		Render("🖼 " + filename)

	// Second line: loading/meta (centered)
	var statusText string
	if m.loading {
		statusText = fmt.Sprintf("%s Loading image preview…", m.spin.View())
	} else if m.err != nil {
		statusText = theme.CreateErrorStyle().Render(fmt.Sprintf("Failed to render: %v", m.err))
	} else if m.imgPixelW > 0 && m.imgPixelH > 0 && m.file.Size > 0 {
		statusText = fmt.Sprintf("%dx%d  •  %s", m.imgPixelW, m.imgPixelH, formatFileSize(m.file.Size))
	}
	statusLine := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(statusText)

	// Third line: hint (above image; centered)
	hint := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color(theme.ColorBrightBlack)).
		Render("q/esc/p/P to close • arrows to navigate when closed")

	// Cache info with enhanced styling
	cacheInfo := ""
	if !m.loading && m.err == nil {
		var source string
		switch {
		case m.forceReload && !m.cacheHit:
			source = "🔄 Preview refreshed from R2"
		case m.cacheHit:
			source = "⚡ Preview served from cache"
		default:
			source = "📡 Preview downloaded from R2"
		}
		cacheInfo = theme.CreateHintStyle().
			Width(m.width).
			Align(lipgloss.Center).
			Render(source)
	}

	// 装饰性分隔线（只有当有图片内容时才显示）
	separator := ""
	if !m.loading && m.err == nil && m.rendered != "" {
		separatorStyle := lipgloss.NewStyle().
			Width(m.width).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color(theme.ColorBrightBlue)).
			Bold(true)
		separator = separatorStyle.Render("─────────────────── 🖼 ───────────────────")
	}

	// Image block, horizontally centered by left padding spaces only once
	var imageBlock string
	if !m.loading && m.err == nil && m.rendered != "" {
		// Use actual render cell width from preview
		imgCols := m.imgCols
		if imgCols < 1 {
			imgCols = 1
		}
		if imgCols > m.width {
			imgCols = m.width - 2
		}
		gap := m.width - imgCols
		if gap < 0 {
			gap = 0
		}
		// Absolute horizontal position (exact center by cells)
		col := 1 + gap/2
		if col < 1 {
			col = 1
		}
		if col > m.width {
			col = m.width
		}
		// Start row after header, status, hint, cache info, separator lines + margin
		row := 4 // base: name, status, hint lines
		if cacheInfo != "" {
			row++
		}
		if separator != "" {
			row++
		}
		row++ // extra margin for better spacing
		// Move cursor to target row;column, then write image
		imageBlock = fmt.Sprintf("\x1b[%d;%dH%s\x1b[0m", row, col, m.rendered)
	}

	// Compose without global centering; start from top-left
	var b strings.Builder
	b.WriteString(nameLine)
	b.WriteString("\n")
	b.WriteString(statusLine)
	b.WriteString("\n")
	b.WriteString(hint)
	b.WriteString("\n")
	if cacheInfo != "" {
		b.WriteString(cacheInfo)
		b.WriteString("\n")
	}
	if separator != "" {
		b.WriteString(separator)
		b.WriteString("\n")
	}
	if imageBlock != "" {
		b.WriteString(imageBlock)
	}
	return b.String()
}

// modalClosedMsg is sent to parent model when preview modal is closed
