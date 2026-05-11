package installer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/deps"
	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/state"
)

func TestEnsureDependencies_AllDepsOK(t *testing.T) {
	ctx := context.Background()

	m := manifest.Manifest{
		Packages: map[string]manifest.PackageDef{
			"nvim": {
				Description: "Neovim",
				SupportedOS: []string{"linux", "darwin"},
				Dependencies: []deps.DependencyRef{
					{Name: "git", Version: ""},
				},
			},
		},
	}

	runner := &deps.MockRunner{
		Expectations: []deps.MockExpectation{
			{
				Argv: []string{"git", "--version"},
				Result: deps.RunResult{
					ExitCode: 0,
					Stdout:   []byte("git version 2.40.0"),
				},
			},
		},
	}

	s := state.State{}

	resultState, err := EnsureDependencies(ctx, runner, m, "nvim", false, s)

	assert.NoError(t, err)
	assert.Empty(t, resultState["nvim"].InstalledDependencies)
}

func TestEnsureDependencies_VersionMismatch(t *testing.T) {
	ctx := context.Background()

	m := manifest.Manifest{
		Packages: map[string]manifest.PackageDef{
			"nvim": {
				Description: "Neovim",
				SupportedOS: []string{"linux", "darwin"},
				Dependencies: []deps.DependencyRef{
					{Name: "node", Version: ">=20.0.0"},
				},
			},
		},
	}

	runner := &deps.MockRunner{
		Expectations: []deps.MockExpectation{
			{
				Argv: []string{"node", "--version"},
				Result: deps.RunResult{
					ExitCode: 0,
					Combined: []byte("v18.0.0"),
				},
			},
		},
	}

	s := state.State{}

	_, err := EnsureDependencies(ctx, runner, m, "nvim", false, s)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ensure dependencies: version mismatch detected for package")
}

func TestEnsureDependencies_PackageNotFound(t *testing.T) {
	ctx := context.Background()

	m := manifest.Manifest{
		Packages: map[string]manifest.PackageDef{},
	}

	runner := &deps.MockRunner{}

	_, err := EnsureDependencies(ctx, runner, m, "nonexistent", false, state.State{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in manifest")
}

func TestEnsureDependencies_ProbeUnknownVersion(t *testing.T) {
	ctx := context.Background()

	m := manifest.Manifest{
		Packages: map[string]manifest.PackageDef{
			"nvim": {
				Description: "Neovim",
				SupportedOS: []string{"linux", "darwin"},
				Dependencies: []deps.DependencyRef{
					{Name: "custom-tool", Version: ""},
				},
			},
		},
		CustomDependencies: map[string]deps.CustomDependencyDef{
			"custom-tool": {
				VersionProbe: []string{"custom-tool", "--version"},
				VersionRegex: "version: (\\d+\\.\\d+\\.\\d+)",
			},
		},
	}

	runner := &deps.MockRunner{
		Expectations: []deps.MockExpectation{
			{
				Argv: []string{"custom-tool", "--version"},
				Result: deps.RunResult{
					ExitCode: 0,
					Stdout:   []byte("custom-tool is installed"),
				},
			},
		},
	}

	s := state.State{}

	resultState, err := EnsureDependencies(ctx, runner, m, "nvim", false, s)

	assert.NoError(t, err)
	assert.Empty(t, resultState["nvim"].InstalledDependencies)
}

func TestEnsureDependencies_MergeState(t *testing.T) {
	ctx := context.Background()

	m := manifest.Manifest{
		Packages: map[string]manifest.PackageDef{
			"nvim": {
				Description: "Neovim",
				SupportedOS: []string{"linux", "darwin"},
				Dependencies: []deps.DependencyRef{
					{Name: "git", Version: ""},
				},
			},
		},
	}

	initialState := state.State{
		"nvim": state.PackageState{
			Profile: "default",
			InstalledDependencies: []deps.InstalledDependency{
				{
					Name:              "ripgrep",
					Version:           "13.0.0",
					Method:            "apt",
					InstalledAt:       time.Now(),
					ManagedByEasyrice: true,
				},
			},
		},
	}

	runner := &deps.MockRunner{
		Expectations: []deps.MockExpectation{
			{
				Argv: []string{"git", "--version"},
				Result: deps.RunResult{
					ExitCode: 0,
					Stdout:   []byte("git version 2.40.0"),
				},
			},
		},
	}

	resultState, err := EnsureDependencies(ctx, runner, m, "nvim", false, initialState)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(resultState["nvim"].InstalledDependencies))
	assert.Equal(t, "ripgrep", resultState["nvim"].InstalledDependencies[0].Name)
}

func TestEnsureDependencies_InitializePackageState(t *testing.T) {
	ctx := context.Background()

	m := manifest.Manifest{
		Packages: map[string]manifest.PackageDef{
			"nvim": {
				Description: "Neovim",
				SupportedOS: []string{"linux", "darwin"},
				Dependencies: []deps.DependencyRef{
					{Name: "git", Version: ""},
				},
			},
		},
	}

	runner := &deps.MockRunner{
		Expectations: []deps.MockExpectation{
			{
				Argv: []string{"git", "--version"},
				Result: deps.RunResult{
					ExitCode: 0,
					Stdout:   []byte("git version 2.40.0"),
				},
			},
		},
	}

	s := state.State{}

	resultState, err := EnsureDependencies(ctx, runner, m, "nvim", false, s)

	assert.NoError(t, err)
	assert.NotNil(t, resultState["nvim"])
}
