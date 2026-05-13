//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/state"
)

const demoManifest = `schema_version = 1

[packages.demo]
description = "Demo package"
supported_os = ["linux", "darwin"]

[packages.demo.profiles.default]
sources = [{path = "src", mode = "file", target = "$HOME/.config/demo"}]
`

func writeDemoPkg(t *testing.T, repoRoot string) {
	t.Helper()
	writeManifest(t, repoRoot, demoManifest)
	writeSourceFile(t, repoRoot, "demo/src/file1", "content1\n")
	writeSourceFile(t, repoRoot, "demo/src/file2", "content2\n")
}

func TestE2E_Uninstall_HappyPath(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeDemoPkg(t, repoRoot)

	out, err := runE2ECmd(t, "install", "demo", "--profile", "default", "--state", statePath, "--yes")
	require.NoError(t, err, "install: %s", out)

	resetInstallFlags()
	out, err = runE2ECmd(t, "uninstall", "demo", "--state", statePath, "--yes")
	require.NoError(t, err, "uninstall: %s", out)

	assertNoSymlinkAt(t, filepath.Join(homeDir, ".config", "demo", "file1"))
	assertNoSymlinkAt(t, filepath.Join(homeDir, ".config", "demo", "file2"))
	assertStateMissingPackage(t, statePath, "demo")
}

func TestE2E_Uninstall_ManuallyDeletedSymlink_Skipped(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeDemoPkg(t, repoRoot)

	out, err := runE2ECmd(t, "install", "demo", "--profile", "default", "--state", statePath, "--yes")
	require.NoError(t, err, "install: %s", out)

	link1 := filepath.Join(homeDir, ".config", "demo", "file1")
	link2 := filepath.Join(homeDir, ".config", "demo", "file2")

	manuallyDeleteSymlink(t, link1)

	resetInstallFlags()
	out, err = runE2ECmd(t, "uninstall", "demo", "--state", statePath, "--yes")
	require.NoError(t, err, "uninstall: %s", out)

	assertNoSymlinkAt(t, link1)
	assertNoSymlinkAt(t, link2)
	assertStateMissingPackage(t, statePath, "demo")
}

func TestE2E_Uninstall_SymlinkReplacedByRealFile(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeDemoPkg(t, repoRoot)

	out, err := runE2ECmd(t, "install", "demo", "--profile", "default", "--state", statePath, "--yes")
	require.NoError(t, err, "install: %s", out)

	link1 := filepath.Join(homeDir, ".config", "demo", "file1")
	link2 := filepath.Join(homeDir, ".config", "demo", "file2")

	replaceSymlinkWithFile(t, link1, "user-data")

	resetInstallFlags()
	out, err = runE2ECmd(t, "uninstall", "demo", "--state", statePath, "--yes")
	require.NoError(t, err, "uninstall: %s", out)

	fi, err := os.Lstat(link1)
	require.NoError(t, err, "real file should still exist at %s", link1)
	assert.Equal(t, os.FileMode(0), fi.Mode()&os.ModeSymlink, "should not be a symlink")
	contents, err := os.ReadFile(link1)
	require.NoError(t, err)
	assert.Equal(t, "user-data", string(contents))

	assertNoSymlinkAt(t, link2)
	assertStateMissingPackage(t, statePath, "demo")
}

func TestE2E_Uninstall_SymlinkReplacedByDifferentSymlink(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeDemoPkg(t, repoRoot)

	out, err := runE2ECmd(t, "install", "demo", "--profile", "default", "--state", statePath, "--yes")
	require.NoError(t, err, "install: %s", out)

	link1 := filepath.Join(homeDir, ".config", "demo", "file1")
	link2 := filepath.Join(homeDir, ".config", "demo", "file2")

	// Create a foreign target outside the managed repo
	foreignDir := t.TempDir()
	foreignFile := filepath.Join(foreignDir, "foreign.txt")
	require.NoError(t, os.WriteFile(foreignFile, []byte("foreign"), 0o644))

	replaceSymlinkWithSymlink(t, link1, foreignFile)

	resetInstallFlags()
	out, err = runE2ECmd(t, "uninstall", "demo", "--state", statePath, "--yes")
	require.NoError(t, err, "uninstall: %s", out)

	dest, err := os.Readlink(link1)
	require.NoError(t, err, "foreign symlink should still exist")
	assert.Equal(t, foreignFile, dest)

	assertNoSymlinkAt(t, link2)
	assertStateMissingPackage(t, statePath, "demo")
}

func TestE2E_Uninstall_SymlinkReplacedByDirectory(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeDemoPkg(t, repoRoot)

	out, err := runE2ECmd(t, "install", "demo", "--profile", "default", "--state", statePath, "--yes")
	require.NoError(t, err, "install: %s", out)

	link1 := filepath.Join(homeDir, ".config", "demo", "file1")
	link2 := filepath.Join(homeDir, ".config", "demo", "file2")

	replaceSymlinkWithDir(t, link1)

	resetInstallFlags()
	out, err = runE2ECmd(t, "uninstall", "demo", "--state", statePath, "--yes")
	require.NoError(t, err, "uninstall: %s", out)

	fi, err := os.Lstat(link1)
	require.NoError(t, err)
	assert.True(t, fi.IsDir(), "should still be a directory")

	assertNoSymlinkAt(t, link2)
	assertStateMissingPackage(t, statePath, "demo")
}

