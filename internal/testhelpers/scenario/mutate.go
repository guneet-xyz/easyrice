package scenario

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/guneet-xyz/easyrice/internal/symlink"
)

// applyMutate applies a single filesystem mutation to the scenario environment.
// It supports the following operations:
//   - remove: delete a file or directory tree
//   - write_file: create or overwrite a file with content
//   - replace_symlink: remove a symlink and create a new one
//   - mkdir: create a directory with optional mode
//   - chmod: change file permissions
//
// All paths support <HOME> and <REPO> placeholders which are expanded before
// the operation is applied. Paths are validated to ensure they remain within
// the home or repo directories (containment check).
func applyMutate(home, repo string, op MutateOp) error {
	// Expand placeholders in the path
	path := op.Path
	path = strings.ReplaceAll(path, "<HOME>", home)
	path = strings.ReplaceAll(path, "<REPO>", repo)

	// Containment check: ensure path is within home or repo
	if !isWithinSandbox(path, home, repo) {
		return fmt.Errorf("mutate %s: path %q is outside sandbox (home=%s, repo=%s)", op.Op, path, home, repo)
	}

	switch op.Op {
	case "remove":
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("mutate remove: %w", err)
		}

	case "write_file":
		// Create parent directories
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mutate write_file: %w", err)
		}
		// Write the file
		if err := os.WriteFile(path, []byte(op.Content), 0o644); err != nil {
			return fmt.Errorf("mutate write_file: %w", err)
		}

	case "replace_symlink":
		// Expand placeholders in target as well
		target := op.Target
		target = strings.ReplaceAll(target, "<HOME>", home)
		target = strings.ReplaceAll(target, "<REPO>", repo)

		// Remove the existing symlink
		if err := symlink.RemoveSymlink(path); err != nil {
			return fmt.Errorf("mutate replace_symlink: %w", err)
		}
		// Create the new symlink
		if err := symlink.CreateSymlink(target, path); err != nil {
			return fmt.Errorf("mutate replace_symlink: %w", err)
		}

	case "mkdir":
		mode := op.Mode
		if mode == 0 {
			mode = 0o755
		}
		if err := os.MkdirAll(path, mode); err != nil {
			return fmt.Errorf("mutate mkdir: %w", err)
		}

	case "chmod":
		if err := os.Chmod(path, op.Mode); err != nil {
			return fmt.Errorf("mutate chmod: %w", err)
		}

	default:
		return fmt.Errorf("%w: unknown mutate op %q", ErrInvalidYAML, op.Op)
	}

	return nil
}

// isWithinSandbox checks if a path is within the home or repo directories.
// It resolves symlinks on all paths before comparing to handle cases like
// macOS /tmp → /private/tmp.
func isWithinSandbox(path, home, repo string) bool {
	// Resolve symlinks on all paths
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		// If we can't resolve the path (e.g., it doesn't exist yet),
		// check the parent directory
		parent := filepath.Dir(path)
		if parent != path {
			resolvedPath, err = filepath.EvalSymlinks(parent)
			if err != nil {
				// Can't resolve parent either; use the path as-is
				resolvedPath = path
			}
		} else {
			// Can't resolve; use as-is
			resolvedPath = path
		}
	}

	resolvedHome, err := filepath.EvalSymlinks(home)
	if err != nil {
		resolvedHome = home
	}

	resolvedRepo, err := filepath.EvalSymlinks(repo)
	if err != nil {
		resolvedRepo = repo
	}

	// Check if path is within home or repo
	return strings.HasPrefix(resolvedPath, resolvedHome) ||
		strings.HasPrefix(resolvedPath, resolvedRepo)
}
