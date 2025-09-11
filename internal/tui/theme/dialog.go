package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// Dialog style constants
const (
	DialogOpacity = 0.95
)

// CreateAdvancedDialogStyle creates a styled dialog box with advanced options
func CreateAdvancedDialogStyle(width, height int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorBrightBlue)).
		Background(lipgloss.Color("#000000")).
		Padding(1, 2).
		Align(lipgloss.Center).
		AlignVertical(lipgloss.Center)
}

// CreateOverlayStyle creates a style for overlaying dialogs
func CreateOverlayStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")) // Dim the text for background
}

// CreatePromptStyle creates a style for prompt text in dialogs
func CreatePromptStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorBrightYellow)).
		Bold(true).
		Align(lipgloss.Center)
}

// CreateDialogButtonStyle creates a style for dialog buttons
func CreateDialogButtonStyle(selected bool) lipgloss.Style {
	style := lipgloss.NewStyle().
		Padding(0, 2).
		Margin(0, 1).
		Border(lipgloss.RoundedBorder())

	if selected {
		return style.
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color(ColorBrightYellow)).
			BorderForeground(lipgloss.Color(ColorBrightYellow))
	}

	return style.
		Foreground(lipgloss.Color(ColorBrightCyan)).
		BorderForeground(lipgloss.Color(ColorBrightCyan))
}