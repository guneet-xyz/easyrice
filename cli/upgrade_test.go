package main

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/updater"
)

func saveUpgradeState(t *testing.T) {
	t.Helper()
	origVersion := Version
	origFetch := upgradeFetchFn
	origApply := upgradeApplyFn
	origCheck := flagUpgradeCheck
	origYes := flagYes
	t.Cleanup(func() {
		Version = origVersion
		upgradeFetchFn = origFetch
		upgradeApplyFn = origApply
		flagUpgradeCheck = origCheck
		flagYes = origYes
	})
}

func TestUpgradeDevBuildRefuses(t *testing.T) {
	saveUpgradeState(t)
	resetInstallFlags()
	Version = "dev"

	out, err := runInstallCmd(t, "", "upgrade")
	require.Error(t, err, "out=%s", out)
	assert.True(t, errors.Is(err, updater.ErrDevBuild), "err=%v", err)
}

func TestUpgradeCheckPrintsAvailable(t *testing.T) {
	saveUpgradeState(t)
	resetInstallFlags()
	Version = "v1.0.0"

	upgradeFetchFn = func(ctx context.Context) (*updater.Updater, *updater.Release, error) {
		return nil, &updater.Release{Version: "v2.0.0"}, nil
	}

	out, err := runInstallCmd(t, "", "upgrade", "--check")
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "new version available")
	assert.Contains(t, out, "v1.0.0")
	assert.Contains(t, out, "v2.0.0")
}

func TestUpgradeCheckPrintsUpToDate(t *testing.T) {
	saveUpgradeState(t)
	resetInstallFlags()
	Version = "v1.0.0"

	upgradeFetchFn = func(ctx context.Context) (*updater.Updater, *updater.Release, error) {
		return nil, nil, updater.ErrAlreadyLatest
	}

	out, err := runInstallCmd(t, "", "upgrade", "--check")
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "up to date")
	assert.Contains(t, out, "v1.0.0")
}

func TestUpgradeApplyHappyPath(t *testing.T) {
	saveUpgradeState(t)
	resetInstallFlags()
	Version = "v1.0.0"

	applied := false
	upgradeFetchFn = func(ctx context.Context) (*updater.Updater, *updater.Release, error) {
		return nil, &updater.Release{Version: "v1.5.0"}, nil
	}
	upgradeApplyFn = func(ctx context.Context, u *updater.Updater, rel *updater.Release) error {
		applied = true
		assert.Equal(t, "v1.5.0", rel.Version)
		return nil
	}

	out, err := runInstallCmd(t, "", "--yes", "upgrade")
	require.NoError(t, err, "out=%s", out)
	assert.True(t, applied, "Apply should be called")
	assert.Contains(t, out, "Upgraded easyrice to v1.5.0")
}

func TestUpgradeConfirmDeclined(t *testing.T) {
	saveUpgradeState(t)
	resetInstallFlags()
	Version = "v1.0.0"

	applied := false
	upgradeFetchFn = func(ctx context.Context) (*updater.Updater, *updater.Release, error) {
		return nil, &updater.Release{Version: "v1.5.0"}, nil
	}
	upgradeApplyFn = func(ctx context.Context, u *updater.Updater, rel *updater.Release) error {
		applied = true
		return nil
	}

	out, err := runInstallCmd(t, "n\n", "upgrade")
	require.NoError(t, err, "out=%s", out)
	assert.False(t, applied, "Apply must NOT be called when user declines")
	assert.Contains(t, out, "cancelled")
}
