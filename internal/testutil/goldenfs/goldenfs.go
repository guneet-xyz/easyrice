package goldenfs

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/guneet-xyz/easyrice/internal/state"
)

var updateFlag = flag.Bool("update", false, "update golden files")

// Snapshot walks root and returns a deterministic text representation.
// Format: relative path + type (file/dir/symlink) + symlink target (relative if inside root, <TEMP> if outside).
func Snapshot(t *testing.T, root string) string {
	t.Helper()

	var entries []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		if rel == "." {
			return nil
		}

		var line string
		if d.IsDir() {
			line = fmt.Sprintf("%s/", rel)
		} else if d.Type()&fs.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}

			normalizedTarget := normalizeSymlinkTarget(target, root)
			line = fmt.Sprintf("%s -> %s", rel, normalizedTarget)
		} else {
			line = rel
		}

		entries = append(entries, line)
		return nil
	})

	if err != nil {
		t.Fatalf("failed to walk %s: %v", root, err)
	}

	sort.Strings(entries)

	return strings.Join(entries, "\n")
}

// normalizeSymlinkTarget converts absolute symlink targets to relative (if inside root)
// or <TEMP> (if outside root).
func normalizeSymlinkTarget(target, root string) string {
	if !filepath.IsAbs(target) {
		return target
	}

	rel, err := filepath.Rel(root, target)
	if err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}

	return "<TEMP>"
}

// AssertGolden compares got to the golden file at goldenPath.
// If -update flag is set, rewrites the golden file.
func AssertGolden(t interface {
	Helper()
	Fatalf(string, ...interface{})
	Errorf(string, ...interface{})
}, got string, goldenPath string) {
	t.Helper()

	if *updateFlag {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0755); err != nil {
			t.Fatalf("failed to create golden dir: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0644); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}
		return
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("golden file not found at %s: run with -update to create", goldenPath)
		}
		t.Fatalf("failed to read golden file: %v", err)
	}

	if got != string(expected) {
		t.Errorf("snapshot mismatch\nexpected:\n%s\n\ngot:\n%s", string(expected), got)
	}
}

// SnapshotState loads state from statePath and returns a stable JSON representation.
func SnapshotState(t *testing.T, statePath string) string {
	t.Helper()

	s, err := state.Load(statePath)
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal state: %v", err)
	}

	return string(data)
}
