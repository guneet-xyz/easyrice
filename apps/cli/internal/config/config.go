package config

import (
	"os"
	"path/filepath"
)

const (
	appName     = "easyrice"
	riceDirName = "rice"
	tomlFile    = "easyrice.toml"
)

// Dir returns the easyrice config directory (~/.config/easyrice/).
// Respects $XDG_CONFIG_HOME on Linux.
func Dir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, appName), nil
}

// RiceDir returns the managed git repo directory (~/.config/easyrice/rice/).
func RiceDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, riceDirName), nil
}

// TomlPath returns the path to easyrice.toml inside the managed repo.
func TomlPath() (string, error) {
	riceDir, err := RiceDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(riceDir, tomlFile), nil
}

// DefaultToml returns the initial contents for a new easyrice.toml file.
func DefaultToml() string {
	return `[rice]
name = ""
description = ""
`
}
