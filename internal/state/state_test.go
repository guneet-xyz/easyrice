package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPath(t *testing.T) {
	path := DefaultPath()
	assert.NotEmpty(t, path)
	assert.True(t, filepath.IsAbs(path), "DefaultPath should return absolute path")

	configDir, err := os.UserConfigDir()
	require.NoError(t, err)
	assert.True(t, len(path) > len(configDir) && path[:len(configDir)] == configDir, "DefaultPath should be in config directory")
}

func TestLoadNonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "nonexistent", "state.json")

	s, err := Load(nonExistentPath)
	assert.NoError(t, err, "Load should not error on non-existent file")
	assert.NotNil(t, s, "Load should return empty State, not nil")
	assert.Equal(t, State{}, s, "Load should return empty State for non-existent file")
}

func TestLoadValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Create a valid state file
	testState := State{
		"nvim": PackageState{
			Profile: "default",
			InstalledLinks: []InstalledLink{
				{
					Source: "/home/user/rice/nvim/init.lua",
					Target: "/home/user/.config/nvim/init.lua",
				},
			},
			InstalledAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
	}

	data, err := json.MarshalIndent(testState, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(statePath, data, 0644))

	// Load and verify
	loaded, err := Load(statePath)
	assert.NoError(t, err)
	assert.Equal(t, testState, loaded)
}

func TestLoadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Write invalid JSON
	require.NoError(t, os.WriteFile(statePath, []byte("{invalid json}"), 0644))

	_, err := Load(statePath)
	assert.Error(t, err, "Load should error on invalid JSON")
}

func TestSaveCreatesParentDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "a", "b", "c", "state.json")

	testState := State{
		"ghostty": PackageState{
			Profile:        "minimal",
			InstalledLinks: []InstalledLink{},
			InstalledAt:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
	}

	err := Save(statePath, testState)
	assert.NoError(t, err)
	assert.FileExists(t, statePath)
}

func TestSaveWritesCorrectJSON(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	testState := State{
		"nvim": PackageState{
			Profile: "default",
			InstalledLinks: []InstalledLink{
				{
					Source: "/home/user/rice/nvim/init.lua",
					Target: "/home/user/.config/nvim/init.lua",
				},
			},
			InstalledAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
	}

	err := Save(statePath, testState)
	require.NoError(t, err)

	// Read back and verify
	data, err := os.ReadFile(statePath)
	require.NoError(t, err)

	var loaded State
	err = json.Unmarshal(data, &loaded)
	require.NoError(t, err)
	assert.Equal(t, testState, loaded)

	// Verify pretty-printing (should contain newlines and indentation)
	assert.Contains(t, string(data), "\n")
	assert.Contains(t, string(data), "  ")
}

func TestRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	originalState := State{
		"nvim": PackageState{
			Profile: "default",
			InstalledLinks: []InstalledLink{
				{
					Source: "/home/user/rice/nvim/init.lua",
					Target: "/home/user/.config/nvim/init.lua",
				},
				{
					Source: "/home/user/rice/nvim/lua/config.lua",
					Target: "/home/user/.config/nvim/lua/config.lua",
				},
			},
			InstalledAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
		"ghostty": PackageState{
			Profile: "minimal",
			InstalledLinks: []InstalledLink{
				{
					Source: "/home/user/rice/ghostty/config",
					Target: "/home/user/.config/ghostty/config",
				},
			},
			InstalledAt: time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC),
		},
	}

	// Save
	err := Save(statePath, originalState)
	require.NoError(t, err)

	// Load
	loadedState, err := Load(statePath)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, originalState, loadedState)
}

func TestSaveEmptyState(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	emptyState := State{}

	err := Save(statePath, emptyState)
	require.NoError(t, err)

	loaded, err := Load(statePath)
	require.NoError(t, err)
	assert.Equal(t, emptyState, loaded)
}

func TestLoadEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Create empty file
	require.NoError(t, os.WriteFile(statePath, []byte("{}"), 0644))

	loaded, err := Load(statePath)
	assert.NoError(t, err)
	assert.Equal(t, State{}, loaded)
}
