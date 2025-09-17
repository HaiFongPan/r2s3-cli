package theme

// Crush-inspired glamorous color palette
// Sophisticated colors that make the command line beautiful
const (
	// Primary Crush color palette - sophisticated and glamorous
	ColorText         = "#FAFAFA" // Crush primary text - elegant off-white
	ColorWhite        = "#FFFFFF" // Pure white for high contrast elements
	ColorBrightBlack  = "#6B7280" // Refined gray for secondary text
	ColorHint         = "#9CA3AF" // Subtle gray for hints and disabled text
	ColorBrightBlue   = "#7D56F4" // Crush signature purple - primary accent
	ColorBrightCyan   = "#06D6A0" // Vibrant teal - secondary accent
	ColorBrightGreen  = "#10B981" // Success green - modern and clean
	ColorBrightYellow = "#F59E0B" // Warning amber - warm and visible
	ColorBrightRed    = "#EF4444" // Error red - clear and striking

	// File type colors - Crush-inspired sophisticated palette
	ColorFileImage        = "#8B5CF6" // Rich purple for images
	ColorFileDocument     = "#10B981" // Clean green for documents
	ColorFileSpreadsheet  = "#06D6A0" // Vibrant teal for spreadsheets
	ColorFilePresentation = "#F59E0B" // Warm amber for presentations
	ColorFileArchive      = "#EC4899" // Bright pink for archives
	ColorFileVideo        = "#EF4444" // Rich red for videos
	ColorFileAudio        = "#A855F7" // Deep purple for audio
	ColorFileText         = "#06B6D4" // Sky blue for text files
	ColorFileCode         = "#7D56F4" // Signature purple for code
	ColorFileData         = "#3B82F6" // Bright blue for data
	ColorFileFont         = "#EC4899" // Pink for fonts
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
