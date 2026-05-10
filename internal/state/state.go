package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// InstalledLink represents a single symlink installed by rice.
type InstalledLink struct {
	Source string `json:"source"`           // absolute path to the file in the rice repo
	Target string `json:"target"`           // absolute path to the symlink in $HOME
	IsDir  bool   `json:"is_dir,omitempty"` // true if the source is a directory
}

// PackageState represents the state of a single installed package.
type PackageState struct {
	Profile        string          `json:"profile"`
	InstalledLinks []InstalledLink `json:"installed_links"`
	InstalledAt    time.Time       `json:"installed_at"`
}

// State is the top-level state file structure.
// Key = package name (e.g., "nvim", "ghostty")
type State map[string]PackageState

// DefaultPath returns the platform-appropriate state file path.
// POSIX: ~/.config/easyrice/state.json
// Windows: %APPDATA%/easyrice/state.json
func DefaultPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to home directory if UserConfigDir fails
		homeDir, _ := os.UserHomeDir()
		configDir = filepath.Join(homeDir, ".config")
	}
	return filepath.Join(configDir, "easyrice", "state.json")
}

// Load reads and parses the state file at path.
// If the file does not exist, returns an empty State (not an error).
func Load(path string) (State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{}, nil
		}
		return nil, err
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	return s, nil
}

// Save writes the state to path as pretty-printed JSON.
// Creates parent directories if they don't exist.
func Save(path string, s State) error {
	// Create parent directories if they don't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Marshal to pretty-printed JSON with 2-space indent
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	// Write to file with mode 0644
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}

	return nil
}
