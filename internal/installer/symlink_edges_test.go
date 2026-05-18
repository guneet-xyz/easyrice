//go:build !windows

package installer_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/installer"
	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/state"
)

// buildEdgeRequest builds an InstallRequest pointing at a package whose
// source directory has been hand-crafted on disk under repoRoot/pkgName/srcDir.
// Returns the request and the absolute source dir for further manipulation.
func buildEdgeRequest(t *testing.T, repoRoot, pkgName, profileName, srcDir, mode, target string) installer.InstallRequest {
	t.Helper()
	homeDir := t.TempDir()
	statePath := filepath.Join(t.TempDir(), "state.json")
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	pkg := &manifest.PackageDef{
		Description: "edge test pkg",
		SupportedOS: []string{"linux", "darwin"},
		Profiles: map[string]manifest.ProfileDef{
			profileName: {
				Sources: []manifest.SourceSpec{
					{Path: srcDir, Mode: mode, Target: target},
				},
			},
		},
	}
	specs := []manifest.SourceSpec{{Path: srcDir, Mode: mode, Target: target}}
	return installer.InstallRequest{
		RepoRoot:    repoRoot,
		PackageName: pkgName,
		ProfileName: profileName,
		Pkg:         pkg,
		Specs:       specs,
		CurrentOS:   runtime.GOOS,
		HomeDir:     homeDir,
		StatePath:   statePath,
	}
}

