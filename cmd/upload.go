package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/disintegration/imaging"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/HaiFongPan/r2s3-cli/internal/config"
	"github.com/HaiFongPan/r2s3-cli/internal/r2"
	"github.com/HaiFongPan/r2s3-cli/internal/utils"
)

var (
	uploadBucket      string
	uploadPublic      bool
	uploadContentType string
	uploadOverwrite   bool
	uploadCompress    string
	uploadNoProgress  bool
)

// uploadCmd represents the upload command
var uploadCmd = &cobra.Command{
	Use:   "upload <file-path|folder-path> [remote-path]",
	Short: "Upload a file or folder to R2 storage",
	Long: `Upload a file or folder to the specified R2 bucket.

Examples:
  r2s3-cli upload image.jpg                    # Upload file to root with same name
  r2s3-cli upload image.jpg photos/image.jpg  # Upload file to specific path
  r2s3-cli upload ./photos                    # Upload entire folder recursively
  r2s3-cli upload ./photos images/            # Upload folder to specific path
  r2s3-cli upload image.jpg --public          # Upload with public access
  r2s3-cli upload image.jpg --compress high   # Upload with high compression
  r2s3-cli upload image.jpg --no-progress     # Upload without progress bar
  r2s3-cli upload *.jpg photos/               # Upload multiple files`,
	Args: cobra.MinimumNArgs(1),
	RunE: uploadFile,
}

func init() {
	rootCmd.AddCommand(uploadCmd)

	uploadCmd.Flags().StringVarP(&uploadBucket, "bucket", "b", "", "bucket name (overrides config)")
	uploadCmd.Flags().BoolVar(&uploadPublic, "public", false, "make file publicly accessible")
	uploadCmd.Flags().StringVarP(&uploadContentType, "content-type", "t", "", "specify content type")
	uploadCmd.Flags().BoolVar(&uploadOverwrite, "overwrite", false, "overwrite existing files")
	uploadCmd.Flags().StringVarP(&uploadCompress, "compress", "z", "", "image compression level (high, fine, normal, low)")
	uploadCmd.Flags().BoolVar(&uploadNoProgress, "no-progress", false, "disable progress bar")
}

func uploadFile(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()

	// Create R2 client
	client, err := r2.NewClient(&cfg.R2)
	if err != nil {
		return fmt.Errorf("failed to create R2 client: %w", err)
	}

	// Determine bucket name with priority: --bucket flag > effective bucket from config
	bucketName := cfg.GetEffectiveBucket()
	if uploadBucket != "" {
		bucketName = uploadBucket
	}

	localPath := args[0]

	// Check if it's a directory
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("failed to access %s: %w", localPath, err)
	}

	// Determine remote path
	var remotePath string
	if len(args) > 1 {
		remotePath = args[1]
	} else {
		if fileInfo.IsDir() {
			remotePath = filepath.Base(localPath) + "/"
		} else {
			remotePath = filepath.Base(localPath)
		}
	}

	if fileInfo.IsDir() {
		return uploadDirectory(client, bucketName, localPath, remotePath, cfg, cmd)
	} else {
		return uploadSingleFile(client, bucketName, localPath, remotePath, cfg, cmd)
	}
}

