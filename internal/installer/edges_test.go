package installer

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/guneet-xyz/easyrice/internal/testutil/fsfault"
	"github.com/guneet-xyz/easyrice/internal/testutil/goldenfs"
)

func edgesWriteFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func edgesNewRequest(t *testing.T, repoRoot, homeDir, statePath string, pkg *manifest.PackageDef, pkgName, profileName string) InstallRequest {
	t.Helper()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	return InstallRequest{
		RepoRoot:    repoRoot,
		PackageName: pkgName,
		ProfileName: profileName,
		Pkg:         pkg,
		Specs:       pkg.Profiles[profileName].Sources,
		CurrentOS:   runtime.GOOS,
		HomeDir:     homeDir,
		StatePath:   statePath,
	}
}

func TestInstaller_Edges(t *testing.T) {
	t.Run("BUG-066-PartialRollback", func(t *testing.T) {
		// BUG-066 — Partial install rollback durability.
		// Status: failing (or passing-as-guard depending on implementation).
		// Severity: S2.
		// Package: internal/installer.
		// Spec source: .omo/plans/better-tests.md line 1303 (Task 12).
		// Expected: when symlink op 5 fails, state.json records the 4
		// successful links and exactly 4 symlinks exist on disk.
		// Actual: verified by this test against current implementation.
		t.Log("BUG-066")
		if runtime.GOOS == "windows" {
			t.Skip("partial-rollback uses POSIX fsfault")
		}

		repoRoot := t.TempDir()
		homeDir := t.TempDir()
		statePath := filepath.Join(t.TempDir(), "state.json")

		for i := 0; i < 10; i++ {
			edgesWriteFile(t, filepath.Join(repoRoot, "pkg", "src", "f"+string(rune('0'+i))), "x")
		}

		pkg := &manifest.PackageDef{
			Description: "partial rollback",
			SupportedOS: []string{runtime.GOOS},
			Root:        "pkg",
			Profiles: map[string]manifest.ProfileDef{
				"default": {Sources: []manifest.SourceSpec{
					{Path: "src", Mode: "file", Target: filepath.Join(homeDir, ".config", "pkg")},
				}},
			},
		}
		req := edgesNewRequest(t, repoRoot, homeDir, statePath, pkg, "pkg", "default")

		fsfault.WithSymlink_FailAfterN(t, &installerSymlink, 4)

		_, err := Install(req)
		require.Error(t, err, "BUG-066 expected fault on op 5")

		st, err := state.Load(statePath)
		require.NoError(t, err)
		ps, ok := st["pkg"]
		require.True(t, ok, "BUG-066 state must record package")
		require.Equal(t, 4, len(ps.InstalledLinks), "BUG-066 state has exactly 4 entries")

		var diskLinks int
		require.NoError(t, filepath.WalkDir(homeDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type()&fs.ModeSymlink != 0 {
				diskLinks++
			}
			return nil
		}))
		require.Equal(t, 4, diskLinks, "BUG-066 exactly 4 symlinks on disk")
	})

	t.Run("BUG-067-TargetOutsideHome", func(t *testing.T) {
		// BUG-067 — withinHome bypass via absolute /etc/passwd target.
		// Status: passing (defense-in-depth check enforces).
		// Severity: S1 (security boundary).
		// Spec source: .omo/plans/better-tests.md line 1304.
		// Expected: BuildInstallPlan rejects before any FS op with "outside"
		// or "escapes home" wording.
		t.Log("BUG-067")
		homeDir := t.TempDir()
		repoRoot := t.TempDir()
		edgesWriteFile(t, filepath.Join(repoRoot, "pkg", "src", "a"), "x")

		pkg := &manifest.PackageDef{
			Description: "bypass",
			SupportedOS: []string{runtime.GOOS},
			Root:        "pkg",
			Profiles: map[string]manifest.ProfileDef{
				"default": {Sources: []manifest.SourceSpec{
					{Path: "src", Mode: "file", Target: "/etc/passwd"},
				}},
			},
		}
		req := edgesNewRequest(t, repoRoot, homeDir, filepath.Join(t.TempDir(), "state.json"), pkg, "pkg", "default")

		_, err := BuildInstallPlan(req)
		require.Error(t, err, "BUG-067 must reject /etc/passwd target")
		msg := err.Error()
		assert.True(t,
			strings.Contains(msg, "outside") || strings.Contains(msg, "escapes home"),
			"BUG-067 error %q must mention boundary violation", msg,
		)
	})

	t.Run("BUG-068-SymlinkedHomeBoundary", func(t *testing.T) {
		// BUG-068 — withinHome with symlinked $HOME.
		// Status: depends on resolution policy.
		// Severity: S1.
		// Spec source: .omo/plans/better-tests.md line 1305.
		// Expected: target traversing through a symlinked $HOME using ..
		// is rejected after Clean.
		t.Log("BUG-068")
		if runtime.GOOS == "windows" {
			t.Skip("POSIX symlink semantics")
		}

		realHome := filepath.Join(t.TempDir(), "realhome")
		fakeHome := filepath.Join(t.TempDir(), "fakehome")
		require.NoError(t, os.MkdirAll(realHome, 0o755))
		require.NoError(t, os.Symlink(realHome, fakeHome))

		repoRoot := t.TempDir()
		edgesWriteFile(t, filepath.Join(repoRoot, "pkg", "src", "a"), "x")

		pkg := &manifest.PackageDef{
			Description: "symlinked home",
			SupportedOS: []string{runtime.GOOS},
			Root:        "pkg",
			Profiles: map[string]manifest.ProfileDef{
				"default": {Sources: []manifest.SourceSpec{
					{Path: "src", Mode: "file", Target: fakeHome + "/../escape"},
				}},
			},
		}
		req := edgesNewRequest(t, repoRoot, fakeHome, filepath.Join(t.TempDir(), "state.json"), pkg, "pkg", "default")

		_, err := BuildInstallPlan(req)
		require.Error(t, err, "BUG-068 must reject escape through symlinked home")
	})

	t.Run("BUG-069-DotDotInTarget", func(t *testing.T) {
		// BUG-069 — withinHome with ".." segments in target.
		// Status: passing.
		// Severity: S1.
		// Spec source: .omo/plans/better-tests.md line 1306.
		// Expected: $HOME/../etc rejected after expansion + cleaning.
		t.Log("BUG-069")
		homeDir := t.TempDir()
		repoRoot := t.TempDir()
		edgesWriteFile(t, filepath.Join(repoRoot, "pkg", "src", "a"), "x")

		pkg := &manifest.PackageDef{
			Description: "dotdot",
			SupportedOS: []string{runtime.GOOS},
			Root:        "pkg",
			Profiles: map[string]manifest.ProfileDef{
				"default": {Sources: []manifest.SourceSpec{
					{Path: "src", Mode: "file", Target: "$HOME/../etc"},
				}},
			},
		}
		req := edgesNewRequest(t, repoRoot, homeDir, filepath.Join(t.TempDir(), "state.json"), pkg, "pkg", "default")

		_, err := BuildInstallPlan(req)
		require.Error(t, err, "BUG-069 must reject $HOME/../etc")
	})

	t.Run("BUG-070-FileFolderCollision", func(t *testing.T) {
		// BUG-070 — file+folder mode collision on overlapping target subtree.
		// Status: passing (caught at plan time).
		// Severity: S2.
		// Spec source: .omo/plans/better-tests.md line 1307.
		// Expected: BuildInstallPlan rejects when a folder-mode source and
		// any other source target overlapping paths.
		t.Log("BUG-070")
		homeDir := t.TempDir()
		repoRoot := t.TempDir()
		edgesWriteFile(t, filepath.Join(repoRoot, "pkg", "cfg", "x"), "x")
		edgesWriteFile(t, filepath.Join(repoRoot, "pkg", "files", "y"), "y")

		target := filepath.Join(homeDir, ".config", "app")
		pkg := &manifest.PackageDef{
			Description: "collision",
			SupportedOS: []string{runtime.GOOS},
			Root:        "pkg",
			Profiles: map[string]manifest.ProfileDef{
				"default": {Sources: []manifest.SourceSpec{
					{Path: "cfg", Mode: "folder", Target: target},
					{Path: "files", Mode: "file", Target: target},
				}},
			},
		}
		req := edgesNewRequest(t, repoRoot, homeDir, filepath.Join(t.TempDir(), "state.json"), pkg, "pkg", "default")

		_, err := BuildInstallPlan(req)
		require.Error(t, err, "BUG-070 must reject file+folder collision")
		assert.True(t,
			strings.Contains(err.Error(), "overlap") || strings.Contains(err.Error(), "folder"),
			"BUG-070 error %q must explain the collision", err.Error(),
		)
	})

	t.Run("BUG-071-OverlayLastWins", func(t *testing.T) {
		// BUG-071 — file-mode overlay: last source wins per relative path.
		// Status: passing.
		// Severity: S2.
		// Spec source: .omo/plans/better-tests.md line 1308; AGENTS.md "Source modes".
		// Expected: with sources A, B, C all writing the same target file,
		// the resulting symlink points into source C.
		t.Log("BUG-071")
		if runtime.GOOS == "windows" {
			t.Skip("symlink tree snapshot uses POSIX")
		}

		homeDir := t.TempDir()
		repoRoot := t.TempDir()
		edgesWriteFile(t, filepath.Join(repoRoot, "pkg", "A", "bar"), "A")
		edgesWriteFile(t, filepath.Join(repoRoot, "pkg", "B", "bar"), "B")
		edgesWriteFile(t, filepath.Join(repoRoot, "pkg", "C", "bar"), "C")

		target := filepath.Join(homeDir, ".config", "foo")
		pkg := &manifest.PackageDef{
			Description: "overlay",
			SupportedOS: []string{runtime.GOOS},
			Root:        "pkg",
			Profiles: map[string]manifest.ProfileDef{
				"default": {Sources: []manifest.SourceSpec{
					{Path: "A", Mode: "file", Target: target},
					{Path: "B", Mode: "file", Target: target},
					{Path: "C", Mode: "file", Target: target},
				}},
			},
		}
		req := edgesNewRequest(t, repoRoot, homeDir, filepath.Join(t.TempDir(), "state.json"), pkg, "pkg", "default")

		_, err := Install(req)
		require.NoError(t, err)

		linkPath := filepath.Join(target, "bar")
		dest, err := os.Readlink(linkPath)
		require.NoError(t, err)
		expected := filepath.Join(repoRoot, "pkg", "C", "bar")
		require.Equal(t, expected, dest, "BUG-071 link must point to source C")

		snap := goldenfs.Snapshot(t, homeDir)
		require.Contains(t, snap, ".config/foo/bar ->", "BUG-071 snapshot must contain the overlay symlink")
		require.True(t, strings.Contains(dest, "/C/"), "BUG-071 readlink dest %q must be from source C", dest)
	})

	t.Run("BUG-072-FolderModeDoubleTarget", func(t *testing.T) {
		// BUG-072 — folder-mode is NOT overlayable.
		// Status: passing.
		// Severity: S2.
		// Spec source: .omo/plans/better-tests.md line 1309.
		// Expected: two folder sources targeting same path → reject at plan.
		t.Log("BUG-072")
		homeDir := t.TempDir()
		repoRoot := t.TempDir()
		edgesWriteFile(t, filepath.Join(repoRoot, "pkg", "A", "x"), "A")
		edgesWriteFile(t, filepath.Join(repoRoot, "pkg", "B", "y"), "B")

		target := filepath.Join(homeDir, ".config", "foo")
		pkg := &manifest.PackageDef{
			Description: "double folder",
			SupportedOS: []string{runtime.GOOS},
			Root:        "pkg",
			Profiles: map[string]manifest.ProfileDef{
				"default": {Sources: []manifest.SourceSpec{
					{Path: "A", Mode: "folder", Target: target},
					{Path: "B", Mode: "folder", Target: target},
				}},
			},
		}
		req := edgesNewRequest(t, repoRoot, homeDir, filepath.Join(t.TempDir(), "state.json"), pkg, "pkg", "default")

		_, err := BuildInstallPlan(req)
		require.Error(t, err, "BUG-072 two folder sources to same target must be rejected")
	})

	t.Run("BUG-073-Idempotent", func(t *testing.T) {
		// BUG-073 — Idempotency of ConvergeAll.
		// Status: failing on current impl if installed_at is rewritten.
		// Severity: S2.
		// Spec source: .omo/plans/better-tests.md line 1310.
		// Expected: ConvergeAll twice → second call no-op; installed_at byte-equal.
		t.Log("BUG-073")
		homeDir := t.TempDir()
		repoRoot := t.TempDir()
		statePath := filepath.Join(t.TempDir(), "state.json")
		edgesWriteFile(t, filepath.Join(repoRoot, "pkg", "src", "a"), "x")

		pkg := manifest.PackageDef{
			Description: "idempotent",
			SupportedOS: []string{runtime.GOOS},
			Root:        "pkg",
			Profiles: map[string]manifest.ProfileDef{
				"default": {Sources: []manifest.SourceSpec{
					{Path: "src", Mode: "file", Target: filepath.Join(homeDir, ".config", "pkg")},
				}},
			},
		}
		mf := &manifest.Manifest{SchemaVersion: 1, Packages: map[string]manifest.PackageDef{"pkg": pkg}}
		t.Setenv("HOME", homeDir)
		t.Setenv("USERPROFILE", homeDir)

		req := ConvergeAllRequest{
			RepoRoot: repoRoot, DefaultProfile: "default",
			CurrentOS: runtime.GOOS, HomeDir: homeDir,
			StatePath: statePath, Manifest: mf,
		}

		_, err := ConvergeAll(req)
		require.NoError(t, err)
		before, err := state.Load(statePath)
		require.NoError(t, err)
		beforeAt := before["pkg"].InstalledAt

		_, err = ConvergeAll(req)
		require.NoError(t, err)
		after, err := state.Load(statePath)
		require.NoError(t, err)
		require.Equal(t, beforeAt, after["pkg"].InstalledAt, "BUG-073 installed_at must be unchanged after no-op converge")
	})

	t.Run("BUG-074-ProfileSwitchPreservesInstalledAt", func(t *testing.T) {
		// BUG-074 — profile-switch preserves installed_at.
		// Status: failing on current impl (ExecuteInstallPlan stamps time.Now()).
		// Severity: S3.
		// Spec source: .omo/plans/better-tests.md line 1311.
		// Expected: after switching profiles, installed_at equals the initial value.
		t.Log("BUG-074")
		homeDir := t.TempDir()
		repoRoot := t.TempDir()
		statePath := filepath.Join(t.TempDir(), "state.json")
		edgesWriteFile(t, filepath.Join(repoRoot, "pkg", "A", "a"), "A")
		edgesWriteFile(t, filepath.Join(repoRoot, "pkg", "B", "b"), "B")

		pkg := manifest.PackageDef{
			Description: "switch",
			SupportedOS: []string{runtime.GOOS},
			Root:        "pkg",
			Profiles: map[string]manifest.ProfileDef{
				"a": {Sources: []manifest.SourceSpec{{Path: "A", Mode: "file", Target: filepath.Join(homeDir, ".config", "pkg")}}},
				"b": {Sources: []manifest.SourceSpec{{Path: "B", Mode: "file", Target: filepath.Join(homeDir, ".config", "pkg")}}},
			},
		}
		mf := &manifest.Manifest{SchemaVersion: 1, Packages: map[string]manifest.PackageDef{"pkg": pkg}}
		t.Setenv("HOME", homeDir)
		t.Setenv("USERPROFILE", homeDir)

		pkgCopy := pkg
		creq := ConvergeRequest{
			RepoRoot: repoRoot, PackageName: "pkg", RequestedProfile: "a",
			CurrentOS: runtime.GOOS, HomeDir: homeDir,
			StatePath: statePath, Pkg: &pkgCopy, Manifest: mf,
		}
		cr, err := BuildConvergePlan(creq)
		require.NoError(t, err)
		require.NoError(t, ExecuteConvergePlan(creq, cr))

		before, err := state.Load(statePath)
		require.NoError(t, err)
		beforeAt := before["pkg"].InstalledAt

		creq.RequestedProfile = "b"
		cr2, err := BuildConvergePlan(creq)
		require.NoError(t, err)
		require.NoError(t, ExecuteConvergePlan(creq, cr2))

		after, err := state.Load(statePath)
		require.NoError(t, err)
		require.Equal(t, beforeAt, after["pkg"].InstalledAt, "BUG-074 profile switch must preserve installed_at")
	})

	t.Run("BUG-075-RepairPreservesInstalledAt", func(t *testing.T) {
		// BUG-075 — repair preserves installed_at.
		// Status: failing on current impl (stamps time.Now() on each install).
		// Severity: S3.
		// Spec source: .omo/plans/better-tests.md line 1312.
		// Expected: deleting a link then re-converging keeps installed_at unchanged.
		t.Log("BUG-075")
		if runtime.GOOS == "windows" {
			t.Skip("repair uses POSIX symlinks")
		}
		homeDir := t.TempDir()
		repoRoot := t.TempDir()
		statePath := filepath.Join(t.TempDir(), "state.json")
		edgesWriteFile(t, filepath.Join(repoRoot, "pkg", "src", "a"), "x")

		pkg := manifest.PackageDef{
			Description: "repair",
			SupportedOS: []string{runtime.GOOS},
			Root:        "pkg",
			Profiles: map[string]manifest.ProfileDef{
				"default": {Sources: []manifest.SourceSpec{
					{Path: "src", Mode: "file", Target: filepath.Join(homeDir, ".config", "pkg")},
				}},
			},
		}
		mf := &manifest.Manifest{SchemaVersion: 1, Packages: map[string]manifest.PackageDef{"pkg": pkg}}
		t.Setenv("HOME", homeDir)
		t.Setenv("USERPROFILE", homeDir)

		req := ConvergeAllRequest{
			RepoRoot: repoRoot, DefaultProfile: "default",
			CurrentOS: runtime.GOOS, HomeDir: homeDir,
			StatePath: statePath, Manifest: mf,
		}
		_, err := ConvergeAll(req)
		require.NoError(t, err)
		before, err := state.Load(statePath)
		require.NoError(t, err)
		beforeAt := before["pkg"].InstalledAt

		require.NoError(t, os.Remove(filepath.Join(homeDir, ".config", "pkg", "a")))

		_, err = ConvergeAll(req)
		require.NoError(t, err)
		after, err := state.Load(statePath)
		require.NoError(t, err)
		require.Equal(t, beforeAt, after["pkg"].InstalledAt, "BUG-075 repair must preserve installed_at")
	})

	t.Run("BUG-076-UninstallNotInstalled", func(t *testing.T) {
		// BUG-076 — Uninstall of not-installed package returns error.
		// Status: passing.
		// Severity: S3.
		// Spec source: .omo/plans/better-tests.md line 1313.
		// Expected: error `package %q not installed`.
		t.Log("BUG-076")
		statePath := filepath.Join(t.TempDir(), "state.json")
		err := Uninstall(UninstallRequest{PackageName: "ghost", StatePath: statePath})
		require.Error(t, err, "BUG-076 must error")
		require.Contains(t, err.Error(), "not installed", "BUG-076 error must say not installed")
	})

	t.Run("BUG-077-UnknownProfile", func(t *testing.T) {
		// BUG-077 — install with unknown profile lists available profiles.
		// Status: failing if error text lacks profile list.
		// Severity: S3.
		// Spec source: .omo/plans/better-tests.md line 1314.
		// Expected: error contains the requested-profile name AND at least one known profile.
		t.Log("BUG-077")
		homeDir := t.TempDir()
		repoRoot := t.TempDir()
		statePath := filepath.Join(t.TempDir(), "state.json")
		edgesWriteFile(t, filepath.Join(repoRoot, "pkg", "src", "a"), "x")

		pkg := manifest.PackageDef{
			Description: "known profiles",
			SupportedOS: []string{runtime.GOOS},
			Root:        "pkg",
			Profiles: map[string]manifest.ProfileDef{
				"default": {Sources: []manifest.SourceSpec{
					{Path: "src", Mode: "file", Target: filepath.Join(homeDir, ".config", "pkg")},
				}},
			},
		}
		mf := &manifest.Manifest{SchemaVersion: 1, Packages: map[string]manifest.PackageDef{"pkg": pkg}}
		t.Setenv("HOME", homeDir)
		t.Setenv("USERPROFILE", homeDir)

		pkgCopy := pkg
		creq := ConvergeRequest{
			RepoRoot: repoRoot, PackageName: "pkg", RequestedProfile: "doesnotexist",
			CurrentOS: runtime.GOOS, HomeDir: homeDir,
			StatePath: statePath, Pkg: &pkgCopy, Manifest: mf,
		}
		_, err := BuildConvergePlan(creq)
		require.Error(t, err, "BUG-077 unknown profile must error")
		assert.Contains(t, err.Error(), "default", "BUG-077 error must list available profile %q. got: %s", "default", err.Error())
	})

	t.Run("BUG-078-UndeclaredPackage", func(t *testing.T) {
		// BUG-078 — install of undeclared package hints at `rice status`.
		// Status: failing if hint absent; passing if a clear error references status.
		// Severity: S3.
		// Spec source: .omo/plans/better-tests.md line 1315.
		// Expected: error references the unknown package name and points the user to discoverable state.
		t.Log("BUG-078")
		homeDir := t.TempDir()
		repoRoot := t.TempDir()
		statePath := filepath.Join(t.TempDir(), "state.json")
		mf := &manifest.Manifest{SchemaVersion: 1, Packages: map[string]manifest.PackageDef{}}
		t.Setenv("HOME", homeDir)
		t.Setenv("USERPROFILE", homeDir)

		creq := ConvergeRequest{
			RepoRoot: repoRoot, PackageName: "ghost", RequestedProfile: "default",
			CurrentOS: runtime.GOOS, HomeDir: homeDir,
			StatePath: statePath, Pkg: nil, Manifest: mf,
		}
		_, err := BuildConvergePlan(creq)
		require.Error(t, err, "BUG-078 nil Pkg must error")
		assert.NotEmpty(t, err.Error(), "BUG-078 error must be descriptive")
	})

	t.Run("BUG-079-EmptyProfilesRejected", func(t *testing.T) {
		// BUG-079 — manifest validation rejects empty packages.NAME.profiles.
		// Status: passing (Validate enforces).
		// Severity: S2.
		// Spec source: .omo/plans/better-tests.md line 1316; cross-ref BUG-026.
		// Expected: Validate returns an error when a package has zero profiles.
		t.Log("BUG-079")
		bad := &manifest.Manifest{
			SchemaVersion: 1,
			Packages: map[string]manifest.PackageDef{
				"pkg": {
					Description: "no profiles",
					SupportedOS: []string{runtime.GOOS},
					Profiles:    map[string]manifest.ProfileDef{},
				},
			},
		}
		err := manifest.Validate(bad)
		require.Error(t, err, "BUG-079 empty profiles must be rejected")
		require.Contains(t, err.Error(), "profile", "BUG-079 error must mention profiles")
	})
}
