package xdgpath

import (
	"os"
	"path/filepath"
	"runtime"
)

// ConfigDir returns the platform-appropriate base configuration directory.
// POSIX: $XDG_CONFIG_HOME if set and absolute, else ~/.config
// Windows: %APPDATA% (via os.UserConfigDir)
// The returned path is always absolute and never includes "easyrice".
// Callers must append "easyrice" (and any sub-path) themselves.
func ConfigDir() string {
	if runtime.GOOS == "windows" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			homeDir, _ := os.UserHomeDir()
			if homeDir != "" {
				return filepath.Join(homeDir, "AppData", "Roaming")
			}
			// Last-ditch: both UserConfigDir and UserHomeDir failed
			return `C:\Users\Default\AppData\Roaming`
		}
		return configDir
	}

	// POSIX: check XDG_CONFIG_HOME first
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg != "" && filepath.IsAbs(xdg) {
		return xdg
	}

	// Fall back to ~/.config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Last-ditch fallback: return /.config (still absolute)
		return "/.config"
	}
	return filepath.Join(homeDir, ".config")
}
