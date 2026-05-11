package main

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/guneet-xyz/easyrice/internal/updater"
)

func TestMaybePrintUpdateReminder_CacheError(t *testing.T) {
	saveReminderState(t)
	resetInstallFlags()
	Version = "v1.0.0"
	reminderTTYFn = func() bool { return true }
	reminderCheckFn = func(ctx context.Context, current string) (*updater.CheckResult, error) {
		return nil, errors.New("cache boom")
	}

	stderr := captureStderr(t, func() {
		_, err := runInstallCmd(t, "", "version")
		assert.NoError(t, err)
	})
	assert.NotContains(t, stderr, "A new release", "cache error must be silent")
}

func TestMaybePrintUpdateReminder_NilResult(t *testing.T) {
	saveReminderState(t)
	resetInstallFlags()
	Version = "v1.0.0"
	reminderTTYFn = func() bool { return true }
	reminderCheckFn = func(ctx context.Context, current string) (*updater.CheckResult, error) {
		return nil, nil
	}

	stderr := captureStderr(t, func() {
		_, err := runInstallCmd(t, "", "version")
		assert.NoError(t, err)
	})
	assert.NotContains(t, stderr, "A new release")
}

func TestMaybePrintUpdateReminder_NoUpdateAvailable(t *testing.T) {
	saveReminderState(t)
	resetInstallFlags()
	Version = "v1.0.0"
	reminderTTYFn = func() bool { return true }
	reminderCheckFn = func(ctx context.Context, current string) (*updater.CheckResult, error) {
		return &updater.CheckResult{UpdateAvailable: false, Latest: "v1.0.0"}, nil
	}

	stderr := captureStderr(t, func() {
		_, err := runInstallCmd(t, "", "version")
		assert.NoError(t, err)
	})
	assert.NotContains(t, stderr, "A new release")
}

func TestMaybePrintUpdateReminder_LatestNotNewer(t *testing.T) {
	saveReminderState(t)
	resetInstallFlags()
	Version = "v2.0.0"
	reminderTTYFn = func() bool { return true }
	reminderCheckFn = func(ctx context.Context, current string) (*updater.CheckResult, error) {
		return &updater.CheckResult{UpdateAvailable: true, Latest: "v1.0.0"}, nil
	}

	stderr := captureStderr(t, func() {
		_, err := runInstallCmd(t, "", "version")
		assert.NoError(t, err)
	})
	assert.NotContains(t, stderr, "A new release")
}

func TestMaybePrintUpdateReminder_InvalidSemverInLatest(t *testing.T) {
	saveReminderState(t)
	resetInstallFlags()
	Version = "v1.0.0"
	reminderTTYFn = func() bool { return true }
	reminderCheckFn = func(ctx context.Context, current string) (*updater.CheckResult, error) {
		return &updater.CheckResult{UpdateAvailable: true, Latest: "not-a-version"}, nil
	}

	stderr := captureStderr(t, func() {
		_, err := runInstallCmd(t, "", "version")
		assert.NoError(t, err)
	})
	assert.NotContains(t, stderr, "A new release")
}
