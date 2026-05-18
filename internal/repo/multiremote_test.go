// Multi-remote bug-hunting tests (BUG-103..BUG-109) for internal/repo.
//
// These tests exercise submodule-management primitives against parent repos
// built by the multiremote helper. Subtests are prefixed BUG-NNN so the
// pre-commit gate's failure filter (grep -vE 'BUG[-_]') can distinguish
// expected catalogued failures from real regressions. Several tests target
// behavior that is currently enforced at the CLI layer rather than the repo
// layer - these are EXPECTED to surface bugs (missing sentinel returns from
// repo functions) per the bug-hunting catalog model.
package repo_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/repo"
	"github.com/guneet-xyz/easyrice/internal/testutil/multiremote"
)

func allowFileProtocolMR(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "protocol.file.allow")
	t.Setenv("GIT_CONFIG_VALUE_0", "always")
}

func remoteRice(name string) string {
	return `schema_version = 1

[packages.` + name + `]
description = "remote ` + name + `"
supported_os = ["linux", "darwin", "windows"]
root = "."

[packages.` + name + `.profiles.default]
sources = [{path = ".", mode = "file", target = "$HOME/` + name + `"}]
`
}

func TestMultiRemoteRepoBugs(t *testing.T) {
	t.Run("BUG-103-ErrRemoteInUse", func(t *testing.T) {
		t.Log("BUG-103")
		// Parent rice.toml imports remotes/used; calling repo.SubmoduleRemove
		// directly should refuse with ErrRemoteInUse. NOTE: enforcement
		// currently lives in cli/remote.go, not in internal/repo - this test
		// is expected to FAIL until the check is pushed down.
		allowFileProtocolMR(t)
		fx := multiremote.New(t).
			AddRemoteRaw("used", map[string]string{
				"rice.toml": remoteRice("used"),
				"f":         "x\n",
			}).
			WithParentManifest(`schema_version = 1

[packages.consumer]
description = "imports the used remote"
supported_os = ["linux", "darwin", "windows"]

[packages.consumer.profiles.default]
import = "remotes/used#used.default"
`).Build()
		defer fx.Cleanup()

		err := repo.SubmoduleRemove(context.Background(), fx.ParentRepoPath, "remotes/used")
		assert.Error(t, err, "BUG-103: SubmoduleRemove should error when remote is imported")
		assert.True(t, errors.Is(err, repo.ErrRemoteInUse), "BUG-103: error must wrap ErrRemoteInUse, got %v", err)
	})

	t.Run("BUG-104-ErrRemoteAlreadyExists", func(t *testing.T) {
		t.Log("BUG-104")
		// Adding a second submodule at an existing remotes/<name> path should
		// fail with ErrRemoteAlreadyExists. Enforcement lives in cli/remote.go;
		// expect this test to surface the missing check at the repo layer.
		allowFileProtocolMR(t)
		fx := multiremote.New(t).
			AddRemoteRaw("dup", map[string]string{
				"rice.toml": remoteRice("dup"),
				"f":         "1\n",
			}).Build()
		defer fx.Cleanup()

		secondUpstream := t.TempDir()
		run := func(args ...string) {
			cmd := exec.Command("git", args...)
			cmd.Dir = secondUpstream
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, "BUG-104: setup git %v: %s", args, out)
		}
		run("init", "-b", "main")
		run("config", "user.email", "t@t")
		run("config", "user.name", "t")
		require.NoError(t, os.WriteFile(filepath.Join(secondUpstream, "rice.toml"), []byte(remoteRice("dup")), 0o644), "BUG-104: write second rice.toml")
		run("add", "-A")
		run("commit", "-m", "init")

		err := repo.SubmoduleAdd(context.Background(), fx.ParentRepoPath, "file://"+secondUpstream, "remotes/dup")
		assert.Error(t, err, "BUG-104: SubmoduleAdd should error when path exists")
		assert.True(t, errors.Is(err, repo.ErrRemoteAlreadyExists), "BUG-104: error must wrap ErrRemoteAlreadyExists, got %v", err)
	})

	t.Run("BUG-105-ConcurrentSubmoduleUpdateRace", func(t *testing.T) {
		t.Log("BUG-105")
		// Run SubmoduleUpdate twice concurrently against the same submodule
		// path. Goal: neither goroutine panics or corrupts the working tree.
		// May flake under -race; flakes are valuable signal for this catalog.
		allowFileProtocolMR(t)
		fx := multiremote.New(t).
			AddRemoteRaw("rc", map[string]string{
				"rice.toml": remoteRice("rc"),
				"f":         "v\n",
			}).Build()
		defer fx.Cleanup()

		var wg sync.WaitGroup
		errs := make([]error, 2)
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				errs[idx] = repo.SubmoduleUpdate(context.Background(), fx.ParentRepoPath, "remotes/rc")
			}(i)
		}
		wg.Wait()

		for i, err := range errs {
			assert.NoError(t, err, "BUG-105: concurrent SubmoduleUpdate[%d] should not error", i)
		}
		clean, err := repo.IsClean(context.Background(), fx.ParentRepoPath)
		require.NoError(t, err, "BUG-105: IsClean must not error after concurrent updates")
		assert.True(t, clean, "BUG-105: working tree must remain clean after concurrent updates")
	})

	t.Run("BUG-106-SubmoduleListSurvivesUninitialized", func(t *testing.T) {
		t.Log("BUG-106")
		// SubmoduleList must return entries even when a submodule is deinit'd.
		allowFileProtocolMR(t)
		fx := multiremote.New(t).
			AddRemoteRaw("listed", map[string]string{
				"rice.toml": remoteRice("listed"),
				"f":         "v\n",
			}).
			WithUninitSubmodule("listed").
			Build()
		defer fx.Cleanup()

		subs, err := repo.SubmoduleList(context.Background(), fx.ParentRepoPath)
		require.NoError(t, err, "BUG-106: SubmoduleList must not error on uninit submodules")
		require.Len(t, subs, 1, "BUG-106: must list the one declared submodule")
		assert.Equal(t, "remotes/listed", subs[0].Path, "BUG-106: path must match")
		assert.Equal(t, repo.SubmoduleNotInitialized, subs[0].State, "BUG-106: state must be NotInitialized, got %v", subs[0].State)
	})

	t.Run("BUG-107-ErrRepoDirtyOnRemoteAdd", func(t *testing.T) {
		t.Log("BUG-107")
		// SubmoduleAdd on a parent with uncommitted changes should refuse with
		// ErrRepoDirty. Enforcement lives in cli/remote.go; expect this test
		// to surface the missing check at the repo layer.
		allowFileProtocolMR(t)
		fx := multiremote.New(t).Build()
		defer fx.Cleanup()

		require.NoError(t, os.WriteFile(filepath.Join(fx.ParentRepoPath, "dirty.txt"), []byte("uncommitted\n"), 0o644), "BUG-107: write dirty file")

		dirty, err := repo.HasUncommittedChanges(context.Background(), fx.ParentRepoPath)
		require.NoError(t, err, "BUG-107: HasUncommittedChanges must not error")
		require.True(t, dirty, "BUG-107: setup precondition - tree must be dirty")

		upstream := t.TempDir()
		run := func(args ...string) {
			cmd := exec.Command("git", args...)
			cmd.Dir = upstream
			out, e := cmd.CombinedOutput()
			require.NoError(t, e, "BUG-107: setup git %v: %s", args, out)
		}
		run("init", "-b", "main")
		run("config", "user.email", "t@t")
		run("config", "user.name", "t")
		require.NoError(t, os.WriteFile(filepath.Join(upstream, "rice.toml"), []byte(remoteRice("new")), 0o644), "BUG-107: upstream rice.toml")
		run("add", "-A")
		run("commit", "-m", "init")

		err = repo.SubmoduleAdd(context.Background(), fx.ParentRepoPath, "file://"+upstream, "remotes/new")
		assert.Error(t, err, "BUG-107: SubmoduleAdd should refuse dirty tree")
		assert.True(t, errors.Is(err, repo.ErrRepoDirty), "BUG-107: error must wrap ErrRepoDirty, got %v", err)
	})

	t.Run("BUG-108-CommitPathsScopedToRemoteOnly", func(t *testing.T) {
		t.Log("BUG-108")
		// CommitPaths must scope `git add` to specified paths only. Place an
		// unrelated dirty file and assert it is NOT in the resulting commit.
		allowFileProtocolMR(t)
		fx := multiremote.New(t).
			AddRemoteRaw("scoped", map[string]string{
				"rice.toml": remoteRice("scoped"),
				"f":         "v\n",
			}).Build()
		defer fx.Cleanup()

		strayPath := filepath.Join(fx.ParentRepoPath, "stray.txt")
		require.NoError(t, os.WriteFile(strayPath, []byte("must-not-commit\n"), 0o644), "BUG-108: write stray")

		modulesPath := filepath.Join(fx.ParentRepoPath, ".gitmodules")
		data, err := os.ReadFile(modulesPath)
		require.NoError(t, err, "BUG-108: read .gitmodules")
		require.NoError(t, os.WriteFile(modulesPath, append(data, []byte("\n# touch\n")...), 0o644), "BUG-108: touch .gitmodules")

		err = repo.CommitPaths(context.Background(), fx.ParentRepoPath, []string{".gitmodules"}, "scoped commit")
		require.NoError(t, err, "BUG-108: CommitPaths must succeed")

		cmd := exec.Command("git", "-C", fx.ParentRepoPath, "log", "-1", "--name-only", "--pretty=format:")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "BUG-108: git log: %s", out)
		assert.NotContains(t, string(out), "stray.txt", "BUG-108: stray file must NOT be in latest commit; got %q", string(out))
		assert.Contains(t, string(out), ".gitmodules", "BUG-108: .gitmodules must be in latest commit; got %q", string(out))

		dirty, err := repo.HasUncommittedChanges(context.Background(), fx.ParentRepoPath)
		require.NoError(t, err, "BUG-108: HasUncommittedChanges must not error")
		assert.True(t, dirty, "BUG-108: stray file must remain uncommitted after scoped commit")
	})

	t.Run("BUG-109-ErrRemoteNotFoundOnRemoveMissing", func(t *testing.T) {
		t.Log("BUG-109")
		// Removing a non-existent submodule should return ErrRemoteNotFound.
		// Enforcement lives in cli/remote.go; expect this test to surface the
		// missing check at the repo layer.
		allowFileProtocolMR(t)
		fx := multiremote.New(t).Build()
		defer fx.Cleanup()

		err := repo.SubmoduleRemove(context.Background(), fx.ParentRepoPath, "remotes/ghost")
		assert.Error(t, err, "BUG-109: SubmoduleRemove on missing path should error")
		assert.True(t, errors.Is(err, repo.ErrRemoteNotFound), "BUG-109: error must wrap ErrRemoteNotFound, got %v", err)
	})
}
