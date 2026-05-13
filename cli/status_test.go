package main

import (
	"os"
	"os/exec"
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

func writeRepoManifest(t *testing.T, contents string) string {
	t.Helper()
	repoRoot := repo.DefaultRepoPath()
	require.NoError(t, os.MkdirAll(repoRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "rice.toml"), []byte(contents), 0o644))
	return repoRoot
}

func gitInitRepo(t *testing.T, repoRoot string) {
	t.Helper()
	for _, args := range [][]string{
		{"-C", repoRoot, "init"},
		{"-C", repoRoot, "config", "user.email", "t@t"},
		{"-C", repoRoot, "config", "user.name", "t"},
		{"-C", repoRoot, "checkout", "-B", "main"},
		{"-C", repoRoot, "add", "."},
		{"-C", repoRoot, "commit", "-m", "initial"},
	} {
		out, err := exec.Command("git", args...).CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}
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

const statusManifestMypkg = `schema_version = 1

[packages.mypkg]
description = "Test package"
supported_os = ["linux", "darwin", "windows"]

[packages.mypkg.profiles.common]
sources = [{path = "x", mode = "file", target = "$HOME"}]
`

func TestStatus_RepoNotInitialized(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Rice repo:")
	assert.Contains(t, out, "Rice repo is not initialized")
}

func TestStatus_CleanRepo(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	tmp := t.TempDir()
	statePath, st := installedPkgState(t, tmp, "mypkg")
	writeStatusState(t, statePath, st)

	repoRoot := writeRepoManifest(t, statusManifestMypkg)
	gitInitRepo(t, repoRoot)

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Rice repo: "+repoRoot)
	assert.Contains(t, out, "Git: branch main, clean")
	assert.NotContains(t, out, "Tip: commit your rice changes")
	assert.Contains(t, out, "[OK]")
	assert.Contains(t, out, "mypkg (profile: common)")
}

func TestStatus_DirtyRepo(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	tmp := t.TempDir()
	statePath, st := installedPkgState(t, tmp, "mypkg")
	writeStatusState(t, statePath, st)

	repoRoot := writeRepoManifest(t, statusManifestMypkg)
	gitInitRepo(t, repoRoot)
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "dirty.txt"), []byte("dirty\n"), 0o644))

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Git: branch main, uncommitted changes")
	assert.Contains(t, out, "Tip: commit your rice repo changes to preserve history")
}

func TestStatus_NotInstalled(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	writeStatusState(t, statePath, state.State{})

	repoRoot := writeRepoManifest(t, statusManifestMypkg)
	gitInitRepo(t, repoRoot)

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "[NOT INSTALLED]")
	assert.Contains(t, out, "mypkg")
}

func TestStatus_BrokenLink(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
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

	repoRoot := writeRepoManifest(t, statusManifestMypkg)
	gitInitRepo(t, repoRoot)

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "[BROKEN]")
	assert.Contains(t, out, "mypkg (profile: common)")
	assert.Contains(t, out, target)
}

func TestStatus_RemotesSection(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	writeStatusState(t, statePath, state.State{})

	repoRoot := writeRepoManifest(t, statusManifestMypkg)
	gitInitRepo(t, repoRoot)

	subRoot := filepath.Join(t.TempDir(), "kick.git")
	require.NoError(t, os.MkdirAll(subRoot, 0o755))
	for _, args := range [][]string{
		{"-C", subRoot, "init"},
		{"-C", subRoot, "config", "user.email", "t@t"},
		{"-C", subRoot, "config", "user.name", "t"},
	} {
		out, err := exec.Command("git", args...).CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}
	require.NoError(t, os.WriteFile(filepath.Join(subRoot, "README"), []byte("x"), 0o644))
	for _, args := range [][]string{
		{"-C", subRoot, "add", "."},
		{"-C", subRoot, "commit", "-m", "init"},
	} {
		out, err := exec.Command("git", args...).CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}

	addOut, addErr := exec.Command("git", "-C", repoRoot,
		"-c", "protocol.file.allow=always",
		"submodule", "add", "--", subRoot, "remotes/kick").CombinedOutput()
	require.NoErrorf(t, addErr, "submodule add: %s", addOut)

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Remotes:")
	assert.Contains(t, out, "kick")
	assert.NotContains(t, out, "Remotes: none")
}

