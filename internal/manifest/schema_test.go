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

func TestSchema_DependenciesAndCustomDependencies(t *testing.T) {
	tomlStr := `schema_version = 1

[custom_dependencies.foo]
version_probe = ["foo", "--version"]
version_regex = "foo (\\d+\\.\\d+\\.\\d+)"

[custom_dependencies.foo.install.linux]
description = "Install foo on Linux"
shell_payload = "apt-get install foo"
distro_families = ["debian"]

[packages.myapp]
description = "My application"
supported_os = ["linux", "darwin"]
dependencies = [{name = "foo", version = ">=1.0.0"}]

[packages.myapp.profiles.default]
sources = [{path = "config", mode = "file", target = "$HOME/.config/myapp"}]
`

	var m Manifest
	_, err := toml.Decode(tomlStr, &m)
	require.NoError(t, err, "failed to decode TOML with dependencies and custom_dependencies")

	// Verify custom_dependencies
	assert.Len(t, m.CustomDependencies, 1)
	foo, ok := m.CustomDependencies["foo"]
	require.True(t, ok, "custom dependency 'foo' not found")
	assert.Equal(t, []string{"foo", "--version"}, foo.VersionProbe)
	assert.Equal(t, "foo (\\d+\\.\\d+\\.\\d+)", foo.VersionRegex)
	assert.Len(t, foo.Install, 1)
	linuxInstall, ok := foo.Install["linux"]
	require.True(t, ok, "linux install method not found")
	assert.Equal(t, "Install foo on Linux", linuxInstall.Description)
	assert.Equal(t, "apt-get install foo", linuxInstall.ShellPayload)
	assert.Equal(t, []string{"debian"}, linuxInstall.DistroFamilies)

	// Verify package dependencies
	myapp, ok := m.Packages["myapp"]
	require.True(t, ok, "myapp package not found")
	assert.Len(t, myapp.Dependencies, 1)
	assert.Equal(t, "foo", myapp.Dependencies[0].Name)
	assert.Equal(t, ">=1.0.0", myapp.Dependencies[0].Version)
}
