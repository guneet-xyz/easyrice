package updater

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCleanupOrphanArtifacts(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create test files
	execPath := filepath.Join(tmpDir, "easyrice")
	newPath := filepath.Join(tmpDir, "easyrice.new")
	oldPath := filepath.Join(tmpDir, "easyrice.old")

	// Create the binary and orphan files
	if err := os.WriteFile(execPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("WriteFile execPath: %v", err)
	}
	if err := os.WriteFile(newPath, []byte("new"), 0644); err != nil {
		t.Fatalf("WriteFile newPath: %v", err)
	}
	if err := os.WriteFile(oldPath, []byte("old"), 0644); err != nil {
		t.Fatalf("WriteFile oldPath: %v", err)
	}

	// Verify files exist before cleanup
	if _, err := os.Stat(execPath); err != nil {
		t.Fatal("execPath should exist before cleanup")
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatal("newPath should exist before cleanup")
	}
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatal("oldPath should exist before cleanup")
	}

	// Call CleanupOrphanArtifacts
	if err := CleanupOrphanArtifacts(execPath); err != nil {
		t.Fatalf("CleanupOrphanArtifacts: %v", err)
	}

	// Verify binary still exists
	if _, err := os.Stat(execPath); err != nil {
		t.Fatal("execPath should still exist after cleanup")
	}

	// Verify .new is removed
	if _, err := os.Stat(newPath); err == nil {
		t.Fatal("newPath should be removed")
	}

	// Verify .old is removed (on non-Windows)
	if runtime.GOOS != "windows" {
		if _, err := os.Stat(oldPath); err == nil {
			t.Fatal("oldPath should be removed on non-Windows")
		}
	}

	// Test idempotency: call again with no orphans
	if err := CleanupOrphanArtifacts(execPath); err != nil {
		t.Fatalf("CleanupOrphanArtifacts (idempotent): %v", err)
	}
}
