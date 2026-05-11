package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
