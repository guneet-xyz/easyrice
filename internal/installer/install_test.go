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

// fixtureRepo copies testdata/install into a temp dir and returns its path.
// We copy because some tests modify the repo and we want isolation.
func fixtureRepo(t *testing.T) string {
	t.Helper()
	src := filepath.Join("..", "..", "testdata", "install")
	dst := t.TempDir()
	require.NoError(t, copyDir(src, dst))
	return dst
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func newRequest(t *testing.T, repoRoot, profile string) InstallRequest {
	t.Helper()
	homeDir := t.TempDir()
	statePath := filepath.Join(t.TempDir(), "state.json")
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	return InstallRequest{
		RepoRoot:    repoRoot,
		PackageName: "mypkg",
		Profile:     profile,
		CurrentOS:   runtime.GOOS,
		HomeDir:     homeDir,
		StatePath:   statePath,
	}
}

func TestBuildInstallPlan_DoesNotTouchFilesystem(t *testing.T) {
	repo := fixtureRepo(t)
	req := newRequest(t, repo, "macbook")

	p, err := BuildInstallPlan(req)
	require.NoError(t, err)
	require.NotNil(t, p)

	// HomeDir must remain empty
	entries, err := os.ReadDir(req.HomeDir)
	require.NoError(t, err)
	assert.Empty(t, entries, "BuildInstallPlan must not create files in HomeDir")

	// State file must not exist
	_, err = os.Stat(req.StatePath)
	assert.True(t, os.IsNotExist(err), "BuildInstallPlan must not write state file")
}

func TestBuildInstallPlan_MultiSourceProfile(t *testing.T) {
	repo := fixtureRepo(t)
	req := newRequest(t, repo, "macbook")

	p, err := BuildInstallPlan(req)
	require.NoError(t, err)
	require.NotNil(t, p)

	assert.Equal(t, "mypkg", p.PackageName)
	assert.Equal(t, "macbook", p.Profile)
	assert.Empty(t, p.Conflicts)

	// Expect 2 ops: base.toml from common, machine.toml from macbook
	targets := make(map[string]string)
	for _, op := range p.Ops {
		assert.Equal(t, plan.OpCreate, op.Kind)
		targets[op.Target] = op.Source
	}

	expectBase := filepath.Join(req.HomeDir, ".config", "mypkg", "base.toml")
	expectMachine := filepath.Join(req.HomeDir, ".config", "mypkg", "machine.toml")
	assert.Contains(t, targets, expectBase)
	assert.Contains(t, targets, expectMachine)

	// Source paths point into the correct subdir
	assert.Contains(t, targets[expectBase], filepath.Join("mypkg", "common", ".config", "mypkg", "base.toml"))
	assert.Contains(t, targets[expectMachine], filepath.Join("mypkg", "macbook", ".config", "mypkg", "machine.toml"))
}

func TestBuildInstallPlan_SingleSourceDot(t *testing.T) {
	repo := fixtureRepo(t)
	req := newRequest(t, repo, "common")

	p, err := BuildInstallPlan(req)
	require.NoError(t, err)
	require.NotNil(t, p)

	// sources = ["."] walks the whole package root.
	// rice.toml must be skipped.
	for _, op := range p.Ops {
		assert.NotEqual(t, "rice.toml", filepath.Base(op.Source),
			"rice.toml must be skipped")
	}

	// Must include .config/mypkg/config.toml at the package root
	wantTarget := filepath.Join(req.HomeDir, ".config", "mypkg", "config.toml")
	found := false
	for _, op := range p.Ops {
		if op.Target == wantTarget {
			found = true
		}
	}
	assert.True(t, found, "expected target %q in plan ops", wantTarget)
}

func TestBuildInstallPlan_SkipsRiceToml(t *testing.T) {
	repo := fixtureRepo(t)
	req := newRequest(t, repo, "common")

	p, err := BuildInstallPlan(req)
	require.NoError(t, err)

	for _, op := range p.Ops {
		assert.NotEqual(t, "rice.toml", filepath.Base(op.Source))
		assert.NotEqual(t, "rice.toml", filepath.Base(op.Target))
	}
}

func TestBuildInstallPlan_ConflictReturnsError(t *testing.T) {
	repo := fixtureRepo(t)
	req := newRequest(t, repo, "macbook")

	// Pre-create a conflicting file in HomeDir
	conflictPath := filepath.Join(req.HomeDir, ".config", "mypkg", "base.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(conflictPath), 0o755))
	require.NoError(t, os.WriteFile(conflictPath, []byte("pre-existing"), 0o644))

	p, err := BuildInstallPlan(req)
	require.Error(t, err)
	require.NotNil(t, p)
	assert.NotEmpty(t, p.Conflicts)

	conflictTargets := make(map[string]bool)
	for _, c := range p.Conflicts {
		conflictTargets[c.Target] = true
	}
	assert.True(t, conflictTargets[conflictPath])
}

func TestExecuteInstallPlan_CreatesSymlinks(t *testing.T) {
	repo := fixtureRepo(t)
	req := newRequest(t, repo, "macbook")

	p, err := BuildInstallPlan(req)
	require.NoError(t, err)

	result, err := ExecuteInstallPlan(p, req.StatePath)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.LinksCreated)

	for _, link := range result.LinksCreated {
		fi, err := os.Lstat(link.Target)
		require.NoError(t, err, "symlink should exist at %s", link.Target)
		assert.NotZero(t, fi.Mode()&os.ModeSymlink, "target must be a symlink")

		ok, err := symlink.IsSymlinkTo(link.Target, link.Source)
		require.NoError(t, err)
		assert.True(t, ok, "symlink at %s should point to %s", link.Target, link.Source)
	}
}

func TestExecuteInstallPlan_UpdatesState(t *testing.T) {
	repo := fixtureRepo(t)
	req := newRequest(t, repo, "macbook")

	p, err := BuildInstallPlan(req)
	require.NoError(t, err)

	_, err = ExecuteInstallPlan(p, req.StatePath)
	require.NoError(t, err)

	st, err := state.Load(req.StatePath)
	require.NoError(t, err)
	pkg, ok := st["mypkg"]
	require.True(t, ok, "state should contain mypkg")
	assert.Equal(t, "macbook", pkg.Profile)
	assert.Len(t, pkg.InstalledLinks, len(p.Ops))
	assert.False(t, pkg.InstalledAt.IsZero())
}

func TestInstall_UnsupportedOS(t *testing.T) {
	repo := fixtureRepo(t)
	req := newRequest(t, repo, "macbook")
	req.CurrentOS = "plan9"

	_, err := Install(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plan9")
}

func TestInstall_UnknownPackage(t *testing.T) {
	repo := fixtureRepo(t)
	req := newRequest(t, repo, "macbook")
	req.PackageName = "nonexistent"

	_, err := Install(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestInstall_UnknownProfile(t *testing.T) {
	repo := fixtureRepo(t)
	req := newRequest(t, repo, "no-such-profile")

	_, err := Install(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no-such-profile")
}

func TestInstall_Idempotent(t *testing.T) {
	repo := fixtureRepo(t)
	req := newRequest(t, repo, "macbook")

	result1, err := Install(req)
	require.NoError(t, err)
	require.NotEmpty(t, result1.LinksCreated)

	// Second run with the same args must succeed: existing-correct symlinks are not conflicts.
	result2, err := Install(req)
	require.NoError(t, err)
	assert.Equal(t, len(result1.LinksCreated), len(result2.LinksCreated))
}

func TestInstall_FullFlow(t *testing.T) {
	repo := fixtureRepo(t)
	req := newRequest(t, repo, "macbook")

	result, err := Install(req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.LinksCreated)

	// Verify state persisted
	st, err := state.Load(req.StatePath)
	require.NoError(t, err)
	assert.Contains(t, st, "mypkg")

	// Verify all symlinks exist and point correctly
	for _, link := range result.LinksCreated {
		ok, err := symlink.IsSymlinkTo(link.Target, link.Source)
		require.NoError(t, err)
		assert.True(t, ok)
	}
}
