package updater

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultCacheDir_ReturnsValidPath(t *testing.T) {
	cacheDir := DefaultCacheDir()

	assert.NotEmpty(t, cacheDir)
	assert.True(t, strings.Contains(cacheDir, "easyrice"))
	assert.True(t, filepath.IsAbs(cacheDir))
}

func TestDefaultCacheDir_FallbackOnBadHome(t *testing.T) {
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	t.Setenv("HOME", "")

	cacheDir := DefaultCacheDir()

	assert.NotEmpty(t, cacheDir)
	assert.True(t, strings.Contains(cacheDir, "easyrice"))
}

func TestDefaultCacheDir_FallbackOnInvalidHome(t *testing.T) {
	t.Setenv("HOME", "/nonexistent/invalid/path/that/does/not/exist")

	cacheDir := DefaultCacheDir()

	assert.NotEmpty(t, cacheDir)
	assert.True(t, strings.Contains(cacheDir, "easyrice"))
}

func TestDefaultCacheDir_Consistency(t *testing.T) {
	dir1 := DefaultCacheDir()
	dir2 := DefaultCacheDir()
	dir3 := DefaultCacheDir()

	assert.Equal(t, dir1, dir2)
	assert.Equal(t, dir2, dir3)
}

func TestDefaultCacheDir_ContainsEasyrice(t *testing.T) {
	cacheDir := DefaultCacheDir()

	parts := strings.Split(cacheDir, string(filepath.Separator))
	require.NotEmpty(t, parts)

	assert.Equal(t, "easyrice", parts[len(parts)-1])
}
