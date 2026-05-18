//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_Install_FreshSmoke verifies the happy path: a single package with a
// single source file installs cleanly and is recorded in state.
func TestE2E_Install_FreshSmoke(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeManifest(t, repoRoot, `schema_version = 1
[packages.demo]
supported_os = ["linux", "darwin"]
[packages.demo.profiles.default]
sources = [{path = "common", mode = "file", target = "$HOME/.config/demo"}]
`)
	writeSourceFile(t, repoRoot, "demo/common/dotfile", "content\n")

	out, err := runE2ECmd(t, "--state", statePath, "--yes", "install", "demo",
		"--profile", "default", "--skip-deps")
	require.NoError(t, err, "install failed: %s", out)

	src := filepath.Join(repoRoot, "demo", "common", "dotfile")
	tgt := filepath.Join(homeDir, ".config", "demo", "dotfile")
	assertSymlinkPointsTo(t, tgt, src)
	assertStateHasPackage(t, statePath, "demo", "default", 1)
}

// TestE2E_Install_DeepNestedTarget_CreatesDirs migrated to scenario
// "install_deep_nested_target" (see scenarios_migrated_test.go).

// TestE2E_Install_HomeExpansion migrated to scenario "install_home_expansion".

// TestE2E_Install_OverlayLastWins migrated to scenario "install_overlay_last_wins".

// INLINE: planting an os.Symlink inside the source tree pre-install has no
// equivalent scenario mutate op (replace_symlink requires a pre-existing link;
// create_symlink does not exist).
func TestE2E_Install_SourceWithSymlinkSkipped(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeManifest(t, repoRoot, `schema_version = 1
[packages.demo]
supported_os = ["linux", "darwin"]
[packages.demo.profiles.default]
sources = [{path = "common", mode = "file", target = "$HOME/.config/demo"}]
`)
	writeSourceFile(t, repoRoot, "demo/common/real.conf", "real\n")
	// Plant a symlink inside the source tree pointing somewhere irrelevant.
	require.NoError(t, os.Symlink("/tmp/something",
		filepath.Join(repoRoot, "demo", "common", "link.conf")))

	out, err := runE2ECmd(t, "--state", statePath, "--yes", "install", "demo",
		"--profile", "default", "--skip-deps")
	require.NoError(t, err, "install failed: %s", out)

	src := filepath.Join(repoRoot, "demo", "common", "real.conf")
	realTgt := filepath.Join(homeDir, ".config", "demo", "real.conf")
	linkTgt := filepath.Join(homeDir, ".config", "demo", "link.conf")
	assertSymlinkPointsTo(t, realTgt, src)
	assertNoSymlinkAt(t, linkTgt)
	assertStateHasPackage(t, statePath, "demo", "default", 1)
}

// INLINE: empty source directory cannot be checked into testdata seeds (git
// drops empty dirs); a single mkdir + zero-link assertion stays clearer here.
func TestE2E_Install_EmptySourceDir_Success(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, _ := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeManifest(t, repoRoot, `schema_version = 1
[packages.demo]
supported_os = ["linux", "darwin"]
[packages.demo.profiles.default]
sources = [{path = "common", mode = "file", target = "$HOME/.config/demo"}]
`)
	// Create the source dir but leave it empty.
	require.NoError(t, os.MkdirAll(filepath.Join(repoRoot, "demo", "common"), 0o755))

	out, err := runE2ECmd(t, "--state", statePath, "--yes", "install", "demo",
		"--profile", "default", "--skip-deps")
	require.NoError(t, err, "install failed: %s", out)

	assertStateHasPackage(t, statePath, "demo", "default", 0)
}

// INLINE: error-case test - asserts on error string ("does not support") and
// state-file absence; scenario format prefers snapshotable outcomes.
func TestE2E_Install_UnsupportedOS_Error(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, _ := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeManifest(t, repoRoot, `schema_version = 1
[packages.demo]
supported_os = ["windows"]
[packages.demo.profiles.default]
sources = [{path = "common", mode = "file", target = "$HOME/.config/demo"}]
`)
	writeSourceFile(t, repoRoot, "demo/common/file", "x\n")

	out, err := runE2ECmd(t, "--state", statePath, "--yes", "install", "demo",
		"--profile", "default", "--skip-deps")
	require.Error(t, err)
	combined := out + err.Error()
	assert.Contains(t, combined, "does not support")
	// State must not record the package.
	_, statErr := os.Stat(statePath)
	if statErr == nil {
		assertStateMissingPackage(t, statePath, "demo")
	}
}

// INLINE: error-case test - asserts on error string ("nonexistent" / "not declared").
func TestE2E_Install_PackageNotDeclared_Error(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, _ := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeManifest(t, repoRoot, `schema_version = 1
[packages.demo]
supported_os = ["linux", "darwin"]
[packages.demo.profiles.default]
sources = [{path = "common", mode = "file", target = "$HOME/.config/demo"}]
`)
	writeSourceFile(t, repoRoot, "demo/common/file", "x\n")

	out, err := runE2ECmd(t, "--state", statePath, "--yes", "install",
		"nonexistent", "--profile", "default", "--skip-deps")
	require.Error(t, err)
	combined := out + err.Error()
	assert.Contains(t, combined, "nonexistent")
	assert.Contains(t, combined, "not declared")
}

// INLINE: error-case test - asserts on error string ("badprofile").
func TestE2E_Install_ProfileNotDeclared_Error(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, _ := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeManifest(t, repoRoot, `schema_version = 1
[packages.demo]
supported_os = ["linux", "darwin"]
[packages.demo.profiles.default]
sources = [{path = "common", mode = "file", target = "$HOME/.config/demo"}]
`)
	writeSourceFile(t, repoRoot, "demo/common/file", "x\n")

	out, err := runE2ECmd(t, "--state", statePath, "--yes", "install", "demo",
		"--profile", "badprofile", "--skip-deps")
	require.Error(t, err)
	combined := out + err.Error()
	assert.Contains(t, combined, "badprofile")
}