func checkFileExists(client *r2.Client, bucket, key string) (bool, error) {
	_, err := client.GetS3Client().(*s3.Client).HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		// Check for various "not found" error types
		var nsk *types.NoSuchKey
		var nf *types.NotFound
		if errors.As(err, &nsk) || errors.As(err, &nf) {
			return false, nil
		}

		// Also check for HTTP 404 status code in error message
		if strings.Contains(err.Error(), "StatusCode: 404") || strings.Contains(err.Error(), "NotFound") {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// isImageFile checks if the file is an image based on extension
func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".bmp" || ext == ".tiff" || ext == ".webp"
}

// compressImage compresses an image based on the specified quality level
func compressImage(file *os.File, quality string) ([]byte, int64, error) {
	// Reset file position
	file.Seek(0, 0)

	// Decode image
	img, format, err := image.Decode(file)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to decode image: %w", err)
	}

	// Determine JPEG quality based on compression level
	var jpegQuality int
	switch quality {
	case "high":
		jpegQuality = 95
	case "fine":
		jpegQuality = 85
	case "normal":
		jpegQuality = 75
	case "low":
		jpegQuality = 60
	default:
		return nil, 0, fmt.Errorf("invalid compression level: %s (use: high, fine, normal, low)", quality)
	}

	// Create buffer for compressed image
	var buf bytes.Buffer

	// Encode based on original format, but convert to JPEG for compression
	switch format {
	case "jpeg":
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegQuality})
	case "png":
		// Convert PNG to JPEG for better compression
		// Create new image without alpha channel for JPEG compatibility
		bounds := img.Bounds()
		rgbImg := image.NewRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				rgbImg.Set(x, y, img.At(x, y))
			}
		}
		err = jpeg.Encode(&buf, rgbImg, &jpeg.Options{Quality: jpegQuality})
	default:
		// For other formats, try to resize and encode as JPEG
		resized := imaging.Fit(img, 1920, 1920, imaging.Lanczos) // Resize to max 1920x1920
		err = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: jpegQuality})
	}

	if err != nil {
		return nil, 0, fmt.Errorf("failed to encode compressed image: %w", err)
	}

	compressedData := buf.Bytes()
	return compressedData, int64(len(compressedData)), nil
}

// uploadSingleFile uploads a single file
func uploadSingleFile(client *r2.Client, bucketName, filePath, remotePath string, cfg *config.Config, cmd *cobra.Command) error {
	// Determine overwrite behavior (CLI flag > config > default)
	shouldOverwrite := uploadOverwrite
	if !cmd.Flags().Changed("overwrite") {
		shouldOverwrite = cfg.Upload.DefaultOverwrite
	}

	// Determine compression level (CLI flag > config > default)
	compressionLevel := uploadCompress
	if !cmd.Flags().Changed("compress") {
		compressionLevel = cfg.Upload.DefaultCompress
	}

	// Check if file exists
	if !shouldOverwrite {
		exists, err := checkFileExists(client, bucketName, remotePath)
		if err != nil {
			return fmt.Errorf("failed to check if file exists: %w", err)
		}
		if exists {
			return fmt.Errorf("file %s already exists (use --overwrite to replace)", remotePath)
		}
	}

	// Open local file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Handle image compression if requested
	var uploadBody io.Reader = file
	var finalSize int64 = fileInfo.Size()
	if compressionLevel != "" && isImageFile(filePath) {
		compressedData, compressedSize, err := compressImage(file, compressionLevel)
		if err != nil {
			return fmt.Errorf("failed to compress image: %w", err)
		}
		uploadBody = bytes.NewReader(compressedData)
		finalSize = compressedSize
		logrus.Infof("Compressed image from %d bytes to %d bytes (%.1f%% reduction)",
			fileInfo.Size(), compressedSize, float64(fileInfo.Size()-compressedSize)/float64(fileInfo.Size())*100)
	}

	// If not compressing, reset file position
	if compressionLevel == "" || !isImageFile(filePath) {
		file.Seek(0, 0)
	}

	logrus.Infof("Uploading %s (%d bytes) to %s", filePath, finalSize, remotePath)

	// Determine content type
	var contentType string
	if uploadContentType != "" {
		// Use explicitly specified content type
		contentType = uploadContentType
	} else if cfg.Upload.AutoDetectContentType {
		// Auto-detect content type
		detectedType, err := utils.DetectContentType(filePath, file)
		if err != nil {
			logrus.Warnf("Failed to detect content type: %v", err)
			contentType = "application/octet-stream"
		} else {
			contentType = detectedType
		}
		// Reset file position after reading for content type detection (only if not compressing)
		if compressionLevel == "" || !isImageFile(filePath) {
			file.Seek(0, 0)
		}
	}

	// Reset upload body position before wrapping with progress bar
	if seeker, ok := uploadBody.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	}

	// Wrap upload body with progress bar (if enabled, not in quiet mode, and file is large enough)
	if !uploadNoProgress && !quiet && finalSize > 1024*100 { // Show progress bar for files larger than 100KB
		description := fmt.Sprintf("Uploading %s", filepath.Base(filePath))
		progressReader := utils.NewProgressReader(uploadBody, finalSize, description)
		uploadBody = progressReader
		defer progressReader.Close()
	}

	// Determine public access (CLI flag > config > default)
	shouldMakePublic := uploadPublic
	if !cmd.Flags().Changed("public") {
		shouldMakePublic = cfg.Upload.DefaultPublic
	}

	// Prepare upload input
	input := &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(remotePath),
		Body:   uploadBody,
	}

	// Set content type if determined
	if contentType != "" {
		input.ContentType = aws.String(contentType)
		logrus.Debugf("Setting content type: %s", contentType)
	}

	// Set ACL if public
	if shouldMakePublic {
		input.ACL = "public-read"
		logrus.Debugf("Setting public access")
	}

	// Upload file
	_, err = client.GetS3Client().(*s3.Client).PutObject(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	logrus.Infof("Successfully uploaded %s to %s", filePath, remotePath)
	return nil
}

