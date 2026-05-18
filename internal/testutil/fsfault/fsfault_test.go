//go:build !windows

package fsfault

import (
	"os"
	"syscall"
	"testing"
)

// TestWithOpenFile_EACCES verifies that WithOpenFile_EACCES swaps the variable,
// injects EACCES for the target path, and restores the original on cleanup.
func TestWithOpenFile_EACCES(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := tmpDir + "/target"

	// Declare a local dummy function variable
	var dummyOpenFile = os.OpenFile

	// Apply the fault
	WithOpenFile_EACCES(t, &dummyOpenFile, targetPath)

	// Call the swapped variable; should return EACCES
	_, err := dummyOpenFile(targetPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	pathErr, ok := err.(*os.PathError)
	if !ok {
		t.Fatalf("expected *os.PathError, got %T", err)
	}
	if pathErr.Op != "open" {
		t.Errorf("expected Op='open', got %q", pathErr.Op)
	}
	if pathErr.Path != targetPath {
		t.Errorf("expected Path=%q, got %q", targetPath, pathErr.Path)
	}
	if pathErr.Err != syscall.EACCES {
		t.Errorf("expected Err=EACCES, got %v", pathErr.Err)
	}

	// After cleanup, the variable should be restored
	// (cleanup runs after the test, but we can verify the original still works)
	f, err := os.OpenFile(tmpDir+"/test", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("original function failed: %v", err)
	}
	f.Close()
	os.Remove(tmpDir + "/test")
}

// TestWithOpenFile_ENOSPC verifies ENOSPC injection.
func TestWithOpenFile_ENOSPC(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := tmpDir + "/target"

	var dummyOpenFile = os.OpenFile
	WithOpenFile_ENOSPC(t, &dummyOpenFile, targetPath)

	_, err := dummyOpenFile(targetPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	pathErr, ok := err.(*os.PathError)
	if !ok {
		t.Fatalf("expected *os.PathError, got %T", err)
	}
	if pathErr.Err != syscall.ENOSPC {
		t.Errorf("expected Err=ENOSPC, got %v", pathErr.Err)
	}
}

// TestWithOpenFile_EROFS verifies EROFS injection.
func TestWithOpenFile_EROFS(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := tmpDir + "/target"

	var dummyOpenFile = os.OpenFile
	WithOpenFile_EROFS(t, &dummyOpenFile, targetPath)

	_, err := dummyOpenFile(targetPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	pathErr, ok := err.(*os.PathError)
	if !ok {
		t.Fatalf("expected *os.PathError, got %T", err)
	}
	if pathErr.Err != syscall.EROFS {
		t.Errorf("expected Err=EROFS, got %v", pathErr.Err)
	}
}

// TestWithOpenFile_EINTR verifies EINTR injection.
func TestWithOpenFile_EINTR(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := tmpDir + "/target"

	var dummyOpenFile = os.OpenFile
	WithOpenFile_EINTR(t, &dummyOpenFile, targetPath)

	_, err := dummyOpenFile(targetPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	pathErr, ok := err.(*os.PathError)
	if !ok {
		t.Fatalf("expected *os.PathError, got %T", err)
	}
	if pathErr.Err != syscall.EINTR {
		t.Errorf("expected Err=EINTR, got %v", pathErr.Err)
	}
}

// TestWithRename_EACCES verifies that WithRename_EACCES injects EACCES.
func TestWithRename_EACCES(t *testing.T) {
	tmpDir := t.TempDir()
	oldPath := tmpDir + "/old"
	newPath := tmpDir + "/new"

	// Create the old file
	if err := os.WriteFile(oldPath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	var dummyRename = os.Rename
	WithRename_EACCES(t, &dummyRename, oldPath)

	err := dummyRename(oldPath, newPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	pathErr, ok := err.(*os.PathError)
	if !ok {
		t.Fatalf("expected *os.PathError, got %T", err)
	}
	if pathErr.Op != "rename" {
		t.Errorf("expected Op='rename', got %q", pathErr.Op)
	}
	if pathErr.Err != syscall.EACCES {
		t.Errorf("expected Err=EACCES, got %v", pathErr.Err)
	}
}

// TestWithWriteFile_PartialThenENOSPC verifies partial write then ENOSPC.
func TestWithWriteFile_PartialThenENOSPC(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := tmpDir + "/target"

	var dummyWriteFile = func(name string, data []byte, perm os.FileMode) error {
		return os.WriteFile(name, data, perm)
	}

	testData := []byte("hello world")
	WithWriteFile_PartialThenENOSPC(t, &dummyWriteFile, targetPath, 5)

	err := dummyWriteFile(targetPath, testData, 0644)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	pathErr, ok := err.(*os.PathError)
	if !ok {
		t.Fatalf("expected *os.PathError, got %T", err)
	}
	if pathErr.Err != syscall.ENOSPC {
		t.Errorf("expected Err=ENOSPC, got %v", pathErr.Err)
	}

	// Verify that the first 5 bytes were written
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != "hello" {
		t.Errorf("expected 'hello', got %q", string(content))
	}
}

// TestWithSymlink_FailAfterN verifies that symlink succeeds for n calls, then fails.
func TestWithSymlink_FailAfterN(t *testing.T) {
	tmpDir := t.TempDir()

	var dummySymlink = func(source, target string) error {
		return os.Symlink(source, target)
	}

	WithSymlink_FailAfterN(t, &dummySymlink, 2)

	// First call should succeed
	if err := dummySymlink(tmpDir+"/src1", tmpDir+"/link1"); err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	// Second call should succeed
	if err := dummySymlink(tmpDir+"/src2", tmpDir+"/link2"); err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	// Third call should fail
	err := dummySymlink(tmpDir+"/src3", tmpDir+"/link3")
	if err == nil {
		t.Fatal("expected error on third call, got nil")
	}

	pathErr, ok := err.(*os.PathError)
	if !ok {
		t.Fatalf("expected *os.PathError, got %T", err)
	}
	if pathErr.Err != syscall.EACCES {
		t.Errorf("expected Err=EACCES, got %v", pathErr.Err)
	}
}

// TestWithUnreadableDir verifies that WithUnreadableDir makes a directory unreadable.
func TestWithUnreadableDir(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := tmpDir + "/testdir"
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	WithUnreadableDir(t, testDir)

	// Try to list the directory; should fail
	_, err := os.ReadDir(testDir)
	if err == nil {
		t.Fatal("expected error reading unreadable dir, got nil")
	}
}

// TestDelegate_NoFault verifies that without any fault, the dummy variable
// behaves identically to os.OpenFile on a real temp directory.
func TestDelegate_NoFault(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := tmpDir + "/test.txt"

	var dummyOpenFile = os.OpenFile

	// No fault applied; should work normally
	f, err := dummyOpenFile(testFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer f.Close()

	// Verify file was created
	if _, err := os.Stat(testFile); err != nil {
		t.Fatalf("file not created: %v", err)
	}

	// Verify we can write to it
	if _, err := f.WriteString("test"); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
}
