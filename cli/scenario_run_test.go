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

// scenarioSandbox is the per-test sandbox for scenario-driven tests.
type scenarioSandbox struct {
	HomeDir   string
	RepoRoot  string
	StateFile string
}

// setupScenarioSandbox creates an isolated $HOME, ensures the managed repo
// root exists, and returns paths a scenario can render its steps.yaml against.
// It does NOT copy the scenario's repo/ seed - callers do that themselves so
// they can interleave Go-side preludes (e.g. git submodule setup) between
// seeding and rendering.
func setupScenarioSandbox(t *testing.T) scenarioSandbox {
	t.Helper()
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
	t.Setenv("AppData", filepath.Join(homeDir, "AppData"))

	repoRoot := repo.DefaultRepoPath()
	require.NoError(t, os.MkdirAll(repoRoot, 0o755))

	stateFile := filepath.Join(homeDir, ".config", "easyrice", "state.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(stateFile), 0o755))
	return scenarioSandbox{HomeDir: homeDir, RepoRoot: repoRoot, StateFile: stateFile}
}

// renderScenario copies the scenario at srcDir into a fresh temp dir,
// substitutes __HOME__/__REPO__/__STATE__ placeholders in steps.yaml using
// the sandbox values, and returns the rendered scenario dir ready for
// scenario.Run.
func renderScenario(t *testing.T, srcDir string, sb scenarioSandbox) string {
	t.Helper()
	scenarioDir := t.TempDir()
	copyTree(t, srcDir, scenarioDir)

	stepsPath := filepath.Join(scenarioDir, "steps.yaml")
	raw, err := os.ReadFile(stepsPath)
	require.NoError(t, err)
	rendered := strings.NewReplacer(
		"__HOME__", sb.HomeDir,
		"__REPO__", sb.RepoRoot,
		"__STATE__", sb.StateFile,
	).Replace(string(raw))
	require.NoError(t, os.WriteFile(stepsPath, []byte(rendered), 0o644))
	return scenarioDir
}

func TestScenario_InstallProfileHappy(t *testing.T) {
	skipOnWindows(t)
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	srcDir, err := filepath.Abs(filepath.Join("testdata", "scenarios", "install_profile_happy"))
	require.NoError(t, err)

	sb := setupScenarioSandbox(t)
	copyTree(t, filepath.Join(srcDir, "repo"), sb.RepoRoot)

	scenarioDir := renderScenario(t, srcDir, sb)
	scenario.Run(t, scenarioDir, newScenarioConfig())
}

func TestScenario_InstallDepsMock(t *testing.T) {
	skipOnWindows(t)
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	srcDir, err := filepath.Abs(filepath.Join("testdata", "scenarios", "install_deps_mock"))
	require.NoError(t, err)

	sb := setupScenarioSandbox(t)
	copyTree(t, filepath.Join(srcDir, "repo"), sb.RepoRoot)

	scenarioDir := renderScenario(t, srcDir, sb)
	scenario.Run(t, scenarioDir, newScenarioConfig())
}

func TestScenario_InstallStdinConfirm(t *testing.T) {
	skipOnWindows(t)
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	srcDir, err := filepath.Abs(filepath.Join("testdata", "scenarios", "install_stdin_confirm"))
	require.NoError(t, err)

	sb := setupScenarioSandbox(t)
	copyTree(t, filepath.Join(srcDir, "repo"), sb.RepoRoot)

	scenarioDir := renderScenario(t, srcDir, sb)
	scenario.Run(t, scenarioDir, newScenarioConfig())
}
