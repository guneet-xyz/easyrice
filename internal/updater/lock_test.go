package updater

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcquireLock_Success(t *testing.T) {
	dir := t.TempDir()
	release, err := acquireLock(dir)
	require.NoError(t, err)
	require.NotNil(t, release)

	_, statErr := os.Stat(filepath.Join(dir, lockFileName))
	assert.NoError(t, statErr, "lockfile should exist after acquire")

	release()
	_, statErr = os.Stat(filepath.Join(dir, lockFileName))
	assert.True(t, os.IsNotExist(statErr), "lockfile should be removed after release")
}

func TestAcquireLock_Busy(t *testing.T) {
	dir := t.TempDir()
	release1, err := acquireLock(dir)
	require.NoError(t, err)
	defer release1()

	release2, err := acquireLock(dir)
	assert.Nil(t, release2)
	assert.True(t, errors.Is(err, ErrLockBusy), "expected ErrLockBusy, got %v", err)
}

func TestAcquireLock_StaleReclaimed(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, lockFileName)
	require.NoError(t, os.WriteFile(lockPath, []byte("12345\n"), 0o644))

	old := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(lockPath, old, old))

	release, err := acquireLock(dir)
	require.NoError(t, err, "stale lock must be reclaimed")
	require.NotNil(t, release)
	defer release()

	info, err := os.Stat(lockPath)
	require.NoError(t, err)
	assert.True(t, time.Since(info.ModTime()) < time.Minute, "reclaimed lockfile should have fresh mtime")
}

func TestAcquireLock_FreshNotReclaimed(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, lockFileName)
	require.NoError(t, os.WriteFile(lockPath, []byte("99999\n"), 0o644))

	release, err := acquireLock(dir)
	assert.Nil(t, release)
	assert.True(t, errors.Is(err, ErrLockBusy))
}

func TestAcquireLock_ReleaseIdempotent(t *testing.T) {
	dir := t.TempDir()
	release, err := acquireLock(dir)
	require.NoError(t, err)

	release()
	assert.NotPanics(t, func() { release() }, "release should be idempotent")
}

func TestAcquireLock_SuccessWhenNoLock(t *testing.T) {
	dir := t.TempDir()
	release, err := acquireLock(dir)
	require.NoError(t, err)
	require.NotNil(t, release)
	defer release()

	_, statErr := os.Stat(filepath.Join(dir, lockFileName))
	require.NoError(t, statErr, "lockfile must exist after acquire")
}

func TestAcquireLock_ErrLockBusyWhenLockHeld(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, lockFileName)
	require.NoError(t, os.WriteFile(lockPath, []byte("1\n"), 0o644))
	now := time.Now()
	require.NoError(t, os.Chtimes(lockPath, now, now))

	release, err := acquireLock(dir)
	assert.Nil(t, release)
	assert.True(t, errors.Is(err, ErrLockBusy), "expected ErrLockBusy, got %v", err)
}

func TestAcquireLock_ReclaimsStaleLock(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, lockFileName)
	require.NoError(t, os.WriteFile(lockPath, []byte("9999\n"), 0o644))

	old := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(lockPath, old, old))

	release, err := acquireLock(dir)
	require.NoError(t, err)
	require.NotNil(t, release)
	defer release()

	info, err := os.Stat(lockPath)
	require.NoError(t, err)
	assert.True(t, time.Since(info.ModTime()) < time.Minute, "reclaimed lock should have fresh mtime")
}

func TestAcquireLock_ReleasesOnClose(t *testing.T) {
	dir := t.TempDir()
	release, err := acquireLock(dir)
	require.NoError(t, err)
	require.NotNil(t, release)

	lockPath := filepath.Join(dir, lockFileName)
	_, statErr := os.Stat(lockPath)
	require.NoError(t, statErr)

	release()

	_, statErr = os.Stat(lockPath)
	assert.True(t, os.IsNotExist(statErr), "lockfile should be removed after release")
}