// uploadDirectory uploads an entire directory recursively
func uploadDirectory(client *r2.Client, bucketName, localPath, remotePath string, cfg *config.Config, cmd *cobra.Command) error {
	// Ensure remote path ends with /
	if !strings.HasSuffix(remotePath, "/") {
		remotePath += "/"
	}

	// Collect all files to upload
	var filesToUpload []struct {
		localPath  string
		remotePath string
	}

	err := filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logrus.Warnf("Error accessing %s: %v", path, err)
			return nil // Continue walking
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Calculate relative path from the source directory
		relPath, err := filepath.Rel(localPath, path)
		if err != nil {
			logrus.Warnf("Error calculating relative path for %s: %v", path, err)
			return nil
		}

		// Convert file separators to forward slashes for S3
		relPath = filepath.ToSlash(relPath)

		// Build remote path
		remoteFilePath := remotePath + relPath

		filesToUpload = append(filesToUpload, struct {
			localPath  string
			remotePath string
		}{
			localPath:  path,
			remotePath: remoteFilePath,
		})

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory %s: %w", localPath, err)
	}

	if len(filesToUpload) == 0 {
		logrus.Infof("No files found in directory %s", localPath)
		return nil
	}

	logrus.Infof("Found %d files to upload from directory %s", len(filesToUpload), localPath)

	// Calculate total size for progress tracking
	var totalBytes int64
	for _, file := range filesToUpload {
		if fileInfo, err := os.Stat(file.localPath); err == nil {
			totalBytes += fileInfo.Size()
		}
	}

	// Create multi-file progress tracker
	var progress *utils.MultiFileProgress
	if !uploadNoProgress && !quiet {
		progress = utils.NewMultiFileProgress(len(filesToUpload), totalBytes)
	}

	// Upload files with progress tracking
	uploadedCount := 0
	errorCount := 0

	for _, file := range filesToUpload {
		// Get file size for progress tracking
		var fileSize int64
		if fileInfo, err := os.Stat(file.localPath); err == nil {
			fileSize = fileInfo.Size()
		}

		// Update progress - start file
		if progress != nil {
			progress.StartFile(filepath.Base(file.localPath), fileSize)
		}

		err := uploadSingleFile(client, bucketName, file.localPath, file.remotePath, cfg, cmd)
		if err != nil {
			logrus.Errorf("Failed to upload %s: %v", file.localPath, err)
			errorCount++
			continue
		}
		uploadedCount++

		// Update progress - finish file
		if progress != nil {
			progress.FinishFile(fileSize)
		}
	}

	// Finish progress display
	if progress != nil {
		progress.Finish()
	}

	if errorCount > 0 {
		logrus.Warnf("Directory upload completed with errors: %d succeeded, %d failed", uploadedCount, errorCount)
		return fmt.Errorf("%d files failed to upload", errorCount)
	}

	logrus.Infof("Successfully uploaded directory %s: %d files uploaded to %s", localPath, uploadedCount, remotePath)
	return nil
}
