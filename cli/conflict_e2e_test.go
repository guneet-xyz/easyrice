//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// demoManifestFileMode is a minimal file-mode manifest with a single source file.
const demoManifestFileMode = `schema_version = 1

[packages.demo]
supported_os = ["linux", "darwin"]

[packages.demo.profiles.default]
sources = [{path = "src", mode = "file", target = "$HOME/.config/demo"}]
`

// demoManifestFolderMode is a folder-mode manifest with one source dir.
const demoManifestFolderMode = `schema_version = 1

[packages.demo]
supported_os = ["linux", "darwin"]

[packages.demo.profiles.default]
sources = [{path = "src", mode = "folder", target = "$HOME/.config/demo"}]
`

// writeDemoFileMode wires up a file-mode demo package with one source file.
func writeDemoFileMode(t *testing.T, repoRoot string) {
	t.Helper()
	writeManifest(t, repoRoot, demoManifestFileMode)
	writeSourceFile(t, repoRoot, "demo/src/dotfile", "source-content\n")
}

// writeDemoFolderMode wires up a folder-mode demo package with one source dir.
func writeDemoFolderMode(t *testing.T, repoRoot string) {
	t.Helper()
	writeManifest(t, repoRoot, demoManifestFolderMode)
	writeSourceFile(t, repoRoot, "demo/src/dotfile", "source-content\n")
}

// TestE2E_Conflict_PreExistingRealFile_AbortsWithoutYes: a real file at the
// planned target causes install to abort; the file's contents are preserved
// and state remains untouched.
func TestE2E_Conflict_PreExistingRealFile_AbortsWithoutYes(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeDemoFileMode(t, repoRoot)

	tgt := filepath.Join(homeDir, ".config", "demo", "dotfile")
	require.NoError(t, os.MkdirAll(filepath.Dir(tgt), 0o755))
	require.NoError(t, os.WriteFile(tgt, []byte("user-data"), 0o644))

	out, err := runE2ECmd(t, "install", "demo", "--profile", "default", "--state", statePath)
	require.Error(t, err, "expected conflict error")
	assert.Contains(t, out, "CONFLICT")

	data, readErr := os.ReadFile(tgt)
	require.NoError(t, readErr)
	assert.Equal(t, "user-data", string(data))

	assertStateMissingPackage(t, statePath, "demo")
}

// TestE2E_Conflict_PreExistingRealFile_OverwritesWithYes: the production code
// does NOT overwrite real files even when `--yes` is supplied; pre-flight
// conflict detection still aborts. The pre-existing file is preserved.
func TestE2E_Conflict_PreExistingRealFile_OverwritesWithYes(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeDemoFileMode(t, repoRoot)

	tgt := filepath.Join(homeDir, ".config", "demo", "dotfile")
	require.NoError(t, os.MkdirAll(filepath.Dir(tgt), 0o755))
	require.NoError(t, os.WriteFile(tgt, []byte("user-data"), 0o644))

	out, err := runE2ECmd(t, "install", "demo", "--profile", "default", "--state", statePath, "--yes")
	require.Error(t, err, "conflicts must abort install even with --yes (no overwrite path exists)")
	assert.Contains(t, out, "CONFLICT")

	data, readErr := os.ReadFile(tgt)
	require.NoError(t, readErr)
	assert.Equal(t, "user-data", string(data))

	assertStateMissingPackage(t, statePath, "demo")
}

// TestE2E_Conflict_PreExistingSymlinkElsewhere_AbortsWithoutYes: a symlink at
// the target that points to a foreign path causes install to abort and the
// foreign symlink is left intact.
func TestE2E_Conflict_PreExistingSymlinkElsewhere_AbortsWithoutYes(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeDemoFileMode(t, repoRoot)

	foreign := filepath.Join(t.TempDir(), "foreign-target")
	require.NoError(t, os.WriteFile(foreign, []byte("foreign"), 0o644))

	tgt := filepath.Join(homeDir, ".config", "demo", "dotfile")
	require.NoError(t, os.MkdirAll(filepath.Dir(tgt), 0o755))
	require.NoError(t, os.Symlink(foreign, tgt))

	out, err := runE2ECmd(t, "install", "demo", "--profile", "default", "--state", statePath)
	require.Error(t, err)
	assert.Contains(t, out, "CONFLICT")

	dest, readErr := os.Readlink(tgt)
	require.NoError(t, readErr)
	assert.Equal(t, foreign, dest, "foreign symlink must remain intact")

	assertStateMissingPackage(t, statePath, "demo")
}

