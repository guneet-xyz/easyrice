package xdgpath

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigDir_POSIX_DefaultsToHomeDotConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX-only test")
	}
	t.Setenv("HOME", "/tmp/xyz")
	t.Setenv("XDG_CONFIG_HOME", "")

	got := ConfigDir()
	assert.Equal(t, "/tmp/xyz/.config", got)
	assert.True(t, filepath.IsAbs(got))
}

func TestConfigDir_POSIX_HonorsXDGConfigHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX-only test")
	}
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	t.Setenv("HOME", "/tmp/home")

	got := ConfigDir()
	assert.Equal(t, "/tmp/xdg", got)
	assert.True(t, filepath.IsAbs(got))
}

func TestConfigDir_POSIX_IgnoresRelativeXDG(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX-only test")
	}
	t.Setenv("XDG_CONFIG_HOME", "relative/path")
	t.Setenv("HOME", "/tmp/home")

	got := ConfigDir()
	// Should fall back to home-based path, not use the relative XDG
	assert.Equal(t, "/tmp/home/.config", got)
	assert.True(t, filepath.IsAbs(got))
}

func TestConfigDir_POSIX_HomeUnsetFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX-only test")
	}
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	got := ConfigDir()
	assert.True(t, filepath.IsAbs(got), "result must be absolute even when HOME is unset")
	assert.NotPanics(t, func() { ConfigDir() })
}

func TestConfigDir_AlwaysAbsolute(t *testing.T) {
	// Test that ConfigDir always returns an absolute path
	// regardless of environment state
	t.Setenv("HOME", "/tmp/test")
	t.Setenv("XDG_CONFIG_HOME", "")

	got := ConfigDir()
	assert.True(t, filepath.IsAbs(got), "ConfigDir must always return an absolute path")
}
