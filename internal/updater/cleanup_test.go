package updater

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	symlinkpkg "github.com/guneet-xyz/easyrice/internal/symlink"
)

func TestCleanupOrphanArtifacts_RemovesNewFile(t *testing.T) {
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "easyrice")
	newPath := filepath.Join(tmpDir, "easyrice.new")

	// Create binary and .new file
	if err := os.WriteFile(execPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("WriteFile execPath: %v", err)
	}
	if err := os.WriteFile(newPath, []byte("new"), 0644); err != nil {
		t.Fatalf("WriteFile newPath: %v", err)
	}

	// Verify .new exists before cleanup
	if _, err := os.Stat(newPath); err != nil {
		t.Fatal(".new file should exist before cleanup")
	}

	// Call CleanupOrphanArtifacts
	if err := CleanupOrphanArtifacts(execPath); err != nil {
		t.Fatalf("CleanupOrphanArtifacts: %v", err)
	}

	// Verify binary still exists
	if _, err := os.Stat(execPath); err != nil {
		t.Fatal("binary should still exist after cleanup")
	}

	// Verify .new is removed
	if _, err := os.Stat(newPath); err == nil {
		t.Fatal(".new file should be removed")
	}
}

func TestCleanupOrphanArtifacts_RemovesOldFile(t *testing.T) {
	// Skip on Windows (cleanup doesn't remove .old on Windows)
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows: .old removal not supported")
	}

	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "easyrice")
	oldPath := filepath.Join(tmpDir, "easyrice.old")

	// Create binary and .old file
	if err := os.WriteFile(execPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("WriteFile execPath: %v", err)
	}
	if err := os.WriteFile(oldPath, []byte("old"), 0644); err != nil {
		t.Fatalf("WriteFile oldPath: %v", err)
	}

	// Verify .old exists before cleanup
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatal(".old file should exist before cleanup")
	}

	// Call CleanupOrphanArtifacts
	if err := CleanupOrphanArtifacts(execPath); err != nil {
		t.Fatalf("CleanupOrphanArtifacts: %v", err)
	}

	// Verify binary still exists
	if _, err := os.Stat(execPath); err != nil {
		t.Fatal("binary should still exist after cleanup")
	}

	// Verify .old is removed
	if _, err := os.Stat(oldPath); err == nil {
		t.Fatal(".old file should be removed")
	}
}

func TestCleanupOrphanArtifacts_NoErrorWhenNoneExist(t *testing.T) {
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "easyrice")

	// Create only the binary, no orphans
	if err := os.WriteFile(execPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("WriteFile execPath: %v", err)
	}

	// Call CleanupOrphanArtifacts with no orphans present
	if err := CleanupOrphanArtifacts(execPath); err != nil {
		t.Fatalf("CleanupOrphanArtifacts should return nil when no orphans exist: %v", err)
	}

	// Verify binary still exists
	if _, err := os.Stat(execPath); err != nil {
		t.Fatal("binary should still exist after cleanup")
	}
}

func TestCleanupOrphanArtifacts_RemovesBoth(t *testing.T) {
	// Skip on Windows (cleanup doesn't remove .old on Windows)
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows: .old removal not supported")
	}

	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "easyrice")
	newPath := filepath.Join(tmpDir, "easyrice.new")
	oldPath := filepath.Join(tmpDir, "easyrice.old")

	// Create binary and both orphan files
	if err := os.WriteFile(execPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("WriteFile execPath: %v", err)
	}
	if err := os.WriteFile(newPath, []byte("new"), 0644); err != nil {
		t.Fatalf("WriteFile newPath: %v", err)
	}
	if err := os.WriteFile(oldPath, []byte("old"), 0644); err != nil {
		t.Fatalf("WriteFile oldPath: %v", err)
	}

	// Verify both orphans exist before cleanup
	if _, err := os.Stat(newPath); err != nil {
		t.Fatal(".new file should exist before cleanup")
	}
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatal(".old file should exist before cleanup")
	}

	// Call CleanupOrphanArtifacts
	if err := CleanupOrphanArtifacts(execPath); err != nil {
		t.Fatalf("CleanupOrphanArtifacts: %v", err)
	}

	// Verify binary still exists
	if _, err := os.Stat(execPath); err != nil {
		t.Fatal("binary should still exist after cleanup")
	}

	// Verify both orphans are removed
	if _, err := os.Stat(newPath); err == nil {
		t.Fatal(".new file should be removed")
	}
	if _, err := os.Stat(oldPath); err == nil {
		t.Fatal(".old file should be removed")
	}
}

