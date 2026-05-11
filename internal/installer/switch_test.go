package installer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
