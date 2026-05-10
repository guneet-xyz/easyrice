package manifest

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestdataDir(t *testing.T) string {
	// Get the directory of this test file
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get caller info")
	testDir := filepath.Dir(file)
	// Navigate up to repo root: internal/manifest -> internal -> . (repo root)
	repoRoot := filepath.Join(testDir, "..", "..")
	return filepath.Join(repoRoot, "testdata", "manifest_valid")
}

func getTestdataManifestDir(t *testing.T) string {
	// Get the directory of this test file
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get caller info")
	testDir := filepath.Dir(file)
	// Navigate up to repo root: internal/manifest -> internal -> . (repo root)
	repoRoot := filepath.Join(testDir, "..", "..")
	return filepath.Join(repoRoot, "testdata", "manifest")
}

func TestLoad_HappyPath(t *testing.T) {
	dir := filepath.Join(getTestdataManifestDir(t), "nvim")
	m, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, 1, m.SchemaVersion)
	assert.Equal(t, "nvim", m.Name)
	assert.Equal(t, "Neovim configuration", m.Description)
	assert.Equal(t, []string{"linux", "darwin"}, m.SupportedOS)
	assert.Equal(t, "os", m.ProfileKey)
	assert.Len(t, m.Profiles, 2)
	assert.Contains(t, m.Profiles, "linux")
	assert.Contains(t, m.Profiles, "darwin")
	assert.Equal(t, []SourceSpec{{Path: "common", Mode: "file", Target: "$HOME/.config/nvim"}, {Path: "linux", Mode: "file", Target: "$HOME/.config/nvim"}}, m.Profiles["linux"].Sources)
	assert.Equal(t, []SourceSpec{{Path: "common", Mode: "file", Target: "$HOME/.config/nvim"}, {Path: "darwin", Mode: "file", Target: "$HOME/.config/nvim"}}, m.Profiles["darwin"].Sources)
}

func TestLoad_FileNotFound(t *testing.T) {
	dir := filepath.Join(getTestdataManifestDir(t), "nonexistent")
	m, err := Load(dir)
	assert.Nil(t, m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rice.toml not found")
}

func TestLoad_InvalidTOML(t *testing.T) {
	// Create a temporary directory with invalid TOML
	tmpDir := t.TempDir()
	ricePath := filepath.Join(tmpDir, "rice.toml")
	err := os.WriteFile(ricePath, []byte("invalid toml [[["), 0644)
	require.NoError(t, err)

	m, err := Load(tmpDir)
	assert.Nil(t, m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse rice.toml")
}

func TestLoad_ValidationFailure(t *testing.T) {
	dir := filepath.Join(getTestdataManifestDir(t), "bad")
	m, err := Load(dir)
	assert.Nil(t, m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "manifest validation failed")
	assert.Contains(t, err.Error(), "unsupported schema_version")
}

func TestDiscover_FindsAllValidManifests(t *testing.T) {
	repoRoot := getTestdataDir(t)
	manifests, err := Discover(repoRoot)
	require.NoError(t, err)

	// Should find nvim and ghostty, but not bad (validation fails)
	assert.Len(t, manifests, 2)
	assert.Contains(t, manifests, "nvim")
	assert.Contains(t, manifests, "ghostty")
	assert.NotContains(t, manifests, "bad")

	// Verify nvim manifest
	nvimManifest := manifests["nvim"]
	assert.Equal(t, "nvim", nvimManifest.Name)
	assert.Len(t, nvimManifest.Profiles, 2)

	// Verify ghostty manifest
	ghosttyManifest := manifests["ghostty"]
	assert.Equal(t, "ghostty", ghosttyManifest.Name)
	assert.Len(t, ghosttyManifest.Profiles, 2)
}

func TestDiscover_ReturnsErrorOnInvalidManifest(t *testing.T) {
	// Create a temporary directory with a valid directory but invalid manifest
	tmpDir := t.TempDir()
	badDir := filepath.Join(tmpDir, "badpkg")
	err := os.Mkdir(badDir, 0755)
	require.NoError(t, err)

	ricePath := filepath.Join(badDir, "rice.toml")
	err = os.WriteFile(ricePath, []byte("schema_version = 99\nname = \"bad\"\nsupported_os = [\"linux\"]\nprofile_key = \"os\"\n[profiles.default]\nsources = [{path = \"common\", mode = \"file\", target = \"$HOME\"}]"), 0644)
	require.NoError(t, err)

	manifests, err := Discover(tmpDir)
	assert.Nil(t, manifests)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load manifest")
}

func TestDiscover_SkipsDirectoriesWithoutRiceToml(t *testing.T) {
	// Create a temporary directory with mixed content
	tmpDir := t.TempDir()

	// Create a directory without rice.toml
	emptyDir := filepath.Join(tmpDir, "empty")
	err := os.Mkdir(emptyDir, 0755)
	require.NoError(t, err)

	// Create a directory with valid rice.toml
	validDir := filepath.Join(tmpDir, "valid")
	err = os.Mkdir(validDir, 0755)
	require.NoError(t, err)

	ricePath := filepath.Join(validDir, "rice.toml")
	validToml := `schema_version = 1
name = "valid"
supported_os = ["linux"]
profile_key = "os"

[profiles.default]
sources = [{path = "common", mode = "file", target = "$HOME/.config/valid"}]`
	err = os.WriteFile(ricePath, []byte(validToml), 0644)
	require.NoError(t, err)

	// Create a file (not directory) at root level
	filePath := filepath.Join(tmpDir, "somefile.txt")
	err = os.WriteFile(filePath, []byte("content"), 0644)
	require.NoError(t, err)

	manifests, err := Discover(tmpDir)
	require.NoError(t, err)

	// Should only find the valid directory
	assert.Len(t, manifests, 1)
	assert.Contains(t, manifests, "valid")
	assert.NotContains(t, manifests, "empty")
	assert.NotContains(t, manifests, "somefile.txt")
}

func TestDiscover_EmptyRepository(t *testing.T) {
	tmpDir := t.TempDir()
	manifests, err := Discover(tmpDir)
	require.NoError(t, err)
	assert.Len(t, manifests, 0)
}

func TestDiscover_MultiProfileManifest(t *testing.T) {
	repoRoot := getTestdataDir(t)
	manifests, err := Discover(repoRoot)
	require.NoError(t, err)

	ghosttyManifest := manifests["ghostty"]
	assert.Equal(t, "ghostty", ghosttyManifest.Name)
	assert.Equal(t, "machine", ghosttyManifest.ProfileKey)
	assert.Len(t, ghosttyManifest.Profiles, 2)
	assert.Equal(t, []SourceSpec{{Path: "common", Mode: "file", Target: "$HOME/.config/ghostty"}, {Path: "macbook", Mode: "file", Target: "$HOME/.config/ghostty"}}, ghosttyManifest.Profiles["macbook"].Sources)
	assert.Equal(t, []SourceSpec{{Path: "common", Mode: "file", Target: "$HOME/.config/ghostty"}, {Path: "devstick", Mode: "file", Target: "$HOME/.config/ghostty"}}, ghosttyManifest.Profiles["devstick"].Sources)
}
