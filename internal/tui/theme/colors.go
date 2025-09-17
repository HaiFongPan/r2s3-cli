package theme

// Terminal-compatible color constants using ANSI standard colors
// These colors work consistently across different terminal themes
const (
	// Primary colors (ANSI standard - adaptive to terminal theme)
	ColorText         = ""        // Use terminal's default foreground color
	ColorWhite        = "15"      // ANSI 15 - bright white, better contrast
	ColorBrightBlack  = "8"       // ANSI 8 - bright black/gray
	ColorHint         = "240"     // ANSI 240 - darker gray for hints
	ColorBrightBlue   = "12"      // ANSI 12 - bright blue
	ColorBrightCyan   = "14"      // ANSI 14 - bright cyan
	ColorBrightGreen  = "10"      // ANSI 10 - bright green
	ColorBrightYellow = "11"      // ANSI 11 - bright yellow
	ColorBrightRed    = "9"       // ANSI 9 - bright red

	// File type colors (using ANSI palette)
	ColorFileImage        = "#74C0FC" // Light blue
	ColorFileDocument     = "#51CF66" // Green
	ColorFileSpreadsheet  = "#69DB7C" // Light green
	ColorFilePresentation = "#FFD43B" // Yellow
	ColorFileArchive      = "#FCC419" // Amber
	ColorFileVideo        = "#FF8787" // Light red
	ColorFileAudio        = "#DA77F2" // Purple
	ColorFileText         = "#74C0FC" // Cyan
	ColorFileCode         = "#B197FC" // Light purple
	ColorFileData         = "#74C0FC" // Light blue
	ColorFileFont         = "#FFB3BA" // Light pink
)

// GetFileColor returns the color for a given file category
func GetFileColor(category string) string {
	switch category {
	case "image":
		return ColorFileImage
	case "document":
		return ColorFileDocument
	case "spreadsheet":
		return ColorFileSpreadsheet
	case "presentation":
		return ColorFilePresentation
	case "archive":
		return ColorFileArchive
	case "video":
		return ColorFileVideo
	case "audio":
		return ColorFileAudio
	case "text":
		return ColorFileText
	case "code":
		return ColorFileCode
	case "data":
		return ColorFileData
	case "font":
		return ColorFileFont
	default:
		return ColorText
	}
}

// Message type constants for compatibility
const (
	MessageInfo = iota
	MessageSuccess
	MessageWarning
	MessageError
)

// GetMessageColor returns the color for a given message type
func GetMessageColor(messageType int) string {
	switch messageType {
	case MessageError:
		return ColorBrightRed
	case MessageSuccess:
		return ColorBrightGreen
	case MessageWarning:
		return ColorBrightYellow
	default: // MessageInfo
		return ColorBrightCyan
	}
}

// GetMessageIcon returns the icon for a given message type
func GetMessageIcon(messageType int) string {
	switch messageType {
	case MessageError:
		return "❌ "
	case MessageSuccess:
		return "✅ "
	case MessageWarning:
		return "⚠️ "
	default: // MessageInfo
		return "ℹ️ "
	}
}