// TestE2E_Conflict_PreExistingDirectory_FileMode: a real directory at a
// file-mode planned target aborts install and the directory is left intact.
func TestE2E_Conflict_PreExistingDirectory_FileMode(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeDemoFileMode(t, repoRoot)

	tgt := filepath.Join(homeDir, ".config", "demo", "dotfile")
	require.NoError(t, os.MkdirAll(tgt, 0o755))

	out, err := runE2ECmd(t, "install", "demo", "--profile", "default", "--state", statePath)
	require.Error(t, err)
	assert.Contains(t, out, "CONFLICT")

	fi, statErr := os.Stat(tgt)
	require.NoError(t, statErr)
	assert.True(t, fi.IsDir(), "directory must still exist")

	assertStateMissingPackage(t, statePath, "demo")
}

// TestE2E_Conflict_PreExistingDirectory_FolderMode: a real directory at a
// folder-mode planned target aborts install and the directory is unchanged.
func TestE2E_Conflict_PreExistingDirectory_FolderMode(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeDemoFolderMode(t, repoRoot)

	tgt := filepath.Join(homeDir, ".config", "demo")
	require.NoError(t, os.MkdirAll(tgt, 0o755))
	preserved := filepath.Join(tgt, "user-file")
	require.NoError(t, os.WriteFile(preserved, []byte("user"), 0o644))

	out, err := runE2ECmd(t, "install", "demo", "--profile", "default", "--state", statePath)
	require.Error(t, err)
	assert.Contains(t, out, "CONFLICT")

	fi, statErr := os.Stat(tgt)
	require.NoError(t, statErr)
	assert.True(t, fi.IsDir(), "directory must still exist")

	data, readErr := os.ReadFile(preserved)
	require.NoError(t, readErr)
	assert.Equal(t, "user", string(data))

	assertStateMissingPackage(t, statePath, "demo")
}

// TestE2E_Conflict_DanglingSymlink_AbortsWithoutYes: a dangling symlink at the
// target aborts install and is left in place.
func TestE2E_Conflict_DanglingSymlink_AbortsWithoutYes(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeDemoFileMode(t, repoRoot)

	bogus := filepath.Join(t.TempDir(), "does-not-exist")
	tgt := filepath.Join(homeDir, ".config", "demo", "dotfile")
	require.NoError(t, os.MkdirAll(filepath.Dir(tgt), 0o755))
	require.NoError(t, os.Symlink(bogus, tgt))

	out, err := runE2ECmd(t, "install", "demo", "--profile", "default", "--state", statePath)
	require.Error(t, err)
	assert.Contains(t, out, "CONFLICT")

	_, lstatErr := os.Lstat(tgt)
	require.NoError(t, lstatErr, "dangling symlink should still exist")
	_, statErr := os.Stat(tgt)
	require.Error(t, statErr, "symlink must remain dangling")

	assertStateMissingPackage(t, statePath, "demo")
}

// TestE2E_Conflict_DanglingSymlink_ReplacesWithYes: even with `--yes`, the
// production code refuses to replace a dangling symlink; conflict detection
// runs before any prompt and aborts the install.
func TestE2E_Conflict_DanglingSymlink_ReplacesWithYes(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeDemoFileMode(t, repoRoot)

	bogus := filepath.Join(t.TempDir(), "does-not-exist")
	tgt := filepath.Join(homeDir, ".config", "demo", "dotfile")
	require.NoError(t, os.MkdirAll(filepath.Dir(tgt), 0o755))
	require.NoError(t, os.Symlink(bogus, tgt))

	out, err := runE2ECmd(t, "install", "demo", "--profile", "default", "--state", statePath, "--yes")
	require.Error(t, err, "conflicts abort install even with --yes")
	assert.Contains(t, out, "CONFLICT")

	dest, readErr := os.Readlink(tgt)
	require.NoError(t, readErr)
	assert.Equal(t, bogus, dest, "dangling symlink must remain in place")

	assertStateMissingPackage(t, statePath, "demo")
}

// TestE2E_Conflict_MultipleConflicts_AllReported: when several planned links
// each conflict, every target path appears in the rendered output.
func TestE2E_Conflict_MultipleConflicts_AllReported(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeManifest(t, repoRoot, `schema_version = 1

[packages.demo]
supported_os = ["linux", "darwin"]

[packages.demo.profiles.default]
sources = [{path = "src", mode = "file", target = "$HOME/.config/demo"}]
`)
	for _, name := range []string{"a", "b", "c"} {
		writeSourceFile(t, repoRoot, filepath.Join("demo", "src", name), name+"\n")
	}

	var tgts []string
	for _, name := range []string{"a", "b", "c"} {
		tgt := filepath.Join(homeDir, ".config", "demo", name)
		require.NoError(t, os.MkdirAll(filepath.Dir(tgt), 0o755))
		require.NoError(t, os.WriteFile(tgt, []byte("user-"+name), 0o644))
		tgts = append(tgts, tgt)
	}

	out, err := runE2ECmd(t, "install", "demo", "--profile", "default", "--state", statePath)
	require.Error(t, err)
	for _, tgt := range tgts {
		assert.Contains(t, out, tgt, "every conflicting target must be reported")
	}

	assertStateMissingPackage(t, statePath, "demo")
}

