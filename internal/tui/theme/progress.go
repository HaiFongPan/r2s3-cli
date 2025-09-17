package theme

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Progress styling and components

// CreateProgressBarStyle creates a style for progress bars
func CreateProgressBarStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorBrightBlue)).
		Padding(1)
}

// CreateCrushProgressOptions creates Crush-styled progress bar options
func CreateCrushProgressOptions() []ProgressOption {
	return []ProgressOption{
		WithSolidFill(ColorBrightBlue),
		WithEmptyFill(ColorBrightBlack),
		WithoutPercentage(),
	}
}

// ProgressOption represents a progress bar configuration option
type ProgressOption func(*ProgressConfig)

// ProgressConfig holds progress bar styling configuration
type ProgressConfig struct {
	SolidFill      string
	EmptyFill      string
	ShowPercentage bool
}

// WithSolidFill sets the filled portion color
func WithSolidFill(color string) ProgressOption {
	return func(c *ProgressConfig) {
		c.SolidFill = color
	}
}

// WithEmptyFill sets the empty portion color
func WithEmptyFill(color string) ProgressOption {
	return func(c *ProgressConfig) {
		c.EmptyFill = color
	}
}

// WithoutPercentage hides the percentage display
func WithoutPercentage() ProgressOption {
	return func(c *ProgressConfig) {
		c.ShowPercentage = false
	}
}

// CreateProgressTextStyle creates a style for progress text
func CreateProgressTextStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorBrightCyan)).
		Bold(true)
}

// CreateStatusIndicatorStyle creates a style for status indicators
func CreateStatusIndicatorStyle(success bool) lipgloss.Style {
	color := ColorBrightRed
	if success {
		color = ColorBrightGreen
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Bold(true)
}

// FormatProgressMessage formats a progress message with consistent styling
func FormatProgressMessage(operation, filename string, percentage float64) string {
	if percentage >= 0 {
		return fmt.Sprintf("%s %s... %.1f%%", operation, filename, percentage)
	}
	return fmt.Sprintf("%s %s...", operation, filename)
}

// FormatSuccessMessage formats a success message
func FormatSuccessMessage(operation, filename string) string {
	return fmt.Sprintf("%s %s successfully", filename, operation)
}

// FormatErrorMessage formats an error message
func FormatErrorMessage(operation string, err error) string {
	return fmt.Sprintf("%s failed: %v", operation, err)
}
