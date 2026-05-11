package installer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/plan"
	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/guneet-xyz/easyrice/internal/symlink"
)

// switchSetup installs the package with `initialProfile` and returns the
// configured SwitchRequest pointing to `newProfile`. HomeDir, RepoRoot and
// StatePath are isolated per test.
func switchSetup(t *testing.T, initialProfile, newProfile string) (SwitchRequest, *InstallResult) {
	t.Helper()
	repo := fixtureRepo(t)

	installReq := newRequest(t, repo, "ghostty", initialProfile)

	res, err := Install(installReq)
	require.NoError(t, err)
	require.NotNil(t, res)

	return SwitchRequest{
		RepoRoot:    repo,
		PackageName: "ghostty",
		NewProfile:  newProfile,
		CurrentOS:   runtime.GOOS,
		HomeDir:     installReq.HomeDir,
		StatePath:   installReq.StatePath,
	}, res
}

func TestBuildSwitchPlan_PackageNotInstalled(t *testing.T) {
	repo := fixtureRepo(t)
	installReq := newRequest(t, repo, "ghostty", "macbook")

	req := SwitchRequest{
		RepoRoot:    repo,
		PackageName: "ghostty",
		NewProfile:  "macbook",
		CurrentOS:   runtime.GOOS,
		HomeDir:     installReq.HomeDir,
		StatePath:   installReq.StatePath,
	}

	sp, err := BuildSwitchPlan(req)
	require.Error(t, err)
	assert.Nil(t, sp)
	assert.Contains(t, err.Error(), "not installed")
}

func TestBuildSwitchPlan_HappyPath_DoesNotTouchFilesystem(t *testing.T) {
	req, initialResult := switchSetup(t, "macbook", "common")

	stateBefore, err := state.Load(req.StatePath)
	require.NoError(t, err)

	sp, err := BuildSwitchPlan(req)
	require.NoError(t, err)
	require.NotNil(t, sp)
	require.NotNil(t, sp.Uninstall)
	require.NotNil(t, sp.Install)

	assert.Equal(t, "macbook", sp.Uninstall.Profile)
	assert.Equal(t, "common", sp.Install.Profile)
	assert.Equal(t, "ghostty", sp.Uninstall.PackageName)
	assert.Equal(t, "ghostty", sp.Install.PackageName)
	assert.Empty(t, sp.Install.Conflicts)
	assert.Len(t, sp.Uninstall.Ops, len(initialResult.LinksCreated))

	// State unchanged after build.
	stateAfter, err := state.Load(req.StatePath)
	require.NoError(t, err)
	assert.Equal(t, stateBefore["ghostty"].Profile, stateAfter["ghostty"].Profile)
	assert.Equal(t, stateBefore["ghostty"].InstalledLinks, stateAfter["ghostty"].InstalledLinks)

	// Existing symlinks remain intact, no new ones created.
	for _, link := range initialResult.LinksCreated {
		isOurs, checkErr := symlink.IsSymlinkTo(link.Target, link.Source)
		require.NoError(t, checkErr)
		assert.True(t, isOurs, "existing symlink should remain")
	}
}

func TestBuildSwitchPlan_NewProfileMissing(t *testing.T) {
	req, _ := switchSetup(t, "macbook", "doesnotexist")

	sp, err := BuildSwitchPlan(req)
	require.Error(t, err)
	assert.Nil(t, sp)
}

func TestBuildSwitchPlan_PreFlight_ConflictFromForeignFile(t *testing.T) {
	req, _ := switchSetup(t, "macbook", "common")

	// Create a non-rice file at a target that the new profile would create
	// but the old profile did NOT manage (config.toml from package root, only
	// present in the `common` profile).
	conflictPath := filepath.Join(req.HomeDir, ".config", "ghostty", "settings")
	require.NoError(t, os.MkdirAll(filepath.Dir(conflictPath), 0o755))
	require.NoError(t, os.WriteFile(conflictPath, []byte("foreign"), 0o644))

	sp, err := BuildSwitchPlan(req)
	require.Error(t, err)
	require.NotNil(t, sp)
	require.NotNil(t, sp.Install)
	assert.NotEmpty(t, sp.Install.Conflicts)

	conflictTargets := make(map[string]bool)
	for _, c := range sp.Install.Conflicts {
		conflictTargets[c.Target] = true
	}
	assert.True(t, conflictTargets[conflictPath], "foreign file should be reported as conflict")
}

