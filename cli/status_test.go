package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/state"
)

func writeStatusState(t *testing.T, statePath string, s state.State) {
	t.Helper()
	require.NoError(t, state.Save(statePath, s))
}

func TestStatus_NoPackagesInstalled(t *testing.T) {
	resetInstallFlags()
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "No packages installed.")
}

func TestStatus_OnePackageHealthy(t *testing.T) {
	resetInstallFlags()
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	source := filepath.Join(tmp, "src.toml")
	target := filepath.Join(tmp, "tgt.toml")
	require.NoError(t, os.WriteFile(source, []byte("x"), 0o644))
	require.NoError(t, os.Symlink(source, target))

	writeStatusState(t, statePath, state.State{
		"mypkg": state.PackageState{
			Profile: "common",
			InstalledLinks: []state.InstalledLink{
				{Source: source, Target: target},
			},
			InstalledAt: time.Now(),
		},
	})

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Package: mypkg (profile: common)")
	assert.Contains(t, out, "OK")
	assert.Contains(t, out, target)
	assert.NotContains(t, out, "BROKEN")
}

func TestStatus_FilterByPackage(t *testing.T) {
	resetInstallFlags()
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	source := filepath.Join(tmp, "src.toml")
	target := filepath.Join(tmp, "tgt.toml")
	require.NoError(t, os.WriteFile(source, []byte("x"), 0o644))
	require.NoError(t, os.Symlink(source, target))

	writeStatusState(t, statePath, state.State{
		"mypkg": state.PackageState{
			Profile:        "common",
			InstalledLinks: []state.InstalledLink{{Source: source, Target: target}},
			InstalledAt:    time.Now(),
		},
		"otherpkg": state.PackageState{
			Profile:        "macbook",
			InstalledLinks: []state.InstalledLink{{Source: source, Target: target}},
			InstalledAt:    time.Now(),
		},
	})

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status", "mypkg",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Package: mypkg")
	assert.NotContains(t, out, "Package: otherpkg")
}

func TestStatus_BrokenSymlink(t *testing.T) {
	resetInstallFlags()
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	expectedSource := filepath.Join(tmp, "expected.toml")
	otherSource := filepath.Join(tmp, "other.toml")
	target := filepath.Join(tmp, "tgt.toml")
	require.NoError(t, os.WriteFile(expectedSource, []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(otherSource, []byte("y"), 0o644))
	require.NoError(t, os.Symlink(otherSource, target))

	writeStatusState(t, statePath, state.State{
		"mypkg": state.PackageState{
			Profile:        "common",
			InstalledLinks: []state.InstalledLink{{Source: expectedSource, Target: target}},
			InstalledAt:    time.Now(),
		},
	})

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "BROKEN")
}

func TestStatus_FilterUnknownPackagePrintsNothing(t *testing.T) {
	resetInstallFlags()
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	source := filepath.Join(tmp, "src.toml")
	target := filepath.Join(tmp, "tgt.toml")
	require.NoError(t, os.WriteFile(source, []byte("x"), 0o644))
	require.NoError(t, os.Symlink(source, target))

	writeStatusState(t, statePath, state.State{
		"mypkg": state.PackageState{
			Profile:        "common",
			InstalledLinks: []state.InstalledLink{{Source: source, Target: target}},
			InstalledAt:    time.Now(),
		},
	})

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status", "unknownpkg",
	)
	require.NoError(t, err, "out=%s", out)
	assert.NotContains(t, out, "Package:")
	assert.NotContains(t, out, "mypkg")
}
