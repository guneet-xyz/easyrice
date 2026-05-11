package installer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/manifest"
)

// TestBuildSwitchPlan_InstallPlanNil_WhenSourceDirMissing exercises the
// `installPlan == nil` branch by deleting the source dir of the new profile
// before BuildSwitchPlan runs.
func TestBuildSwitchPlan_InstallPlanNil_WhenSourceDirMissing(t *testing.T) {
	req, _ := switchSetup(t, "macbook", "common")

	// Remove the source dir referenced by the new profile so BuildInstallPlan
	// returns (nil, error) — driving switch.go's installPlan==nil branch.
	require.NoError(t, os.RemoveAll(filepath.Join(req.RepoRoot, "ghostty", "common")))

	sp, err := BuildSwitchPlan(req)
	require.Error(t, err)
	assert.Nil(t, sp)
	assert.Contains(t, err.Error(), "failed to build install plan")
}

// TestExecuteInstallPlan_StateSaveError exercises install.go line 329 (success
// path Save failure). We make the state file's parent dir read-only after
// BuildInstallPlan and before ExecuteInstallPlan so symlink creation succeeds
// but state.Save fails.
func TestExecuteInstallPlan_StateSaveError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based unwritable dir does not behave the same on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses unwritable dir permissions")
	}

	repo := fixtureRepo(t)
	req := newRequest(t, repo, "ghostty", "common")

	stateDir := filepath.Dir(req.StatePath)
	require.NoError(t, os.MkdirAll(stateDir, 0o755))

	p, err := BuildInstallPlan(req)
	require.NoError(t, err)
	require.NotNil(t, p)

	require.NoError(t, os.Chmod(stateDir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(stateDir, 0o755) })

	_, execErr := ExecuteInstallPlan(p, req.StatePath)
	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "save state")
}

// TestBuildInstallPlan_SourceIsFile covers install.go line 115 — when the
// resolved source is a plain file, not a directory, BuildInstallPlan returns
// an error.
func TestBuildInstallPlan_SourceIsFile(t *testing.T) {
	repoRoot := t.TempDir()
	homeDir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(repoRoot, "pkg"), 0o755))
	// "src" is a file, not a directory.
	require.NoError(t, os.WriteFile(
		filepath.Join(repoRoot, "pkg", "src"),
		[]byte("data"), 0o644,
	))

	pkg := &manifest.PackageDef{
		SupportedOS: []string{runtime.GOOS},
		Root:        "pkg",
		Profiles: map[string]manifest.ProfileDef{
			"default": {
				Sources: []manifest.SourceSpec{
					{Path: "src", Mode: "file", Target: "$HOME/x"},
				},
			},
		},
	}

	t.Setenv("HOME", homeDir)
	req := InstallRequest{
		RepoRoot:    repoRoot,
		PackageName: "pkg",
		ProfileName: "default",
		Pkg:         pkg,
		Specs:       pkg.Profiles["default"].Sources,
		CurrentOS:   runtime.GOOS,
		HomeDir:     homeDir,
		StatePath:   filepath.Join(t.TempDir(), "state.json"),
	}

	_, err := BuildInstallPlan(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not a directory")
}

// TestBuildInstallPlan_NilPkg covers install.go line 70-72: passing Pkg=nil
// must return an error.
func TestBuildInstallPlan_NilPkg(t *testing.T) {
	req := InstallRequest{
		RepoRoot:    t.TempDir(),
		PackageName: "missing",
		Pkg:         nil,
		HomeDir:     t.TempDir(),
		CurrentOS:   runtime.GOOS,
	}
	_, err := BuildInstallPlan(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Pkg must not be nil")
}

// TestBuildSwitchPlan_StateLoadError covers switch.go line 45 — corrupt state
// file makes state.Load return an error.
func TestBuildSwitchPlan_StateLoadError(t *testing.T) {
	repo := fixtureRepo(t)
	homeDir := t.TempDir()
	statePath := filepath.Join(t.TempDir(), "state.json")
	require.NoError(t, os.WriteFile(statePath, []byte("{not valid json"), 0o644))

	sp, err := BuildSwitchPlan(SwitchRequest{
		RepoRoot:    repo,
		PackageName: "ghostty",
		NewProfile:  "common",
		CurrentOS:   runtime.GOOS,
		HomeDir:     homeDir,
		StatePath:   statePath,
	})
	require.Error(t, err)
	assert.Nil(t, sp)
	assert.Contains(t, err.Error(), "failed to load state")
}

// TestBuildSwitchPlan_ManifestMissing covers switch.go line 74 — rice.toml
// removed after install makes manifest.LoadFile fail.
func TestBuildSwitchPlan_ManifestMissing(t *testing.T) {
	req, _ := switchSetup(t, "macbook", "common")
	require.NoError(t, os.Remove(filepath.Join(req.RepoRoot, "rice.toml")))

	sp, err := BuildSwitchPlan(req)
	require.Error(t, err)
	assert.Nil(t, sp)
	assert.Contains(t, err.Error(), "failed to load manifest")
}
