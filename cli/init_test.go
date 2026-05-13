package main

import (
	"bytes"
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

func makeBareRepo(t *testing.T, content string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	bare := filepath.Join(t.TempDir(), "bare.git")
	wt := t.TempDir()
	run := func(dir string, args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %s: %s", strings.Join(args, " "), string(out))
	}
	require.NoError(t, os.MkdirAll(bare, 0o755))
	run(bare, "init", "--bare", "-b", "main")
	run(wt, "init", "-b", "main")
	run(wt, "config", "user.email", "test@test.com")
	run(wt, "config", "user.name", "Test")
	require.NoError(t, os.WriteFile(filepath.Join(wt, "rice.toml"), []byte(content), 0o644))
	run(wt, "add", ".")
	run(wt, "commit", "-m", "init")
	run(wt, "remote", "add", "origin", bare)
	run(wt, "push", "origin", "main")
	return bare
}

func runRootCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetIn(strings.NewReader(""))
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	rootCmd.SetIn(os.Stdin)
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	return buf.String(), err
}

func setIsolatedHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("AppData", filepath.Join(home, "AppData"))
	return home
}

func TestInit_ClonesRepo(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)

	bareURL := makeBareRepo(t, "schema_version = 1\n\n[packages.demo]\nsupported_os = [\"linux\", \"darwin\"]\n\n[packages.demo.profiles.common]\nsources = [{path = \"x\", mode = \"file\", target = \"$HOME\"}]\n")

	out, err := runRootCmd(t, "init", bareURL)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Cloned rice repo to")

	dest := repo.DefaultRepoPath()
	_, err = os.Stat(filepath.Join(dest, "rice.toml"))
	require.NoError(t, err, "rice.toml should exist in cloned repo")
}

func TestInit_AlreadyInitialized(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)

	dest := repo.DefaultRepoPath()
	require.NoError(t, os.MkdirAll(dest, 0o755))

	out, err := runRootCmd(t, "init", "https://example.invalid/repo.git")
	require.Error(t, err, "out=%s", out)
	assert.Contains(t, err.Error(), "already initialized")

	assert.False(t, errors.Is(err, repo.ErrRepoNotInitialized))
}

func TestInit_AlreadyExists(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)

	dest := repo.DefaultRepoPath()
	require.NoError(t, os.MkdirAll(dest, 0o755))

	out, err := runRootCmd(t, "init", "https://example.invalid/repo.git")
	require.Error(t, err, "out=%s", out)
	msg := strings.ToLower(err.Error())
	assert.True(t, strings.Contains(msg, "already") || strings.Contains(msg, "exists"),
		"expected error to mention already/exists; got: %s", err.Error())
}

func TestInit_GitNotOnPath(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)

	emptyDir := t.TempDir()
	t.Setenv("PATH", emptyDir)

	out, err := runRootCmd(t, "init", "https://example.invalid/repo.git")
	require.Error(t, err, "out=%s", out)
}

func TestInit_MissingURLArg(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)

	out, err := runRootCmd(t, "init")
	require.Error(t, err, "out=%s", out)
}

func TestInit_SuccessWithFakeGit(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)

	binDir := t.TempDir()
	gitPath := filepath.Join(binDir, "git")
	script := "#!/bin/sh\nif [ \"$1\" = \"clone\" ]; then\n  /bin/mkdir -p \"$3\"\n  exit 0\nfi\nexit 0\n"
	require.NoError(t, os.WriteFile(gitPath, []byte(script), 0o755))
	t.Setenv("PATH", binDir)

	out, err := runRootCmd(t, "init", "https://example.invalid/repo.git")
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Cloned rice repo to")

	dest := repo.DefaultRepoPath()
	_, err = os.Stat(dest)
	require.NoError(t, err, "dest dir should exist after fake git clone")
}

func TestInit_ExistsStatError(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)

	dest := repo.DefaultRepoPath()
	parent := filepath.Dir(dest)
	require.NoError(t, os.MkdirAll(parent, 0o755))
	require.NoError(t, os.MkdirAll(dest, 0o755))
	require.NoError(t, os.Chmod(parent, 0o000))
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })

	out, err := runRootCmd(t, "init", "https://example.invalid/repo.git")
	require.Error(t, err, "out=%s", out)
}
