package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func folderFixtureRepo(t *testing.T) string {
	t.Helper()
	src := filepath.Join("..", "..", "testdata", "install_v2")
	dst := t.TempDir()
	require.NoError(t, copyDir(src, dst))
	return dst
}

func TestBuildInstallPlan_FolderMode(t *testing.T) {
	repo := folderFixtureRepo(t)
	// nvim uses folder mode with root = "nvim-custom", profile "default"
	req := newRequest(t, repo, "nvim", "default")

	p, err := BuildInstallPlan(req)
	require.NoError(t, err)
	require.NotNil(t, p)

	require.Len(t, p.Ops, 1, "folder-mode source must emit exactly one op")
	op := p.Ops[0]
	assert.True(t, op.IsDir, "folder-mode op must have IsDir=true")

	assert.True(t, strings.HasSuffix(op.Source, string(os.PathSeparator)+"config"),
		"Source should end in /config, got %q", op.Source)
	assert.True(t, filepath.IsAbs(op.Source), "Source should be absolute, got %q", op.Source)

	expectTarget := filepath.Join(req.HomeDir, ".config", "nvim")
	assert.Equal(t, expectTarget, op.Target)

	assert.Empty(t, p.Conflicts)
}

func TestDetectConflicts_FolderMode_NoConflict(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "src")
	require.NoError(t, os.Mkdir(source, 0o755))

	target := filepath.Join(tmp, "abs", "ent", "myfolder")

	planned := []PlannedLink{{Source: source, Target: target, IsDir: true}}
	conflicts := DetectConflicts(planned, nil)
	assert.Empty(t, conflicts, "absent target must not be a conflict")
}

func TestDetectConflicts_FolderMode_ExistingDir(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "src")
	require.NoError(t, os.Mkdir(source, 0o755))

	target := filepath.Join(tmp, "target")
	require.NoError(t, os.Mkdir(target, 0o755))

	planned := []PlannedLink{{Source: source, Target: target, IsDir: true}}
	conflicts := DetectConflicts(planned, nil)
	require.Len(t, conflicts, 1, "existing real directory must conflict in folder-mode")
	assert.Equal(t, target, conflicts[0].Target)
	assert.True(t, conflicts[0].IsDir)
	assert.Contains(t, conflicts[0].Reason, "existing directory")
}

func TestDetectConflicts_FolderMode_OurSymlink(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "src")
	require.NoError(t, os.Mkdir(source, 0o755))

	target := filepath.Join(tmp, "target")
	require.NoError(t, os.Symlink(source, target))

	planned := []PlannedLink{{Source: source, Target: target, IsDir: true}}
	conflicts := DetectConflicts(planned, nil)
	assert.Empty(t, conflicts, "our own symlink must not conflict (idempotent)")
}

func TestDetectConflicts_FolderMode_ForeignSymlink(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "src")
	require.NoError(t, os.Mkdir(source, 0o755))

	otherSource := filepath.Join(tmp, "other_src")
	require.NoError(t, os.Mkdir(otherSource, 0o755))

	target := filepath.Join(tmp, "target")
	require.NoError(t, os.Symlink(otherSource, target))

	planned := []PlannedLink{{Source: source, Target: target, IsDir: true}}
	conflicts := DetectConflicts(planned, nil)
	require.Len(t, conflicts, 1, "foreign symlink must conflict")
	assert.Equal(t, target, conflicts[0].Target)
	assert.True(t, conflicts[0].IsDir)
	assert.Contains(t, conflicts[0].Reason, "symlink points to")
}

func TestBuildInstallPlan_OverlayRejection_TwoFolderSources(t *testing.T) {
	repo := folderFixtureRepo(t)
	req := newRequest(t, repo, "folder-overlay-pkg", "common")

	p, err := BuildInstallPlan(req)
	require.Error(t, err, "two folder-mode sources targeting same path must fail planning")
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "planning error")
}
