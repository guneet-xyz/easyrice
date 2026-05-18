package scenario

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// captureHomeSnapshot walks the home directory and returns a deterministic
// snapshot of its structure. It resolves symlinks in the home path once
// (to handle macOS /tmp → /private/tmp), skips .git/ subtrees, and formats
// each entry as:
//   - Regular file: "<rel> [file]\n"
//   - Directory: "<rel> [dir]\n"
//   - Symlink: "<rel> -> <target>\n"
func captureHomeSnapshot(home string) ([]byte, error) {
	// Resolve symlinks in the home path once (handles macOS /tmp → /private/tmp)
	resolvedHome, err := filepath.EvalSymlinks(home)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve home symlinks: %w", err)
	}

	var entries []string

	// Walk the directory tree
	err = filepath.WalkDir(resolvedHome, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip .git/ subtrees entirely
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}

		// Get relative path
		rel, err := filepath.Rel(resolvedHome, path)
		if err != nil {
			return err
		}

		// Skip the root entry itself
		if rel == "." {
			return nil
		}

		// Format the entry
		if d.IsDir() {
			entries = append(entries, rel+" [dir]")
		} else if d.Type()&os.ModeSymlink != 0 {
			// Symlink: read the target
			target, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("failed to read symlink %s: %w", path, err)
			}
			entries = append(entries, rel+" -> "+target)
		} else {
			// Regular file
			entries = append(entries, rel+" [file]")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort entries for determinism (WalkDir is already sorted, but be explicit)
	sort.Strings(entries)

	// Build output
	var buf bytes.Buffer
	for _, entry := range entries {
		buf.WriteString(entry)
		buf.WriteString("\n")
	}

	return buf.Bytes(), nil
}

// captureRepoSnapshot walks the repo directory and returns a deterministic
// snapshot of its structure. Same format as captureHomeSnapshot.
func captureRepoSnapshot(repoRoot string) ([]byte, error) {
	// Resolve symlinks in the repo path once
	resolvedRepo, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve repo symlinks: %w", err)
	}

	var entries []string

	// Walk the directory tree
	err = filepath.WalkDir(resolvedRepo, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip .git/ subtrees entirely
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}

		// Get relative path
		rel, err := filepath.Rel(resolvedRepo, path)
		if err != nil {
			return err
		}

		// Skip the root entry itself
		if rel == "." {
			return nil
		}

		// Format the entry
		if d.IsDir() {
			entries = append(entries, rel+" [dir]")
		} else if d.Type()&os.ModeSymlink != 0 {
			// Symlink: read the target
			target, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("failed to read symlink %s: %w", path, err)
			}
			entries = append(entries, rel+" -> "+target)
		} else {
			// Regular file
			entries = append(entries, rel+" [file]")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort entries for determinism
	sort.Strings(entries)

	// Build output
	var buf bytes.Buffer
	for _, entry := range entries {
		buf.WriteString(entry)
		buf.WriteString("\n")
	}

	return buf.Bytes(), nil
}

// captureStateSnapshot reads the state.json file and returns a deterministic
// JSON representation with sorted keys. If the file does not exist, returns
// []byte("<no state file>") with nil error.
func captureStateSnapshot(statePath string) ([]byte, error) {
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []byte("<no state file>"), nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	// Unmarshal into a generic map to ensure sorted keys on re-marshal
	var state map[string]interface{}
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Normalize volatile fields: replace per-package "installed_at" timestamps with a
	// stable placeholder so snapshots are comparable across runs.
	for _, v := range state {
		pkg, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		if _, present := pkg["installed_at"]; present {
			pkg["installed_at"] = "<IGNORE>"
		}
	}

	// Re-marshal with sorted keys and 2-space indent. Disable HTML escaping so
	// placeholder tokens like <IGNORE> survive the round-trip verbatim.
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(state); err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}
	normalized := bytes.TrimRight(out.Bytes(), "\n")

	return normalized, nil
}

// normalizeStdout removes ANSI escape sequences, normalizes line endings,
// trims trailing whitespace, and removes trailing blank lines.
func normalizeStdout(raw []byte) []byte {
	// Strip ANSI escape sequences: \x1b\[[0-9;]*m
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	stripped := ansiRegex.ReplaceAll(raw, []byte{})

	// Normalize CRLF → LF
	normalized := bytes.ReplaceAll(stripped, []byte("\r\n"), []byte("\n"))

	// Split into lines
	lines := bytes.Split(normalized, []byte("\n"))

	// Trim trailing whitespace on each line
	for i, line := range lines {
		lines[i] = bytes.TrimRight(line, " \t")
	}

	// Remove trailing blank lines
	for len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}

	// Rejoin
	return bytes.Join(lines, []byte("\n"))
}

// expandPlaceholders replaces literal placeholders in the expected content:
//   - <HOME> → home
//   - <REPO> → repo
//   - <STATE> → filepath.Join(home, ".config", "easyrice", "state.json")
func expandPlaceholders(expected []byte, home, repo string) []byte {
	result := expected
	result = bytes.ReplaceAll(result, []byte("<HOME>"), []byte(home))
	result = bytes.ReplaceAll(result, []byte("<REPO>"), []byte(repo))

	statePath := filepath.Join(home, ".config", "easyrice", "state.json")
	result = bytes.ReplaceAll(result, []byte("<STATE>"), []byte(statePath))

	return result
}

// diff compares expected and actual byte slices and returns a unified-diff-style
// output. If they are equal, returns an empty string. Otherwise, returns lines
// prefixed with "-" (expected only), "+" (actual only), or " " (common).
func diff(expected, actual []byte) string {
	if bytes.Equal(expected, actual) {
		return ""
	}

	expectedLines := strings.Split(string(expected), "\n")
	actualLines := strings.Split(string(actual), "\n")

	// Simple line-by-line diff
	var result strings.Builder
	maxLines := len(expectedLines)
	if len(actualLines) > maxLines {
		maxLines = len(actualLines)
	}

	for i := 0; i < maxLines; i++ {
		var expLine, actLine string
		if i < len(expectedLines) {
			expLine = expectedLines[i]
		}
		if i < len(actualLines) {
			actLine = actualLines[i]
		}

		if expLine == actLine {
			result.WriteString(" " + expLine + "\n")
		} else {
			if i < len(expectedLines) {
				result.WriteString("-" + expLine + "\n")
			}
			if i < len(actualLines) {
				result.WriteString("+" + actLine + "\n")
			}
		}
	}

	return result.String()
}

// CompareSnapshotFile reads the expected snapshot file, expands placeholders,
// and compares it with the actual bytes. If they differ, calls t.Errorf with
// the diff. If the expected file is missing, calls t.Errorf with ErrMissingExpected.
func CompareSnapshotFile(t testing.TB, expectedFile string, actual []byte, home, repo string) {
	t.Helper()

	// Read the expected file
	expected, err := os.ReadFile(expectedFile)
	if err != nil {
		if os.IsNotExist(err) {
			t.Errorf("%v: %s", ErrMissingExpected, expectedFile)
			return
		}
		t.Errorf("failed to read expected snapshot file %s: %v", expectedFile, err)
		return
	}

	// Expand placeholders
	expected = expandPlaceholders(expected, home, repo)

	// Compare
	diffOutput := diff(expected, actual)
	if diffOutput != "" {
		t.Errorf("snapshot mismatch for %s:\n%s", expectedFile, diffOutput)
	}
}
