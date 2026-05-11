package logger

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultLogPath_FallbackWhenHomeUnset(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("env-based fallback path differs on Windows")
	}

	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	got := DefaultLogPath()
	assert.NotEmpty(t, got)
	assert.Contains(t, got, "easyrice.log")
}
