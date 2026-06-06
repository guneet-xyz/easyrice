package main

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
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

// TestScenarios_AllDiscovered runs every auto-discoverable scenario.
func TestScenarios_AllDiscovered(t *testing.T) {
	skipOnWindows(t)
	scenarios := discoverScenarios(t)
	for _, name := range scenarios {
		name := name
		t.Run(name, func(t *testing.T) {
			runScenarioFromTestdata(t, name)
		})
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