// TestE2E_Conflict_FolderModeTargetHasFiles_AbortsWithoutYes: folder-mode
// install refuses to clobber an existing directory; user files inside the
// directory are preserved.
func TestE2E_Conflict_FolderModeTargetHasFiles_AbortsWithoutYes(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeDemoFolderMode(t, repoRoot)

	tgt := filepath.Join(homeDir, ".config", "demo")
	require.NoError(t, os.MkdirAll(tgt, 0o755))
	inner := filepath.Join(tgt, "user-config")
	require.NoError(t, os.WriteFile(inner, []byte("important"), 0o644))

	out, err := runE2ECmd(t, "install", "demo", "--profile", "default", "--state", statePath)
	require.Error(t, err)
	assert.Contains(t, out, "CONFLICT")

	data, readErr := os.ReadFile(inner)
	require.NoError(t, readErr)
	assert.Equal(t, "important", string(data), "user files inside dir preserved")

	assertStateMissingPackage(t, statePath, "demo")
}

// TestE2E_Conflict_TwoPackagesSameTarget: installing pkgA and then attempting
// to install pkgB which targets the same path aborts pkgB and leaves pkgA's
// symlink intact in state and on disk.
func TestE2E_Conflict_TwoPackagesSameTarget(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeManifest(t, repoRoot, `schema_version = 1

[packages.pkga]
supported_os = ["linux", "darwin"]
[packages.pkga.profiles.default]
sources = [{path = "src", mode = "file", target = "$HOME/.config/shared"}]

[packages.pkgb]
supported_os = ["linux", "darwin"]
[packages.pkgb.profiles.default]
sources = [{path = "src", mode = "file", target = "$HOME/.config/shared"}]
`)
	writeSourceFile(t, repoRoot, "pkga/src/file", "from-a\n")
	writeSourceFile(t, repoRoot, "pkgb/src/file", "from-b\n")

	outA, errA := runE2ECmd(t, "install", "pkga", "--profile", "default", "--state", statePath, "--yes")
	require.NoError(t, errA, "pkga install: %s", outA)

	tgt := filepath.Join(homeDir, ".config", "shared", "file")
	srcA, absErr := filepath.Abs(filepath.Join(repoRoot, "pkga", "src", "file"))
	require.NoError(t, absErr)
	assertSymlinkPointsTo(t, tgt, srcA)

	outB, errB := runE2ECmd(t, "install", "pkgb", "--profile", "default", "--state", statePath)
	require.Error(t, errB, "pkgb must conflict with pkga's symlink")
	assert.Contains(t, outB, "CONFLICT")

	// pkga's symlink still points where it did
	assertSymlinkPointsTo(t, tgt, srcA)

	assertStateHasPackage(t, statePath, "pkga", "default", 1)
	assertStateMissingPackage(t, statePath, "pkgb")
}

// TestE2E_Conflict_NoConflictWhenAlreadyOurs: installing the same package
// twice is idempotent — the second run reports no conflict.
func TestE2E_Conflict_NoConflictWhenAlreadyOurs(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, _, _ := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeDemoFileMode(t, repoRoot)

	out1, err1 := runE2ECmd(t, "install", "demo", "--profile", "default", "--state", statePath, "--yes")
	require.NoError(t, err1, "first install: %s", out1)
	assertStateHasPackage(t, statePath, "demo", "default", 1)

	out2, err2 := runE2ECmd(t, "install", "demo", "--profile", "default", "--state", statePath, "--yes")
	require.NoError(t, err2, "second install must be a no-op: %s", out2)
	assert.NotContains(t, out2, "CONFLICT")
	assert.NotContains(t, out2, "conflicts detected")

	assertStateHasPackage(t, statePath, "demo", "default", 1)
}

// TestE2E_Conflict_ReportContainsPathAndReason: the rendered conflict report
// contains both the offending target path and a human-readable reason.
func TestE2E_Conflict_ReportContainsPathAndReason(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeDemoFileMode(t, repoRoot)

	tgt := filepath.Join(homeDir, ".config", "demo", "dotfile")
	require.NoError(t, os.MkdirAll(filepath.Dir(tgt), 0o755))
	require.NoError(t, os.WriteFile(tgt, []byte("user-data"), 0o644))

	out, err := runE2ECmd(t, "install", "demo", "--profile", "default", "--state", statePath)
	require.Error(t, err)

	// prompt.RenderConflicts emits: "CONFLICT  <target>: <reason>"
	// and conflict.go uses "existing file" as the reason for a regular file.
	assert.Contains(t, out, "CONFLICT")
	assert.Contains(t, out, tgt, "rendered conflict must include the target path")
	assert.Contains(t, out, "existing file", "rendered conflict must include a reason")
}
