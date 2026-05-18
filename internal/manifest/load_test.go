package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFile_HappyPath(t *testing.T) {
	dir := t.TempDir()
	manifest := `schema_version = 1

[packages.ghostty]
description = "Ghostty terminal emulator configuration"
supported_os = ["linux", "darwin"]

[packages.ghostty.profiles.common]
sources = [{path = "common", mode = "file", target = "$HOME/.config/ghostty"}]

[packages.ghostty.profiles.macbook]
sources = [
  {path = "common", mode = "file", target = "$HOME/.config/ghostty"},
  {path = "macbook", mode = "file", target = "$HOME/.config/ghostty"},
]

[packages.nvim]
description = "Neovim configuration"
supported_os = ["linux", "darwin"]
root = "nvim-custom"

[packages.nvim.profiles.default]
sources = [{path = "config", mode = "folder", target = "$HOME/.config/nvim"}]
`
	path := filepath.Join(dir, "rice.toml")
	require.NoError(t, os.WriteFile(path, []byte(manifest), 0o644))

	m, err := LoadFile(path)
	require.NoError(t, err)
	require.NotNil(t, m)

	_, ok := m.Packages["ghostty"]
	assert.True(t, ok, "expected Packages[\"ghostty\"] to exist")
}

func TestLoadFile_NotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent", "rice.toml")

	m, err := LoadFile(path)
	require.Error(t, err)
	assert.Nil(t, m)
	assert.Contains(t, err.Error(), "not found")
}

func TestLoadFile_InvalidTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rice.toml")
	require.NoError(t, os.WriteFile(path, []byte("this is = = not valid toml [[["), 0o644))

	m, err := LoadFile(path)
	require.Error(t, err)
	assert.Nil(t, m)
	assert.False(t, strings.Contains(err.Error(), "not found"), "expected decode error, not not-found")
}

func TestLoadFile_FailsValidation(t *testing.T) {
	dir := t.TempDir()
	manifest := `schema_version = 1
[packages]
# Empty packages section - no actual packages defined
`
	path := filepath.Join(dir, "rice.toml")
	require.NoError(t, os.WriteFile(path, []byte(manifest), 0o644))

	m, err := LoadFile(path)
	require.Error(t, err)
	assert.Nil(t, m)
	assert.Contains(t, err.Error(), "validate")
}
