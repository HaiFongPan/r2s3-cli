package utils

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sirupsen/logrus"
)

type FileDownloader struct {
	s3Client   *s3.Client
	bucketName string
}

func NewFileDownloader(s3Client *s3.Client, bucketName string) *FileDownloader {
	return &FileDownloader{
		s3Client:   s3Client,
		bucketName: bucketName,
	}
}

func (d *FileDownloader) DownloadFile(key string) error {
	// Get user's Downloads directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	downloadsDir := filepath.Join(homeDir, "Downloads")

	// Create Downloads directory if it doesn't exist
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		return fmt.Errorf("failed to create downloads directory: %w", err)
	}

	// Get the filename from the key
	filename := filepath.Base(key)
	localPath := filepath.Join(downloadsDir, filename)

	// Handle file name conflicts
	localPath = d.resolveFileNameConflict(localPath)

	// Download the file from S3
	result, err := d.s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(d.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to get object from S3: %w", err)
	}
	defer result.Body.Close()

	// Create the local file
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer file.Close()

	// Copy the content
	_, err = io.Copy(file, result.Body)
	if err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	logrus.Infof("File downloaded successfully to: %s", localPath)
	return nil
}

func (d *FileDownloader) resolveFileNameConflict(originalPath string) string {
	if _, err := os.Stat(originalPath); os.IsNotExist(err) {
		// File doesn't exist, use original path
		return originalPath
	}

	// File exists, find an alternative name
	ext := filepath.Ext(originalPath)
	baseName := originalPath[:len(originalPath)-len(ext)]

	for i := 1; i < 1000; i++ {
		newPath := fmt.Sprintf("%s (%d)%s", baseName, i, ext)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
	}

	// If we can't find a unique name after 999 attempts, use timestamp
	return fmt.Sprintf("%s_%d%s", baseName, os.Getpid(), ext)
}

func (d *FileDownloader) GetDownloadPath(key string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	downloadsDir := filepath.Join(homeDir, "Downloads")
	filename := filepath.Base(key)
	return filepath.Join(downloadsDir, filename)
}

func (d *FileDownloader) CheckFileExists(key string) bool {
	path := d.GetDownloadPath(key)
	if path == "" {
		return false
	}

	_, err := os.Stat(path)
	return err == nil
}

// DEPRECATED: TUIProgressReader - replaced by CallbackProgressReader
// TUIProgressReader wraps an io.Reader and sends progress updates for TUI
// type TUIProgressReader struct {
// 	io.Reader
// 	total      int64
// 	read       int64
// 	progressCh chan tea.Msg
// }
//
// func (pr *TUIProgressReader) Read(p []byte) (n int, err error) {
// 	n, err = pr.Reader.Read(p)
// 	if n > 0 {
// 		pr.read += int64(n)
//
// 		if pr.total > 0 {
// 			progress := float64(pr.read) / float64(pr.total)
// 			// Ensure progress doesn't exceed 1.0
// 			if progress > 1.0 {
// 				progress = 1.0
// 			}
//
// 			logrus.Debugf("TUIProgressReader: read=%d, total=%d, progress=%.4f", pr.read, pr.total, progress)
//
// 			// Send progress update non-blocking
// 			select {
// 			case pr.progressCh <- DownloadProgressMsg{Progress: progress}:
// 				logrus.Debugf("TUIProgressReader: sent progress update: %.2f", progress)
// 			default:
// 				logrus.Debug("TUIProgressReader: progress channel full, skipping update")
// 			}
// 		} else {
// 			logrus.Warnf("TUIProgressReader: total is 0, cannot calculate progress (read=%d)", pr.read)
// 		}
// 	}
//
// 	return n, err
// }

type DownloadProgressMsg struct {
	Progress float64
}

type DownloadStartedMsg struct {
	Filename string
}

type DownloadCompletedMsg struct {
	Err error
}

