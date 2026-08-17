package utils

import (
	"log/slog"
	"os"
)

func SetupLogger() error {
	f, err := os.OpenFile(LogFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		slog.Error("Failed to open log file", "path", LogFile, "error", err)
		return err
	}

	slog.SetDefault(slog.New(slog.NewMultiHandler(slog.NewTextHandler(os.Stdin, nil), slog.NewJSONHandler(f, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}))))

	return nil
}
