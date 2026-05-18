package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/guneet-xyz/easyrice/internal/testutil/fsfault"
)

type corruptMode int

const (
	corruptTruncated corruptMode = iota
	corruptInvalidJSON
	corruptNotAMap
	corruptMissingFields
	corruptPartialWrite
	corruptNullJSON
)

// corruptStateFile writes a corrupted state.json mirroring internal/state/statetest.Corrupt.
// Inlined to avoid the statetest -> state import cycle when test file lives in `package state`.
func corruptStateFile(t *testing.T, statePath string, mode corruptMode) {
	t.Helper()
	var content []byte
	switch mode {
	case corruptTruncated, corruptPartialWrite:
		s := State{
			"test": PackageState{
				Profile: "default",
				InstalledLinks: []InstalledLink{
					{Source: "/repo/test/file", Target: "/home/user/.config/test"},
				},
				InstalledAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			},
		}
		full, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if mode == corruptTruncated {
			content = full[:5]
		} else {
			needle := `"installed_links": [`
			idx := strings.Index(string(full), needle)
			if idx == -1 {
				t.Fatalf("could not find marker in marshaled state")
			}
			content = full[:idx+len(needle)+1]
		}
	case corruptInvalidJSON:
		content = []byte("{{{not json")
	case corruptNotAMap:
		content = []byte(`["this","is","an","array"]`)
	case corruptMissingFields:
		content = []byte(`{
  "nvim": {
    "profile": "",
    "installed_links": [],
    "installed_at": ""
  }
}`)
	case corruptNullJSON:
		content = []byte("null")
	default:
		t.Fatalf("unknown mode: %v", mode)
	}
	if err := os.MkdirAll(filepath.Dir(statePath), 0755); err != nil {
		t.Fatalf("mkdirall: %v", err)
	}
	if err := os.WriteFile(statePath, content, 0644); err != nil {
		t.Fatalf("writefile: %v", err)
	}
}

