package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
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

func setupFolderTestRepo(t *testing.T) (repoRoot, statePath, homeDir string) {
	t.Helper()
	homeDir = setIsolatedHome(t)
	repoRoot = repo.DefaultRepoPath()
	statePath = filepath.Join(t.TempDir(), "state.json")

	pkgDir := filepath.Join(repoRoot, "folderpkg")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "cfg"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(pkgDir, "cfg", "init.conf"),
		[]byte("k=v\n"), 0o644))

	manifest := `schema_version = 1

[packages.folderpkg]
description = "Folder-mode test package"
supported_os = ["linux", "darwin", "windows"]

[packages.folderpkg.profiles.common]
sources = [{path = "cfg", mode = "folder", target = "$HOME/.config/folderpkg"}]

[packages.folderpkg.profiles.filemode]
sources = [{path = "cfg", mode = "file", target = "$HOME"}]
`
	require.NoError(t, os.WriteFile(
		filepath.Join(repoRoot, "rice.toml"), []byte(manifest), 0o644))

	return
}

func resetInstallFlags() {
	flagProfile = ""
	flagYes = false
	flagState = state.DefaultPath()
	flagLogLevel = ""
}

func setupTestRepo(t *testing.T) (repoRoot, statePath, homeDir string) {
	t.Helper()
	homeDir = setIsolatedHome(t)
	repoRoot = repo.DefaultRepoPath()
	statePath = filepath.Join(t.TempDir(), "state.json")

	pkgDir := filepath.Join(repoRoot, "mypkg")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "common", ".config", "mypkg"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "macbook", ".config", "mypkg"), 0o755))

	manifest := `schema_version = 1

[packages.mypkg]
description = "Test package"
supported_os = ["linux", "darwin", "windows"]

[packages.mypkg.profiles.common]
sources = [{path = "common", mode = "file", target = "$HOME"}]

[packages.mypkg.profiles.macbook]
sources = [
  {path = "common", mode = "file", target = "$HOME"},
  {path = "macbook", mode = "file", target = "$HOME"},
]
`
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "rice.toml"), []byte(manifest), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "common", ".config", "mypkg", "base.toml"), []byte("base=true\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "macbook", ".config", "mypkg", "machine.toml"), []byte("machine=true\n"), 0o644))

	return
}

func runInstallCmd(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetIn(strings.NewReader(stdin))
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	rootCmd.SetIn(os.Stdin)
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	return buf.String(), err
}

func TestInstall_WithYesFlag(t *testing.T) {
	resetInstallFlags()
	_, statePath, homeDir := setupTestRepo(t)

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"install", "mypkg",
		"--profile", "common",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Plan: install mypkg")
	assert.Contains(t, out, "CREATE")

	link := filepath.Join(homeDir, ".config", "mypkg", "base.toml")
	fi, err := os.Lstat(link)
	require.NoError(t, err)
	assert.NotZero(t, fi.Mode()&os.ModeSymlink, "expected symlink at %s", link)
}

