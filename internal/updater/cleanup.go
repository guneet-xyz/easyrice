package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// CleanupOrphanArtifacts removes leftover .new and .old sibling files from a binary path.
// It is idempotent: returns nil if no orphans are present.
// On non-Windows, removes both .new and .old siblings.
// On Windows, removes only .new (leaves .old hidden per go-selfupdate behavior).
// Returns a wrapped error only on permission/IO failure; os.IsNotExist is treated as success.
func CleanupOrphanArtifacts(execPath string) error {
	// Resolve symlinks; fall back to original on error (never block).
	resolved, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		resolved = execPath
	}

	// Get the base name and directory.
	dir := filepath.Dir(resolved)
	base := filepath.Base(resolved)

	// Remove .new sibling (all platforms).
	newPath := filepath.Join(dir, base+".new")
	if err := os.Remove(newPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cleanup: remove .new: %w", err)
	}

	// Remove .old sibling (non-Windows only).
	if runtime.GOOS != "windows" {
		oldPath := filepath.Join(dir, base+".old")
		if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("cleanup: remove .old: %w", err)
		}
	}

	return nil
}
