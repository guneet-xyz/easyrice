//go:build !windows

// Package fsfault provides test helpers for injecting filesystem faults.
// It is fully generic and imports no easyrice production packages.
// Production seams (unexported function variables) are addressed by white-box
// tests in their own packages (e.g., internal/state, internal/installer).
package fsfault

import (
	"os"
	"syscall"
	"testing"
)

// WithOpenFile_EACCES swaps the provided function variable to return EACCES
// for the specified path. The original is restored via t.Cleanup.
func WithOpenFile_EACCES(t *testing.T, varPtr *func(string, int, os.FileMode) (*os.File, error), path string) {
	t.Helper()
	orig := *varPtr
	*varPtr = func(name string, flag int, perm os.FileMode) (*os.File, error) {
		if name == path {
			return nil, &os.PathError{Op: "open", Path: path, Err: syscall.EACCES}
		}
		return orig(name, flag, perm)
	}
	t.Cleanup(func() { *varPtr = orig })
}

// WithOpenFile_ENOSPC swaps the provided function variable to return ENOSPC
// for the specified path. The original is restored via t.Cleanup.
func WithOpenFile_ENOSPC(t *testing.T, varPtr *func(string, int, os.FileMode) (*os.File, error), path string) {
	t.Helper()
	orig := *varPtr
	*varPtr = func(name string, flag int, perm os.FileMode) (*os.File, error) {
		if name == path {
			return nil, &os.PathError{Op: "open", Path: path, Err: syscall.ENOSPC}
		}
		return orig(name, flag, perm)
	}
	t.Cleanup(func() { *varPtr = orig })
}

// WithOpenFile_EROFS swaps the provided function variable to return EROFS
// for the specified path. The original is restored via t.Cleanup.
func WithOpenFile_EROFS(t *testing.T, varPtr *func(string, int, os.FileMode) (*os.File, error), path string) {
	t.Helper()
	orig := *varPtr
	*varPtr = func(name string, flag int, perm os.FileMode) (*os.File, error) {
		if name == path {
			return nil, &os.PathError{Op: "open", Path: path, Err: syscall.EROFS}
		}
		return orig(name, flag, perm)
	}
	t.Cleanup(func() { *varPtr = orig })
}

// WithOpenFile_EINTR swaps the provided function variable to return EINTR
// for the specified path. The original is restored via t.Cleanup.
func WithOpenFile_EINTR(t *testing.T, varPtr *func(string, int, os.FileMode) (*os.File, error), path string) {
	t.Helper()
	orig := *varPtr
	*varPtr = func(name string, flag int, perm os.FileMode) (*os.File, error) {
		if name == path {
			return nil, &os.PathError{Op: "open", Path: path, Err: syscall.EINTR}
		}
		return orig(name, flag, perm)
	}
	t.Cleanup(func() { *varPtr = orig })
}

// WithRename_EACCES swaps the provided function variable to return EACCES
// for the specified old path. The original is restored via t.Cleanup.
func WithRename_EACCES(t *testing.T, varPtr *func(string, string) error, oldPath string) {
	t.Helper()
	orig := *varPtr
	*varPtr = func(oldname, newname string) error {
		if oldname == oldPath {
			return &os.PathError{Op: "rename", Path: oldPath, Err: syscall.EACCES}
		}
		return orig(oldname, newname)
	}
	t.Cleanup(func() { *varPtr = orig })
}

// WithWriteFile_PartialThenENOSPC swaps the provided function variable to write
// the first 'bytes' bytes to disk, then return ENOSPC. The original is restored
// via t.Cleanup.
func WithWriteFile_PartialThenENOSPC(t *testing.T, varPtr *func(string, []byte, os.FileMode) error, path string, bytes int) {
	t.Helper()
	orig := *varPtr
	*varPtr = func(name string, data []byte, perm os.FileMode) error {
		if name == path {
			// Write partial data to disk
			if bytes > 0 && bytes < len(data) {
				if err := os.WriteFile(name, data[:bytes], perm); err != nil {
					return err
				}
			}
			return &os.PathError{Op: "write", Path: path, Err: syscall.ENOSPC}
		}
		return orig(name, data, perm)
	}
	t.Cleanup(func() { *varPtr = orig })
}

// WithSymlink_FailAfterN swaps the provided function variable to succeed for
// the first n calls, then fail with EACCES. The original is restored via t.Cleanup.
func WithSymlink_FailAfterN(t *testing.T, varPtr *func(string, string) error, n int) {
	t.Helper()
	orig := *varPtr
	counter := 0
	*varPtr = func(source, target string) error {
		if counter < n {
			counter++
			return orig(source, target)
		}
		return &os.PathError{Op: "symlink", Path: target, Err: syscall.EACCES}
	}
	t.Cleanup(func() { *varPtr = orig })
}

// WithUnreadableDir makes the specified directory unreadable via chmod 0000.
// The original permissions are restored via t.Cleanup.
// Skips the test if running as root (os.Geteuid() == 0).
func WithUnreadableDir(t *testing.T, path string) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("requires non-root")
	}
	// Save original permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat %s: %v", path, err)
	}
	origPerm := info.Mode().Perm()

	// Make unreadable
	if err := os.Chmod(path, 0000); err != nil {
		t.Fatalf("failed to chmod %s to 0000: %v", path, err)
	}

	// Restore on cleanup
	t.Cleanup(func() {
		if err := os.Chmod(path, origPerm); err != nil {
			t.Logf("warning: failed to restore permissions on %s: %v", path, err)
		}
	})
}
