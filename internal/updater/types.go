package updater

import (
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Release represents a GitHub release.
type Release struct {
	Version  string
	URL      string
	AssetURL string
}

// CheckResult represents the result of a version check.
type CheckResult struct {
	Current         string
	Latest          string
	UpdateAvailable bool
	CheckedAt       time.Time
}

// Options configures an Updater.
type Options struct {
	Owner      string
	Repo       string
	Timeout    time.Duration
	CacheDir   string
	HTTPClient *http.Client
}

// DefaultCacheDir returns the platform-appropriate cache directory for updates.
// POSIX: ~/.config/easyrice/update-check.json
// Windows: %APPDATA%/easyrice/update-check.json
func DefaultCacheDir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		homeDir, _ := os.UserHomeDir()
		configDir = filepath.Join(homeDir, ".config")
	}
	return filepath.Join(configDir, "easyrice")
}
