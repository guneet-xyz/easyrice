package main

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/repo"
	"github.com/guneet-xyz/easyrice/internal/testhelpers/scenario"
)

// scenariosNeedingCustomPrep lists scenario names that require git submodule
// setup or other custom preparation that scenario.Run cannot handle.
// These are kept as hardcoded TestScenario_* functions in remote_e2e_test.go.
var scenariosNeedingCustomPrep = map[string]bool{
	"remote_import_resolves":   true,
	"remote_missing_submodule": true,
}

// discoverScenarios returns a sorted list of scenario directory names under
// cli/testdata/scenarios/ where steps.yaml exists, excluding scenarios that
// need custom prep.
func discoverScenarios(t *testing.T) []string {
	t.Helper()
	base, err := filepath.Abs(filepath.Join("testdata", "scenarios"))
	require.NoError(t, err)
	entries, err := os.ReadDir(base)
	require.NoError(t, err)
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if scenariosNeedingCustomPrep[e.Name()] {
			continue
		}
		if _, err := os.Stat(filepath.Join(base, e.Name(), "steps.yaml")); err != nil {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

// scenarioWantsStateInsideHome reports whether the scenario's expected
// home.txt snapshots reference `state.json`. When true, the scenario was
// authored against setupScenarioSandbox (state inside $HOME); otherwise
// it was authored against runScenarioFromTestdata (state outside $HOME).
func scenarioWantsStateInsideHome(t *testing.T, srcDir string) bool {
	t.Helper()
	expectedDir := filepath.Join(srcDir, "expected")
	wants := false
	_ = filepath.WalkDir(expectedDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if filepath.Base(path) != "home.txt" {
			return nil
		}
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		if strings.Contains(string(raw), "state.json") {
			wants = true
		}
		return nil
	})
	return wants
}

// runDiscoveredScenario drives a single scenario, picking the sandbox style
// (state inside vs outside $HOME) based on the expected snapshots.
func runDiscoveredScenario(t *testing.T, name string) {
	t.Helper()
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	srcDir, err := filepath.Abs(filepath.Join("testdata", "scenarios", name))
	require.NoError(t, err)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
	t.Setenv("AppData", filepath.Join(homeDir, "AppData"))

	repoRoot := repo.DefaultRepoPath()
	require.NoError(t, os.MkdirAll(repoRoot, 0o755))

	repoSrc := filepath.Join(srcDir, "repo")
	if _, err := os.Stat(repoSrc); err == nil {
		copyTree(t, repoSrc, repoRoot)
	}

	var stateFile string
	if scenarioWantsStateInsideHome(t, srcDir) {
		stateFile = filepath.Join(homeDir, ".config", "easyrice", "state.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(stateFile), 0o755))
	} else {
		stateFile = filepath.Join(t.TempDir(), "state.json")
	}

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

// TestScenarios_AllDiscovered runs every auto-discoverable scenario.
func TestScenarios_AllDiscovered(t *testing.T) {
	skipOnWindows(t)
	scenarios := discoverScenarios(t)
	for _, name := range scenarios {
		name := name
		t.Run(name, func(t *testing.T) {
			runDiscoveredScenario(t, name)
		})
	}
}

// TestScenarios_GlobMatchesHardcoded verifies that the glob walker discovers
// exactly the same set of scenarios as the hardcoded TestScenario_* functions
// (plus the custom-prep skip-list). The hardcoded list below is intentionally
// static: deriving it dynamically would defeat the purpose of detecting drift
// between the two enumerations.
func TestScenarios_GlobMatchesHardcoded(t *testing.T) {
	// Hardcoded scenario names extracted from the harness files:
	// - cli/scenario_run_test.go
	// - cli/scenarios_migrated_test.go
	// - cli/converge_scenarios_test.go
	// - cli/conflict_e2e_test.go
	// - cli/remote_e2e_test.go (custom-prep, also in scenariosNeedingCustomPrep)
	hardcoded := []string{
		// scenario_run_test.go
		"install_profile_happy",
		"install_deps_mock",
		"install_stdin_confirm",
		// scenarios_migrated_test.go
		"uninstall_happy",
		"uninstall_manually_deleted",
		"uninstall_replaced_by_file",
		"uninstall_replaced_by_dir",
		"uninstall_folder_mode_replaced",
		"uninstall_preserves_others",
		"install_deep_nested_target",
		"install_home_expansion",
		"install_overlay_last_wins",
		"install_no_args_converges_all",
		// converge_scenarios_test.go
		"converge-install-fresh",
		"converge-profile-switch",
		"converge-repair-broken-symlink",
		"converge-noop",
		// conflict_e2e_test.go
		"conflict_preexisting_file",
		"conflict_two_packages_same_target",
	}
	for name := range scenariosNeedingCustomPrep {
		hardcoded = append(hardcoded, name)
	}
	sort.Strings(hardcoded)

	discovered := discoverScenarios(t)
	full := append([]string{}, discovered...)
	for name := range scenariosNeedingCustomPrep {
		full = append(full, name)
	}
	sort.Strings(full)

	if !reflect.DeepEqual(hardcoded, full) {
		hardcodedMap := map[string]bool{}
		for _, n := range hardcoded {
			hardcodedMap[n] = true
		}
		fullMap := map[string]bool{}
		for _, n := range full {
			fullMap[n] = true
		}

		var inHardcodedOnly, inDiscoveredOnly []string
		for n := range hardcodedMap {
			if !fullMap[n] {
				inHardcodedOnly = append(inHardcodedOnly, n)
			}
		}
		for n := range fullMap {
			if !hardcodedMap[n] {
				inDiscoveredOnly = append(inDiscoveredOnly, n)
			}
		}
		sort.Strings(inHardcodedOnly)
		sort.Strings(inDiscoveredOnly)

		t.Errorf("glob discovery mismatch:\n  in hardcoded only: %v\n  in discovered only: %v",
			inHardcodedOnly, inDiscoveredOnly)
	}
}

// TestScenario_Count is a sentinel that fails if the scenario walker discovers
// fewer scenarios than expected. This guards against path bugs that would
// silently run 0 tests.
func TestScenario_Count(t *testing.T) {
	scenarios := discoverScenarios(t)
	total := len(scenarios) + len(scenariosNeedingCustomPrep)
	if total < 21 {
		t.Fatalf("scenario discovery found only %d scenarios (discovered=%d, customPrep=%d); "+
			"expected >= 21. Possible path bug. Found: %v",
			total, len(scenarios), len(scenariosNeedingCustomPrep), scenarios)
	}
}
