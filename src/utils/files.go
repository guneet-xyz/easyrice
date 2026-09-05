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

func ReadBytes(path string) ([]byte, error) {
	slog.Debug("Reading bytes", "path", path)

	bytes, err := os.ReadFile(path)

	if err == nil {
		slog.Debug("Read bytes", "length", len(bytes), "path", path)
		return bytes, nil
	}

	if os.IsNotExist(err) {
		slog.Error("Couldn't read file because it does not exist.", "path", path, "error", err)
		return nil, err
	}

	if os.IsPermission(err) {
		slog.Error("Couldn't read file because of insufficient permissions.", "path", path, "error", err)
		return nil, err
	}

	slog.Error("Couldn't read file because unknown error", "path", path, "error", err)
	return nil, err
}

func WriteBytes(path string, bytes []byte) error {
	slog.Debug("Writing bytes", "path", path, "length", len(bytes))

	err := os.WriteFile(path, bytes, 0644)

	if err != nil {

		if os.IsPermission(err) {
			slog.Warn("Couldn't write file because of insufficient permissions.", "path", path, "error", err)
			return err
		}

		slog.Warn("Couldn't write file because unknown error", "path", path, "error", err)
		return err
	}

	slog.Debug("Written to file", "path", path, "length", len(bytes))
	return nil
}
