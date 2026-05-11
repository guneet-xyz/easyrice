package installer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/plan"
	"github.com/guneet-xyz/easyrice/internal/symlink"
)

// TestDetectConflicts_NoConflictWhenTargetMissing verifies that a planned link
// whose target path does not exist on disk produces NO conflict.
func TestDetectConflicts_NoConflictWhenTargetMissing(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "src")
	require.NoError(t, os.WriteFile(source, []byte("x"), 0o644))

	planned := []PlannedLink{
		{Source: source, Target: filepath.Join(tmp, "missing-target")},
	}
	conflicts := DetectConflicts(planned, nil)
	assert.Empty(t, conflicts)
}

// TestDetectConflicts_ConflictWhenRegularFile verifies that an existing
// non-symlink regular file at the target path produces a conflict with the
// "existing file" reason.
func TestDetectConflicts_ConflictWhenRegularFile(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "src")
	target := filepath.Join(tmp, "target")
	require.NoError(t, os.WriteFile(source, []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(target, []byte("pre-existing"), 0o644))

	planned := []PlannedLink{{Source: source, Target: target}}
	conflicts := DetectConflicts(planned, nil)
	require.Len(t, conflicts, 1)
	assert.Equal(t, target, conflicts[0].Target)
	assert.Equal(t, source, conflicts[0].Source)
	assert.Equal(t, "existing file", conflicts[0].Reason)
}

// TestDetectConflicts_NoConflictWhenIdempotentSymlink verifies that when the
// target is already a symlink pointing to the planned source (created via the
// symlink package), DetectConflicts treats it as idempotent (no conflict).
func TestDetectConflicts_NoConflictWhenIdempotentSymlink(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "src")
	target := filepath.Join(tmp, "target")
	require.NoError(t, os.WriteFile(source, []byte("x"), 0o644))

	// Use internal/symlink (not os.Symlink) — the package is the only
	// permitted producer of symlinks in the codebase.
	require.NoError(t, symlink.CreateSymlink(source, target))

	planned := []PlannedLink{{Source: source, Target: target}}
	conflicts := DetectConflicts(planned, nil)
	assert.Empty(t, conflicts, "symlink already pointing to planned source must be idempotent")
}

// TestDetectConflicts_ConflictWhenDifferentSymlink verifies that a target which
// is a symlink pointing to some OTHER path produces a conflict whose Reason
// mentions the actual destination.
func TestDetectConflicts_ConflictWhenDifferentSymlink(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "src")
	other := filepath.Join(tmp, "other")
	target := filepath.Join(tmp, "target")
	require.NoError(t, os.WriteFile(source, []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(other, []byte("y"), 0o644))

	require.NoError(t, symlink.CreateSymlink(other, target))

	planned := []PlannedLink{{Source: source, Target: target}}
	conflicts := DetectConflicts(planned, nil)
	require.Len(t, conflicts, 1)
	assert.Equal(t, target, conflicts[0].Target)
	assert.Equal(t, source, conflicts[0].Source)
	assert.Contains(t, conflicts[0].Reason, "symlink points to")
	assert.Contains(t, conflicts[0].Reason, other)
}

