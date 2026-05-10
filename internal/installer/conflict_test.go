package installer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectConflicts_NoConflicts_TargetsDontExist(t *testing.T) {
	tmpDir := t.TempDir()

	planned := []PlannedLink{
		{Source: "/src/a", Target: filepath.Join(tmpDir, "target1")},
		{Source: "/src/b", Target: filepath.Join(tmpDir, "target2")},
	}

	conflicts := DetectConflicts(planned, nil)
	assert.Empty(t, conflicts)
}

func TestDetectConflicts_NoConflict_SymlinkAlreadyOurs(t *testing.T) {
	tmpDir := t.TempDir()
	source := filepath.Join(tmpDir, "source")
	target := filepath.Join(tmpDir, "target")

	// Create source file
	err := os.WriteFile(source, []byte("content"), 0644)
	require.NoError(t, err)

	// Create symlink pointing to source
	err = os.Symlink(source, target)
	require.NoError(t, err)

	planned := []PlannedLink{
		{Source: source, Target: target},
	}

	conflicts := DetectConflicts(planned, nil)
	assert.Empty(t, conflicts, "symlink already pointing to source should not be a conflict")
}

func TestDetectConflicts_Conflict_RegularFile(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target")

	// Create a regular file at target
	err := os.WriteFile(target, []byte("existing"), 0644)
	require.NoError(t, err)

	planned := []PlannedLink{
		{Source: "/src/a", Target: target},
	}

	conflicts := DetectConflicts(planned, nil)
	require.Len(t, conflicts, 1)
	assert.Equal(t, target, conflicts[0].Target)
	assert.Equal(t, "/src/a", conflicts[0].Source)
	assert.Equal(t, "existing file", conflicts[0].Reason)
}

func TestDetectConflicts_Conflict_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target")

	// Create a directory at target
	err := os.Mkdir(target, 0755)
	require.NoError(t, err)

	planned := []PlannedLink{
		{Source: "/src/a", Target: target},
	}

	conflicts := DetectConflicts(planned, nil)
	require.Len(t, conflicts, 1)
	assert.Equal(t, target, conflicts[0].Target)
	assert.Equal(t, "/src/a", conflicts[0].Source)
	assert.Equal(t, "existing directory", conflicts[0].Reason)
}

func TestDetectConflicts_Conflict_SymlinkPointsElsewhere(t *testing.T) {
	tmpDir := t.TempDir()
	otherSource := filepath.Join(tmpDir, "other_source")
	target := filepath.Join(tmpDir, "target")

	// Create other source file
	err := os.WriteFile(otherSource, []byte("other"), 0644)
	require.NoError(t, err)

	// Create symlink pointing to other source
	err = os.Symlink(otherSource, target)
	require.NoError(t, err)

	planned := []PlannedLink{
		{Source: "/src/a", Target: target},
	}

	conflicts := DetectConflicts(planned, nil)
	require.Len(t, conflicts, 1)
	assert.Equal(t, target, conflicts[0].Target)
	assert.Equal(t, "/src/a", conflicts[0].Source)
	assert.Contains(t, conflicts[0].Reason, "symlink points to")
	assert.Contains(t, conflicts[0].Reason, otherSource)
}

func TestDetectConflicts_IgnoreTargets(t *testing.T) {
	tmpDir := t.TempDir()
	target1 := filepath.Join(tmpDir, "target1")
	target2 := filepath.Join(tmpDir, "target2")

	// Create regular files at both targets
	err := os.WriteFile(target1, []byte("file1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(target2, []byte("file2"), 0644)
	require.NoError(t, err)

	planned := []PlannedLink{
		{Source: "/src/a", Target: target1},
		{Source: "/src/b", Target: target2},
	}

	// Ignore target1
	ignoreTargets := map[string]struct{}{
		target1: {},
	}

	conflicts := DetectConflicts(planned, ignoreTargets)
	require.Len(t, conflicts, 1)
	assert.Equal(t, target2, conflicts[0].Target)
}

func TestDetectConflicts_MultipleConflicts(t *testing.T) {
	tmpDir := t.TempDir()
	target1 := filepath.Join(tmpDir, "target1")
	target2 := filepath.Join(tmpDir, "target2")
	target3 := filepath.Join(tmpDir, "target3")

	// Create a regular file at target1
	err := os.WriteFile(target1, []byte("file"), 0644)
	require.NoError(t, err)

	// Create a directory at target2
	err = os.Mkdir(target2, 0755)
	require.NoError(t, err)

	// Create a symlink pointing elsewhere at target3
	otherSource := filepath.Join(tmpDir, "other")
	err = os.WriteFile(otherSource, []byte("other"), 0644)
	require.NoError(t, err)
	err = os.Symlink(otherSource, target3)
	require.NoError(t, err)

	planned := []PlannedLink{
		{Source: "/src/a", Target: target1},
		{Source: "/src/b", Target: target2},
		{Source: "/src/c", Target: target3},
	}

	conflicts := DetectConflicts(planned, nil)
	require.Len(t, conflicts, 3)

	// Check that all three conflicts are present
	targets := map[string]bool{
		target1: false,
		target2: false,
		target3: false,
	}
	for _, c := range conflicts {
		targets[c.Target] = true
	}
	assert.True(t, targets[target1])
	assert.True(t, targets[target2])
	assert.True(t, targets[target3])
}

func TestConflict_Error(t *testing.T) {
	c := Conflict{
		Target: "/path/to/target",
		Source: "/path/to/source",
		Reason: "existing file",
	}

	errMsg := c.Error()
	assert.Contains(t, errMsg, "/path/to/target")
	assert.Contains(t, errMsg, "existing file")
}
