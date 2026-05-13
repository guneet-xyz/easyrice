package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetRootCmd() {
	flagState = ""
	flagLogLevel = ""
	flagYes = false
}

func TestVersionCommand(t *testing.T) {
	resetRootCmd()
	cmd := rootCmd
	cmd.SetArgs([]string{"version"})
	err := cmd.Execute()
	require.NoError(t, err)
}

func TestInvalidLogLevel(t *testing.T) {
	resetRootCmd()
	cmd := rootCmd
	cmd.SetArgs([]string{"--log-level", "invalid", "version"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid log level")
	assert.Contains(t, err.Error(), "choose one of")
}

func TestHelpContainsFlags(t *testing.T) {
	resetRootCmd()
	buf := &bytes.Buffer{}
	cmd := rootCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "--log-level")
	assert.Contains(t, output, "--yes")
	assert.Contains(t, output, "--state")
}

func TestEASYRICELogLevelEnvVar(t *testing.T) {
	resetRootCmd()
	oldVal := os.Getenv("EASYRICE_LOG_LEVEL")
	defer func() {
		if oldVal != "" {
			os.Setenv("EASYRICE_LOG_LEVEL", oldVal)
		} else {
			os.Unsetenv("EASYRICE_LOG_LEVEL")
		}
	}()

	os.Setenv("EASYRICE_LOG_LEVEL", "debug")

	cmd := rootCmd
	cmd.SetArgs([]string{"version"})
	err := cmd.Execute()
	require.NoError(t, err)
}

func TestLogLevelFlagOverridesEnv(t *testing.T) {
	resetRootCmd()
	oldVal := os.Getenv("EASYRICE_LOG_LEVEL")
	defer func() {
		if oldVal != "" {
			os.Setenv("EASYRICE_LOG_LEVEL", oldVal)
		} else {
			os.Unsetenv("EASYRICE_LOG_LEVEL")
		}
	}()

	os.Setenv("EASYRICE_LOG_LEVEL", "debug")

	cmd := rootCmd
	cmd.SetArgs([]string{"--log-level", "warn", "version"})
	err := cmd.Execute()
	require.NoError(t, err)
}

func TestRootCmd_InvalidLogLevelHidesUsage(t *testing.T) {
	resetRootCmd()
	buf := &bytes.Buffer{}
	cmd := rootCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--log-level=invalid", "version"})
	t.Cleanup(func() {
		cmd.SetArgs(nil)
	})

	err := cmd.Execute()
	require.Error(t, err)

	output := buf.String()
	assert.NotContains(t, output, "Usage:")
	assert.Contains(t, output, "invalid log level")
}
