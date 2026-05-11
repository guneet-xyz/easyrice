package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/deps"
	"github.com/guneet-xyz/easyrice/internal/repo"
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

func writeRepoManifest(t *testing.T, contents string) string {
	t.Helper()
	repoRoot := repo.DefaultRepoPath()
	require.NoError(t, os.MkdirAll(repoRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "rice.toml"), []byte(contents), 0o644))
	return repoRoot
}

func installedPkgState(t *testing.T, tmp, pkgName string) (string, state.State) {
	t.Helper()
	source := filepath.Join(tmp, pkgName+"-src")
	target := filepath.Join(tmp, pkgName+"-tgt")
	require.NoError(t, os.WriteFile(source, []byte("x"), 0o644))
	require.NoError(t, os.Symlink(source, target))
	return filepath.Join(tmp, "state.json"), state.State{
		pkgName: state.PackageState{
			Profile:        "common",
			InstalledLinks: []state.InstalledLink{{Source: source, Target: target}},
			InstalledAt:    time.Now(),
		},
	}
}

// TestStatus_FilterRepoMissing pins the silent-nil contract: when the managed
// repo does not exist, runStatus must NOT print "Warning: dependency check
// failed" (the absence assertion is the whole point).
func TestStatus_FilterRepoMissing(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	tmp := t.TempDir()
	statePath, st := installedPkgState(t, tmp, "mypkg")
	writeStatusState(t, statePath, st)

	repoRoot := repo.DefaultRepoPath()
	_, statErr := os.Stat(repoRoot)
	require.True(t, os.IsNotExist(statErr), "repo dir should not exist; got %v", statErr)

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status", "mypkg",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Package: mypkg")
	assert.NotContains(t, out, "Warning: dependency check failed")
	assert.NotContains(t, out, "Declared dependencies:")
}

func TestStatus_FilterManifestError(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	tmp := t.TempDir()
	statePath, st := installedPkgState(t, tmp, "mypkg")
	writeStatusState(t, statePath, st)

	writeRepoManifest(t, "this is not = valid toml [[[")

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status", "mypkg",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Package: mypkg")
	assert.Contains(t, out, "Warning: dependency check failed")
	assert.Contains(t, out, "load manifest")
}

func TestStatus_FilterPackageNotInManifest(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	tmp := t.TempDir()
	statePath, st := installedPkgState(t, tmp, "mypkg")
	writeStatusState(t, statePath, st)

	writeRepoManifest(t, `schema_version = 1

[packages.otherpkg]
description = "Some other package"
supported_os = ["linux", "darwin", "windows"]

[packages.otherpkg.profiles.common]
sources = [{path = "x", mode = "file", target = "$HOME"}]
`)

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status", "mypkg",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Package: mypkg")
	assert.NotContains(t, out, "Warning: dependency check failed")
	assert.NotContains(t, out, "Declared dependencies:")
}

func TestStatus_FilterEmptyDependencies(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	tmp := t.TempDir()
	statePath, st := installedPkgState(t, tmp, "mypkg")
	writeStatusState(t, statePath, st)

	writeRepoManifest(t, `schema_version = 1

[packages.mypkg]
description = "Test package, no deps"
supported_os = ["linux", "darwin", "windows"]

[packages.mypkg.profiles.common]
sources = [{path = "x", mode = "file", target = "$HOME"}]
`)

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status", "mypkg",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Package: mypkg")
	assert.NotContains(t, out, "Warning: dependency check failed")
	assert.NotContains(t, out, "Declared dependencies:")
}

func TestStatus_FilterCheckError(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	tmp := t.TempDir()
	statePath, st := installedPkgState(t, tmp, "mypkg")
	writeStatusState(t, statePath, st)

	writeRepoManifest(t, `schema_version = 1

[packages.mypkg]
description = "Test package with bogus dep"
supported_os = ["linux", "darwin", "windows"]
dependencies = [{name = "definitely-not-a-real-dep-xyz"}]

[packages.mypkg.profiles.common]
sources = [{path = "x", mode = "file", target = "$HOME"}]
`)

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status", "mypkg",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Package: mypkg")
	assert.Contains(t, out, "Warning: dependency check failed")
	assert.Contains(t, out, "check dependencies")
}

func TestStatus_FilterRendersDeclaredDependencies(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	tmp := t.TempDir()
	statePath, st := installedPkgState(t, tmp, "mypkg")
	writeStatusState(t, statePath, st)

	writeRepoManifest(t, `schema_version = 1

[packages.mypkg]
description = "Test package using a registry dep"
supported_os = ["linux", "darwin", "windows"]
dependencies = [{name = "git"}]

[packages.mypkg.profiles.common]
sources = [{path = "x", mode = "file", target = "$HOME"}]
`)

	mock := &deps.MockRunner{
		Expectations: []deps.MockExpectation{
			{Argv: []string{"git", "--version"}, Result: deps.RunResult{ExitCode: 0, Combined: []byte("git version 2.42.0\n")}},
		},
	}
	withMockDepsRunner(t, mock)

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status", "mypkg",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Package: mypkg")
	assert.Contains(t, out, "Declared dependencies:")
	assert.NotContains(t, out, "Warning: dependency check failed")
}

func TestStatus_LoadStateError(t *testing.T) {
	resetInstallFlags()
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	require.NoError(t, os.WriteFile(statePath, []byte("{not valid json"), 0o644))

	_, err := runInstallCmd(t, "",
		"--state", statePath,
		"status",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load state")
}

func TestStatus_PrintsInstalledDependencies(t *testing.T) {
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
			InstalledDependencies: []deps.InstalledDependency{
				{Name: "neovim", Version: "0.9.0", Method: "brew", InstalledAt: time.Now()},
			},
		},
	})

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Installed dependencies:")
	assert.Contains(t, out, "neovim")
	assert.Contains(t, out, "0.9.0")
	assert.Contains(t, out, "brew")
}
