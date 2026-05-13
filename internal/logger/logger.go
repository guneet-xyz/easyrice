package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/guneet-xyz/easyrice/internal/xdgpath"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Level represents a log level.
type Level int8

const (
	DebugLevel    Level = Level(zapcore.DebugLevel)
	InfoLevel     Level = Level(zapcore.InfoLevel)
	WarnLevel     Level = Level(zapcore.WarnLevel)
	ErrorLevel    Level = Level(zapcore.ErrorLevel)
	CriticalLevel Level = Level(zapcore.ErrorLevel) + 1
)

// L is the package-level logger. Initialized by Init().
var L *zap.Logger = zap.NewNop()

// ParseLevel parses a level string (case-insensitive).
// Valid values: debug, info, warn, error, critical
// Returns error listing valid values if unrecognized.
func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return DebugLevel, nil
	case "info":
		return InfoLevel, nil
	case "warn":
		return WarnLevel, nil
	case "error":
		return ErrorLevel, nil
	case "critical":
		return CriticalLevel, nil
	default:
		return 0, fmt.Errorf("invalid log level %q; valid values: debug, info, warn, error, critical", s)
	}
}

// Init sets up the global logger with a tee: console (STDERR) at consoleLevel,
// and a file at logFilePath always at DebugLevel (JSON format).
// Creates parent dir of logFilePath if needed.
// Returns error if file cannot be opened.
func Init(consoleLevel Level, logFilePath string) error {
	// Create parent directory if needed
	logDir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Console encoder: human-readable with colors
	consoleEncoderConfig := zap.NewDevelopmentEncoderConfig()
	consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderConfig)
	consoleCore := zapcore.NewCore(
		consoleEncoder,
		zapcore.AddSync(os.Stderr),
		zapcore.Level(consoleLevel),
	)

	// File encoder: JSON format, always at DebugLevel
	fileEncoderConfig := zap.NewProductionEncoderConfig()
	fileEncoder := zapcore.NewJSONEncoder(fileEncoderConfig)
	fileCore := zapcore.NewCore(
		fileEncoder,
		zapcore.AddSync(logFile),
		zapcore.DebugLevel,
	)

	// Tee both cores
	teeCore := zapcore.NewTee(consoleCore, fileCore)

	// Create logger
	L = zap.New(teeCore)

	return nil
}

// DefaultLogPath returns ~/.config/easyrice/logs/easyrice.log (POSIX)
// or %APPDATA%/easyrice/logs/easyrice.log (Windows).
func DefaultLogPath() string {
	return filepath.Join(xdgpath.ConfigDir(), "easyrice", "logs", "easyrice.log")
}

// Sync flushes the logger. Call from PersistentPostRun.
func Sync() {
	_ = L.Sync()
}

// Package-level log functions (use L internally):

// Debug logs a debug message.
func Debug(msg string, fields ...zap.Field) {
	L.Debug(msg, fields...)
}

// Info logs an info message.
func Info(msg string, fields ...zap.Field) {
	L.Info(msg, fields...)
}

// Warn logs a warning message.
func Warn(msg string, fields ...zap.Field) {
	L.Warn(msg, fields...)
}

// Error logs an error message.
func Error(msg string, fields ...zap.Field) {
	L.Error(msg, fields...)
}

// Critical logs at CriticalLevel and always adds github_issue_url field.
func Critical(msg string, fields ...zap.Field) {
	fields = append(fields, zap.String("github_issue_url", "https://github.com/guneet-xyz/easyrice/issues/new"))
	L.Log(zapcore.Level(CriticalLevel), msg, fields...)
}
