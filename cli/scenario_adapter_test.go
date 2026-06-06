package main

import (
	"bytes"
	"io"
	"os"
	"runtime"
	"testing"

	"github.com/guneet-xyz/easyrice/internal/deps"
	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/guneet-xyz/easyrice/internal/testhelpers/scenario"
)

// skipOnWindows skips the test on Windows since scenario tests require POSIX symlinks.
func skipOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("scenario tests require POSIX symlinks")
	}
}

// scenarioMockRegistry maps mock names to setup functions for scenario-based tests.
// Individual test files add entries as needed.
var scenarioMockRegistry = map[string]func(testing.TB){
	// deps_mock injects a deps.MockRunner that reports `ripgrep` is already
	// installed (single rg --version probe returning exit 0). It restores the
	// previous DepsRunner via t.Cleanup. Scenarios that use this mock must
	// declare a package whose only dependency is `ripgrep`.
	"deps_mock": func(t testing.TB) {
		t.Helper()
		mock := &deps.MockRunner{
			Expectations: []deps.MockExpectation{
				{Argv: []string{"rg", "--version"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("ripgrep 14.1.0\n")}},
			},
		}
		orig := DepsRunner
		DepsRunner = mock
		t.Cleanup(func() { DepsRunner = orig })
	},
}

// newScenarioConfig builds a scenario.Config wired to the real cobra rootCmd.
func newScenarioConfig() scenario.Config {
	return scenario.Config{
		Runner: func(args []string, stdin io.Reader) (string, error) {
			buf := &bytes.Buffer{}
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetIn(stdin)
			rootCmd.SetArgs(args)
			err := rootCmd.Execute()
			rootCmd.SetOut(os.Stdout)
			rootCmd.SetErr(os.Stderr)
			rootCmd.SetIn(os.Stdin)
			return buf.String(), err
		},
		Reset: func() {
			flagProfile = ""
			flagYes = false
			flagSkipDeps = false
			flagState = state.DefaultPath()
			flagLogLevel = ""
		},
		MockRegistry: scenarioMockRegistry,
	}
}
