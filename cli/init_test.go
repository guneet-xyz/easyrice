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
	assert.Contains(t, out, "Cloned to")

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
