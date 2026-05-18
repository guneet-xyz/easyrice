//go:build !windows

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/repo"
	"github.com/guneet-xyz/easyrice/internal/updater"
)

// TestCLI_SentinelErrors covers BUG-160..BUG-178: every documented sentinel
// error path must surface a clear human-readable message via the CLI.
//
// Boundary: invokes the CLI in-process via the same helpers existing tests
// already use (runInstallCmd / runRootCmd / runRemoteCmd from peer test files).
// e2e_helpers_test.go is consulted, not modified.
//
// The pre-commit gate ignores BUG-tagged failures, so sub-tests that encode
// the *documented* contract (rather than current production behavior) may
// legitimately fail and remain in the catalog as `failing`.
func TestCLI_SentinelErrors(t *testing.T) {
	// BUG-160 — ErrRepoNotInitialized surfaced via `rice install`.
	// Contract: diagnosis (repo not initialized) AND next-step hint (`rice init`).
	t.Run("BUG-160-RepoNotInitialized-Install", func(t *testing.T) {
		t.Log("BUG-160")
		resetInstallFlags()
		t.Cleanup(resetInstallFlags)
		setIsolatedHome(t)
		statePath := filepath.Join(t.TempDir(), "state.json")

		out, err := runInstallCmd(t, "",
			"--state", statePath,
			"--yes",
			"install", "anything",
		)
		require.Error(t, err, "BUG-160: install must error when no managed repo; out=%s", out)
		assert.True(t, errors.Is(err, repo.ErrRepoNotInitialized),
			"BUG-160: error chain must include ErrRepoNotInitialized; got: %v", err)
		msg := err.Error()
		assert.Contains(t, msg, "not initialized",
			"BUG-160: stderr must diagnose 'not initialized'; got: %s", msg)
		assert.Contains(t, msg, "rice init",
			"BUG-160: stderr must hint next step 'rice init'; got: %s", msg)
	})

	// BUG-161 — ErrRepoNotInitialized surfaced via `rice update`.
	t.Run("BUG-161-RepoNotInitialized-Update", func(t *testing.T) {
		t.Log("BUG-161")
		resetInstallFlags()
		t.Cleanup(resetInstallFlags)
		setIsolatedHome(t)

		out, err := runRootCmd(t, "update")
		require.Error(t, err, "BUG-161: update must error when no managed repo; out=%s", out)
		assert.True(t, errors.Is(err, repo.ErrRepoNotInitialized),
			"BUG-161: error chain must include ErrRepoNotInitialized; got: %v", err)
		assert.Contains(t, err.Error(), "rice init",
			"BUG-161: stderr must hint next step 'rice init'; got: %s", err.Error())
	})

	// BUG-162 — ErrRepoNotInitialized surfaced via `rice remote list`.
	t.Run("BUG-162-RepoNotInitialized-RemoteList", func(t *testing.T) {
		t.Log("BUG-162")
		resetInstallFlags()
		t.Cleanup(resetInstallFlags)
		setIsolatedHome(t)

		out, err := runRootCmd(t, "remote", "list")
		require.Error(t, err, "BUG-162: remote list must error when no managed repo; out=%s", out)
		assert.True(t, errors.Is(err, repo.ErrRepoNotInitialized),
			"BUG-162: error chain must include ErrRepoNotInitialized; got: %v", err)
		assert.Contains(t, err.Error(), "rice init",
			"BUG-162: stderr must hint next step 'rice init'; got: %s", err.Error())
	})

	// BUG-163 — ErrPackageNotDeclared names the missing package and lists
	// available packages (or points to `rice status`).
	t.Run("BUG-163-PackageNotDeclared", func(t *testing.T) {
		t.Log("BUG-163")
		resetInstallFlags()
		t.Cleanup(resetInstallFlags)
		_, statePath, _ := setupTestRepo(t)

		_, err := runInstallCmd(t, "",
			"--state", statePath,
			"--yes",
			"install", "ghost",
			"--profile", "common",
		)
		require.Error(t, err, "BUG-163: install of undeclared package must error")
		msg := err.Error()
		assert.Contains(t, msg, "ghost",
			"BUG-163: error must name the missing package; got: %s", msg)
		assert.Contains(t, msg, "not declared",
			"BUG-163: error must say 'not declared'; got: %s", msg)
		// Documented UX contract: list available packages OR point to `rice status`.
		// (production today does neither — this assertion is the failing
		// half of BUG-163 and exists to drive the fix.)
		hasList := strings.Contains(msg, "mypkg")
		hasStatusHint := strings.Contains(msg, "rice status")
		assert.True(t, hasList || hasStatusHint,
			"BUG-163: error must list available packages or hint `rice status`; got: %s", msg)
	})

	// BUG-164 — ErrRepoDirty on `rice remote add`: refusing because of uncommitted changes.
	t.Run("BUG-164-RepoDirty-RemoteAdd", func(t *testing.T) {
		t.Log("BUG-164")
		if _, lookErr := exec.LookPath("git"); lookErr != nil {
			t.Skip("BUG-164: git not on PATH")
		}
		setIsolatedHome(t)
		root := makeManagedRepo(t, "")
		// Dirty the working tree.
		require.NoError(t, os.WriteFile(filepath.Join(root, "dirty.txt"), []byte("x"), 0o644))

		out, err := runRemoteCmd(t, "remote", "add", "https://example.invalid/x.git", "--name", "r1")
		require.Error(t, err, "BUG-164: remote add must refuse on dirty tree; out=%s", out)
		assert.True(t, errors.Is(err, repo.ErrRepoDirty),
			"BUG-164: error chain must include ErrRepoDirty; got: %v", err)
		msg := err.Error()
		assert.Contains(t, msg, "uncommitted",
			"BUG-164: stderr must say 'uncommitted'; got: %s", msg)
	})

	// BUG-165 — ErrRemoteAlreadyExists on duplicate `rice remote add`.
	t.Run("BUG-165-RemoteAlreadyExists", func(t *testing.T) {
		t.Log("BUG-165")
		if _, lookErr := exec.LookPath("git"); lookErr != nil {
			t.Skip("BUG-165: git not on PATH")
		}
		setIsolatedHome(t)
		makeManagedRepo(t, "")
		upstream := makeBareUpstreamRice(t)

		_, err := runRemoteCmd(t, "remote", "add", upstream, "--name", "dup")
		require.NoError(t, err, "BUG-165: first remote add must succeed")

		_, err = runRemoteCmd(t, "remote", "add", upstream, "--name", "dup")
		require.Error(t, err, "BUG-165: second remote add must fail")
		assert.True(t, errors.Is(err, repo.ErrRemoteAlreadyExists),
			"BUG-165: error chain must include ErrRemoteAlreadyExists; got: %v", err)
		// Documented UX: stderr names the conflicting remote.
		assert.Contains(t, err.Error(), "dup",
			"BUG-165: error must name the conflicting remote 'dup'; got: %s", err.Error())
	})

	// BUG-166 — ErrRemoteNotFound on `rice remote remove <missing>`.
	t.Run("BUG-166-RemoteNotFound", func(t *testing.T) {
		t.Log("BUG-166")
		if _, lookErr := exec.LookPath("git"); lookErr != nil {
			t.Skip("BUG-166: git not on PATH")
		}
		setIsolatedHome(t)
		makeManagedRepo(t, "")

		_, err := runRemoteCmd(t, "remote", "remove", "ghostremote")
		require.Error(t, err, "BUG-166: remove of missing remote must error")
		assert.True(t, errors.Is(err, repo.ErrRemoteNotFound),
			"BUG-166: error chain must include ErrRemoteNotFound; got: %v", err)
		// Documented UX: stderr names the remote.
		assert.Contains(t, err.Error(), "ghostremote",
			"BUG-166: error must name the missing remote 'ghostremote'; got: %s", err.Error())
	})

	// BUG-167 — ErrRemoteInUse names BOTH the remote and the consuming profile.
	t.Run("BUG-167-RemoteInUse", func(t *testing.T) {
		t.Log("BUG-167")
		if _, lookErr := exec.LookPath("git"); lookErr != nil {
			t.Skip("BUG-167: git not on PATH")
		}
		setIsolatedHome(t)
		manifestContent := `schema_version = 1

[packages.local]
description = "local"
supported_os = ["linux", "darwin", "windows"]

[packages.local.profiles.common]
import = "remotes/used#upstreamtool.common"
sources = [{path = "common", mode = "file", target = "$HOME"}]
`
		makeManagedRepo(t, manifestContent)
		upstream := makeBareUpstreamRice(t)

		_, err := runRemoteCmd(t, "remote", "add", upstream, "--name", "used")
		require.NoError(t, err)

		_, err = runRemoteCmd(t, "remote", "remove", "used")
		require.Error(t, err, "BUG-167: remove of in-use remote must error")
		assert.True(t, errors.Is(err, repo.ErrRemoteInUse),
			"BUG-167: error chain must include ErrRemoteInUse; got: %v", err)
		msg := err.Error()
		// Documented UX: names BOTH the remote and which profile uses it.
		assert.Contains(t, msg, "used",
			"BUG-167: error must name the remote 'used'; got: %s", msg)
		assert.Contains(t, msg, "local.common",
			"BUG-167: error must name the consuming profile 'local.common'; got: %s", msg)
	})

	// BUG-168 — ErrSubmoduleNotInitialized surfaces when a profile imports
	// from a missing/uninitialized remote; message hints `rice remote update`.
	t.Run("BUG-168-SubmoduleNotInitialized", func(t *testing.T) {
		t.Log("BUG-168")
		resetInstallFlags()
		t.Cleanup(resetInstallFlags)
		setIsolatedHome(t)
		repoRoot := repo.DefaultRepoPath()
		statePath := filepath.Join(t.TempDir(), "state.json")

		// Manifest with an import pointing at a remote that does not exist
		// on disk → ResolveSpecs returns ErrSubmoduleNotInitialized.
		manifestBody := `schema_version = 1

[packages.local]
description = "local"
supported_os = ["linux", "darwin", "windows"]

[packages.local.profiles.common]
import = "remotes/missing#upstreamtool.common"
`
		require.NoError(t, os.MkdirAll(repoRoot, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(repoRoot, "rice.toml"), []byte(manifestBody), 0o644))

		_, err := runInstallCmd(t, "",
			"--state", statePath,
			"--yes",
			"install", "local",
			"--profile", "common",
		)
		require.Error(t, err, "BUG-168: install must error when import remote is missing")
		// Either the typed sentinel propagates or its message text appears.
		msg := err.Error()
		hasSentinel := errors.Is(err, repo.ErrSubmoduleNotInitialized)
		hasHint := strings.Contains(msg, "rice remote update")
		assert.True(t, hasSentinel || hasHint,
			"BUG-168: error must be ErrSubmoduleNotInitialized or hint `rice remote update`; got: %v", err)
	})

	// BUG-169 — Updater ErrDevBuild on `rice upgrade`.
	t.Run("BUG-169-Updater-DevBuild", func(t *testing.T) {
		t.Log("BUG-169")
		saveUpgradeState(t)
		resetInstallFlags()
		Version = "dev"

		out, err := runInstallCmd(t, "", "upgrade")
		require.Error(t, err, "BUG-169: dev build must refuse upgrade; out=%s", out)
		assert.True(t, errors.Is(err, updater.ErrDevBuild),
			"BUG-169: error chain must include ErrDevBuild; got: %v", err)
		// Combined cobra buffer captures stderr-targeted Fprintln in runUpgrade.
		assert.Contains(t, out, "dev build",
			"BUG-169: stderr must say 'dev build'; got: %s", out)
	})

	// BUG-170 — Updater ErrAlreadyLatest on `rice upgrade` prints version.
	t.Run("BUG-170-Updater-AlreadyLatest", func(t *testing.T) {
		t.Log("BUG-170")
		saveUpgradeState(t)
		resetInstallFlags()
		Version = "v1.0.0"

		upgradeFetchFn = func(ctx context.Context) (*updater.Updater, *updater.Release, error) {
			return nil, nil, updater.ErrAlreadyLatest
		}

		out, err := runInstallCmd(t, "", "upgrade")
		require.NoError(t, err, "BUG-170: ErrAlreadyLatest must surface as a clean success; out=%s", out)
		assert.Contains(t, out, "up to date",
			"BUG-170: output must say 'up to date'; got: %s", out)
		assert.Contains(t, out, "v1.0.0",
			"BUG-170: output must include the version; got: %s", out)
	})

	// BUG-171 — Updater ErrLockBusy on `rice upgrade`.
	t.Run("BUG-171-Updater-LockBusy", func(t *testing.T) {
		t.Log("BUG-171")
		saveUpgradeState(t)
		resetInstallFlags()
		Version = "v1.0.0"

		upgradeFetchFn = func(ctx context.Context) (*updater.Updater, *updater.Release, error) {
			return nil, nil, fmt.Errorf("lock contention: %w", updater.ErrLockBusy)
		}

		out, err := runInstallCmd(t, "", "--yes", "upgrade")
		require.Error(t, err, "BUG-171: ErrLockBusy must surface as an error; out=%s", out)
		assert.True(t, errors.Is(err, updater.ErrLockBusy),
			"BUG-171: error chain must include ErrLockBusy; got: %v", err)
		// Documented UX: message points at another in-progress upgrade.
		msg := err.Error()
		assert.True(t,
			strings.Contains(msg, "another") || strings.Contains(msg, "in progress") ||
				strings.Contains(msg, "update is in progress"),
			"BUG-171: error must hint another upgrade is in progress; got: %s", msg)
	})

	// BUG-172 — Updater ErrNoChecksum on `rice upgrade`.
	t.Run("BUG-172-Updater-NoChecksum", func(t *testing.T) {
		t.Log("BUG-172")
		saveUpgradeState(t)
		resetInstallFlags()
		Version = "v1.0.0"

		upgradeFetchFn = func(ctx context.Context) (*updater.Updater, *updater.Release, error) {
			return nil, nil, fmt.Errorf("fetch latest: %w", updater.ErrNoChecksum)
		}

		out, err := runInstallCmd(t, "", "--yes", "upgrade")
		require.Error(t, err, "BUG-172: ErrNoChecksum must surface as an error; out=%s", out)
		assert.True(t, errors.Is(err, updater.ErrNoChecksum),
			"BUG-172: error chain must include ErrNoChecksum; got: %v", err)
		assert.Contains(t, err.Error(), "checksum",
			"BUG-172: error must mention 'checksum'; got: %s", err.Error())
	})

	// BUG-173 — Updater ErrInvalidSemver on `rice upgrade`.
	t.Run("BUG-173-Updater-InvalidSemver", func(t *testing.T) {
		t.Log("BUG-173")
		saveUpgradeState(t)
		resetInstallFlags()
		Version = "v1.0.0"

		upgradeFetchFn = func(ctx context.Context) (*updater.Updater, *updater.Release, error) {
			return nil, nil, fmt.Errorf("parse version: %w", updater.ErrInvalidSemver)
		}

		out, err := runInstallCmd(t, "", "--yes", "upgrade")
		require.Error(t, err, "BUG-173: ErrInvalidSemver must surface as an error; out=%s", out)
		assert.True(t, errors.Is(err, updater.ErrInvalidSemver),
			"BUG-173: error chain must include ErrInvalidSemver; got: %v", err)
		msg := err.Error()
		assert.True(t,
			strings.Contains(msg, "semver") || strings.Contains(msg, "version"),
			"BUG-173: error must explain the invalid version; got: %s", msg)
	})

	// BUG-174 — unknown profile lists available profiles for the package.
	t.Run("BUG-174-UnknownProfile", func(t *testing.T) {
		t.Log("BUG-174")
		resetInstallFlags()
		t.Cleanup(resetInstallFlags)
		_, statePath, _ := setupTestRepo(t)

		_, err := runInstallCmd(t, "",
			"--state", statePath,
			"--yes",
			"install", "mypkg",
			"--profile", "ghost-profile",
		)
		require.Error(t, err, "BUG-174: install with unknown profile must error")
		msg := err.Error()
		assert.Contains(t, msg, "ghost-profile",
			"BUG-174: error must name the missing profile; got: %s", msg)
		// Documented UX: list available profiles for the package.
		hasMacbook := strings.Contains(msg, "macbook")
		hasCommon := strings.Contains(msg, "common")
		assert.True(t, hasMacbook || hasCommon,
			"BUG-174: error must list available profiles (macbook/common); got: %s", msg)
	})

	// BUG-175 — no rice.toml at repo root: clear "rice.toml not found" with path.
	t.Run("BUG-175-ManifestMissing", func(t *testing.T) {
		t.Log("BUG-175")
		resetInstallFlags()
		t.Cleanup(resetInstallFlags)
		setIsolatedHome(t)
		repoRoot := repo.DefaultRepoPath()
		require.NoError(t, os.MkdirAll(repoRoot, 0o755))
		statePath := filepath.Join(t.TempDir(), "state.json")

		_, err := runInstallCmd(t, "",
			"--state", statePath,
			"--yes",
			"install", "anything",
			"--profile", "common",
		)
		require.Error(t, err, "BUG-175: install must error when rice.toml is missing")
		msg := err.Error()
		assert.Contains(t, msg, "rice.toml",
			"BUG-175: error must name 'rice.toml'; got: %s", msg)
		assert.Contains(t, msg, "not found",
			"BUG-175: error must say 'not found'; got: %s", msg)
		assert.Contains(t, msg, repoRoot,
			"BUG-175: error must include the repo path; got: %s", msg)
	})

	// BUG-176 — `rice doctor` reports legacy state file drift clearly.
	t.Run("BUG-176-DoctorLegacyState", func(t *testing.T) {
		t.Log("BUG-176")
		resetInstallFlags()
		t.Cleanup(resetInstallFlags)
		home := setIsolatedHome(t)
		// Place a legacy state file (no new state file).
		legacyDir := filepath.Join(home, ".config", "rice")
		require.NoError(t, os.MkdirAll(legacyDir, 0o755))
		legacyPath := filepath.Join(legacyDir, "state.json")
		require.NoError(t, os.WriteFile(legacyPath, []byte("{}"), 0o644))

		// Point --state at a non-existent path so the new-state check fails-open.
		statePath := filepath.Join(t.TempDir(), "state.json")

		out, _ := runInstallCmd(t, "",
			"--state", statePath,
			"doctor",
		)
		// Doctor may return error or nil depending on git availability; we only
		// require the legacy-state warning to appear in combined output.
		assert.Contains(t, out, "Legacy rice state",
			"BUG-176: doctor must warn about legacy state; got: %s", out)
		assert.Contains(t, out, legacyPath,
			"BUG-176: warning must include the legacy path; got: %s", out)
	})

	// BUG-177 — --state with un-creatable parent yields a clear error.
	t.Run("BUG-177-StateParentUncreatable", func(t *testing.T) {
		t.Log("BUG-177")
		resetInstallFlags()
		t.Cleanup(resetInstallFlags)
		_, _, _ = setupTestRepo(t)

		// Parent is a regular file: state.Save's os.MkdirAll(dir) will fail
		// with ENOTDIR, surfacing the "cannot create state directory" path.
		blocker := filepath.Join(t.TempDir(), "blocker")
		require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))
		statePath := filepath.Join(blocker, "subdir", "state.json")

		_, err := runInstallCmd(t, "",
			"--state", statePath,
			"--yes",
			"install", "mypkg",
			"--profile", "common",
		)
		require.Error(t, err, "BUG-177: install must error when state parent dir cannot be created")
		msg := err.Error()
		// Documented UX: error mentions inability to create the state directory.
		hasState := strings.Contains(msg, "state")
		hasDirHint := strings.Contains(msg, "directory") || strings.Contains(msg, "create") ||
			strings.Contains(msg, "save") || strings.Contains(msg, "load")
		assert.True(t, hasState && hasDirHint,
			"BUG-177: error must mention state + create/directory; got: %s", msg)
	})

	// BUG-178 — --log-level invalid lists valid levels.
	t.Run("BUG-178-InvalidLogLevel", func(t *testing.T) {
		t.Log("BUG-178")
		resetInstallFlags()
		t.Cleanup(resetInstallFlags)
		setIsolatedHome(t)

		_, err := runRootCmd(t, "--log-level", "bogus", "version")
		require.Error(t, err, "BUG-178: invalid --log-level must error")
		msg := err.Error()
		assert.Contains(t, msg, "log level",
			"BUG-178: error must mention 'log level'; got: %s", msg)
		// Documented UX: list of valid levels.
		for _, lvl := range []string{"debug", "info", "warn", "error", "critical"} {
			assert.Contains(t, msg, lvl,
				"BUG-178: error must list valid level %q; got: %s", lvl, msg)
		}
	})
}
