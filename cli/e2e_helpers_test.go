//go:build !windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/state"
)

// writeManifest writes tomlBody to filepath.Join(repoRoot, "rice.toml").
// Creates parent directories if needed.
func writeManifest(t *testing.T, repoRoot, tomlBody string) {
	t.Helper()
	full := filepath.Join(repoRoot, "rice.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(tomlBody), 0o644))
}

// writeSourceFile writes content to filepath.Join(repoRoot, relPath).
// Creates parent directories if needed.
func writeSourceFile(t *testing.T, repoRoot, relPath, content string) {
	t.Helper()
	full := filepath.Join(repoRoot, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
}

// assertSymlinkPointsTo asserts target exists, is a symlink, and os.Readlink(target) == expectedSource.
func assertSymlinkPointsTo(t *testing.T, target, expectedSource string) {
	t.Helper()
	dest, err := os.Readlink(target)
	require.NoError(t, err)
	assert.Equal(t, expectedSource, dest)
}

// assertNotSymlink asserts path either does not exist OR is not a symlink.
func assertNotSymlink(t *testing.T, path string) {
	t.Helper()
	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return // path doesn't exist, which is fine
		}
		require.NoError(t, err)
	}
	// path exists; check it's not a symlink
	assert.Equal(t, os.FileMode(0), fi.Mode()&os.ModeSymlink)
}

// assertNoSymlinkAt asserts path does not exist at all.
func assertNoSymlinkAt(t *testing.T, path string) {
	t.Helper()
	_, err := os.Lstat(path)
	require.True(t, os.IsNotExist(err), "expected path to not exist: %s", path)
}

// loadE2EState calls state.Load(statePath), require.NoError, returns the map.
func loadE2EState(t *testing.T, statePath string) state.State {
	t.Helper()
	s, err := state.Load(statePath)
	require.NoError(t, err)
	return s
}

// assertStateHasPackage loads state, asserts pkg key exists, profile matches, len(InstalledLinks) == expectedLinkCount.
func assertStateHasPackage(t *testing.T, statePath, pkg, expectedProfile string, expectedLinkCount int) {
	t.Helper()
	s := loadE2EState(t, statePath)
	pkgState, ok := s[pkg]
	require.True(t, ok, "package %q not found in state", pkg)
	assert.Equal(t, expectedProfile, pkgState.Profile)
	assert.Equal(t, expectedLinkCount, len(pkgState.InstalledLinks))
}

// assertStateMissingPackage loads state (if file exists), asserts pkg key is absent.
// If state file doesn't exist, that also satisfies "missing".
func assertStateMissingPackage(t *testing.T, statePath, pkg string) {
	t.Helper()
	s := loadE2EState(t, statePath)
	_, ok := s[pkg]
	assert.False(t, ok, "package %q should not be in state", pkg)
}

// replaceSymlinkWithFile removes path, then writes content to it.
func replaceSymlinkWithFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.Remove(path))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// replaceSymlinkWithDir removes path, then creates it as a directory.
func replaceSymlinkWithDir(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.Remove(path))
	require.NoError(t, os.MkdirAll(path, 0o755))
}

// replaceSymlinkWithSymlink removes path, then creates a new symlink to newTarget.
func replaceSymlinkWithSymlink(t *testing.T, path, newTarget string) {
	t.Helper()
	require.NoError(t, os.Remove(path))
	require.NoError(t, os.Symlink(newTarget, path))
}

// manuallyDeleteSymlink removes path.
func manuallyDeleteSymlink(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.Remove(path))
}

// breakSymlink removes the source file at filepath.Join(repoRoot, relSourcePath),
// leaving the symlink dangling.
func breakSymlink(t *testing.T, repoRoot, relSourcePath string) {
	t.Helper()
	full := filepath.Join(repoRoot, relSourcePath)
	require.NoError(t, os.Remove(full))
}

// corruptStateFile writes invalid JSON to statePath.
func corruptStateFile(t *testing.T, statePath string) {
	t.Helper()
	require.NoError(t, os.WriteFile(statePath, []byte("{not valid json"), 0o644))
}

// writeStaleState constructs a state.State with the given package and writes it to statePath.
func writeStaleState(t *testing.T, statePath, pkg, profile string, links []state.InstalledLink) {
	t.Helper()
	s := state.State{
		pkg: state.PackageState{
			Profile:        profile,
			InstalledLinks: links,
		},
	}
	require.NoError(t, state.Save(statePath, s))
}

// initGitRepo initializes a git repo at repoRoot with rice.toml committed.
// Skips if git not on PATH.
func initGitRepo(t *testing.T, repoRoot string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git not on PATH")
	}

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoRoot
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %s: %s", strings.Join(args, " "), string(out))
	}

	run("init", "-b", "main")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
	run("add", "rice.toml")
	run("commit", "-m", "init")
}

// setupRemoteSubmodule creates a bare upstream repo from remoteFiles, adds it as a submodule,
// and commits the result. Returns the path to the submodule directory.
// Skips if git not on PATH.
func setupRemoteSubmodule(t *testing.T, repoRoot, remoteName string, remoteFiles map[string]string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git not on PATH")
	}

	bareURL := setupBareUpstreamFromTree(t, remoteName, remoteFiles)

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoRoot
		cmd.Env = append(os.Environ(),
			"GIT_CONFIG_COUNT=1",
			"GIT_CONFIG_KEY_0=protocol.file.allow",
			"GIT_CONFIG_VALUE_0=always",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %s: %s", strings.Join(args, " "), string(out))
	}

	run("submodule", "add", bareURL, "remotes/"+remoteName)
	run("commit", "-m", "add remote "+remoteName)

	return filepath.Join(repoRoot, "remotes", remoteName)
}
