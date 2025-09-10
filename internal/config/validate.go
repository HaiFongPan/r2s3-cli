package config

import (
	"fmt"
	"strings"
)

// Validate validates the configuration and returns an error if invalid
func Validate(config *Config) error {
	if err := validateR2Config(&config.R2); err != nil {
		return fmt.Errorf("R2 config validation failed: %w", err)
	}

	if err := validateLogConfig(&config.Log); err != nil {
		return fmt.Errorf("log config validation failed: %w", err)
	}

	if err := validateGeneralConfig(&config.General); err != nil {
		return fmt.Errorf("general config validation failed: %w", err)
	}

	if err := validateUploadConfig(&config.Upload); err != nil {
		return fmt.Errorf("upload config validation failed: %w", err)
	}

	return nil
}

// validateR2Config validates R2 specific configuration
func validateR2Config(config *R2Config) error {
	if strings.TrimSpace(config.AccountID) == "" {
		return fmt.Errorf("account_id is required")
	}

	if strings.TrimSpace(config.AccessKeyID) == "" {
		return fmt.Errorf("access_key_id is required")
	}

	if strings.TrimSpace(config.AccessKeySecret) == "" {
		return fmt.Errorf("access_key_secret is required")
	}

	if strings.TrimSpace(config.BucketName) == "" {
		return fmt.Errorf("bucket_name is required")
	}

	// Validate bucket name format (simplified S3 bucket name rules)
	if !isValidBucketName(config.BucketName) {
		return fmt.Errorf("invalid bucket_name format: %s", config.BucketName)
	}

	return nil
}

// validateLogConfig validates log configuration
func validateLogConfig(config *LogConfig) error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
		"panic": true,
	}

	level := strings.ToLower(config.Level)
	if !validLevels[level] {
		return fmt.Errorf("invalid log level: %s (valid: debug, info, warn, error, fatal, panic)", config.Level)
	}

	validFormats := map[string]bool{
		"text": true,
		"json": true,
	}

	format := strings.ToLower(config.Format)
	if !validFormats[format] {
		return fmt.Errorf("invalid log format: %s (valid: text, json)", config.Format)
	}

	return nil
}

// validateGeneralConfig validates general configuration
func validateGeneralConfig(config *GeneralConfig) error {
	if config.DefaultTimeout <= 0 {
		return fmt.Errorf("default_timeout must be positive, got: %d", config.DefaultTimeout)
	}

	if config.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be non-negative, got: %d", config.MaxRetries)
	}

	return nil
}

// isValidBucketName checks if the bucket name follows basic S3 naming rules
func isValidBucketName(name string) bool {
	if len(name) < 3 || len(name) > 63 {
		return false
	}

	// Must start and end with letter or number
	if !isAlphaNum(name[0]) || !isAlphaNum(name[len(name)-1]) {
		return false
	}

	// Check each character
	for i, char := range name {
		if !isAlphaNum(byte(char)) && char != '-' && char != '.' {
			return false
		}

		// Cannot have consecutive periods or period-dash combinations
		if i > 0 {
			prev := name[i-1]
			if char == '.' && (prev == '.' || prev == '-') {
				return false
			}
			if char == '-' && prev == '.' {
				return false
			}
		}
	}

	return true
}

// validateUploadConfig validates upload configuration
func validateUploadConfig(config *UploadConfig) error {
	// Upload config fields are all boolean or boolean-like, no specific validation needed
	// Just ensure the config struct is not nil
	if config == nil {
		return fmt.Errorf("upload config cannot be nil")
	}
	return nil
}

// isAlphaNum checks if a byte is alphanumeric
func isAlphaNum(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}
