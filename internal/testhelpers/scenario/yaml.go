package scenario

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadScenario reads and parses a scenario from the given directory.
// It expects a steps.yaml file at <scenarioDir>/steps.yaml and validates
// all referenced expected snapshot files exist.
func LoadScenario(scenarioDir string) (*Scenario, error) {
	stepsPath := filepath.Join(scenarioDir, "steps.yaml")

	// Read the YAML file
	data, err := os.ReadFile(stepsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: steps.yaml not found in %s", ErrInvalidYAML, scenarioDir)
		}
		return nil, fmt.Errorf("%w: failed to read steps.yaml: %v", ErrInvalidYAML, err)
	}

	// Unmarshal into raw struct
	var rawScenario struct {
		Mocks []string `yaml:"mocks"`
		Steps []struct {
			Name   string                   `yaml:"name"`
			Args   []string                 `yaml:"args"`
			Stdin  string                   `yaml:"stdin"`
			Env    map[string]string        `yaml:"env"`
			Mutate []map[string]interface{} `yaml:"mutate"`
			Expect struct {
				ExitCode          int      `yaml:"exit_code"`
				StdoutContains    []string `yaml:"stdout_contains"`
				StdoutNotContains []string `yaml:"stdout_not_contains"`
				StdoutEquals      string   `yaml:"stdout_equals"`
				State             string   `yaml:"state"`
				Home              string   `yaml:"home"`
				Repo              string   `yaml:"repo"`
			} `yaml:"expect"`
		} `yaml:"steps"`
	}

	if err := yaml.Unmarshal(data, &rawScenario); err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal YAML: %v", ErrInvalidYAML, err)
	}

	// Validate and convert steps
	steps := make([]Step, len(rawScenario.Steps))
	for i, rawStep := range rawScenario.Steps {
		// Validate step name
		if rawStep.Name == "" {
			return nil, fmt.Errorf("%w: step %d has empty name", ErrInvalidYAML, i)
		}

		// Validate args (unless step only has mutate ops)
		if len(rawStep.Args) == 0 && len(rawStep.Mutate) == 0 {
			return nil, fmt.Errorf("%w: step %q has no args", ErrInvalidYAML, rawStep.Name)
		}

		// Parse and validate mutate ops
		mutateOps := make([]MutateOp, len(rawStep.Mutate))
		validOps := map[string]bool{
			"remove":          true,
			"write_file":      true,
			"replace_symlink": true,
			"mkdir":           true,
			"chmod":           true,
		}

		for j, rawOp := range rawStep.Mutate {
			opType, ok := rawOp["op"].(string)
			if !ok || opType == "" {
				return nil, fmt.Errorf("%w: step %q mutate op %d missing or invalid op field", ErrInvalidYAML, rawStep.Name, j)
			}

			if !validOps[opType] {
				return nil, fmt.Errorf("%w: step %q has unknown mutate op %q", ErrInvalidYAML, rawStep.Name, opType)
			}

			path, _ := rawOp["path"].(string)
			content, _ := rawOp["content"].(string)
			target, _ := rawOp["target"].(string)

			// Parse mode if present
			var mode os.FileMode
			if modeVal, ok := rawOp["mode"]; ok {
				switch v := modeVal.(type) {
				case int:
					mode = os.FileMode(v)
				case float64:
					mode = os.FileMode(int(v))
				case string:
					// Try to parse as octal
					var m int
					_, _ = fmt.Sscanf(v, "%o", &m)
					mode = os.FileMode(m)
				}
			}

			mutateOps[j] = MutateOp{
				Op:      opType,
				Path:    path,
				Content: content,
				Target:  target,
				Mode:    mode,
			}
		}

		// Validate expected snapshot files exist
		expectedPaths := []string{
			rawStep.Expect.State,
			rawStep.Expect.Home,
			rawStep.Expect.Repo,
		}

		for _, expectedPath := range expectedPaths {
			if expectedPath != "" {
				fullPath := filepath.Join(scenarioDir, expectedPath)
				if _, err := os.Stat(fullPath); err != nil {
					if os.IsNotExist(err) {
						return nil, fmt.Errorf("%w: expected file %q not found", ErrInvalidYAML, expectedPath)
					}
					return nil, fmt.Errorf("%w: failed to stat expected file %q: %v", ErrInvalidYAML, expectedPath, err)
				}
			}
		}

		steps[i] = Step{
			Name:   rawStep.Name,
			Args:   rawStep.Args,
			Stdin:  rawStep.Stdin,
			Env:    rawStep.Env,
			Mutate: mutateOps,
			Expect: Expect{
				ExitCode:          rawStep.Expect.ExitCode,
				StdoutContains:    rawStep.Expect.StdoutContains,
				StdoutNotContains: rawStep.Expect.StdoutNotContains,
				StdoutEquals:      rawStep.Expect.StdoutEquals,
				State:             rawStep.Expect.State,
				Home:              rawStep.Expect.Home,
				Repo:              rawStep.Expect.Repo,
			},
		}
	}

	return &Scenario{
		Dir:   scenarioDir,
		Steps: steps,
		Mocks: rawScenario.Mocks,
	}, nil
}