func TestBuildSwitchPlan_PreFlight_OldLinkReusedNotConflict(t *testing.T) {
	req, initialResult := switchSetup(t, "macbook", "macbook")

	// Pick an old-link target and overwrite the symlink with a foreign file
	// that the install plan will see as a conflict (unless ignored).
	require.NotEmpty(t, initialResult.LinksCreated)
	target := initialResult.LinksCreated[0].Target
	require.NoError(t, os.Remove(target))
	require.NoError(t, os.WriteFile(target, []byte("foreign-content"), 0o644))

	sp, err := BuildSwitchPlan(req)
	require.NoError(t, err, "old-link target reused by new profile must not produce a conflict")
	require.NotNil(t, sp)
	assert.Empty(t, sp.Install.Conflicts, "reused old-link targets must not be conflicts")

	// Sanity: the target appears in both the uninstall ops (ignoreTargets source)
	// and the install ops (would-be conflict).
	uninstallHas := false
	for _, op := range sp.Uninstall.Ops {
		if op.Target == target {
			uninstallHas = true
			break
		}
	}
	installHas := false
	for _, op := range sp.Install.Ops {
		if op.Target == target {
			installHas = true
			break
		}
	}
	assert.True(t, uninstallHas, "uninstall plan must include the reused target")
	assert.True(t, installHas, "install plan must include the reused target")
}

func TestExecuteSwitchPlan_HappyPath(t *testing.T) {
	req, initialResult := switchSetup(t, "macbook", "common")

	sp, err := BuildSwitchPlan(req)
	require.NoError(t, err)

	require.NoError(t, ExecuteSwitchPlan(sp, req.StatePath))

	// State reflects new profile.
	st, err := state.Load(req.StatePath)
	require.NoError(t, err)
	pkg, ok := st["ghostty"]
	require.True(t, ok)
	assert.Equal(t, "common", pkg.Profile)
	assert.NotEmpty(t, pkg.InstalledLinks)

	configTarget := filepath.Join(req.HomeDir, ".config", "ghostty", "config")
	fi, err := os.Lstat(configTarget)
	require.NoError(t, err)
	assert.NotZero(t, fi.Mode()&os.ModeSymlink, "config.toml should be a symlink")

	// All new state links are real symlinks pointing to their sources.
	for _, link := range pkg.InstalledLinks {
		isOurs, checkErr := symlink.IsSymlinkTo(link.Target, link.Source)
		require.NoError(t, checkErr)
		assert.True(t, isOurs, "new link should point at its source")
	}

	// Old initial links that are NOT in the new profile should be gone.
	newTargets := make(map[string]struct{}, len(pkg.InstalledLinks))
	for _, l := range pkg.InstalledLinks {
		newTargets[l.Target] = struct{}{}
	}
	for _, oldLink := range initialResult.LinksCreated {
		if _, stillUsed := newTargets[oldLink.Target]; stillUsed {
			continue
		}
		_, statErr := os.Lstat(oldLink.Target)
		assert.True(t, os.IsNotExist(statErr), "old-only link %q should be removed", oldLink.Target)
	}
}

// TestExecuteSwitchPlan_Success installs profile A, switches to profile B, and
// asserts old links removed, new links created, and state.json updated.
func TestExecuteSwitchPlan_Success(t *testing.T) {
	req, initialResult := switchSetup(t, "macbook", "common")

	// Capture the set of old targets BEFORE the switch.
	oldTargets := make(map[string]string, len(initialResult.LinksCreated))
	for _, l := range initialResult.LinksCreated {
		oldTargets[l.Target] = l.Source
	}

	sp, err := BuildSwitchPlan(req)
	require.NoError(t, err)
	require.NoError(t, ExecuteSwitchPlan(sp, req.StatePath))

	st, err := state.Load(req.StatePath)
	require.NoError(t, err)
	pkg, ok := st["ghostty"]
	require.True(t, ok, "package must remain in state after switch")
	assert.Equal(t, "common", pkg.Profile, "state profile must reflect new profile")
	require.NotEmpty(t, pkg.InstalledLinks, "new profile must produce installed links")

	// All new links must be live symlinks pointing at their source.
	newTargets := make(map[string]struct{}, len(pkg.InstalledLinks))
	for _, l := range pkg.InstalledLinks {
		newTargets[l.Target] = struct{}{}
		isOurs, checkErr := symlink.IsSymlinkTo(l.Target, l.Source)
		require.NoError(t, checkErr)
		assert.True(t, isOurs, "new link %q must point at %q", l.Target, l.Source)
	}

	// Old-only links (present in profile A but not in profile B) must be gone.
	for tgt := range oldTargets {
		if _, reused := newTargets[tgt]; reused {
			continue
		}
		_, statErr := os.Lstat(tgt)
		assert.True(t, os.IsNotExist(statErr),
			"old-only target %q should no longer exist after switch", tgt)
	}
}

