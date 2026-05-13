package repo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func skipIfNoGit(t *testing.T) {
	t.Helper()
	if !GitOnPath() {
		t.Skip("git not on PATH")
	}
}

func allowFileProtocol(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "protocol.file.allow")
	t.Setenv("GIT_CONFIG_VALUE_0", "always")
}

func mustGitConfig(t *testing.T, repoPath string) {
	t.Helper()
	for _, args := range [][]string{
		{"git", "-C", repoPath, "config", "user.email", "test@test.com"},
		{"git", "-C", repoPath, "config", "user.name", "Test"},
	} {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			t.Fatalf("git config: %v: %s", err, out)
		}
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, "git", "init", "-b", "main", dir)
	mustGitConfig(t, dir)
	return dir
}

func initRepoWithCommit(t *testing.T) string {
	t.Helper()
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "seed.txt"), []byte("seed"), 0o644); err != nil {
		t.Fatalf("write seed: %v", err)
	}
	run(t, "git", "-C", dir, "add", "seed.txt")
	run(t, "git", "-C", dir, "commit", "-m", "seed")
	return dir
}

func setupBareUpstream(t *testing.T) string {
	t.Helper()
	bare := t.TempDir()
	run(t, "git", "init", "--bare", "-b", "main", bare)
	wt := t.TempDir()
	run(t, "git", "clone", bare, wt)
	mustGitConfig(t, wt)
	if err := os.WriteFile(filepath.Join(wt, "file.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run(t, "git", "-C", wt, "add", "file.txt")
	run(t, "git", "-C", wt, "commit", "-m", "init")
	run(t, "git", "-C", wt, "push", "-u", "origin", "main")
	return bare
}

func TestIsGitRepo(t *testing.T) {
	skipIfNoGit(t)
	t.Run("true_for_git_dir", func(t *testing.T) {
		dir := initRepo(t)
		ok, err := IsGitRepo(dir)
		if err != nil {
			t.Fatalf("IsGitRepo: %v", err)
		}
		if !ok {
			t.Fatalf("IsGitRepo = false, want true")
		}
	})
	t.Run("false_for_empty_dir", func(t *testing.T) {
		dir := t.TempDir()
		ok, err := IsGitRepo(dir)
		if err != nil {
			t.Fatalf("IsGitRepo: %v", err)
		}
		if ok {
			t.Fatalf("IsGitRepo = true, want false")
		}
	})
}

func TestIsClean(t *testing.T) {
	skipIfNoGit(t)
	ctx := context.Background()
	t.Run("clean_after_commit", func(t *testing.T) {
		dir := initRepoWithCommit(t)
		clean, err := IsClean(ctx, dir)
		if err != nil {
			t.Fatalf("IsClean: %v", err)
		}
		if !clean {
			t.Fatalf("IsClean = false, want true")
		}
	})
	t.Run("dirty_after_write", func(t *testing.T) {
		dir := initRepoWithCommit(t)
		if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		clean, err := IsClean(ctx, dir)
		if err != nil {
			t.Fatalf("IsClean: %v", err)
		}
		if clean {
			t.Fatalf("IsClean = true, want false")
		}
	})
}

func TestHasUncommittedChanges(t *testing.T) {
	skipIfNoGit(t)
	ctx := context.Background()
	dir := initRepoWithCommit(t)
	got, err := HasUncommittedChanges(ctx, dir)
	if err != nil {
		t.Fatalf("HasUncommittedChanges: %v", err)
	}
	if got {
		t.Fatalf("HasUncommittedChanges = true, want false")
	}
	if err := os.WriteFile(filepath.Join(dir, "n.txt"), []byte("y"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err = HasUncommittedChanges(ctx, dir)
	if err != nil {
		t.Fatalf("HasUncommittedChanges: %v", err)
	}
	if !got {
		t.Fatalf("HasUncommittedChanges = false, want true")
	}
}

func TestCurrentBranch(t *testing.T) {
	skipIfNoGit(t)
	ctx := context.Background()
	dir := initRepoWithCommit(t)
	got, err := CurrentBranch(ctx, dir)
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	if got != "main" {
		t.Fatalf("CurrentBranch = %q, want %q", got, "main")
	}
}

func TestCommitPaths_HappyPath(t *testing.T) {
	skipIfNoGit(t)
	ctx := context.Background()
	dir := initRepoWithCommit(t)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("A"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := CommitPaths(ctx, dir, []string{"a.txt"}, "add a.txt"); err != nil {
		t.Fatalf("CommitPaths: %v", err)
	}
	out, err := exec.Command("git", "-C", dir, "log", "--oneline").CombinedOutput()
	if err != nil {
		t.Fatalf("git log: %v: %s", err, out)
	}
	if !strings.Contains(string(out), "add a.txt") {
		t.Fatalf("commit not in log: %s", out)
	}
	clean, err := IsClean(ctx, dir)
	if err != nil {
		t.Fatalf("IsClean: %v", err)
	}
	if !clean {
		t.Fatalf("expected clean tree after commit")
	}
}

func TestCommitPaths_RefusesEmpty(t *testing.T) {
	ctx := context.Background()
	err := CommitPaths(ctx, t.TempDir(), nil, "msg")
	if err == nil {
		t.Fatalf("CommitPaths(nil) = nil, want error")
	}
	err = CommitPaths(ctx, t.TempDir(), []string{}, "msg")
	if err == nil {
		t.Fatalf("CommitPaths([]) = nil, want error")
	}
}

func TestSubmoduleAdd(t *testing.T) {
	skipIfNoGit(t)
	allowFileProtocol(t)
	ctx := context.Background()
	parent := initRepoWithCommit(t)
	bare := setupBareUpstream(t)
	if err := SubmoduleAdd(ctx, parent, bare, "sub"); err != nil {
		t.Fatalf("SubmoduleAdd: %v", err)
	}
	subs, err := SubmoduleList(ctx, parent)
	if err != nil {
		t.Fatalf("SubmoduleList: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("len(subs) = %d, want 1", len(subs))
	}
	if subs[0].Path != "sub" {
		t.Fatalf("Path = %q, want %q", subs[0].Path, "sub")
	}
	if subs[0].Name != "sub" {
		t.Fatalf("Name = %q, want %q", subs[0].Name, "sub")
	}
	if subs[0].State != SubmoduleInitialized {
		t.Fatalf("State = %v, want SubmoduleInitialized", subs[0].State)
	}
	if subs[0].SHA == "" {
		t.Fatalf("SHA empty")
	}
}

func TestSubmoduleRemove(t *testing.T) {
	skipIfNoGit(t)
	allowFileProtocol(t)
	ctx := context.Background()
	parent := initRepoWithCommit(t)
	bare := setupBareUpstream(t)
	if err := SubmoduleAdd(ctx, parent, bare, "sub"); err != nil {
		t.Fatalf("SubmoduleAdd: %v", err)
	}
	if err := SubmoduleRemove(ctx, parent, "sub"); err != nil {
		t.Fatalf("SubmoduleRemove: %v", err)
	}
	subs, err := SubmoduleList(ctx, parent)
	if err != nil {
		t.Fatalf("SubmoduleList: %v", err)
	}
	if len(subs) != 0 {
		t.Fatalf("len(subs) = %d, want 0", len(subs))
	}
	if _, err := os.Stat(filepath.Join(parent, ".git", "modules", "sub")); !os.IsNotExist(err) {
		t.Fatalf(".git/modules/sub still exists: err=%v", err)
	}
}

func TestSubmoduleUpdate(t *testing.T) {
	skipIfNoGit(t)
	allowFileProtocol(t)
	ctx := context.Background()
	parent := initRepoWithCommit(t)
	bare := setupBareUpstream(t)
	if err := SubmoduleAdd(ctx, parent, bare, "sub"); err != nil {
		t.Fatalf("SubmoduleAdd: %v", err)
	}
	if err := SubmoduleUpdate(ctx, parent, ""); err != nil {
		t.Fatalf("SubmoduleUpdate(all): %v", err)
	}
	if err := SubmoduleUpdate(ctx, parent, "sub"); err != nil {
		t.Fatalf("SubmoduleUpdate(sub): %v", err)
	}
}

func TestSubmoduleList_States(t *testing.T) {
	skipIfNoGit(t)
	allowFileProtocol(t)
	ctx := context.Background()
	parent := initRepoWithCommit(t)
	bare := setupBareUpstream(t)
	if err := SubmoduleAdd(ctx, parent, bare, "sub"); err != nil {
		t.Fatalf("SubmoduleAdd: %v", err)
	}
	subs, err := SubmoduleList(ctx, parent)
	if err != nil {
		t.Fatalf("SubmoduleList: %v", err)
	}
	if len(subs) != 1 || subs[0].State != SubmoduleInitialized {
		t.Fatalf("expected one initialized submodule, got %+v", subs)
	}
	if err := os.RemoveAll(filepath.Join(parent, "sub")); err != nil {
		t.Fatalf("remove sub: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(parent, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	subs, err = SubmoduleList(ctx, parent)
	if err != nil {
		t.Fatalf("SubmoduleList: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("len = %d, want 1", len(subs))
	}
	if subs[0].State != SubmoduleNotInitialized {
		t.Fatalf("State = %v, want SubmoduleNotInitialized", subs[0].State)
	}
}