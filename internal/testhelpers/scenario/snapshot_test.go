package scenario

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCaptureHomeSnapshot_Deterministic(t *testing.T) {
	// Create a temp directory with 3 files + 1 symlink
	home := t.TempDir()

	// Create files
	if err := os.WriteFile(filepath.Join(home, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("failed to create file1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, "file2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatalf("failed to create file2: %v", err)
	}

	// Create a subdirectory with a file
	subdir := filepath.Join(home, "subdir")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "file3.txt"), []byte("content3"), 0644); err != nil {
		t.Fatalf("failed to create file3: %v", err)
	}

	// Create a symlink
	target := filepath.Join(home, "file1.txt")
	link := filepath.Join(home, "link_to_file1")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Capture twice and verify byte-identical
	snap1, err := captureHomeSnapshot(home)
	if err != nil {
		t.Fatalf("first capture failed: %v", err)
	}

	snap2, err := captureHomeSnapshot(home)
	if err != nil {
		t.Fatalf("second capture failed: %v", err)
	}

	if !bytes.Equal(snap1, snap2) {
		t.Errorf("snapshots not byte-identical:\nfirst:\n%s\nsecond:\n%s", snap1, snap2)
	}
}

func TestCaptureStateSnapshot_Missing(t *testing.T) {
	// Call captureStateSnapshot on a non-existent path
	nonExistent := filepath.Join(t.TempDir(), "nonexistent", "state.json")
	result, err := captureStateSnapshot(nonExistent)

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	expected := []byte("<no state file>")
	if !bytes.Equal(result, expected) {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCaptureStateSnapshot_SortsKeys(t *testing.T) {
	// Create a temp state file with keys in reverse order
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Write JSON with keys in reverse order (z, y, x)
	stateData := map[string]interface{}{
		"z_package": map[string]interface{}{
			"profile": "default",
		},
		"y_package": map[string]interface{}{
			"profile": "work",
		},
		"x_package": map[string]interface{}{
			"profile": "personal",
		},
	}

	data, err := json.Marshal(stateData)
	if err != nil {
		t.Fatalf("failed to marshal state: %v", err)
	}

	if err := os.WriteFile(statePath, data, 0644); err != nil {
		t.Fatalf("failed to write state file: %v", err)
	}

	// Capture the snapshot
	result, err := captureStateSnapshot(statePath)
	if err != nil {
		t.Fatalf("capture failed: %v", err)
	}

	// Verify keys are sorted by checking the order in the output
	resultStr := string(result)
	xPos := bytes.Index(result, []byte("x_package"))
	yPos := bytes.Index(result, []byte("y_package"))
	zPos := bytes.Index(result, []byte("z_package"))

	if xPos == -1 || yPos == -1 || zPos == -1 {
		t.Fatalf("not all keys found in result: %s", resultStr)
	}

	if !(xPos < yPos && yPos < zPos) {
		t.Errorf("keys not sorted: x_pos=%d, y_pos=%d, z_pos=%d", xPos, yPos, zPos)
	}
}

func TestNormalizeStdout_StripsAnsi(t *testing.T) {
	// Input with ANSI color codes
	input := []byte("\x1b[32mgreen\x1b[0m text")
	expected := []byte("green text")

	result := normalizeStdout(input)

	if !bytes.Equal(result, expected) {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestExpandPlaceholders_NoMangle(t *testing.T) {
	// Input without placeholders
	input := []byte("this is plain text without placeholders")
	home := "/home/user"
	repo := "/home/user/dotfiles"

	result := expandPlaceholders(input, home, repo)

	if !bytes.Equal(result, input) {
		t.Errorf("expected %q, got %q", input, result)
	}
}

func TestNormalizeStdout_NormalizesLineEndings(t *testing.T) {
	// Input with CRLF
	input := []byte("line1\r\nline2\r\nline3")
	expected := []byte("line1\nline2\nline3")

	result := normalizeStdout(input)

	if !bytes.Equal(result, expected) {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestNormalizeStdout_TrimsTrailingWhitespace(t *testing.T) {
	// Input with trailing whitespace on lines
	input := []byte("line1  \nline2\t\nline3   ")
	expected := []byte("line1\nline2\nline3")

	result := normalizeStdout(input)

	if !bytes.Equal(result, expected) {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestNormalizeStdout_RemovesTrailingBlankLines(t *testing.T) {
	// Input with trailing blank lines
	input := []byte("line1\nline2\n\n\n")
	expected := []byte("line1\nline2")

	result := normalizeStdout(input)

	if !bytes.Equal(result, expected) {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestExpandPlaceholders_ReplacesHome(t *testing.T) {
	input := []byte("config at <HOME>/.config")
	home := "/home/user"
	repo := "/home/user/dotfiles"

	result := expandPlaceholders(input, home, repo)
	expected := []byte("config at /home/user/.config")

	if !bytes.Equal(result, expected) {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestExpandPlaceholders_ReplacesRepo(t *testing.T) {
	input := []byte("repo at <REPO>/nvim")
	home := "/home/user"
	repo := "/home/user/dotfiles"

	result := expandPlaceholders(input, home, repo)
	expected := []byte("repo at /home/user/dotfiles/nvim")

	if !bytes.Equal(result, expected) {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestExpandPlaceholders_ReplacesState(t *testing.T) {
	input := []byte("state at <STATE>")
	home := "/home/user"
	repo := "/home/user/dotfiles"

	result := expandPlaceholders(input, home, repo)
	expected := []byte("state at /home/user/.config/easyrice/state.json")

	if !bytes.Equal(result, expected) {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestDiff_Equal(t *testing.T) {
	expected := []byte("line1\nline2\nline3")
	actual := []byte("line1\nline2\nline3")

	result := diff(expected, actual)

	if result != "" {
		t.Errorf("expected empty diff, got %q", result)
	}
}

func TestDiff_Different(t *testing.T) {
	expected := []byte("line1\nline2\nline3")
	actual := []byte("line1\nmodified\nline3")

	result := diff(expected, actual)

	if result == "" {
		t.Errorf("expected non-empty diff")
	}

	// Verify diff contains expected markers
	if !bytes.Contains([]byte(result), []byte("-line2")) {
		t.Errorf("diff missing expected '-line2'")
	}
	if !bytes.Contains([]byte(result), []byte("+modified")) {
		t.Errorf("diff missing expected '+modified'")
	}
}

func TestCaptureRepoSnapshot_Deterministic(t *testing.T) {
	// Create a temp repo directory with files
	repo := t.TempDir()

	// Create files
	if err := os.WriteFile(filepath.Join(repo, "rice.toml"), []byte("[packages]"), 0644); err != nil {
		t.Fatalf("failed to create rice.toml: %v", err)
	}

	pkgDir := filepath.Join(repo, "nvim")
	if err := os.Mkdir(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create nvim dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(pkgDir, "init.lua"), []byte("-- config"), 0644); err != nil {
		t.Fatalf("failed to create init.lua: %v", err)
	}

	// Capture twice and verify byte-identical
	snap1, err := captureRepoSnapshot(repo)
	if err != nil {
		t.Fatalf("first capture failed: %v", err)
	}

	snap2, err := captureRepoSnapshot(repo)
	if err != nil {
		t.Fatalf("second capture failed: %v", err)
	}

	if !bytes.Equal(snap1, snap2) {
		t.Errorf("snapshots not byte-identical:\nfirst:\n%s\nsecond:\n%s", snap1, snap2)
	}
}

func TestCompareSnapshotFile_Missing(t *testing.T) {
	expectedFile := filepath.Join(t.TempDir(), "nonexistent.txt")
	actual := []byte("some content")
	home := "/home/user"
	repo := "/home/user/dotfiles"

	subT := &testingTWrapper{T: t}
	CompareSnapshotFile(subT, expectedFile, actual, home, repo)

	if !subT.errorfCalled {
		t.Errorf("expected CompareSnapshotFile to call t.Errorf for missing file")
	}
}

func TestCompareSnapshotFile_Match(t *testing.T) {
	tmpDir := t.TempDir()
	expectedFile := filepath.Join(tmpDir, "expected.txt")
	if err := os.WriteFile(expectedFile, []byte("expected content"), 0644); err != nil {
		t.Fatalf("failed to create expected file: %v", err)
	}

	actual := []byte("expected content")
	home := "/home/user"
	repo := "/home/user/dotfiles"

	subT := &testingTWrapper{T: t}
	CompareSnapshotFile(subT, expectedFile, actual, home, repo)

	if subT.errorfCalled {
		t.Errorf("expected CompareSnapshotFile not to call t.Errorf for matching snapshots")
	}
}

func TestCompareSnapshotFile_Mismatch(t *testing.T) {
	tmpDir := t.TempDir()
	expectedFile := filepath.Join(tmpDir, "expected.txt")
	if err := os.WriteFile(expectedFile, []byte("expected content"), 0644); err != nil {
		t.Fatalf("failed to create expected file: %v", err)
	}

	actual := []byte("actual content")
	home := "/home/user"
	repo := "/home/user/dotfiles"

	subT := &testingTWrapper{T: t}
	CompareSnapshotFile(subT, expectedFile, actual, home, repo)

	if !subT.errorfCalled {
		t.Errorf("expected CompareSnapshotFile to call t.Errorf for mismatched snapshots")
	}
}

type testingTWrapper struct {
	*testing.T
	errorfCalled bool
}

func (w *testingTWrapper) Errorf(format string, args ...interface{}) {
	w.errorfCalled = true
}