// TestExecuteSwitchPlan_AtomicOnUninstallFailure asserts that when the
// uninstall phase fails, the install phase does NOT run and no new symlinks
// from the new profile are created.
func TestExecuteSwitchPlan_AtomicOnUninstallFailure(t *testing.T) {
	req, initialResult := switchSetup(t, "macbook", "common")

	sp, err := BuildSwitchPlan(req)
	require.NoError(t, err)

	// Snapshot install plan targets BEFORE we corrupt the statePath.
	installTargets := make([]string, 0, len(sp.Install.Ops))
	for _, op := range sp.Install.Ops {
		installTargets = append(installTargets, op.Target)
	}
	require.NotEmpty(t, installTargets)

	// Remember which targets were already present from the initial install
	// (intersection of old & new profile = reused targets that already exist
	// as symlinks). For "macbook" -> "common" the `config` target is reused.
	preexisting := make(map[string]struct{}, len(initialResult.LinksCreated))
	for _, l := range initialResult.LinksCreated {
		preexisting[l.Target] = struct{}{}
	}

	// Force ExecuteUninstallPlan -> state.Save to fail by making the parent
	// of statePath a regular file (MkdirAll then refuses).
	badParent := filepath.Join(t.TempDir(), "not-a-dir")
	require.NoError(t, os.WriteFile(badParent, []byte("file-not-dir"), 0o644))
	badStatePath := filepath.Join(badParent, "state.json")

	err = ExecuteSwitchPlan(sp, badStatePath)
	require.Error(t, err, "switch must fail when uninstall phase errors")
	assert.Contains(t, err.Error(), "uninstall phase")

	// Install phase MUST NOT have run: any new-only target must not exist.
	for _, tgt := range installTargets {
		if _, existed := preexisting[tgt]; existed {
			continue
		}
		_, statErr := os.Lstat(tgt)
		assert.True(t, os.IsNotExist(statErr),
			"install phase must not run after uninstall failure; target %q exists", tgt)
	}

	// The original (good) state file must be untouched: package still on profile A.
	st, err := state.Load(req.StatePath)
	require.NoError(t, err)
	pkg, ok := st["ghostty"]
	require.True(t, ok, "original state must be unchanged after failed switch")
	assert.Equal(t, "macbook", pkg.Profile, "state profile must remain unchanged on failure")
}

// TestExecuteSwitchPlan_StateReflectsNewProfile loads the state file after a
// successful switch and asserts the profile field matches the new profile.
func TestExecuteSwitchPlan_StateReflectsNewProfile(t *testing.T) {
	req, _ := switchSetup(t, "common", "macbook")

	sp, err := BuildSwitchPlan(req)
	require.NoError(t, err)
	require.NoError(t, ExecuteSwitchPlan(sp, req.StatePath))

	st, err := state.Load(req.StatePath)
	require.NoError(t, err)
	pkg, ok := st["ghostty"]
	require.True(t, ok)
	assert.Equal(t, "macbook", pkg.Profile,
		"state.json profile field must equal the new profile name")
	assert.NotEmpty(t, pkg.InstalledLinks, "new profile links must be recorded in state")
	assert.False(t, pkg.InstalledAt.IsZero(), "InstalledAt must be set after switch")
}

