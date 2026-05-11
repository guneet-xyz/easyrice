package updater

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSwapper struct {
	err           error
	calls         int
	gotAssetURL   string
	gotAssetName  string
	gotRealExe    string
}

func (f *fakeSwapper) Swap(ctx context.Context, assetURL, assetFileName, realExe string) error {
	f.calls++
	f.gotAssetURL = assetURL
	f.gotAssetName = assetFileName
	f.gotRealExe = realExe
	return f.err
}

func newTestUpdater(t *testing.T, sw *fakeSwapper) *Updater {
	t.Helper()
	u, err := New(Options{
		Owner:    "test-owner",
		Repo:     "test-repo",
		CacheDir: t.TempDir(),
	})
	require.NoError(t, err)
	u.swapper = sw
	return u
}

func TestApply_SuccessViaSwapSeam(t *testing.T) {
	sw := &fakeSwapper{}
	u := newTestUpdater(t, sw)

	rel := &Release{
		Version:  "v1.2.3",
		URL:      "https://example.com/release",
		AssetURL: "https://example.com/easyrice_v1.2.3_linux_amd64.tar.gz",
	}

	err := u.Apply(context.Background(), rel)
	require.NoError(t, err)

	assert.Equal(t, 1, sw.calls, "swapper.Swap should be called exactly once")
	assert.Equal(t, rel.AssetURL, sw.gotAssetURL)
	assert.Equal(t, "easyrice_v1.2.3_linux_amd64.tar.gz", sw.gotAssetName)
	assert.NotEmpty(t, sw.gotRealExe, "realExe should be resolved and passed through")

	// Lock must be released (cleaned up) after success.
	_, statErr := os.Stat(filepath.Join(u.opts.CacheDir, lockFileName))
	assert.True(t, os.IsNotExist(statErr), "lockfile should be removed after Apply success")
}

func TestApply_SwapperErrorPropagated(t *testing.T) {
	sw := &fakeSwapper{err: errors.New("swap failed")}
	u := newTestUpdater(t, sw)

	rel := &Release{
		Version:  "v1.2.3",
		AssetURL: "https://example.com/asset.tar.gz",
	}

	err := u.Apply(context.Background(), rel)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "swap failed", "underlying swapper error must be wrapped")
	assert.Equal(t, 1, sw.calls)

	// Lock must still be released after failure.
	_, statErr := os.Stat(filepath.Join(u.opts.CacheDir, lockFileName))
	assert.True(t, os.IsNotExist(statErr), "lockfile should be removed even on swap failure")
}

func TestApply_LockBusyWhenLockHeld(t *testing.T) {
	sw := &fakeSwapper{}
	u := newTestUpdater(t, sw)

	// Pre-create a fresh lockfile so acquireLock returns ErrLockBusy.
	lockPath := filepath.Join(u.opts.CacheDir, lockFileName)
	require.NoError(t, os.WriteFile(lockPath, []byte("99999\n"), 0o644))

	rel := &Release{Version: "v1.2.3", AssetURL: "https://example.com/asset.tar.gz"}
	err := u.Apply(context.Background(), rel)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrLockBusy), "expected ErrLockBusy, got %v", err)
	assert.Equal(t, 0, sw.calls, "swapper must NOT be invoked when lock is busy")
}

func TestApply_StaleLockReclaimed(t *testing.T) {
	sw := &fakeSwapper{}
	u := newTestUpdater(t, sw)

	// Pre-create a stale lockfile (mtime older than staleLockAge).
	lockPath := filepath.Join(u.opts.CacheDir, lockFileName)
	require.NoError(t, os.WriteFile(lockPath, []byte("12345\n"), 0o644))
	old := time.Now().Add(-2 * staleLockAge)
	require.NoError(t, os.Chtimes(lockPath, old, old))

	rel := &Release{Version: "v1.2.3", AssetURL: "https://example.com/asset.tar.gz"}
	err := u.Apply(context.Background(), rel)

	require.NoError(t, err, "stale lock should be reclaimed and Apply should succeed")
	assert.Equal(t, 1, sw.calls, "swapper should be invoked after stale lock reclaim")
}

func TestApply_NilReleaseRejected(t *testing.T) {
	sw := &fakeSwapper{}
	u := newTestUpdater(t, sw)

	err := u.Apply(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "release is nil")
	assert.Equal(t, 0, sw.calls, "swapper must NOT be invoked for nil release")
}

// TestApply_DevBuildNotGatedHere documents the contract from swap.go's
// docstring: "The caller is responsible for the IsDevBuild() gate; Apply does
// not re-check." Even when the local binary is a dev build, Apply still
// delegates to the swapper. The dev-build gate lives in CheckCached, not Apply.
func TestApply_DevBuildNotGatedHere(t *testing.T) {
	require.True(t, IsDevBuild("dev"), "sanity: 'dev' must be a dev build per version.go")

	sw := &fakeSwapper{}
	u := newTestUpdater(t, sw)

	rel := &Release{Version: "v1.2.3", AssetURL: "https://example.com/asset.tar.gz"}
	err := u.Apply(context.Background(), rel)
	require.NoError(t, err)
	assert.Equal(t, 1, sw.calls, "Apply does not re-check IsDevBuild; swap must still run")
}

// TestApply_AlreadyLatestNotGatedHere documents the contract: Apply does not
// compare release.Version to the running binary. The "already latest" gate
// lives in the CheckCached path, not Apply. Passing a Release whose Version
// equals the running version still triggers the swap.
func TestApply_AlreadyLatestNotGatedHere(t *testing.T) {
	sw := &fakeSwapper{}
	u := newTestUpdater(t, sw)

	rel := &Release{Version: "v0.0.0-current", AssetURL: "https://example.com/asset.tar.gz"}
	err := u.Apply(context.Background(), rel)
	require.NoError(t, err)
	assert.Equal(t, 1, sw.calls, "Apply does not compare versions; swap must still run")
}
