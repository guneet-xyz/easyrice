package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/repo"
	"github.com/guneet-xyz/easyrice/internal/testhelpers/scenario"
)

func TestScenario_RemoteImportResolves(t *testing.T) {
	skipOnWindows(t)
	requireGit(t)
	resetRemoteE2EFlags(t)
	t.Cleanup(func() { resetRemoteE2EFlags(t) })

	srcDir, err := filepath.Abs(filepath.Join("testdata", "scenarios", "remote_import_resolves"))
	require.NoError(t, err)

	sb := setupScenarioSandbox(t)

	managedManifest := `schema_version = 1

[packages.demo]
description = "demo importing base.common"
supported_os = ["linux", "darwin"]

[packages.demo.profiles.default]
import = "remotes/upstream#base.common"
`
	writeManagedManifestAndInit(t, sb.RepoRoot, managedManifest)
	setupRemoteSubmodule(t, sb.RepoRoot, "upstream", baseUpstreamFiles())

	scenarioDir := renderScenario(t, srcDir, sb)
	scenario.Run(t, scenarioDir, newScenarioConfig())
}

func TestScenario_RemoteMissingSubmodule(t *testing.T) {
	skipOnWindows(t)
	requireGit(t)
	resetRemoteE2EFlags(t)
	t.Cleanup(func() { resetRemoteE2EFlags(t) })

	srcDir, err := filepath.Abs(filepath.Join("testdata", "scenarios", "remote_missing_submodule"))
	require.NoError(t, err)

	sb := setupScenarioSandbox(t)

	managedManifest := `schema_version = 1

[packages.demo]
description = "demo importing from missing remote"
supported_os = ["linux", "darwin"]

[packages.demo.profiles.default]
import = "remotes/missing#base.common"
`
	writeManagedManifestAndInit(t, sb.RepoRoot, managedManifest)

	scenarioDir := renderScenario(t, srcDir, sb)
	scenario.Run(t, scenarioDir, newScenarioConfig())
}

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

// TestE2E_Remote_ImportThenOverlayLocalLastWins verifies that when a profile
// has BOTH an import and local file-mode sources, the local source overlays
// the imported source on a colliding target (last-wins).
// INLINE: asserts file-system symlink target identity; richer than a
// stdout/state snapshot would capture.
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

// TestE2E_Remote_ImportCycle_Error verifies cross-remote import cycles are
// detected and surface a "cycle"/"circular" error without mutating state.
// INLINE: asserts error message tokens via OR semantics; not expressible as
// a simple stdout_contains list.
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
// INLINE: multi-step git+manifest setup with disjunctive error matching; no
// scenario migration warranted.
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

// TestScenario_RemoteAddInvalidName verifies `rice remote add` rejects
// --name values that don't match ^[a-zA-Z0-9_-]+$.
// INLINE: validates --name flag validation without needing a remote repo.
func TestScenario_RemoteAddInvalidName(t *testing.T) {
	skipOnWindows(t)
	requireGit(t)
	resetRemoteE2EFlags(t)
	t.Cleanup(func() { resetRemoteE2EFlags(t) })

	setIsolatedHome(t)
	root := makeManagedRepo(t, "")
	_ = root // suppress unused warning

	// Run remote add with an invalid name (contains space)
	out, err := runRemoteCmd(t, "remote", "add", "https://example.com/upstream.git", "--name", "bad name with spaces")
	require.Error(t, err)
	assert.Contains(t, out, "invalid")
}