func TestCleanupOrphanArtifacts_WithSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	realBinary := filepath.Join(tmpDir, "real_easyrice")
	symlink := filepath.Join(tmpDir, "easyrice")
	newPath := filepath.Join(tmpDir, "real_easyrice.new")

	// Create real binary
	if err := os.WriteFile(realBinary, []byte("binary"), 0755); err != nil {
		t.Fatalf("WriteFile realBinary: %v", err)
	}

	// Create symlink to real binary
	if err := symlinkpkg.CreateSymlink(realBinary, symlink); err != nil {
		t.Fatalf("CreateSymlink: %v", err)
	}

	// Create .new file next to real binary
	if err := os.WriteFile(newPath, []byte("new"), 0644); err != nil {
		t.Fatalf("WriteFile newPath: %v", err)
	}

	// Call CleanupOrphanArtifacts with symlink path
	if err := CleanupOrphanArtifacts(symlink); err != nil {
		t.Fatalf("CleanupOrphanArtifacts: %v", err)
	}

	// Verify .new is removed (should be resolved to real binary location)
	if _, err := os.Stat(newPath); err == nil {
		t.Fatal(".new file should be removed even when called with symlink")
	}
}

func TestCleanupOrphanArtifacts_BrokenSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	symlink := filepath.Join(tmpDir, "easyrice")
	nonexistent := filepath.Join(tmpDir, "nonexistent")

	// Create symlink to nonexistent target
	if err := symlinkpkg.CreateSymlink(nonexistent, symlink); err != nil {
		t.Fatalf("CreateSymlink: %v", err)
	}

	// Call CleanupOrphanArtifacts with broken symlink
	// Should not error; falls back to original path
	if err := CleanupOrphanArtifacts(symlink); err != nil {
		t.Fatalf("CleanupOrphanArtifacts should handle broken symlink gracefully: %v", err)
	}
}

func TestCleanupOrphanArtifacts_PermissionDeniedOnNew(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows: permission model differs")
	}

	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "easyrice")
	newPath := filepath.Join(tmpDir, "easyrice.new")

	// Create binary and .new file
	if err := os.WriteFile(execPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("WriteFile execPath: %v", err)
	}
	if err := os.WriteFile(newPath, []byte("new"), 0644); err != nil {
		t.Fatalf("WriteFile newPath: %v", err)
	}

	if err := os.Chmod(tmpDir, 0555); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	defer os.Chmod(tmpDir, 0755)

	err := CleanupOrphanArtifacts(execPath)
	if err == nil {
		t.Fatal("CleanupOrphanArtifacts should return error when permission denied")
	}
}

func TestCleanupOrphanArtifacts_PermissionDeniedOnOld(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows: .old removal not supported")
	}

	tmpDir := t.TempDir()
	subdir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	execPath := filepath.Join(subdir, "easyrice")
	oldPath := filepath.Join(subdir, "easyrice.old")

	// Create binary and .old file
	if err := os.WriteFile(execPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("WriteFile execPath: %v", err)
	}
	if err := os.WriteFile(oldPath, []byte("old"), 0644); err != nil {
		t.Fatalf("WriteFile oldPath: %v", err)
	}

	if err := os.Chmod(subdir, 0555); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	defer os.Chmod(subdir, 0755)

	err := CleanupOrphanArtifacts(execPath)
	if err == nil {
		t.Fatal("CleanupOrphanArtifacts should return error when permission denied on .old removal")
	}
}
