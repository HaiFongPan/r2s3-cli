package theme

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// URL styling constants
const (
	URLColor     = "#5C7CFA" // Bright blue for URLs
	URLColorCode = "51"      // ANSI color code for URLs
)

// CreateURLSectionStyle creates a style for URL section headers
func CreateURLSectionStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorBrightGreen)).
		Bold(true)
}

// CreatePreviewURLSectionStyle creates a style for preview URL section headers
func CreatePreviewURLSectionStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorBrightYellow)).
		Bold(true)
}

// FormatClickableURL formats a URL as a clickable hyperlink with terminal-compatible colors
func FormatClickableURL(displayText, url string) string {
	// OSC 8 escape sequence for hyperlinks: \033]8;;url\033\\text\033]8;;\033\\
	hyperlink := fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, displayText)
	
	// Add terminal-compatible colors and underline
	return fmt.Sprintf("\033[38;5;%sm\033[4m%s\033[0m", URLColorCode, hyperlink)
}

// CreateHintStyle creates a style for URL hints and tips
func CreateHintStyle() lipgloss.Style {
	return CreateSecondaryTextStyle()
}