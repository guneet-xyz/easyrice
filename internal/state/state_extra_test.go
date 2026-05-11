package state

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultPath_HomeUnsetFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("env-based fallback differs on Windows")
	}
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	got := DefaultPath()
	assert.NotEmpty(t, got)
	assert.Contains(t, got, "easyrice")
	assert.Contains(t, got, "state.json")
	assert.Equal(t, filepath.Base(got), "state.json")
}
