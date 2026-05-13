package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
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

// setupBareUpstreamFromTree builds a bare upstream git repo whose initial
// commit contains the given file map (relative path -> contents). Returns a
// file:// URL suitable for `git clone` and `git submodule add`.
func setupBareUpstreamFromTree(t *testing.T, label string, files map[string]string) string {
	t.Helper()
	gitOnPath(t)

	bareDir := t.TempDir()
	bare := filepath.Join(bareDir, label+".git")
	require.NoError(t, os.MkdirAll(bare, 0o755))
	gitRunRemote(t, bareDir, "init", "--bare", "-b", "main", label+".git")

	wt := t.TempDir()
	gitRunRemote(t, wt, "init", "-b", "main")
	gitRunRemote(t, wt, "config", "user.email", "test@test.com")
	gitRunRemote(t, wt, "config", "user.name", "Test")

	addArgs := []string{"add", "--"}
	for rel, content := range files {
		full := filepath.Join(wt, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
		addArgs = append(addArgs, rel)
	}
	require.Greater(t, len(addArgs), 2, "no files in upstream tree")
	gitRunRemote(t, wt, addArgs...)
	gitRunRemote(t, wt, "commit", "-m", "init "+label)
	gitRunRemote(t, wt, "remote", "add", "origin", "file://"+bare)
	gitRunRemote(t, wt, "push", "origin", "main")
	return "file://" + bare
}

// TestE2E_FullManagedRiceFlow exercises the entire managed-rice lifecycle in
// one test: init -> install all -> remote add -> import + overlay -> install
// with profile -> status -> doctor -> remote remove blocked (in-use) -> rewrite
// to drop import -> install switch back -> remote remove succeeds -> deleted
// `switch` command surfaces "unknown command".
//
// All output is recorded to a transcript and saved under
// .sisyphus/evidence/task-16-e2e-transcript.txt for plan verification.
func TestE2E_FullManagedRiceFlow(t *testing.T) {
	ensureLinuxOrDarwin(t)
	gitOnPath(t)

	homeDir := setIsolatedHome(t)
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	transcript := &bytes.Buffer{}
	record := func(label, body string, err error) {
		fmt.Fprintf(transcript, "=== %s ===\n", label)
		if err != nil {
			fmt.Fprintf(transcript, "ERR: %v\n", err)
		}
		if body != "" {
			fmt.Fprintf(transcript, "%s\n", strings.TrimRight(body, "\n"))
		}
		fmt.Fprintln(transcript)
	}
	t.Cleanup(func() {
		// Best-effort: persist transcript for plan evidence. Walk up from
		// cli/ to repo root before writing.
		wd, err := os.Getwd()
		if err != nil {
			return
		}
		evidenceDir := filepath.Join(filepath.Dir(wd), ".sisyphus", "evidence")
		if mkErr := os.MkdirAll(evidenceDir, 0o755); mkErr != nil {
			return
		}
		_ = os.WriteFile(
			filepath.Join(evidenceDir, "task-16-e2e-transcript.txt"),
			transcript.Bytes(),
			0o644,
		)
	})

	repoRoot := repo.DefaultRepoPath()

	// ----- step 1: bare upstream A (main rice, mirrors manifest_valid_v2 spirit) -----
	mainManifest := `schema_version = 1

[packages.ghostty]
description = "Ghostty terminal emulator configuration"
supported_os = ["linux", "darwin", "windows"]

[packages.ghostty.profiles.default]
sources = [{path = "common", mode = "file", target = "$HOME/.config/ghostty"}]

[packages.nvim]
description = "Neovim configuration"
supported_os = ["linux", "darwin", "windows"]

[packages.nvim.profiles.default]
sources = [{path = "config", mode = "folder", target = "$HOME/.config/nvim"}]
`
	upstreamA := setupBareUpstreamFromTree(t, "main", map[string]string{
		"rice.toml":             mainManifest,
		"ghostty/common/config": "font-size = 14\n",
		"nvim/config/init.lua":  "-- main rice nvim\n",
	})

	// ----- step 2: bare upstream B (remote rice, file-mode so overlay works) -----
	remoteManifest := `schema_version = 1

[packages.nvim]
description = "Neovim configuration (kickstart)"
supported_os = ["linux", "darwin", "windows"]

[packages.nvim.profiles.default]
sources = [{path = "config", mode = "file", target = "$HOME/.config/nvim"}]
`
	upstreamB := setupBareUpstreamFromTree(t, "kickstart", map[string]string{
		"rice.toml":            remoteManifest,
		"nvim/config/init.lua": "-- kickstart init.lua\n",
		"nvim/config/lazy.lua": "-- kickstart lazy.lua\n",
	})

	// ----- step 3: rice init <upstreamA> -----
	out, err := runRemoteCmd(t, "init", upstreamA)
	record("step03 init", out, err)
	require.NoError(t, err, "init: %s", out)
	assert.Contains(t, out, "Cloned to")

	gitDir := filepath.Join(repoRoot, ".git")
	_, err = os.Stat(gitDir)
	require.NoError(t, err, "managed repo .git must exist at %s", gitDir)

	logOut, lerr := exec.Command("git", "-C", repoRoot, "log", "--oneline").CombinedOutput()
	require.NoError(t, lerr, "git log after init: %s", logOut)
	require.NotEmpty(t, strings.TrimSpace(string(logOut)), "managed repo must have commits after init")

	// Configure local user identity for subsequent auto-commits in case env
	// vars are stripped by any sub-process.
	gitRunRemote(t, repoRoot, "config", "user.email", "test@test.com")
	gitRunRemote(t, repoRoot, "config", "user.name", "Test")

	// ----- step 4: rice install (no args) -- converge all packages -----
	statePath := filepath.Join(t.TempDir(), "state.json")
	out, err = runRemoteCmd(t, "--state", statePath, "--yes", "install", "--profile", "default")
	record("step04 install all", out, err)
	require.NoError(t, err, "install all: %s", out)

	st, err := state.Load(statePath)
	require.NoError(t, err)
	for _, pkg := range []string{"ghostty", "nvim"} {
		ps, ok := st[pkg]
		require.True(t, ok, "package %q must be in state after install all", pkg)
		assert.Equal(t, "default", ps.Profile, "package %q profile", pkg)
	}

	// ----- step 5: rice remote add <upstreamB> --name kickstart -----
	out, err = runRemoteCmd(t, "remote", "add", upstreamB, "--name", "kickstart")
	record("step05 remote add", out, err)
	require.NoError(t, err, "remote add: %s", out)
	assert.Contains(t, out, "Added remote rice")

	gitmodules, err := os.ReadFile(filepath.Join(repoRoot, ".gitmodules"))
	require.NoError(t, err, ".gitmodules must exist after remote add")
	assert.Contains(t, string(gitmodules), "kickstart")

	_, err = os.Stat(filepath.Join(repoRoot, "remotes", "kickstart", "rice.toml"))
	require.NoError(t, err, "remotes/kickstart/rice.toml must exist after remote add")

	subjOut, subjErr := exec.Command("git", "-C", repoRoot, "log", "-1", "--format=%s").CombinedOutput()
	require.NoError(t, subjErr, "git log subject: %s", subjOut)
	assert.Equal(t, "Add remote rice kickstart", strings.TrimSpace(string(subjOut)))

	// ----- step 6: add kickstart-overlay profile to nvim with import + local overlay -----
	overlayManifest := `schema_version = 1

[packages.ghostty]
description = "Ghostty terminal emulator configuration"
supported_os = ["linux", "darwin", "windows"]

[packages.ghostty.profiles.default]
sources = [{path = "common", mode = "file", target = "$HOME/.config/ghostty"}]

[packages.nvim]
description = "Neovim configuration"
supported_os = ["linux", "darwin", "windows"]

[packages.nvim.profiles.default]
sources = [{path = "config", mode = "folder", target = "$HOME/.config/nvim"}]

[packages.nvim.profiles.kickstart-overlay]
import = "remotes/kickstart#nvim.default"
sources = [{path = "overlay", mode = "file", target = "$HOME/.config/nvim"}]
`
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "rice.toml"), []byte(overlayManifest), 0o644))
	overlayDir := filepath.Join(repoRoot, "nvim", "overlay")
	require.NoError(t, os.MkdirAll(overlayDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(overlayDir, "init.lua"),
		[]byte("-- LOCAL OVERLAY init.lua\n"), 0o644))
	gitRunRemote(t, repoRoot, "add", "--", "rice.toml", "nvim/overlay/init.lua")
	gitRunRemote(t, repoRoot, "commit", "-m", "add kickstart-overlay profile")

	staleLink := filepath.Join(homeDir, ".config", "nvim")
	if fi, lerr := os.Lstat(staleLink); lerr == nil && fi.Mode()&os.ModeSymlink != 0 {
		require.NoError(t, os.Remove(staleLink))
	}
	st, err = state.Load(statePath)
	require.NoError(t, err)
	delete(st, "nvim")
	require.NoError(t, state.Save(statePath, st))

	// ----- step 7: rice install nvim --profile kickstart-overlay -----
	out, err = runRemoteCmd(t, "--state", statePath, "--yes", "install", "nvim", "--profile", "kickstart-overlay")
	record("step07 install nvim --profile kickstart-overlay", out, err)
	require.NoError(t, err, "install kickstart-overlay: %s", out)

	st, err = state.Load(statePath)
	require.NoError(t, err)
	nvimPS, ok := st["nvim"]
	require.True(t, ok, "nvim must be in state after kickstart-overlay install")
	assert.Equal(t, "kickstart-overlay", nvimPS.Profile)

	// At least one installed link must point under remotes/kickstart/...
	remotePrefix := filepath.Join(repoRoot, "remotes", "kickstart") + string(filepath.Separator)
	var sawRemote bool
	for _, link := range nvimPS.InstalledLinks {
		if strings.HasPrefix(link.Source, remotePrefix) {
			sawRemote = true
			break
		}
	}
	assert.True(t, sawRemote, "expected at least one installed link under %s; got %+v", remotePrefix, nvimPS.InstalledLinks)

	// init.lua under target should resolve to the LOCAL overlay file (last-wins).
	initLink := filepath.Join(homeDir, ".config", "nvim", "init.lua")
	dst, rerr := os.Readlink(initLink)
	require.NoError(t, rerr, "init.lua should be a symlink at %s", initLink)
	assert.Equal(t, filepath.Join(overlayDir, "init.lua"), dst,
		"local overlay must override imported init.lua via file-mode last-wins")

	// ----- step 8: rice status -----
	out, err = runRemoteCmd(t, "--state", statePath, "status")
	record("step08 status", out, err)
	require.NoError(t, err, "status: %s", out)
	assert.Contains(t, out, "Git:", "status output should include git header")
	assert.Contains(t, out, "ghostty", "status should list ghostty package")
	assert.Contains(t, out, "nvim", "status should list nvim package")

	// ----- step 9: rice doctor -----
	out, err = runRemoteCmd(t, "--state", statePath, "doctor")
	record("step09 doctor", out, err)
	require.NoError(t, err, "doctor: %s", out)

	// ----- step 10: rice remote remove kickstart -- BLOCKED because of import -----
	out, err = runRemoteCmd(t, "remote", "remove", "kickstart")
	record("step10 remote remove (expect ErrRemoteInUse)", out, err)
	require.Error(t, err, "remote remove must fail while import exists; out=%s", out)
	assert.True(t, errors.Is(err, repo.ErrRemoteInUse),
		"expected ErrRemoteInUse, got %v (out=%s)", err, out)
	assert.Contains(t, err.Error(), "nvim.kickstart-overlay",
		"error should name the importing profile")

	// ----- step 11: edit rice.toml to drop the kickstart-overlay profile + commit -----
	cleanManifest := `schema_version = 1

[packages.ghostty]
description = "Ghostty terminal emulator configuration"
supported_os = ["linux", "darwin", "windows"]

[packages.ghostty.profiles.default]
sources = [{path = "common", mode = "file", target = "$HOME/.config/ghostty"}]

[packages.nvim]
description = "Neovim configuration"
supported_os = ["linux", "darwin", "windows"]

[packages.nvim.profiles.default]
sources = [{path = "config", mode = "folder", target = "$HOME/.config/nvim"}]
`
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "rice.toml"), []byte(cleanManifest), 0o644))
	gitRunRemote(t, repoRoot, "add", "--", "rice.toml")
	gitRunRemote(t, repoRoot, "commit", "-m", "remove kickstart-overlay profile")

	configNvim := filepath.Join(homeDir, ".config", "nvim")
	require.NoError(t, os.RemoveAll(configNvim))
	st, err = state.Load(statePath)
	require.NoError(t, err)
	delete(st, "nvim")
	require.NoError(t, state.Save(statePath, st))

	// ----- step 12: rice install nvim --profile default (switch back) -----
	out, err = runRemoteCmd(t, "--state", statePath, "--yes", "install", "nvim", "--profile", "default")
	record("step12 install nvim --profile default", out, err)
	require.NoError(t, err, "install default: %s", out)

	st, err = state.Load(statePath)
	require.NoError(t, err)
	nvimPS, ok = st["nvim"]
	require.True(t, ok, "nvim must be in state after switch back")
	assert.Equal(t, "default", nvimPS.Profile)

	// nvim target should now be a folder-mode dir symlink to <repo>/nvim/config.
	dst2, rerr2 := os.Readlink(configNvim)
	require.NoError(t, rerr2, "nvim should be a folder-mode symlink at %s", configNvim)
	assert.Equal(t, filepath.Join(repoRoot, "nvim", "config"), dst2)

	// ----- step 13: rice remote remove kickstart -- now succeeds -----
	out, err = runRemoteCmd(t, "remote", "remove", "kickstart")
	record("step13 remote remove (expect success)", out, err)
	require.NoError(t, err, "remote remove: %s", out)

	_, statErr := os.Stat(filepath.Join(repoRoot, "remotes", "kickstart"))
	assert.True(t, os.IsNotExist(statErr), "remotes/kickstart must be gone after remove")

	subjOut, subjErr = exec.Command("git", "-C", repoRoot, "log", "-1", "--format=%s").CombinedOutput()
	require.NoError(t, subjErr, "git log subject after remove: %s", subjOut)
	assert.Equal(t, "Remove remote rice kickstart", strings.TrimSpace(string(subjOut)))

	// ----- step 14: rice switch -- deleted; cobra should report unknown command -----
	out, err = runRemoteCmd(t, "switch", "nvim", "default")
	record("step14 switch (expect unknown command)", out, err)
	require.Error(t, err, "switch must be removed; out=%s", out)
	combined := strings.ToLower(out + " " + err.Error())
	assert.Contains(t, combined, "unknown command",
		"expected 'unknown command' diagnostic for removed `switch`; got out=%s err=%v", out, err)
}
