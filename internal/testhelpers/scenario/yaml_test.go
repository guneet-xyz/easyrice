package scenario

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadScenario_HappyPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create expected snapshot files
	expectedDir := filepath.Join(tmpDir, "expected", "step-1")
	if err := os.MkdirAll(expectedDir, 0o755); err != nil {
		t.Fatalf("failed to create expected dir: %v", err)
	}

	stateFile := filepath.Join(expectedDir, "state.json")
	if err := os.WriteFile(stateFile, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("failed to write state file: %v", err)
	}

	homeFile := filepath.Join(expectedDir, "home.txt")
	if err := os.WriteFile(homeFile, []byte("home content"), 0o644); err != nil {
		t.Fatalf("failed to write home file: %v", err)
	}

	// Create steps.yaml
	stepsYAML := `mocks:
  - mock-1

steps:
  - name: "install foo"
    args: ["install", "foo", "--profile", "common", "--yes"]
    stdin: "y\n"
    env:
      MY_VAR: "value"
    mutate:
      - op: write_file
        path: <HOME>/.config/foo/extra
        content: "extra content"
    expect:
      exit_code: 0
      stdout_contains:
        - "Installed foo"
      stdout_not_contains:
        - "error"
      stdout_equals: ""
      state: expected/step-1/state.json
      home: expected/step-1/home.txt
`

	stepsPath := filepath.Join(tmpDir, "steps.yaml")
	if err := os.WriteFile(stepsPath, []byte(stepsYAML), 0o644); err != nil {
		t.Fatalf("failed to write steps.yaml: %v", err)
	}

	// Load scenario
	scenario, err := LoadScenario(tmpDir)
	if err != nil {
		t.Fatalf("LoadScenario failed: %v", err)
	}

	// Verify basic structure
	if scenario.Dir != tmpDir {
		t.Errorf("scenario.Dir = %q, want %q", scenario.Dir, tmpDir)
	}

	if len(scenario.Mocks) != 1 || scenario.Mocks[0] != "mock-1" {
		t.Errorf("scenario.Mocks = %v, want [mock-1]", scenario.Mocks)
	}

	if len(scenario.Steps) != 1 {
		t.Errorf("len(scenario.Steps) = %d, want 1", len(scenario.Steps))
	}

	step := scenario.Steps[0]
	if step.Name != "install foo" {
		t.Errorf("step.Name = %q, want %q", step.Name, "install foo")
	}

	if len(step.Args) != 5 {
		t.Errorf("len(step.Args) = %d, want 5", len(step.Args))
	}

	if step.Stdin != "y\n" {
		t.Errorf("step.Stdin = %q, want %q", step.Stdin, "y\n")
	}

	if step.Env["MY_VAR"] != "value" {
		t.Errorf("step.Env[MY_VAR] = %q, want %q", step.Env["MY_VAR"], "value")
	}

	if len(step.Mutate) != 1 {
		t.Errorf("len(step.Mutate) = %d, want 1", len(step.Mutate))
	}

	mutateOp := step.Mutate[0]
	if mutateOp.Op != "write_file" {
		t.Errorf("mutateOp.Op = %q, want %q", mutateOp.Op, "write_file")
	}

	if mutateOp.Content != "extra content" {
		t.Errorf("mutateOp.Content = %q, want %q", mutateOp.Content, "extra content")
	}

	if step.Expect.ExitCode != 0 {
		t.Errorf("step.Expect.ExitCode = %d, want 0", step.Expect.ExitCode)
	}

	if len(step.Expect.StdoutContains) != 1 || step.Expect.StdoutContains[0] != "Installed foo" {
		t.Errorf("step.Expect.StdoutContains = %v, want [Installed foo]", step.Expect.StdoutContains)
	}

	if step.Expect.State != "expected/step-1/state.json" {
		t.Errorf("step.Expect.State = %q, want %q", step.Expect.State, "expected/step-1/state.json")
	}
}

func TestLoadScenario_MissingName(t *testing.T) {
	tmpDir := t.TempDir()

	stepsYAML := `steps:
  - name: ""
    args: ["install", "foo"]
`

	stepsPath := filepath.Join(tmpDir, "steps.yaml")
	if err := os.WriteFile(stepsPath, []byte(stepsYAML), 0o644); err != nil {
		t.Fatalf("failed to write steps.yaml: %v", err)
	}

	_, err := LoadScenario(tmpDir)
	if err == nil {
		t.Fatal("LoadScenario should have failed for empty step name")
	}

	if !errors.Is(err, ErrInvalidYAML) {
		t.Errorf("error = %v, want wrapped ErrInvalidYAML", err)
	}
}

func TestLoadScenario_UnknownMutateOp(t *testing.T) {
	tmpDir := t.TempDir()

	stepsYAML := `steps:
  - name: "test step"
    args: ["install", "foo"]
    mutate:
      - op: "explode"
        path: "/some/path"
`

	stepsPath := filepath.Join(tmpDir, "steps.yaml")
	if err := os.WriteFile(stepsPath, []byte(stepsYAML), 0o644); err != nil {
		t.Fatalf("failed to write steps.yaml: %v", err)
	}

	_, err := LoadScenario(tmpDir)
	if err == nil {
		t.Fatal("LoadScenario should have failed for unknown mutate op")
	}

	if !errors.Is(err, ErrInvalidYAML) {
		t.Errorf("error = %v, want wrapped ErrInvalidYAML", err)
	}
}

func TestLoadScenario_MissingExpectedFile(t *testing.T) {
	tmpDir := t.TempDir()

	stepsYAML := `steps:
  - name: "test step"
    args: ["install", "foo"]
    expect:
      state: expected/step-1/state.json
`

	stepsPath := filepath.Join(tmpDir, "steps.yaml")
	if err := os.WriteFile(stepsPath, []byte(stepsYAML), 0o644); err != nil {
		t.Fatalf("failed to write steps.yaml: %v", err)
	}

	_, err := LoadScenario(tmpDir)
	if err == nil {
		t.Fatal("LoadScenario should have failed for missing expected file")
	}

	if !errors.Is(err, ErrInvalidYAML) {
		t.Errorf("error = %v, want wrapped ErrInvalidYAML", err)
	}
}

func TestLoadScenario_MissingStepsFile(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := LoadScenario(tmpDir)
	if err == nil {
		t.Fatal("LoadScenario should have failed for missing steps.yaml")
	}

	if !errors.Is(err, ErrInvalidYAML) {
		t.Errorf("error = %v, want wrapped ErrInvalidYAML", err)
	}
}

func TestLoadScenario_NoArgsNoMutate(t *testing.T) {
	tmpDir := t.TempDir()

	stepsYAML := `steps:
  - name: "test step"
`

	stepsPath := filepath.Join(tmpDir, "steps.yaml")
	if err := os.WriteFile(stepsPath, []byte(stepsYAML), 0o644); err != nil {
		t.Fatalf("failed to write steps.yaml: %v", err)
	}

	_, err := LoadScenario(tmpDir)
	if err == nil {
		t.Fatal("LoadScenario should have failed for step with no args and no mutate")
	}

	if !errors.Is(err, ErrInvalidYAML) {
		t.Errorf("error = %v, want wrapped ErrInvalidYAML", err)
	}
}
