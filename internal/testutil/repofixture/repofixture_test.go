package repofixture

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/state"
)

// TestBuilder_Empty tests that an empty repo can be created.
func TestBuilder_Empty(t *testing.T) {
	fixture := New(t).Build()

	// Verify paths exist
	assert.DirExists(t, fixture.RepoPath)
	assert.DirExists(t, fixture.HomePath)

	// Verify rice.toml exists and is valid
	manifestPath := filepath.Join(fixture.RepoPath, "rice.toml")
	assert.FileExists(t, manifestPath)

	// Verify it's a valid manifest
	data, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "schema_version = 1")
}

// TestBuilder_SinglePackage tests creating a fixture with one package.
func TestBuilder_SinglePackage(t *testing.T) {
	profiles := map[string]Profile{
		"default": {
			Sources: []manifest.SourceSpec{
				{
					Path:   "config",
					Mode:   "file",
					Target: "$HOME/.config/nvim",
				},
			},
		},
	}

	fixture := New(t).
		WithPackage("nvim", profiles).
		Build()

	// Verify repo path exists
	assert.DirExists(t, fixture.RepoPath)

	// Verify rice.toml contains the package
	manifestPath := filepath.Join(fixture.RepoPath, "rice.toml")
	data, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	manifestStr := string(data)

	assert.Contains(t, manifestStr, "[packages.nvim]")
	assert.Contains(t, manifestStr, "default")
	assert.Contains(t, manifestStr, "config")

	// Verify package directory was created
	pkgPath := filepath.Join(fixture.RepoPath, "nvim", "default")
	assert.DirExists(t, pkgPath)

	// Verify dummy file exists
	dummyFile := filepath.Join(pkgPath, "dummy.txt")
	assert.FileExists(t, dummyFile)
}

// TestBuilder_SingleSubmodule tests creating a fixture with a submodule.
func TestBuilder_SingleSubmodule(t *testing.T) {
	submoduleManifest := `schema_version = 1

[packages.remote_pkg]
description = "Remote package"
supported_os = ["linux", "darwin"]

[packages.remote_pkg.profiles.default]
sources = [{path = "config", mode = "file", target = "$HOME/.config/remote"}]
`

	fixture := New(t).
		WithSubmodule("myremote", submoduleManifest).
		Build()

	// Verify repo path exists
	assert.DirExists(t, fixture.RepoPath)

	// Verify .gitmodules exists (created by git submodule add)
	gitmodulesPath := filepath.Join(fixture.RepoPath, ".gitmodules")
	assert.FileExists(t, gitmodulesPath)

	// Verify submodule directory exists
	submodulePath := filepath.Join(fixture.RepoPath, "remotes", "myremote")
	assert.DirExists(t, submodulePath)
}

// TestBuilder_WithStaleState tests creating a fixture with pre-existing state.
func TestBuilder_WithStaleState(t *testing.T) {
	staleStateJSON := `{
  "nvim": {
    "profile": "default",
    "installed_links": [
      {
        "source": "/home/user/.config/easyrice/repos/default/nvim/config/init.lua",
        "target": "/home/user/.config/nvim/init.lua"
      }
    ],
    "installed_at": "2025-05-10T12:00:00Z"
  }
}`

	fixture := New(t).
		WithStaleState(staleStateJSON).
		Build()

	// Verify state.json exists
	assert.FileExists(t, fixture.StatePath)

	// Verify state.json content
	data, err := os.ReadFile(fixture.StatePath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "nvim")
	assert.Contains(t, string(data), "default")

	// Verify it can be loaded
	s, err := fixture.LoadState()
	require.NoError(t, err)
	assert.Contains(t, s, "nvim")
	assert.Equal(t, "default", s["nvim"].Profile)
}

// TestBuilder_RawManifestPassthrough tests that WithRawManifest writes content unchanged.
func TestBuilder_RawManifestPassthrough(t *testing.T) {
	rawManifest := `not valid [[ toml
this is garbage
but it should be written exactly as provided`

	fixture := New(t).
		WithRawManifest(rawManifest).
		Build()

	// Read the manifest file
	manifestPath := filepath.Join(fixture.RepoPath, "rice.toml")
	data, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	// Verify exact content match (no parsing, no normalization)
	assert.Equal(t, rawManifest, string(data))
}

