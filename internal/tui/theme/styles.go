package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// CreateUnifiedPanelStyle creates a consistent panel style with Crush elegance
func CreateUnifiedPanelStyle(width, height int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorBrightBlue)).
		Padding(1, 2).
		Foreground(lipgloss.Color(ColorText))
}

// CreateSectionHeaderStyle creates a glamorous section header style
func CreateSectionHeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorBrightCyan)).
		MarginBottom(1).
		PaddingLeft(1)
}

// CreateInfoTextStyle creates a consistent info text style
func CreateInfoTextStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorText))
}

// CreateSecondaryTextStyle creates a consistent secondary text style
func CreateSecondaryTextStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorBrightBlack)).
		Italic(true)
}

// CreateDialogStyle creates an elegant dialog style with Crush aesthetics
func CreateDialogStyle(width int, borderColor string) lipgloss.Style {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(3, 4).
		Width(width).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color(ColorText))

	if borderColor != "" {
		style = style.BorderForeground(lipgloss.Color(borderColor))
	} else {
		style = style.BorderForeground(lipgloss.Color(ColorBrightBlue))
	}

	return style
}

// CreateHeaderStyle creates a glamorous header style
func CreateHeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorBrightCyan)).
		MarginBottom(1).
		MarginLeft(2).
		PaddingLeft(1)
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

// CreateHighlightStyle creates a Crush-inspired highlight style
func CreateHighlightStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(ColorBrightBlue)).
		Foreground(lipgloss.Color(ColorWhite)).
		Padding(0, 1).
		Bold(true)
}

// CreateCardStyle creates an elegant card-like container
func CreateCardStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorBrightBlack)).
		Padding(1, 2).
		Width(width).
		Foreground(lipgloss.Color(ColorText))
}

// CreateAccentTextStyle creates emphasized accent text
func CreateAccentTextStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorBrightBlue)).
		Bold(true)
}
