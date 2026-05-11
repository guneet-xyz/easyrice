package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/deps"
	"github.com/guneet-xyz/easyrice/internal/repo"
	"github.com/guneet-xyz/easyrice/internal/state"
)

func setupE2ERepo(t *testing.T) (repoRoot, statePath, homeDir string) {
	t.Helper()
	homeDir = setIsolatedHome(t)
	repoRoot = repo.DefaultRepoPath()
	require.NoError(t, os.MkdirAll(repoRoot, 0o755))
	statePath = filepath.Join(t.TempDir(), "state.json")
	return
}

func withMockRunner(t *testing.T, mock deps.Runner) {
	t.Helper()
	orig := DepsRunner
	DepsRunner = mock
	t.Cleanup(func() { DepsRunner = orig })
}

// withStdin redirects os.Stdin to a pipe pre-loaded with payload.
// Required because installer.EnsureDependencies reads from os.Stdin directly
// (via prompt.Confirm) for shell_payload custom-dependency confirmations,
// which do NOT respect autoAccept inside SelectInstallMethod.
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

func writeRepoFile(t *testing.T, repoRoot, rel, content string) {
	t.Helper()
	full := filepath.Join(repoRoot, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
}

func runE2ECmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetIn(strings.NewReader(""))
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	rootCmd.SetIn(os.Stdin)
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	return buf.String(), err
}

// installDepsFixture writes an inline manifest mirroring testdata/manifest_with_deps,
// using "myformat" as the custom dep (mdformat is in the registry; using it would
// trigger validate: name is already in the registry).
func installDepsFixture(t *testing.T, repoRoot string) {
	t.Helper()
	manifest := `schema_version = 1

[custom_dependencies.myformat]
description = "Custom formatter"
version_probe = ["myformat", "--version"]
version_regex = "myformat ([0-9.]+)"

[custom_dependencies.myformat.install.linux_debian]
description = "Install via pip"
shell_payload = "pip install myformat"

[custom_dependencies.myformat.install.darwin]
description = "Install via pip"
shell_payload = "pip install myformat"

[packages.nvim]
description = "Neovim configuration"
supported_os = ["linux", "darwin"]
dependencies = [
  {name = "neovim", version = ">=0.10"},
  {name = "ripgrep"},
  {name = "myformat"},
]

[packages.nvim.profiles.default]
sources = [{path = "config", mode = "folder", target = "$HOME/.config/nvim"}]
`
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "rice.toml"), []byte(manifest), 0o644))
	writeRepoFile(t, repoRoot, "nvim/config/init.lua", "-- noop\n")
}

// requireDarwinDeps skips on non-darwin: registry install methods on linux
// (apt/dnf/pacman/apk) all require root.
func requireDarwinDeps(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Skipf("requires darwin (linux registry methods need root); GOOS=%s", runtime.GOOS)
	}
}

// requireLinuxDeps skips on non-linux: for linux-specific registry install tests.
func requireLinuxDeps(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skipf("requires linux; GOOS=%s", runtime.GOOS)
	}
}

func ensureLinuxOrDarwin(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skipf("fixture supports linux/darwin only; GOOS=%s", runtime.GOOS)
	}
}

