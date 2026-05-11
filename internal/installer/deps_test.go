package installer

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/deps"
	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/state"
)

func withStdin(t *testing.T, payload string) {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	if payload != "" {
		_, err = w.Write([]byte(payload))
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = orig
		_ = r.Close()
	})
}

func customDepWithMethods(probeArgv []string, regex, payload string) deps.CustomDependencyDef {
	return deps.CustomDependencyDef{
		VersionProbe: probeArgv,
		VersionRegex: regex,
		Install: map[string]deps.CustomInstallMethod{
			"linux":  {Description: "install via shell (linux)", ShellPayload: payload},
			"darwin": {Description: "install via shell (darwin)", ShellPayload: payload},
		},
	}
}

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

func TestEnsureDependencies_AutoAcceptInstallsAndRecordsState(t *testing.T) {
	withStdin(t, "y\n")
	ctx := context.Background()

	customDep := customDepWithMethods(
		[]string{"mytool", "--version"},
		`mytool (\d+\.\d+\.\d+)`,
		"echo install mytool",
	)

	m := manifest.Manifest{
		Packages: map[string]manifest.PackageDef{
			"nvim": {
				Description:  "Neovim",
				SupportedOS:  []string{"linux", "darwin"},
				Dependencies: []deps.DependencyRef{{Name: "mytool"}},
			},
		},
		CustomDependencies: map[string]deps.CustomDependencyDef{"mytool": customDep},
	}

	runner := &deps.MockRunner{
		Expectations: []deps.MockExpectation{
			{Argv: []string{"mytool", "--version"}, Result: deps.RunResult{ExitCode: 1}},
			{Argv: []string{"sh", "-c", "echo install mytool"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("ok")}},
			{Argv: []string{"mytool", "--version"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("mytool 1.2.3")}},
		},
	}

	resultState, err := EnsureDependencies(ctx, runner, m, "nvim", true, state.State{})

	require.NoError(t, err)
	require.Len(t, resultState["nvim"].InstalledDependencies, 1)
	got := resultState["nvim"].InstalledDependencies[0]
	assert.Equal(t, "mytool", got.Name)
	assert.Equal(t, "1.2.3", got.Version)
	assert.True(t, got.ManagedByEasyrice)
	assert.NotEmpty(t, got.Method)
}

func TestEnsureDependencies_DeclineConfirmation(t *testing.T) {
	withStdin(t, "n\n")
	ctx := context.Background()

	customDep := customDepWithMethods(
		[]string{"mytool", "--version"},
		`mytool (\d+\.\d+\.\d+)`,
		"echo install mytool",
	)

	m := manifest.Manifest{
		Packages: map[string]manifest.PackageDef{
			"nvim": {
				Description:  "Neovim",
				SupportedOS:  []string{"linux", "darwin"},
				Dependencies: []deps.DependencyRef{{Name: "mytool"}},
			},
		},
		CustomDependencies: map[string]deps.CustomDependencyDef{"mytool": customDep},
	}

	runner := &deps.MockRunner{
		Expectations: []deps.MockExpectation{
			{Argv: []string{"mytool", "--version"}, Result: deps.RunResult{ExitCode: 1}},
		},
	}

	initial := state.State{"nvim": state.PackageState{Profile: "default"}}
	resultState, err := EnsureDependencies(ctx, runner, m, "nvim", true, initial)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "declined")
	assert.Equal(t, initial, resultState)
}

func TestEnsureDependencies_DeclineOuterConfirmation(t *testing.T) {
	withStdin(t, "\n")
	ctx := context.Background()

	m := manifest.Manifest{
		Packages: map[string]manifest.PackageDef{
			"nvim": {
				Description:  "Neovim",
				SupportedOS:  []string{"linux", "darwin"},
				Dependencies: []deps.DependencyRef{{Name: "mdformat"}},
			},
		},
	}

	runner := &deps.MockRunner{
		Expectations: []deps.MockExpectation{
			{Argv: []string{"mdformat", "--version"}, Result: deps.RunResult{ExitCode: 1}},
		},
	}

	initial := state.State{"nvim": state.PackageState{Profile: "default"}}
	resultState, err := EnsureDependencies(ctx, runner, m, "nvim", false, initial)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "declined to install dependencies")
	assert.Equal(t, initial, resultState)
}

func TestEnsureDependencies_ReplaceExistingDep(t *testing.T) {
	withStdin(t, "y\n")
	ctx := context.Background()

	customDep := customDepWithMethods(
		[]string{"mytool", "--version"},
		`mytool (\d+\.\d+\.\d+)`,
		"echo install mytool",
	)

	m := manifest.Manifest{
		Packages: map[string]manifest.PackageDef{
			"nvim": {
				Description:  "Neovim",
				SupportedOS:  []string{"linux", "darwin"},
				Dependencies: []deps.DependencyRef{{Name: "mytool"}},
			},
		},
		CustomDependencies: map[string]deps.CustomDependencyDef{"mytool": customDep},
	}

	runner := &deps.MockRunner{
		Expectations: []deps.MockExpectation{
			{Argv: []string{"mytool", "--version"}, Result: deps.RunResult{ExitCode: 1}},
			{Argv: []string{"sh", "-c", "echo install mytool"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("ok")}},
			{Argv: []string{"mytool", "--version"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("mytool 2.0.0")}},
		},
	}

	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	initial := state.State{
		"nvim": state.PackageState{
			Profile: "default",
			InstalledDependencies: []deps.InstalledDependency{
				{Name: "mytool", Version: "1.0.0", Method: "stale", InstalledAt: old, ManagedByEasyrice: true},
				{Name: "ripgrep", Version: "13.0.0", Method: "apt", InstalledAt: old, ManagedByEasyrice: true},
			},
		},
	}

	resultState, err := EnsureDependencies(ctx, runner, m, "nvim", true, initial)

	require.NoError(t, err)
	pkg := resultState["nvim"]
	require.Len(t, pkg.InstalledDependencies, 2)

	byName := map[string]deps.InstalledDependency{}
	for _, d := range pkg.InstalledDependencies {
		byName[d.Name] = d
	}
	assert.Equal(t, "2.0.0", byName["mytool"].Version)
	assert.NotEqual(t, "stale", byName["mytool"].Method)
	assert.Equal(t, "13.0.0", byName["ripgrep"].Version)
	assert.Equal(t, "default", pkg.Profile)
}

