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

// TestE2E_Install_DeepNestedTarget_CreatesDirs verifies installer auto-creates
// arbitrarily deep parent directories on the target side.
func TestE2E_Install_DeepNestedTarget_CreatesDirs(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeManifest(t, repoRoot, `schema_version = 1
[packages.demo]
supported_os = ["linux", "darwin"]
[packages.demo.profiles.default]
sources = [{path = "src", mode = "file", target = "$HOME/.config/a/b/c/d"}]
`)
	writeSourceFile(t, repoRoot, "demo/src/file", "deep\n")

	out, err := runE2ECmd(t, "--state", statePath, "--yes", "install", "demo",
		"--profile", "default", "--skip-deps")
	require.NoError(t, err, "install failed: %s", out)

	src := filepath.Join(repoRoot, "demo", "src", "file")
	tgt := filepath.Join(homeDir, ".config", "a", "b", "c", "d", "file")
	assertSymlinkPointsTo(t, tgt, src)
	assertStateHasPackage(t, statePath, "demo", "default", 1)
}

// TestE2E_Install_HomeExpansion verifies $HOME in target is expanded via
// os.ExpandEnv (not treated as a literal).
func TestE2E_Install_HomeExpansion(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeManifest(t, repoRoot, `schema_version = 1
[packages.demo]
supported_os = ["linux", "darwin"]
[packages.demo.profiles.default]
sources = [{path = "src", mode = "file", target = "$HOME/.config/expand"}]
`)
	writeSourceFile(t, repoRoot, "demo/src/file", "expand\n")

	out, err := runE2ECmd(t, "--state", statePath, "--yes", "install", "demo",
		"--profile", "default", "--skip-deps")
	require.NoError(t, err, "install failed: %s", out)

	tgt := filepath.Join(homeDir, ".config", "expand", "file")
	src := filepath.Join(repoRoot, "demo", "src", "file")
	assertSymlinkPointsTo(t, tgt, src)
	// And of course the literal "$HOME/..." path must NOT exist.
	literal := filepath.Join(repoRoot, "$HOME", ".config", "expand", "file")
	assertNoSymlinkAt(t, literal)
}

// TestE2E_Install_OverlayLastWins verifies that for file-mode sources sharing a
// target subtree, later sources override earlier ones on identical relpaths.
func TestE2E_Install_OverlayLastWins(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeManifest(t, repoRoot, `schema_version = 1
[packages.demo]
supported_os = ["linux", "darwin"]
[packages.demo.profiles.default]
sources = [
  {path = "common", mode = "file", target = "$HOME/.config/demo"},
  {path = "work",   mode = "file", target = "$HOME/.config/demo"},
]
`)
	writeSourceFile(t, repoRoot, "demo/common/shared.conf", "common")
	writeSourceFile(t, repoRoot, "demo/work/shared.conf", "work")

	out, err := runE2ECmd(t, "--state", statePath, "--yes", "install", "demo",
		"--profile", "default", "--skip-deps")
	require.NoError(t, err, "install failed: %s", out)

	want := filepath.Join(repoRoot, "demo", "work", "shared.conf")
	tgt := filepath.Join(homeDir, ".config", "demo", "shared.conf")
	assertSymlinkPointsTo(t, tgt, want)
	assertStateHasPackage(t, statePath, "demo", "default", 1)
}

// TestE2E_Install_SourceWithSymlinkSkipped verifies that symlinks present
// inside the source tree are skipped during walk (we manage real files only).
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

// TestE2E_Install_EmptySourceDir_Success verifies that an empty source dir is
// a valid no-link install (no error, 0 links).
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

// TestE2E_Install_UnsupportedOS_Error verifies the OS-gate refuses a package
// declared as only-windows when running on linux/darwin.
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

// TestE2E_Install_PackageNotDeclared_Error verifies referencing an unknown
// package name yields a clear error.
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

// TestE2E_Install_ProfileNotDeclared_Error verifies a missing profile name
// produces an error.
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

// TestE2E_Install_SourceDirMissing_Error verifies a manifest pointing at a
// non-existent source path is rejected with a descriptive error.
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

// TestE2E_Install_TargetEscapesHome_Refused verifies the withinHome guard
// rejects absolute targets outside the isolated $HOME.
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

// TestE2E_Install_NoArgs_ConvergesAll verifies `rice install` with no positional
// argument converges every declared package.
func TestE2E_Install_NoArgs_ConvergesAll(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)
	repoRoot, _, homeDir := setupE2ERepo(t)
	statePath := filepath.Join(t.TempDir(), "state.json")

	writeManifest(t, repoRoot, `schema_version = 1
[packages.pkgA]
supported_os = ["linux", "darwin"]
[packages.pkgA.profiles.default]
sources = [{path = "common", mode = "file", target = "$HOME/.config/pkgA"}]

[packages.pkgB]
supported_os = ["linux", "darwin"]
[packages.pkgB.profiles.default]
sources = [{path = "common", mode = "file", target = "$HOME/.config/pkgB"}]
`)
	writeSourceFile(t, repoRoot, "pkgA/common/a.conf", "a\n")
	writeSourceFile(t, repoRoot, "pkgB/common/b.conf", "b\n")

	out, err := runE2ECmd(t, "--state", statePath, "--yes", "install",
		"--profile", "default", "--skip-deps")
	require.NoError(t, err, "install failed: %s", out)

	assertSymlinkPointsTo(t,
		filepath.Join(homeDir, ".config", "pkgA", "a.conf"),
		filepath.Join(repoRoot, "pkgA", "common", "a.conf"))
	assertSymlinkPointsTo(t,
		filepath.Join(homeDir, ".config", "pkgB", "b.conf"),
		filepath.Join(repoRoot, "pkgB", "common", "b.conf"))
	assertStateHasPackage(t, statePath, "pkgA", "default", 1)
	assertStateHasPackage(t, statePath, "pkgB", "default", 1)
}

// TestE2E_Install_MalformedManifest_Error verifies syntactically broken TOML
// produces an error before any work is done.
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

// TestE2E_Install_StateFileMissing_CreatedFresh verifies the installer creates
// the state file on first install when it doesn't yet exist.
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

// TestE2E_Install_StateFileCorrupted_Errors verifies install fails fast (and
// leaves the corrupted state file untouched) when state.json is unreadable.
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
