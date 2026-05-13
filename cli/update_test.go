package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/repo"
)

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %s in %s: %s", strings.Join(args, " "), dir, string(out))
}

func TestUpdate_PullsNewCommit(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)

	bareURL := makeBareRepo(t, "schema_version = 1\n\n[packages.demo]\nsupported_os = [\"linux\", \"darwin\"]\n\n[packages.demo.profiles.common]\nsources = [{path = \"x\", mode = \"file\", target = \"$HOME\"}]\n")

	out, err := runRootCmd(t, "init", bareURL)
	require.NoError(t, err, "init out=%s", out)

	dest := repo.DefaultRepoPath()

	wt := t.TempDir()
	gitRun(t, wt, "clone", bareURL, wt)
	gitRun(t, wt, "config", "user.email", "test@test.com")
	gitRun(t, wt, "config", "user.name", "Test")
	require.NoError(t, os.WriteFile(filepath.Join(wt, "newfile.txt"), []byte("hi\n"), 0o644))
	gitRun(t, wt, "add", ".")
	gitRun(t, wt, "commit", "-m", "add newfile")
	gitRun(t, wt, "push", "origin", "main")

	out, err = runRootCmd(t, "update")
	require.NoError(t, err, "update out=%s", out)
	assert.Contains(t, out, "Updated rice repo from origin.")

	_, err = os.Stat(filepath.Join(dest, "newfile.txt"))
	require.NoError(t, err, "newfile.txt should be present after pull")
}

func TestUpdate_NotInitialized(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)

	out, err := runRootCmd(t, "update")
	require.Error(t, err, "out=%s", out)
	assert.True(t, errors.Is(err, repo.ErrRepoNotInitialized), "err=%v", err)
}

func TestUpdate_GitNotOnPath(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)

	// Create repos/default/ so Exists() passes, but git will fail
	repoPath := repo.DefaultRepoPath()
	require.NoError(t, os.MkdirAll(repoPath, 0o755))

	// Set PATH to empty directory so git is not found
	emptyDir := t.TempDir()
	t.Setenv("PATH", emptyDir)

	out, err := runRootCmd(t, "update")
	require.Error(t, err, "update should fail when git not on PATH")
	assert.Contains(t, err.Error(), "pull:", "error should mention pull operation")
	assert.Contains(t, out, "executable file not found", "output should mention git not found")
}

func TestUpdate_SuccessWithFakeGit(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)

	// Create repos/default/ so Exists() passes
	repoPath := repo.DefaultRepoPath()
	require.NoError(t, os.MkdirAll(repoPath, 0o755))

	// Create a fake git script that exits 0
	fakeGitDir := t.TempDir()
	fakeGitPath := filepath.Join(fakeGitDir, "git")
	require.NoError(t, os.WriteFile(fakeGitPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))

	// Prepend fake git dir to PATH
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", fakeGitDir+":"+oldPath)

	out, err := runRootCmd(t, "update")
	require.NoError(t, err, "update should succeed with fake git: %s", out)
	assert.Contains(t, out, "Updated rice repo from origin.")
}
