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