func TestInstall_StdinYesProceeds(t *testing.T) {
	resetInstallFlags()
	_, statePath, homeDir := setupTestRepo(t)

	out, err := runInstallCmd(t, "y\n",
		"--state", statePath,
		"install", "mypkg",
		"--profile", "common",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Plan: install mypkg")

	link := filepath.Join(homeDir, ".config", "mypkg", "base.toml")
	_, err = os.Lstat(link)
	require.NoError(t, err, "symlink should exist")
}

func TestInstall_StdinNoAborts(t *testing.T) {
	resetInstallFlags()
	_, statePath, homeDir := setupTestRepo(t)

	out, err := runInstallCmd(t, "n\n",
		"--state", statePath,
		"install", "mypkg",
		"--profile", "common",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Aborted.")

	link := filepath.Join(homeDir, ".config", "mypkg", "base.toml")
	_, err = os.Lstat(link)
	assert.True(t, os.IsNotExist(err), "symlink should NOT exist; lstat err=%v", err)
}

func TestInstall_NoArgsErrors(t *testing.T) {
	resetInstallFlags()
	_, err := runInstallCmd(t, "", "install")
	require.Error(t, err)
}

func TestInstall_ShowsConflictDetails(t *testing.T) {
	resetInstallFlags()
	_, statePath, homeDir := setupTestRepo(t)

	conflictTarget := filepath.Join(homeDir, ".config", "mypkg", "base.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(conflictTarget), 0o755))
	require.NoError(t, os.WriteFile(conflictTarget, []byte("existing\n"), 0o644))

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"install", "mypkg",
		"--profile", "common",
	)
	require.Error(t, err, "expected error due to conflict")
	assert.Contains(t, out, "CONFLICT")
	assert.Contains(t, out, conflictTarget)
}

func TestInstall_WithProfileFlag(t *testing.T) {
	resetInstallFlags()
	_, statePath, homeDir := setupTestRepo(t)

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"install", "mypkg",
		"--profile", "macbook",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "profile: macbook")

	for _, rel := range []string{".config/mypkg/base.toml", ".config/mypkg/machine.toml"} {
		_, err := os.Lstat(filepath.Join(homeDir, rel))
		assert.NoError(t, err, "expected symlink %s", rel)
	}
}

func TestInstall_FolderMode_CreatesSingleSymlink(t *testing.T) {
	resetInstallFlags()
	repoRoot, statePath, homeDir := setupFolderTestRepo(t)

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"install", "folderpkg",
		"--profile", "common",
	)
	require.NoError(t, err, "out=%s", out)

	target := filepath.Join(homeDir, ".config", "folderpkg")
	fi, err := os.Lstat(target)
	require.NoError(t, err)
	assert.NotZero(t, fi.Mode()&os.ModeSymlink, "expected symlink at %s", target)

	wantSrc, err := filepath.Abs(filepath.Join(repoRoot, "folderpkg", "cfg"))
	require.NoError(t, err)
	got, err := os.Readlink(target)
	require.NoError(t, err)
	assert.Equal(t, wantSrc, got, "symlink should point to source dir")

	st, err := state.Load(statePath)
	require.NoError(t, err)
	pkg, ok := st["folderpkg"]
	require.True(t, ok, "folderpkg missing from state")
	require.Len(t, pkg.InstalledLinks, 1)
	assert.True(t, pkg.InstalledLinks[0].IsDir, "InstalledLinks[0].IsDir should be true")
}

// fileSHA256 returns the sha256 of file contents, or "" if the file does not exist.
func fileSHA256(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return ""
	}
	require.NoError(t, err)
	sum := sha256.Sum256(data)
	return string(sum[:])
}

// TestInstall_NoRepo asserts install fails with ErrRepoNotInitialized when
// the managed repo has not been cloned.
func TestInstall_NoRepo(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	setIsolatedHome(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	_, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"install", "anything",
		"--profile", "common",
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, repo.ErrRepoNotInitialized),
		"expected ErrRepoNotInitialized, got: %v", err)

	_, statErr := os.Stat(statePath)
	assert.True(t, os.IsNotExist(statErr), "state.json must not be created on no-repo failure")
}

// TestInstall_PackageNotDeclared asserts install fails when the requested
// package is absent from rice.toml.
func TestInstall_PackageNotDeclared(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	_, statePath, _ := setupTestRepo(t)

	_, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"install", "ghost-package",
		"--profile", "common",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ghost-package",
		"error must name the missing package; got: %v", err)
	assert.Contains(t, err.Error(), "not declared",
		"error must indicate package not declared; got: %v", err)

	_, statErr := os.Stat(statePath)
	assert.True(t, os.IsNotExist(statErr), "state.json must not be created on package-not-declared failure")
}

// TestInstall_ProfileNotDeclared asserts install fails when the package
// exists but the requested profile is not defined.
func TestInstall_ProfileNotDeclared(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	_, statePath, _ := setupTestRepo(t)

	_, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"install", "mypkg",
		"--profile", "ghost-profile",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ghost-profile",
		"error must name the missing profile; got: %v", err)
	assert.Contains(t, err.Error(), "not defined",
		"error must indicate profile not defined; got: %v", err)

	_, statErr := os.Stat(statePath)
	assert.True(t, os.IsNotExist(statErr), "state.json must not be created on profile-not-declared failure")
}

