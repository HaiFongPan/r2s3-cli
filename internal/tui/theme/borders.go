package theme

import "github.com/charmbracelet/lipgloss"

// Border styles using Unicode box drawing characters
var (
	BorderStyleUnified = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "┌",
		TopRight:    "┐",
		BottomLeft:  "└",
		BottomRight: "┘",
	}

	BorderStyleSeparator = "│"
)