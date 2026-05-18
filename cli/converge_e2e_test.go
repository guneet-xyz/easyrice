//go:build !windows

package main

// E2E converge / profile-switch / repair scenarios driving rootCmd in-process.
// All tests run on an isolated $HOME and an explicit --state file; never touch
// the real home dir or default state path.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Idempotent double install, basic profile switch, and the repair recreate
// case have been migrated to YAML scenarios under
// cli/testdata/scenarios/converge-* (see cli/converge_scenarios_test.go).
// The tests below cover behaviors that the scenario runner cannot easily
// express: last-wins overlays, conflict detection, mixed success/failure
// across packages, and prompt-bypass with --yes.

// Profile switch with overlapping source files (last-wins) — overlap is preserved
// pointing to the new winning source, and obsolete links are dropped.
func TestE2E_Converge_ProfileSwitch_PreservesOverlap(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, statePath, homeDir := setupE2ERepo(t)

	manifest := `schema_version = 1

[packages.demo]
supported_os = ["linux", "darwin"]

[packages.demo.profiles.common]
sources = [{path = "common", mode = "file", target = "$HOME/.config/demo"}]

[packages.demo.profiles.workmac]
sources = [
  {path = "work", mode = "file", target = "$HOME/.config/demo"},
]
`
	writeManifest(t, repoRoot, manifest)
	writeSourceFile(t, repoRoot, "demo/common/shared.conf", "common\n")
	writeSourceFile(t, repoRoot, "demo/common/common-only.conf", "c\n")
	writeSourceFile(t, repoRoot, "demo/work/shared.conf", "work\n")
	writeSourceFile(t, repoRoot, "demo/work/work-only.conf", "w\n")

	out, err := runE2ECmd(t,
		"--state", statePath,
		"--yes",
		"install", "demo",
		"--profile", "common",
	)
	require.NoError(t, err, "install common: out=%s", out)

	shared := filepath.Join(homeDir, ".config", "demo", "shared.conf")
	commonOnly := filepath.Join(homeDir, ".config", "demo", "common-only.conf")
	workOnly := filepath.Join(homeDir, ".config", "demo", "work-only.conf")

	assertSymlinkPointsTo(t, shared, filepath.Join(repoRoot, "demo", "common", "shared.conf"))
	assertSymlinkPointsTo(t, commonOnly, filepath.Join(repoRoot, "demo", "common", "common-only.conf"))
	assertStateHasPackage(t, statePath, "demo", "common", 2)

	resetInstallFlags()
	out, err = runE2ECmd(t,
		"--state", statePath,
		"--yes",
		"install", "demo",
		"--profile", "workmac",
	)
	require.NoError(t, err, "install workmac: out=%s", out)
	assert.Contains(t, out, "Switched demo from profile common to workmac.")

	// shared.conf now points to the work copy (last-wins overlay).
	assertSymlinkPointsTo(t, shared, filepath.Join(repoRoot, "demo", "work", "shared.conf"))
	assertSymlinkPointsTo(t, workOnly, filepath.Join(repoRoot, "demo", "work", "work-only.conf"))
	assertNoSymlinkAt(t, commonOnly)
	assertStateHasPackage(t, statePath, "demo", "workmac", 2)
}

// Repair: recreate a manually-deleted link — migrated to YAML scenario
// converge-repair-broken-symlink.

// When a managed target was replaced by a real file, converge must error out
// (conflict) and preserve the user's file.
func TestE2E_Converge_Repair_ConflictWhenTargetReplacedByFile(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, statePath, homeDir := setupE2ERepo(t)

	manifest := `schema_version = 1

[packages.demo]
supported_os = ["linux", "darwin"]

[packages.demo.profiles.default]
sources = [{path = "default", mode = "file", target = "$HOME/.config/demo"}]
`
	writeManifest(t, repoRoot, manifest)
	writeSourceFile(t, repoRoot, "demo/default/file1", "src\n")

	out, err := runE2ECmd(t,
		"--state", statePath,
		"--yes",
		"install", "demo",
		"--profile", "default",
	)
	require.NoError(t, err, "first install: out=%s", out)

	link := filepath.Join(homeDir, ".config", "demo", "file1")
	replaceSymlinkWithFile(t, link, "user-edited\n")

	resetInstallFlags()
	out, err = runE2ECmd(t,
		"--state", statePath,
		// no --yes: conflict must be detected before any prompt
		"install", "demo",
		"--profile", "default",
	)
	require.Error(t, err, "expected conflict error; out=%s", out)

	// The user's real file must still be there, untouched.
	data, rerr := os.ReadFile(link)
	require.NoError(t, rerr)
	assert.Equal(t, "user-edited\n", string(data))

	// State still records the package; no destructive change happened.
	assertStateHasPackage(t, statePath, "demo", "default", 1)
}

