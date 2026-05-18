// Package profile circular-import bug tests.
//
// Task: better-tests Task 10 — Profile circular imports (BUG-040..BUG-049).
// Spec: .omo/plans/better-tests.md lines 1112-1199.
// Schema spec: AGENTS.md "rice.toml Schema" import section.
// Source-mode overlay spec: AGENTS.md "Source modes" file-mode last-wins rule.
//
// These tests assert error-message QUALITY and overlay semantics. Some bugs
// are expected to FAIL on `main` because production currently emits a
// generic cycle key ("RepoRoot|remote|pkg|profile") rather than the
// human-readable arrow path the catalog requires. Every assertion failure
// carries the BUG-04X marker via t.Log + subtest name.
//
// Fixtures use internal/testutil/multiremote so the actual on-disk layout
// matches what a real managed repo would have (submodules under remotes/).
// ResolveSpecs only reads rice.toml files via repo.RemoteTomlPath, so the
// local PackageDef is hand-built in Go to point at the remote-A profile.
package profile

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/testutil/multiremote"
)

// remoteWithImport returns a remote rice.toml whose pkg.profile imports from
// another remote (cycle setup). pkgName and profileName are the names INSIDE
// this remote.
func remoteWithImport(pkgName, profileName, importSpec string) string {
	return `schema_version = 1

[packages.` + pkgName + `]
description = "test"
supported_os = ["linux", "darwin", "windows"]

[packages.` + pkgName + `.profiles.` + profileName + `]
import = "` + importSpec + `"
`
}

// remoteWithSources returns a remote rice.toml whose pkg.profile defines
// concrete sources (used as cycle terminators or for overlay tests).
func remoteWithSources(pkgName, profileName, path, mode, target string) string {
	return `schema_version = 1

[packages.` + pkgName + `]
description = "test"
supported_os = ["linux", "darwin", "windows"]

[packages.` + pkgName + `.profiles.` + profileName + `]
sources = [{path = "` + path + `", mode = "` + mode + `", target = "` + target + `"}]
`
}