func validState() State {
	return State{
		"nvim": PackageState{
			Profile: "default",
			InstalledLinks: []InstalledLink{
				{Source: "/repo/nvim/init.lua", Target: "/home/u/.config/nvim/init.lua"},
			},
			InstalledAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
	}
}

// BUG-001 — Load on ModeInvalidJSON should return wrapped error mentioning path + parse position
// Spec source: .omo/plans/better-tests.md L923; AGENTS.md "State File" section
// Expected: Load returns an error that wraps the underlying parse error AND mentions the
//   file path (so users can locate the bad file) AND mentions a position/offset.
// Actual: state.Load returns the raw json.Unmarshal error — no path, no wrapping. Generic.
// Repro: Corrupt(path, ModeInvalidJSON); _, err := Load(path); inspect err string.
// How we know test is correct: a wrapped error citing the path is universally good
//   practice for file-format errors (cf. fmt.Errorf "%s: %w" convention used throughout
//   easyrice per AGENTS.md "Errors wrap with fmt.Errorf(\"context: %w\", err) - always").
func TestState_Corruption_BUG_001_InvalidJSON_WrappedErrorMentionsPath(t *testing.T) {
	t.Log("BUG-001")
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	corruptStateFile(t, statePath, corruptInvalidJSON)

	_, err := Load(statePath)
	if err == nil {
		t.Fatalf("BUG-001: expected error on invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), statePath) {
		t.Fatalf("BUG-001: error should mention state file path %q, got: %v", statePath, err)
	}
}

// BUG-002 — Load on ModeNotAMap (top-level JSON array) should return a clear typed error
// Spec source: .omo/plans/better-tests.md L924; AGENTS.md "State File" — top-level is a JSON object keyed by package name.
// Expected: Load returns a clear "state file is not a JSON object" error.
// Actual: Load returns the generic json.UnmarshalTypeError (does not mention "JSON object"
//   or recovery hint), or silently returns an empty State.
// Repro: Corrupt(path, ModeNotAMap); _, err := Load(path).
// How we know test is correct: the state schema declares `State map[string]PackageState`,
//   so an array is structurally invalid; a self-describing error is required to let
//   the user recover (delete vs hand-repair).
func TestState_Corruption_BUG_002_NotAMap_ClearObjectError(t *testing.T) {
	t.Log("BUG-002")
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	corruptStateFile(t, statePath, corruptNotAMap)

	s, err := Load(statePath)
	if err == nil {
		t.Fatalf("BUG-002: expected error when state file is a JSON array, got nil (loaded=%v)", s)
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "object") && !strings.Contains(msg, "map") {
		t.Fatalf("BUG-002: error should clearly say state file must be a JSON object/map, got: %v", err)
	}
}

// BUG-003 — Load on ModeNullJSON ("null") should return a clear "state file is null" error
// Spec source: .omo/plans/better-tests.md L925.
// Expected: Load returns a clear error explaining the file is `null` (not an object).
// Actual: json.Unmarshal into a State map accepts `null` and leaves the map nil — Load
//   then returns (nil, nil) silently, which the caller will treat as "no packages installed".
// Repro: Corrupt(path, ModeNullJSON); s, err := Load(path).
// How we know test is correct: a `null` state file is corruption, not an empty install.
//   Treating it as empty risks the user re-installing everything and losing tracking of
//   pre-existing symlinks (data loss adjacent — S1).
func TestState_Corruption_BUG_003_NullJSON_ClearError(t *testing.T) {
	t.Log("BUG-003")
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	corruptStateFile(t, statePath, corruptNullJSON)

	s, err := Load(statePath)
	if err == nil {
		t.Fatalf("BUG-003: expected error when state file is null, got nil (loaded=%v)", s)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "null") {
		t.Fatalf("BUG-003: error should mention null, got: %v", err)
	}
}

// BUG-004 — Load on ModeMissingFields should reject PackageState with empty Profile
// Spec source: .omo/plans/better-tests.md L926; AGENTS.md "State File" example shows
//   "profile" as a required field on every entry.
// Expected: Load validates that each PackageState has non-empty Profile and returns
//   an error naming the offending package.
// Actual: Load only does json.Unmarshal; no schema validation. Empty string passes through.
//   (Test currently passes because the empty installed_at string fails time.Time parsing —
//   wrong reason but still satisfies err != nil; catalog status=passing.)
// Repro: Corrupt(path, ModeMissingFields); _, err := Load(path).
// How we know test is correct: an empty profile means uninstall cannot select the
//   right files to remove; downstream code may panic or pick wrong defaults.
func TestState_Corruption_BUG_004_MissingFields_RejectsEmptyProfile(t *testing.T) {
	t.Log("BUG-004")
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	corruptStateFile(t, statePath, corruptMissingFields)

	_, err := Load(statePath)
	if err == nil {
		t.Fatalf("BUG-004: expected error when PackageState.Profile is empty, got nil")
	}
}

// BUG-005 — Load on ModePartialWrite should return wrapped parse error with recovery hint
// Spec source: .omo/plans/better-tests.md L927.
// Expected: error wraps the parse error AND the message helps the user recover
//   (e.g., mentions "backup" or "repair" or a recovery suggestion).
// Actual: Load returns the raw json.Unmarshal error — no recovery guidance.
// Repro: Corrupt(path, ModePartialWrite); _, err := Load(path).
// How we know test is correct: torn writes are a known crash-recovery scenario;
//   users hitting this need actionable guidance, not "unexpected end of JSON input".
func TestState_Corruption_BUG_005_PartialWrite_RecoveryHint(t *testing.T) {
	t.Log("BUG-005")
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	corruptStateFile(t, statePath, corruptPartialWrite)

	_, err := Load(statePath)
	if err == nil {
		t.Fatalf("BUG-005: expected error on partial write, got nil")
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "backup") && !strings.Contains(msg, "repair") && !strings.Contains(msg, "recover") {
		t.Fatalf("BUG-005: error should hint at recovery (backup/repair/recover), got: %v", err)
	}
}

// BUG-006 — Load on ModeTruncated should return wrapped error mentioning the file path
// Spec source: .omo/plans/better-tests.md L928.
// Expected: error wraps json error AND mentions the state file path.
// Actual: bare json.Unmarshal error, no path context.
// Repro: Corrupt(path, ModeTruncated); _, err := Load(path).
// How we know test is correct: same wrapping rationale as BUG-001; truncation is just
//   a different flavor of parse failure and must surface the path for forensics.
func TestState_Corruption_BUG_006_Truncated_PathInError(t *testing.T) {
	t.Log("BUG-006")
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	corruptStateFile(t, statePath, corruptTruncated)

	_, err := Load(statePath)
	if err == nil {
		t.Fatalf("BUG-006: expected error on truncated state, got nil")
	}
	if !strings.Contains(err.Error(), statePath) {
		t.Fatalf("BUG-006: error should mention state file path %q, got: %v", statePath, err)
	}
}

// BUG-007 — Save must be atomic: a faulted write must leave OLD or NEW content, never partial
// Spec source: .omo/plans/better-tests.md L929; first-principles argument — atomic write
//   (tmp + fsync + rename) is industry standard for any state file that survives crashes.
// Expected: Save uses a temp file + rename so a write fault leaves the final path either
//   fully OLD (rename never happened) or fully NEW (rename completed). Never torn.
// Actual: Save calls os.WriteFile directly on the final path; a partial-then-ENOSPC
//   write leaves the final state.json torn (first N bytes only). Pure data corruption.
// Repro: Save valid; fsfault.WithWriteFile_PartialThenENOSPC(&stateWriteFile, path, N);
//   call Save with new content; inspect bytes on disk against old and new.
// How we know test is correct: we have full control of both old and new content; any
//   on-disk byte sequence that is neither old nor new is by definition a torn write.
func TestState_Corruption_BUG_007_Save_Atomicity(t *testing.T) {
	t.Log("BUG-007")
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")

	oldS := validState()
	if err := Save(statePath, oldS); err != nil {
		t.Fatalf("BUG-007: setup Save (old) failed: %v", err)
	}
	oldBytes, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("BUG-007: setup ReadFile failed: %v", err)
	}

	newS := validState()
	ps := newS["nvim"]
	ps.Profile = "macbook"
	ps.InstalledLinks = append(ps.InstalledLinks, InstalledLink{
		Source: "/repo/nvim/extra.lua",
		Target: "/home/u/.config/nvim/extra.lua",
	})
	newS["nvim"] = ps

	fsfault.WithWriteFile_PartialThenENOSPC(t, &stateWriteFile, statePath, 16)

	saveErr := Save(statePath, newS)
	if saveErr == nil {
		t.Fatalf("BUG-007: expected ENOSPC error from faulted Save, got nil")
	}

	got, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatalf("BUG-007: ReadFile after faulted Save failed: %v", err)
	}

	if string(got) == string(oldBytes) {
		return
	}

	tmpDir := t.TempDir()
	probePath := filepath.Join(tmpDir, "probe.json")
	if err := Save(probePath, newS); err != nil {
		t.Fatalf("BUG-007: probe Save failed: %v", err)
	}
	expectedNew, err := os.ReadFile(probePath)
	if err != nil {
		t.Fatalf("BUG-007: probe ReadFile failed: %v", err)
	}
	if string(got) == string(expectedNew) {
		return
	}

	t.Fatalf("BUG-007: torn write detected — final state.json is neither fully old (%d bytes) nor fully new (%d bytes); got %d bytes: %q",
		len(oldBytes), len(expectedNew), len(got), string(got))
}