// INLINE: error-case test - missing source dir, asserts on error string.
func TestE2E_Install_SourceDirMissing_Error(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, _ := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeManifest(t, repoRoot, `schema_version = 1
[packages.demo]
supported_os = ["linux", "darwin"]
[packages.demo.profiles.default]
sources = [{path = "missing", mode = "file", target = "$HOME/.config/demo"}]
`)
	// Intentionally do NOT create demo/missing/.

	out, err := runE2ECmd(t, "--state", statePath, "--yes", "install", "demo",
		"--profile", "default", "--skip-deps")
	require.Error(t, err)
	combined := out + err.Error()
	assert.Contains(t, combined, "missing")
}

// INLINE: deliberately targets a path OUTSIDE the home/repo sandbox to
// exercise withinHome; scenario containment guard would refuse the assertion.
func TestE2E_Install_TargetEscapesHome_Refused(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, _ := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	escapeDir := "/tmp/easyrice-outside-test"
	// Make sure no leftover dir exists from a previous run.
	_ = os.RemoveAll(escapeDir)

	writeManifest(t, repoRoot, `schema_version = 1
[packages.demo]
supported_os = ["linux", "darwin"]
[packages.demo.profiles.default]
sources = [{path = "common", mode = "file", target = "/tmp/easyrice-outside-test"}]
`)
	writeSourceFile(t, repoRoot, "demo/common/file", "x\n")

	out, err := runE2ECmd(t, "--state", statePath, "--yes", "install", "demo",
		"--profile", "default", "--skip-deps")
	require.Error(t, err)
	combined := out + err.Error()
	assert.Contains(t, combined, "outside")
	assertNoSymlinkAt(t, filepath.Join(escapeDir, "file"))
}

// TestE2E_Install_NoArgs_ConvergesAll migrated to scenario "install_no_args_converges_all".

// INLINE: error-case test - asserts on parse/invalid manifest error variants
// and state-file absence.
func TestE2E_Install_MalformedManifest_Error(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, _ := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	// Write structurally invalid TOML directly (writeManifest accepts a string).
	writeManifest(t, repoRoot, "schema_version = 1\n[packages.demo\nbroken")

	out, err := runE2ECmd(t, "--state", statePath, "--yes", "install", "demo",
		"--profile", "default", "--skip-deps")
	require.Error(t, err)
	combined := out + err.Error()
	// Cover the various parse/validation error phrasings.
	assert.True(t,
		containsAny(combined, "manifest", "parse", "invalid", "toml"),
		"expected parse/invalid manifest error, got: %s", combined)
	// State file must not have been created.
	_, statErr := os.Stat(statePath)
	assert.True(t, os.IsNotExist(statErr), "state file should not be created on manifest parse error")
}

// INLINE: --state points at a nested tempdir path outside the home sandbox to
// verify on-the-fly state-file creation; scenario expect.state captures from a
// single canonical path.
func TestE2E_Install_StateFileMissing_CreatedFresh(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, _ := setupE2ERepo(t)
	// Point --state at a path inside a fresh tempdir; do NOT create the file.
	statePath := filepath.Join(t.TempDir(), "nested", "state.json")

	writeManifest(t, repoRoot, `schema_version = 1
[packages.demo]
supported_os = ["linux", "darwin"]
[packages.demo.profiles.default]
sources = [{path = "common", mode = "file", target = "$HOME/.config/demo"}]
`)
	writeSourceFile(t, repoRoot, "demo/common/file", "x\n")

	out, err := runE2ECmd(t, "--state", statePath, "--yes", "install", "demo",
		"--profile", "default", "--skip-deps")
	require.NoError(t, err, "install failed: %s", out)

	_, statErr := os.Stat(statePath)
	require.NoError(t, statErr, "state file should be created on first install")
	assertStateHasPackage(t, statePath, "demo", "default", 1)
}

// INLINE: byte-equality assertion on the corrupted state file (untouched on
// failure) does not map cleanly to scenario state-snapshot diffing.
func TestE2E_Install_StateFileCorrupted_Errors(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, _ := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")
	corruptStateFile(t, statePath)

	writeManifest(t, repoRoot, `schema_version = 1
[packages.demo]
supported_os = ["linux", "darwin"]
[packages.demo.profiles.default]
sources = [{path = "common", mode = "file", target = "$HOME/.config/demo"}]
`)
	writeSourceFile(t, repoRoot, "demo/common/file", "x\n")

	before, readErr := os.ReadFile(statePath)
	require.NoError(t, readErr)

	out, err := runE2ECmd(t, "--state", statePath, "--yes", "install", "demo",
		"--profile", "default", "--skip-deps")
	require.Error(t, err, "out=%s", out)

	after, readErr := os.ReadFile(statePath)
	require.NoError(t, readErr)
	assert.Equal(t, before, after, "corrupted state file should be unchanged on failure")
}

// containsAny reports whether s contains any of the given substrings (case-insensitive).
func containsAny(s string, subs ...string) bool {
	low := toLowerASCII(s)
	for _, sub := range subs {
		if indexASCII(low, toLowerASCII(sub)) >= 0 {
			return true
		}
	}
	return false
}

// toLowerASCII lowercases only ASCII letters; sufficient for our error scanning.
func toLowerASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}

// indexASCII is a tiny strings.Index replacement to avoid importing strings.
func indexASCII(s, sub string) int {
	if len(sub) == 0 {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
