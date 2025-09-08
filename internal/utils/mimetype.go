package utils

import (
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

// DetectContentType detects the MIME type of a file using multiple methods
func DetectContentType(filePath string, reader io.Reader) (string, error) {
	// First, try to detect from file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if contentType := mime.TypeByExtension(ext); contentType != "" {
		return contentType, nil
	}

	// If extension detection fails, try to detect from file content
	if reader != nil {
		// Read the first 512 bytes for content detection
		buffer := make([]byte, 512)
		n, err := reader.Read(buffer)
		if err != nil && err != io.EOF {
			return "", err
		}

		// Detect content type from the buffer
		contentType := http.DetectContentType(buffer[:n])
		
		// If it's not generic, return it
		if contentType != "application/octet-stream" {
			return contentType, nil
		}
	}

	// Common file type mappings for cases where standard detection fails
	commonTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".bmp":  "image/bmp",
		".svg":  "image/svg+xml",
		".ico":  "image/x-icon",
		".tiff": "image/tiff",
		".tif":  "image/tiff",
		".pdf":  "application/pdf",
		".txt":  "text/plain",
		".md":   "text/markdown",
		".html": "text/html",
		".htm":  "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".xml":  "application/xml",
		".zip":  "application/zip",
		".tar":  "application/x-tar",
		".gz":   "application/gzip",
		".mp4":  "video/mp4",
		".mov":  "video/quicktime",
		".avi":  "video/x-msvideo",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".flac": "audio/flac",
	}

	if contentType, ok := commonTypes[ext]; ok {
		return contentType, nil
	}

	// Default fallback
	return "application/octet-stream", nil
}

// IsImageType checks if the content type represents an image
func IsImageType(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}

// GetFileCategory returns a general category for the content type
func GetFileCategory(contentType string) string {
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return "image"
	case strings.HasPrefix(contentType, "video/"):
		return "video"
	case strings.HasPrefix(contentType, "audio/"):
		return "audio"
	case strings.HasPrefix(contentType, "text/"):
		return "text"
	case strings.Contains(contentType, "pdf"):
		return "document"
	case strings.Contains(contentType, "zip") || strings.Contains(contentType, "tar") || strings.Contains(contentType, "gzip"):
		return "archive"
	default:
		return "other"
	}
}