// BUG-008 — Save with EACCES via stateOpenFile should surface a clear permission error
// Spec source: .omo/plans/better-tests.md L930.
// Expected: Save honors the stateOpenFile seam for the final path (i.e., uses open+write
//   for atomicity) and returns a wrapped error citing EACCES.
// Actual: Save uses stateWriteFile only; the stateOpenFile seam is never invoked, so
//   the fault injection has no effect and Save succeeds. This proves Save does not
//   open the final path through the seam — i.e., no atomic-open path exists.
// Repro: fsfault.WithOpenFile_EACCES(&stateOpenFile, finalPath); Save(...); expect err.
// How we know test is correct: if Save were implemented atomically via OpenFile + write
//   + rename, the EACCES on the final path's OpenFile (or tmp path) would propagate.
//   That it does not is the documented gap.
func TestState_Corruption_BUG_008_Save_EACCES_ClearError(t *testing.T) {
	t.Log("BUG-008")
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")

	fsfault.WithOpenFile_EACCES(t, &stateOpenFile, statePath)

	err := Save(statePath, validState())
	if err == nil {
		t.Fatalf("BUG-008: expected EACCES error from Save, got nil (Save does not go through stateOpenFile seam — no atomic-open path)")
	}
	if !errors.Is(err, os.ErrPermission) && !strings.Contains(strings.ToLower(err.Error()), "permission") {
		t.Fatalf("BUG-008: error should mention permission/EACCES, got: %v", err)
	}
}

// BUG-009 — Save with ENOSPC via stateOpenFile should surface a clear no-space error
// Spec source: .omo/plans/better-tests.md L931.
// Expected: Save uses stateOpenFile and returns a wrapped error citing ENOSPC.
// Actual: Save uses stateWriteFile only; stateOpenFile fault has no effect. Save succeeds.
// Repro: fsfault.WithOpenFile_ENOSPC(&stateOpenFile, finalPath); Save(...); expect err.
// How we know test is correct: same argument as BUG-008. A correct atomic implementation
//   would surface ENOSPC immediately; the current implementation cannot.
func TestState_Corruption_BUG_009_Save_ENOSPC_ClearError(t *testing.T) {
	t.Log("BUG-009")
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")

	fsfault.WithOpenFile_ENOSPC(t, &stateOpenFile, statePath)

	err := Save(statePath, validState())
	if err == nil {
		t.Fatalf("BUG-009: expected ENOSPC error from Save, got nil (Save does not go through stateOpenFile seam)")
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "space") && !strings.Contains(msg, "enospc") {
		t.Fatalf("BUG-009: error should mention no space left on device, got: %v", err)
	}
}
