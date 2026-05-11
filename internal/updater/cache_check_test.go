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

type fakeFetcher struct {
	release *Release
	err     error
	calls   int
}

func (f *fakeFetcher) FetchLatest(ctx context.Context) (*Release, error) {
	f.calls++
	return f.release, f.err
}

func newFakeUpdater(t *testing.T, fetch *fakeFetcher) *Updater {
	t.Helper()
	u, err := New(Options{
		Owner:    "guneet-xyz",
		Repo:     "easyrice",
		CacheDir: t.TempDir(),
	})
	require.NoError(t, err)
	u.fetcher = fetch
	return u
}

func TestCheckCachedFirstRun(t *testing.T) {
	fake := &fakeFetcher{}
	u := newFakeUpdater(t, fake)

	res, err := u.CheckCached(context.Background(), "v1.2.3")
	require.NoError(t, err)
	assert.False(t, res.UpdateAvailable, "first run must not signal update")
	assert.Equal(t, "v1.2.3", res.Latest)
	assert.Equal(t, 0, fake.calls, "first run must NOT call fetcher")

	c, err := loadCache(u.opts.CacheDir)
	require.NoError(t, err)
	require.NotNil(t, c, "sentinel cache must be seeded")
	assert.Equal(t, "v1.2.3", c.LatestVersion)
}

func TestCheckCachedFresh(t *testing.T) {
	fake := &fakeFetcher{}
	u := newFakeUpdater(t, fake)

	require.NoError(t, saveCache(u.opts.CacheDir, &cacheFile{
		LatestVersion:  "v2.0.0",
		LastChecked:    time.Now().Add(-1 * time.Hour),
		CurrentVersion: "v1.0.0",
	}))

	res, err := u.CheckCached(context.Background(), "v1.0.0")
	require.NoError(t, err)
	assert.True(t, res.UpdateAvailable)
	assert.Equal(t, "v2.0.0", res.Latest)
	assert.Equal(t, 0, fake.calls, "fresh cache must not call fetcher")
}

func TestCheckCachedStaleSuccess(t *testing.T) {
	fake := &fakeFetcher{
		release: &Release{Version: "v3.0.0"},
	}
	u := newFakeUpdater(t, fake)

	require.NoError(t, saveCache(u.opts.CacheDir, &cacheFile{
		LatestVersion:  "v1.0.0",
		LastChecked:    time.Now().Add(-48 * time.Hour),
		CurrentVersion: "v1.0.0",
	}))

	res, err := u.CheckCached(context.Background(), "v1.0.0")
	require.NoError(t, err)
	assert.True(t, res.UpdateAvailable)
	assert.Equal(t, "v3.0.0", res.Latest)
	assert.Equal(t, 1, fake.calls, "stale cache must call fetcher exactly once")

	c, err := loadCache(u.opts.CacheDir)
	require.NoError(t, err)
	assert.Equal(t, "v3.0.0", c.LatestVersion, "cache must be refreshed with new latest")
}

func TestCheckCachedStaleFailureSilent(t *testing.T) {
	fake := &fakeFetcher{err: errors.New("network down")}
	u := newFakeUpdater(t, fake)

	require.NoError(t, saveCache(u.opts.CacheDir, &cacheFile{
		LatestVersion:  "v1.0.0",
		LastChecked:    time.Now().Add(-48 * time.Hour),
		CurrentVersion: "v1.0.0",
	}))

	res, err := u.CheckCached(context.Background(), "v1.0.0")
	require.NoError(t, err, "fail-silent: must not bubble fetch error")
	assert.False(t, res.UpdateAvailable, "fail-silent: UpdateAvailable=false on fetch error")
	assert.Equal(t, 1, fake.calls)
}

func TestCheckCachedDevShortCircuit(t *testing.T) {
	fake := &fakeFetcher{}
	u := newFakeUpdater(t, fake)

	res, err := u.CheckCached(context.Background(), "dev")
	require.NoError(t, err)
	assert.False(t, res.UpdateAvailable)
	assert.Equal(t, 0, fake.calls, "dev build must not call fetcher")

	_, err = os.Stat(filepath.Join(u.opts.CacheDir, cacheFileName))
	assert.True(t, os.IsNotExist(err), "dev build must not touch cache")
}

func TestCheckCachedClockSkew(t *testing.T) {
	fake := &fakeFetcher{
		release: &Release{Version: "v2.0.0"},
	}
	u := newFakeUpdater(t, fake)

	require.NoError(t, saveCache(u.opts.CacheDir, &cacheFile{
		LatestVersion:  "v1.0.0",
		LastChecked:    time.Now().Add(2 * time.Hour),
		CurrentVersion: "v1.0.0",
	}))

	res, err := u.CheckCached(context.Background(), "v1.0.0")
	require.NoError(t, err)
	assert.True(t, res.UpdateAvailable, "future-dated cache must be treated as stale")
	assert.Equal(t, "v2.0.0", res.Latest)
	assert.Equal(t, 1, fake.calls, "clock skew must trigger refresh")
}

func TestCheckCachedCorruptCacheTreatedAsFirstRun(t *testing.T) {
	fake := &fakeFetcher{}
	u := newFakeUpdater(t, fake)

	require.NoError(t, os.MkdirAll(u.opts.CacheDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(u.opts.CacheDir, cacheFileName), []byte("not json"), 0o600))

	res, err := u.CheckCached(context.Background(), "v1.2.3")
	require.NoError(t, err)
	assert.False(t, res.UpdateAvailable)
	assert.Equal(t, 0, fake.calls)
}

func TestCheckCachedFreshInvalidCachedVersion(t *testing.T) {
	fake := &fakeFetcher{}
	u := newFakeUpdater(t, fake)

	require.NoError(t, saveCache(u.opts.CacheDir, &cacheFile{
		LatestVersion:  "garbage",
		LastChecked:    time.Now().Add(-1 * time.Hour),
		CurrentVersion: "v1.0.0",
	}))

	res, err := u.CheckCached(context.Background(), "v1.0.0")
	require.NoError(t, err)
	assert.False(t, res.UpdateAvailable, "invalid cached version must not signal update")
}
