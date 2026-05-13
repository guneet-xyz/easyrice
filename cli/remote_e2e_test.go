//go:build !windows

package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requireGit skips the test if `git` is not on PATH.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
}

// resetRemoteE2EFlags resets flags shared by install + remote subcommands so each
// in-process invocation gets a clean slate. Always called before AND deferred.
func resetRemoteE2EFlags(t *testing.T) {
	t.Helper()
	resetInstallFlags()
	resetRemoteFlags()
}

// writeManagedManifestAndInit writes rice.toml at repoRoot, then initializes the
// managed repo as a real git repo (a prerequisite for adding submodules).
func writeManagedManifestAndInit(t *testing.T, repoRoot, manifestBody string) {
	t.Helper()
	writeManifest(t, repoRoot, manifestBody)
	initGitRepo(t, repoRoot)
}

// rewriteManagedManifest replaces rice.toml in the managed repo and commits the
// change (so the working tree stays clean for subsequent remote operations).
func rewriteManagedManifest(t *testing.T, repoRoot, manifestBody string) {
	t.Helper()
	writeManifest(t, repoRoot, manifestBody)
	cmd := exec.Command("git", "add", "rice.toml")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git add rice.toml: %s", string(out))
	cmd = exec.Command("git", "commit", "-m", "update rice.toml")
	cmd.Dir = repoRoot
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, "git commit: %s", string(out))
}

// baseUpstreamFiles returns the canonical files for the "base.common" upstream
// rice used by most remote-import e2e tests.
func baseUpstreamFiles() map[string]string {
	return map[string]string{
		"rice.toml": `schema_version = 1

[packages.base]
description = "base"
supported_os = ["linux", "darwin"]

[packages.base.profiles.common]
sources = [{path = "common", mode = "file", target = "$HOME/.config/base"}]
`,
		"base/common/basefile": "base-content\n",
	}
}

// TestE2E_Remote_ImportResolvesAndInstalls verifies an import-only profile (no
// local sources) resolves through a submoduled remote and installs the
// imported file into $HOME.
func TestE2E_Remote_ImportResolvesAndInstalls(t *testing.T) {
	requireGit(t)
	resetRemoteE2EFlags(t)
	t.Cleanup(func() { resetRemoteE2EFlags(t) })

	repoRoot, statePath, homeDir := setupE2ERepo(t)

	managedManifest := `schema_version = 1

[packages.demo]
description = "demo importing base.common"
supported_os = ["linux", "darwin"]

[packages.demo.profiles.default]
import = "remotes/upstream#base.common"
`
	writeManagedManifestAndInit(t, repoRoot, managedManifest)
	setupRemoteSubmodule(t, repoRoot, "upstream", baseUpstreamFiles())

	out, err := runE2ECmd(t,
		"--state", statePath,
		"--yes",
		"install", "demo",
		"--profile", "default",
		"--skip-deps",
	)
	require.NoError(t, err, "out=%s", out)

	link := filepath.Join(homeDir, ".config", "base", "basefile")
	expectedSrc := filepath.Join(repoRoot, "remotes", "upstream", "base", "common", "basefile")
	assertSymlinkPointsTo(t, link, expectedSrc)

	assertStateHasPackage(t, statePath, "demo", "default", 1)
}

// TestE2E_Remote_ImportThenOverlayLocalLastWins verifies that when a profile
// has BOTH an import and local file-mode sources, the local source overlays
// the imported source on a colliding target (last-wins).
func TestE2E_Remote_ImportThenOverlayLocalLastWins(t *testing.T) {
	requireGit(t)
	resetRemoteE2EFlags(t)
	t.Cleanup(func() { resetRemoteE2EFlags(t) })

	repoRoot, statePath, homeDir := setupE2ERepo(t)

	managedManifest := `schema_version = 1

[packages.demo]
description = "demo importing base.common with local overlay"
supported_os = ["linux", "darwin"]

[packages.demo.profiles.default]
import = "remotes/upstream#base.common"
sources = [{path = "local", mode = "file", target = "$HOME/.config/base"}]
`
	writeManagedManifestAndInit(t, repoRoot, managedManifest)
	setupRemoteSubmodule(t, repoRoot, "upstream", baseUpstreamFiles())

	// Local source overlays the same relative file ("basefile") - last-wins
	// rules should pick this one.
	writeSourceFile(t, repoRoot, "demo/local/basefile", "local-override\n")

	out, err := runE2ECmd(t,
		"--state", statePath,
		"--yes",
		"install", "demo",
		"--profile", "default",
		"--skip-deps",
	)
	require.NoError(t, err, "out=%s", out)

	link := filepath.Join(homeDir, ".config", "base", "basefile")
	localSrc := filepath.Join(repoRoot, "demo", "local", "basefile")
	assertSymlinkPointsTo(t, link, localSrc)

	assertStateHasPackage(t, statePath, "demo", "default", 1)
}

