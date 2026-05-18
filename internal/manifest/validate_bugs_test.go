// Package manifest_test — BUG-driven validation gap tests for Task 9.
//
// Spec source: AGENTS.md "rice.toml Schema" + .omo/plans/better-tests.md
// Task 9 (lines 1017-1110). Each subtest is named `BUG-NNN-Title` via
// `t.Run` so every `--- FAIL` subtest line carries the BUG marker.
// All adversarial fixtures use repofixture.WithRawManifest (documented
// raw-passthrough); no production code under internal/manifest/ is modified.
package manifest_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/testutil/repofixture"
)

// errContains: case-insensitive substring AND across all needles.
func errContains(t *testing.T, err error, needles ...string) bool {
	t.Helper()
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, n := range needles {
		if !strings.Contains(msg, strings.ToLower(n)) {
			return false
		}
	}
	return true
}

func writeRawManifest(t *testing.T, raw string) string {
	t.Helper()
	fx := repofixture.New(t).WithRawManifest(raw).Build()
	return filepath.Join(fx.RepoPath, "rice.toml")
}

func TestManifest_Validation_Bugs(t *testing.T) {
	// =====================================================================
	// BUG-020 — Duplicate package name silently accepted
	// Spec: AGENTS.md "rice.toml Schema" — packages keyed by name imply
	//   uniqueness; duplicates should be rejected with error mentioning
	//   "duplicate" and the offending name.
	// Expected: LoadFile error containing "duplicate" and "foo".
	// Actual: go-toml emits "has already been defined" (generic).
	// Why correct: a single keyed table cannot meaningfully have two defs.
	// =====================================================================
	t.Run("BUG-020-DuplicatePackage", func(t *testing.T) {
		t.Log("BUG-020")
		raw := `schema_version = 1

[packages.foo]
supported_os = ["linux"]
[packages.foo.profiles.default]
sources = [{path = "a", mode = "file", target = "$HOME/a"}]

[packages.foo]
supported_os = ["darwin"]
[packages.foo.profiles.default]
sources = [{path = "b", mode = "file", target = "$HOME/b"}]
`
		path := writeRawManifest(t, raw)
		_, err := manifest.LoadFile(path)
		if !errContains(t, err, "duplicate", "foo") {
			t.Errorf("BUG-020: expected error containing 'duplicate' and 'foo', got: %v", err)
		}
	})

	// =====================================================================
	// BUG-021 — Duplicate profile name within a package silently accepted
	// Spec: AGENTS.md — profiles keyed by name.
	// Expected: error containing "duplicate" and "bar".
	// Actual: generic toml parser error.
	// =====================================================================
	t.Run("BUG-021-DuplicateProfile", func(t *testing.T) {
		t.Log("BUG-021")
		raw := `schema_version = 1

[packages.foo]
supported_os = ["linux"]

[packages.foo.profiles.bar]
sources = [{path = "a", mode = "file", target = "$HOME/a"}]

[packages.foo.profiles.bar]
sources = [{path = "b", mode = "file", target = "$HOME/b"}]
`
		path := writeRawManifest(t, raw)
		_, err := manifest.LoadFile(path)
		if !errContains(t, err, "duplicate", "bar") {
			t.Errorf("BUG-021: expected error containing 'duplicate' and 'bar', got: %v", err)
		}
	})

	// =====================================================================
	// BUG-022 — Missing schema_version not explicitly rejected
	// Spec: AGENTS.md schema (required field).
	// Expected: error "schema_version is required".
	// Actual: "unsupported schema_version: 0" — misleading wording.
	// =====================================================================
	t.Run("BUG-022-SchemaVersionMissing", func(t *testing.T) {
		t.Log("BUG-022")
		raw := `[packages.foo]
supported_os = ["linux"]
[packages.foo.profiles.default]
sources = [{path = "a", mode = "file", target = "$HOME/a"}]
`
		path := writeRawManifest(t, raw)
		_, err := manifest.LoadFile(path)
		if !errContains(t, err, "schema_version is required") {
			t.Errorf("BUG-022: expected 'schema_version is required', got: %v", err)
		}
	})

	// =====================================================================
	// BUG-023 — schema_version = 0 should error
	// =====================================================================
	t.Run("BUG-023-SchemaVersionZero", func(t *testing.T) {
		t.Log("BUG-023")
		raw := `schema_version = 0

[packages.foo]
supported_os = ["linux"]
[packages.foo.profiles.default]
sources = [{path = "a", mode = "file", target = "$HOME/a"}]
`
		path := writeRawManifest(t, raw)
		_, err := manifest.LoadFile(path)
		if !errContains(t, err, "schema_version") {
			t.Errorf("BUG-023: expected error mentioning schema_version, got: %v", err)
		}
	})

	// =====================================================================
	// BUG-024 — schema_version = 2 should produce forward-compat hint
	// Expected: "unsupported schema_version 2 (this binary supports 1)".
	// Actual: "unsupported schema_version: 2" — no forward-compat hint.
	// =====================================================================
	t.Run("BUG-024-SchemaVersionFuture2", func(t *testing.T) {
		t.Log("BUG-024")
		raw := `schema_version = 2

[packages.foo]
supported_os = ["linux"]
[packages.foo.profiles.default]
sources = [{path = "a", mode = "file", target = "$HOME/a"}]
`
		path := writeRawManifest(t, raw)
		_, err := manifest.LoadFile(path)
		if !errContains(t, err, "unsupported schema_version 2", "this binary supports 1") {
			t.Errorf("BUG-024: expected hint 'unsupported schema_version 2 (this binary supports 1)', got: %v", err)
		}
	})

	// =====================================================================
	// BUG-025 — schema_version = 999 — same wording requirement as BUG-024
	// =====================================================================
	t.Run("BUG-025-SchemaVersionFuture999", func(t *testing.T) {
		t.Log("BUG-025")
		raw := `schema_version = 999

[packages.foo]
supported_os = ["linux"]
[packages.foo.profiles.default]
sources = [{path = "a", mode = "file", target = "$HOME/a"}]
`
		path := writeRawManifest(t, raw)
		_, err := manifest.LoadFile(path)
		if !errContains(t, err, "unsupported schema_version 999", "this binary supports 1") {
			t.Errorf("BUG-025: expected hint 'unsupported schema_version 999 (this binary supports 1)', got: %v", err)
		}
	})

	// =====================================================================
	// BUG-026 — Empty [packages] table rejected
	// Expected: error containing "no packages declared".
	// =====================================================================
	t.Run("BUG-026-EmptyPackages", func(t *testing.T) {
		t.Log("BUG-026")
		raw := `schema_version = 1

[packages]
`
		path := writeRawManifest(t, raw)
		_, err := manifest.LoadFile(path)
		if !errContains(t, err, "no packages declared") {
			t.Errorf("BUG-026: expected 'no packages declared', got: %v", err)
		}
	})

	// =====================================================================
	// BUG-027 — Bare-string source rejected at LoadFile integration level
	// Spec: AGENTS.md — inline table form ONLY.
	// =====================================================================
	t.Run("BUG-027-BareStringSource", func(t *testing.T) {
		t.Log("BUG-027")
		raw := `schema_version = 1

[packages.foo]
supported_os = ["linux"]
[packages.foo.profiles.default]
sources = ["bare-string"]
`
		path := writeRawManifest(t, raw)
		_, err := manifest.LoadFile(path)
		if err == nil {
			t.Errorf("BUG-027: expected error for bare-string source, got nil")
		}
	})

	// =====================================================================
	// BUG-028 — supported_os = [] rejected
	// =====================================================================
	t.Run("BUG-028-SupportedOSEmpty", func(t *testing.T) {
		t.Log("BUG-028")
		raw := `schema_version = 1

[packages.foo]
supported_os = []
[packages.foo.profiles.default]
sources = [{path = "a", mode = "file", target = "$HOME/a"}]
`
		path := writeRawManifest(t, raw)
		_, err := manifest.LoadFile(path)
		if !errContains(t, err, "supported_os") {
			t.Errorf("BUG-028: expected error mentioning supported_os, got: %v", err)
		}
	})

	// =====================================================================
	// BUG-029 — supported_os = ["bsd"] rejected (invalid OS)
	// =====================================================================
	t.Run("BUG-029-SupportedOSInvalid", func(t *testing.T) {
		t.Log("BUG-029")
		raw := `schema_version = 1

[packages.foo]
supported_os = ["bsd"]
[packages.foo.profiles.default]
sources = [{path = "a", mode = "file", target = "$HOME/a"}]
`
		path := writeRawManifest(t, raw)
		_, err := manifest.LoadFile(path)
		if !errContains(t, err, "bsd") {
			t.Errorf("BUG-029: expected error mentioning 'bsd', got: %v", err)
		}
	})

	// =====================================================================
	// BUG-030 — Source path "../escape" rejected (path traversal)
	// =====================================================================
	t.Run("BUG-030-PathTraversal", func(t *testing.T) {
		t.Log("BUG-030")
		raw := `schema_version = 1

[packages.foo]
supported_os = ["linux"]
[packages.foo.profiles.default]
sources = [{path = "../escape", mode = "file", target = "$HOME/a"}]
`
		path := writeRawManifest(t, raw)
		_, err := manifest.LoadFile(path)
		if !errContains(t, err, "..") {
			t.Errorf("BUG-030: expected error mentioning '..', got: %v", err)
		}
	})

	// =====================================================================
	// BUG-031 — Source path "/absolute" rejected (must be relative)
	// =====================================================================
	t.Run("BUG-031-AbsolutePath", func(t *testing.T) {
		t.Log("BUG-031")
		raw := `schema_version = 1

[packages.foo]
supported_os = ["linux"]
[packages.foo.profiles.default]
sources = [{path = "/absolute", mode = "file", target = "$HOME/a"}]
`
		path := writeRawManifest(t, raw)
		_, err := manifest.LoadFile(path)
		if !errContains(t, err, "relative") {
			t.Errorf("BUG-031: expected error mentioning 'relative', got: %v", err)
		}
	})

	// =====================================================================
	// BUG-032 — Source mode = "bogus" rejected
	// =====================================================================
	t.Run("BUG-032-BogusMode", func(t *testing.T) {
		t.Log("BUG-032")
		raw := `schema_version = 1

[packages.foo]
supported_os = ["linux"]
[packages.foo.profiles.default]
sources = [{path = "a", mode = "bogus", target = "$HOME/a"}]
`
		path := writeRawManifest(t, raw)
		_, err := manifest.LoadFile(path)
		if !errContains(t, err, "mode") {
			t.Errorf("BUG-032: expected error mentioning 'mode', got: %v", err)
		}
	})

	// =====================================================================
	// BUG-033 — Source target = "" rejected
	// =====================================================================
	t.Run("BUG-033-EmptyTarget", func(t *testing.T) {
		t.Log("BUG-033")
		raw := `schema_version = 1

[packages.foo]
supported_os = ["linux"]
[packages.foo.profiles.default]
sources = [{path = "a", mode = "file", target = ""}]
`
		path := writeRawManifest(t, raw)
		_, err := manifest.LoadFile(path)
		if !errContains(t, err, "target") {
			t.Errorf("BUG-033: expected error mentioning 'target', got: %v", err)
		}
	})

	// =====================================================================
	// BUG-034 — Unreadable manifest (chmod 000) returns wrapped error that
	//   names file path AND permission detail.
	// Spec: fmt.Errorf("context: %w", err) — debugging permission errors
	//   needs path + permission indication.
	// Skip if euid 0 (root bypasses DAC).
	// =====================================================================
	t.Run("BUG-034-UnreadableManifest", func(t *testing.T) {
		t.Log("BUG-034")
		if os.Geteuid() == 0 {
			t.Log("BUG-034: skipped, running as root")
			t.Skip("requires non-root")
		}

		fx := repofixture.New(t).WithRawManifest(`schema_version = 1

[packages.foo]
supported_os = ["linux"]
[packages.foo.profiles.default]
sources = [{path = "a", mode = "file", target = "$HOME/a"}]
`).Build()
		path := filepath.Join(fx.RepoPath, "rice.toml")

		if err := os.Chmod(path, 0o000); err != nil {
			t.Fatalf("BUG-034: failed to chmod manifest to 000: %v", err)
		}
		t.Cleanup(func() {
			_ = os.Chmod(path, 0o644)
		})

		_, err := manifest.LoadFile(path)
		if err == nil {
			t.Fatalf("BUG-034: expected error reading chmod-000 manifest, got nil")
		}
		hasPath := strings.Contains(err.Error(), path) || strings.Contains(err.Error(), "rice.toml")
		hasPerm := errContains(t, err, "permission") || errContains(t, err, "denied")
		if !hasPath || !hasPerm {
			t.Errorf("BUG-034: expected error to mention file path AND permission detail, got: %v", err)
		}
	})
}
