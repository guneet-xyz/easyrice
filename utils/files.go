package utils

import (
	"log/slog"
	"os"
)

func MkdirAll(path string) error {
	slog.Debug("Making directories", "path", path)
	err := os.MkdirAll(path, 0700)
	if os.IsPermission(err) {
		slog.Error("Unable to make directory due to insufficient permission", "path", path, "error", err)
		return err
	}
	if err != nil {
		slog.Error("Unable to make directory due to unknown error", "path", path, "error", err)
		return err
	}
	return nil
}