func TestE2E_InstallWithDeps(t *testing.T) {
	requireDarwinDeps(t)
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, statePath, homeDir := setupE2ERepo(t)
	installDepsFixture(t, repoRoot)

	mock := &deps.MockRunner{
		Expectations: []deps.MockExpectation{
			{Argv: []string{"nvim", "--version"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("NVIM v0.10.0\n")}},
			{Argv: []string{"rg", "--version"}, Result: deps.RunResult{ExitCode: 1}},
			{Argv: []string{"myformat", "--version"}, Result: deps.RunResult{ExitCode: 1}},
			{Argv: []string{"brew", "install", "ripgrep"}, Result: deps.RunResult{ExitCode: 0}},
			{Argv: []string{"rg", "--version"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("ripgrep 14.1.0\n")}},
			{Argv: []string{"sh", "-c", "pip install myformat"}, Result: deps.RunResult{ExitCode: 0}},
			{Argv: []string{"myformat", "--version"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("myformat 1.0.0\n")}},
		},
	}
	withMockRunner(t, mock)
	withStdin(t, "y\n")

	out, err := runE2ECmd(t,
		"--state", statePath,
		"--yes",
		"install", "nvim",
		"--profile", "default",
	)
	require.NoError(t, err, "out=%s", out)

	s, err := state.Load(statePath)
	require.NoError(t, err)
	pkg, ok := s["nvim"]
	require.True(t, ok, "nvim should be in state")
	assert.Equal(t, "default", pkg.Profile)

	link := filepath.Join(homeDir, ".config", "nvim")
	fi, err := os.Lstat(link)
	require.NoError(t, err)
	assert.NotZero(t, fi.Mode()&os.ModeSymlink, "expected nvim symlink at %s", link)

	assert.Equal(t, len(mock.Expectations), len(mock.Calls), "all mock calls consumed")
}

func TestE2E_InstallWithDeps_Linux(t *testing.T) {
	requireLinuxDeps(t)
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, statePath, homeDir := setupE2ERepo(t)
	installDepsFixture(t, repoRoot)

	mock := &deps.MockRunner{
		Expectations: []deps.MockExpectation{
			{Argv: []string{"nvim", "--version"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("NVIM v0.10.0\n")}},
			{Argv: []string{"rg", "--version"}, Result: deps.RunResult{ExitCode: 1}},
			{Argv: []string{"myformat", "--version"}, Result: deps.RunResult{ExitCode: 1}},
			{Argv: []string{"apt-get", "install", "-y", "ripgrep"}, Result: deps.RunResult{ExitCode: 0}},
			{Argv: []string{"rg", "--version"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("ripgrep 14.1.0\n")}},
			{Argv: []string{"sh", "-c", "pip install myformat"}, Result: deps.RunResult{ExitCode: 0}},
			{Argv: []string{"myformat", "--version"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("myformat 1.0.0\n")}},
		},
	}
	withMockRunner(t, mock)
	withStdin(t, "y\n")

	out, err := runE2ECmd(t,
		"--state", statePath,
		"--yes",
		"install", "nvim",
		"--profile", "default",
	)
	require.NoError(t, err, "out=%s", out)

	s, err := state.Load(statePath)
	require.NoError(t, err)
	pkg, ok := s["nvim"]
	require.True(t, ok, "nvim should be in state")
	assert.Equal(t, "default", pkg.Profile)

	link := filepath.Join(homeDir, ".config", "nvim")
	fi, err := os.Lstat(link)
	require.NoError(t, err)
	assert.NotZero(t, fi.Mode()&os.ModeSymlink, "expected nvim symlink at %s", link)

	assert.Equal(t, len(mock.Expectations), len(mock.Calls), "all mock calls consumed")
}

func TestE2E_VersionMismatchAbort(t *testing.T) {
	ensureLinuxOrDarwin(t)
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, statePath, _ := setupE2ERepo(t)
	installDepsFixture(t, repoRoot)

	mock := &deps.MockRunner{
		Expectations: []deps.MockExpectation{
			{Argv: []string{"nvim", "--version"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("NVIM v0.9.0\n")}},
			{Argv: []string{"rg", "--version"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("ripgrep 14.1.0\n")}},
			{Argv: []string{"myformat", "--version"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("myformat 1.0.0\n")}},
		},
	}
	withMockRunner(t, mock)

	out, err := runE2ECmd(t,
		"--state", statePath,
		"--yes",
		"install", "nvim",
		"--profile", "default",
	)
	require.Error(t, err, "expected version-mismatch error; out=%s", out)
	assert.Contains(t, err.Error(), "version mismatch")

	s, _ := state.Load(statePath)
	_, ok := s["nvim"]
	assert.False(t, ok, "nvim should NOT be in state after version-mismatch abort")
}