// TestBuildInstallPlan_LastWinsOverlay verifies that when two file-mode sources
// have a file at the same relative path, the LATER source's file wins (i.e.
// the resulting plan op for that target points to the later source).
func TestBuildInstallPlan_LastWinsOverlay(t *testing.T) {
	repoRoot := t.TempDir()

	// Build a synthetic package: pkg root has two source dirs "first" and
	// "second", each containing a file with the same relative path "shared".
	pkgRoot := filepath.Join(repoRoot, "pkg")
	firstDir := filepath.Join(pkgRoot, "first")
	secondDir := filepath.Join(pkgRoot, "second")
	require.NoError(t, os.MkdirAll(firstDir, 0o755))
	require.NoError(t, os.MkdirAll(secondDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(firstDir, "shared"), []byte("first"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(secondDir, "shared"), []byte("second"), 0o644))

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	pkg := manifest.PackageDef{
		Description: "overlay test",
		SupportedOS: []string{runtime.GOOS},
		Root:        "pkg",
	}
	specs := []manifest.SourceSpec{
		{Path: "first", Mode: "file", Target: filepath.Join(homeDir, "out")},
		{Path: "second", Mode: "file", Target: filepath.Join(homeDir, "out")},
	}

	req := InstallRequest{
		RepoRoot:    repoRoot,
		PackageName: "pkg",
		ProfileName: "p",
		Pkg:         &pkg,
		Specs:       specs,
		CurrentOS:   runtime.GOOS,
		HomeDir:     homeDir,
		StatePath:   filepath.Join(t.TempDir(), "state.json"),
	}

	p, err := BuildInstallPlan(req)
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Empty(t, p.Conflicts)

	wantTarget := filepath.Join(homeDir, "out", "shared")
	var op *plan.Op
	for i := range p.Ops {
		if p.Ops[i].Target == wantTarget {
			op = &p.Ops[i]
		}
	}
	require.NotNil(t, op, "expected an op for target %q", wantTarget)
	// Later source ("second") must win.
	assert.Contains(t, op.Source, filepath.Join("pkg", "second", "shared"))
	assert.NotContains(t, op.Source, filepath.Join("pkg", "first", "shared"))

	// Also verify only ONE op exists for the shared target.
	count := 0
	for _, o := range p.Ops {
		if o.Target == wantTarget {
			count++
		}
	}
	assert.Equal(t, 1, count, "last-wins overlay must collapse to one op per target")
}

// TestBuildInstallPlan_NilPkgReturnsError ensures the guard against a nil
// PackageDef pointer fires before any filesystem walk.
func TestBuildInstallPlan_NilPkgReturnsError(t *testing.T) {
	req := InstallRequest{
		PackageName: "pkg",
		ProfileName: "p",
		Pkg:         nil,
		CurrentOS:   runtime.GOOS,
		HomeDir:     t.TempDir(),
		StatePath:   filepath.Join(t.TempDir(), "state.json"),
	}
	p, err := BuildInstallPlan(req)
	require.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "Pkg must not be nil")
}

// TestBuildInstallPlan_SourceIsNotDirectory exercises the branch where the
// declared source path exists but is a regular file, not a directory.
func TestBuildInstallPlan_SourceIsNotDirectory(t *testing.T) {
	repoRoot := t.TempDir()
	pkgRoot := filepath.Join(repoRoot, "pkg")
	require.NoError(t, os.MkdirAll(pkgRoot, 0o755))
	// Create a FILE where a directory is expected.
	require.NoError(t, os.WriteFile(filepath.Join(pkgRoot, "src"), []byte("x"), 0o644))

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	pkg := manifest.PackageDef{SupportedOS: []string{runtime.GOOS}, Root: "pkg"}
	req := InstallRequest{
		RepoRoot:    repoRoot,
		PackageName: "pkg",
		ProfileName: "p",
		Pkg:         &pkg,
		Specs:       []manifest.SourceSpec{{Path: "src", Mode: "file", Target: filepath.Join(homeDir, "out")}},
		CurrentOS:   runtime.GOOS,
		HomeDir:     homeDir,
		StatePath:   filepath.Join(t.TempDir(), "state.json"),
	}
	p, err := BuildInstallPlan(req)
	require.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "is not a directory")
}

// TestBuildInstallPlan_TargetEscapesHome covers the defense-in-depth check
// that rejects targets resolving outside HomeDir.
func TestBuildInstallPlan_TargetEscapesHome(t *testing.T) {
	repoRoot := t.TempDir()
	pkgRoot := filepath.Join(repoRoot, "pkg")
	srcDir := filepath.Join(pkgRoot, "src")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "f"), []byte("x"), 0o644))

	homeDir := t.TempDir()
	escapeTarget := t.TempDir() // sibling of homeDir, definitely outside
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	pkg := manifest.PackageDef{SupportedOS: []string{runtime.GOOS}, Root: "pkg"}
	req := InstallRequest{
		RepoRoot:    repoRoot,
		PackageName: "pkg",
		ProfileName: "p",
		Pkg:         &pkg,
		Specs:       []manifest.SourceSpec{{Path: "src", Mode: "file", Target: escapeTarget}},
		CurrentOS:   runtime.GOOS,
		HomeDir:     homeDir,
		StatePath:   filepath.Join(t.TempDir(), "state.json"),
	}
	p, err := BuildInstallPlan(req)
	require.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "escapes home")
}

