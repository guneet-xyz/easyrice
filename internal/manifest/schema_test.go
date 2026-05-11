package manifest

import (
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchema_RoundTrip(t *testing.T) {
	tomlStr := `schema_version = 1

[packages.ghostty]
description = "Ghostty terminal"
supported_os = ["darwin"]
root = "ghostty-cfg"

[packages.ghostty.profiles.macbook]
sources = [{path = "common", mode = "file", target = "$HOME/.config/ghostty"}]
`

	var m Manifest
	_, err := toml.Decode(tomlStr, &m)
	require.NoError(t, err, "failed to decode TOML")

	// Verify top-level schema
	assert.Equal(t, 1, m.SchemaVersion)
	assert.Len(t, m.Packages, 1)

	// Verify ghostty package exists
	ghostty, ok := m.Packages["ghostty"]
	require.True(t, ok, "ghostty package not found")

	// Verify package metadata
	assert.Equal(t, "Ghostty terminal", ghostty.Description)
	assert.Equal(t, []string{"darwin"}, ghostty.SupportedOS)
	assert.Equal(t, "ghostty-cfg", ghostty.Root)

	// Verify profiles
	assert.Len(t, ghostty.Profiles, 1)
	macbook, ok := ghostty.Profiles["macbook"]
	require.True(t, ok, "macbook profile not found")

	// Verify sources
	assert.Len(t, macbook.Sources, 1)
	assert.Equal(t, SourceSpec{
		Path:   "common",
		Mode:   "file",
		Target: "$HOME/.config/ghostty",
	}, macbook.Sources[0])
}

func TestSchema_RejectsBareString(t *testing.T) {
	tomlStr := `schema_version = 1

[packages.ghostty]
description = "Ghostty terminal"
supported_os = ["darwin"]

[packages.ghostty.profiles.macbook]
sources = ["bad/path"]
`

	var m Manifest
	_, err := toml.Decode(tomlStr, &m)
	require.Error(t, err, "expected error for bare string in sources")
	assert.Contains(t, err.Error(), "expected a table", "error should mention expected table form")
}
