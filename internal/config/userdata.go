package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// UserData holds user-specific settings that are stored locally
type UserData struct {
	MainBucket string    `json:"main_bucket"`
	LastUsed   string    `json:"last_used"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// LoadUserData loads user data from the user.data file in the executable directory
func LoadUserData() (*UserData, error) {
	userDataPath, err := getUserDataPath()
	if err != nil {
		return createDefaultUserData(), nil
	}

	// Check if file exists
	if _, err := os.Stat(userDataPath); os.IsNotExist(err) {
		// File doesn't exist, return default
		return createDefaultUserData(), nil
	}

	// Read the file
	data, err := os.ReadFile(userDataPath)
	if err != nil {
		return createDefaultUserData(), nil
	}

	// Parse JSON
	var userData UserData
	if err := json.Unmarshal(data, &userData); err != nil {
		// Invalid JSON, return default
		return createDefaultUserData(), nil
	}

	return &userData, nil
}

// SaveUserData saves user data to the user.data file in the executable directory
func (ud *UserData) SaveUserData() error {
	userDataPath, err := getUserDataPath()
	if err != nil {
		return err
	}

	// Update timestamp
	ud.UpdatedAt = time.Now()
	if ud.CreatedAt.IsZero() {
		ud.CreatedAt = ud.UpdatedAt
	}

	// Convert to JSON
	data, err := json.MarshalIndent(ud, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(userDataPath, data, 0644)
}

// SetMainBucket sets the main bucket and saves to file
func (ud *UserData) SetMainBucket(bucket string) error {
	ud.MainBucket = bucket
	// Clear LastUsed when setting MainBucket so MainBucket takes priority
	ud.LastUsed = ""
	return ud.SaveUserData()
}

// SetLastUsed sets the last used bucket and saves to file
func (ud *UserData) SetLastUsed(bucket string) error {
	ud.LastUsed = bucket
	return ud.SaveUserData()
}

// createDefaultUserData creates a new UserData with default values
func createDefaultUserData() *UserData {
	now := time.Now()
	return &UserData{
		MainBucket: "",
		LastUsed:   "",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// getUserDataPath returns the path to the user.data file
func getUserDataPath() (string, error) {
	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Use the same directory as config file
	configDir := filepath.Join(homeDir, ".r2s3-cli")
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "user.data"), nil
}