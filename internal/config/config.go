package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the complete application configuration
type Config struct {
	R2      R2Config      `mapstructure:"r2"`
	Log     LogConfig     `mapstructure:"log"`
	General GeneralConfig `mapstructure:"general"`
	Upload  UploadConfig  `mapstructure:"upload"`
	UI      UIConfig      `mapstructure:"ui"`
}

// R2Config holds R2/S3 specific configuration
type R2Config struct {
	AccountID       string   `mapstructure:"account_id"`
	AccessKeyID     string   `mapstructure:"access_key_id"`
	AccessKeySecret string   `mapstructure:"access_key_secret"`
	BucketName      string   `mapstructure:"bucket_name"`
	Endpoint        string   `mapstructure:"endpoint"`
	Region          string   `mapstructure:"region"`
	CustomDomains   []string `mapstructure:"custom_domains"`
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// GeneralConfig holds general application configuration
type GeneralConfig struct {
	DefaultTimeout int    `mapstructure:"default_timeout"`
	MaxRetries     int    `mapstructure:"max_retries"`
	ConfigPath     string `mapstructure:"config_path"`
}

// UploadConfig holds upload-specific configuration
type UploadConfig struct {
	DefaultOverwrite       bool   `mapstructure:"default_overwrite"`
	DefaultPublic          bool   `mapstructure:"default_public"`
	AutoDetectContentType  bool   `mapstructure:"auto_detect_content_type"`
	DefaultCompress        string `mapstructure:"default_compress"`
}

// UIConfig holds user interface configuration
type UIConfig struct {
	InteractiveMode      bool   `mapstructure:"interactive_mode"`
	ImagePreviewMethod   string `mapstructure:"image_preview_method"`
}

// Load loads configuration from multiple sources with priority:
// 1. Command line flags (highest)
// 2. Environment variables
// 3. Configuration file
// 4. Defaults (lowest)
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Set environment variable prefix
	v.SetEnvPrefix("R2CLI")
	v.AutomaticEnv()

	// Environment variable mappings
	v.BindEnv("r2.account_id", "R2CLI_ACCOUNT_ID")
	v.BindEnv("r2.access_key_id", "R2CLI_ACCESS_KEY_ID")
	v.BindEnv("r2.access_key_secret", "R2CLI_ACCESS_KEY_SECRET")
	v.BindEnv("r2.bucket_name", "R2CLI_BUCKET_NAME")
	v.BindEnv("r2.endpoint", "R2CLI_ENDPOINT")
	v.BindEnv("r2.region", "R2CLI_REGION")
	v.BindEnv("r2.custom_domains", "R2CLI_CUSTOM_DOMAINS")
	v.BindEnv("log.level", "R2CLI_LOG_LEVEL")
	v.BindEnv("log.format", "R2CLI_LOG_FORMAT")
	v.BindEnv("upload.default_overwrite", "R2CLI_UPLOAD_DEFAULT_OVERWRITE")
	v.BindEnv("upload.default_public", "R2CLI_UPLOAD_DEFAULT_PUBLIC")
	v.BindEnv("upload.auto_detect_content_type", "R2CLI_UPLOAD_AUTO_DETECT_CONTENT_TYPE")
	v.BindEnv("upload.default_compress", "R2CLI_UPLOAD_DEFAULT_COMPRESS")
	v.BindEnv("ui.interactive_mode", "R2CLI_UI_INTERACTIVE_MODE")
	v.BindEnv("ui.image_preview_method", "R2CLI_UI_IMAGE_PREVIEW_METHOD")

	// Configuration file handling
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Look for config in common locations
		v.SetConfigName("config")
		v.SetConfigType("toml")
		
		// Add config search paths
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.r2s3-cli")
		v.AddConfigPath("/etc/r2s3-cli/")
	}

	// Read configuration file if it exists
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is not an error - we can use defaults and env vars
	}

	// Unmarshal configuration
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := Validate(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// setDefaults sets default values for configuration
func setDefaults(v *viper.Viper) {
	// R2 defaults
	v.SetDefault("r2.endpoint", "auto")
	v.SetDefault("r2.region", "auto")

	// Log defaults
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")

	// General defaults
	v.SetDefault("general.default_timeout", 30)
	v.SetDefault("general.max_retries", 3)

	// Upload defaults
	v.SetDefault("upload.default_overwrite", false)
	v.SetDefault("upload.default_public", false)
	v.SetDefault("upload.auto_detect_content_type", true)
	v.SetDefault("upload.default_compress", "")

	// UI defaults
	v.SetDefault("ui.interactive_mode", true)
	v.SetDefault("ui.image_preview_method", "auto")
}

// GetDefaultConfigPath returns the default configuration file path
func GetDefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./config.toml"
	}
	return filepath.Join(homeDir, ".r2s3-cli", "config.toml")
}

// EnsureConfigDir creates the configuration directory if it doesn't exist
func EnsureConfigDir() error {
	configPath := GetDefaultConfigPath()
	dir := filepath.Dir(configPath)
	return os.MkdirAll(dir, 0700)
}