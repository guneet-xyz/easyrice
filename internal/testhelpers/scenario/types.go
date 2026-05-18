package scenario

import (
	"io"
	"os"
	"testing"
)

// Runner is the cobra-agnostic execution callback supplied by each consuming test package.
// It receives the command args and an optional stdin reader, and returns the captured stdout
// and any error returned by the underlying command (e.g. cobra's Execute()).
type Runner func(args []string, stdin io.Reader) (stdout string, err error)

// Config is the wiring contract passed to Run by each test package.
type Config struct {
	// Runner is the cobra-agnostic execution callback.
	Runner Runner
	// Reset is called before each step to clear cobra global state.
	Reset func()
	// MockRegistry maps mock names to setup functions invoked before the scenario runs.
	MockRegistry map[string]func(testing.TB)
}

// Scenario holds the parsed contents of a scenario directory.
type Scenario struct {
	// Dir is the absolute path to the scenario directory.
	Dir string
	// Steps is the ordered list of steps to execute.
	Steps []Step
	// Mocks lists mock names to activate from the MockRegistry before any step runs.
	Mocks []string
}

// Step describes a single command invocation within a scenario.
type Step struct {
	// Name is a human-readable label shown in test output.
	Name string
	// Args is the command-line arguments passed to the Runner.
	Args []string
	// Stdin is optional text piped to the command's standard input.
	Stdin string
	// Env is a map of additional environment variables set for this step only.
	Env map[string]string
	// Mutate is an ordered list of filesystem mutations applied before the Runner is invoked.
	Mutate []MutateOp
	// Expect describes the assertions to make after the Runner returns.
	Expect Expect
}

// Expect describes the assertions made after a step executes.
type Expect struct {
	// ExitCode is the expected exit code: 0 if Runner returns nil, 1 otherwise.
	ExitCode int
	// StdoutContains is a list of substrings that must all appear in the normalized stdout.
	StdoutContains []string
	// StdoutNotContains is a list of substrings that must NOT appear in the normalized stdout.
	StdoutNotContains []string
	// StdoutEquals, if non-empty, requires the normalized stdout to match exactly.
	StdoutEquals string
	// State is a path relative to the scenario dir pointing to the expected state.json snapshot.
	State string
	// Home is a path relative to the scenario dir pointing to the expected home-tree snapshot.
	Home string
	// Repo is a path relative to the scenario dir pointing to the expected repo-tree snapshot.
	Repo string
}

// MutateOp describes a single filesystem mutation applied mid-scenario.
// Supported Op values: "remove", "write_file", "replace_symlink", "mkdir", "chmod".
type MutateOp struct {
	// Op is the operation kind. Must be one of the supported values.
	Op string
	// Path is the target path. Supports <HOME> and <REPO> placeholders.
	Path string
	// Content is the file content for write_file operations.
	Content string
	// Target is the symlink target for replace_symlink operations.
	Target string
	// Mode is the file permission for mkdir and chmod operations. Defaults to 0o755 for mkdir.
	Mode os.FileMode
}
