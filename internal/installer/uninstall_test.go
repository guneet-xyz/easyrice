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
