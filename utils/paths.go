package utils

import (
	"log/slog"
	"os"
	"path/filepath"
)

var ConfigDir string
var CacheDir string
var LogFile string
var ConfigFile string

func init() {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		slog.Error("Unable to get user config directory", "error", err)
		panic(err)
	}

	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		slog.Error("Unable to get user cache directory", "error", err)
		panic(err)
	}

	ConfigDir = filepath.Join(userConfigDir, "easyrice")
	CacheDir = filepath.Join(userCacheDir, "easyrice")
	LogFile = filepath.Join(CacheDir, "easyrice.log")
	ConfigFile = filepath.Join(ConfigDir, "easyrice.toml")

	err = MkdirAll(ConfigDir)
	if err != nil {
		slog.Error("Error while accessing config directory", "path", ConfigDir, "error", err)
		panic(err)
	}

	err = MkdirAll(CacheDir)
	if err != nil {
		slog.Error("Error while accessing cache directory", "path", CacheDir, "error", err)
		panic(err)
	}
}
