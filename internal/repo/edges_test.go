package repo

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// ErrDetachedHEAD is the sentinel BUG-142 asserts production should return
// from CurrentBranch when HEAD is detached. It is declared here (test-only)
// because production has not yet introduced it — that absence IS BUG-142.
// errors.Is against this value will be false until production catches up,
// which causes BUG-142 to FAIL at the assertion site (intended).
var ErrDetachedHEAD = errors.New("repo: HEAD is detached (no current branch)")

// TestRepo_Edges pins behavioural contracts for git CLI integration edges.
// Each sub-test carries a BUG-NNN marker; failures here are catalogued in
// .omo/known-bugs.md. See .omo/plans/better-tests.md lines 1661-1737.
//
// NOTE: this file uses exec.Command("git", ...) only for test setup (detached
// HEAD, dirty trees). Production code MUST still go through internal/repo/.
func TestRepo_Edges(t *testing.T) {
	t.Run("BUG-140-GitNotOnPath", func(t *testing.T) {
		t.Log("BUG-140")
		t.Setenv("PATH", "")
		t.Setenv("GIT_EXEC_PATH", "")

		if _, err := exec.LookPath("git"); err == nil {
			t.Fatalf("BUG-140: precondition failed, git still resolvable after PATH=\"\"")
		}

		if GitOnPath() {
			t.Errorf("BUG-140: GitOnPath() = true, want false when git absent from PATH")
		}

		ctx := context.Background()
		dest := filepath.Join(t.TempDir(), "clone-dest")
		if err := Clone(ctx, "file:///nonexistent", dest); err == nil {
			t.Errorf("BUG-140: Clone returned nil error with git absent")
		} else if !strings.Contains(err.Error(), "git") {
			t.Errorf("BUG-140: Clone error %q does not mention git", err)
		}

		fakeRepo := t.TempDir()
		if err := Pull(ctx, fakeRepo); err == nil {
			t.Errorf("BUG-140: Pull returned nil error with git absent")
		} else if !strings.Contains(err.Error(), "git") {
			t.Errorf("BUG-140: Pull error %q does not mention git", err)
		}

		if err := SubmoduleAdd(ctx, fakeRepo, "file:///x", "remotes/x"); err == nil {
			t.Errorf("BUG-140: SubmoduleAdd returned nil error with git absent")
		} else if !strings.Contains(err.Error(), "git") {
			t.Errorf("BUG-140: SubmoduleAdd error %q does not mention git", err)
		}

		if err := SubmoduleUpdate(ctx, fakeRepo, ""); err == nil {
			t.Errorf("BUG-140: SubmoduleUpdate returned nil error with git absent")
		} else if !strings.Contains(err.Error(), "git") {
			t.Errorf("BUG-140: SubmoduleUpdate error %q does not mention git", err)
		}
	})

	t.Run("BUG-141-NonExistentRepoPath", func(t *testing.T) {
		t.Log("BUG-141")
		skipIfNoGit(t)
		missing := filepath.Join(t.TempDir(), "does-not-exist")
		ctx := context.Background()

		if _, err := IsClean(ctx, missing); err == nil {
			t.Errorf("BUG-141: IsClean returned nil error for missing repo path")
		} else if !errors.Is(err, ErrRepoNotInitialized) {
			t.Errorf("BUG-141: IsClean error %v is not ErrRepoNotInitialized", err)
		}

		if _, err := HasUncommittedChanges(ctx, missing); err == nil {
			t.Errorf("BUG-141: HasUncommittedChanges returned nil error for missing repo path")
		} else if !errors.Is(err, ErrRepoNotInitialized) {
			t.Errorf("BUG-141: HasUncommittedChanges error %v is not ErrRepoNotInitialized", err)
		}

		if _, err := CurrentBranch(ctx, missing); err == nil {
			t.Errorf("BUG-141: CurrentBranch returned nil error for missing repo path")
		} else if !errors.Is(err, ErrRepoNotInitialized) {
			t.Errorf("BUG-141: CurrentBranch error %v is not ErrRepoNotInitialized", err)
		}
	})

	t.Run("BUG-142-DetachedHEAD", func(t *testing.T) {
		t.Log("BUG-142")
		skipIfNoGit(t)
		repoPath := initRepoWithCommit(t)

		out, err := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD").CombinedOutput()
		if err != nil {
			t.Fatalf("BUG-142: setup git rev-parse: %v: %s", err, out)
		}
		sha := strings.TrimSpace(string(out))

		if out, err := exec.Command("git", "-C", repoPath, "checkout", sha).CombinedOutput(); err != nil {
			t.Fatalf("BUG-142: setup git checkout %s: %v: %s", sha, err, out)
		}

		ctx := context.Background()
		branch, err := CurrentBranch(ctx, repoPath)
		if err == nil {
			t.Errorf("BUG-142: CurrentBranch on detached HEAD returned nil error, branch=%q (want \"\" + ErrDetachedHEAD)", branch)
		} else if !errors.Is(err, ErrDetachedHEAD) {
			t.Errorf("BUG-142: CurrentBranch error %v is not ErrDetachedHEAD", err)
		}
		if branch != "" {
			t.Errorf("BUG-142: CurrentBranch returned %q on detached HEAD, want \"\"", branch)
		}
	})

	t.Run("BUG-143-EmptyRepoIsClean", func(t *testing.T) {
		t.Log("BUG-143")
		skipIfNoGit(t)
		repoPath := initRepo(t)

		ctx := context.Background()
		clean, err := IsClean(ctx, repoPath)
		if err != nil {
			t.Errorf("BUG-143: IsClean on empty repo returned error %v, want nil", err)
		}
		if !clean {
			t.Errorf("BUG-143: IsClean = false on empty repo (no commits, no untracked), want true")
		}
	})

	t.Run("BUG-144-CommitPathsEmptySlice", func(t *testing.T) {
		t.Log("BUG-144")
		skipIfNoGit(t)
		repoPath := initRepoWithCommit(t)

		err := CommitPaths(context.Background(), repoPath, []string{}, "msg")
		if err == nil {
			t.Errorf("BUG-144: CommitPaths with empty paths returned nil error")
		} else {
			msg := strings.ToLower(err.Error())
			if !strings.Contains(msg, "empty") && !strings.Contains(msg, "no paths") {
				t.Errorf("BUG-144: CommitPaths empty-paths error %q does not mention 'empty' or 'no paths'", err)
			}
		}
	})

	t.Run("BUG-145-CommitPathsWithSpaces", func(t *testing.T) {
		t.Log("BUG-145")
		skipIfNoGit(t)
		repoPath := initRepoWithCommit(t)

		rel := "file with spaces.txt"
		full := filepath.Join(repoPath, rel)
		if err := os.WriteFile(full, []byte("hello"), 0o644); err != nil {
			t.Fatalf("BUG-145: setup write: %v", err)
		}

		ctx := context.Background()
		if err := CommitPaths(ctx, repoPath, []string{rel}, "add spaced file"); err != nil {
			t.Errorf("BUG-145: CommitPaths with spaced path returned error: %v", err)
			return
		}

		out, err := exec.Command("git", "-C", repoPath, "log", "-1", "--name-only", "--pretty=format:").CombinedOutput()
		if err != nil {
			t.Fatalf("BUG-145: verify git log: %v: %s", err, out)
		}
		if !strings.Contains(string(out), rel) {
			t.Errorf("BUG-145: latest commit does not contain %q; got:\n%s", rel, out)
		}
	})

	t.Run("BUG-146-CommitPathsOutsideRepo", func(t *testing.T) {
		t.Log("BUG-146")
		skipIfNoGit(t)
		repoPath := initRepoWithCommit(t)

		outsideRel := filepath.Join("..", "outside.txt")

		err := CommitPaths(context.Background(), repoPath, []string{outsideRel}, "msg")
		if err == nil {
			t.Errorf("BUG-146: CommitPaths accepted path outside repo (%q) with no error", outsideRel)
		}
	})

	t.Run("BUG-147-CommitPathsAbsolutePath", func(t *testing.T) {
		t.Log("BUG-147")
		skipIfNoGit(t)
		repoPath := initRepoWithCommit(t)

		absPath := filepath.Join(repoPath, "abs.txt")
		if err := os.WriteFile(absPath, []byte("x"), 0o644); err != nil {
			t.Fatalf("BUG-147: setup write: %v", err)
		}

		err := CommitPaths(context.Background(), repoPath, []string{absPath}, "msg")
		if err == nil {
			t.Errorf("BUG-147: CommitPaths accepted absolute path (%q) with no error; paths must be relative", absPath)
		}
	})

	t.Run("BUG-148-PullOnDirtyTree", func(t *testing.T) {
		t.Log("BUG-148")
		skipIfNoGit(t)
		allowFileProtocol(t)

		bare := filepath.Join(t.TempDir(), "bare.git")
		run(t, "git", "init", "--bare", "-b", "main", bare)

		seed := filepath.Join(t.TempDir(), "seed")
		run(t, "git", "clone", bare, seed)
		mustGitConfig(t, seed)
		if err := os.WriteFile(filepath.Join(seed, "seed.txt"), []byte("seed"), 0o644); err != nil {
			t.Fatalf("BUG-148: seed write: %v", err)
		}
		run(t, "git", "-C", seed, "add", "seed.txt")
		run(t, "git", "-C", seed, "commit", "-m", "seed")
		run(t, "git", "-C", seed, "push", "origin", "main")

		clone := filepath.Join(t.TempDir(), "clone")
		run(t, "git", "clone", bare, clone)
		mustGitConfig(t, clone)

		if err := os.WriteFile(filepath.Join(clone, "dirty.txt"), []byte("dirty"), 0o644); err != nil {
			t.Fatalf("BUG-148: dirty write: %v", err)
		}

		err := Pull(context.Background(), clone)
		if err == nil {
			t.Errorf("BUG-148: Pull on dirty tree returned nil error; want ErrRepoDirty")
		} else if !errors.Is(err, ErrRepoDirty) {
			t.Errorf("BUG-148: Pull on dirty tree error %v is not ErrRepoDirty", err)
		}
	})

	t.Run("BUG-149-CloneToExistingPath", func(t *testing.T) {
		t.Log("BUG-149")
		skipIfNoGit(t)
		allowFileProtocol(t)

		bare := filepath.Join(t.TempDir(), "bare.git")
		run(t, "git", "init", "--bare", "-b", "main", bare)
		seed := filepath.Join(t.TempDir(), "seed")
		run(t, "git", "clone", bare, seed)
		mustGitConfig(t, seed)
		if err := os.WriteFile(filepath.Join(seed, "x.txt"), []byte("x"), 0o644); err != nil {
			t.Fatalf("BUG-149: seed write: %v", err)
		}
		run(t, "git", "-C", seed, "add", "x.txt")
		run(t, "git", "-C", seed, "commit", "-m", "x")
		run(t, "git", "-C", seed, "push", "origin", "main")

		dest := filepath.Join(t.TempDir(), "dest")
		if err := os.MkdirAll(dest, 0o755); err != nil {
			t.Fatalf("BUG-149: mkdir dest: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dest, "prior.txt"), []byte("prior"), 0o644); err != nil {
			t.Fatalf("BUG-149: prior write: %v", err)
		}

		err := Clone(context.Background(), bare, dest)
		if err == nil {
			t.Errorf("BUG-149: Clone to existing non-empty path returned nil error")
		}
	})

	t.Run("BUG-150-CloneInvalidURL", func(t *testing.T) {
		t.Log("BUG-150")
		skipIfNoGit(t)
		t.Setenv("GIT_TERMINAL_PROMPT", "0")

		dest := filepath.Join(t.TempDir(), "dest")
		badURL := "not-a-valid-url::::"

		err := Clone(context.Background(), badURL, dest)
		if err == nil {
			t.Errorf("BUG-150: Clone with invalid URL %q returned nil error", badURL)
			return
		}
		msg := err.Error()
		if !strings.Contains(msg, badURL) && !strings.Contains(strings.ToLower(msg), "url") && !strings.Contains(strings.ToLower(msg), "clone") {
			t.Errorf("BUG-150: Clone invalid-URL error %q does not mention URL or 'clone'", err)
		}
	})

	t.Run("BUG-151-PullNoUpstream", func(t *testing.T) {
		t.Log("BUG-151")
		skipIfNoGit(t)
		t.Setenv("GIT_TERMINAL_PROMPT", "0")

		repoPath := initRepoWithCommit(t)

		err := Pull(context.Background(), repoPath)
		if err == nil {
			t.Errorf("BUG-151: Pull with no upstream returned nil error")
			return
		}
		msg := strings.ToLower(err.Error())
		if !strings.Contains(msg, "upstream") && !strings.Contains(msg, "no such remote") && !strings.Contains(msg, "remote") && !strings.Contains(msg, "pull") {
			t.Errorf("BUG-151: Pull no-upstream error %q is not actionable", err)
		}
	})

	t.Run("BUG-152-SubmoduleAddMaliciousURL", func(t *testing.T) {
		t.Log("BUG-152")
		skipIfNoGit(t)
		t.Setenv("GIT_TERMINAL_PROMPT", "0")
		t.Setenv("GIT_SSH_COMMAND", "ssh -o BatchMode=yes -o StrictHostKeyChecking=no -o ConnectTimeout=2")
		repoPath := initRepoWithCommit(t)

		// Sentinel file outside the repo. If shell expansion of `;rm -rf /.git`
		// occurred, we'd see catastrophic side effects.
		sentinelDir := t.TempDir()
		sentinelFile := filepath.Join(sentinelDir, "sentinel.txt")
		if err := os.WriteFile(sentinelFile, []byte("safe"), 0o644); err != nil {
			t.Fatalf("BUG-152: sentinel write: %v", err)
		}

		evilURL := "git@evil.com:repo;rm -rf /.git"
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = SubmoduleAdd(ctx, repoPath, evilURL, "remotes/evil")

		// 1. Sentinel still exists and is untouched.
		data, err := os.ReadFile(sentinelFile)
		if err != nil {
			t.Errorf("BUG-152: sentinel file gone — possible shell expansion! %v", err)
		} else if string(data) != "safe" {
			t.Errorf("BUG-152: sentinel content mutated: %q", data)
		}

		// 2. Root filesystem markers still exist (best-effort smoke check on POSIX).
		if runtime.GOOS != "windows" {
			for _, p := range []string{"/etc", "/usr"} {
				if _, err := os.Stat(p); err != nil {
					t.Errorf("BUG-152: %s missing — catastrophic shell expansion suspected: %v", p, err)
				}
			}
		}

		// 3. The repo's own .git directory still exists.
		if _, err := os.Stat(filepath.Join(repoPath, ".git")); err != nil {
			t.Errorf("BUG-152: repo .git removed — shell injection succeeded: %v", err)
		}
	})
}
