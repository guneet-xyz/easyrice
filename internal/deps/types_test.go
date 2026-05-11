package deps

import (
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDependencyRefUnmarshalTOML_TableForm(t *testing.T) {
	input := `
dependencies = [
  {name = "neovim", version = ">=0.10"},
  {name = "git"}
]
`
	var result struct {
		Dependencies []DependencyRef `toml:"dependencies"`
	}
	err := toml.Unmarshal([]byte(input), &result)
	require.NoError(t, err)
	require.Len(t, result.Dependencies, 2)

	assert.Equal(t, "neovim", result.Dependencies[0].Name)
	assert.Equal(t, ">=0.10", result.Dependencies[0].Version)

	assert.Equal(t, "git", result.Dependencies[1].Name)
	assert.Equal(t, "", result.Dependencies[1].Version)
}

func TestDependencyRefUnmarshalTOML_BareStringRejected(t *testing.T) {
	input := `dependencies = ["neovim"]`
	var result struct {
		Dependencies []DependencyRef `toml:"dependencies"`
	}
	err := toml.Unmarshal([]byte(input), &result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected a table")
}

func TestDependencyRefUnmarshalTOML_InvalidFieldType(t *testing.T) {
	input := `
dependencies = [
  {name = 123, version = ">=0.10"}
]
`
	var result struct {
		Dependencies []DependencyRef `toml:"dependencies"`
	}
	err := toml.Unmarshal([]byte(input), &result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be a string")
}

func TestPlatformString(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		expected string
	}{
		{
			name:     "with distro family",
			platform: Platform{OS: "linux", DistroFamily: "debian"},
			expected: "linux/debian",
		},
		{
			name:     "without distro family",
			platform: Platform{OS: "darwin"},
			expected: "darwin",
		},
		{
			name:     "empty platform",
			platform: Platform{},
			expected: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.platform.String())
		})
	}
}
