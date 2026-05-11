package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/repo"
	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/guneet-xyz/easyrice/internal/symlink"
)

func setupIntegrationRepo(t *testing.T) (repoRoot, statePath, homeDir string) {
	t.Helper()
	homeDir = setIsolatedHome(t)
	repoRoot = repo.DefaultRepoPath()
	statePath = filepath.Join(t.TempDir(), "state.json")

	pkgDir := filepath.Join(repoRoot, "demo")
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "common"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(pkgDir, "work"), 0o755))

	require.NoError(t, os.WriteFile(
		filepath.Join(pkgDir, "common", "dotfile1"),
		[]byte("common-content\n"), 0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(pkgDir, "work", "dotfile2"),
		[]byte("work-content\n"), 0o644))

	manifest := `schema_version = 1

[packages.demo]
description = "Demo package for integration test"
supported_os = ["linux", "darwin", "windows"]

[packages.demo.profiles.common]
sources = [{path = "common", mode = "file", target = "$HOME/.config/demo"}]

[packages.demo.profiles.work]
sources = [{path = "work", mode = "file", target = "$HOME/.config/demo"}]
`
	require.NoError(t, os.WriteFile(
		filepath.Join(repoRoot, "rice.toml"), []byte(manifest), 0o644))

	return
}

func dirSnapshot(t *testing.T, path string) string {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		require.NoError(t, err)
	}
	return fi.ModTime().String() + "|" + fi.Mode().String()
}

func TestIntegration_InstallStatusSwitchUninstall(t *testing.T) {
	realConfigDir, _ := os.UserConfigDir()
	realEasyriceDir := ""
	realEasyriceSnap := ""
	if realConfigDir != "" {
		realEasyriceDir = filepath.Join(realConfigDir, "easyrice")
		realEasyriceSnap = dirSnapshot(t, realEasyriceDir)
	}

	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	repoRoot, statePath, homeDir := setupIntegrationRepo(t)

	srcCommon := filepath.Join(repoRoot, "demo", "common", "dotfile1")
	srcWork := filepath.Join(repoRoot, "demo", "work", "dotfile2")
	tgtCommon := filepath.Join(homeDir, ".config", "demo", "dotfile1")
	tgtWork := filepath.Join(homeDir, ".config", "demo", "dotfile2")

	commonContent, err := os.ReadFile(srcCommon)
	require.NoError(t, err)
	workContent, err := os.ReadFile(srcWork)
	require.NoError(t, err)

	t.Run("install_common", func(t *testing.T) {
		resetInstallFlags()
		out, err := runInstallCmd(t, "",
			"--state", statePath,
			"--yes",
			"install", "demo",
			"--profile", "common",
		)
		require.NoError(t, err, "out=%s", out)

		ok, err := symlink.IsSymlinkTo(tgtCommon, srcCommon)
		require.NoError(t, err)
		assert.True(t, ok, "expected %s -> %s", tgtCommon, srcCommon)

		_, err = os.Lstat(tgtWork)
		assert.True(t, os.IsNotExist(err), "dotfile2 must not exist after common install")

		s, err := state.Load(statePath)
		require.NoError(t, err)
		ps, present := s["demo"]
		require.True(t, present, "state must contain 'demo'")
		assert.Equal(t, "common", ps.Profile)
		require.Len(t, ps.InstalledLinks, 1)
		assert.Equal(t, srcCommon, ps.InstalledLinks[0].Source)
		assert.Equal(t, tgtCommon, ps.InstalledLinks[0].Target)
	})

	t.Run("status_all", func(t *testing.T) {
		resetInstallFlags()
		out, err := runInstallCmd(t, "",
			"--state", statePath,
			"status",
		)
		require.NoError(t, err, "out=%s", out)
		assert.Contains(t, out, "demo")
		assert.Contains(t, out, "common")
	})

	t.Run("status_demo", func(t *testing.T) {
		resetInstallFlags()
		out, err := runInstallCmd(t, "",
			"--state", statePath,
			"status", "demo",
		)
		require.NoError(t, err, "out=%s", out)
		assert.Contains(t, out, "demo")
	})

	t.Run("switch_to_work", func(t *testing.T) {
		resetInstallFlags()
		out, err := runInstallCmd(t, "",
			"--state", statePath,
			"--yes",
			"switch", "demo", "work",
		)
		require.NoError(t, err, "out=%s", out)

		_, err = os.Lstat(tgtCommon)
		assert.True(t, os.IsNotExist(err), "no orphan common symlink after switch; lstat err=%v", err)

		ok, err := symlink.IsSymlinkTo(tgtWork, srcWork)
		require.NoError(t, err)
		assert.True(t, ok, "expected %s -> %s after switch", tgtWork, srcWork)

		s, err := state.Load(statePath)
		require.NoError(t, err)
		ps, present := s["demo"]
		require.True(t, present, "state must still contain 'demo' after switch")
		assert.Equal(t, "work", ps.Profile)
		require.Len(t, ps.InstalledLinks, 1)
		assert.Equal(t, srcWork, ps.InstalledLinks[0].Source)
		assert.Equal(t, tgtWork, ps.InstalledLinks[0].Target)
	})

	t.Run("uninstall_demo", func(t *testing.T) {
		resetInstallFlags()
		out, err := runInstallCmd(t, "",
			"--state", statePath,
			"--yes",
			"uninstall", "demo",
		)
		require.NoError(t, err, "out=%s", out)

		_, err = os.Lstat(tgtCommon)
		assert.True(t, os.IsNotExist(err), "common link must be gone")
		_, err = os.Lstat(tgtWork)
		assert.True(t, os.IsNotExist(err), "work link must be gone")

		s, err := state.Load(statePath)
		require.NoError(t, err)
		_, present := s["demo"]
		assert.False(t, present, "state must NOT contain 'demo' after uninstall")

		got, err := os.ReadFile(srcCommon)
		require.NoError(t, err)
		assert.Equal(t, commonContent, got, "common source file must be unchanged")
		got, err = os.ReadFile(srcWork)
		require.NoError(t, err)
		assert.Equal(t, workContent, got, "work source file must be unchanged")
	})

	if realEasyriceDir != "" {
		assert.Equal(t, realEasyriceSnap, dirSnapshot(t, realEasyriceDir),
			"real %s must be untouched by integration test", realEasyriceDir)
	}
}
