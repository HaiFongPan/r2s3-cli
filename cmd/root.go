package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/HaiFongPan/r2s3-cli/internal/config"
	"github.com/HaiFongPan/r2s3-cli/internal/r2"
	"github.com/HaiFongPan/r2s3-cli/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	cfgFile      string
	verbose      bool
	quiet        bool
	globalConfig *config.Config
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "r2s3-cli",
	Short: "A simple CLI tool for managing Cloudflare R2 storage",
	Long: `R2S3-CLI is a command line tool for managing files in Cloudflare R2 storage.
It supports uploading, downloading, deleting, and listing files with flexible
configuration management using TOML files, environment variables, and CLI flags.

Example usage:
  r2s3-cli # Interactive file browser 
  r2s3-cli upload image.jpg
  r2s3-cli list photos/
  r2s3-cli delete old-file.jpg
  r2s3-cli preview image.jpg --url`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// When called without subcommands, directly enter interactive file browser
		return runDefaultInteractiveList()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is ~/.r2s3-cli/config.toml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "enable quiet mode")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() error {
	var err error
	globalConfig, err = config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Configure logging
	setupLogging()

	return nil
}

// setupLogging configures the global logger based on config and flags
func setupLogging() {
	// Set log level
	level := globalConfig.Log.Level
	if verbose {
		level = "debug"
	} else if quiet {
		level = "error"
	}

	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logrus.Warnf("Invalid log level %s, using info", level)
		logLevel = logrus.InfoLevel
	}
	logrus.SetLevel(logLevel)

	// Redirect all logs to file to prevent UI interference
	logDir := "/tmp/r2s3-cli"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// Fallback to stderr if can't create log directory
		logrus.Warnf("Failed to create log directory %s: %v", logDir, err)
	} else {
		logFile := filepath.Join(logDir, "app.log")
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			logrus.Warnf("Failed to open log file %s: %v", logFile, err)
		} else {
			logrus.SetOutput(file)
		}
	}

	// Set log format
	if globalConfig.Log.Format == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			DisableTimestamp: quiet,
			FullTimestamp:    verbose,
		})
	}
}

// GetConfig returns the global configuration
func GetConfig() *config.Config {
	return globalConfig
}

// runDefaultInteractiveList runs the interactive file browser with default settings
func runDefaultInteractiveList() error {
	cfg := globalConfig

	// Create R2 client
	client, err := r2.NewClient(&cfg.R2)
	if err != nil {
		return fmt.Errorf("failed to create R2 client: %w", err)
	}

	// Get effective bucket and empty prefix for default list
	effectiveBucket := cfg.GetEffectiveBucket()
	prefix := ""

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

	_, err = program.Run()
	return err
}
