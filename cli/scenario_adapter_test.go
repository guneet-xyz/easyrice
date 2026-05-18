//go:build !windows

package main

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/guneet-xyz/easyrice/internal/testhelpers/scenario"
)

// scenarioMockRegistry maps mock names to setup functions for scenario-based tests.
// Individual test files add entries as needed.
var scenarioMockRegistry = map[string]func(testing.TB){}

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