func TestE2E_ReservedSelfDepError(t *testing.T) {
	ensureLinuxOrDarwin(t)
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, statePath, _ := setupE2ERepo(t)
	manifest := `schema_version = 1

[packages.neovim]
description = "Neovim configuration"
supported_os = ["linux", "darwin"]
dependencies = [{name = "neovim"}]

[packages.neovim.profiles.default]
sources = [{path = "config", mode = "folder", target = "$HOME/.config/nvim"}]
`
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "rice.toml"), []byte(manifest), 0o644))
	writeRepoFile(t, repoRoot, "neovim/config/init.lua", "-- noop\n")

	out, err := runE2ECmd(t,
		"--state", statePath,
		"--yes",
		"install", "neovim",
		"--profile", "default",
	)
	require.Error(t, err, "expected reserved-name error; out=%s", out)
	assert.Contains(t, strings.ToLower(err.Error()), "reserved")
}

// TestE2E_SwitchReEvalsDeps: switching profiles re-runs the dep flow.
// ExecuteSwitchPlan does uninstall+install which wipes state.InstalledDependencies,
// so we assert the dep mock was *invoked* during switch (proving re-evaluation),
// not that the state retains the records.
func TestE2E_SwitchReEvalsDeps(t *testing.T) {
	requireDarwinDeps(t)
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, statePath, _ := setupE2ERepo(t)

	manifest := `schema_version = 1

[packages.demo]
description = "Demo"
supported_os = ["linux", "darwin"]
dependencies = [
  {name = "ripgrep"},
]

[packages.demo.profiles.minimal]
sources = [{path = "minimal", mode = "file", target = "$HOME/.config/demo"}]

[packages.demo.profiles.full]
sources = [{path = "full", mode = "file", target = "$HOME/.config/demo"}]
`
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "rice.toml"), []byte(manifest), 0o644))
	writeRepoFile(t, repoRoot, "demo/minimal/demo.cfg", "minimal\n")
	writeRepoFile(t, repoRoot, "demo/full/demo.cfg", "full\n")

	mock := &deps.MockRunner{
		Expectations: []deps.MockExpectation{
			{Argv: []string{"rg", "--version"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("ripgrep 14.1.0\n")}},
			{Argv: []string{"rg", "--version"}, Result: deps.RunResult{ExitCode: 1}},
			{Argv: []string{"brew", "install", "ripgrep"}, Result: deps.RunResult{ExitCode: 0}},
			{Argv: []string{"rg", "--version"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("ripgrep 14.1.0\n")}},
		},
	}
	withMockRunner(t, mock)

	out, err := runE2ECmd(t,
		"--state", statePath,
		"--yes",
		"install", "demo",
		"--profile", "minimal",
	)
	require.NoError(t, err, "initial install failed: out=%s", out)
	require.Equal(t, 1, len(mock.Calls), "initial install: only ripgrep probe")

	resetInstallFlags()
	out, err = runE2ECmd(t,
		"--state", statePath,
		"--yes",
		"switch", "demo", "full",
	)
	require.NoError(t, err, "switch failed: out=%s", out)

	assert.Equal(t, len(mock.Expectations), len(mock.Calls),
		"switch must re-evaluate deps (probe+install+post-probe)")

	s, err := state.Load(statePath)
	require.NoError(t, err)
	pkg, ok := s["demo"]
	require.True(t, ok)
	assert.Equal(t, "full", pkg.Profile, "switch must update profile in state")
}