// TestProfile_CircularImports covers BUG-040..BUG-049 — circular-import
// detection, missing-target diagnostics, spec-parser hygiene, and the
// import+local-sources overlay ordering.
func TestProfile_CircularImports(t *testing.T) {
	// -----------------------------------------------------------------
	// BUG-040 — A→B→A 2-hop cycle.
	// Spec: .omo/plans/better-tests.md line 1117. ResolveSpecs MUST return
	// an error of the form "import cycle detected: a -> b -> a" so the
	// user sees the full path, not an internal cache key.
	// How we know test is correct: AGENTS.md mandates cycle detection;
	// the better-tests plan promises a human-readable arrow path.
	// -----------------------------------------------------------------
	t.Run("BUG-040_A_B_A", func(t *testing.T) {
		t.Log("BUG-040")
		fix := multiremote.New(t).
			AddRemote("a", remoteWithImport("p", "x", "remotes/b#p.x")).
			AddRemote("b", remoteWithImport("p", "x", "remotes/a#p.x")).
			Build()
		t.Cleanup(fix.Cleanup)

		localPkg := &manifest.PackageDef{
			Profiles: map[string]manifest.ProfileDef{
				"start": {Import: "remotes/a#p.x"},
			},
		}
		_, err := ResolveSpecs(fix.ParentRepoPath, localPkg, "top", "start")
		require.Error(t, err, "BUG-040: expected cycle error, got nil")
		msg := err.Error()
		assert.Contains(t, msg, "cycle", "BUG-040: error must mention cycle; got: %s", msg)
		assert.Contains(t, msg, "a -> b -> a",
			"BUG-040: error must show full cycle path 'a -> b -> a'; got: %s", msg)
	})

	// -----------------------------------------------------------------
	// BUG-041 — A→B→C→A 3-hop cycle.
	// Spec: .omo/plans/better-tests.md line 1118. Error must list the full
	// path so multi-remote cycles are debuggable.
	// How we know test is correct: same arrow-path contract as BUG-040,
	// extended to three hops.
	// -----------------------------------------------------------------
	t.Run("BUG-041_A_B_C_A", func(t *testing.T) {
		t.Log("BUG-041")
		fix := multiremote.New(t).
			AddRemote("a", remoteWithImport("p", "x", "remotes/b#p.x")).
			AddRemote("b", remoteWithImport("p", "x", "remotes/c#p.x")).
			AddRemote("c", remoteWithImport("p", "x", "remotes/a#p.x")).
			Build()
		t.Cleanup(fix.Cleanup)

		localPkg := &manifest.PackageDef{
			Profiles: map[string]manifest.ProfileDef{
				"start": {Import: "remotes/a#p.x"},
			},
		}
		_, err := ResolveSpecs(fix.ParentRepoPath, localPkg, "top", "start")
		require.Error(t, err, "BUG-041: expected cycle error, got nil")
		msg := err.Error()
		assert.Contains(t, msg, "cycle", "BUG-041: error must mention cycle; got: %s", msg)
		assert.Contains(t, msg, "a -> b -> c -> a",
			"BUG-041: error must list full path 'a -> b -> c -> a'; got: %s", msg)
	})

	// -----------------------------------------------------------------
	// BUG-042 — Self-import via a remote.
	// Spec: .omo/plans/better-tests.md line 1119. A profile that imports
	// `remotes/x#x.p` where remote `x` re-imports `remotes/x#x.p` must be
	// caught.
	// How we know test is correct: AGENTS.md states cycles are detected;
	// a single-remote loop is the degenerate cycle case.
	// -----------------------------------------------------------------
	t.Run("BUG-042_Self_Import_Via_Remote", func(t *testing.T) {
		t.Log("BUG-042")
		fix := multiremote.New(t).
			AddRemote("x", remoteWithImport("x", "p", "remotes/x#x.p")).
			Build()
		t.Cleanup(fix.Cleanup)

		localPkg := &manifest.PackageDef{
			Profiles: map[string]manifest.ProfileDef{
				"start": {Import: "remotes/x#x.p"},
			},
		}
		_, err := ResolveSpecs(fix.ParentRepoPath, localPkg, "top", "start")
		require.Error(t, err, "BUG-042: expected self-cycle error")
		assert.Contains(t, err.Error(), "cycle",
			"BUG-042: error must say 'cycle'; got: %s", err.Error())
	})

	// -----------------------------------------------------------------
	// BUG-043 — Direct self-reference via same-name profile in remote.
	// Spec: .omo/plans/better-tests.md line 1120. Profile imports a remote
	// whose same-name profile imports back — must still be detected.
	// How we know test is correct: cycle detection is name-agnostic; a
	// two-remote ping-pong with identical names is still a cycle.
	// -----------------------------------------------------------------
	t.Run("BUG-043_Direct_Self_Reference", func(t *testing.T) {
		t.Log("BUG-043")
		// Remote 'a' has package 'pkg' profile 'p' that imports back to
		// remote 'b'.pkg.p which imports remotes/a#pkg.p. Same names
		// throughout; cycle must still resolve.
		fix := multiremote.New(t).
			AddRemote("a", remoteWithImport("pkg", "p", "remotes/b#pkg.p")).
			AddRemote("b", remoteWithImport("pkg", "p", "remotes/a#pkg.p")).
			Build()
		t.Cleanup(fix.Cleanup)

		localPkg := &manifest.PackageDef{
			Profiles: map[string]manifest.ProfileDef{
				"p": {Import: "remotes/a#pkg.p"},
			},
		}
		_, err := ResolveSpecs(fix.ParentRepoPath, localPkg, "pkg", "p")
		require.Error(t, err, "BUG-043: expected cycle error on same-name reference")
		assert.Contains(t, err.Error(), "cycle",
			"BUG-043: error must say 'cycle'; got: %s", err.Error())
	})

	// -----------------------------------------------------------------
	// BUG-044 — Import points to non-existent remote.
	// Spec: .omo/plans/better-tests.md line 1121. Error must say
	// `remote "ghost" not initialized` AND hint at `rice remote add`.
	// How we know test is correct: AGENTS.md "Managed Repo" section
	// documents `rice remote add` as the entry point; the hint must
	// point users there, not at `rice remote update` (which assumes the
	// submodule already exists).
	// -----------------------------------------------------------------
	t.Run("BUG-044_Missing_Remote", func(t *testing.T) {
		t.Log("BUG-044")
		// Build with NO remotes — any import will reference a ghost.
		fix := multiremote.New(t).Build()
		t.Cleanup(fix.Cleanup)

		localPkg := &manifest.PackageDef{
			Profiles: map[string]manifest.ProfileDef{
				"default": {Import: "remotes/ghost#pkg.default"},
			},
		}
		_, err := ResolveSpecs(fix.ParentRepoPath, localPkg, "pkg", "default")
		require.Error(t, err, "BUG-044: expected missing-remote error")
		msg := err.Error()
		assert.Contains(t, msg, "ghost",
			"BUG-044: error must name the missing remote; got: %s", msg)
		assert.Contains(t, msg, "not initialized",
			"BUG-044: error must say 'not initialized'; got: %s", msg)
		assert.Contains(t, msg, "rice remote add",
			"BUG-044: error must hint at `rice remote add`; got: %s", msg)
	})

	// -----------------------------------------------------------------
	// BUG-045 — Missing package in existing remote.
	// Spec: .omo/plans/better-tests.md line 1122. Error must name BOTH
	// the remote and the missing package.
	// How we know test is correct: a precise diagnostic is required so
	// users can fix the import without re-reading every remote's toml.
	// -----------------------------------------------------------------
	t.Run("BUG-045_Missing_Package_In_Remote", func(t *testing.T) {
		t.Log("BUG-045")
		fix := multiremote.New(t).
			AddRemote("kick", remoteWithSources("realpkg", "default", "config", "file", "$HOME/.config/x")).
			Build()
		t.Cleanup(fix.Cleanup)

		localPkg := &manifest.PackageDef{
			Profiles: map[string]manifest.ProfileDef{
				"default": {Import: "remotes/kick#missingpkg.default"},
			},
		}
		_, err := ResolveSpecs(fix.ParentRepoPath, localPkg, "top", "default")
		require.Error(t, err, "BUG-045: expected missing-package error")
		msg := err.Error()
		assert.Contains(t, msg, "kick",
			"BUG-045: error must name the remote; got: %s", msg)
		assert.Contains(t, msg, "missingpkg",
			"BUG-045: error must name the missing package; got: %s", msg)
	})

	// -----------------------------------------------------------------
	// BUG-046 — Missing profile in existing package.
	// Spec: .omo/plans/better-tests.md line 1123. Error must name the
	// remote, the package, AND the missing profile.
	// How we know test is correct: same precision argument as BUG-045,
	// extended to the profile level.
	// -----------------------------------------------------------------
	t.Run("BUG-046_Missing_Profile_In_Package", func(t *testing.T) {
		t.Log("BUG-046")
		fix := multiremote.New(t).
			AddRemote("kick", remoteWithSources("realpkg", "default", "config", "file", "$HOME/.config/x")).
			Build()
		t.Cleanup(fix.Cleanup)

		localPkg := &manifest.PackageDef{
			Profiles: map[string]manifest.ProfileDef{
				"default": {Import: "remotes/kick#realpkg.ghostprofile"},
			},
		}
		_, err := ResolveSpecs(fix.ParentRepoPath, localPkg, "top", "default")
		require.Error(t, err, "BUG-046: expected missing-profile error")
		msg := err.Error()
		assert.Contains(t, msg, "kick",
			"BUG-046: error must name the remote; got: %s", msg)
		assert.Contains(t, msg, "realpkg",
			"BUG-046: error must name the package; got: %s", msg)
		assert.Contains(t, msg, "ghostprofile",
			"BUG-046: error must name the missing profile; got: %s", msg)
	})

	// -----------------------------------------------------------------
	// BUG-047 — Malformed spec: embedded space.
	// Spec: .omo/plans/better-tests.md line 1124. `ParseImportSpec` must
	// reject `remotes/foo bar` (space breaks remote name).
	// How we know test is correct: the documented grammar in AGENTS.md
	// disallows whitespace inside remote names.
	// -----------------------------------------------------------------
	t.Run("BUG-047_Spec_With_Space", func(t *testing.T) {
		t.Log("BUG-047")
		_, err := manifest.ParseImportSpec("remotes/foo bar")
		require.Error(t, err,
			"BUG-047: ParseImportSpec must reject `remotes/foo bar`")
		// Must be rejected for a syntactic reason — either missing '#'
		// or invalid characters. Either way, not a silent success.
		msg := err.Error()
		assert.True(t,
			strings.Contains(msg, "#") || strings.Contains(msg, "remote") || strings.Contains(msg, "separator"),
			"BUG-047: rejection must cite the parse problem; got: %s", msg)
	})

	// -----------------------------------------------------------------
	// BUG-048 — Empty parts in spec.
	// Spec: .omo/plans/better-tests.md line 1125. Each empty component
	// (remote, package, profile) must be rejected with an error that
	// names WHICH part is empty.
	// How we know test is correct: AGENTS.md documents all three parts
	// must be non-empty; the diagnostic must be specific so the user
	// can fix the right segment.
	// -----------------------------------------------------------------
	t.Run("BUG-048_Empty_Parts", func(t *testing.T) {
		t.Log("BUG-048")
		cases := []struct {
			name      string
			input     string
			wantToken string // substring of the offending-part name
		}{
			{"empty_remote", "remotes/#pkg.default", "remote"},
			{"empty_package", "remotes/foo#.default", "package"},
			{"empty_profile", "remotes/foo#pkg.", "profile"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Log("BUG-048")
				_, err := manifest.ParseImportSpec(tc.input)
				require.Error(t, err,
					"BUG-048: %q must be rejected", tc.input)
				assert.Contains(t, err.Error(), tc.wantToken,
					"BUG-048: error for %q must name the empty part %q; got: %s",
					tc.input, tc.wantToken, err.Error())
			})
		}
	})

	// -----------------------------------------------------------------
	// BUG-049 — Import + local sources overlay ordering.
	// Spec: .omo/plans/better-tests.md line 1126. When a profile has both
	// `import` and `sources`, the resolved list must be imported-first,
	// local-last (so file-mode last-wins overlays the imported tree per
	// AGENTS.md "Source modes").
	// How we know test is correct: AGENTS.md mandates file-mode last-wins
	// and documents that `import` is resolved before the local `sources`
	// in the resolution layer; the SourceSpec list order is the contract.
	// -----------------------------------------------------------------
	t.Run("BUG-049_Import_Plus_Local_Overlay", func(t *testing.T) {
		t.Log("BUG-049")
		fix := multiremote.New(t).
			AddRemote("kick", remoteWithSources("nvim", "default", "config", "file", "$HOME/.config/nvim")).
			Build()
		t.Cleanup(fix.Cleanup)

		localPkg := &manifest.PackageDef{
			Profiles: map[string]manifest.ProfileDef{
				"default": {
					Import: "remotes/kick#nvim.default",
					Sources: []manifest.SourceSpec{
						{Path: "local-override", Mode: "file", Target: "$HOME/.config/nvim"},
					},
				},
			},
		}
		got, err := ResolveSpecs(fix.ParentRepoPath, localPkg, "nvim", "default")
		require.NoError(t, err, "BUG-049: overlay resolution must succeed")
		require.Len(t, got, 2,
			"BUG-049: expected 2 specs (1 imported + 1 local); got %d: %+v", len(got), got)

		// Imported spec is absolutized to remotes/kick/nvim/config.
		assert.Contains(t, got[0].Path, "remotes",
			"BUG-049: first spec must be the imported (absolute) one; got: %+v", got[0])
		assert.Contains(t, got[0].Path, "kick",
			"BUG-049: first spec must reference remote 'kick'; got: %+v", got[0])

		// Local source comes AFTER imports (last-wins for file mode).
		assert.Equal(t, "local-override", got[1].Path,
			"BUG-049: local source must be appended AFTER imports (last-wins overlay); got order: %+v", got)
		assert.Equal(t, "file", got[1].Mode)
		assert.Equal(t, "$HOME/.config/nvim", got[1].Target)
	})
}