// TestBuildInstallPlan_FolderModeTargetEscapesHome covers the same guard
// applied to folder-mode sources (separate code path from file-mode).
func TestBuildInstallPlan_FolderModeTargetEscapesHome(t *testing.T) {
	repoRoot := t.TempDir()
	pkgRoot := filepath.Join(repoRoot, "pkg")
	srcDir := filepath.Join(pkgRoot, "src")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))

	homeDir := t.TempDir()
	escapeTarget := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	pkg := manifest.PackageDef{SupportedOS: []string{runtime.GOOS}, Root: "pkg"}
	req := InstallRequest{
		RepoRoot:    repoRoot,
		PackageName: "pkg",
		ProfileName: "p",
		Pkg:         &pkg,
		Specs:       []manifest.SourceSpec{{Path: "src", Mode: "folder", Target: escapeTarget}},
		CurrentOS:   runtime.GOOS,
		HomeDir:     homeDir,
		StatePath:   filepath.Join(t.TempDir(), "state.json"),
	}
	p, err := BuildInstallPlan(req)
	require.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "escapes home")
}

// TestDetectConflicts_ConflictWhenDirectoryFolderMode covers the IsDir branch
// where a folder-mode planned link finds an existing real directory at target.
func TestDetectConflicts_ConflictWhenDirectoryFolderMode(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "target")
	require.NoError(t, os.Mkdir(target, 0o755))

	planned := []PlannedLink{{Source: "/src", Target: target, IsDir: true}}
	conflicts := DetectConflicts(planned, nil)
	require.Len(t, conflicts, 1)
	assert.True(t, conflicts[0].IsDir)
	assert.Contains(t, conflicts[0].Reason, "folder-mode requires symlink or absent path")
}

// TestDetectConflicts_IgnoreTargetsSkipsCheck verifies that targets in the
// ignoreTargets set are unconditionally skipped, even if they would otherwise
// produce a conflict.
func TestDetectConflicts_IgnoreTargetsSkipsCheck(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "target")
	require.NoError(t, os.WriteFile(target, []byte("would-conflict"), 0o644))

	planned := []PlannedLink{{Source: "/src", Target: target}}
	ignore := map[string]struct{}{target: {}}
	conflicts := DetectConflicts(planned, ignore)
	assert.Empty(t, conflicts, "ignoreTargets entry must suppress conflict detection")
}

// TestDetectConflicts_LstatErrorTreatedAsConflict covers the branch where
// os.Lstat returns an error other than ErrNotExist (here: parent path is a
// non-directory, producing ENOTDIR / "not a directory" on traversal).
func TestDetectConflicts_LstatErrorTreatedAsConflict(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ENOTDIR semantics differ on windows")
	}
	tmp := t.TempDir()
	// Create a regular file, then try to Lstat a path THROUGH it.
	// e.g. /tmp/X/file is a file; /tmp/X/file/child triggers ENOTDIR on Lstat.
	parentFile := filepath.Join(tmp, "notadir")
	require.NoError(t, os.WriteFile(parentFile, []byte("x"), 0o644))
	target := filepath.Join(parentFile, "child")

	planned := []PlannedLink{{Source: "/src", Target: target}}
	conflicts := DetectConflicts(planned, nil)
	require.Len(t, conflicts, 1)
	assert.Equal(t, target, conflicts[0].Target)
	assert.Contains(t, conflicts[0].Reason, "failed to check target")
}

// TestBuildInstallPlan_MissingSourceDir verifies that BuildInstallPlan returns
// a wrapped error when a declared source path does not exist on disk.
func TestBuildInstallPlan_MissingSourceDir(t *testing.T) {
	repoRoot := t.TempDir()
	// Create the package root but NOT the source dir.
	require.NoError(t, os.MkdirAll(filepath.Join(repoRoot, "pkg"), 0o755))

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	pkg := manifest.PackageDef{
		Description: "missing source test",
		SupportedOS: []string{runtime.GOOS},
		Root:        "pkg",
	}
	specs := []manifest.SourceSpec{
		{Path: "does-not-exist", Mode: "file", Target: filepath.Join(homeDir, "out")},
	}

	req := InstallRequest{
		RepoRoot:    repoRoot,
		PackageName: "pkg",
		ProfileName: "p",
		Pkg:         &pkg,
		Specs:       specs,
		CurrentOS:   runtime.GOOS,
		HomeDir:     homeDir,
		StatePath:   filepath.Join(t.TempDir(), "state.json"),
	}

	p, err := BuildInstallPlan(req)
	require.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "does-not-exist")
	assert.Contains(t, err.Error(), "pkg")
}
