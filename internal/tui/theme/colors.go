package theme

// Terminal-compatible color constants using ANSI standard colors
// These colors work consistently across different terminal themes
const (
	// Primary colors (ANSI standard)
	ColorWhite        = "#FFFFFF" // ANSI 15 - primary text
	ColorBrightBlack  = "#808080" // ANSI 8 - secondary text
	ColorBrightBlue   = "#5C7CFA" // ANSI 12 - primary accent
	ColorBrightCyan   = "#51CF66" // ANSI 14 - secondary accent
	ColorBrightGreen  = "#51CF66" // ANSI 10 - success/links
	ColorBrightYellow = "#FFD43B" // ANSI 11 - warning
	ColorBrightRed    = "#FF6B6B" // ANSI 9 - error

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
		return ColorWhite
	}
}

// GetMessageColor returns the color for a given message type
func GetMessageColor(messageType int) string {
	const (
		MessageError = iota
		MessageSuccess
		MessageWarning
		MessageInfo
	)

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
	const (
		MessageError = iota
		MessageSuccess
		MessageWarning
		MessageInfo
	)

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