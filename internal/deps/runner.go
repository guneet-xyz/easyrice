package deps

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

// Runner executes external commands. Use ExecRunner in production, MockRunner in tests.
type Runner interface {
	Run(ctx context.Context, argv []string, opts RunOpts) (RunResult, error)
}

// RunOpts configures command execution.
type RunOpts struct {
	Stdin          io.Reader
	Env            []string
	CombinedOutput bool // merge stdout+stderr into Combined
}

// RunResult holds the output of a completed command.
type RunResult struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
	Combined []byte
}

// ExecRunner is the production Runner that shells out via os/exec.
type ExecRunner struct{}

// Run executes argv[0] with argv[1:] as arguments.
// Returns (RunResult{ExitCode: N}, nil) for non-zero exits — NOT an error.
// Returns (_, error) only if the process could not be started.
// Refuses to run empty argv.
func (r *ExecRunner) Run(ctx context.Context, argv []string, opts RunOpts) (RunResult, error) {
	if len(argv) == 0 {
		return RunResult{}, fmt.Errorf("runner: empty argv")
	}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	if opts.Stdin != nil {
		cmd.Stdin = opts.Stdin
	}
	if len(opts.Env) > 0 {
		cmd.Env = opts.Env
	}

	var stdout, stderr bytes.Buffer
	if opts.CombinedOutput {
		combined := &bytes.Buffer{}
		cmd.Stdout = combined
		cmd.Stderr = combined
		err := cmd.Run()
		result := RunResult{Combined: combined.Bytes()}
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				result.ExitCode = exitErr.ExitCode()
				return result, nil // non-zero exit is NOT an error
			}
			return RunResult{}, fmt.Errorf("runner: start %q: %w", argv[0], err)
		}
		return result, nil
	}

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := RunResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		return RunResult{}, fmt.Errorf("runner: start %q: %w", argv[0], err)
	}
	return result, nil
}

// MockRunner is a test double that returns canned responses in order.
// It panics with a helpful message if invocations exceed expectations or argv doesn't match.
type MockRunner struct {
	// Expectations is the ordered list of expected calls and their responses.
	Expectations []MockExpectation
	// Calls records all actual invocations for assertion.
	Calls []MockCall
	idx   int
}

type MockExpectation struct {
	Argv   []string // nil = match any argv
	Result RunResult
	Err    error
}

type MockCall struct {
	Argv []string
	Opts RunOpts
}

func (m *MockRunner) Run(ctx context.Context, argv []string, opts RunOpts) (RunResult, error) {
	m.Calls = append(m.Calls, MockCall{Argv: argv, Opts: opts})
	if m.idx >= len(m.Expectations) {
		panic(fmt.Sprintf("MockRunner: unexpected call #%d with argv %v (no more expectations)", m.idx+1, argv))
	}
	exp := m.Expectations[m.idx]
	m.idx++
	if exp.Argv != nil {
		if len(exp.Argv) != len(argv) {
			panic(fmt.Sprintf("MockRunner: call #%d argv mismatch: want %v, got %v", m.idx, exp.Argv, argv))
		}
		for i := range exp.Argv {
			if exp.Argv[i] != argv[i] {
				panic(fmt.Sprintf("MockRunner: call #%d argv[%d] mismatch: want %q, got %q", m.idx, i, exp.Argv[i], argv[i]))
			}
		}
	}
	return exp.Result, exp.Err
}

// RunShell executes payload via ["sh", "-c", payload].
// This is the ONLY function that synthesizes sh -c; all other callers use explicit argv.
func RunShell(ctx context.Context, runner Runner, payload string) (RunResult, error) {
	return runner.Run(ctx, []string{"sh", "-c", payload}, RunOpts{CombinedOutput: true})
}