func TestEnsureDependencies_NilStateAutoInit(t *testing.T) {
	withStdin(t, "y\n")
	ctx := context.Background()

	customDep := customDepWithMethods(
		[]string{"mytool", "--version"},
		`mytool (\d+\.\d+\.\d+)`,
		"echo install mytool",
	)

	m := manifest.Manifest{
		Packages: map[string]manifest.PackageDef{
			"nvim": {
				Description:  "Neovim",
				SupportedOS:  []string{"linux", "darwin"},
				Dependencies: []deps.DependencyRef{{Name: "mytool"}},
			},
		},
		CustomDependencies: map[string]deps.CustomDependencyDef{"mytool": customDep},
	}

	runner := &deps.MockRunner{
		Expectations: []deps.MockExpectation{
			{Argv: []string{"mytool", "--version"}, Result: deps.RunResult{ExitCode: 1}},
			{Argv: []string{"sh", "-c", "echo install mytool"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("ok")}},
			{Argv: []string{"mytool", "--version"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("mytool 1.2.3")}},
		},
	}

	resultState, err := EnsureDependencies(ctx, runner, m, "nvim", true, nil)

	require.NoError(t, err)
	require.NotNil(t, resultState)
	require.Len(t, resultState["nvim"].InstalledDependencies, 1)
	assert.Equal(t, "1.2.3", resultState["nvim"].InstalledDependencies[0].Version)
}

func TestEnsureDependencies_PostInstallUnknownVersionWarns(t *testing.T) {
	ctx := context.Background()

	customDep := deps.CustomDependencyDef{
		VersionProbe: []string{"already-here", "--version"},
		VersionRegex: `nope (\d+)`,
		Install: map[string]deps.CustomInstallMethod{
			"linux":  {Description: "shell", ShellPayload: "true"},
			"darwin": {Description: "shell", ShellPayload: "true"},
		},
	}

	m := manifest.Manifest{
		Packages: map[string]manifest.PackageDef{
			"nvim": {
				Description: "Neovim",
				SupportedOS: []string{"linux", "darwin"},
				Dependencies: []deps.DependencyRef{
					{Name: "mdformat"},
					{Name: "already-here"},
				},
			},
		},
		CustomDependencies: map[string]deps.CustomDependencyDef{"already-here": customDep},
	}

	runner := &deps.MockRunner{
		Expectations: []deps.MockExpectation{
			{Argv: []string{"mdformat", "--version"}, Result: deps.RunResult{ExitCode: 1}},
			{Argv: []string{"already-here", "--version"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("installed but no version match")}},
			{Argv: nil, Result: deps.RunResult{ExitCode: 0}},
			{Argv: []string{"mdformat", "--version"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("1.0.0")}},
		},
	}

	resultState, err := EnsureDependencies(ctx, runner, m, "nvim", true, state.State{})

	require.NoError(t, err)
	require.Len(t, resultState["nvim"].InstalledDependencies, 1)
	assert.Equal(t, "mdformat", resultState["nvim"].InstalledDependencies[0].Name)
}
func TestEnsureDependencies_InstallFailurePreservesPriorState(t *testing.T) {
	withStdin(t, "y\n")
	ctx := context.Background()

	customDep := customDepWithMethods(
		[]string{"mytool", "--version"},
		`mytool (\d+\.\d+\.\d+)`,
		"false",
	)

	m := manifest.Manifest{
		Packages: map[string]manifest.PackageDef{
			"nvim": {
				Description:  "Neovim",
				SupportedOS:  []string{"linux", "darwin"},
				Dependencies: []deps.DependencyRef{{Name: "mytool"}},
			},
		},
		CustomDependencies: map[string]deps.CustomDependencyDef{"mytool": customDep},
	}

	runner := &deps.MockRunner{
		Expectations: []deps.MockExpectation{
			{Argv: []string{"mytool", "--version"}, Result: deps.RunResult{ExitCode: 1}},
			{Argv: []string{"sh", "-c", "false"}, Result: deps.RunResult{ExitCode: 1, Combined: []byte("boom")}},
		},
	}

	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	initial := state.State{
		"nvim": state.PackageState{
			Profile: "default",
			InstalledDependencies: []deps.InstalledDependency{
				{Name: "ripgrep", Version: "13.0.0", Method: "apt", InstalledAt: old, ManagedByEasyrice: true},
			},
		},
	}

	resultState, err := EnsureDependencies(ctx, runner, m, "nvim", true, initial)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "install failed")
	require.Len(t, resultState["nvim"].InstalledDependencies, 1)
	assert.Equal(t, "ripgrep", resultState["nvim"].InstalledDependencies[0].Name)
}
