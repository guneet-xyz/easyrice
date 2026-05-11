package installer

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/plan"
	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/guneet-xyz/easyrice/internal/symlink"
)

func TestBuildUninstallPlan_DoesNotTouchFilesystem(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")

	// Set up state with a package
	s := state.State{
		"mypkg": state.PackageState{
			Profile: "macbook",
			InstalledLinks: []state.InstalledLink{
				{
					Source: "/repo/mypkg/config.toml",
					Target: filepath.Join(tempDir, ".config", "mypkg", "config.toml"),
				},
			},
			InstalledAt: time.Now(),
		},
	}
	require.NoError(t, state.Save(statePath, s))

	req := UninstallRequest{
		PackageName: "mypkg",
		StatePath:   statePath,
	}

	p, err := BuildUninstallPlan(req)
	require.NoError(t, err)
	require.NotNil(t, p)

	// Verify state file was not modified
	s2, err := state.Load(statePath)
	require.NoError(t, err)
	assert.Equal(t, s["mypkg"].Profile, s2["mypkg"].Profile)
	assert.Equal(t, s["mypkg"].InstalledLinks, s2["mypkg"].InstalledLinks)
}

func TestBuildUninstallPlan_ReturnsCorrectOps(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")

	// Set up state with multiple links
	s := state.State{
		"mypkg": state.PackageState{
			Profile: "macbook",
			InstalledLinks: []state.InstalledLink{
				{
					Source: "/repo/mypkg/config.toml",
					Target: filepath.Join(tempDir, ".config", "mypkg", "config.toml"),
				},
				{
					Source: "/repo/mypkg/init.vim",
					Target: filepath.Join(tempDir, ".config", "nvim", "init.vim"),
				},
			},
			InstalledAt: time.Now(),
		},
	}
	require.NoError(t, state.Save(statePath, s))

	req := UninstallRequest{
		PackageName: "mypkg",
		StatePath:   statePath,
	}

	p, err := BuildUninstallPlan(req)
	require.NoError(t, err)
	require.NotNil(t, p)

	assert.Equal(t, "mypkg", p.PackageName)
	assert.Equal(t, "macbook", p.Profile)
	assert.Len(t, p.Ops, 2)

	// Verify all ops are OpRemove
	for _, op := range p.Ops {
		assert.Equal(t, plan.OpRemove, op.Kind)
	}

	// Verify targets match
	targets := make(map[string]bool)
	for _, op := range p.Ops {
		targets[op.Target] = true
	}
	assert.True(t, targets[filepath.Join(tempDir, ".config", "mypkg", "config.toml")])
	assert.True(t, targets[filepath.Join(tempDir, ".config", "nvim", "init.vim")])
}

func TestBuildUninstallPlan_PackageNotInstalled(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")

	// Create empty state
	s := state.State{}
	require.NoError(t, state.Save(statePath, s))

	req := UninstallRequest{
		PackageName: "nonexistent",
		StatePath:   statePath,
	}

	p, err := BuildUninstallPlan(req)
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "not installed")
}

func TestBuildUninstallPlan_StateFileNotExist(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "nonexistent", "state.json")

	req := UninstallRequest{
		PackageName: "mypkg",
		StatePath:   statePath,
	}

	p, err := BuildUninstallPlan(req)
	// Should return empty state (not error) per state.Load behavior
	assert.Error(t, err)
	assert.Nil(t, p)
}

