package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/repo"
	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/guneet-xyz/easyrice/internal/updater"
)

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
