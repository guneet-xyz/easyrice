package scenario

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func setupScenarioDirs(t *testing.T) (scenarioDir, home, repo, statePath string) {
	t.Helper()
	root := t.TempDir()
	scenarioDir = filepath.Join(root, "scenario")
	home = filepath.Join(root, "home")
	repo = filepath.Join(root, "repo")
	statePath = filepath.Join(root, "state.json")

	require.NoError(t, os.MkdirAll(scenarioDir, 0o755))
	require.NoError(t, os.MkdirAll(home, 0o755))
	require.NoError(t, os.MkdirAll(repo, 0o755))
	return scenarioDir, home, repo, statePath
}

// TestIntegration_HappyPath exercises LoadScenario → runner → assertions → home snapshot.
func TestIntegration_HappyPath(t *testing.T) {
	scenarioDir, home, repo, statePath := setupScenarioDirs(t)

	repoConfig := filepath.Join(repo, "pkg", "config")
	writeFile(t, repoConfig, "pkg config body\n")

	stepsYAML := strings.NewReplacer(
		"__HOME__", home,
		"__REPO__", repo,
		"__STATE__", statePath,
	).Replace(`
steps:
  - name: install pkg
    args: ["install", "pkg"]
    env:
      HOME: __HOME__
      EASYRICE_REPO: __REPO__
      EASYRICE_STATE: __STATE__
    expect:
      exit_code: 0
      stdout_contains: ["installed"]
      home: "expected/step-0/home.txt"
`)
	writeFile(t, filepath.Join(scenarioDir, "steps.yaml"), stepsYAML)

	expectedHome := ".config [dir]\n" +
		".config/pkg [dir]\n" +
		".config/pkg/config -> <REPO>/pkg/config\n"
	writeFile(t, filepath.Join(scenarioDir, "expected", "step-0", "home.txt"), expectedHome)

	runnerCalled := false
	cfg := Config{
		Runner: func(args []string, stdin io.Reader) (string, error) {
			runnerCalled = true
			targetDir := filepath.Join(home, ".config", "pkg")
			require.NoError(t, os.MkdirAll(targetDir, 0o755))
			require.NoError(t, os.Symlink(repoConfig, filepath.Join(targetDir, "config")))
			require.NoError(t, os.WriteFile(statePath, []byte(`{"pkg":{}}`), 0o644))
			return "installed pkg\n", nil
		},
	}

	Run(t, scenarioDir, cfg)

	require.True(t, runnerCalled, "runner must be invoked")
}

// TestIntegration_MultiStep runs install then uninstall, each with its own snapshot.
func TestIntegration_MultiStep(t *testing.T) {
	scenarioDir, home, repo, statePath := setupScenarioDirs(t)

	repoConfig := filepath.Join(repo, "pkg", "config")
	writeFile(t, repoConfig, "pkg config body\n")

	stepsYAML := `
steps:
  - name: install
    args: ["install", "pkg"]
    env:
      HOME: ` + home + `
      EASYRICE_REPO: ` + repo + `
      EASYRICE_STATE: ` + statePath + `
    expect:
      exit_code: 0
      stdout_contains: ["installed"]
      home: "expected/step-0/home.txt"
  - name: uninstall
    args: ["uninstall", "pkg"]
    env:
      HOME: ` + home + `
      EASYRICE_REPO: ` + repo + `
      EASYRICE_STATE: ` + statePath + `
    expect:
      exit_code: 0
      stdout_contains: ["removed"]
      home: "expected/step-1/home.txt"
`
	writeFile(t, filepath.Join(scenarioDir, "steps.yaml"), stepsYAML)

	writeFile(t, filepath.Join(scenarioDir, "expected", "step-0", "home.txt"),
		".config [dir]\n.config/pkg [dir]\n.config/pkg/config -> <REPO>/pkg/config\n")
	writeFile(t, filepath.Join(scenarioDir, "expected", "step-1", "home.txt"), "")

	var order []string
	cfg := Config{
		Runner: func(args []string, stdin io.Reader) (string, error) {
			switch args[0] {
			case "install":
				order = append(order, "install")
				targetDir := filepath.Join(home, ".config", "pkg")
				require.NoError(t, os.MkdirAll(targetDir, 0o755))
				require.NoError(t, os.Symlink(repoConfig, filepath.Join(targetDir, "config")))
				return "installed pkg\n", nil
			case "uninstall":
				order = append(order, "uninstall")
				require.NoError(t, os.RemoveAll(filepath.Join(home, ".config")))
				return "removed pkg\n", nil
			}
			return "", nil
		},
	}

	Run(t, scenarioDir, cfg)

	require.Equal(t, []string{"install", "uninstall"}, order)
}

// TestIntegration_MutateBeforeRunner verifies mutate ops fire before the runner.
func TestIntegration_MutateBeforeRunner(t *testing.T) {
	scenarioDir, home, repo, statePath := setupScenarioDirs(t)

	stepsYAML := `
steps:
  - name: read pre-existing
    args: ["read"]
    env:
      HOME: ` + home + `
      EASYRICE_REPO: ` + repo + `
      EASYRICE_STATE: ` + statePath + `
    mutate:
      - op: write_file
        path: "<HOME>/pre-existing.txt"
        content: "hello"
    expect:
      exit_code: 0
      stdout_contains: ["hello"]
`
	writeFile(t, filepath.Join(scenarioDir, "steps.yaml"), stepsYAML)

	cfg := Config{
		Runner: func(args []string, stdin io.Reader) (string, error) {
			data, err := os.ReadFile(filepath.Join(home, "pre-existing.txt"))
			if err != nil {
				return "", err
			}
			return string(data) + "\n", nil
		},
	}

	Run(t, scenarioDir, cfg)
}

// TestIntegration_MockActivated verifies registered mocks fire before steps.
func TestIntegration_MockActivated(t *testing.T) {
	scenarioDir, home, repo, statePath := setupScenarioDirs(t)

	stepsYAML := `
mocks: ["my-mock"]
steps:
  - name: trivial
    args: ["noop"]
    env:
      HOME: ` + home + `
      EASYRICE_REPO: ` + repo + `
      EASYRICE_STATE: ` + statePath + `
    expect:
      exit_code: 0
`
	writeFile(t, filepath.Join(scenarioDir, "steps.yaml"), stepsYAML)

	mockCalled := false
	cfg := Config{
		Runner: func(args []string, stdin io.Reader) (string, error) {
			return "", nil
		},
		MockRegistry: map[string]func(testing.TB){
			"my-mock": func(tb testing.TB) {
				mockCalled = true
			},
		},
	}

	Run(t, scenarioDir, cfg)

	require.True(t, mockCalled, "registered mock must be activated")
}