// TestSymlink_Edges pins installer behavior at symlink edge cases per
// AGENTS.md NOTES/GOTCHAS + .omo/plans/better-tests.md lines 1578-1585.
func TestSymlink_Edges(t *testing.T) {
	t.Run("BUG-120-SourceIsSymlink", func(t *testing.T) {
		t.Log("BUG-120")
		// Spec (AGENTS.md): "Symlinks inside source trees are skipped during walk —
		// we only manage real files." A symlinked entry inside the source dir must
		// NOT produce an Op in the plan.
		repoRoot := t.TempDir()
		pkgDir := filepath.Join(repoRoot, "pkg", "src")
		require.NoError(t, os.MkdirAll(pkgDir, 0o755), "BUG-120: setup failed")

		realFile := filepath.Join(pkgDir, "real.conf")
		require.NoError(t, os.WriteFile(realFile, []byte("real"), 0o644), "BUG-120: setup failed")

		// External pointee so the symlinked entry inside the source tree points OUTSIDE.
		external := filepath.Join(t.TempDir(), "external_secret.txt")
		require.NoError(t, os.WriteFile(external, []byte("external"), 0o644), "BUG-120: setup failed")
		symlinkInSource := filepath.Join(pkgDir, "evil_link.conf")
		require.NoError(t, os.Symlink(external, symlinkInSource), "BUG-120: setup failed")

		req := buildEdgeRequest(t, repoRoot, "pkg", "default", "src", "file", "$HOME/.config/pkg")

		p, err := installer.BuildInstallPlan(req)
		require.NoError(t, err, "BUG-120: BuildInstallPlan must succeed and skip the symlink, not error")
		require.NotNil(t, p, "BUG-120: plan must not be nil")

		for _, op := range p.Ops {
			assert.NotEqual(t, symlinkInSource, op.Source,
				"BUG-120: plan must NOT include an Op whose Source is the symlinked entry %q", symlinkInSource)
			assert.NotContains(t, op.Target, "evil_link.conf",
				"BUG-120: plan must NOT include a target derived from the skipped symlink, got %q", op.Target)
		}
	})

	t.Run("BUG-121-SourceRootIsSymlink", func(t *testing.T) {
		t.Log("BUG-121")
		// Spec: source dir ROOT being a symlink is undocumented; intended contract
		// (pin via test) is to REJECT with a clear "source path must be a real
		// directory, not a symlink" error. If production silently follows the
		// symlink, this test FAILS (real bug found).
		repoRoot := t.TempDir()
		realSrc := filepath.Join(t.TempDir(), "real_source_dir")
		require.NoError(t, os.MkdirAll(realSrc, 0o755), "BUG-121: setup failed")
		require.NoError(t, os.WriteFile(filepath.Join(realSrc, "f.conf"), []byte("x"), 0o644), "BUG-121: setup failed")

		pkgRoot := filepath.Join(repoRoot, "pkg")
		require.NoError(t, os.MkdirAll(pkgRoot, 0o755), "BUG-121: setup failed")
		// The source root is itself a symlink to a real directory outside the repo.
		symlinkedRoot := filepath.Join(pkgRoot, "src")
		require.NoError(t, os.Symlink(realSrc, symlinkedRoot), "BUG-121: setup failed")

		req := buildEdgeRequest(t, repoRoot, "pkg", "default", "src", "file", "$HOME/.config/pkg")

		_, err := installer.BuildInstallPlan(req)
		require.Error(t, err, "BUG-121: BuildInstallPlan must reject a symlinked source root (security: attacker-controlled redirect)")
		msg := strings.ToLower(err.Error())
		assert.True(t,
			strings.Contains(msg, "symlink") || strings.Contains(msg, "real directory") || strings.Contains(msg, "not a symlink"),
			"BUG-121: error must mention symlink/real directory contract, got %q", err.Error())
	})

	t.Run("BUG-122-BrokenSymlinkAtTarget", func(t *testing.T) {
		t.Log("BUG-122")
		// Spec: DetectConflicts is idempotent; any pre-existing target that is not
		// our own state-tracked symlink IS a conflict. A dangling symlink must
		// surface as a Conflict — silent overwrite would be a footgun.
		homeDir := t.TempDir()
		repoRoot := t.TempDir()
		// Build a realistic planned link: target inside $HOME, source under repo.
		target := filepath.Join(homeDir, ".config", "broken_target")
		require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755), "BUG-122: setup failed")
		require.NoError(t, os.Symlink(filepath.Join(repoRoot, "does_not_exist"), target), "BUG-122: setup failed")

		// Sanity: target must be a dangling symlink.
		_, statErr := os.Stat(target)
		require.True(t, os.IsNotExist(statErr), "BUG-122: target must be dangling, got %v", statErr)

		planned := []installer.PlannedLink{
			{Source: filepath.Join(repoRoot, "pkg", "src", "real.conf"), Target: target},
		}
		conflicts := installer.DetectConflicts(planned, nil)
		require.NotEmpty(t, conflicts, "BUG-122: DetectConflicts must report a Conflict for a pre-existing dangling symlink, got none")

		var found bool
		for _, c := range conflicts {
			if c.Target == target {
				found = true
				assert.NotEmpty(t, c.Reason, "BUG-122: conflict reason must be non-empty for %q", target)
				break
			}
		}
		assert.True(t, found, "BUG-122: no conflict entry matched the dangling target %q", target)
	})

	t.Run("BUG-123-SpacesInPath", func(t *testing.T) {
		t.Log("BUG-123")
		runRoundTrip(t, "BUG-123", "My Configs", "spacefile.conf")
	})

	t.Run("BUG-124-UnicodeInPath", func(t *testing.T) {
		t.Log("BUG-124")
		runRoundTrip(t, "BUG-124", "日本語", "設定.conf")
	})

	t.Run("BUG-125-ShellMetacharsInPath", func(t *testing.T) {
		t.Log("BUG-125")
		// $ ; ` should be unexpanded by os.ExpandEnv since they are not part of
		// a valid $VAR sequence. Round-trip the literal characters.
		runRoundTrip(t, "BUG-125", "weird;dir`name", "file.conf")
	})

	t.Run("BUG-126-LongPath", func(t *testing.T) {
		t.Log("BUG-126")
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("BUG-126 panicked: %v", r)
			}
		}()
		// Build a target near PATH_MAX (4096 on Linux). Using one very long
		// component pushes past most filesystem NAME_MAX (255) and PATH_MAX.
		repoRoot := t.TempDir()
		pkgDir := filepath.Join(repoRoot, "pkg", "src")
		require.NoError(t, os.MkdirAll(pkgDir, 0o755), "BUG-126: setup failed")
		require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "f.conf"), []byte("x"), 0o644), "BUG-126: setup failed")

		longName := strings.Repeat("a", 300)
		targetBase := "$HOME/" + longName + "/" + strings.Repeat("b", 300) + "/" + strings.Repeat("c", 300)

		req := buildEdgeRequest(t, repoRoot, "pkg", "default", "src", "file", targetBase)

		_, err := installer.Install(req)
		require.Error(t, err, "BUG-126: install must return an error for a path near PATH_MAX, never panic or succeed silently")
		msg := strings.ToLower(err.Error())
		assert.True(t,
			strings.Contains(msg, "path") || strings.Contains(msg, "name too long") || strings.Contains(msg, "long"),
			"BUG-126: error must mention path/name length, got %q", err.Error())
	})

	t.Run("BUG-127-TargetIsDirectory", func(t *testing.T) {
		t.Log("BUG-127")
		// Spec: target path pre-exists as a directory containing a file. Install
		// MUST detect a conflict and MUST NOT silently delete the directory.
		// This defends against the most dangerous regression — silent dir deletion.
		homeDir := t.TempDir()
		repoRoot := t.TempDir()
		pkgDir := filepath.Join(repoRoot, "pkg", "src")
		require.NoError(t, os.MkdirAll(pkgDir, 0o755), "BUG-127: setup failed")
		require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "f.conf"), []byte("incoming"), 0o644), "BUG-127: setup failed")

		// Pre-create the target as a REAL DIRECTORY with a file inside.
		target := filepath.Join(homeDir, ".config", "pkg", "f.conf")
		require.NoError(t, os.MkdirAll(target, 0o755), "BUG-127: setup failed")
		innerFile := filepath.Join(target, "precious_user_data.txt")
		require.NoError(t, os.WriteFile(innerFile, []byte("DO NOT LOSE"), 0o644), "BUG-127: setup failed")

		pkg := &manifest.PackageDef{
			SupportedOS: []string{"linux", "darwin"},
			Profiles: map[string]manifest.ProfileDef{
				"default": {Sources: []manifest.SourceSpec{{Path: "src", Mode: "file", Target: "$HOME/.config/pkg"}}},
			},
		}
		statePath := filepath.Join(t.TempDir(), "state.json")
		t.Setenv("HOME", homeDir)
		t.Setenv("USERPROFILE", homeDir)
		req := installer.InstallRequest{
			RepoRoot:    repoRoot,
			PackageName: "pkg",
			ProfileName: "default",
			Pkg:         pkg,
			Specs:       []manifest.SourceSpec{{Path: "src", Mode: "file", Target: "$HOME/.config/pkg"}},
			CurrentOS:   runtime.GOOS,
			HomeDir:     homeDir,
			StatePath:   statePath,
		}

		_, err := installer.Install(req)
		require.Error(t, err, "BUG-127: install must error when target is a real directory; silent deletion would be catastrophic")
		assert.Contains(t, strings.ToLower(err.Error()), "conflict",
			"BUG-127: error must mention conflict, got %q", err.Error())

		// THE critical assertion: the directory must still exist after the failed install.
		assert.DirExists(t, target, "BUG-127: target directory must still exist after failed install — silent directory deletion is forbidden")
		assert.FileExists(t, innerFile, "BUG-127: user data inside the target directory must be preserved")
		content, readErr := os.ReadFile(innerFile)
		require.NoError(t, readErr, "BUG-127: failed to re-read precious file")
		assert.Equal(t, "DO NOT LOSE", string(content), "BUG-127: precious file content must be unchanged")
	})
}

