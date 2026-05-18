//go:build !windows

package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/repo"
	"github.com/guneet-xyz/easyrice/internal/testhelpers/scenario"
)

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	require.NoError(t, filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	}))
}

func TestScenario_InstallProfileHappy(t *testing.T) {
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	srcDir, err := filepath.Abs(filepath.Join("testdata", "scenarios", "install_profile_happy"))
	require.NoError(t, err)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
	t.Setenv("AppData", filepath.Join(homeDir, "AppData"))

	repoRoot := repo.DefaultRepoPath()
	require.NoError(t, os.MkdirAll(repoRoot, 0o755))
	copyTree(t, filepath.Join(srcDir, "repo"), repoRoot)

	stateFile := filepath.Join(t.TempDir(), "state.json")

	scenarioDir := t.TempDir()
	copyTree(t, srcDir, scenarioDir)

	stepsPath := filepath.Join(scenarioDir, "steps.yaml")
	raw, err := os.ReadFile(stepsPath)
	require.NoError(t, err)
	rendered := strings.NewReplacer(
		"__HOME__", homeDir,
		"__REPO__", repoRoot,
		"__STATE__", stateFile,
	).Replace(string(raw))
	require.NoError(t, os.WriteFile(stepsPath, []byte(rendered), 0o644))

	scenario.Run(t, scenarioDir, newScenarioConfig())
}
