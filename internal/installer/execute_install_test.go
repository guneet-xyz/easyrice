package installer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/plan"
	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/guneet-xyz/easyrice/internal/symlink"
)

func makeSourceFile(t *testing.T, dir, name, contents string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(contents), 0o644))
	abs, err := filepath.Abs(p)
	require.NoError(t, err)
	return abs
}

func twoOpPlan(t *testing.T, repoDir, homeDir string) *plan.Plan {
	t.Helper()
	src1 := makeSourceFile(t, filepath.Join(repoDir, "pkg"), "a.conf", "alpha")
	src2 := makeSourceFile(t, filepath.Join(repoDir, "pkg"), "b.conf", "bravo")
	return &plan.Plan{
		PackageName: "pkg",
		Profile:     "default",
		Ops: []plan.Op{
			{Kind: plan.OpCreate, Source: src1, Target: filepath.Join(homeDir, ".config", "pkg", "a.conf")},
			{Kind: plan.OpCreate, Source: src2, Target: filepath.Join(homeDir, ".config", "pkg", "b.conf")},
		},
	}
}

func TestExecuteInstallPlan_Success(t *testing.T) {
	repoDir := t.TempDir()
	homeDir := t.TempDir()
	statePath := filepath.Join(t.TempDir(), "state.json")

	p := twoOpPlan(t, repoDir, homeDir)

	result, err := ExecuteInstallPlan(p, statePath)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.LinksCreated, 2)

	for _, op := range p.Ops {
		ok, err := symlink.IsSymlinkTo(op.Target, op.Source)
		require.NoError(t, err)
		assert.True(t, ok, "symlink at %s must point to %s", op.Target, op.Source)
	}

	st, err := state.Load(statePath)
	require.NoError(t, err)
	pkgState, ok := st["pkg"]
	require.True(t, ok, "state should contain pkg")
	assert.Equal(t, "default", pkgState.Profile)
	assert.Len(t, pkgState.InstalledLinks, 2)
	assert.False(t, pkgState.InstalledAt.IsZero())
}

func TestExecuteInstallPlan_PartialFailure(t *testing.T) {
	repoDir := t.TempDir()
	homeDir := t.TempDir()
	statePath := filepath.Join(t.TempDir(), "state.json")

	src1 := makeSourceFile(t, filepath.Join(repoDir, "pkg"), "a.conf", "alpha")
	src2 := makeSourceFile(t, filepath.Join(repoDir, "pkg"), "b.conf", "bravo")

	// Plant a regular file where the second op's parent dir would live so
	// MkdirAll on a path beneath it fails, forcing the second op to error.
	blocker := filepath.Join(homeDir, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("not a dir"), 0o644))

	p := &plan.Plan{
		PackageName: "pkg",
		Profile:     "default",
		Ops: []plan.Op{
			{Kind: plan.OpCreate, Source: src1, Target: filepath.Join(homeDir, "good", "a.conf")},
			{Kind: plan.OpCreate, Source: src2, Target: filepath.Join(blocker, "sub", "b.conf")},
		},
	}

	result, err := ExecuteInstallPlan(p, statePath)
	require.Error(t, err, "partial failure must return an error")
	require.NotNil(t, result)

	ok, lerr := symlink.IsSymlinkTo(p.Ops[0].Target, p.Ops[0].Source)
	require.NoError(t, lerr)
	assert.True(t, ok, "first symlink should have been created")

	blockerInfo, blockerErr := os.Lstat(blocker)
	require.NoError(t, blockerErr)
	assert.True(t, blockerInfo.Mode().IsRegular(), "blocker must remain a regular file (proves second op never created a directory)")

	assert.Len(t, result.LinksCreated, 1)
	assert.Equal(t, p.Ops[0].Target, result.LinksCreated[0].Target)

	st, err := state.Load(statePath)
	require.NoError(t, err)
	pkgState, present := st["pkg"]
	require.True(t, present, "partial state must still record the package")
	assert.Equal(t, "default", pkgState.Profile)
	assert.Len(t, pkgState.InstalledLinks, 1, "state must record only the successful op")
	assert.Equal(t, p.Ops[0].Target, pkgState.InstalledLinks[0].Target)
}