// runRoundTrip performs an install + uninstall round-trip with a tricky path
// segment, then asserts that state.json round-trip equality holds (the recorded
// links survive marshal/unmarshal without mutation).
func runRoundTrip(t *testing.T, marker, dirSegment, fileName string) {
	t.Helper()
	homeDir := t.TempDir()
	repoRoot := t.TempDir()
	pkgDir := filepath.Join(repoRoot, "pkg", "src")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755), "%s: setup failed", marker)
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, fileName), []byte("payload"), 0o644), "%s: setup failed", marker)

	target := "$HOME/" + dirSegment
	pkg := &manifest.PackageDef{
		SupportedOS: []string{"linux", "darwin"},
		Profiles: map[string]manifest.ProfileDef{
			"default": {Sources: []manifest.SourceSpec{{Path: "src", Mode: "file", Target: target}}},
		},
	}
	statePath := filepath.Join(t.TempDir(), "state.json")
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	req := installer.InstallRequest{
		RepoRoot:    repoRoot,
		PackageName: "pkg",
		ProfileName: "default",
		Pkg:         pkg,
		Specs:       []manifest.SourceSpec{{Path: "src", Mode: "file", Target: target}},
		CurrentOS:   runtime.GOOS,
		HomeDir:     homeDir,
		StatePath:   statePath,
	}

	res, err := installer.Install(req)
	require.NoError(t, err, "%s: install must succeed with special-char path %q", marker, dirSegment)
	require.NotNil(t, res, "%s: install result must not be nil", marker)
	require.NotEmpty(t, res.LinksCreated, "%s: install must create at least one link", marker)

	// state.json round-trip: load, re-save, load again, compare.
	st1, err := state.Load(statePath)
	require.NoError(t, err, "%s: state.Load failed", marker)
	require.NoError(t, state.Save(statePath, st1), "%s: state.Save failed", marker)
	st2, err := state.Load(statePath)
	require.NoError(t, err, "%s: state.Load (round 2) failed", marker)
	assert.Equal(t, st1, st2, "%s: state.json round-trip must be identity (special chars must not be mangled)", marker)

	pkgState, ok := st1["pkg"]
	require.True(t, ok, "%s: state must contain entry for pkg", marker)
	require.NotEmpty(t, pkgState.InstalledLinks, "%s: state must record at least one InstalledLink", marker)
	for _, ln := range pkgState.InstalledLinks {
		assert.Contains(t, ln.Target, dirSegment, "%s: stored target must contain special segment %q (got %q)", marker, dirSegment, ln.Target)
	}

	// Uninstall round-trip.
	require.NoError(t, installer.Uninstall(installer.UninstallRequest{PackageName: "pkg", StatePath: statePath}),
		"%s: uninstall must succeed with special-char path", marker)
}
