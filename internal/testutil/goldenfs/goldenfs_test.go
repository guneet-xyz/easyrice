package goldenfs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSnapshot_EmptyDir(t *testing.T) {
	root := t.TempDir()
	snap := Snapshot(t, root)
	if snap != "" {
		t.Errorf("expected empty snapshot, got: %q", snap)
	}
}

func TestSnapshot_WithFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "file1.txt"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "file2.txt"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	snap := Snapshot(t, root)
	expected := "file1.txt\nfile2.txt"
	if snap != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, snap)
	}
}

func TestSnapshot_WithSubdirs(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "subdir", "file.txt"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	snap := Snapshot(t, root)
	expected := "subdir/\nsubdir/file.txt"
	if snap != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, snap)
	}
}

func TestSnapshot_WithInternalSymlink(t *testing.T) {
	root := t.TempDir()
	targetFile := filepath.Join(root, "target.txt")
	if err := os.WriteFile(targetFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	linkPath := filepath.Join(root, "link.txt")
	if err := os.Symlink(targetFile, linkPath); err != nil {
		t.Fatal(err)
	}

	snap := Snapshot(t, root)
	expected := "link.txt -> target.txt\ntarget.txt"
	if snap != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, snap)
	}
}

func TestSnapshot_WithExternalSymlink(t *testing.T) {
	root := t.TempDir()
	externalTarget := "/tmp/external-target"

	linkPath := filepath.Join(root, "external-link")
	if err := os.Symlink(externalTarget, linkPath); err != nil {
		t.Fatal(err)
	}

	snap := Snapshot(t, root)
	expected := "external-link -> <TEMP>"
	if snap != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, snap)
	}
}

func TestSnapshot_Deterministic(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 5; i++ {
		if err := os.WriteFile(filepath.Join(root, "file"+string(rune('a'+i))+".txt"), []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	snap1 := Snapshot(t, root)
	snap2 := Snapshot(t, root)
	snap3 := Snapshot(t, root)

	if snap1 != snap2 || snap2 != snap3 {
		t.Errorf("snapshots not deterministic:\n1: %s\n2: %s\n3: %s", snap1, snap2, snap3)
	}
}

type mockT struct {
	failed bool
	logs   []string
}

func (m *mockT) Errorf(format string, args ...interface{}) {
	m.failed = true
	m.logs = append(m.logs, fmt.Sprintf(format, args...))
}

func (m *mockT) Fatalf(format string, args ...interface{}) {
	m.failed = true
	m.logs = append(m.logs, fmt.Sprintf(format, args...))
}

func (m *mockT) Helper() {}

func (m *mockT) Error(args ...interface{}) {
	m.failed = true
	m.logs = append(m.logs, fmt.Sprint(args...))
}

func (m *mockT) Fatal(args ...interface{}) {
	m.failed = true
	m.logs = append(m.logs, fmt.Sprint(args...))
}

func (m *mockT) Fail() {
	m.failed = true
}

func (m *mockT) FailNow() {
	m.failed = true
}

func (m *mockT) Log(args ...interface{}) {
	m.logs = append(m.logs, fmt.Sprint(args...))
}

func (m *mockT) Logf(format string, args ...interface{}) {
	m.logs = append(m.logs, fmt.Sprintf(format, args...))
}

func (m *mockT) Name() string {
	return "mockT"
}

func (m *mockT) Skip(args ...interface{}) {
}

func (m *mockT) Skipf(format string, args ...interface{}) {
}

func (m *mockT) SkipNow() {
}

func (m *mockT) Skipped() bool {
	return false
}

func (m *mockT) TempDir() string {
	return ""
}

func TestAssertGolden_MissingFile(t *testing.T) {
	goldenPath := filepath.Join(t.TempDir(), "missing.golden")
	got := "test content"

	mock := &mockT{}
	AssertGolden(mock, got, goldenPath)

	if !mock.failed {
		t.Errorf("expected AssertGolden to fail for missing golden file")
	}

	if len(mock.logs) == 0 {
		t.Errorf("expected error message to be logged")
	}

	if !strings.Contains(mock.logs[0], "run with -update to create") {
		t.Errorf("expected error message to contain 'run with -update to create', got: %s", mock.logs[0])
	}
}

func TestUpdate_RewritesGolden(t *testing.T) {
	if !*updateFlag {
		t.Skip("skipping update test without -update flag")
	}

	goldenDir := t.TempDir()
	goldenPath := filepath.Join(goldenDir, "test.golden")
	got := "updated content"

	AssertGolden(t, got, goldenPath)

	data, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}

	if string(data) != got {
		t.Errorf("golden file not updated correctly: expected %q, got %q", got, string(data))
	}
}

func TestSnapshotState_EmptyState(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")

	snap := SnapshotState(t, stateFile)
	expected := "{}"
	if snap != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, snap)
	}
}
