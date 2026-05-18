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

// The following uninstall E2E tests have been migrated to YAML scenarios under
// cli/testdata/scenarios/ (see scenarios_migrated_test.go for the inventory):
//   - TestE2E_Uninstall_HappyPath                       -> uninstall_happy
//   - TestE2E_Uninstall_ManuallyDeletedSymlink_Skipped  -> uninstall_manually_deleted
//   - TestE2E_Uninstall_SymlinkReplacedByRealFile       -> uninstall_replaced_by_file
//   - TestE2E_Uninstall_SymlinkReplacedByDirectory      -> uninstall_replaced_by_dir
//   - TestE2E_Uninstall_FolderMode_ReplacedByDirectory  -> uninstall_folder_mode_replaced
//   - TestE2E_Uninstall_PreservesOtherPackages          -> uninstall_preserves_others
//
// The tests below remain inline because they either inspect symlink targets
// directly (foreign-symlink case) or assert single-step error semantics that
// don't benefit from snapshot fixtures.

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
