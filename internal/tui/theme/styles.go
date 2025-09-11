package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// CreateUnifiedPanelStyle creates a consistent panel style
func CreateUnifiedPanelStyle(width, height int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(BorderStyleUnified).
		BorderForeground(lipgloss.Color(ColorBrightBlue)).
		Padding(1).
		Foreground(lipgloss.Color(ColorWhite))
}

// CreateSectionHeaderStyle creates a consistent section header style
func CreateSectionHeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorBrightCyan)).
		MarginBottom(1)
}

// CreateInfoTextStyle creates a consistent info text style
func CreateInfoTextStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorWhite))
}

// CreateSecondaryTextStyle creates a consistent secondary text style
func CreateSecondaryTextStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorBrightBlack)).
		Italic(true)
}

// CreateDialogStyle creates a consistent dialog style
func CreateDialogStyle(width int, borderColor string) lipgloss.Style {
	style := lipgloss.NewStyle().
		Border(BorderStyleUnified).
		Padding(2, 3).
		Width(width).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color(ColorWhite))

	if borderColor != "" {
		style = style.BorderForeground(lipgloss.Color(borderColor))
	} else {
		style = style.BorderForeground(lipgloss.Color(ColorBrightBlue))
	}

	return style
}

// CreateHeaderStyle creates a consistent header style
func CreateHeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorBrightCyan)).
		MarginBottom(1).
		MarginLeft(1) // Align with left panel border
}

// CreateFooterStyle creates a consistent footer style
func CreateFooterStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorBrightBlack)).
		MarginTop(1).
		MarginLeft(1) // Align with left panel border
}

// CreateLoadingStyle creates a consistent loading state style
func CreateLoadingStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorBrightYellow))
}

// CreateErrorStyle creates a consistent error style
func CreateErrorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorBrightRed))
}