// 6. `rice install` with no arg: pkgA succeeds, pkgB fails (missing source dir).
func TestE2E_Converge_All_MixedSuccessAndFailure(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, statePath, homeDir := setupE2ERepo(t)

	manifest := `schema_version = 1

[packages.pkgA]
supported_os = ["linux", "darwin"]
[packages.pkgA.profiles.common]
sources = [{path = "common", mode = "file", target = "$HOME"}]

[packages.pkgB]
supported_os = ["linux", "darwin"]
[packages.pkgB.profiles.common]
sources = [{path = "missing", mode = "file", target = "$HOME"}]
`
	writeManifest(t, repoRoot, manifest)
	writeSourceFile(t, repoRoot, "pkgA/common/a.conf", "a\n")
	// pkgB has no source dir on disk on purpose.

	out, err := runE2ECmd(t,
		"--state", statePath,
		"--yes",
		"install",
		"--profile", "common",
	)
	require.Error(t, err, "expected converge-all to fail because of pkgB; out=%s", out)
	assert.Contains(t, out, "Installed pkgA")
	assert.Contains(t, out, "pkgB")

	// pkgA succeeded: symlink + state.
	assertSymlinkPointsTo(t, filepath.Join(homeDir, "a.conf"),
		filepath.Join(repoRoot, "pkgA", "common", "a.conf"))
	assertStateHasPackage(t, statePath, "pkgA", "common", 1)

	// pkgB failed: must not be in state.
	assertStateMissingPackage(t, statePath, "pkgB")
}

// Second install produces a no-op status message — migrated to YAML scenario
// converge-noop.

// Profile switch with --yes: no prompt, links transition cleanly.
func TestE2E_Converge_ProfileSwitch_WithYes(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, statePath, homeDir := setupE2ERepo(t)

	manifest := `schema_version = 1

[packages.demo]
supported_os = ["linux", "darwin"]

[packages.demo.profiles.common]
sources = [{path = "common", mode = "file", target = "$HOME/.config/demo"}]

[packages.demo.profiles.work]
sources = [{path = "work", mode = "file", target = "$HOME/.config/demo"}]
`
	writeManifest(t, repoRoot, manifest)
	writeSourceFile(t, repoRoot, "demo/common/common.conf", "c\n")
	writeSourceFile(t, repoRoot, "demo/work/work.conf", "w\n")

	out, err := runE2ECmd(t,
		"--state", statePath,
		"--yes",
		"install", "demo",
		"--profile", "common",
	)
	require.NoError(t, err, "install common: out=%s", out)

	commonLink := filepath.Join(homeDir, ".config", "demo", "common.conf")
	workLink := filepath.Join(homeDir, ".config", "demo", "work.conf")
	assertSymlinkPointsTo(t, commonLink, filepath.Join(repoRoot, "demo", "common", "common.conf"))

	resetInstallFlags()
	out, err = runE2ECmd(t,
		"--state", statePath,
		"--yes",
		"install", "demo",
		"--profile", "work",
	)
	require.NoError(t, err, "switch to work: out=%s", out)
	assert.Contains(t, out, "Switched demo from profile common to work.")

	assertNoSymlinkAt(t, commonLink)
	assertSymlinkPointsTo(t, workLink, filepath.Join(repoRoot, "demo", "work", "work.conf"))
	assertStateHasPackage(t, statePath, "demo", "work", 1)
}