func TestStatus_NoRemotes(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	writeStatusState(t, statePath, state.State{})

	repoRoot := writeRepoManifest(t, statusManifestMypkg)
	gitInitRepo(t, repoRoot)

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Remotes: none")
}

func TestStatus_FilterByPackage(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
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

	repoRoot := writeRepoManifest(t, statusManifestMypkg)
	gitInitRepo(t, repoRoot)

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status", "mypkg",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "mypkg")
	assert.NotContains(t, out, "otherpkg")
}

func TestStatus_LoadStateError(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
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

func TestStatus_FilterRendersDeclaredDependencies(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	tmp := t.TempDir()
	statePath, st := installedPkgState(t, tmp, "mypkg")
	writeStatusState(t, statePath, st)

	repoRoot := writeRepoManifest(t, `schema_version = 1

[packages.mypkg]
description = "Test package using a registry dep"
supported_os = ["linux", "darwin", "windows"]
dependencies = [{name = "git"}]

[packages.mypkg.profiles.common]
sources = [{path = "x", mode = "file", target = "$HOME"}]
`)
	gitInitRepo(t, repoRoot)

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
	assert.Contains(t, out, "Declared dependencies for mypkg:")
	assert.NotContains(t, out, "Warning: dependency check failed")
}

func TestStatus_PrintsBrokenLinkLine(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	source := filepath.Join(tmp, "missing-source")
	target := filepath.Join(tmp, "tgt")

	writeStatusState(t, statePath, state.State{
		"mypkg": state.PackageState{
			Profile:        "common",
			InstalledLinks: []state.InstalledLink{{Source: source, Target: target}},
			InstalledAt:    time.Now(),
		},
	})

	repoRoot := writeRepoManifest(t, statusManifestMypkg)
	gitInitRepo(t, repoRoot)

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "[BROKEN]")
	assert.Contains(t, out, " broken link: "+target)
}