// TestInstall_ConflictAbortsWithoutYes asserts that when a pre-existing file
// at the install target produces a conflict, the install aborts and
// state.json is NOT mutated.
func TestInstall_ConflictAbortsWithoutYes(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	_, statePath, homeDir := setupTestRepo(t)

	conflictTarget := filepath.Join(homeDir, ".config", "mypkg", "base.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(conflictTarget), 0o755))
	require.NoError(t, os.WriteFile(conflictTarget, []byte("pre-existing\n"), 0o644))

	stateBefore := fileSHA256(t, statePath)

	out, err := runInstallCmd(t, "n\n",
		"--state", statePath,
		"install", "mypkg",
		"--profile", "common",
	)
	require.Error(t, err, "expected conflict error; out=%s", err)
	assert.Contains(t, out, "CONFLICT")

	// Conflict should be reported BEFORE any prompt or state mutation.
	stateAfter := fileSHA256(t, statePath)
	assert.Equal(t, stateBefore, stateAfter,
		"state.json must NOT be mutated after a conflict-abort")

	// The pre-existing file must still be intact and a regular file.
	fi, err := os.Lstat(conflictTarget)
	require.NoError(t, err)
	assert.Zero(t, fi.Mode()&os.ModeSymlink,
		"pre-existing file must not be replaced with a symlink")
}

// TestInstall_OSNotSupported asserts install fails when the package's
// supported_os excludes the current OS.
func TestInstall_OSNotSupported(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	homeDir := setIsolatedHome(t)
	repoRoot := repo.DefaultRepoPath()
	statePath := filepath.Join(t.TempDir(), "state.json")

	pkgDir := filepath.Join(repoRoot, "alienpkg")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "common"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "common", "f.txt"), []byte("x"), 0o644))

	otherOS := "windows"
	if runtime.GOOS == "windows" {
		otherOS = "linux"
	}
	manifest := `schema_version = 1

[packages.alienpkg]
description = "OS-restricted package"
supported_os = ["` + otherOS + `"]

[packages.alienpkg.profiles.common]
sources = [{path = "common", mode = "file", target = "$HOME"}]
`
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "rice.toml"), []byte(manifest), 0o644))

	_, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"install", "alienpkg",
		"--profile", "common",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "os check",
		"error must be wrapped with 'os check'; got: %v", err)
	assert.Contains(t, err.Error(), "alienpkg",
		"error must name the package; got: %v", err)

	_, statErr := os.Stat(statePath)
	assert.True(t, os.IsNotExist(statErr), "state.json must not be created on os-check failure")
	_ = homeDir
}

// TestInstall_MalformedManifestErrors asserts install surfaces a wrapped
// error when rice.toml fails to parse.
func TestInstall_MalformedManifestErrors(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	setIsolatedHome(t)
	repoRoot := repo.DefaultRepoPath()
	statePath := filepath.Join(t.TempDir(), "state.json")

	require.NoError(t, os.MkdirAll(repoRoot, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(repoRoot, "rice.toml"),
		[]byte("this is = not [valid toml\n"), 0o644))

	_, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"install", "anything",
		"--profile", "common",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load manifest",
		"error must be wrapped with 'load manifest'; got: %v", err)
}

func withMockDepsRunner(t *testing.T, m *deps.MockRunner) {
	t.Helper()
	prev := DepsRunner
	DepsRunner = m
	t.Cleanup(func() { DepsRunner = prev })
}

