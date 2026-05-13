package installer

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/plan"
	"github.com/guneet-xyz/easyrice/internal/profile"
	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/guneet-xyz/easyrice/internal/symlink"
)

func fixtureRepo(t *testing.T) string {
	t.Helper()
	src := filepath.Join("..", "..", "testdata", "install_v2")
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

func newRequest(t *testing.T, repoRoot, pkgName, profileName string) InstallRequest {
	t.Helper()
	m, err := manifest.LoadFile(filepath.Join(repoRoot, "rice.toml"))
	require.NoError(t, err)
	pkg, ok := m.Packages[pkgName]
	require.True(t, ok, "package %q not found in fixture", pkgName)
	specs, err := profile.ResolveSpecs(repoRoot, &pkg, pkgName, profileName)
	require.NoError(t, err)
	homeDir := t.TempDir()
	statePath := filepath.Join(t.TempDir(), "state.json")
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	return InstallRequest{
		RepoRoot:    repoRoot,
		PackageName: pkgName,
		ProfileName: profileName,
		Pkg:         &pkg,
		Specs:       specs,
		CurrentOS:   runtime.GOOS,
		HomeDir:     homeDir,
		StatePath:   statePath,
	}
}

func TestBuildInstallPlan_DoesNotTouchFilesystem(t *testing.T) {
	repo := fixtureRepo(t)
	req := newRequest(t, repo, "ghostty", "macbook")

	p, err := BuildInstallPlan(req)
	require.NoError(t, err)
	require.NotNil(t, p)

	entries, err := os.ReadDir(req.HomeDir)
	require.NoError(t, err)
	assert.Empty(t, entries, "BuildInstallPlan must not create files in HomeDir")

	_, err = os.Stat(req.StatePath)
	assert.True(t, os.IsNotExist(err), "BuildInstallPlan must not write state file")
}

func TestBuildInstallPlan_MultiSourceProfile(t *testing.T) {
	repo := fixtureRepo(t)
	req := newRequest(t, repo, "ghostty", "macbook")

	p, err := BuildInstallPlan(req)
	require.NoError(t, err)
	require.NotNil(t, p)

	assert.Equal(t, "ghostty", p.PackageName)
	assert.Equal(t, "macbook", p.Profile)
	assert.Empty(t, p.Conflicts)

	targets := make(map[string]string)
	for _, op := range p.Ops {
		assert.Equal(t, plan.OpCreate, op.Kind)
		targets[op.Target] = op.Source
	}

	expectConfig := filepath.Join(req.HomeDir, ".config", "ghostty", "config")
	expectExtra := filepath.Join(req.HomeDir, ".config", "ghostty", "extra")
	assert.Contains(t, targets, expectConfig)
	assert.Contains(t, targets, expectExtra)

	assert.Contains(t, targets[expectConfig], filepath.Join("ghostty", "common", "config"))
	assert.Contains(t, targets[expectExtra], filepath.Join("ghostty", "macbook", "extra"))
}

func TestBuildInstallPlan_SingleSourceProfile(t *testing.T) {
	repo := fixtureRepo(t)
	req := newRequest(t, repo, "ghostty", "common")

	p, err := BuildInstallPlan(req)
	require.NoError(t, err)
	require.NotNil(t, p)

	wantTarget := filepath.Join(req.HomeDir, ".config", "ghostty", "config")
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
	require.NoError(t, os.WriteFile(
		filepath.Join(repo, "ghostty", "common", "rice.toml"),
		[]byte("ignored"), 0o644))
	req := newRequest(t, repo, "ghostty", "common")

	p, err := BuildInstallPlan(req)
	require.NoError(t, err)

	for _, op := range p.Ops {
		assert.NotEqual(t, "rice.toml", filepath.Base(op.Source))
		assert.NotEqual(t, "rice.toml", filepath.Base(op.Target))
	}
}

func TestBuildInstallPlan_ConflictReturnsError(t *testing.T) {
	repo := fixtureRepo(t)
	req := newRequest(t, repo, "ghostty", "macbook")

	conflictPath := filepath.Join(req.HomeDir, ".config", "ghostty", "config")
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
	req := newRequest(t, repo, "ghostty", "macbook")

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
	req := newRequest(t, repo, "ghostty", "macbook")

	p, err := BuildInstallPlan(req)
	require.NoError(t, err)

	_, err = ExecuteInstallPlan(p, req.StatePath)
	require.NoError(t, err)

	st, err := state.Load(req.StatePath)
	require.NoError(t, err)
	pkg, ok := st["ghostty"]
	require.True(t, ok, "state should contain ghostty")
	assert.Equal(t, "macbook", pkg.Profile)
	assert.Len(t, pkg.InstalledLinks, len(p.Ops))
	assert.False(t, pkg.InstalledAt.IsZero())
}

func TestInstall_Idempotent(t *testing.T) {
	repo := fixtureRepo(t)
	req := newRequest(t, repo, "ghostty", "macbook")

	result1, err := Install(req)
	require.NoError(t, err)
	require.NotEmpty(t, result1.LinksCreated)

	result2, err := Install(req)
	require.NoError(t, err)
	assert.Equal(t, len(result1.LinksCreated), len(result2.LinksCreated))
}

func TestInstall_FullFlow(t *testing.T) {
	repo := fixtureRepo(t)
	req := newRequest(t, repo, "ghostty", "macbook")

	result, err := Install(req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.LinksCreated)

	st, err := state.Load(req.StatePath)
	require.NoError(t, err)
	assert.Contains(t, st, "ghostty")

	for _, link := range result.LinksCreated {
		ok, err := symlink.IsSymlinkTo(link.Target, link.Source)
		require.NoError(t, err)
		assert.True(t, ok)
	}
}

func TestBuildInstallPlan_RootDefaultsToName(t *testing.T) {
	repo := fixtureRepo(t)
	req := newRequest(t, repo, "ghostty", "common")

	require.Empty(t, req.Pkg.Root, "ghostty fixture must have empty Root")

	p, err := BuildInstallPlan(req)
	require.NoError(t, err)
	require.NotEmpty(t, p.Ops)

	for _, op := range p.Ops {
		assert.True(t, strings.Contains(op.Source, string(os.PathSeparator)+"ghostty"+string(os.PathSeparator)),
			"source path must traverse the package-named directory, got %q", op.Source)
	}
}

// TestInstall_WrapperPropagatesBuildError covers the Install convenience
// wrapper's Build-error branch: a pre-existing foreign file at a planned
// target makes BuildInstallPlan return an error which the wrapper must
// surface verbatim with a nil result.
func TestInstall_WrapperPropagatesBuildError(t *testing.T) {
	repo := fixtureRepo(t)
	req := newRequest(t, repo, "ghostty", "macbook")

	conflictPath := filepath.Join(req.HomeDir, ".config", "ghostty", "config")
	require.NoError(t, os.MkdirAll(filepath.Dir(conflictPath), 0o755))
	require.NoError(t, os.WriteFile(conflictPath, []byte("pre-existing"), 0o644))

	result, err := Install(req)
	require.Error(t, err)
	assert.Nil(t, result, "wrapper must return nil result when Build fails")

	_, statErr := os.Stat(req.StatePath)
	assert.True(t, os.IsNotExist(statErr), "no state should be written when Build fails")
}

func TestBuildInstallPlan_RootCustom(t *testing.T) {
	repo := fixtureRepo(t)
	req := newRequest(t, repo, "nvim", "default")

	require.Equal(t, "nvim-custom", req.Pkg.Root, "nvim fixture must declare Root=nvim-custom")

	p, err := BuildInstallPlan(req)
	require.NoError(t, err)
	require.NotEmpty(t, p.Ops)

	for _, op := range p.Ops {
		assert.True(t, strings.Contains(op.Source, string(os.PathSeparator)+"nvim-custom"+string(os.PathSeparator)),
			"source path must traverse the custom Root directory, got %q", op.Source)
		assert.False(t, strings.Contains(op.Source, string(os.PathSeparator)+"nvim"+string(os.PathSeparator)+"config"),
			"source path must NOT use package name when Root is set, got %q", op.Source)
	}
}

// TestBuildInstallPlan_AbsoluteSpecPath verifies that when a SourceSpec.Path
// is absolute (as produced by the profile resolver for imported specs), the
// installer uses it directly rather than joining it under the repo root.
func TestBuildInstallPlan_AbsoluteSpecPath(t *testing.T) {
	tmpDir := t.TempDir()
	absSourceDir := filepath.Join(tmpDir, "abs_source")
	require.NoError(t, os.MkdirAll(absSourceDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(absSourceDir, "file.txt"), []byte("content"), 0o644))

	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	repoRoot := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(repoRoot, 0o755))

	req := InstallRequest{
		RepoRoot:    repoRoot,
		PackageName: "testpkg",
		ProfileName: "default",
		Pkg:         &manifest.PackageDef{Root: "testpkg"},
		Specs: []manifest.SourceSpec{
			{Path: absSourceDir, Mode: "file", Target: filepath.Join(homeDir, "config")},
		},
		CurrentOS: runtime.GOOS,
		HomeDir:   homeDir,
		StatePath: filepath.Join(tmpDir, "state.json"),
	}

	p, err := BuildInstallPlan(req)
	require.NoError(t, err)
	require.Len(t, p.Ops, 1)
	expectedSource, _ := filepath.Abs(filepath.Join(absSourceDir, "file.txt"))
	assert.Equal(t, expectedSource, p.Ops[0].Source,
		"absolute spec.Path must be used directly, not joined under repoRoot")
	assert.Equal(t, plan.OpCreate, p.Ops[0].Kind)
}
