package utils

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// ProgressReader wraps an io.Reader and displays upload progress
type ProgressReader struct {
	reader      io.Reader
	total       int64
	read        int64
	description string
	startTime   time.Time
	lastPrint   time.Time
	finished    bool
	lastLineLen int
}

// NewProgressReader creates a new progress reader
func NewProgressReader(reader io.Reader, total int64, description string) *ProgressReader {
	return &ProgressReader{
		reader:      reader,
		total:       total,
		description: description,
		startTime:   time.Now(),
		lastPrint:   time.Now(),
	}
}

// Read implements io.Reader interface and shows progress
func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)

	if n > 0 {
		pr.read += int64(n)

		// Update progress every 200ms or on EOF/error
		now := time.Now()
		if now.Sub(pr.lastPrint) > 200*time.Millisecond || err != nil {
			pr.printProgress()
			pr.lastPrint = now
		}
	}

	// Mark as finished when we reach the end
	// if err == io.EOF || pr.read >= pr.total {
	if err == io.EOF {
		pr.finished = true
		pr.printProgress()      // Final progress display
		fmt.Fprintln(os.Stderr) // New line after completion
	}

	return n, err
}

// Seek implements io.Seeker interface for AWS SDK retry support
func (pr *ProgressReader) Seek(offset int64, whence int) (int64, error) {
	if seeker, ok := pr.reader.(io.Seeker); ok {
		pos, err := seeker.Seek(offset, whence)
		if err == nil {
			pr.read = pos
		}
		return pos, err
	}
	return 0, fmt.Errorf("underlying reader does not support seeking")
}

// printProgress displays the current progress
func (pr *ProgressReader) printProgress() {
	if pr.total <= 0 {
		return
	}

	percentage := float64(pr.read) / float64(pr.total) * 100

	// Calculate transfer speed
	elapsed := time.Since(pr.startTime)
	var speed string
	if elapsed.Seconds() > 0.1 {
		bytesPerSec := float64(pr.read) / elapsed.Seconds()
		speed = fmt.Sprintf(" %s/s", formatBytes(int64(bytesPerSec)))
	}

	// Create progress bar (40 characters wide)
	barWidth := 40
	filled := int(percentage * float64(barWidth) / 100)
	if filled > barWidth {
		filled = barWidth
	}

	bar := "[" + strings.Repeat("=", filled) + strings.Repeat(" ", barWidth-filled) + "]"

	// Build the progress line
	line := fmt.Sprintf("%s %s %.1f%% (%s/%s)%s",
		pr.description,
		bar,
		percentage,
		formatBytes(pr.read),
		formatBytes(pr.total),
		speed)

	// Clear previous line if it was longer
	if pr.lastLineLen > len(line) {
		// Move cursor back to beginning and clear the line
		fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", pr.lastLineLen))
	}

	// Print the new progress line
	fmt.Fprintf(os.Stderr, "\r%s", line)
	pr.lastLineLen = len(line)
}

// Close finishes the progress display
func (pr *ProgressReader) Close() error {
	// Only print newline if we haven't already finished
	if !pr.finished {
		pr.printProgress()
		fmt.Fprintln(os.Stderr) // New line after completion
	}
	return nil
}

// formatBytes formats bytes in human readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
