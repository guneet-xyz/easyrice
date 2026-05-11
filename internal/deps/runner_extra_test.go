package deps

import (
	"context"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDependencyRefUnmarshalTOML_VersionWrongType(t *testing.T) {
	input := `dependencies = [{name = "git", version = 7}]`
	var result struct {
		Dependencies []DependencyRef `toml:"dependencies"`
	}
	err := toml.Unmarshal([]byte(input), &result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"version" must be a string`)
}

func TestExecRunner_CombinedOutputStartFailure(t *testing.T) {
	runner := &ExecRunner{}
	_, err := runner.Run(
		context.Background(),
		[]string{"definitely_not_a_real_binary_xyz_12345"},
		RunOpts{CombinedOutput: true},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "start")
}

func TestExecRunner_SeparateOutputStartFailure(t *testing.T) {
	runner := &ExecRunner{}
	_, err := runner.Run(
		context.Background(),
		[]string{"definitely_not_a_real_binary_xyz_67890"},
		RunOpts{CombinedOutput: false},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "start")
}

func TestExecRunner_SeparateOutputNonZeroExit(t *testing.T) {
	runner := &ExecRunner{}
	result, err := runner.Run(
		context.Background(),
		[]string{"sh", "-c", "exit 5"},
		RunOpts{CombinedOutput: false},
	)
	require.NoError(t, err, "non-zero exit must not be an error")
	assert.Equal(t, 5, result.ExitCode)
}

func TestExecRunner_WithEnv(t *testing.T) {
	runner := &ExecRunner{}
	result, err := runner.Run(
		context.Background(),
		[]string{"sh", "-c", "echo $RICE_TEST_VAR"},
		RunOpts{Env: []string{"RICE_TEST_VAR=hello"}, CombinedOutput: true},
	)
	require.NoError(t, err)
	assert.Contains(t, string(result.Combined), "hello")
}

func TestMockRunner_PanicsOnArgvLengthMismatch(t *testing.T) {
	mock := &MockRunner{
		Expectations: []MockExpectation{
			{Argv: []string{"echo", "hi"}, Result: RunResult{}},
		},
	}
	defer func() {
		r := recover()
		require.NotNil(t, r, "expected panic on argv length mismatch")
		msg, _ := r.(string)
		assert.Contains(t, msg, "argv mismatch")
	}()
	_, _ = mock.Run(context.Background(), []string{"echo"}, RunOpts{})
}
