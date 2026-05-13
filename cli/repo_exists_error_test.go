package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/guneet-xyz/easyrice/internal/xdgpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setRepoParentAsFile makes <XDG_CONFIG_HOME>/easyrice/repos a regular file so
// repo.Exists() stats the would-be repo path and gets "not a directory" — a
// non-NotExist error. The logs subtree is left as a real directory so logger
// initialization succeeds.
func setRepoParentAsFile(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	t.Setenv("AppData", filepath.Join(tmp, "AppData"))
	cfg := xdgpath.ConfigDir()
	require.NoError(t, os.MkdirAll(filepath.Join(cfg, "easyrice"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(cfg, "easyrice", "repos"), []byte("x"), 0o644))
}

// TestInstall_RepoExistsError exercises cli/install.go:47-49 — the
// "check repo: %w" branch when repo.Exists returns a non-NotExist stat error.
func TestInstall_RepoExistsError(t *testing.T) {
	resetInstallFlags()
	setRepoParentAsFile(t)

	out, err := runInstallCmd(t, "", "install", "mypkg")
	require.Error(t, err, "out=%s", out)
	assert.Contains(t, err.Error(), "check repo")
}

// TestUpdate_RepoExistsError exercises cli/update.go:25-27.
func TestUpdate_RepoExistsError(t *testing.T) {
	resetInstallFlags()
	setRepoParentAsFile(t)

	_, err := runInstallCmd(t, "", "update")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check repo")
}

// TestInit_RepoExistsError exercises cli/init.go:33-35.
func TestInit_RepoExistsError(t *testing.T) {
	resetInstallFlags()
	setRepoParentAsFile(t)

	_, err := runInstallCmd(t, "", "init", "https://example.invalid/repo.git")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check repo")
}

// TestStatus_RepoExistsError exercises cli/status.go:79-81 in
// showDeclaredDependencies (called via `status <pkg>` when the package is in state).
func TestStatus_RepoExistsError(t *testing.T) {
	resetInstallFlags()
	setRepoParentAsFile(t)

	statePath := filepath.Join(t.TempDir(), "state.json")
	require.NoError(t, os.WriteFile(statePath, []byte(`{"mypkg":{"profile":"common","installed_links":[],"installed_at":"2025-01-01T00:00:00Z"}}`), 0o644))

	out, err := runInstallCmd(t, "", "--state", statePath, "status", "mypkg")
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "could not check declared dependencies")
	assert.Contains(t, out, "check repo")
}

// TestDoctor_StateLoadError exercises cli/doctor.go:51-55 — the branch printing
// the state-load error and continuing with an empty state.
func TestDoctor_StateLoadError(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(t.TempDir(), "x"), 0o755))

	statePath := filepath.Join(t.TempDir(), "state.json")
	require.NoError(t, os.WriteFile(statePath, []byte("not-json"), 0o644))

	out, err := runInstallCmd(t, "", "--state", statePath, "doctor")
	require.Error(t, err, "out=%s", out)
	assert.Contains(t, out, "State: could not read")
}

// TestInstall_ExecutePlanError exercises the converge-plan error path when
// state.Load fails inside BuildConvergePlan due to a corrupt state file.
func TestInstall_ExecutePlanError(t *testing.T) {
	resetInstallFlags()
	_, _, _ = setupTestRepo(t)

	statePath := filepath.Join(t.TempDir(), "state.json")
	require.NoError(t, os.WriteFile(statePath, []byte("{not json"), 0o644))

	_, err := runInstallCmd(t, "",
		"--state", statePath,
		"--yes",
		"install", "mypkg",
		"--profile", "common",
		"--skip-deps",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load state")
}