// DEPRECATED: DownloadFileWithProgress - replaced by DownloadFileWithProgressCallback
// func (d *FileDownloader) DownloadFileWithProgress(ctx context.Context, key string, progressCh chan tea.Msg) error {
// 	// Don't close the channel here - let the caller handle it
//
// 	// Get user's Downloads directory
// 	homeDir, err := os.UserHomeDir()
// 	if err != nil {
// 		progressCh <- DownloadCompletedMsg{Err: fmt.Errorf("failed to get user home directory: %w", err)}
// 		return fmt.Errorf("failed to get user home directory: %w", err)
// 	}
//
// 	downloadsDir := filepath.Join(homeDir, "Downloads")
//
// 	// Create Downloads directory if it doesn't exist
// 	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
// 		progressCh <- DownloadCompletedMsg{Err: fmt.Errorf("failed to create downloads directory: %w", err)}
// 		return fmt.Errorf("failed to create downloads directory: %w", err)
// 	}
//
// 	// Get the filename from the key
// 	filename := filepath.Base(key)
// 	localPath := filepath.Join(downloadsDir, filename)
//
// 	// Handle file name conflicts
// 	localPath = d.resolveFileNameConflict(localPath)
//
// 	// Send download started message
// 	logrus.Infof("DownloadFileWithProgress: sending DownloadStartedMsg for %s", filename)
// 	progressCh <- DownloadStartedMsg{Filename: filename}
//
// 	// Get object info first to get content length
// 	headResult, err := d.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
// 		Bucket: aws.String(d.bucketName),
// 		Key:    aws.String(key),
// 	})
// 	if err != nil {
// 		progressCh <- DownloadCompletedMsg{Err: fmt.Errorf("failed to get object info: %w", err)}
// 		return fmt.Errorf("failed to get object info: %w", err)
// 	}
//
// 	// Download the file from S3
// 	result, err := d.s3Client.GetObject(ctx, &s3.GetObjectInput{
// 		Bucket: aws.String(d.bucketName),
// 		Key:    aws.String(key),
// 	})
// 	if err != nil {
// 		progressCh <- DownloadCompletedMsg{Err: fmt.Errorf("failed to get object from S3: %w", err)}
// 		return fmt.Errorf("failed to get object from S3: %w", err)
// 	}
// 	defer result.Body.Close()
//
// 	// Create progress reader
// 	contentLength := aws.ToInt64(headResult.ContentLength)
// 	logrus.Infof("DownloadFileWithProgress: content length: %d bytes", contentLength)
// 	progressReader := &TUIProgressReader{
// 		Reader:     result.Body,
// 		total:      contentLength,
// 		progressCh: progressCh,
// 	}
//
// 	// Create the local file
// 	file, err := os.Create(localPath)
// 	if err != nil {
// 		progressCh <- DownloadCompletedMsg{Err: fmt.Errorf("failed to create local file: %w", err)}
// 		return fmt.Errorf("failed to create local file: %w", err)
// 	}
// 	defer file.Close()
//
// 	// Copy the content with progress tracking
// 	_, err = io.Copy(file, progressReader)
// 	if err != nil {
// 		progressCh <- DownloadCompletedMsg{Err: fmt.Errorf("failed to write file content: %w", err)}
// 		return fmt.Errorf("failed to write file content: %w", err)
// 	}
//
// 	// Send completion message
// 	logrus.Info("DownloadFileWithProgress: sending DownloadCompletedMsg with success")
// 	progressCh <- DownloadCompletedMsg{Err: nil}
// 	logrus.Infof("File downloaded successfully to: %s", localPath)
// 	return nil
// }

// DownloadFileWithProgressCallback downloads a file with progress updates via callback
func (d *FileDownloader) DownloadFileWithProgressCallback(ctx context.Context, key string, callback func(tea.Msg)) error {
	// Get user's Downloads directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	downloadsDir := filepath.Join(homeDir, "Downloads")

	// Create Downloads directory if it doesn't exist
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		return fmt.Errorf("failed to create downloads directory: %w", err)
	}

	// Get the filename from the key
	filename := filepath.Base(key)
	localPath := filepath.Join(downloadsDir, filename)

	// Handle file name conflicts
	localPath = d.resolveFileNameConflict(localPath)

	// Get object info first to get content length
	headResult, err := d.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(d.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to get object info: %w", err)
	}

	// Download the file from S3
	result, err := d.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(d.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to get object from S3: %w", err)
	}
	defer result.Body.Close()

	// Create progress reader with callback
	contentLength := aws.ToInt64(headResult.ContentLength)
	// logrus.Infof("DownloadFileWithProgressCallback: content length: %d bytes", contentLength)
	progressReader := &CallbackProgressReader{
		Reader:   result.Body,
		total:    contentLength,
		callback: callback,
	}

	// Create the local file
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer file.Close()

	// Copy the content with progress tracking
	_, err = io.Copy(file, progressReader)
	if err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	logrus.Infof("File downloaded successfully to: %s", localPath)
	return nil
}

// CallbackProgressReader wraps an io.Reader and calls a callback for progress updates
type CallbackProgressReader struct {
	io.Reader
	total    int64
	read     int64
	callback func(tea.Msg)
	lastSent float64
}

func (pr *CallbackProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.Reader.Read(p)
	if n > 0 {
		pr.read += int64(n)

		if pr.total > 0 {
			progress := float64(pr.read) / float64(pr.total)
			// Ensure progress doesn't exceed 1.0
			if progress > 1.0 {
				progress = 1.0
			}

			// Throttle progress messages - only send if progress changed significantly
			if progress-pr.lastSent >= 0.05 || progress >= 1.0 {
				logrus.Debugf("CallbackProgressReader: sending progress: %.2f", progress)
				pr.callback(DownloadProgressMsg{Progress: progress})
				pr.lastSent = progress
			}
		} else {
			logrus.Warnf("CallbackProgressReader: total is 0, cannot calculate progress (read=%d)", pr.read)
		}
	}

	return n, err
}
