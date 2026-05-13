package installer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/guneet-xyz/easyrice/internal/symlink"
)

func newConvergeRequest(t *testing.T, repoRoot, pkgName, profileName string) ConvergeRequest {
	t.Helper()
	m, err := manifest.LoadFile(filepath.Join(repoRoot, "rice.toml"))
	require.NoError(t, err)
	pkg, ok := m.Packages[pkgName]
	require.True(t, ok, "package %q not found", pkgName)
	homeDir := t.TempDir()
	statePath := filepath.Join(t.TempDir(), "state.json")
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	return ConvergeRequest{
		RepoRoot:         repoRoot,
		PackageName:      pkgName,
		RequestedProfile: profileName,
		CurrentOS:        runtime.GOOS,
		HomeDir:          homeDir,
		StatePath:        statePath,
		Pkg:              &pkg,
		Manifest:         m,
	}
}

func TestBuildConvergePlan_NotInstalled(t *testing.T) {
	repo := fixtureRepo(t)
	req := newConvergeRequest(t, repo, "ghostty", "macbook")

	cr, err := BuildConvergePlan(req)
	require.NoError(t, err)
	require.NotNil(t, cr)
	assert.Equal(t, OutcomeInstalled, cr.Outcome)
	assert.Equal(t, "macbook", cr.NewProfile)
	assert.Empty(t, cr.OldProfile)
	require.NotNil(t, cr.Plan)
	assert.NotEmpty(t, cr.Plan.Ops)
}

func TestBuildConvergePlan_SameProfileClean(t *testing.T) {
	repo := fixtureRepo(t)
	req := newConvergeRequest(t, repo, "ghostty", "macbook")

	require.NoError(t, ExecuteConvergePlan(req, mustBuild(t, req)))

	cr2, err := BuildConvergePlan(req)
	require.NoError(t, err)
	assert.Equal(t, OutcomeNoOp, cr2.Outcome)
	assert.Equal(t, "macbook", cr2.NewProfile)
	assert.NotEmpty(t, cr2.LinksAfter)
}

func TestBuildConvergePlan_SameProfileBroken(t *testing.T) {
	repo := fixtureRepo(t)
	req := newConvergeRequest(t, repo, "ghostty", "macbook")

	require.NoError(t, ExecuteConvergePlan(req, mustBuild(t, req)))

	st, err := state.Load(req.StatePath)
	require.NoError(t, err)
	links := st["ghostty"].InstalledLinks
	require.NotEmpty(t, links)
	require.NoError(t, os.Remove(links[0].Target))

	cr2, err := BuildConvergePlan(req)
	require.NoError(t, err)
	assert.Equal(t, OutcomeRepaired, cr2.Outcome)
	require.NotNil(t, cr2.Plan)
	assert.NotEmpty(t, cr2.Plan.Ops)

	require.NoError(t, ExecuteConvergePlan(req, cr2))

	ok, err := symlink.IsSymlinkTo(links[0].Target, links[0].Source)
	require.NoError(t, err)
	assert.True(t, ok, "broken link must be repaired")
}

func TestBuildConvergePlan_ProfileSwitch(t *testing.T) {
	repo := fixtureRepo(t)
	req := newConvergeRequest(t, repo, "ghostty", "common")

	require.NoError(t, ExecuteConvergePlan(req, mustBuild(t, req)))

	switchReq := req
	switchReq.RequestedProfile = "macbook"

	cr2, err := BuildConvergePlan(switchReq)
	require.NoError(t, err)
	assert.Equal(t, OutcomeProfileSwitched, cr2.Outcome)
	assert.Equal(t, "common", cr2.OldProfile)
	assert.Equal(t, "macbook", cr2.NewProfile)
	require.NotNil(t, cr2.Plan)

	require.NoError(t, ExecuteConvergePlan(switchReq, cr2))

	st, err := state.Load(switchReq.StatePath)
	require.NoError(t, err)
	assert.Equal(t, "macbook", st["ghostty"].Profile)
}

func TestBuildConvergePlan_AbsoluteSpecPath(t *testing.T) {
	tmpDir := t.TempDir()
	absSourceDir := filepath.Join(tmpDir, "abs_source")
	require.NoError(t, os.MkdirAll(absSourceDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(absSourceDir, "file.txt"), []byte("x"), 0o644))

	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))
	statePath := filepath.Join(tmpDir, "state.json")
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	repoRoot := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(repoRoot, 0o755))

	pkg := manifest.PackageDef{
		Description: "absimport",
		SupportedOS: []string{"linux", "darwin"},
		Root:        "absimport",
		Profiles: map[string]manifest.ProfileDef{
			"default": {
				Sources: []manifest.SourceSpec{
					{Path: absSourceDir, Mode: "file", Target: filepath.Join(homeDir, "config")},
				},
			},
		},
	}
	m := &manifest.Manifest{SchemaVersion: 1, Packages: map[string]manifest.PackageDef{"absimport": pkg}}

	req := ConvergeRequest{
		RepoRoot:         repoRoot,
		PackageName:      "absimport",
		RequestedProfile: "default",
		CurrentOS:        runtime.GOOS,
		HomeDir:          homeDir,
		StatePath:        statePath,
		Pkg:              &pkg,
		Manifest:         m,
	}

	cr, err := BuildConvergePlan(req)
	require.NoError(t, err)
	require.NotNil(t, cr.Plan)
	require.Len(t, cr.Plan.Ops, 1)
	expectedSource, _ := filepath.Abs(filepath.Join(absSourceDir, "file.txt"))
	assert.Equal(t, expectedSource, cr.Plan.Ops[0].Source)

	require.NoError(t, ExecuteConvergePlan(req, cr))
	ok, err := symlink.IsSymlinkTo(filepath.Join(homeDir, "config", "file.txt"), expectedSource)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestConvergeAll_MultiPackage(t *testing.T) {
	tmpDir := t.TempDir()
	repoRoot := filepath.Join(tmpDir, "repo")
	require.NoError(t, os.MkdirAll(filepath.Join(repoRoot, "pkga", "src"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(repoRoot, "pkgb", "src"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "pkga", "src", "a.txt"), []byte("a"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "pkgb", "src", "b.txt"), []byte("b"), 0o644))

	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	mf := `schema_version = 1
[packages.pkga]
supported_os = ["linux", "darwin"]
[packages.pkga.profiles.default]
sources = [{path = "src", mode = "file", target = "$HOME/a"}]

[packages.pkgb]
supported_os = ["linux", "darwin"]
[packages.pkgb.profiles.default]
sources = [{path = "src", mode = "file", target = "$HOME/b"}]
`
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "rice.toml"), []byte(mf), 0o644))

	m, err := manifest.LoadFile(filepath.Join(repoRoot, "rice.toml"))
	require.NoError(t, err)

	statePath := filepath.Join(tmpDir, "state.json")
	results, err := ConvergeAll(ConvergeAllRequest{
		RepoRoot:       repoRoot,
		DefaultProfile: "default",
		CurrentOS:      runtime.GOOS,
		HomeDir:        homeDir,
		StatePath:      statePath,
		Manifest:       m,
	})
	require.NoError(t, err)
	require.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, OutcomeInstalled, r.Outcome)
	}

	st, err := state.Load(statePath)
	require.NoError(t, err)
	assert.Contains(t, st, "pkga")
	assert.Contains(t, st, "pkgb")
}

func mustBuild(t *testing.T, req ConvergeRequest) *ConvergeResult {
	t.Helper()
	cr, err := BuildConvergePlan(req)
	require.NoError(t, err)
	return cr
}
