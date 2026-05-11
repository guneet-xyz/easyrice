package logger

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Level
		wantErr bool
	}{
		{"debug lowercase", "debug", DebugLevel, false},
		{"debug uppercase", "DEBUG", DebugLevel, false},
		{"debug mixed case", "DeBuG", DebugLevel, false},
		{"debug with spaces", "  debug  ", DebugLevel, false},
		{"info lowercase", "info", InfoLevel, false},
		{"info uppercase", "INFO", InfoLevel, false},
		{"warn lowercase", "warn", WarnLevel, false},
		{"warn uppercase", "WARN", WarnLevel, false},
		{"error lowercase", "error", ErrorLevel, false},
		{"error uppercase", "ERROR", ErrorLevel, false},
		{"critical lowercase", "critical", CriticalLevel, false},
		{"critical uppercase", "CRITICAL", CriticalLevel, false},
		{"invalid level", "verbose", 0, true},
		{"empty string", "", 0, true},
		{"invalid with spaces", "  invalid  ", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLevel(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid log level")
				assert.Contains(t, err.Error(), "valid values: debug, info, warn, error, critical")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestLevelOrdering(t *testing.T) {
	// Verify CriticalLevel > ErrorLevel
	assert.Greater(t, int8(CriticalLevel), int8(ErrorLevel))
	assert.Equal(t, int8(CriticalLevel), int8(ErrorLevel)+1)
}

func TestInitCreatesLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "subdir", "easyrice.log")

	err := Init(WarnLevel, logPath)
	require.NoError(t, err)

	// Verify file was created
	assert.FileExists(t, logPath)

	// Verify parent directory was created
	assert.DirExists(t, filepath.Dir(logPath))

	// Cleanup
	L = zap.NewNop()
}

func TestInitFileAlwaysDebugLevel(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "easyrice.log")

	// Initialize with WarnLevel console
	err := Init(WarnLevel, logPath)
	require.NoError(t, err)

	// Log at different levels
	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")

	// Sync to ensure all writes are flushed
	Sync()

	// Read log file
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)

	logContent := string(content)

	// File should contain all messages (always at DebugLevel)
	assert.Contains(t, logContent, "debug message")
	assert.Contains(t, logContent, "info message")
	assert.Contains(t, logContent, "warn message")
	assert.Contains(t, logContent, "error message")

	// Cleanup
	L = zap.NewNop()
}

func TestConsoleFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "easyrice.log")

	// Capture stderr
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	// Initialize with WarnLevel console
	err = Init(WarnLevel, logPath)
	require.NoError(t, err)

	// Log at different levels
	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")

	// Close stderr to flush
	w.Close()
	os.Stderr = oldStderr

	// Read captured stderr
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	r.Close()

	stderrContent := buf.String()

	// Console should NOT contain debug or info (WarnLevel filters them)
	assert.NotContains(t, stderrContent, "debug message")
	assert.NotContains(t, stderrContent, "info message")

	// Console SHOULD contain warn and error
	assert.Contains(t, stderrContent, "warn message")
	assert.Contains(t, stderrContent, "error message")

	// Cleanup
	L = zap.NewNop()
}

func TestCriticalIncludesGitHubIssueURL(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "easyrice.log")

	err := Init(DebugLevel, logPath)
	require.NoError(t, err)

	Critical("critical issue occurred")

	Sync()

	// Read log file
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)

	logContent := string(content)

	// Parse JSON log entries
	lines := strings.Split(strings.TrimSpace(logContent), "\n")
	assert.Greater(t, len(lines), 0)

	// Find the critical log entry
	var found bool
	for _, line := range lines {
		if strings.Contains(line, "critical issue occurred") {
			var logEntry map[string]interface{}
			err := json.Unmarshal([]byte(line), &logEntry)
			require.NoError(t, err)

			// Verify github_issue_url field exists
			assert.Contains(t, logEntry, "github_issue_url")
			assert.Equal(t, "https://github.com/guneet-xyz/easyrice/issues/new", logEntry["github_issue_url"])
			found = true
			break
		}
	}
	assert.True(t, found, "critical log entry not found in file")

	// Cleanup
	L = zap.NewNop()
}