func TestTryCreateLock_SuccessOnFirstCall(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, lockFileName)

	f, err := tryCreateLock(lockPath)
	require.NoError(t, err)
	require.NotNil(t, f)
	defer func() {
		_ = f.Close()
		_ = os.Remove(lockPath)
	}()

	contents, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	assert.NotEmpty(t, contents, "lockfile should contain PID")
}

func TestTryCreateLock_FailsWhenExists(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, lockFileName)
	require.NoError(t, os.WriteFile(lockPath, []byte("preexisting\n"), 0o644))

	f, err := tryCreateLock(lockPath)
	assert.Nil(t, f)
	require.Error(t, err)
	assert.True(t, os.IsExist(err), "expected os.IsExist error, got %v", err)
}

func TestAcquireLock_MkdirAllFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root; permission bits are bypassed")
	}
	parent := t.TempDir()
	require.NoError(t, os.Chmod(parent, 0o500))
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })

	release, err := acquireLock(filepath.Join(parent, "child", "cache"))
	assert.Nil(t, release)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create cache dir")
}

func TestAcquireLock_NonExistErrorFromTryCreate(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root; permission bits are bypassed")
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o500))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	release, err := acquireLock(dir)
	assert.Nil(t, release)
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrLockBusy))
	assert.Contains(t, err.Error(), "open lockfile")
}

func TestTryCreateLock_NonExistError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root; permission bits are bypassed")
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o500))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	f, err := tryCreateLock(filepath.Join(dir, lockFileName))
	assert.Nil(t, f)
	require.Error(t, err)
	assert.False(t, os.IsExist(err))
}

func TestAcquireLock_RemoveStaleFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root; permission bits are bypassed")
	}
	dir := t.TempDir()
	lockPath := filepath.Join(dir, lockFileName)
	require.NoError(t, os.WriteFile(lockPath, []byte("9999\n"), 0o644))
	old := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(lockPath, old, old))

	require.NoError(t, os.Chmod(dir, 0o500))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	release, err := acquireLock(dir)
	assert.Nil(t, release)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remove stale lockfile")
}

func TestTryCreateLock_WritePidFails(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "subdir-as-lock")
	require.NoError(t, os.Mkdir(lockPath, 0o755))

	f, err := tryCreateLock(lockPath)
	if err == nil {
		_ = f.Close()
		t.Skip("platform allows O_CREATE|O_EXCL|O_WRONLY on a directory; cannot trigger write failure this way")
	}
	assert.Nil(t, f)
	assert.True(t, os.IsExist(err))
}

func TestAcquireLock_FreshBusy(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, lockFileName)

	require.NoError(t, os.WriteFile(lockPath, []byte("99999\n"), 0o644))
	now := time.Now()
	require.NoError(t, os.Chtimes(lockPath, now, now))

	release, err := acquireLock(dir)
	assert.Nil(t, release)
	assert.True(t, errors.Is(err, ErrLockBusy))
}

func TestAcquireLock_StaleReclaim(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, lockFileName)

	require.NoError(t, os.WriteFile(lockPath, []byte("88888\n"), 0o644))
	old := time.Now().Add(-2 * staleLockAge)
	require.NoError(t, os.Chtimes(lockPath, old, old))

	info, err := os.Stat(lockPath)
	require.NoError(t, err)
	require.True(t, time.Since(info.ModTime()) > staleLockAge)

	release, err := acquireLock(dir)
	require.NoError(t, err)
	require.NotNil(t, release)
	defer release()

	info, err = os.Stat(lockPath)
	require.NoError(t, err)
	assert.True(t, time.Since(info.ModTime()) < time.Minute)
}

func TestTryCreateLock_WritePidError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root; cannot test write permission failures")
	}

	dir := t.TempDir()
	readOnlyDir := filepath.Join(dir, "readonly")
	require.NoError(t, os.Mkdir(readOnlyDir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(readOnlyDir, 0o755) })

	f, err := tryCreateLock(filepath.Join(readOnlyDir, lockFileName))
	assert.Nil(t, f)
	require.Error(t, err)
	assert.False(t, os.IsExist(err))
}
