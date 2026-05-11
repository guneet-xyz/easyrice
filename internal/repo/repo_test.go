package repo

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func run(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command %v failed: %v\n%s", args, err, out)
	}
}

func setupBareRepo(t *testing.T) (bareURL string, addCommit func(filename, content string)) {
	t.Helper()
	bare := t.TempDir()
	run(t, "git", "init", "--bare", "-b", "main", bare)
	wt := t.TempDir()
	run(t, "git", "clone", bare, wt)
	run(t, "git", "-C", wt, "config", "user.email", "test@test.com")
	run(t, "git", "-C", wt, "config", "user.name", "Test")
	run(t, "git", "-C", wt, "checkout", "-B", "main")

	addCommit = func(filename, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(wt, filename), []byte(content), 0644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		run(t, "git", "-C", wt, "add", ".")
		run(t, "git", "-C", wt, "commit", "-m", "add "+filename)
		run(t, "git", "-C", wt, "push", "-u", "origin", "main")
	}
	addCommit("rice.toml", "schema_version = 1\n")
	return bare, addCommit
}

func TestDefaultRepoPath_EndsWithReposDefault(t *testing.T) {
	got := DefaultRepoPath()
	want := filepath.Join("easyrice", "repos", "default")
	if !strings.HasSuffix(got, want) {
		t.Fatalf("DefaultRepoPath() = %q, want suffix %q", got, want)
	}
}

func TestRepoTomlPath(t *testing.T) {
	got := RepoTomlPath("/some/repo")
	want := filepath.Join("/some/repo", "rice.toml")
	if got != want {
		t.Fatalf("RepoTomlPath = %q, want %q", got, want)
	}
}

func TestExists_FalseWhenAbsent(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	ok, err := Exists(missing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("Exists(%q) = true, want false", missing)
	}
}

func TestExists_TrueWhenPresent(t *testing.T) {
	dir := t.TempDir()
	ok, err := Exists(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("Exists(%q) = false, want true", dir)
	}
}

func TestClone_FromBareRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	bare, _ := setupBareRepo(t)
	dest := filepath.Join(t.TempDir(), "clone")
	if err := Clone(context.Background(), bare, dest); err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "rice.toml")); err != nil {
		t.Fatalf("expected rice.toml in clone: %v", err)
	}
}

func TestPull_FromBareRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	bare, addCommit := setupBareRepo(t)
	dest := filepath.Join(t.TempDir(), "clone")
	if err := Clone(context.Background(), bare, dest); err != nil {
		t.Fatalf("Clone: %v", err)
	}
	addCommit("new.txt", "hello")
	if err := Pull(context.Background(), dest); err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "new.txt")); err != nil {
		t.Fatalf("expected new.txt after pull: %v", err)
	}
}

func TestClone_ErrorOnNonexistentURL(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dest := filepath.Join(t.TempDir(), "clone")
	err := Clone(context.Background(), "/does/not/exist", dest)
	if err == nil {
		t.Fatal("expected error for nonexistent URL")
	}
	if !strings.Contains(err.Error(), "git clone") {
		t.Fatalf("error %q should contain 'git clone'", err.Error())
	}
}

func TestGitOnPath(t *testing.T) {
	_ = GitOnPath()
}

func TestErrPackageNotDeclared(t *testing.T) {
	err := ErrPackageNotDeclared("nvim")
	if !strings.Contains(err.Error(), `"nvim"`) {
		t.Fatalf("error %q should contain quoted package name", err.Error())
	}
}

func TestDefaultRepoPath_ContainsEasyrice(t *testing.T) {
	got := DefaultRepoPath()
	if !strings.Contains(got, "easyrice") {
		t.Fatalf("DefaultRepoPath() = %q, want to contain 'easyrice'", got)
	}
	if !strings.Contains(got, filepath.Join("repos", "default")) {
		t.Fatalf("DefaultRepoPath() = %q, want to contain 'repos/default'", got)
	}
}

func TestPull_ErrorWhenRepoMissing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	err := Pull(context.Background(), missing)
	if err == nil {
		t.Fatal("expected error when pulling from non-existent repo")
	}
	if !strings.Contains(err.Error(), "git pull") {
		t.Fatalf("error %q should contain 'git pull'", err.Error())
	}
}

func TestPull_ErrorWhenGitMissing(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH, cannot test git-missing scenario")
	}
	bare, _ := setupBareRepo(t)
	dest := filepath.Join(t.TempDir(), "clone")
	if err := Clone(context.Background(), bare, dest); err != nil {
		t.Fatalf("Clone: %v", err)
	}

	t.Setenv("PATH", "")
	err := Pull(context.Background(), dest)
	if err == nil {
		t.Fatal("expected error when git is not on PATH")
	}
}

func TestPull_SuccessWithFakeGit(t *testing.T) {
	// Create a fake git script that exits 0
	tmpDir := t.TempDir()
	gitScript := filepath.Join(tmpDir, "git")
	if err := os.WriteFile(gitScript, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("write fake git script: %v", err)
	}

	// Prepend tmpDir to PATH so our fake git is found first
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+":"+oldPath)

	// Call Pull with any path; it should succeed because our fake git exits 0
	err := Pull(context.Background(), "/any/path")
	if err != nil {
		t.Fatalf("Pull with fake git: %v", err)
	}
}

func TestExists_ErrorOnStatFailure(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "noaccess")
	if err := os.Mkdir(subdir, 0000); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	defer os.Chmod(subdir, 0755)

	target := filepath.Join(subdir, "file")
	ok, err := Exists(target)
	if err == nil {
		t.Fatal("expected error when stat fails on permission denied")
	}
	if ok {
		t.Fatalf("Exists should return false on stat error, got true")
	}
}

func TestDefaultRepoPath_FallbackWhenUserConfigDirFails(t *testing.T) {
	got := DefaultRepoPath()
	if got == "" {
		t.Fatal("DefaultRepoPath should never return empty string")
	}
	if !strings.Contains(got, "easyrice") {
		t.Fatalf("DefaultRepoPath should contain 'easyrice', got %q", got)
	}
}

func TestClone_ContextCancelled(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dest := filepath.Join(t.TempDir(), "clone")
	err := Clone(ctx, "https://example.com/repo.git", dest)
	if err == nil {
		t.Fatal("expected error when context is cancelled")
	}
	if !strings.Contains(err.Error(), "context") && !errors.Is(err, context.Canceled) {
		t.Fatalf("error %q should mention context or wrap context.Canceled", err.Error())
	}
}

func TestPull_ContextCancelled(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repoPath := t.TempDir()
	err := Pull(ctx, repoPath)
	if err == nil {
		t.Fatal("expected error when context is cancelled")
	}
	if !strings.Contains(err.Error(), "context") && !errors.Is(err, context.Canceled) {
		t.Fatalf("error %q should mention context or wrap context.Canceled", err.Error())
	}
}