func TestE2E_Uninstall_FolderMode_ReplacedByDirectory(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeManifest(t, repoRoot, `schema_version = 1

[packages.folderpkg]
description = "Folder mode test"
supported_os = ["linux", "darwin"]

[packages.folderpkg.profiles.default]
sources = [{path = "cfg", mode = "folder", target = "$HOME/.config/folderpkg"}]
`)
	writeSourceFile(t, repoRoot, "folderpkg/cfg/init.conf", "k=v\n")

	out, err := runE2ECmd(t, "install", "folderpkg", "--profile", "default", "--state", statePath, "--yes")
	require.NoError(t, err, "install: %s", out)

	targetDir := filepath.Join(homeDir, ".config", "folderpkg")

	require.NoError(t, os.Remove(targetDir))
	require.NoError(t, os.MkdirAll(targetDir, 0o755))
	innerFile := filepath.Join(targetDir, "user-file.conf")
	require.NoError(t, os.WriteFile(innerFile, []byte("user content"), 0o644))

	resetInstallFlags()
	out, err = runE2ECmd(t, "uninstall", "folderpkg", "--state", statePath, "--yes")
	require.NoError(t, err, "uninstall: %s", out)

	fi, err := os.Lstat(targetDir)
	require.NoError(t, err)
	assert.True(t, fi.IsDir(), "directory should still exist")
	contents, err := os.ReadFile(innerFile)
	require.NoError(t, err)
	assert.Equal(t, "user content", string(contents))

	assertStateMissingPackage(t, statePath, "folderpkg")
}

func TestE2E_Uninstall_PackageNotInState_Error(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, _ := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeDemoPkg(t, repoRoot)

	out, err := runE2ECmd(t, "uninstall", "demo", "--state", statePath, "--yes")
	require.Error(t, err, "expected error; out=%s", out)
	combined := out + err.Error()
	assert.Contains(t, combined, "not installed")
}

func TestE2E_Uninstall_StateFileMissing_Error(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, _ := setupE2ERepo(t)
	writeDemoPkg(t, repoRoot)

	missing := filepath.Join(t.TempDir(), "nonexistent", "state.json")

	out, err := runE2ECmd(t, "uninstall", "demo", "--state", missing, "--yes")
	require.Error(t, err, "expected error; out=%s", out)
	combined := out + err.Error()
	assert.NotEmpty(t, combined, "expected a non-empty error message")
}

func TestE2E_Uninstall_StaleStateEntry(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeDemoPkg(t, repoRoot)

	staleLinks := []state.InstalledLink{
		{
			Source: filepath.Join(repoRoot, "demo", "src", "ghost1"),
			Target: filepath.Join(homeDir, ".config", "demo", "ghost1"),
		},
		{
			Source: filepath.Join(repoRoot, "demo", "src", "ghost2"),
			Target: filepath.Join(homeDir, ".config", "demo", "ghost2"),
		},
	}
	writeStaleState(t, statePath, "demo", "default", staleLinks)

	out, err := runE2ECmd(t, "uninstall", "demo", "--state", statePath, "--yes")
	require.NoError(t, err, "uninstall: %s", out)

	assertStateMissingPackage(t, statePath, "demo")
}

func TestE2E_Uninstall_PreservesOtherPackages(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeManifest(t, repoRoot, `schema_version = 1

[packages.pkgA]
description = "Package A"
supported_os = ["linux", "darwin"]

[packages.pkgA.profiles.default]
sources = [{path = "src", mode = "file", target = "$HOME/.config/pkgA"}]

[packages.pkgB]
description = "Package B"
supported_os = ["linux", "darwin"]

[packages.pkgB.profiles.default]
sources = [{path = "src", mode = "file", target = "$HOME/.config/pkgB"}]
`)
	writeSourceFile(t, repoRoot, "pkgA/src/aconf", "a\n")
	writeSourceFile(t, repoRoot, "pkgB/src/bconf", "b\n")

	out, err := runE2ECmd(t, "install", "pkgA", "--profile", "default", "--state", statePath, "--yes")
	require.NoError(t, err, "install pkgA: %s", out)

	resetInstallFlags()
	out, err = runE2ECmd(t, "install", "pkgB", "--profile", "default", "--state", statePath, "--yes")
	require.NoError(t, err, "install pkgB: %s", out)

	linkA := filepath.Join(homeDir, ".config", "pkgA", "aconf")
	linkB := filepath.Join(homeDir, ".config", "pkgB", "bconf")

	_, err = os.Lstat(linkA)
	require.NoError(t, err)
	_, err = os.Lstat(linkB)
	require.NoError(t, err)

	resetInstallFlags()
	out, err = runE2ECmd(t, "uninstall", "pkgA", "--state", statePath, "--yes")
	require.NoError(t, err, "uninstall pkgA: %s", out)

	assertNoSymlinkAt(t, linkA)
	assertStateMissingPackage(t, statePath, "pkgA")

	_, err = os.Lstat(linkB)
	require.NoError(t, err, "pkgB symlink should still exist")
	assertStateHasPackage(t, statePath, "pkgB", "default", 1)
}
