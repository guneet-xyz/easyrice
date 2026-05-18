package scenario

import (
	"path/filepath"
	"strings"
	"testing"
)

// Run loads a scenario from scenarioDir and executes each step against the
// supplied Runner inside a t.Run subtest. Setup failures are fatal; assertion
// failures use t.Errorf so all assertions in a step run before it fails.
func Run(t *testing.T, scenarioDir string, cfg Config) {
	t.Helper()

	scenario, err := LoadScenario(scenarioDir)
	if err != nil {
		t.Fatalf("LoadScenario(%q): %v", scenarioDir, err)
	}

	for _, name := range scenario.Mocks {
		setup, ok := cfg.MockRegistry[name]
		if !ok {
			t.Fatalf("scenario %q: mock %q not found in MockRegistry", scenarioDir, name)
		}
		setup(t)
	}

	for _, step := range scenario.Steps {
		step := step
		t.Run(step.Name, func(t *testing.T) {
			home := step.Env["HOME"]
			repo := step.Env["EASYRICE_REPO"]
			statePath := step.Env["EASYRICE_STATE"]

			for _, op := range step.Mutate {
				if err := applyMutate(home, repo, op); err != nil {
					t.Fatalf("step %q: applyMutate(%+v): %v", step.Name, op, err)
				}
			}

			for k, v := range step.Env {
				t.Setenv(k, v)
			}

			stdout, runErr := cfg.Runner(step.Args, strings.NewReader(step.Stdin))

			if cfg.Reset != nil {
				cfg.Reset()
			}

			if step.Expect.ExitCode == 0 {
				if runErr != nil {
					t.Errorf("step %q: expected exit 0, got error: %v", step.Name, runErr)
				}
			} else if runErr == nil {
				t.Errorf("step %q: expected non-zero exit, got nil error", step.Name)
			}

			for _, s := range step.Expect.StdoutContains {
				if !strings.Contains(stdout, s) {
					t.Errorf("step %q: stdout does not contain %q\nstdout:\n%s", step.Name, s, stdout)
				}
			}

			for _, s := range step.Expect.StdoutNotContains {
				if strings.Contains(stdout, s) {
					t.Errorf("step %q: stdout should not contain %q\nstdout:\n%s", step.Name, s, stdout)
				}
			}

			if step.Expect.StdoutEquals != "" && stdout != step.Expect.StdoutEquals {
				t.Errorf("step %q: stdout mismatch\nwant: %q\ngot:  %q", step.Name, step.Expect.StdoutEquals, stdout)
			}

			if step.Expect.Home != "" {
				if home == "" {
					t.Fatalf("step %q: snapshot %s requested but env var not set", step.Name, "HOME")
				}
				actual, err := captureHomeSnapshot(home)
				if err != nil {
					t.Fatalf("step %q: captureHomeSnapshot: %v", step.Name, err)
				}
				CompareSnapshotFile(t, filepath.Join(scenarioDir, step.Expect.Home), actual, home, repo)
			}

			if step.Expect.Repo != "" {
				if repo == "" {
					t.Fatalf("step %q: snapshot %s requested but env var not set", step.Name, "EASYRICE_REPO")
				}
				actual, err := captureRepoSnapshot(repo)
				if err != nil {
					t.Fatalf("step %q: captureRepoSnapshot: %v", step.Name, err)
				}
				CompareSnapshotFile(t, filepath.Join(scenarioDir, step.Expect.Repo), actual, home, repo)
			}

			if step.Expect.State != "" {
				if statePath == "" {
					t.Fatalf("step %q: snapshot %s requested but env var not set", step.Name, "EASYRICE_STATE")
				}
				actual, err := captureStateSnapshot(statePath)
				if err != nil {
					t.Fatalf("step %q: captureStateSnapshot: %v", step.Name, err)
				}
				CompareSnapshotFile(t, filepath.Join(scenarioDir, step.Expect.State), actual, home, repo)
			}
		})
	}
}
