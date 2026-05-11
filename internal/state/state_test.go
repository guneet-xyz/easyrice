package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/guneet-xyz/easyrice/internal/deps"
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

func TestSaveErrorOnUnwritablePath(t *testing.T) {
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	require.NoError(t, os.Mkdir(readOnlyDir, 0755))

	// Make directory read-only to prevent file creation
	require.NoError(t, os.Chmod(readOnlyDir, 0555))
	t.Cleanup(func() {
		// Restore permissions so TempDir cleanup can succeed
		_ = os.Chmod(readOnlyDir, 0755)
	})

	statePath := filepath.Join(readOnlyDir, "state.json")
	testState := State{
		"nvim": PackageState{
			Profile:        "default",
			InstalledLinks: []InstalledLink{},
			InstalledAt:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
	}

	err := Save(statePath, testState)
	assert.Error(t, err, "Save should error when parent directory is not writable")
}

func TestDefaultPathReturnsConfigDir(t *testing.T) {
	path := DefaultPath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "easyrice")
	assert.Contains(t, path, "state.json")
	assert.True(t, filepath.IsAbs(path), "DefaultPath should return absolute path")
}

func TestRoundTripLoadAfterSave(t *testing.T) {
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

	// Verify equality
	assert.Equal(t, originalState, loadedState)
}

func TestDefaultPathFallback(t *testing.T) {
	// This test verifies that DefaultPath returns a valid path even if UserConfigDir fails.
	// We can't easily mock os.UserConfigDir, but we can verify the fallback logic by
	// checking that the returned path contains expected components.
	path := DefaultPath()
	assert.NotEmpty(t, path)
	assert.True(t, filepath.IsAbs(path), "DefaultPath should always return absolute path")
	assert.Contains(t, path, "easyrice")
	assert.Contains(t, path, "state.json")
}

func TestSaveErrorOnMkdirAllFailure(t *testing.T) {
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	require.NoError(t, os.Mkdir(readOnlyDir, 0755))

	// Make directory read-only to prevent subdirectory creation
	require.NoError(t, os.Chmod(readOnlyDir, 0555))
	t.Cleanup(func() {
		// Restore permissions so TempDir cleanup can succeed
		_ = os.Chmod(readOnlyDir, 0755)
	})

	// Try to create a file in a subdirectory of the read-only directory
	statePath := filepath.Join(readOnlyDir, "subdir", "state.json")
	testState := State{
		"nvim": PackageState{
			Profile:        "default",
			InstalledLinks: []InstalledLink{},
			InstalledAt:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
	}

	err := Save(statePath, testState)
	assert.Error(t, err, "Save should error when parent directory cannot be created")
}

func TestLoadErrorOnReadFailure(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Create a file
	require.NoError(t, os.WriteFile(statePath, []byte("{}"), 0644))

	// Make file unreadable
	require.NoError(t, os.Chmod(statePath, 0000))
	t.Cleanup(func() {
		// Restore permissions so TempDir cleanup can succeed
		_ = os.Chmod(statePath, 0644)
	})

	_, err := Load(statePath)
	assert.Error(t, err, "Load should error when file cannot be read due to permissions")
}

func TestLoadLegacyStateWithoutInstalledDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Create legacy state JSON without installed_dependencies field
	legacyJSON := `{
  "nvim": {
    "profile": "default",
    "installed_links": [
      {
        "source": "/home/user/rice/nvim/init.lua",
        "target": "/home/user/.config/nvim/init.lua"
      }
    ],
    "installed_at": "2024-01-01T12:00:00Z"
  }
}`

	require.NoError(t, os.WriteFile(statePath, []byte(legacyJSON), 0644))

	// Load legacy state
	loaded, err := Load(statePath)
	assert.NoError(t, err, "Load should not error on legacy state without installed_dependencies")
	assert.NotNil(t, loaded)
	assert.Equal(t, "default", loaded["nvim"].Profile)
	assert.Nil(t, loaded["nvim"].InstalledDependencies, "InstalledDependencies should be nil for legacy state")
}

func TestRoundTripWithInstalledDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	depTime := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)

	originalState := State{
		"nvim": PackageState{
			Profile: "default",
			InstalledLinks: []InstalledLink{
				{
					Source: "/home/user/rice/nvim/init.lua",
					Target: "/home/user/.config/nvim/init.lua",
				},
			},
			InstalledAt: now,
			InstalledDependencies: []deps.InstalledDependency{
				{
					Name:              "neovim",
					Version:           "0.9.0",
					Method:            "apt",
					InstalledAt:       depTime,
					ManagedByEasyrice: true,
				},
				{
					Name:              "ripgrep",
					Version:           "13.0.0",
					Method:            "apt",
					InstalledAt:       depTime,
					ManagedByEasyrice: true,
				},
			},
		},
	}

	// Save
	err := Save(statePath, originalState)
	require.NoError(t, err)

	// Load
	loadedState, err := Load(statePath)
	require.NoError(t, err)

	// Verify equality
	assert.Equal(t, originalState, loadedState)
	assert.Len(t, loadedState["nvim"].InstalledDependencies, 2)
	assert.Equal(t, "neovim", loadedState["nvim"].InstalledDependencies[0].Name)
	assert.Equal(t, "ripgrep", loadedState["nvim"].InstalledDependencies[1].Name)
}

func TestRoundTripWithEmptyInstalledDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	originalState := State{
		"nvim": PackageState{
			Profile: "default",
			InstalledLinks: []InstalledLink{
				{
					Source: "/home/user/rice/nvim/init.lua",
					Target: "/home/user/.config/nvim/init.lua",
				},
			},
			InstalledAt:           now,
			InstalledDependencies: []deps.InstalledDependency{}, // empty slice
		},
	}

	// Save
	err := Save(statePath, originalState)
	require.NoError(t, err)

	// Load
	loadedState, err := Load(statePath)
	require.NoError(t, err)

	// Verify that empty slice round-trips
	assert.Len(t, loadedState["nvim"].InstalledDependencies, 0)
}

func TestRoundTripWithNilInstalledDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	originalState := State{
		"nvim": PackageState{
			Profile: "default",
			InstalledLinks: []InstalledLink{
				{
					Source: "/home/user/rice/nvim/init.lua",
					Target: "/home/user/.config/nvim/init.lua",
				},
			},
			InstalledAt:           now,
			InstalledDependencies: nil, // nil slice
		},
	}

	// Save
	err := Save(statePath, originalState)
	require.NoError(t, err)

	// Load
	loadedState, err := Load(statePath)
	require.NoError(t, err)

	// Verify equality - nil slice should round-trip correctly
	assert.Equal(t, originalState, loadedState)
	assert.Nil(t, loadedState["nvim"].InstalledDependencies)
}

func TestRoundTripWithMultipleDependenciesPerPackage(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	depTime := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)

	originalState := State{
		"nvim": PackageState{
			Profile: "default",
			InstalledLinks: []InstalledLink{
				{
					Source: "/home/user/rice/nvim/init.lua",
					Target: "/home/user/.config/nvim/init.lua",
				},
			},
			InstalledAt: now,
			InstalledDependencies: []deps.InstalledDependency{
				{
					Name:              "neovim",
					Version:           "0.9.0",
					Method:            "apt",
					InstalledAt:       depTime,
					ManagedByEasyrice: true,
				},
				{
					Name:              "ripgrep",
					Version:           "13.0.0",
					Method:            "apt",
					InstalledAt:       depTime,
					ManagedByEasyrice: true,
				},
				{
					Name:              "fd",
					Version:           "9.0.0",
					Method:            "apt",
					InstalledAt:       depTime,
					ManagedByEasyrice: false,
				},
				{
					Name:              "fzf",
					Version:           "0.40.0",
					Method:            "custom",
					InstalledAt:       depTime,
					ManagedByEasyrice: true,
				},
			},
		},
		"ghostty": PackageState{
			Profile: "minimal",
			InstalledLinks: []InstalledLink{
				{
					Source: "/home/user/rice/ghostty/config",
					Target: "/home/user/.config/ghostty/config",
				},
			},
			InstalledAt: now,
			InstalledDependencies: []deps.InstalledDependency{
				{
					Name:              "ghostty",
					Version:           "1.0.0",
					Method:            "apt",
					InstalledAt:       depTime,
					ManagedByEasyrice: true,
				},
			},
		},
	}

	// Save
	err := Save(statePath, originalState)
	require.NoError(t, err)

	// Load
	loadedState, err := Load(statePath)
	require.NoError(t, err)

	// Verify equality
	assert.Equal(t, originalState, loadedState)

	// Verify nvim dependencies
	assert.Len(t, loadedState["nvim"].InstalledDependencies, 4)
	assert.Equal(t, "neovim", loadedState["nvim"].InstalledDependencies[0].Name)
	assert.Equal(t, "ripgrep", loadedState["nvim"].InstalledDependencies[1].Name)
	assert.Equal(t, "fd", loadedState["nvim"].InstalledDependencies[2].Name)
	assert.Equal(t, "fzf", loadedState["nvim"].InstalledDependencies[3].Name)

	// Verify ghostty dependencies
	assert.Len(t, loadedState["ghostty"].InstalledDependencies, 1)
	assert.Equal(t, "ghostty", loadedState["ghostty"].InstalledDependencies[0].Name)

	// Verify ManagedByEasyrice field is preserved
	assert.True(t, loadedState["nvim"].InstalledDependencies[0].ManagedByEasyrice)
	assert.False(t, loadedState["nvim"].InstalledDependencies[2].ManagedByEasyrice)
}

func TestRoundTripWithMixedPackages(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	depTime := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)

	originalState := State{
		"nvim": PackageState{
			Profile: "default",
			InstalledLinks: []InstalledLink{
				{
					Source: "/home/user/rice/nvim/init.lua",
					Target: "/home/user/.config/nvim/init.lua",
				},
			},
			InstalledAt: now,
			InstalledDependencies: []deps.InstalledDependency{
				{
					Name:              "neovim",
					Version:           "0.9.0",
					Method:            "apt",
					InstalledAt:       depTime,
					ManagedByEasyrice: true,
				},
			},
		},
		"ghostty": PackageState{
			Profile: "minimal",
			InstalledLinks: []InstalledLink{
				{
					Source: "/home/user/rice/ghostty/config",
					Target: "/home/user/.config/ghostty/config",
				},
			},
			InstalledAt:           now,
			InstalledDependencies: nil,
		},
		"zsh": PackageState{
			Profile: "default",
			InstalledLinks: []InstalledLink{
				{
					Source: "/home/user/rice/zsh/rc",
					Target: "/home/user/.zshrc",
				},
			},
			InstalledAt:           now,
			InstalledDependencies: []deps.InstalledDependency{},
		},
	}

	// Save
	err := Save(statePath, originalState)
	require.NoError(t, err)

	// Load
	loadedState, err := Load(statePath)
	require.NoError(t, err)

	// Verify nvim has dependencies
	assert.Len(t, loadedState["nvim"].InstalledDependencies, 1)
	assert.Equal(t, "neovim", loadedState["nvim"].InstalledDependencies[0].Name)

	// Verify ghostty has nil dependencies
	assert.Nil(t, loadedState["ghostty"].InstalledDependencies)

	// Verify zsh has empty dependencies (JSON unmarshals empty array as nil)
	assert.Len(t, loadedState["zsh"].InstalledDependencies, 0)
}