func TestCriticalWithAdditionalFields(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "easyrice.log")

	err := Init(DebugLevel, logPath)
	require.NoError(t, err)

	Critical("critical issue", zap.String("component", "installer"))

	Sync()

	// Read log file
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)

	logContent := string(content)

	// Parse JSON log entries
	lines := strings.Split(strings.TrimSpace(logContent), "\n")
	assert.Greater(t, len(lines), 0)

	// Find the critical log entry
	var found bool
	for _, line := range lines {
		if strings.Contains(line, "critical issue") {
			var logEntry map[string]interface{}
			err := json.Unmarshal([]byte(line), &logEntry)
			require.NoError(t, err)

			// Verify both fields exist
			assert.Contains(t, logEntry, "github_issue_url")
			assert.Contains(t, logEntry, "component")
			assert.Equal(t, "https://github.com/guneet-xyz/easyrice/issues/new", logEntry["github_issue_url"])
			assert.Equal(t, "installer", logEntry["component"])
			found = true
			break
		}
	}
	assert.True(t, found, "critical log entry not found in file")

	// Cleanup
	L = zap.NewNop()
}

func TestDefaultLogPath(t *testing.T) {
	path := DefaultLogPath()

	// Should contain easyrice/logs/easyrice.log
	assert.Contains(t, path, "easyrice")
	assert.Contains(t, path, "logs")
	assert.Contains(t, path, "easyrice.log")

	// Should be an absolute path
	assert.True(t, filepath.IsAbs(path), "path should be absolute")
}

func TestSyncDoesNotPanic(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "easyrice.log")

	err := Init(DebugLevel, logPath)
	require.NoError(t, err)

	// Should not panic
	assert.NotPanics(t, func() {
		Sync()
	})

	// Cleanup
	L = zap.NewNop()
}

func TestInitWithInvalidPath(t *testing.T) {
	// Try to create a file in a path that cannot be created
	// (e.g., /dev/null/subdir/file.log on Unix)
	invalidPath := "/dev/null/subdir/easyrice.log"

	err := Init(DebugLevel, invalidPath)
	assert.Error(t, err)

	// Cleanup
	L = zap.NewNop()
}

func TestPackageLevelFunctionsWithNopLogger(t *testing.T) {
	// Reset to nop logger
	L = zap.NewNop()

	// These should not panic even with nop logger
	assert.NotPanics(t, func() {
		Debug("debug")
		Info("info")
		Warn("warn")
		Error("error")
		Critical("critical")
		Sync()
	})
}

func TestLogFileIsJSON(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "easyrice.log")

	err := Init(DebugLevel, logPath)
	require.NoError(t, err)

	Info("test message", zap.String("key", "value"))

	Sync()

	// Read log file
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)

	logContent := string(content)
	lines := strings.Split(strings.TrimSpace(logContent), "\n")

	// Each line should be valid JSON
	for _, line := range lines {
		if line == "" {
			continue
		}
		var logEntry map[string]interface{}
		err := json.Unmarshal([]byte(line), &logEntry)
		assert.NoError(t, err, "log line should be valid JSON: %s", line)
	}

	// Cleanup
	L = zap.NewNop()
}

func TestCriticalLevelValue(t *testing.T) {
	// CriticalLevel should be exactly ErrorLevel + 1
	assert.Equal(t, Level(zapcore.ErrorLevel)+1, CriticalLevel)
}

func TestDefaultLogPath_ReturnsValidPath(t *testing.T) {
	path := DefaultLogPath()

	// Should be non-empty
	assert.NotEmpty(t, path)

	// Should contain "easyrice"
	assert.Contains(t, path, "easyrice")

	// Should be an absolute path
	assert.True(t, filepath.IsAbs(path), "path should be absolute")
}

func TestDefaultLogPath_FallbackOnBadHome(t *testing.T) {
	// Set HOME to an invalid/empty value to force fallback
	t.Setenv("HOME", "")

	// On POSIX systems, this should trigger the fallback chain
	// The function should still return a valid path (either from fallback or hardcoded)
	path := DefaultLogPath()

	// Should be non-empty (either fallback path or hardcoded "easyrice.log")
	assert.NotEmpty(t, path)

	// Should contain "easyrice" (either in the path or as the filename)
	assert.Contains(t, path, "easyrice")
}

func TestDefaultLogPath_ContainsLogsDirectory(t *testing.T) {
	path := DefaultLogPath()

	// Should contain "logs" directory component
	assert.Contains(t, path, "logs")

	// Should end with easyrice.log
	assert.True(t, strings.HasSuffix(path, "easyrice.log"))
}

func TestParseLevel_ValidLevels(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", DebugLevel},
		{"info", InfoLevel},
		{"warn", WarnLevel},
		{"error", ErrorLevel},
		{"critical", CriticalLevel},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			parsed, err := ParseLevel(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, parsed)
		})
	}
}

func TestParseLevel_InvalidLevel(t *testing.T) {
	invalidLevels := []string{"verbose", "trace", "fatal", "unknown", ""}

	for _, level := range invalidLevels {
		t.Run(level, func(t *testing.T) {
			_, err := ParseLevel(level)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid log level")
			assert.Contains(t, err.Error(), "valid values: debug, info, warn, error, critical")
		})
	}
}
