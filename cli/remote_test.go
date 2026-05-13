package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/repo"
)

func gitOnPath(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
}

func gitRunRemote(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=protocol.file.allow",
		"GIT_CONFIG_VALUE_0=always",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %s: %s", strings.Join(args, " "), string(out))
}

func makeBareUpstreamRice(t *testing.T) string {
	t.Helper()
	gitOnPath(t)

	bare := filepath.Join(t.TempDir(), "upstream.git")
	require.NoError(t, os.MkdirAll(bare, 0o755))
	gitRunRemote(t, filepath.Dir(bare), "init", "--bare", "-b", "main", filepath.Base(bare))

	wt := t.TempDir()
	gitRunRemote(t, wt, "init", "-b", "main")
	gitRunRemote(t, wt, "config", "user.email", "test@test.com")
	gitRunRemote(t, wt, "config", "user.name", "Test")

	manifestContent := `schema_version = 1

[packages.upstreamtool]
description = "upstream package"
supported_os = ["linux", "darwin", "windows"]

[packages.upstreamtool.profiles.common]
sources = [{path = "common", mode = "file", target = "$HOME"}]
`
	require.NoError(t, os.WriteFile(filepath.Join(wt, "rice.toml"), []byte(manifestContent), 0o644))
	gitRunRemote(t, wt, "add", "rice.toml")
	gitRunRemote(t, wt, "commit", "-m", "init upstream")
	gitRunRemote(t, wt, "remote", "add", "origin", bare)
	gitRunRemote(t, wt, "push", "origin", "main")

	return bare
}

func makeManagedRepo(t *testing.T, manifestContent string) string {
	t.Helper()
	gitOnPath(t)

	root := repo.DefaultRepoPath()
	require.NoError(t, os.MkdirAll(root, 0o755))
	gitRunRemote(t, root, "init", "-b", "main")
	gitRunRemote(t, root, "config", "user.email", "test@test.com")
	gitRunRemote(t, root, "config", "user.name", "Test")

	if manifestContent == "" {
		manifestContent = `schema_version = 1
`
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, "rice.toml"), []byte(manifestContent), 0o644))
	gitRunRemote(t, root, "add", "rice.toml")
	gitRunRemote(t, root, "commit", "-m", "init managed")

	return root
}

func resetRemoteFlags() {
	remoteAddNameFlag = ""
}

func runRemoteCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetInstallFlags()
	resetRemoteFlags()
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "protocol.file.allow")
	t.Setenv("GIT_CONFIG_VALUE_0", "always")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@test.com")
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@test.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	return runRootCmd(t, args...)
}

func TestRemoteAdd_Happy(t *testing.T) {
	setIsolatedHome(t)
	root := makeManagedRepo(t, "")
	upstream := makeBareUpstreamRice(t)

	out, err := runRemoteCmd(t, "remote", "add", upstream, "--name", "myremote")
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Added remote")

	_, err = os.Stat(filepath.Join(root, ".gitmodules"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(root, "remotes", "myremote", "rice.toml"))
	require.NoError(t, err)

	logCmd := exec.Command("git", "-C", root, "log", "--oneline")
	logOut, err := logCmd.CombinedOutput()
	require.NoError(t, err)
	assert.Contains(t, string(logOut), "Add remote rice myremote")
}

func TestRemoteAdd_DirtyBlocks(t *testing.T) {
	setIsolatedHome(t)
	root := makeManagedRepo(t, "")
	upstream := makeBareUpstreamRice(t)

	require.NoError(t, os.WriteFile(filepath.Join(root, "dirty.txt"), []byte("x"), 0o644))

	out, err := runRemoteCmd(t, "remote", "add", upstream, "--name", "r1")
	require.Error(t, err, "out=%s", out)
	assert.ErrorIs(t, err, repo.ErrRepoDirty)
}

func TestRemoteAdd_InvalidName(t *testing.T) {
	setIsolatedHome(t)
	makeManagedRepo(t, "")
	upstream := makeBareUpstreamRice(t)

	out, err := runRemoteCmd(t, "remote", "add", upstream, "--name", "bad/name")
	require.Error(t, err, "out=%s", out)
	assert.Contains(t, err.Error(), "invalid")
}

func TestRemoteAdd_AlreadyExists(t *testing.T) {
	setIsolatedHome(t)
	makeManagedRepo(t, "")
	upstream := makeBareUpstreamRice(t)

	_, err := runRemoteCmd(t, "remote", "add", upstream, "--name", "dup")
	require.NoError(t, err)

	out, err := runRemoteCmd(t, "remote", "add", upstream, "--name", "dup")
	require.Error(t, err, "out=%s", out)
	assert.ErrorIs(t, err, repo.ErrRemoteAlreadyExists)
}

func TestRemoteRemove_BlockedByImport(t *testing.T) {
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

	out, err := runRemoteCmd(t, "remote", "remove", "used")
	require.Error(t, err, "out=%s", out)
	assert.ErrorIs(t, err, repo.ErrRemoteInUse)
	assert.Contains(t, err.Error(), "local.common")
}

func TestRemoteRemove_Happy(t *testing.T) {
	setIsolatedHome(t)
	root := makeManagedRepo(t, "")
	upstream := makeBareUpstreamRice(t)

	_, err := runRemoteCmd(t, "remote", "add", upstream, "--name", "togo")
	require.NoError(t, err)

	out, err := runRemoteCmd(t, "remote", "remove", "togo")
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Removed remote")

	_, err = os.Stat(filepath.Join(root, "remotes", "togo"))
	assert.True(t, os.IsNotExist(err))
}

func TestRemoteList_Empty(t *testing.T) {
	setIsolatedHome(t)
	makeManagedRepo(t, "")

	out, err := runRemoteCmd(t, "remote", "list")
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "No remotes configured.")
}

func TestRemoteList_WithRemote(t *testing.T) {
	setIsolatedHome(t)
	makeManagedRepo(t, "")
	upstream := makeBareUpstreamRice(t)

	_, err := runRemoteCmd(t, "remote", "add", upstream, "--name", "listed")
	require.NoError(t, err)

	out, err := runRemoteCmd(t, "remote", "list")
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "listed")
	assert.Contains(t, out, "remotes/listed")
}