func setupDepsTestRepo(t *testing.T) (repoRoot, statePath, homeDir string) {
	t.Helper()
	homeDir = setIsolatedHome(t)
	repoRoot = repo.DefaultRepoPath()
	statePath = filepath.Join(t.TempDir(), "state.json")

	pkgDir := filepath.Join(repoRoot, "mypkg")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "common", ".config", "mypkg"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(pkgDir, "common", ".config", "mypkg", "base.toml"),
		[]byte("base=true\n"), 0o644))

	manifest := `schema_version = 1

[packages.mypkg]
description = "Pkg with deps"
supported_os = ["linux", "darwin", "windows"]
dependencies = [{name = "neovim"}]

[packages.mypkg.profiles.common]
sources = [{path = "common", mode = "file", target = "$HOME"}]
`
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "rice.toml"), []byte(manifest), 0o644))
	return
}

// TestInstall_WithDeps_StatePersisted asserts that when `rice install` installs
// a missing dependency, state.json gains BOTH installed_links AND
// installed_dependencies entries for the package.
func TestInstall_WithDeps_StatePersisted(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("dependency install path uses brew (no-root); skipping non-darwin runners")
	}
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	_, statePath, homeDir := setupDepsTestRepo(t)

	mock := &deps.MockRunner{
		Expectations: []deps.MockExpectation{
			{
				Argv:   []string{"nvim", "--version"},
				Result: deps.RunResult{ExitCode: 1, Combined: []byte("")},
			},
			{
				Argv:   []string{"brew", "install", "neovim"},
				Result: deps.RunResult{ExitCode: 0},
			},
			{
				Argv:   []string{"nvim", "--version"},
				Result: deps.RunResult{ExitCode: 0, Combined: []byte("NVIM v0.10.0\n")},
			},
		},
	}
	withMockDepsRunner(t, mock)

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"install", "mypkg",
		"--profile", "common",
	)
	require.NoError(t, err, "out=%s", out)

	link := filepath.Join(homeDir, ".config", "mypkg", "base.toml")
	fi, err := os.Lstat(link)
	require.NoError(t, err, "symlink should exist after install")
	assert.NotZero(t, fi.Mode()&os.ModeSymlink, "expected symlink at %s", link)

	data, err := os.ReadFile(statePath)
	require.NoError(t, err, "state.json must be created")

	var s state.State
	require.NoError(t, json.Unmarshal(data, &s))

	pkgState, ok := s["mypkg"]
	require.True(t, ok, "state must contain mypkg entry; got: %s", string(data))

	assert.NotEmpty(t, pkgState.InstalledLinks, "InstalledLinks must be populated; got: %s", string(data))
	require.Len(t, pkgState.InstalledDependencies, 1, "InstalledDependencies must record the installed dep; got: %s", string(data))
	assert.Equal(t, "neovim", pkgState.InstalledDependencies[0].Name)
	assert.Equal(t, "0.10.0", pkgState.InstalledDependencies[0].Version)
	assert.Equal(t, "brew", pkgState.InstalledDependencies[0].Method)
	assert.True(t, pkgState.InstalledDependencies[0].ManagedByEasyrice)
}

// TestInstall_EnsureDepsError asserts that when the dependency probe surfaces
// an error, install exits non-zero, no symlinks are created, and state.json
// is not written.
func TestInstall_EnsureDepsError(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	_, statePath, homeDir := setupDepsTestRepo(t)

	mock := &deps.MockRunner{
		Expectations: []deps.MockExpectation{
			{
				Argv: []string{"nvim", "--version"},
				Err:  errors.New("simulated probe failure"),
			},
		},
	}
	withMockDepsRunner(t, mock)

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"install", "mypkg",
		"--profile", "common",
	)
	require.Error(t, err, "install must fail when dep probe errors; out=%s", out)
	assert.Contains(t, err.Error(), "ensure dependencies",
		"error must be wrapped with 'ensure dependencies'; got: %v", err)

	link := filepath.Join(homeDir, ".config", "mypkg", "base.toml")
	_, lstatErr := os.Lstat(link)
	assert.True(t, os.IsNotExist(lstatErr),
		"no symlink should exist when dep step fails; lstat err=%v", lstatErr)

	_, statErr := os.Stat(statePath)
	assert.True(t, os.IsNotExist(statErr),
		"state.json must not be created when EnsureDependencies fails; statErr=%v", statErr)
}
