package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/repo"
	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/guneet-xyz/easyrice/internal/updater"
)

func initGitRepoAt(t *testing.T, dir string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	require.NoError(t, os.MkdirAll(dir, 0o755))
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, string(out))
	}
	run("init", "-b", "main")
	run("config", "user.email", "t@t.com")
	run("config", "user.name", "T")
	run("config", "commit.gpgsign", "false")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "seed"), []byte("x"), 0o644))
	run("add", ".")
	run("commit", "-m", "init")
}

func setupDoctorRepo(t *testing.T) {
	t.Helper()
	setIsolatedHome(t)
	require.NoError(t, os.MkdirAll(repo.DefaultRepoPath(), 0o755))
}

func TestDoctor_NoStateFile(t *testing.T) {
	resetInstallFlags()
	setupDoctorRepo(t)
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "missing-state.json")

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"doctor",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "All checks passed.")
}

func TestDoctor_HealthyPackage(t *testing.T) {
	resetInstallFlags()
	setupDoctorRepo(t)
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	source := filepath.Join(tmp, "src.toml")
	target := filepath.Join(tmp, "tgt.toml")
	require.NoError(t, os.WriteFile(source, []byte("x"), 0o644))
	require.NoError(t, os.Symlink(source, target))

	require.NoError(t, state.Save(statePath, state.State{
		"mypkg": state.PackageState{
			Profile:        "common",
			InstalledLinks: []state.InstalledLink{{Source: source, Target: target}},
			InstalledAt:    time.Now(),
		},
	}))

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"doctor",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "All checks passed.")
	assert.NotContains(t, out, "[ERROR]")
}

func TestDoctor_MissingSymlink(t *testing.T) {
	resetInstallFlags()
	setupDoctorRepo(t)
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	source := filepath.Join(tmp, "src.toml")
	target := filepath.Join(tmp, "tgt.toml")
	require.NoError(t, os.WriteFile(source, []byte("x"), 0o644))

	require.NoError(t, state.Save(statePath, state.State{
		"mypkg": state.PackageState{
			Profile:        "common",
			InstalledLinks: []state.InstalledLink{{Source: source, Target: target}},
			InstalledAt:    time.Now(),
		},
	}))

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"doctor",
	)
	require.Error(t, err, "out=%s", out)
	assert.Contains(t, out, "[ERROR]")
	assert.Contains(t, out, "missing symlink")
	assert.Contains(t, out, "mypkg")
	assert.Contains(t, out, "1 issue(s) found")
}

func TestDoctor_ReplacedSymlink(t *testing.T) {
	resetInstallFlags()
	setupDoctorRepo(t)
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	source := filepath.Join(tmp, "src.toml")
	target := filepath.Join(tmp, "tgt.toml")
	require.NoError(t, os.WriteFile(source, []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(target, []byte("regular"), 0o644))

	require.NoError(t, state.Save(statePath, state.State{
		"mypkg": state.PackageState{
			Profile:        "common",
			InstalledLinks: []state.InstalledLink{{Source: source, Target: target}},
			InstalledAt:    time.Now(),
		},
	}))

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"doctor",
	)
	require.Error(t, err, "out=%s", out)
	assert.Contains(t, out, "[ERROR]")
	assert.Contains(t, out, "symlink replaced")
	assert.Contains(t, out, "mypkg")
}

func TestDoctor_RepoMissing(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"doctor",
	)
	require.Error(t, err, "out=%s", out)
	assert.Contains(t, out, "rice init")
	assert.Contains(t, out, "[ERROR]")
}

func TestDoctor_AllPass(t *testing.T) {
	resetInstallFlags()
	setupDoctorRepo(t)
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"doctor",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "[OK] git available")
	assert.Contains(t, out, "[OK] repo initialized")
	assert.Contains(t, out, "All checks passed.")
}

func TestDoctorReminderAppendedOnHealthyTTY(t *testing.T) {
	saveReminderState(t)
	resetInstallFlags()
	setupDoctorRepo(t)
	Version = "v1.0.0"
	reminderTTYFn = func() bool { return true }
	reminderCheckFn = func(ctx context.Context, current string) (*updater.CheckResult, error) {
		return &updater.CheckResult{
			Current:         current,
			Latest:          "v2.0.0",
			UpdateAvailable: true,
		}, nil
	}

	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	stderr := captureStderr(t, func() {
		out, err := runInstallCmd(t, "",
			"--state", statePath,
			"doctor",
		)
		require.NoError(t, err, "out=%s", out)
		assert.Contains(t, out, "All checks passed.")
	})

	assert.Contains(t, stderr, "A new release of easyrice is available")
	assert.Contains(t, stderr, "v2.0.0")
}

func TestDoctorNoReminderOnFailure(t *testing.T) {
	saveReminderState(t)
	resetInstallFlags()
	setIsolatedHome(t)
	Version = "v1.0.0"
	reminderTTYFn = func() bool { return true }
	checkCalled := false
	reminderCheckFn = func(ctx context.Context, current string) (*updater.CheckResult, error) {
		checkCalled = true
		return &updater.CheckResult{Latest: "v2.0.0", UpdateAvailable: true}, nil
	}

	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	stderr := captureStderr(t, func() {
		out, err := runInstallCmd(t, "",
			"--state", statePath,
			"doctor",
		)
		require.Error(t, err, "out=%s", out)
		assert.Contains(t, out, "issue(s) found")
	})

	assert.False(t, checkCalled, "reminder check must NOT run when doctor reports issues")
	assert.NotContains(t, stderr, "A new release")
}

