package repo

import (
	"context"
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
