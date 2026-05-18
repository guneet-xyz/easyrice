package scenario

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeScenario(t *testing.T, yaml string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "steps.yaml"), []byte(yaml), 0o644))
	return dir
}

func TestRun_HappyPath(t *testing.T) {
	home := t.TempDir()
	scenarioDir := writeScenario(t, `
steps:
  - name: greet
    args: ["echo", "hello"]
    env:
      HOME: `+home+`
    expect:
      exit_code: 0
      stdout_contains: ["hello"]
`)

	var capturedArgs []string
	var capturedStdin string
	cfg := Config{
		Runner: func(args []string, stdin io.Reader) (string, error) {
			capturedArgs = args
			b, _ := io.ReadAll(stdin)
			capturedStdin = string(b)
			return "hello world\n", nil
		},
	}

	Run(t, scenarioDir, cfg)

	require.Equal(t, []string{"echo", "hello"}, capturedArgs)
	require.Equal(t, "", capturedStdin)
}

func TestRun_NonZeroExitExpected(t *testing.T) {
	home := t.TempDir()
	scenarioDir := writeScenario(t, `
steps:
  - name: failure
    args: ["fail"]
    env:
      HOME: `+home+`
    expect:
      exit_code: 1
`)

	cfg := Config{
		Runner: func(args []string, stdin io.Reader) (string, error) {
			return "", errors.New("boom")
		},
	}

	Run(t, scenarioDir, cfg)
}

func TestRun_MutateApplied(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	target := filepath.Join(home, "seed.txt")

	scenarioDir := writeScenario(t, `
steps:
  - name: write-then-run
    args: ["noop"]
    env:
      HOME: `+home+`
      EASYRICE_REPO: `+repo+`
    mutate:
      - op: write_file
        path: <HOME>/seed.txt
        content: "seeded"
    expect:
      exit_code: 0
`)

	called := false
	cfg := Config{
		Runner: func(args []string, stdin io.Reader) (string, error) {
			called = true
			b, err := os.ReadFile(target)
			require.NoError(t, err, "mutate should have created file before runner ran")
			require.Equal(t, "seeded", string(b))
			return "", nil
		},
	}

	Run(t, scenarioDir, cfg)
	require.True(t, called, "runner must have been invoked")

	b, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, "seeded", string(b))
}

func TestRun_MockActivated(t *testing.T) {
	home := t.TempDir()
	scenarioDir := writeScenario(t, `
mocks:
  - my-mock
steps:
  - name: with-mock
    args: ["noop"]
    env:
      HOME: `+home+`
    expect:
      exit_code: 0
`)

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
	require.True(t, mockCalled, "mock setup function should have been called")
}

func TestRun_ResetCalled(t *testing.T) {
	home := t.TempDir()
	scenarioDir := writeScenario(t, `
steps:
  - name: s1
    args: ["a"]
    env:
      HOME: `+home+`
    expect:
      exit_code: 0
  - name: s2
    args: ["b"]
    env:
      HOME: `+home+`
    expect:
      exit_code: 0
`)

	resetCount := 0
	cfg := Config{
		Runner: func(args []string, stdin io.Reader) (string, error) {
			return "", nil
		},
		Reset: func() { resetCount++ },
	}

	Run(t, scenarioDir, cfg)
	require.Equal(t, 2, resetCount)
}

func TestRun_StdinPassed(t *testing.T) {
	home := t.TempDir()
	scenarioDir := writeScenario(t, `
steps:
  - name: stdin-step
    args: ["read"]
    stdin: "piped-input"
    env:
      HOME: `+home+`
    expect:
      exit_code: 0
`)

	var got string
	cfg := Config{
		Runner: func(args []string, stdin io.Reader) (string, error) {
			b, _ := io.ReadAll(stdin)
			got = string(b)
			return "", nil
		},
	}

	Run(t, scenarioDir, cfg)
	require.Equal(t, "piped-input", got)
	require.True(t, strings.HasPrefix(got, "piped"))
}