// TestScenario_RemoteAddMissingRiceToml verifies `rice remote add` fails
// when the remote repo does not contain a rice.toml, and the submodule is
// rolled back (remotes/<name>/ does not exist after the error).
// INLINE: creates a bare upstream without rice.toml, asserts rollback.
func TestScenario_RemoteAddMissingRiceToml(t *testing.T) {
	skipOnWindows(t)
	requireGit(t)
	resetRemoteE2EFlags(t)
	t.Cleanup(func() { resetRemoteE2EFlags(t) })

	setIsolatedHome(t)
	root := makeManagedRepo(t, "")

	// Create a bare upstream WITHOUT rice.toml
	bareDir := t.TempDir()
	gitRunRemote(t, bareDir, "init", "--bare", "-b", "main", "upstream.git")

	// Create a working tree, add a dummy file (NOT rice.toml), push
	wt := t.TempDir()
	gitRunRemote(t, wt, "init", "-b", "main")
	gitRunRemote(t, wt, "config", "user.email", "test@test.com")
	gitRunRemote(t, wt, "config", "user.name", "Test")

	// Write a file that is NOT rice.toml
	require.NoError(t, os.WriteFile(filepath.Join(wt, "README.md"), []byte("hello"), 0o644))
	gitRunRemote(t, wt, "add", "--", "README.md")
	gitRunRemote(t, wt, "commit", "-m", "init without rice.toml")
	gitRunRemote(t, wt, "remote", "add", "origin", "file://"+filepath.Join(bareDir, "upstream.git"))
	gitRunRemote(t, wt, "push", "origin", "main")

	upstreamURL := "file://" + filepath.Join(bareDir, "upstream.git")

	// Run remote add - should fail because upstream has no rice.toml
	out, err := runRemoteCmd(t, "remote", "add", upstreamURL, "--name", "kick")
	require.Error(t, err)
	assert.Contains(t, out, "rice.toml")

	// Verify rollback: remotes/kick should NOT exist
	_, statErr := os.Stat(filepath.Join(root, "remotes", "kick"))
	assert.True(t, os.IsNotExist(statErr), "remotes/kick must not exist after rollback; got %v", statErr)
}

// TestScenario_RemoteListEmpty verifies `rice remote list` with no remotes
// shows "No remotes configured."
func TestScenario_RemoteListEmpty(t *testing.T) {
	skipOnWindows(t)
	requireGit(t)
	resetRemoteE2EFlags(t)
	t.Cleanup(func() { resetRemoteE2EFlags(t) })

	setIsolatedHome(t)
	makeManagedRepo(t, "")

	out, err := runRemoteCmd(t, "remote", "list")
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "No remotes configured.")
}

// TestScenario_RemoteListPopulated verifies `rice remote list` with 1 remote
// shows it in the output.
func TestScenario_RemoteListPopulated(t *testing.T) {
	skipOnWindows(t)
	requireGit(t)
	resetRemoteE2EFlags(t)
	t.Cleanup(func() { resetRemoteE2EFlags(t) })

	setIsolatedHome(t)
	makeManagedRepo(t, "")
	upstream := setupBareUpstream(t, fixtureAbs(t, "remote_rice"))

	_, err := runRemoteCmd(t, "remote", "add", upstream, "--name", "kick")
	require.NoError(t, err)

	out, err := runRemoteCmd(t, "remote", "list")
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "kick")
	assert.Contains(t, out, "remotes/kick")
}

// TestScenario_RemoteUpdateHappy verifies `rice remote update kick` succeeds
// when the remote exists.
func TestScenario_RemoteUpdateHappy(t *testing.T) {
	skipOnWindows(t)
	requireGit(t)
	resetRemoteE2EFlags(t)
	t.Cleanup(func() { resetRemoteE2EFlags(t) })

	setIsolatedHome(t)
	makeManagedRepo(t, "")
	upstream := setupBareUpstream(t, fixtureAbs(t, "remote_rice"))

	_, err := runRemoteCmd(t, "remote", "add", upstream, "--name", "kick")
	require.NoError(t, err)

	out, err := runRemoteCmd(t, "remote", "update", "kick")
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Updated remote")
	assert.Contains(t, out, "kick")
}

// TestScenario_RemoteUpdateNotFound verifies `rice remote update nonexistent`
// fails with ErrRemoteNotFound.
func TestScenario_RemoteUpdateNotFound(t *testing.T) {
	skipOnWindows(t)
	requireGit(t)
	resetRemoteE2EFlags(t)
	t.Cleanup(func() { resetRemoteE2EFlags(t) })

	setIsolatedHome(t)
	makeManagedRepo(t, "")

	out, err := runRemoteCmd(t, "remote", "update", "nonexistent")
	require.Error(t, err, "out=%s", out)
	assert.ErrorIs(t, err, repo.ErrRemoteNotFound)
}