func TestExecuteUninstallPlan_RemovesSymlinks(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	homeDir := filepath.Join(tempDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	// Create actual symlinks
	configDir := filepath.Join(homeDir, ".config", "mypkg")
	require.NoError(t, os.MkdirAll(configDir, 0o755))

	source1 := filepath.Join(tempDir, "repo", "config.toml")
	target1 := filepath.Join(configDir, "config.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(source1), 0o755))
	require.NoError(t, os.WriteFile(source1, []byte("config"), 0o644))
	require.NoError(t, symlink.CreateSymlink(source1, target1))

	source2 := filepath.Join(tempDir, "repo", "init.vim")
	target2 := filepath.Join(homeDir, ".config", "nvim", "init.vim")
	require.NoError(t, os.MkdirAll(filepath.Dir(target2), 0o755))
	require.NoError(t, os.WriteFile(source2, []byte("vim"), 0o644))
	require.NoError(t, symlink.CreateSymlink(source2, target2))

	// Set up state
	s := state.State{
		"mypkg": state.PackageState{
			Profile: "macbook",
			InstalledLinks: []state.InstalledLink{
				{Source: source1, Target: target1},
				{Source: source2, Target: target2},
			},
			InstalledAt: time.Now(),
		},
	}
	require.NoError(t, state.Save(statePath, s))

	// Build and execute plan
	req := UninstallRequest{
		PackageName: "mypkg",
		StatePath:   statePath,
	}
	p, err := BuildUninstallPlan(req)
	require.NoError(t, err)

	err = ExecuteUninstallPlan(p, statePath)
	require.NoError(t, err)

	// Verify symlinks are removed
	_, err = os.Lstat(target1)
	assert.True(t, os.IsNotExist(err), "target1 should be removed")

	_, err = os.Lstat(target2)
	assert.True(t, os.IsNotExist(err), "target2 should be removed")

	// Verify package removed from state
	s2, err := state.Load(statePath)
	require.NoError(t, err)
	_, ok := s2["mypkg"]
	assert.False(t, ok, "mypkg should be removed from state")
}

func TestExecuteUninstallPlan_SkipsMissingLinks(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	homeDir := filepath.Join(tempDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	// Create one symlink, but not the other
	source1 := filepath.Join(tempDir, "repo", "config.toml")
	target1 := filepath.Join(homeDir, ".config", "mypkg", "config.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(source1), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(target1), 0o755))
	require.NoError(t, os.WriteFile(source1, []byte("config"), 0o644))
	require.NoError(t, symlink.CreateSymlink(source1, target1))

	// target2 is missing (simulating manual deletion)
	source2 := filepath.Join(tempDir, "repo", "init.vim")
	target2 := filepath.Join(homeDir, ".config", "nvim", "init.vim")
	require.NoError(t, os.WriteFile(source2, []byte("vim"), 0o644))

	// Set up state with both links
	s := state.State{
		"mypkg": state.PackageState{
			Profile: "macbook",
			InstalledLinks: []state.InstalledLink{
				{Source: source1, Target: target1},
				{Source: source2, Target: target2},
			},
			InstalledAt: time.Now(),
		},
	}
	require.NoError(t, state.Save(statePath, s))

	// Build and execute plan
	req := UninstallRequest{
		PackageName: "mypkg",
		StatePath:   statePath,
	}
	p, err := BuildUninstallPlan(req)
	require.NoError(t, err)

	// Should not error even though target2 is missing
	err = ExecuteUninstallPlan(p, statePath)
	require.NoError(t, err)

	// Verify target1 is removed
	_, err = os.Lstat(target1)
	assert.True(t, os.IsNotExist(err), "target1 should be removed")

	// Verify package removed from state
	s2, err := state.Load(statePath)
	require.NoError(t, err)
	_, ok := s2["mypkg"]
	assert.False(t, ok, "mypkg should be removed from state")
}

func TestExecuteUninstallPlan_SkipsReplacedLinks(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	homeDir := filepath.Join(tempDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	// Create one symlink
	source1 := filepath.Join(tempDir, "repo", "config.toml")
	target1 := filepath.Join(homeDir, ".config", "mypkg", "config.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(source1), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(target1), 0o755))
	require.NoError(t, os.WriteFile(source1, []byte("config"), 0o644))
	require.NoError(t, symlink.CreateSymlink(source1, target1))

	// target2 is replaced by a regular file
	source2 := filepath.Join(tempDir, "repo", "init.vim")
	target2 := filepath.Join(homeDir, ".config", "nvim", "init.vim")
	require.NoError(t, os.MkdirAll(filepath.Dir(target2), 0o755))
	require.NoError(t, os.WriteFile(target2, []byte("manual edit"), 0o644))

	// Set up state with both links
	s := state.State{
		"mypkg": state.PackageState{
			Profile: "macbook",
			InstalledLinks: []state.InstalledLink{
				{Source: source1, Target: target1},
				{Source: source2, Target: target2},
			},
			InstalledAt: time.Now(),
		},
	}
	require.NoError(t, state.Save(statePath, s))

	// Build and execute plan
	req := UninstallRequest{
		PackageName: "mypkg",
		StatePath:   statePath,
	}
	p, err := BuildUninstallPlan(req)
	require.NoError(t, err)

	// Should not error even though target2 is a regular file
	err = ExecuteUninstallPlan(p, statePath)
	require.NoError(t, err)

	// Verify target1 is removed
	_, err = os.Lstat(target1)
	assert.True(t, os.IsNotExist(err), "target1 should be removed")

	// Verify target2 still exists (not removed because it's not a symlink)
	_, err = os.Lstat(target2)
	assert.NoError(t, err, "target2 should still exist (regular file)")

	// Verify package removed from state
	s2, err := state.Load(statePath)
	require.NoError(t, err)
	_, ok := s2["mypkg"]
	assert.False(t, ok, "mypkg should be removed from state")
}

func TestExecuteUninstallPlan_SkipsWrongSymlinks(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	homeDir := filepath.Join(tempDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	// Create one symlink
	source1 := filepath.Join(tempDir, "repo", "config.toml")
	target1 := filepath.Join(homeDir, ".config", "mypkg", "config.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(source1), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(target1), 0o755))
	require.NoError(t, os.WriteFile(source1, []byte("config"), 0o644))
	require.NoError(t, symlink.CreateSymlink(source1, target1))

	// target2 is a symlink pointing to a different source
	source2 := filepath.Join(tempDir, "repo", "init.vim")
	target2 := filepath.Join(homeDir, ".config", "nvim", "init.vim")
	otherSource := filepath.Join(tempDir, "other", "init.vim")
	require.NoError(t, os.MkdirAll(filepath.Dir(target2), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(otherSource), 0o755))
	require.NoError(t, os.WriteFile(otherSource, []byte("other"), 0o644))
	require.NoError(t, symlink.CreateSymlink(otherSource, target2))

	// Set up state with both links
	s := state.State{
		"mypkg": state.PackageState{
			Profile: "macbook",
			InstalledLinks: []state.InstalledLink{
				{Source: source1, Target: target1},
				{Source: source2, Target: target2},
			},
			InstalledAt: time.Now(),
		},
	}
	require.NoError(t, state.Save(statePath, s))

	// Build and execute plan
	req := UninstallRequest{
		PackageName: "mypkg",
		StatePath:   statePath,
	}
	p, err := BuildUninstallPlan(req)
	require.NoError(t, err)

	// Should not error even though target2 points elsewhere
	err = ExecuteUninstallPlan(p, statePath)
	require.NoError(t, err)

	// Verify target1 is removed
	_, err = os.Lstat(target1)
	assert.True(t, os.IsNotExist(err), "target1 should be removed")

	// Verify target2 still exists (not removed because it points elsewhere)
	_, err = os.Lstat(target2)
	assert.NoError(t, err, "target2 should still exist (points elsewhere)")

	// Verify package removed from state
	s2, err := state.Load(statePath)
	require.NoError(t, err)
	_, ok := s2["mypkg"]
	assert.False(t, ok, "mypkg should be removed from state")
}

func TestUninstall_EndToEnd(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	homeDir := filepath.Join(tempDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	// Create actual symlinks
	source1 := filepath.Join(tempDir, "repo", "config.toml")
	target1 := filepath.Join(homeDir, ".config", "mypkg", "config.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(source1), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(target1), 0o755))
	require.NoError(t, os.WriteFile(source1, []byte("config"), 0o644))
	require.NoError(t, symlink.CreateSymlink(source1, target1))

	// Set up state
	s := state.State{
		"mypkg": state.PackageState{
			Profile: "macbook",
			InstalledLinks: []state.InstalledLink{
				{Source: source1, Target: target1},
			},
			InstalledAt: time.Now(),
		},
	}
	require.NoError(t, state.Save(statePath, s))

	// Call Uninstall convenience wrapper
	req := UninstallRequest{
		PackageName: "mypkg",
		StatePath:   statePath,
	}
	err := Uninstall(req)
	require.NoError(t, err)

	// Verify symlink is removed
	_, err = os.Lstat(target1)
	assert.True(t, os.IsNotExist(err), "target1 should be removed")

	// Verify package removed from state
	s2, err := state.Load(statePath)
	require.NoError(t, err)
	_, ok := s2["mypkg"]
	assert.False(t, ok, "mypkg should be removed from state")
}

// TestExecuteUninstallPlan_Success installs links via state, then uninstalls them.
// Asserts every symlink is removed and the package key disappears from state.json.
func TestExecuteUninstallPlan_Success(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	homeDir := filepath.Join(tempDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	source1 := filepath.Join(tempDir, "repo", "config.toml")
	target1 := filepath.Join(homeDir, ".config", "mypkg", "config.toml")
	require.NoError(t, os.MkdirAll(filepath.Dir(source1), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(target1), 0o755))
	require.NoError(t, os.WriteFile(source1, []byte("c"), 0o644))
	require.NoError(t, symlink.CreateSymlink(source1, target1))

	source2 := filepath.Join(tempDir, "repo", "init.vim")
	target2 := filepath.Join(homeDir, ".config", "nvim", "init.vim")
	require.NoError(t, os.MkdirAll(filepath.Dir(target2), 0o755))
	require.NoError(t, os.WriteFile(source2, []byte("v"), 0o644))
	require.NoError(t, symlink.CreateSymlink(source2, target2))

	s := state.State{
		"mypkg": state.PackageState{
			Profile: "macbook",
			InstalledLinks: []state.InstalledLink{
				{Source: source1, Target: target1},
				{Source: source2, Target: target2},
			},
			InstalledAt: time.Now(),
		},
	}
	require.NoError(t, state.Save(statePath, s))

	p := &plan.Plan{
		PackageName: "mypkg",
		Profile:     "macbook",
		Ops: []plan.Op{
			{Kind: plan.OpRemove, Source: source1, Target: target1},
			{Kind: plan.OpRemove, Source: source2, Target: target2},
		},
	}

	require.NoError(t, ExecuteUninstallPlan(p, statePath))

	_, err := os.Lstat(target1)
	assert.True(t, os.IsNotExist(err), "target1 should be removed")
	_, err = os.Lstat(target2)
	assert.True(t, os.IsNotExist(err), "target2 should be removed")

	s2, err := state.Load(statePath)
	require.NoError(t, err)
	_, ok := s2["mypkg"]
	assert.False(t, ok, "package key should be absent from state.json")
}

// TestExecuteUninstallPlan_SkipsAlreadyRemovedLinks verifies resilience when a
// link recorded in state has already been deleted from the filesystem.
func TestExecuteUninstallPlan_SkipsAlreadyRemovedLinks(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	homeDir := filepath.Join(tempDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	source1 := filepath.Join(tempDir, "repo", "a")
	target1 := filepath.Join(homeDir, "a")
	require.NoError(t, os.MkdirAll(filepath.Dir(source1), 0o755))
	require.NoError(t, os.WriteFile(source1, []byte("a"), 0o644))
	require.NoError(t, symlink.CreateSymlink(source1, target1))

	source2 := filepath.Join(tempDir, "repo", "b")
	target2 := filepath.Join(homeDir, "b")
	require.NoError(t, os.WriteFile(source2, []byte("b"), 0o644))

	s := state.State{
		"mypkg": state.PackageState{
			Profile: "default",
			InstalledLinks: []state.InstalledLink{
				{Source: source1, Target: target1},
				{Source: source2, Target: target2},
			},
			InstalledAt: time.Now(),
		},
	}
	require.NoError(t, state.Save(statePath, s))

	p := &plan.Plan{
		PackageName: "mypkg",
		Profile:     "default",
		Ops: []plan.Op{
			{Kind: plan.OpRemove, Source: source1, Target: target1},
			{Kind: plan.OpRemove, Source: source2, Target: target2},
		},
	}

	require.NoError(t, ExecuteUninstallPlan(p, statePath))

	_, err := os.Lstat(target1)
	assert.True(t, os.IsNotExist(err))

	s2, err := state.Load(statePath)
	require.NoError(t, err)
	_, ok := s2["mypkg"]
	assert.False(t, ok)
}

// TestExecuteUninstallPlan_StateFileUpdated verifies that after a successful
// uninstall, loading the state file shows the package key has been removed
// while unrelated package entries remain intact.
func TestExecuteUninstallPlan_StateFileUpdated(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	homeDir := filepath.Join(tempDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	source1 := filepath.Join(tempDir, "repo", "x")
	target1 := filepath.Join(homeDir, "x")
	require.NoError(t, os.MkdirAll(filepath.Dir(source1), 0o755))
	require.NoError(t, os.WriteFile(source1, []byte("x"), 0o644))
	require.NoError(t, symlink.CreateSymlink(source1, target1))

	s := state.State{
		"mypkg": state.PackageState{
			Profile:        "default",
			InstalledLinks: []state.InstalledLink{{Source: source1, Target: target1}},
			InstalledAt:    time.Now(),
		},
		"otherpkg": state.PackageState{
			Profile:        "default",
			InstalledLinks: []state.InstalledLink{{Source: "/some/other", Target: "/never/touched"}},
			InstalledAt:    time.Now(),
		},
	}
	require.NoError(t, state.Save(statePath, s))

	p := &plan.Plan{
		PackageName: "mypkg",
		Profile:     "default",
		Ops:         []plan.Op{{Kind: plan.OpRemove, Source: source1, Target: target1}},
	}
	require.NoError(t, ExecuteUninstallPlan(p, statePath))

	loaded, err := state.Load(statePath)
	require.NoError(t, err)
	_, hasMy := loaded["mypkg"]
	assert.False(t, hasMy, "uninstalled package key must be absent")
	_, hasOther := loaded["otherpkg"]
	assert.True(t, hasOther, "unrelated package key must remain")
}

// TestExecuteUninstallPlan_FolderModeReplacedByDirectory exercises the IsDir
// branch where the folder symlink was replaced by a real directory; the op
// must be skipped without error.
func TestExecuteUninstallPlan_FolderModeReplacedByDirectory(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	homeDir := filepath.Join(tempDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	sourceDir := filepath.Join(tempDir, "repo", "nvim")
	require.NoError(t, os.MkdirAll(sourceDir, 0o755))

	targetDir := filepath.Join(homeDir, ".config", "nvim")
	require.NoError(t, os.MkdirAll(targetDir, 0o755))

	s := state.State{
		"nvim": state.PackageState{
			Profile: "default",
			InstalledLinks: []state.InstalledLink{
				{Source: sourceDir, Target: targetDir, IsDir: true},
			},
			InstalledAt: time.Now(),
		},
	}
	require.NoError(t, state.Save(statePath, s))

	p := &plan.Plan{
		PackageName: "nvim",
		Profile:     "default",
		Ops:         []plan.Op{{Kind: plan.OpRemove, Source: sourceDir, Target: targetDir, IsDir: true}},
	}

	require.NoError(t, ExecuteUninstallPlan(p, statePath))

	fi, err := os.Lstat(targetDir)
	require.NoError(t, err)
	assert.True(t, fi.IsDir())
	assert.Equal(t, os.FileMode(0), fi.Mode()&os.ModeSymlink)

	s2, err := state.Load(statePath)
	require.NoError(t, err)
	_, ok := s2["nvim"]
	assert.False(t, ok)
}

// TestExecuteUninstallPlan_FolderModeStillSymlinkRemoved exercises the IsDir
// branch where the folder symlink is still present and ours; it must be removed.
func TestExecuteUninstallPlan_FolderModeStillSymlinkRemoved(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	homeDir := filepath.Join(tempDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	sourceDir := filepath.Join(tempDir, "repo", "nvim")
	require.NoError(t, os.MkdirAll(sourceDir, 0o755))

	targetDir := filepath.Join(homeDir, ".config", "nvim")
	require.NoError(t, os.MkdirAll(filepath.Dir(targetDir), 0o755))
	require.NoError(t, symlink.CreateSymlink(sourceDir, targetDir))

	s := state.State{
		"nvim": state.PackageState{
			Profile: "default",
			InstalledLinks: []state.InstalledLink{
				{Source: sourceDir, Target: targetDir, IsDir: true},
			},
			InstalledAt: time.Now(),
		},
	}
	require.NoError(t, state.Save(statePath, s))

	p := &plan.Plan{
		PackageName: "nvim",
		Profile:     "default",
		Ops:         []plan.Op{{Kind: plan.OpRemove, Source: sourceDir, Target: targetDir, IsDir: true}},
	}

	require.NoError(t, ExecuteUninstallPlan(p, statePath))

	_, err := os.Lstat(targetDir)
	assert.True(t, os.IsNotExist(err), "folder symlink should be removed")
}

// TestExecuteUninstallPlan_StateLoadError exercises the failure path when
// state.Load fails (path is a directory, not a file).
func TestExecuteUninstallPlan_StateLoadError(t *testing.T) {
	tempDir := t.TempDir()
	// Make statePath a directory so os.ReadFile fails.
	statePath := filepath.Join(tempDir, "state.json")
	require.NoError(t, os.MkdirAll(statePath, 0o755))

	p := &plan.Plan{
		PackageName: "mypkg",
		Profile:     "default",
		Ops:         []plan.Op{{Kind: plan.OpRemove, Source: "/x", Target: "/y"}},
	}
	err := ExecuteUninstallPlan(p, statePath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load state")
}

// TestExecuteUninstallPlan_PackageMissingFromState exercises the defensive
// branch where the package key is absent from the loaded state during execution.
// Each op is then skipped without error.
func TestExecuteUninstallPlan_PackageMissingFromState(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	// State exists but does NOT contain the package referenced by the plan.
	require.NoError(t, state.Save(statePath, state.State{}))

	p := &plan.Plan{
		PackageName: "ghost",
		Profile:     "default",
		Ops: []plan.Op{
			{Kind: plan.OpRemove, Source: "/r/a", Target: filepath.Join(tempDir, "a")},
		},
	}
	require.NoError(t, ExecuteUninstallPlan(p, statePath))

	s, err := state.Load(statePath)
	require.NoError(t, err)
	_, ok := s["ghost"]
	assert.False(t, ok)
}

// TestExecuteUninstallPlan_NonRemoveOpsIgnored ensures Op kinds other than
// OpRemove are skipped silently (covers the early-continue branch).
func TestExecuteUninstallPlan_NonRemoveOpsIgnored(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")
	require.NoError(t, state.Save(statePath, state.State{
		"mypkg": state.PackageState{Profile: "default", InstalledAt: time.Now()},
	}))

	p := &plan.Plan{
		PackageName: "mypkg",
		Profile:     "default",
		Ops: []plan.Op{
			{Kind: plan.OpCreate, Source: "/r/a", Target: filepath.Join(tempDir, "a")},
		},
	}
	require.NoError(t, ExecuteUninstallPlan(p, statePath))

	s, err := state.Load(statePath)
	require.NoError(t, err)
	_, ok := s["mypkg"]
	assert.False(t, ok, "package should still be removed from state")
}

// TestExecuteUninstallPlan_FolderModeLstatError exercises the IsDir branch
// where os.Lstat fails because a parent directory is not searchable.
func TestExecuteUninstallPlan_FolderModeLstatError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses directory permissions")
	}
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "state.json")

	// Build an unreadable parent so Lstat on a child fails with EACCES.
	parentDir := filepath.Join(tempDir, "noaccess")
	require.NoError(t, os.MkdirAll(parentDir, 0o755))
	targetDir := filepath.Join(parentDir, "child")
	require.NoError(t, os.MkdirAll(targetDir, 0o755))
	// Strip search permission on the parent.
	require.NoError(t, os.Chmod(parentDir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(parentDir, 0o755) })

	sourceDir := filepath.Join(tempDir, "repo", "x")
	require.NoError(t, os.MkdirAll(sourceDir, 0o755))

	require.NoError(t, state.Save(statePath, state.State{
		"x": state.PackageState{
			Profile: "default",
			InstalledLinks: []state.InstalledLink{
				{Source: sourceDir, Target: targetDir, IsDir: true},
			},
			InstalledAt: time.Now(),
		},
	}))

	p := &plan.Plan{
		PackageName: "x",
		Profile:     "default",
		Ops:         []plan.Op{{Kind: plan.OpRemove, Source: sourceDir, Target: targetDir, IsDir: true}},
	}
	// Resilient: must NOT error, just skip the unreachable target.
	require.NoError(t, ExecuteUninstallPlan(p, statePath))
}