func TestStatus_SummaryLine_AllStates(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	// Create symlinks for OK and BROKEN packages
	okSource := filepath.Join(tmp, "ok-src")
	okTarget := filepath.Join(tmp, "ok-tgt")
	require.NoError(t, os.WriteFile(okSource, []byte("x"), 0o644))
	require.NoError(t, os.Symlink(okSource, okTarget))

	brokenSource := filepath.Join(tmp, "broken-src")
	brokenTarget := filepath.Join(tmp, "broken-tgt")
	otherSource := filepath.Join(tmp, "other-src")
	require.NoError(t, os.WriteFile(brokenSource, []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(otherSource, []byte("y"), 0o644))
	require.NoError(t, os.Symlink(otherSource, brokenTarget))

	writeStatusState(t, statePath, state.State{
		"okpkg": state.PackageState{
			Profile:        "common",
			InstalledLinks: []state.InstalledLink{{Source: okSource, Target: okTarget}},
			InstalledAt:    time.Now(),
		},
		"brokenpkg": state.PackageState{
			Profile:        "common",
			InstalledLinks: []state.InstalledLink{{Source: brokenSource, Target: brokenTarget}},
			InstalledAt:    time.Now(),
		},
		"untrackedpkg": state.PackageState{
			Profile:        "custom",
			InstalledLinks: []state.InstalledLink{{Source: okSource, Target: okTarget}},
			InstalledAt:    time.Now(),
		},
	})

	repoRoot := writeRepoManifest(t, `schema_version = 1

[packages.okpkg]
description = "OK package"
supported_os = ["linux", "darwin", "windows"]

[packages.okpkg.profiles.common]
sources = [{path = "x", mode = "file", target = "$HOME"}]

[packages.brokenpkg]
description = "Broken package"
supported_os = ["linux", "darwin", "windows"]

[packages.brokenpkg.profiles.common]
sources = [{path = "x", mode = "file", target = "$HOME"}]

[packages.notinstalledpkg1]
description = "Not installed 1"
supported_os = ["linux", "darwin", "windows"]

[packages.notinstalledpkg1.profiles.common]
sources = [{path = "x", mode = "file", target = "$HOME"}]

[packages.notinstalledpkg2]
description = "Not installed 2"
supported_os = ["linux", "darwin", "windows"]

[packages.notinstalledpkg2.profiles.common]
sources = [{path = "x", mode = "file", target = "$HOME"}]
`)
	gitInitRepo(t, repoRoot)

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Total: 5 packages — 3 installed, 2 not installed, 1 broken, 1 untracked.")
}

func TestStatus_SummaryLine_NoUntracked(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	okSource := filepath.Join(tmp, "ok-src")
	okTarget := filepath.Join(tmp, "ok-tgt")
	require.NoError(t, os.WriteFile(okSource, []byte("x"), 0o644))
	require.NoError(t, os.Symlink(okSource, okTarget))

	brokenSource := filepath.Join(tmp, "broken-src")
	brokenTarget := filepath.Join(tmp, "broken-tgt")
	otherSource := filepath.Join(tmp, "other-src")
	require.NoError(t, os.WriteFile(brokenSource, []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(otherSource, []byte("y"), 0o644))
	require.NoError(t, os.Symlink(otherSource, brokenTarget))

	writeStatusState(t, statePath, state.State{
		"okpkg": state.PackageState{
			Profile:        "common",
			InstalledLinks: []state.InstalledLink{{Source: okSource, Target: okTarget}},
			InstalledAt:    time.Now(),
		},
		"brokenpkg": state.PackageState{
			Profile:        "common",
			InstalledLinks: []state.InstalledLink{{Source: brokenSource, Target: brokenTarget}},
			InstalledAt:    time.Now(),
		},
	})

	repoRoot := writeRepoManifest(t, `schema_version = 1

[packages.okpkg]
description = "OK package"
supported_os = ["linux", "darwin", "windows"]

[packages.okpkg.profiles.common]
sources = [{path = "x", mode = "file", target = "$HOME"}]

[packages.brokenpkg]
description = "Broken package"
supported_os = ["linux", "darwin", "windows"]

[packages.brokenpkg.profiles.common]
sources = [{path = "x", mode = "file", target = "$HOME"}]

[packages.notinstalledpkg]
description = "Not installed"
supported_os = ["linux", "darwin", "windows"]

[packages.notinstalledpkg.profiles.common]
sources = [{path = "x", mode = "file", target = "$HOME"}]
`)
	gitInitRepo(t, repoRoot)

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Total: 3 packages — 2 installed, 1 not installed, 1 broken.")
	assert.NotContains(t, out, "untracked")
}

func TestStatus_SummaryLine_FilterSuppresses(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	tmp := t.TempDir()
	statePath, st := installedPkgState(t, tmp, "mypkg")
	writeStatusState(t, statePath, st)

	repoRoot := writeRepoManifest(t, statusManifestMypkg)
	gitInitRepo(t, repoRoot)

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"status", "mypkg",
	)
	require.NoError(t, err, "out=%s", out)
	assert.NotContains(t, out, "Total:")
}

func TestStatus_SummaryLine_PlainMode(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	tmp := t.TempDir()
	statePath, st := installedPkgState(t, tmp, "mypkg")
	writeStatusState(t, statePath, st)

	repoRoot := writeRepoManifest(t, statusManifestMypkg)
	gitInitRepo(t, repoRoot)

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"--plain",
		"status",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Total: 1 packages -- 1 installed, 0 not installed, 0 broken.")
	assert.NotContains(t, out, "—")
}
