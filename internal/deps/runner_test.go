package deps

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecRunnerNonZeroExit(t *testing.T) {
	runner := &ExecRunner{}
	result, err := runner.Run(context.Background(), []string{"sh", "-c", "exit 3"}, RunOpts{CombinedOutput: true})
	require.NoError(t, err, "non-zero exit should not be an error")
	assert.Equal(t, 3, result.ExitCode)
	assert.Empty(t, result.Combined)
}

func TestExecRunnerEmptyArgv(t *testing.T) {
	runner := &ExecRunner{}
	_, err := runner.Run(context.Background(), []string{}, RunOpts{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty argv")
}

func TestExecRunnerSeparateOutput(t *testing.T) {
	runner := &ExecRunner{}
	result, err := runner.Run(
		context.Background(),
		[]string{"sh", "-c", "echo stdout; echo stderr >&2"},
		RunOpts{CombinedOutput: false},
	)
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, string(result.Stdout), "stdout")
	assert.Contains(t, string(result.Stderr), "stderr")
	assert.Empty(t, result.Combined)
}

func TestExecRunnerCombinedOutput(t *testing.T) {
	runner := &ExecRunner{}
	result, err := runner.Run(
		context.Background(),
		[]string{"sh", "-c", "echo stdout; echo stderr >&2"},
		RunOpts{CombinedOutput: true},
	)
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, string(result.Combined), "stdout")
	assert.Contains(t, string(result.Combined), "stderr")
	assert.Empty(t, result.Stdout)
	assert.Empty(t, result.Stderr)
}

func TestExecRunnerWithStdin(t *testing.T) {
	runner := &ExecRunner{}
	stdin := bytes.NewBufferString("hello")
	result, err := runner.Run(
		context.Background(),
		[]string{"cat"},
		RunOpts{Stdin: stdin, CombinedOutput: true},
	)
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, string(result.Combined), "hello")
}

func TestExecRunnerCommandNotFound(t *testing.T) {
	runner := &ExecRunner{}
	_, err := runner.Run(context.Background(), []string{"nonexistent_command_xyz"}, RunOpts{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "start")
}

func TestMockRunnerCannedResults(t *testing.T) {
	mock := &MockRunner{
		Expectations: []MockExpectation{
			{
				Argv:   []string{"echo", "hello"},
				Result: RunResult{ExitCode: 0, Stdout: []byte("hello\n")},
				Err:    nil,
			},
			{
				Argv:   []string{"false"},
				Result: RunResult{ExitCode: 1},
				Err:    nil,
			},
		},
	}

	result, err := mock.Run(context.Background(), []string{"echo", "hello"}, RunOpts{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, []byte("hello\n"), result.Stdout)

	result, err = mock.Run(context.Background(), []string{"false"}, RunOpts{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.ExitCode)
}

func TestMockRunnerArgvMismatch(t *testing.T) {
	mock := &MockRunner{
		Expectations: []MockExpectation{
			{
				Argv:   []string{"echo", "hello"},
				Result: RunResult{ExitCode: 0},
				Err:    nil,
			},
		},
	}

	assert.Panics(t, func() {
		mock.Run(context.Background(), []string{"echo", "goodbye"}, RunOpts{})
	}, "should panic on argv mismatch")
}

func TestMockRunnerExhaustedExpectations(t *testing.T) {
	mock := &MockRunner{
		Expectations: []MockExpectation{
			{
				Argv:   []string{"echo", "hello"},
				Result: RunResult{ExitCode: 0},
				Err:    nil,
			},
		},
	}

	mock.Run(context.Background(), []string{"echo", "hello"}, RunOpts{})

	assert.Panics(t, func() {
		mock.Run(context.Background(), []string{"echo", "world"}, RunOpts{})
	}, "should panic when expectations exhausted")
}

func TestMockRunnerAnyArgv(t *testing.T) {
	mock := &MockRunner{
		Expectations: []MockExpectation{
			{
				Argv:   nil, // match any argv
				Result: RunResult{ExitCode: 0, Stdout: []byte("ok")},
				Err:    nil,
			},
		},
	}

	result, err := mock.Run(context.Background(), []string{"anything", "goes", "here"}, RunOpts{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, []byte("ok"), result.Stdout)
}

func TestMockRunnerRecordsCalls(t *testing.T) {
	mock := &MockRunner{
		Expectations: []MockExpectation{
			{Argv: nil, Result: RunResult{}, Err: nil},
			{Argv: nil, Result: RunResult{}, Err: nil},
		},
	}

	opts1 := RunOpts{CombinedOutput: true}
	opts2 := RunOpts{Stdin: bytes.NewBufferString("test")}

	mock.Run(context.Background(), []string{"cmd1"}, opts1)
	mock.Run(context.Background(), []string{"cmd2"}, opts2)

	require.Len(t, mock.Calls, 2)
	assert.Equal(t, []string{"cmd1"}, mock.Calls[0].Argv)
	assert.Equal(t, opts1, mock.Calls[0].Opts)
	assert.Equal(t, []string{"cmd2"}, mock.Calls[1].Argv)
}

func TestRunShell(t *testing.T) {
	mock := &MockRunner{
		Expectations: []MockExpectation{
			{
				Argv:   []string{"sh", "-c", "echo test"},
				Result: RunResult{ExitCode: 0, Combined: []byte("test\n")},
				Err:    nil,
			},
		},
	}

	result, err := RunShell(context.Background(), mock, "echo test")
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, []byte("test\n"), result.Combined)

	require.Len(t, mock.Calls, 1)
	assert.Equal(t, []string{"sh", "-c", "echo test"}, mock.Calls[0].Argv)
	assert.True(t, mock.Calls[0].Opts.CombinedOutput)
}

func TestRunShellWithExecRunner(t *testing.T) {
	runner := &ExecRunner{}
	result, err := RunShell(context.Background(), runner, "echo hello")
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, string(result.Combined), "hello")
}
