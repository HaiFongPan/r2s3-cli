package theme

import "github.com/charmbracelet/lipgloss"

// Border styles with Crush-inspired elegance
var (
	// Legacy unified border for backward compatibility
	BorderStyleUnified = lipgloss.RoundedBorder()

	// Elegant separator for panels
	BorderStyleSeparator = "│"

	// Refined borders for different use cases
	BorderStyleCard = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
	}

	// Double border for emphasis
	BorderStyleEmphasized = lipgloss.DoubleBorder()
)
