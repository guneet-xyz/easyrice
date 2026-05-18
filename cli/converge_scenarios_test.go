//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/repo"
	"github.com/guneet-xyz/easyrice/internal/testhelpers/scenario"
)

func runConvergeScenario(t *testing.T, scenarioName string) {
	t.Helper()
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	srcDir, err := filepath.Abs(filepath.Join("testdata", "scenarios", scenarioName))
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

func TestScenario_ConvergeInstallFresh(t *testing.T) {
	runConvergeScenario(t, "converge-install-fresh")
}

func TestScenario_ConvergeProfileSwitch(t *testing.T) {
	runConvergeScenario(t, "converge-profile-switch")
}

func TestScenario_ConvergeRepairBrokenSymlink(t *testing.T) {
	runConvergeScenario(t, "converge-repair-broken-symlink")
}

func TestScenario_ConvergeNoOp(t *testing.T) {
	runConvergeScenario(t, "converge-noop")
}