// TestE2E_Remote_MissingSubmodule_Error verifies that importing from a remote
// that was never added as a submodule produces an error and no state mutation.
func TestE2E_Remote_MissingSubmodule_Error(t *testing.T) {
	requireGit(t)
	resetRemoteE2EFlags(t)
	t.Cleanup(func() { resetRemoteE2EFlags(t) })

	repoRoot, statePath, _ := setupE2ERepo(t)

	managedManifest := `schema_version = 1

[packages.demo]
description = "demo importing from missing remote"
supported_os = ["linux", "darwin"]

[packages.demo.profiles.default]
import = "remotes/missing#base.common"
`
	writeManagedManifestAndInit(t, repoRoot, managedManifest)

	out, err := runE2ECmd(t,
		"--state", statePath,
		"--yes",
		"install", "demo",
		"--profile", "default",
		"--skip-deps",
	)
	require.Error(t, err, "out=%s", out)
	combined := strings.ToLower(out + " " + err.Error())
	assert.True(t,
		strings.Contains(combined, "missing") ||
			strings.Contains(combined, "not initialized") ||
			strings.Contains(combined, "not found"),
		"expected missing-remote error message, got: %s", combined)

	assertStateMissingPackage(t, statePath, "demo")
}

// TestE2E_Remote_ImportCycle_Error verifies cross-remote import cycles are
// detected and surface a "cycle"/"circular" error without mutating state.
func TestE2E_Remote_ImportCycle_Error(t *testing.T) {
	requireGit(t)
	resetRemoteE2EFlags(t)
	t.Cleanup(func() { resetRemoteE2EFlags(t) })

	repoRoot, statePath, _ := setupE2ERepo(t)

	managedManifest := `schema_version = 1

[packages.demo]
description = "demo importing cyclic remote"
supported_os = ["linux", "darwin"]

[packages.demo.profiles.default]
import = "remotes/repoA#pkgA.default"
`
	writeManagedManifestAndInit(t, repoRoot, managedManifest)

	// repoA's pkgA.default imports repoB#pkgB.default
	repoAFiles := map[string]string{
		"rice.toml": `schema_version = 1

[packages.pkgA]
description = "pkgA"
supported_os = ["linux", "darwin"]

[packages.pkgA.profiles.default]
import = "remotes/repoB#pkgB.default"
`,
	}
	// repoB's pkgB.default imports repoA#pkgA.default - completing the cycle
	repoBFiles := map[string]string{
		"rice.toml": `schema_version = 1

[packages.pkgB]
description = "pkgB"
supported_os = ["linux", "darwin"]

[packages.pkgB.profiles.default]
import = "remotes/repoA#pkgA.default"
`,
	}
	setupRemoteSubmodule(t, repoRoot, "repoA", repoAFiles)
	setupRemoteSubmodule(t, repoRoot, "repoB", repoBFiles)

	out, err := runE2ECmd(t,
		"--state", statePath,
		"--yes",
		"install", "demo",
		"--profile", "default",
		"--skip-deps",
	)
	require.Error(t, err, "out=%s", out)
	combined := strings.ToLower(out + " " + err.Error())
	assert.True(t,
		strings.Contains(combined, "cycle") ||
			strings.Contains(combined, "circular"),
		"expected cycle error message, got: %s", combined)

	assertStateMissingPackage(t, statePath, "demo")
}

// TestE2E_Remote_RemoveInUse_Refused verifies `rice remote remove <name>`
// refuses when any profile in rice.toml still imports from that remote, and
// leaves the submodule directory intact.
func TestE2E_Remote_RemoveInUse_Refused(t *testing.T) {
	requireGit(t)
	resetRemoteE2EFlags(t)
	t.Cleanup(func() { resetRemoteE2EFlags(t) })

	repoRoot, statePath, _ := setupE2ERepo(t)

	// First commit a minimal manifest so the repo is clean for `submodule add`.
	bareManifest := `schema_version = 1
`
	writeManagedManifestAndInit(t, repoRoot, bareManifest)
	setupRemoteSubmodule(t, repoRoot, "base", baseUpstreamFiles())

	// Now rewrite rice.toml to declare an import that references "base",
	// and commit the change so the working tree is clean.
	managedManifest := `schema_version = 1

[packages.demo]
description = "demo using base"
supported_os = ["linux", "darwin"]

[packages.demo.profiles.default]
import = "remotes/base#base.common"
`
	rewriteManagedManifest(t, repoRoot, managedManifest)

	submodulePath := filepath.Join(repoRoot, "remotes", "base")

	out, err := runE2ECmd(t,
		"--state", statePath,
		"--yes",
		"remote", "remove", "base",
	)
	require.Error(t, err, "out=%s", out)
	combined := strings.ToLower(out + " " + err.Error())
	assert.True(t,
		strings.Contains(combined, "in use") ||
			strings.Contains(combined, "referenced by an import") ||
			strings.Contains(combined, "import"),
		"expected ErrRemoteInUse message, got: %s", combined)

	// Submodule directory must still exist.
	_, statErr := exec.Command("test", "-d", submodulePath).CombinedOutput()
	require.NoError(t, statErr, "submodule directory should still exist at %s", submodulePath)
}
