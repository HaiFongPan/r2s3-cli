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
	Use:   "upload <file-path> [remote-path]",
	Short: "Upload a file to R2 storage",
	Long: `Upload a file to the specified R2 bucket.

Examples:
  r2s3-cli upload image.jpg                    # Upload to root with same name
  r2s3-cli upload image.jpg photos/image.jpg  # Upload to specific path
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

	// Determine bucket name
	bucketName := client.GetBucketName()
	if uploadBucket != "" {
		bucketName = uploadBucket
	}

	filePath := args[0]
	
	// Determine remote path
	var remotePath string
	if len(args) > 1 {
		remotePath = args[1]
	} else {
		remotePath = filepath.Base(filePath)
	}

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
	_, err = client.GetS3Client().PutObject(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	logrus.Infof("Successfully uploaded %s to %s", filePath, remotePath)
	return nil
}

func checkFileExists(client *r2.Client, bucket, key string) (bool, error) {
	_, err := client.GetS3Client().HeadObject(context.TODO(), &s3.HeadObjectInput{
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