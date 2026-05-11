package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/updater"
)

func saveReminderState(t *testing.T) {
	t.Helper()
	origVersion := Version
	origTTY := reminderTTYFn
	origCheck := reminderCheckFn
	t.Cleanup(func() {
		Version = origVersion
		reminderTTYFn = origTTY
		reminderCheckFn = origCheck
	})
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	orig := os.Stderr
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = orig })

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	out := <-done
	os.Stderr = orig
	return out
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	orig := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig })

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	out := <-done
	os.Stdout = orig
	return out
}

func TestVersionStdoutUnchanged(t *testing.T) {
	saveReminderState(t)
	resetInstallFlags()
	Version = "v1.0.0"
	reminderTTYFn = func() bool { return false }

	stdout := captureStdout(t, func() {
		_, err := runInstallCmd(t, "", "version")
		require.NoError(t, err)
	})

	expected := "easyrice version v1.0.0\n"
	assert.Equal(t, expected, stdout, "version stdout must be exactly %q", expected)
}

func TestVersionReminderOnTTYNewer(t *testing.T) {
	saveReminderState(t)
	resetInstallFlags()
	Version = "v1.0.0"
	reminderTTYFn = func() bool { return true }
	reminderCheckFn = func(ctx context.Context, current string) (*updater.CheckResult, error) {
		return &updater.CheckResult{
			Current:         current,
			Latest:          "v2.0.0",
			UpdateAvailable: true,
		}, nil
	}

	stderr := captureStderr(t, func() {
		out, err := runInstallCmd(t, "", "version")
		require.NoError(t, err, "out=%s", out)
	})

	assert.Contains(t, stderr, "A new release of easyrice is available")
	assert.Contains(t, stderr, "v1.0.0")
	assert.Contains(t, stderr, "v2.0.0")
}

func TestVersionNoReminderDev(t *testing.T) {
	saveReminderState(t)
	resetInstallFlags()
	Version = "dev"
	reminderTTYFn = func() bool { return true }
	checkCalled := false
	reminderCheckFn = func(ctx context.Context, current string) (*updater.CheckResult, error) {
		checkCalled = true
		return nil, nil
	}

	stderr := captureStderr(t, func() {
		_, err := runInstallCmd(t, "", "version")
		require.NoError(t, err)
	})

	assert.False(t, checkCalled, "checkCachedFn must NOT be called for dev builds")
	assert.NotContains(t, stderr, "A new release")
}

func TestVersionNoReminderOptOut(t *testing.T) {
	saveReminderState(t)
	resetInstallFlags()
	Version = "v1.0.0"
	t.Setenv("EASYRICE_NO_UPDATE_CHECK", "1")
	reminderTTYFn = func() bool { return true }
	checkCalled := false
	reminderCheckFn = func(ctx context.Context, current string) (*updater.CheckResult, error) {
		checkCalled = true
		return nil, nil
	}

	stderr := captureStderr(t, func() {
		_, err := runInstallCmd(t, "", "version")
		require.NoError(t, err)
	})

	assert.False(t, checkCalled, "checkCachedFn must NOT be called when opted out")
	assert.NotContains(t, stderr, "A new release")
}

func TestVersionNoReminderFlag(t *testing.T) {
	saveReminderState(t)
	resetInstallFlags()
	Version = "v1.0.0"
	reminderTTYFn = func() bool { return true }
	checkCalled := false
	reminderCheckFn = func(ctx context.Context, current string) (*updater.CheckResult, error) {
		checkCalled = true
		return nil, nil
	}

	stderr := captureStderr(t, func() {
		_, err := runInstallCmd(t, "", "--no-update-check", "version")
		require.NoError(t, err)
	})

	assert.False(t, checkCalled, "checkCachedFn must NOT be called when --no-update-check")
	assert.NotContains(t, stderr, "A new release")
}

func TestVersionNoReminderNonTTY(t *testing.T) {
	saveReminderState(t)
	resetInstallFlags()
	Version = "v1.0.0"
	reminderTTYFn = func() bool { return false }
	checkCalled := false
	reminderCheckFn = func(ctx context.Context, current string) (*updater.CheckResult, error) {
		checkCalled = true
		return nil, nil
	}

	stderr := captureStderr(t, func() {
		_, err := runInstallCmd(t, "", "version")
		require.NoError(t, err)
	})

	assert.False(t, checkCalled, "checkCachedFn must NOT be called when non-TTY")
	assert.NotContains(t, stderr, "A new release")
}
