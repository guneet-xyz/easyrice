package symlink

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateSymlink_HappyPath(t *testing.T) {
	tmpDir := t.TempDir()
	source := "source.txt"
	target := filepath.Join(tmpDir, "link.txt")

	err := CreateSymlink(source, target)
	if err != nil {
		t.Fatalf("CreateSymlink failed: %v", err)
	}

	// Verify symlink exists and points to source
	dest, err := os.Readlink(target)
	if err != nil {
		t.Fatalf("Readlink failed: %v", err)
	}
	if dest != source {
		t.Errorf("symlink points to %q, want %q", dest, source)
	}
}

func TestCreateSymlink_CreatesParentDirs(t *testing.T) {
	tmpDir := t.TempDir()
	source := "source.txt"
	target := filepath.Join(tmpDir, "a", "b", "c", "link.txt")

	err := CreateSymlink(source, target)
	if err != nil {
		t.Fatalf("CreateSymlink failed: %v", err)
	}

	// Verify symlink exists
	dest, err := os.Readlink(target)
	if err != nil {
		t.Fatalf("Readlink failed: %v", err)
	}
	if dest != source {
		t.Errorf("symlink points to %q, want %q", dest, source)
	}

	// Verify parent directories were created
	if _, err := os.Stat(filepath.Dir(target)); err != nil {
		t.Errorf("parent directory not created: %v", err)
	}
}

func TestCreateSymlink_FailsIfTargetExists_RegularFile(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "existing.txt")

	// Create a regular file at target
	if err := os.WriteFile(target, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	err := CreateSymlink("source.txt", target)
	if err == nil {
		t.Fatal("CreateSymlink should fail when target is a regular file")
	}
}

func TestCreateSymlink_FailsIfTargetExists_Symlink(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "existing_link.txt")

	// Create a symlink at target
	if err := os.Symlink("old_source.txt", target); err != nil {
		t.Fatalf("failed to create test symlink: %v", err)
	}

	err := CreateSymlink("new_source.txt", target)
	if err == nil {
		t.Fatal("CreateSymlink should fail when target is an existing symlink")
	}
}

func TestRemoveSymlink_HappyPath(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "link.txt")

	// Create a symlink
	if err := os.Symlink("source.txt", target); err != nil {
		t.Fatalf("failed to create test symlink: %v", err)
	}

	err := RemoveSymlink(target)
	if err != nil {
		t.Fatalf("RemoveSymlink failed: %v", err)
	}

	// Verify symlink is gone
	if _, err := os.Lstat(target); err == nil {
		t.Fatal("symlink still exists after removal")
	}
}

func TestRemoveSymlink_FailsIfTargetDoesNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "nonexistent.txt")

	err := RemoveSymlink(target)
	if err == nil {
		t.Fatal("RemoveSymlink should fail when target does not exist")
	}
}

func TestRemoveSymlink_FailsIfTargetIsRegularFile(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "regular.txt")

	// Create a regular file
	if err := os.WriteFile(target, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	err := RemoveSymlink(target)
	if err == nil {
		t.Fatal("RemoveSymlink should fail when target is not a symlink")
	}
}

func TestIsSymlinkTo_ReturnsTrue_CorrectSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	source := "source.txt"
	target := filepath.Join(tmpDir, "link.txt")

	// Create a symlink
	if err := os.Symlink(source, target); err != nil {
		t.Fatalf("failed to create test symlink: %v", err)
	}

	ok, err := IsSymlinkTo(target, source)
	if err != nil {
		t.Fatalf("IsSymlinkTo failed: %v", err)
	}
	if !ok {
		t.Error("IsSymlinkTo returned false, want true")
	}
}

func TestIsSymlinkTo_ReturnsFalse_WrongTarget(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "link.txt")

	// Create a symlink pointing to source1
	if err := os.Symlink("source1.txt", target); err != nil {
		t.Fatalf("failed to create test symlink: %v", err)
	}

	// Check if it points to source2
	ok, err := IsSymlinkTo(target, "source2.txt")
	if err != nil {
		t.Fatalf("IsSymlinkTo failed: %v", err)
	}
	if ok {
		t.Error("IsSymlinkTo returned true, want false")
	}
}

func TestIsSymlinkTo_ReturnsFalse_MissingPath(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "nonexistent.txt")

	ok, err := IsSymlinkTo(target, "source.txt")
	if err != nil {
		t.Fatalf("IsSymlinkTo failed: %v", err)
	}
	if ok {
		t.Error("IsSymlinkTo returned true for missing path, want false")
	}
}

func TestCreateSymlink_ErrorWhenParentDirPermissionDenied(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skipping permission test in CI environment")
	}

	tmpDir := t.TempDir()
	parentDir := filepath.Join(tmpDir, "restricted")

	if err := os.Mkdir(parentDir, 0755); err != nil {
		t.Fatalf("failed to create parent directory: %v", err)
	}
	if err := os.Chmod(parentDir, 0555); err != nil {
		t.Fatalf("failed to remove write permission: %v", err)
	}
	t.Cleanup(func() {
		os.Chmod(parentDir, 0755)
	})

	target := filepath.Join(parentDir, "subdir", "link.txt")
	err := CreateSymlink("source.txt", target)
	if err == nil {
		t.Fatal("CreateSymlink should fail when parent directory is not writable")
	}
}

func TestCreateSymlink_ErrorWhenCheckTargetFails(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skipping permission test in CI environment")
	}

	tmpDir := t.TempDir()
	parentDir := filepath.Join(tmpDir, "restricted")

	if err := os.Mkdir(parentDir, 0755); err != nil {
		t.Fatalf("failed to create parent directory: %v", err)
	}
	if err := os.Chmod(parentDir, 0000); err != nil {
		t.Fatalf("failed to remove all permissions: %v", err)
	}
	t.Cleanup(func() {
		os.Chmod(parentDir, 0755)
	})

	target := filepath.Join(parentDir, "link.txt")
	err := CreateSymlink("source.txt", target)
	if err == nil {
		t.Fatal("CreateSymlink should fail when unable to check target")
	}
}

func TestIsSymlinkTo_ReturnsFalseForRegularFile(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "regular.txt")

	if err := os.WriteFile(target, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	ok, err := IsSymlinkTo(target, "source.txt")
	if err != nil {
		t.Fatalf("IsSymlinkTo failed: %v", err)
	}
	if ok {
		t.Error("IsSymlinkTo returned true for regular file, want false")
	}
}

func TestReadLink_ErrorOnNonSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "regular.txt")

	if err := os.WriteFile(target, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err := ReadLink(target)
	if err == nil {
		t.Fatal("ReadLink should fail when target is not a symlink")
	}
}

func TestReadLink_HappyPath(t *testing.T) {
	tmpDir := t.TempDir()
	source := "source.txt"
	target := filepath.Join(tmpDir, "link.txt")

	if err := os.Symlink(source, target); err != nil {
		t.Fatalf("failed to create test symlink: %v", err)
	}

	dest, err := ReadLink(target)
	if err != nil {
		t.Fatalf("ReadLink failed: %v", err)
	}
	if dest != source {
		t.Errorf("ReadLink returned %q, want %q", dest, source)
	}
}

func TestReadLink_ErrorOnNonExistentPath(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "nonexistent.txt")

	_, err := ReadLink(target)
	if err == nil {
		t.Fatal("ReadLink should fail when path does not exist")
	}
}
