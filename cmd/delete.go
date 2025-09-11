package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/HaiFongPan/r2s3-cli/internal/r2"
)

var (
	deleteBucket    string
	deleteForce     bool
	deleteRecursive bool
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete <remote-path>",
	Short: "Delete a file from R2 storage",
	Long: `Delete a file from the specified R2 bucket.

Examples:
  r2s3-cli delete image.jpg                  # Delete a single file
  r2s3-cli delete photos/old-image.jpg      # Delete from specific path
  r2s3-cli delete photos/ --recursive       # Delete all files with prefix
  r2s3-cli delete image.jpg --force         # Delete without confirmation`,
	Args: cobra.ExactArgs(1),
	RunE: deleteFile,
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().StringVarP(&deleteBucket, "bucket", "b", "", "bucket name (overrides config)")
	deleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "force delete without confirmation")
	deleteCmd.Flags().BoolVarP(&deleteRecursive, "recursive", "r", false, "delete all files with prefix (use with caution)")
}

func deleteFile(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()

	// Create R2 client
	client, err := r2.NewClient(&cfg.R2)
	if err != nil {
		return fmt.Errorf("failed to create R2 client: %w", err)
	}

	// Determine bucket name with priority: --bucket flag > effective bucket from config
	bucketName := cfg.GetEffectiveBucket()
	if deleteBucket != "" {
		bucketName = deleteBucket
	}

	remotePath := args[0]

	if deleteRecursive {
		return deletePrefix(client, bucketName, remotePath)
	}

	return deleteSingleFile(client, bucketName, remotePath)
}

func deleteSingleFile(client *r2.Client, bucketName, key string) error {
	// Check if file exists first
	exists, err := checkFileExists(client, bucketName, key)
	if err != nil {
		return fmt.Errorf("failed to check if file exists: %w", err)
	}

	if !exists {
		return fmt.Errorf("file %s does not exist", key)
	}

	// Ask for confirmation unless --force is used
	if !deleteForce {
		fmt.Printf("Are you sure you want to delete '%s'? (y/N): ", key)
		var response string
		fmt.Scanln(&response)

		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("Delete cancelled.")
			return nil
		}
	}

	// Delete the file
	logrus.Infof("Deleting file: %s", key)

	_, err = client.GetS3Client().(*s3.Client).DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete file %s: %w", key, err)
	}

	logrus.Infof("Successfully deleted: %s", key)
	return nil
}

func deletePrefix(client *r2.Client, bucketName, prefix string) error {
	// List all files with the prefix
	s3Client := client.GetS3Client().(*s3.Client)

	listInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(prefix),
	}

	result, err := s3Client.ListObjectsV2(context.TODO(), listInput)
	if err != nil {
		return fmt.Errorf("failed to list objects with prefix %s: %w", prefix, err)
	}

	if len(result.Contents) == 0 {
		return fmt.Errorf("no files found with prefix: %s", prefix)
	}

	// Show what will be deleted
	fmt.Printf("The following %d files will be deleted:\n", len(result.Contents))
	for _, obj := range result.Contents {
		fmt.Printf("  - %s\n", aws.ToString(obj.Key))
	}

	// Ask for confirmation unless --force is used
	if !deleteForce {
		fmt.Printf("\nAre you sure you want to delete all these files? This cannot be undone! (y/N): ")
		var response string
		fmt.Scanln(&response)

		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("Delete cancelled.")
			return nil
		}
	}

	// Delete all files
	logrus.Infof("Deleting %d files with prefix: %s", len(result.Contents), prefix)

	deleteErrors := []error{}
	deletedCount := 0

	for _, obj := range result.Contents {
		key := aws.ToString(obj.Key)

		_, err := s3Client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(key),
		})

		if err != nil {
			logrus.Errorf("Failed to delete %s: %v", key, err)
			deleteErrors = append(deleteErrors, fmt.Errorf("failed to delete %s: %w", key, err))
		} else {
			logrus.Debugf("Deleted: %s", key)
			deletedCount++
		}
	}

	// Report results
	if len(deleteErrors) > 0 {
		fmt.Printf("Deleted %d files successfully, %d failed:\n", deletedCount, len(deleteErrors))
		for _, err := range deleteErrors {
			fmt.Printf("  Error: %v\n", err)
		}
		return fmt.Errorf("some files could not be deleted")
	}

	logrus.Infof("Successfully deleted %d files", deletedCount)
	return nil
}
