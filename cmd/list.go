package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/HaiFongPan/r2s3-cli/internal/config"
	"github.com/HaiFongPan/r2s3-cli/internal/r2"
	"github.com/HaiFongPan/r2s3-cli/internal/tui"
)

var (
	listBucket string
	listLimit  int64
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
  r2s3-cli list --interactive=false # Show table output`,
	RunE: listFiles,
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVarP(&listBucket, "bucket", "b", "", "bucket name (overrides config)")
	listCmd.Flags().Int64VarP(&listLimit, "limit", "l", 1000, "maximum number of files to list")
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
	// Launch interactive mode if requested
	return runInteractiveBrowser(client, cfg, bucketName, prefix)
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