func TestE2E_SwitchReEvalsDeps_Linux(t *testing.T) {
	requireLinuxDeps(t)
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, statePath, _ := setupE2ERepo(t)

	manifest := `schema_version = 1

[packages.demo]
description = "Demo"
supported_os = ["linux", "darwin"]
dependencies = [
  {name = "ripgrep"},
]

[packages.demo.profiles.minimal]
sources = [{path = "minimal", mode = "file", target = "$HOME/.config/demo"}]

[packages.demo.profiles.full]
sources = [{path = "full", mode = "file", target = "$HOME/.config/demo"}]
`
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "rice.toml"), []byte(manifest), 0o644))
	writeRepoFile(t, repoRoot, "demo/minimal/demo.cfg", "minimal\n")
	writeRepoFile(t, repoRoot, "demo/full/demo.cfg", "full\n")

	mock := &deps.MockRunner{
		Expectations: []deps.MockExpectation{
			{Argv: []string{"rg", "--version"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("ripgrep 14.1.0\n")}},
			{Argv: []string{"rg", "--version"}, Result: deps.RunResult{ExitCode: 1}},
			{Argv: []string{"apt-get", "install", "-y", "ripgrep"}, Result: deps.RunResult{ExitCode: 0}},
			{Argv: []string{"rg", "--version"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("ripgrep 14.1.0\n")}},
		},
	}
	withMockRunner(t, mock)

	out, err := runE2ECmd(t,
		"--state", statePath,
		"--yes",
		"install", "demo",
		"--profile", "minimal",
	)
	require.NoError(t, err, "initial install failed: out=%s", out)
	require.Equal(t, 1, len(mock.Calls), "initial install: only ripgrep probe")

	resetInstallFlags()
	out, err = runE2ECmd(t,
		"--state", statePath,
		"--yes",
		"switch", "demo", "full",
	)
	require.NoError(t, err, "switch failed: out=%s", out)

	assert.Equal(t, len(mock.Expectations), len(mock.Calls),
		"switch must re-evaluate deps (probe+install+post-probe)")

	s, err := state.Load(statePath)
	require.NoError(t, err)
	pkg, ok := s["demo"]
	require.True(t, ok)
	assert.Equal(t, "full", pkg.Profile, "switch must update profile in state")
}

func TestE2E_UninstallClearsState(t *testing.T) {
	ensureLinuxOrDarwin(t)
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, statePath, homeDir := setupE2ERepo(t)
	installDepsFixture(t, repoRoot)

	preState := state.State{
		"nvim": state.PackageState{
			Profile: "default",
			InstalledLinks: []state.InstalledLink{
				{
					Source: filepath.Join(repoRoot, "nvim", "config"),
					Target: filepath.Join(homeDir, ".config", "nvim"),
					IsDir:  true,
				},
			},
			InstalledDependencies: []deps.InstalledDependency{
				{Name: "ripgrep", Version: "14.1.0", Method: "brew", ManagedByEasyrice: true},
			},
		},
	}
	require.NoError(t, state.Save(statePath, preState))
	require.NoError(t, os.MkdirAll(filepath.Join(homeDir, ".config"), 0o755))
	require.NoError(t, os.Symlink(
		filepath.Join(repoRoot, "nvim", "config"),
		filepath.Join(homeDir, ".config", "nvim"),
	))

	out, err := runE2ECmd(t,
		"--state", statePath,
		"--yes",
		"uninstall", "nvim",
	)
	require.NoError(t, err, "uninstall failed: out=%s", out)

	s, err := state.Load(statePath)
	require.NoError(t, err)
	_, ok := s["nvim"]
	assert.False(t, ok, "nvim should be removed from state after uninstall; got %+v", s)
}

func TestE2E_SkipDeps(t *testing.T) {
	ensureLinuxOrDarwin(t)
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, statePath, homeDir := setupE2ERepo(t)
	installDepsFixture(t, repoRoot)

	mock := &deps.MockRunner{}
	withMockRunner(t, mock)

	out, err := runE2ECmd(t,
		"--state", statePath,
		"--yes",
		"install", "nvim",
		"--profile", "default",
		"--skip-deps",
	)
	require.NoError(t, err, "out=%s", out)

	assert.Empty(t, mock.Calls, "deps runner must NOT be invoked with --skip-deps")

	link := filepath.Join(homeDir, ".config", "nvim")
	_, err = os.Lstat(link)
	require.NoError(t, err, "nvim symlink should exist")

	s, err := state.Load(statePath)
	require.NoError(t, err)
	pkg, ok := s["nvim"]
	require.True(t, ok)
	assert.Equal(t, "default", pkg.Profile)
	assert.Empty(t, pkg.InstalledDependencies, "no deps should be recorded with --skip-deps")
}
