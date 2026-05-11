package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/deps"
	"github.com/guneet-xyz/easyrice/internal/repo"
)

func installCommonForSwitch(t *testing.T, repoRoot, statePath string) {
	t.Helper()
	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"install", "mypkg",
		"--profile", "common",
	)
	require.NoError(t, err, "setup install failed: out=%s", out)
}

func TestSwitch_WithYesFlag(t *testing.T) {
	resetInstallFlags()
	repoRoot, statePath, homeDir := setupTestRepo(t)
	installCommonForSwitch(t, repoRoot, statePath)

	resetInstallFlags()
	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"switch", "mypkg", "macbook",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Plan: uninstall mypkg")
	assert.Contains(t, out, "Plan: install mypkg")
	assert.Contains(t, out, "REMOVE")
	assert.Contains(t, out, "CREATE")

	machineLink := filepath.Join(homeDir, ".config", "mypkg", "machine.toml")
	fi, err := os.Lstat(machineLink)
	require.NoError(t, err, "expected machine.toml symlink after switch")
	assert.NotZero(t, fi.Mode()&os.ModeSymlink, "expected symlink")
}

func TestSwitch_StdinYesProceeds(t *testing.T) {
	resetInstallFlags()
	repoRoot, statePath, homeDir := setupTestRepo(t)
	installCommonForSwitch(t, repoRoot, statePath)

	resetInstallFlags()
	out, err := runInstallCmd(t, "y\n",
		"--state", statePath,
		"switch", "mypkg", "macbook",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Plan: install mypkg")

	machineLink := filepath.Join(homeDir, ".config", "mypkg", "machine.toml")
	_, err = os.Lstat(machineLink)
	require.NoError(t, err, "expected machine.toml symlink after switch")
}

func TestSwitch_StdinNoAborts(t *testing.T) {
	resetInstallFlags()
	repoRoot, statePath, homeDir := setupTestRepo(t)
	installCommonForSwitch(t, repoRoot, statePath)

	resetInstallFlags()
	out, err := runInstallCmd(t, "n\n",
		"--state", statePath,
		"switch", "mypkg", "macbook",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Aborted.")

	machineLink := filepath.Join(homeDir, ".config", "mypkg", "machine.toml")
	_, err = os.Lstat(machineLink)
	assert.True(t, os.IsNotExist(err), "machine.toml should not exist after abort")

	baseLink := filepath.Join(homeDir, ".config", "mypkg", "base.toml")
	_, err = os.Lstat(baseLink)
	require.NoError(t, err, "base.toml should still exist after abort")
}

func TestSwitch_NoArgsErrors(t *testing.T) {
	resetInstallFlags()
	_, err := runInstallCmd(t, "", "switch")
	require.Error(t, err)
}

func TestSwitch_OneArgErrors(t *testing.T) {
	resetInstallFlags()
	_, err := runInstallCmd(t, "", "switch", "mypkg")
	require.Error(t, err)
}

func TestSwitch_ShowsConflictDetails(t *testing.T) {
	resetInstallFlags()
	repoRoot, statePath, homeDir := setupTestRepo(t)
	installCommonForSwitch(t, repoRoot, statePath)

	// machine.toml is only in the macbook profile (not common), so the
	// uninstall phase of switch won't touch it. Pre-create a regular file
	// at its target so the install phase reports a conflict.
	conflictTarget := filepath.Join(homeDir, ".config", "mypkg", "machine.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(conflictTarget), 0o755))
	require.NoError(t, os.WriteFile(conflictTarget, []byte("foreign\n"), 0o644))

	resetInstallFlags()
	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"switch", "mypkg", "macbook",
	)
	require.Error(t, err, "expected error due to conflict")
	assert.Contains(t, out, "CONFLICT")
	assert.Contains(t, out, conflictTarget)
}

func TestSwitch_NotInstalledErrors(t *testing.T) {
	resetInstallFlags()
	_, statePath, _ := setupTestRepo(t)

	_, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"switch", "notinstalled", "macbook",
	)
	require.Error(t, err)
}

func TestSwitch_FolderModeToFileMode(t *testing.T) {
	resetInstallFlags()
	_, statePath, homeDir := setupFolderTestRepo(t)

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"install", "folderpkg",
		"--profile", "common",
	)
	require.NoError(t, err, "install profile A failed: out=%s", out)

	folderTarget := filepath.Join(homeDir, ".config", "folderpkg")
	fi, err := os.Lstat(folderTarget)
	require.NoError(t, err)
	require.NotZero(t, fi.Mode()&os.ModeSymlink, "precondition: profile A should be folder symlink")

	resetInstallFlags()
	out, err = runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"switch", "folderpkg", "filemode",
	)
	require.NoError(t, err, "switch failed: out=%s", out)

	_, err = os.Lstat(folderTarget)
	assert.True(t, os.IsNotExist(err), "folder symlink should be gone after switch; err=%v", err)

	fileLink := filepath.Join(homeDir, "init.conf")
	fi, err = os.Lstat(fileLink)
	require.NoError(t, err, "expected individual file symlink at %s", fileLink)
	assert.NotZero(t, fi.Mode()&os.ModeSymlink, "expected file symlink (not folder)")
	assert.False(t, fi.IsDir(), "should be a file symlink, not a directory")
}

