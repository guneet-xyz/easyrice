package git

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/guneet/easyrice/apps/cli/internal/log"
)

// run executes a git command in the given directory and returns any error.
// Stderr output is included in the error message for debuggability.
func run(dir string, args ...string) error {
	log.Get().Log(log.TraceLevel, "exec", "cmd", "git "+strings.Join(args, " "), "dir", dir)

	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %w\n%s", args[0], err, output)
	}

	trimmed := strings.TrimSpace(string(output))
	if trimmed != "" {
		log.Get().Log(log.TraceLevel, "exec ok", "output", trimmed)
	}

	return nil
}

// Init initializes a new git repository in the given directory.
func Init(dir string) error {
	return run(dir, "init")
}

// Add stages the given paths in the repository at dir.
func Add(dir string, paths ...string) error {
	args := append([]string{"add"}, paths...)
	return run(dir, args...)
}

// Commit creates a commit with the given message in the repository at dir.
func Commit(dir string, message string) error {
	return run(dir, "commit", "-m", message)
}