func TestDoctor_LegacyStateDetected(t *testing.T) {
	resetInstallFlags()
	homeDir := setIsolatedHome(t)
	require.NoError(t, os.MkdirAll(repo.DefaultRepoPath(), 0o755))

	legacyDir := filepath.Join(homeDir, ".config", "rice")
	require.NoError(t, os.MkdirAll(legacyDir, 0o755))
	legacyStatePath := filepath.Join(legacyDir, "state.json")
	require.NoError(t, os.WriteFile(legacyStatePath, []byte("{}"), 0o644))

	out, err := runInstallCmd(t, "",
		"doctor",
	)

	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Warning: Legacy state found")
	assert.Contains(t, out, legacyStatePath)
}

func TestDoctor_GitMissing(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	require.NoError(t, os.MkdirAll(repo.DefaultRepoPath(), 0o755))
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	t.Setenv("PATH", "")

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"doctor",
	)
	require.Error(t, err, "out=%s", out)
	assert.Contains(t, out, "[ERROR]")
	assert.Contains(t, strings.ToLower(out), "git")
}

func TestDoctor_InvalidManifest(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	repoPath := repo.DefaultRepoPath()
	require.NoError(t, os.MkdirAll(repoPath, 0o755))

	// Create a rice.toml with invalid TOML syntax
	tomlPath := filepath.Join(repoPath, "rice.toml")
	require.NoError(t, os.WriteFile(tomlPath, []byte("invalid toml [[["), 0o644))

	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"doctor",
	)
	// Doctor reports WARN for manifest load errors but doesn't fail
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "[WARN]")
	assert.Contains(t, out, "Cannot load manifest")
}

func TestDoctor_DeclaredDepsWarnings(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	repoPath := repo.DefaultRepoPath()
	require.NoError(t, os.MkdirAll(repoPath, 0o755))

	// Create a valid rice.toml with a package that has dependencies
	tomlContent := `schema_version = 1

[packages.mypkg]
description = "Test package with deps"
supported_os = ["linux", "darwin"]
dependencies = [
  {name = "missing_dep", version = "1.0.0"}
]

[packages.mypkg.profiles.default]
sources = [{path = "config", mode = "file", target = "$HOME/.config/mypkg"}]
`
	tomlPath := filepath.Join(repoPath, "rice.toml")
	require.NoError(t, os.WriteFile(tomlPath, []byte(tomlContent), 0o644))

	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"doctor",
	)
	// Doctor should report warnings but still exit with error due to missing deps
	require.Error(t, err, "out=%s", out)
	assert.Contains(t, out, "[WARN]")
	assert.Contains(t, out, "mypkg")
}

func TestDoctor_DirtyWarn(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	repoPath := repo.DefaultRepoPath()
	initGitRepoAt(t, repoPath)

	// Make repo dirty without committing.
	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "untracked.txt"), []byte("dirty"), 0o644))

	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"doctor",
	)
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "[WARN]")
	assert.Contains(t, out, "uncommitted changes")
	assert.NotContains(t, out, "issue(s) found")
}

func TestDoctor_UninitSubmodule(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	repoPath := repo.DefaultRepoPath()
	initGitRepoAt(t, repoPath)

	// Create a bare upstream and add it as a submodule, then deinit.
	bareURL := makeBareRepo(t, "schema_version = 1\n")
	subRun := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoPath
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, string(out))
	}
	subRun("-c", "protocol.file.allow=always", "submodule", "add", "--", bareURL, "remotes/sub")
	subRun("commit", "-m", "add submodule")
	subRun("submodule", "deinit", "-f", "--", "remotes/sub")

	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"doctor",
	)
	require.Error(t, err, "out=%s", out)
	assert.Contains(t, out, "[ERROR]")
	assert.Contains(t, out, "submodule sub not initialized")
	assert.Contains(t, out, "issue(s) found")
}

func TestDoctor_DanglingImport(t *testing.T) {
	resetInstallFlags()
	setIsolatedHome(t)
	repoPath := repo.DefaultRepoPath()
	require.NoError(t, os.MkdirAll(repoPath, 0o755))

	tomlContent := `schema_version = 1

[packages.nvim]
description = "Neovim with dangling import"
supported_os = ["linux", "darwin"]

[packages.nvim.profiles.default]
import = "remotes/missing#nvim.default"
`
	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "rice.toml"), []byte(tomlContent), 0o644))

	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")

	out, err := runInstallCmd(t, "",
		"--state", statePath,
		"doctor",
	)
	require.Error(t, err, "out=%s", out)
	assert.Contains(t, out, "[ERROR]")
	assert.Contains(t, out, "package nvim profile default")
	assert.Contains(t, out, "remotes/missing")
	assert.Contains(t, out, "issue(s) found")
}