// TestExecuteSwitchPlan_NoOrphanLinks asserts no symlinks from the old profile
// remain at their old target paths after a successful switch (when the old
// target is not reused by the new profile).
func TestExecuteSwitchPlan_NoOrphanLinks(t *testing.T) {
	req, initialResult := switchSetup(t, "macbook", "common")

	sp, err := BuildSwitchPlan(req)
	require.NoError(t, err)
	require.NoError(t, ExecuteSwitchPlan(sp, req.StatePath))

	st, err := state.Load(req.StatePath)
	require.NoError(t, err)
	pkg, ok := st["ghostty"]
	require.True(t, ok)

	newTargets := make(map[string]struct{}, len(pkg.InstalledLinks))
	for _, l := range pkg.InstalledLinks {
		newTargets[l.Target] = struct{}{}
	}

	// At least one old target should have been orphaned (otherwise the test
	// is degenerate). For macbook -> common, the "extra" file is removed.
	orphanCandidates := 0
	for _, oldLink := range initialResult.LinksCreated {
		if _, reused := newTargets[oldLink.Target]; reused {
			continue
		}
		orphanCandidates++
		fi, statErr := os.Lstat(oldLink.Target)
		if statErr == nil {
			// If anything still exists at that path, it must NOT be our old symlink.
			isOurs, checkErr := symlink.IsSymlinkTo(oldLink.Target, oldLink.Source)
			require.NoError(t, checkErr)
			assert.False(t, isOurs,
				"orphan symlink from old profile remains at %q (mode=%v)",
				oldLink.Target, fi.Mode())
		} else {
			assert.True(t, os.IsNotExist(statErr),
				"unexpected stat error on old target %q: %v", oldLink.Target, statErr)
		}
	}
	require.Greater(t, orphanCandidates, 0,
		"test setup must produce at least one old-only target to verify orphan removal")
}

func TestExecuteSwitchPlan_NilPlanRejected(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")

	require.Error(t, ExecuteSwitchPlan(nil, statePath))
	require.Error(t, ExecuteSwitchPlan(&SwitchPlan{}, statePath))
	require.Error(t, ExecuteSwitchPlan(&SwitchPlan{Uninstall: &plan.Plan{}}, statePath))
	require.Error(t, ExecuteSwitchPlan(&SwitchPlan{Install: &plan.Plan{}}, statePath))
}

// TestExecuteSwitchPlan_InstallPhaseFailureSurfacesRecoveryHint forces the
// install phase to fail by planting a regular file at the parent directory of
// a planned target. The uninstall phase succeeds first; the install phase
// then errors and the returned error must include a recovery hint and the
// package name.
func TestExecuteSwitchPlan_InstallPhaseFailureSurfacesRecoveryHint(t *testing.T) {
	req, _ := switchSetup(t, "macbook", "common")

	sp, err := BuildSwitchPlan(req)
	require.NoError(t, err)
	require.NotEmpty(t, sp.Install.Ops)

	// Plant a regular file at .config/ghostty so MkdirAll for any nested
	// install target fails with "not a directory". Remove the existing
	// directory first (it was created by the initial install).
	blocker := filepath.Join(req.HomeDir, ".config", "ghostty")
	require.NoError(t, os.RemoveAll(blocker))
	require.NoError(t, os.MkdirAll(filepath.Dir(blocker), 0o755))
	require.NoError(t, os.WriteFile(blocker, []byte("blocker"), 0o644))

	err = ExecuteSwitchPlan(sp, req.StatePath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "uninstalled")
	assert.Contains(t, err.Error(), "rice install")
	assert.Contains(t, err.Error(), "ghostty")
}

func TestSwitch_ConvenienceWrapperEndToEnd(t *testing.T) {
	req, _ := switchSetup(t, "common", "macbook")

	require.NoError(t, Switch(req))

	st, err := state.Load(req.StatePath)
	require.NoError(t, err)
	pkg, ok := st["ghostty"]
	require.True(t, ok)
	assert.Equal(t, "macbook", pkg.Profile)

	configTarget := filepath.Join(req.HomeDir, ".config", "ghostty", "config")
	fi, err := os.Lstat(configTarget)
	require.NoError(t, err)
	assert.NotZero(t, fi.Mode()&os.ModeSymlink)

	extraTarget := filepath.Join(req.HomeDir, ".config", "ghostty", "extra")
	fi2, err := os.Lstat(extraTarget)
	require.NoError(t, err)
	assert.NotZero(t, fi2.Mode()&os.ModeSymlink)
}
