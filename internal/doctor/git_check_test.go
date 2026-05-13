package doctor

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/manifest"
)

func TestCheckGitOnPath(t *testing.T) {
	if err := CheckGitOnPath(); err != nil {
		t.Fatalf("expected git on PATH, got error: %v", err)
	}
}

func TestCheckGitOnPath_GitPresent(t *testing.T) {
	tmpDir := t.TempDir()
	gitPath := filepath.Join(tmpDir, "git")

	err := os.WriteFile(gitPath, []byte("#!/bin/sh\nexit 0\n"), 0o755)
	if err != nil {
		t.Fatalf("failed to write fake git script: %v", err)
	}

	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := CheckGitOnPath(); err != nil {
		t.Errorf("expected no error with git on PATH, got: %v", err)
	}
}

func TestCheckGitOnPath_GitMissing(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PATH", tmpDir)

	err := CheckGitOnPath()
	if err == nil {
		t.Error("expected error when git is not on PATH, got nil")
	}
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, string(out))
}

func newTestRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	gitRun(t, dir, "init", "-b", "main")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")
	gitRun(t, dir, "config", "commit.gpgsign", "false")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README"), []byte("hi"), 0o644))
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "init")
	return dir
}

func TestCheckRepoClean_Clean(t *testing.T) {
	repoDir := newTestRepo(t)
	var buf bytes.Buffer

	issues := CheckRepoClean(context.Background(), &buf, repoDir)

	assert.Equal(t, 0, issues)
	assert.Empty(t, buf.String())
}

func TestCheckRepoClean_Dirty(t *testing.T) {
	repoDir := newTestRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "dirty.txt"), []byte("x"), 0o644))

	var buf bytes.Buffer
	issues := CheckRepoClean(context.Background(), &buf, repoDir)

	assert.Equal(t, 0, issues, "dirty repo is a warning, not an error")
	assert.Contains(t, buf.String(), "[WARN]")
	assert.Contains(t, buf.String(), "uncommitted changes")
}

func makeBareRemote(t *testing.T) string {
	t.Helper()
	bare := filepath.Join(t.TempDir(), "remote.git")
	require.NoError(t, os.MkdirAll(bare, 0o755))
	gitRun(t, bare, "init", "--bare", "-b", "main")

	wt := t.TempDir()
	gitRun(t, wt, "init", "-b", "main")
	gitRun(t, wt, "config", "user.email", "t@t.com")
	gitRun(t, wt, "config", "user.name", "T")
	gitRun(t, wt, "config", "commit.gpgsign", "false")
	require.NoError(t, os.WriteFile(filepath.Join(wt, "rice.toml"), []byte("schema_version = 1\n"), 0o644))
	gitRun(t, wt, "add", ".")
	gitRun(t, wt, "commit", "-m", "init")
	gitRun(t, wt, "remote", "add", "origin", bare)
	gitRun(t, wt, "push", "origin", "main")
	return bare
}

func TestCheckSubmodules_NotInitialized(t *testing.T) {
	repoDir := newTestRepo(t)
	bare := makeBareRemote(t)

	gitRun(t, repoDir, "-c", "protocol.file.allow=always", "submodule", "add", "--", bare, "remotes/sub")
	gitRun(t, repoDir, "commit", "-m", "add submodule")
	gitRun(t, repoDir, "submodule", "deinit", "-f", "--", "remotes/sub")

	var buf bytes.Buffer
	issues := CheckSubmodules(context.Background(), &buf, repoDir)

	assert.Equal(t, 1, issues)
	assert.Contains(t, buf.String(), "[ERROR]")
	assert.Contains(t, buf.String(), "Remote sub is not initialized")
	assert.Contains(t, buf.String(), "rice remote update sub")
}

func TestCheckSubmodules_Modified(t *testing.T) {
	repoDir := newTestRepo(t)
	bare := makeBareRemote(t)

	gitRun(t, repoDir, "-c", "protocol.file.allow=always", "submodule", "add", "--", bare, "remotes/sub")
	gitRun(t, repoDir, "commit", "-m", "add submodule")

	subPath := filepath.Join(repoDir, "remotes", "sub")
	require.NoError(t, os.WriteFile(filepath.Join(subPath, "extra.txt"), []byte("local"), 0o644))
	gitRun(t, subPath, "config", "user.email", "t@t.com")
	gitRun(t, subPath, "config", "user.name", "T")
	gitRun(t, subPath, "config", "commit.gpgsign", "false")
	gitRun(t, subPath, "add", ".")
	gitRun(t, subPath, "commit", "-m", "local change")

	var buf bytes.Buffer
	issues := CheckSubmodules(context.Background(), &buf, repoDir)

	assert.Equal(t, 0, issues, "modified submodule is a warning, not an error")
	assert.Contains(t, buf.String(), "[WARN]")
	assert.Contains(t, buf.String(), "Remote sub has local changes")
}

func TestCheckDanglingImports_Missing(t *testing.T) {
	repoDir := t.TempDir()
	mf := manifest.Manifest{
		Packages: map[string]manifest.PackageDef{
			"nvim": {
				Profiles: map[string]manifest.ProfileDef{
					"default": {
						Import: "remotes/kick#nvim.default",
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	issues := CheckDanglingImports(&buf, repoDir, mf)

	assert.Equal(t, 1, issues)
	assert.Contains(t, buf.String(), "[ERROR]")
	assert.Contains(t, buf.String(), "nvim.default")
	assert.Contains(t, buf.String(), "remotes/kick")
}

func TestCheckDanglingImports_OK(t *testing.T) {
	repoDir := t.TempDir()
	remoteToml := filepath.Join(repoDir, "remotes", "kick", "rice.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(remoteToml), 0o755))
	require.NoError(t, os.WriteFile(remoteToml, []byte("schema_version = 1\n"), 0o644))

	mf := manifest.Manifest{
		Packages: map[string]manifest.PackageDef{
			"nvim": {
				Profiles: map[string]manifest.ProfileDef{
					"default": {
						Import: "remotes/kick#nvim.default",
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	issues := CheckDanglingImports(&buf, repoDir, mf)

	assert.Equal(t, 0, issues)
	assert.Empty(t, buf.String())
}
