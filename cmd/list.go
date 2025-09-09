package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/HaiFongPan/r2s3-cli/internal/config"
	"github.com/HaiFongPan/r2s3-cli/internal/r2"
	"github.com/HaiFongPan/r2s3-cli/internal/tui"
)

var (
	listBucket      string
	listLimit       int64
	showSize        bool
	showDate        bool
	listInteractive bool
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list [prefix]",
	Short: "List files in the R2 bucket",
	Long: `List files in the specified R2 bucket with optional prefix filtering.
By default, launches interactive TUI browser. Use --interactive=false for table output.

Examples:
  r2s3-cli list                    # Launch interactive browser
  r2s3-cli list photos/            # Browse files with 'photos/' prefix  
  r2s3-cli list --interactive=false # Show table output
  r2s3-cli list --size --date      # Show file sizes and dates (table mode)`,
	RunE: listFiles,
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVarP(&listBucket, "bucket", "b", "", "bucket name (overrides config)")
	listCmd.Flags().Int64VarP(&listLimit, "limit", "l", 1000, "maximum number of files to list")
	listCmd.Flags().BoolVar(&showSize, "size", true, "show file sizes")
	listCmd.Flags().BoolVar(&showDate, "date", true, "show modification dates")
	listCmd.Flags().BoolVarP(&listInteractive, "interactive", "i", false, "launch interactive browser")
}

func listFiles(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()

	// Determine bucket name with priority: --bucket flag > effective bucket from config
	bucketName := cfg.GetEffectiveBucket()
	if listBucket != "" {
		bucketName = listBucket
	}

	// Determine prefix
	var prefix string
	if len(args) > 0 {
		prefix = args[0]
	}

	// Create R2 client
	client, err := r2.NewClient(&cfg.R2)
	if err != nil {
		return fmt.Errorf("failed to create R2 client: %w", err)
	}

	// Determine if we should use interactive mode
	useInteractive := listInteractive
	if !cmd.Flags().Changed("interactive") {
		useInteractive = cfg.UI.InteractiveMode
	}

	logrus.Debugf("Interactive mode: flag=%t, config=%t, using=%t",
		listInteractive, cfg.UI.InteractiveMode, useInteractive)

	// Launch interactive mode if requested
	if useInteractive {
		return runInteractiveBrowser(client, cfg, bucketName, prefix)
	}

	// Non-interactive mode - existing functionality
	return runNonInteractiveList(client, bucketName, prefix)
}

func runInteractiveBrowser(client *r2.Client, cfg *config.Config, bucketName, prefix string) error {
	// Use the effective bucket from config (which includes any temp bucket changes)
	effectiveBucket := cfg.GetEffectiveBucket()
	
	// Create model
	model := tui.NewFileBrowserModel(client, cfg, effectiveBucket, prefix)
	
	// Launch interactive browser with bubbletea
	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Set program reference in model for direct messaging
	model.SetProgram(program)

	_, err := program.Run()
	return err
}

func runNonInteractiveList(client *r2.Client, bucketName, prefix string) error {
	// List objects
	logrus.Debugf("Listing objects in bucket %s with prefix %s", bucketName, prefix)

	s3Client := client.GetS3Client()
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucketName),
		MaxKeys: aws.Int32(int32(listLimit)),
	}

	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	result, err := s3Client.ListObjectsV2(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
	}

	// Output results in table format
	return outputTable(result.Contents)
}

func outputTable(objects []types.Object) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Header
	header := "NAME"
	if showSize {
		header += "\tSIZE"
	}
	if showDate {
		header += "\tMODIFIED"
	}
	fmt.Fprintln(w, header)

	// Objects
	for _, obj := range objects {
		line := aws.ToString(obj.Key)

		if showSize {
			line += fmt.Sprintf("\t%s", formatFileSize(aws.ToInt64(obj.Size)))
		}

		if showDate {
			line += fmt.Sprintf("\t%s", obj.LastModified.Format(time.RFC3339))
		}

		fmt.Fprintln(w, line)
	}

	return w.Flush()
}

// formatFileSize formats file size in human readable format
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%dB", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(size)/float64(div), "KMGTPE"[exp])
}