func TestExecuteInstallPlan_IdempotentOnRerun(t *testing.T) {
	repoDir := t.TempDir()
	homeDir := t.TempDir()
	statePath := filepath.Join(t.TempDir(), "state.json")

	p := twoOpPlan(t, repoDir, homeDir)

	res1, err := ExecuteInstallPlan(p, statePath)
	require.NoError(t, err)
	require.Len(t, res1.LinksCreated, 2)

	// Rerun: existing symlinks already point to our sources, so the installer's
	// idempotency branch must treat them as success rather than erroring on
	// "target already exists".
	res2, err := ExecuteInstallPlan(p, statePath)
	require.NoError(t, err, "second run must not error (idempotent)")
	require.NotNil(t, res2)
	assert.Len(t, res2.LinksCreated, 2)

	for _, op := range p.Ops {
		ok, lerr := symlink.IsSymlinkTo(op.Target, op.Source)
		require.NoError(t, lerr)
		assert.True(t, ok)
	}

	st, err := state.Load(statePath)
	require.NoError(t, err)
	pkgState, ok := st["pkg"]
	require.True(t, ok)
	assert.Len(t, pkgState.InstalledLinks, 2)
}

func TestExecuteInstallPlan_StateFileWritten(t *testing.T) {
	repoDir := t.TempDir()
	homeDir := t.TempDir()
	statePath := filepath.Join(t.TempDir(), "state.json")

	p := twoOpPlan(t, repoDir, homeDir)

	_, err := ExecuteInstallPlan(p, statePath)
	require.NoError(t, err)

	_, statErr := os.Stat(statePath)
	require.NoError(t, statErr, "state file must be written")

	st, err := state.Load(statePath)
	require.NoError(t, err)

	pkgState, ok := st["pkg"]
	require.True(t, ok, "loaded state must contain pkg")
	assert.Equal(t, "default", pkgState.Profile)
	require.Len(t, pkgState.InstalledLinks, 2)

	wantByTarget := map[string]string{
		p.Ops[0].Target: p.Ops[0].Source,
		p.Ops[1].Target: p.Ops[1].Source,
	}
	for _, link := range pkgState.InstalledLinks {
		wantSrc, present := wantByTarget[link.Target]
		require.True(t, present, "unexpected link target in state: %s", link.Target)
		assert.Equal(t, wantSrc, link.Source)
		assert.False(t, link.IsDir, "file-mode op must be IsDir=false")
	}
	assert.False(t, pkgState.InstalledAt.IsZero())
}

func TestExecuteInstallPlan_CorruptStateFile(t *testing.T) {
	repoDir := t.TempDir()
	homeDir := t.TempDir()
	statePath := filepath.Join(t.TempDir(), "state.json")
	require.NoError(t, os.WriteFile(statePath, []byte("{not json"), 0o644))

	p := twoOpPlan(t, repoDir, homeDir)

	_, err := ExecuteInstallPlan(p, statePath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load state")
}

func TestExecuteInstallPlan_ForeignFileAtTarget(t *testing.T) {
	repoDir := t.TempDir()
	homeDir := t.TempDir()
	statePath := filepath.Join(t.TempDir(), "state.json")

	src := makeSourceFile(t, filepath.Join(repoDir, "pkg"), "a.conf", "alpha")
	target := filepath.Join(homeDir, ".config", "pkg", "a.conf")
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
	require.NoError(t, os.WriteFile(target, []byte("foreign"), 0o644))

	p := &plan.Plan{
		PackageName: "pkg",
		Profile:     "default",
		Ops: []plan.Op{
			{Kind: plan.OpCreate, Source: src, Target: target},
		},
	}

	result, err := ExecuteInstallPlan(p, statePath)
	require.Error(t, err, "foreign file at target must produce an error")
	require.NotNil(t, result)
	assert.Empty(t, result.LinksCreated)

	data, readErr := os.ReadFile(target)
	require.NoError(t, readErr)
	assert.Equal(t, "foreign", string(data), "foreign file must be preserved")
}

func TestExecuteInstallPlan_SkipsOpRemove(t *testing.T) {
	repoDir := t.TempDir()
	homeDir := t.TempDir()
	statePath := filepath.Join(t.TempDir(), "state.json")

	src := makeSourceFile(t, filepath.Join(repoDir, "pkg"), "a.conf", "alpha")
	target := filepath.Join(homeDir, ".config", "pkg", "a.conf")

	p := &plan.Plan{
		PackageName: "pkg",
		Profile:     "default",
		Ops: []plan.Op{
			{Kind: plan.OpRemove, Source: "", Target: filepath.Join(homeDir, "ignored")},
			{Kind: plan.OpCreate, Source: src, Target: target},
		},
	}

	result, err := ExecuteInstallPlan(p, statePath)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.LinksCreated, 1, "OpRemove must be skipped, only OpCreate counted")

	ok, lerr := symlink.IsSymlinkTo(target, src)
	require.NoError(t, lerr)
	assert.True(t, ok)
}
