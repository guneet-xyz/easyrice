package symlink

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// CreateSymlink creates a symlink at target pointing to source.
// Creates parent directories of target if they don't exist.
// Returns error if target already exists (any kind).
//
// Note: On Windows, os.Symlink requires Developer Mode or Administrator privileges.
// This function uses pure Go os.Symlink and does not perform runtime checks for
// Windows Developer Mode — that is the responsibility of the doctor package.
func CreateSymlink(source, target string) error {
	// Check if target already exists
	_, err := os.Lstat(target)
	if err == nil {
		// target exists
		return fmt.Errorf("target already exists: %s", target)
	}
	if !errors.Is(err, os.ErrNotExist) {
		// Some other error (permission denied, etc.)
		return fmt.Errorf("failed to check target: %w", err)
	}

	// Create parent directories
	targetDir := filepath.Dir(target)
	if targetDir != "." && targetDir != "" {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed to create parent directories: %w", err)
		}
	}

	// Create the symlink
	if err := os.Symlink(source, target); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// RemoveSymlink removes the symlink at target.
// Returns error if target does not exist or is not a symlink.
func RemoveSymlink(target string) error {
	// Check if target exists and is a symlink
	fi, err := os.Lstat(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("target does not exist: %s", target)
		}
		return fmt.Errorf("failed to check target: %w", err)
	}

	// Verify it's a symlink
	if fi.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("target is not a symlink: %s", target)
	}

	if err := os.Remove(target); err != nil {
		return fmt.Errorf("failed to remove symlink: %w", err)
	}

	return nil
}

// IsSymlinkTo returns true if target is a symlink pointing to source.
// Returns false (not error) if target doesn't exist or is not a symlink.
func IsSymlinkTo(target, source string) (bool, error) {
	// Check if target exists and is a symlink
	fi, err := os.Lstat(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check target: %w", err)
	}

	// Check if it's a symlink
	if fi.Mode()&os.ModeSymlink == 0 {
		return false, nil
	}

	// Read the symlink destination
	dest, err := os.Readlink(target)
	if err != nil {
		return false, fmt.Errorf("failed to read symlink: %w", err)
	}

	return dest == source, nil
}

// ReadLink returns the destination of the symlink at path.
func ReadLink(path string) (string, error) {
	dest, err := os.Readlink(path)
	if err != nil {
		return "", fmt.Errorf("failed to read symlink: %w", err)
	}
	return dest, nil
}