// TestSwitch_NoRepo asserts switch fails once the repo is gone, even when the
// package was previously installed (state.json populated). Switch must consult
// the manifest to validate the new profile.
func TestSwitch_NoRepo(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, statePath, _ := setupTestRepo(t)
	installCommonForSwitch(t, repoRoot, statePath)

	require.NoError(t, os.RemoveAll(repoRoot), "remove repo to simulate missing state")

	resetInstallFlags()
	_, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"switch", "mypkg", "macbook",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repo not initialized",
		"error must indicate repo not initialized; got: %v", err)
}

// TestSwitch_ProfileNotDeclared asserts switch fails when the new profile is
// not declared in the manifest, even though the package is installed.
func TestSwitch_ProfileNotDeclared(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, statePath, _ := setupTestRepo(t)
	installCommonForSwitch(t, repoRoot, statePath)

	resetInstallFlags()
	_, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"switch", "mypkg", "ghost-profile",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ghost-profile",
		"error must name the missing profile; got: %v", err)
	assert.Contains(t, err.Error(), "not defined",
		"error must indicate profile not defined; got: %v", err)
}

func TestSwitch_RepoExistsStatError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission semantics differ on windows")
	}
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	setIsolatedHome(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	repoRoot := repo.DefaultRepoPath()
	parent := filepath.Dir(repoRoot)
	require.NoError(t, os.MkdirAll(filepath.Dir(parent), 0o755))
	require.NoError(t, os.WriteFile(parent, []byte("blocker"), 0o644))

	_, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"switch", "mypkg", "macbook",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check repo")
}

func TestSwitch_ManifestLoadError(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	setIsolatedHome(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeRepoManifest(t, "this is not = valid toml [[[")

	_, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"switch", "mypkg", "macbook",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load manifest")
}

func TestSwitch_OSCheckError(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	setIsolatedHome(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	other := "linux"
	if runtime.GOOS == "linux" {
		other = "darwin"
	}
	writeRepoManifest(t, `schema_version = 1

[packages.mypkg]
description = "Wrong-OS"
supported_os = ["`+other+`"]

[packages.mypkg.profiles.common]
sources = [{path = "common", mode = "file", target = "$HOME"}]
`)

	_, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"switch", "mypkg", "common",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "os check")
}

func TestSwitch_SkipDepsBypassesRunner(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, statePath, _ := setupTestRepo(t)
	installCommonForSwitch(t, repoRoot, statePath)

	mock := &deps.MockRunner{}
	withMockDepsRunner(t, mock)

	resetInstallFlags()
	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"switch", "mypkg", "macbook",
		"--skip-deps",
	)
	flagSwitchSkipDeps = false
	require.NoError(t, err, "out=%s", out)
	assert.Empty(t, mock.Calls, "deps runner must NOT be invoked with --skip-deps")
}

func TestSwitch_DepsLoadStateError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses permission bits")
	}
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	setIsolatedHome(t)
	repoRoot := repo.DefaultRepoPath()
	require.NoError(t, os.MkdirAll(repoRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "rice.toml"), []byte(`schema_version = 1

[packages.mypkg]
description = "x"
supported_os = ["linux", "darwin", "windows"]

[packages.mypkg.profiles.common]
sources = [{path = "common", mode = "file", target = "$HOME"}]
`), 0o644))

	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))
	statePath := filepath.Join(blocker, "state.json")

	_, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"switch", "mypkg", "common",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load state")
}

func TestSwitch_DepsEnsureError(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	homeDir := setIsolatedHome(t)
	_ = homeDir
	repoRoot := repo.DefaultRepoPath()
	statePath := filepath.Join(t.TempDir(), "state.json")

	pkgDir := filepath.Join(repoRoot, "mypkg")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "common", ".config", "mypkg"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "common", ".config", "mypkg", "base.toml"), []byte("base=true\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "rice.toml"), []byte(`schema_version = 1

[packages.mypkg]
description = "deps pkg"
supported_os = ["linux", "darwin", "windows"]
dependencies = [{name = "neovim"}]

[packages.mypkg.profiles.common]
sources = [{path = "common", mode = "file", target = "$HOME"}]
`), 0o644))

	mock := &deps.MockRunner{
		Expectations: []deps.MockExpectation{
			{Argv: []string{"nvim", "--version"}, Err: errors.New("probe boom")},
		},
	}
	withMockDepsRunner(t, mock)

	_, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"switch", "mypkg", "common",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ensure dependencies")
}

func TestSwitch_PromptReaderError(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, statePath, _ := setupTestRepo(t)
	installCommonForSwitch(t, repoRoot, statePath)

	resetInstallFlags()

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetIn(iotest.ErrReader(errors.New("read failure")))
	rootCmd.SetArgs([]string{"--state", statePath, "switch", "mypkg", "macbook"})
	err := rootCmd.Execute()
	rootCmd.SetIn(os.Stdin)
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	require.Error(t, err, "out=%s", buf.String())
	assert.Contains(t, err.Error(), "read failure")
}

func TestSwitch_ExecutePlanError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses permission bits")
	}
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, statePath, _ := setupTestRepo(t)
	installCommonForSwitch(t, repoRoot, statePath)

	require.NoError(t, os.Chmod(statePath, 0o444))
	t.Cleanup(func() { _ = os.Chmod(statePath, 0o644) })

	resetInstallFlags()
	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"switch", "mypkg", "macbook",
	)
	require.Error(t, err, "out=%s", out)
	assert.Contains(t, err.Error(), "execute plan")
}
