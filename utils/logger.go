package utils

import (
	"charm.land/log/v2"
	"log/slog"
	"os"
)

func SetupLogger() error {
	f, err := os.OpenFile(LogFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		slog.Error("Failed to open log file", "path", LogFile, "error", err)
		return err
	}

	jsonHandler := slog.NewJSONHandler(f, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})

	var handler slog.Handler
	if GetEnvOptions().Debug {
		textHandler := log.NewWithOptions(os.Stderr, log.Options{
			Level: log.Level(slog.LevelDebug),
		})
		handler = slog.NewMultiHandler(textHandler, jsonHandler)
	} else {
		handler = jsonHandler
	}

	slog.SetDefault(slog.New(handler))

	return nil
}
