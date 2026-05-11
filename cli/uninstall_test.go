package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func installForUninstall(t *testing.T, repoRoot, statePath string) {
	t.Helper()
	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"install", "mypkg",
		"--profile", "common",
	)
	require.NoError(t, err, "setup install failed: out=%s", out)
}

func TestUninstall_WithYesFlag(t *testing.T) {
	resetInstallFlags()
	repoRoot, statePath, homeDir := setupTestRepo(t)
	installForUninstall(t, repoRoot, statePath)

	link := filepath.Join(homeDir, ".config", "mypkg", "base.toml")
	_, err := os.Lstat(link)
	require.NoError(t, err, "precondition: link should exist after install")

	resetInstallFlags()
	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"uninstall", "mypkg",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Plan: uninstall mypkg")
	assert.Contains(t, out, "REMOVE")

	_, err = os.Lstat(link)
	assert.True(t, os.IsNotExist(err), "symlink should be removed; lstat err=%v", err)
}

func TestUninstall_StdinYesProceeds(t *testing.T) {
	resetInstallFlags()
	repoRoot, statePath, homeDir := setupTestRepo(t)
	installForUninstall(t, repoRoot, statePath)

	link := filepath.Join(homeDir, ".config", "mypkg", "base.toml")

	resetInstallFlags()
	out, err := runInstallCmd(t, "y\n",
		"--state", statePath,
		"uninstall", "mypkg",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Plan: uninstall mypkg")

	_, err = os.Lstat(link)
	assert.True(t, os.IsNotExist(err), "symlink should be removed")
}

func TestUninstall_StdinNoAborts(t *testing.T) {
	resetInstallFlags()
	repoRoot, statePath, homeDir := setupTestRepo(t)
	installForUninstall(t, repoRoot, statePath)

	link := filepath.Join(homeDir, ".config", "mypkg", "base.toml")

	resetInstallFlags()
	out, err := runInstallCmd(t, "n\n",
		"--state", statePath,
		"uninstall", "mypkg",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Aborted.")

	_, err = os.Lstat(link)
	require.NoError(t, err, "symlink should still exist after abort")
}

func TestUninstall_NoArgsErrors(t *testing.T) {
	resetInstallFlags()
	_, err := runInstallCmd(t, "", "uninstall")
	require.Error(t, err)
}

func TestUninstall_NotInstalledErrors(t *testing.T) {
	resetInstallFlags()
	_, statePath, _ := setupTestRepo(t)

	_, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"uninstall", "notinstalled",
	)
	require.Error(t, err)
}

func installFolderpkg(t *testing.T, repoRoot, statePath string) {
	t.Helper()
	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"install", "folderpkg",
		"--profile", "common",
	)
	require.NoError(t, err, "setup install failed: out=%s", out)
}

func TestUninstall_FolderMode_RemovesSymlinkOnly(t *testing.T) {
	resetInstallFlags()
	repoRoot, statePath, homeDir := setupFolderTestRepo(t)
	installFolderpkg(t, repoRoot, statePath)

	target := filepath.Join(homeDir, ".config", "folderpkg")
	_, err := os.Lstat(target)
	require.NoError(t, err, "precondition: folder symlink should exist")

	resetInstallFlags()
	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"uninstall", "folderpkg",
	)
	require.NoError(t, err, "out=%s", out)

	_, err = os.Lstat(target)
	assert.True(t, os.IsNotExist(err), "folder symlink should be removed; lstat err=%v", err)

	srcFile := filepath.Join(repoRoot, "folderpkg", "cfg", "init.conf")
	_, err = os.Stat(srcFile)
	require.NoError(t, err, "source file should remain untouched")
}

func TestUninstall_FolderMode_SkipsWhenReplacedWithDir(t *testing.T) {
	resetInstallFlags()
	repoRoot, statePath, homeDir := setupFolderTestRepo(t)
	installFolderpkg(t, repoRoot, statePath)

	target := filepath.Join(homeDir, ".config", "folderpkg")
	require.NoError(t, os.Remove(target), "remove symlink")
	require.NoError(t, os.MkdirAll(target, 0o755))
	userFile := filepath.Join(target, "user.conf")
	require.NoError(t, os.WriteFile(userFile, []byte("user-data\n"), 0o644))

	resetInstallFlags()
	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"uninstall", "folderpkg",
	)
	require.NoError(t, err, "uninstall should not fail when target replaced; out=%s", out)

	fi, err := os.Lstat(target)
	require.NoError(t, err, "real directory should still exist")
	assert.True(t, fi.IsDir(), "target should still be a directory")
	assert.Zero(t, fi.Mode()&os.ModeSymlink, "target should not be a symlink")

	data, err := os.ReadFile(userFile)
	require.NoError(t, err, "user file inside dir should still exist")
	assert.Equal(t, "user-data\n", string(data))
}

// TestUninstall_NoRepo documents the actual behavior: uninstall does NOT touch
// the managed repo (it operates from state.json), so a missing repo with an
// empty state surfaces as "package not installed", not ErrRepoNotInitialized.
func TestUninstall_NoRepo(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	setIsolatedHome(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	_, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"uninstall", "anything",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not installed",
		"expected 'not installed' (uninstall reads state, not repo); got: %v", err)
}