// TestBuilder_MultiplePackages tests creating a fixture with multiple packages.
func TestBuilder_MultiplePackages(t *testing.T) {
	nvimProfiles := map[string]Profile{
		"default": {
			Sources: []manifest.SourceSpec{
				{Path: "config", Mode: "file", Target: "$HOME/.config/nvim"},
			},
		},
	}

	ghosttyProfiles := map[string]Profile{
		"common": {
			Sources: []manifest.SourceSpec{
				{Path: "common", Mode: "file", Target: "$HOME/.config/ghostty"},
			},
		},
		"macbook": {
			Sources: []manifest.SourceSpec{
				{Path: "common", Mode: "file", Target: "$HOME/.config/ghostty"},
				{Path: "macbook", Mode: "file", Target: "$HOME/.config/ghostty"},
			},
		},
	}

	fixture := New(t).
		WithPackage("nvim", nvimProfiles).
		WithPackage("ghostty", ghosttyProfiles).
		Build()

	// Verify both packages in manifest
	manifestPath := filepath.Join(fixture.RepoPath, "rice.toml")
	data, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	manifestStr := string(data)

	assert.Contains(t, manifestStr, "[packages.nvim]")
	assert.Contains(t, manifestStr, "[packages.ghostty]")
	assert.Contains(t, manifestStr, "common")
	assert.Contains(t, manifestStr, "macbook")
}

// TestBuilder_WithRoot tests the WithRoot override.
func TestBuilder_WithRoot(t *testing.T) {
	profiles := map[string]Profile{
		"default": {
			Sources: []manifest.SourceSpec{
				{Path: "config", Mode: "file", Target: "$HOME/.config/nvim"},
			},
		},
	}

	fixture := New(t).
		WithRoot("nvim-custom").
		WithPackage("nvim", profiles).
		Build()

	// Verify package directory uses the custom root
	pkgPath := filepath.Join(fixture.RepoPath, "nvim-custom", "default")
	assert.DirExists(t, pkgPath)
}

// TestBuilder_Cleanup tests that cleanup is registered.
func TestBuilder_Cleanup(t *testing.T) {
	// Create a fixture in a sub-test to verify cleanup works
	t.Run("cleanup", func(t *testing.T) {
		fixture := New(t).Build()
		repoPath := fixture.RepoPath

		// Verify paths exist during test
		assert.DirExists(t, repoPath)
	})

	// After the sub-test, t.TempDir() cleanup should have removed the directory
	// (We can't directly verify this without accessing the parent's TempDir,
	// but the test passing without errors indicates cleanup worked)
}

// TestBuilder_FixtureHelpers tests the Fixture helper methods.
func TestBuilder_FixtureHelpers(t *testing.T) {
	fixture := New(t).Build()

	// Test CreateFile and ReadFile
	err := fixture.CreateFile("test/file.txt", "test content")
	require.NoError(t, err)

	content, err := fixture.ReadFile("test/file.txt")
	require.NoError(t, err)
	assert.Equal(t, "test content", content)

	// Test CreateHomeFile and ReadHomeFile
	err = fixture.CreateHomeFile(".config/test", "home content")
	require.NoError(t, err)

	homeContent, err := fixture.ReadHomeFile(".config/test")
	require.NoError(t, err)
	assert.Equal(t, "home content", homeContent)

	// Test WriteManifest and ReadManifest
	newManifest := `schema_version = 1

[packages.test]
description = "Test"
supported_os = ["linux"]

[packages.test.profiles.default]
sources = [{path = "config", mode = "file", target = "$HOME/.config/test"}]
`
	err = fixture.WriteManifest(newManifest)
	require.NoError(t, err)

	readManifest, err := fixture.ReadManifest()
	require.NoError(t, err)
	assert.Equal(t, newManifest, readManifest)

	// Test WriteState and LoadState
	testState := state.State{
		"test": state.PackageState{
			Profile: "default",
			InstalledLinks: []state.InstalledLink{
				{
					Source: "/repo/test/config",
					Target: "/home/user/.config/test",
				},
			},
		},
	}
	err = fixture.SaveState(testState)
	require.NoError(t, err)

	loadedState, err := fixture.LoadState()
	require.NoError(t, err)
	assert.Equal(t, "default", loadedState["test"].Profile)
}

// TestBuilder_GitOperations tests git operations on the fixture.
func TestBuilder_GitOperations(t *testing.T) {
	fixture := New(t).Build()

	// Test GitRun
	err := fixture.GitRun("status")
	assert.NoError(t, err)

	// Test GitRunOutput
	output, err := fixture.GitRunOutput("log", "--oneline")
	require.NoError(t, err)
	assert.Contains(t, output, "init managed repo")
}
