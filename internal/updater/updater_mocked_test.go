package updater

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockedFetcher struct {
	release *Release
	err     error
	calls   int
	lastCtx context.Context
}

func (m *mockedFetcher) FetchLatest(ctx context.Context) (*Release, error) {
	m.calls++
	m.lastCtx = ctx
	return m.release, m.err
}

type mockedClock struct {
	now time.Time
}

func (c *mockedClock) Now() time.Time { return c.now }

type mockedLocker struct {
	acquireErr error
	releaseErr error
	acquired   int
	released   int
}

func (l *mockedLocker) Acquire(_ string) (func() error, error) {
	if l.acquireErr != nil {
		return nil, l.acquireErr
	}
	l.acquired++
	return func() error {
		l.released++
		return l.releaseErr
	}, nil
}

type mockedSwapper struct {
	err        error
	calls      int
	preHook    func()
	gotRealExe string
}

func (s *mockedSwapper) Swap(_ context.Context, _ string, _ string, realExe string) error {
	s.calls++
	s.gotRealExe = realExe
	if s.preHook != nil {
		s.preHook()
	}
	return s.err
}

func newMockedUpdater(t *testing.T, f *mockedFetcher, c *mockedClock, l *mockedLocker) *Updater {
	t.Helper()
	opts := Options{
		Owner:    "guneet-xyz",
		Repo:     "easyrice",
		CacheDir: t.TempDir(),
		Fetcher:  f,
		Clock:    c,
		Locker:   l,
	}
	u, err := New(opts)
	require.NoError(t, err)
	if f != nil {
		u.fetcher = f
	}
	return u
}

func TestUpdater_Mocked(t *testing.T) {
	t.Parallel()

	// BUG-080: ErrDevBuild — CheckCached("dev") MUST short-circuit before
	// touching the fetcher or the cache.
	// Spec: cache.go:88-95 + errors.go:6 + AGENTS.md updater table.
	t.Run("BUG_080_DevBuild", func(t *testing.T) {
		t.Log("BUG-080")
		fetcher := &mockedFetcher{}
		clock := &mockedClock{now: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
		u := newMockedUpdater(t, fetcher, clock, &mockedLocker{})

		res, err := u.CheckCached(context.Background(), "dev")
		require.NoError(t, err, "BUG-080: dev build must not error")
		require.NotNil(t, res)
		assert.False(t, res.UpdateAvailable, "BUG-080: dev build must report no update")
		assert.Equal(t, 0, fetcher.calls, "BUG-080: dev build must NEVER call fetcher")

		_, statErr := os.Stat(filepath.Join(u.opts.CacheDir, cacheFileName))
		assert.True(t, os.IsNotExist(statErr), "BUG-080: dev build must not write cache")
	})

	// BUG-081: ErrAlreadyLatest — fetcher returns the same semver as current.
	// Spec: errors.go:7 + fetch.go:60-67 + version.go:35-48.
	t.Run("BUG_081_AlreadyLatest", func(t *testing.T) {
		t.Log("BUG-081")
		now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		fetcher := &mockedFetcher{release: &Release{Version: "v1.0.0"}}
		u := newMockedUpdater(t, fetcher, &mockedClock{now: now}, &mockedLocker{})

		require.NoError(t, saveCache(u.opts.CacheDir, &cacheFile{
			LatestVersion:  "v0.9.0",
			LastChecked:    now.Add(-48 * time.Hour),
			CurrentVersion: "v1.0.0",
		}))

		res, err := u.CheckCached(context.Background(), "v1.0.0")
		require.NoError(t, err)
		assert.False(t, res.UpdateAvailable,
			"BUG-081: same-version response must NOT signal an update")
		assert.Equal(t, "v1.0.0", res.Latest)
		assert.Equal(t, 1, fetcher.calls)

		fetcher2 := &mockedFetcher{err: ErrAlreadyLatest}
		u2 := newMockedUpdater(t, fetcher2, &mockedClock{now: now}, &mockedLocker{})
		got, err := u2.FetchLatest(context.Background())
		assert.Nil(t, got)
		assert.True(t, errors.Is(err, ErrAlreadyLatest),
			"BUG-081: FetchLatest must propagate ErrAlreadyLatest verbatim")
	})

	// BUG-082: ErrLockBusy — pre-create the on-disk lockfile.
	// Spec: errors.go:8 + lock.go:35-51 + swap.go:36-39.
	t.Run("BUG_082_LockBusy", func(t *testing.T) {
		t.Log("BUG-082")
		sw := &mockedSwapper{}
		u, err := New(Options{
			Owner:    "x",
			Repo:     "y",
			CacheDir: t.TempDir(),
		})
		require.NoError(t, err)
		u.swapper = sw

		lockPath := filepath.Join(u.opts.CacheDir, lockFileName)
		require.NoError(t, os.WriteFile(lockPath, []byte("99999\n"), 0o644))

		marker := filepath.Join(u.opts.CacheDir, "binary-marker")
		require.NoError(t, os.WriteFile(marker, []byte("OLD"), 0o644))

		rel := &Release{Version: "v2.0.0", AssetURL: "https://example.com/asset.tar.gz"}
		err = u.Apply(context.Background(), rel)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrLockBusy),
			"BUG-082: Apply must return ErrLockBusy when lock is held, got %v", err)
		assert.Equal(t, 0, sw.calls,
			"BUG-082: swapper must NOT be invoked when lock is busy")

		got, _ := os.ReadFile(marker)
		assert.Equal(t, "OLD", string(got),
			"BUG-082: binary must remain in pre-Apply state")
	})

	// BUG-083: ErrNoChecksum — fetcher reports a release with no checksums.
	// Spec: errors.go:9 + fetch.go:69-72 + AGENTS.md updater table.
	t.Run("BUG_083_NoChecksum", func(t *testing.T) {
		t.Log("BUG-083")
		fetcher := &mockedFetcher{err: ErrNoChecksum}
		u := newMockedUpdater(t, fetcher, &mockedClock{now: time.Now()}, &mockedLocker{})

		got, err := u.FetchLatest(context.Background())
		assert.Nil(t, got)
		assert.True(t, errors.Is(err, ErrNoChecksum),
			"BUG-083: FetchLatest must surface ErrNoChecksum verbatim, got %v", err)
	})

	// BUG-084: ErrCacheCorrupt — invalid JSON detected by loadCache AND
	// CheckCached MUST fall back to a fresh seed/fetch.
	// Spec: cache.go:30-48 + cache.go:97-103 + errors.go:10.
	t.Run("BUG_084_CacheCorrupt", func(t *testing.T) {
		t.Log("BUG-084")
		now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		fetcher := &mockedFetcher{}
		u := newMockedUpdater(t, fetcher, &mockedClock{now: now}, &mockedLocker{})

		require.NoError(t, os.MkdirAll(u.opts.CacheDir, 0o700))
		corrupt := filepath.Join(u.opts.CacheDir, cacheFileName)
		require.NoError(t, os.WriteFile(corrupt, []byte("{not json"), 0o600))

		_, lerr := loadCache(u.opts.CacheDir)
		assert.True(t, errors.Is(lerr, ErrCacheCorrupt),
			"BUG-084: loadCache must return ErrCacheCorrupt for invalid JSON, got %v", lerr)

		res, err := u.CheckCached(context.Background(), "v1.0.0")
		require.NoError(t, err, "BUG-084: CheckCached must swallow ErrCacheCorrupt")
		assert.False(t, res.UpdateAvailable)
		c, err := loadCache(u.opts.CacheDir)
		require.NoError(t, err, "BUG-084: cache must be re-seeded after corrupt detection")
		require.NotNil(t, c)
		assert.Equal(t, "v1.0.0", c.LatestVersion,
			"BUG-084: re-seeded cache must record the current version")
	})

	// BUG-085: ErrInvalidSemver — non-semver tag rejected by IsNewer.
	// Spec: errors.go:11 + version.go:35-48.
	t.Run("BUG_085_InvalidSemver", func(t *testing.T) {
		t.Log("BUG-085")
		_, err := IsNewer("v1.0.0", "not-a-version")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidSemver),
			"BUG-085: IsNewer must wrap ErrInvalidSemver for non-semver tag, got %v", err)

		_, err = IsNewer("not-a-version", "v1.0.0")
		assert.True(t, errors.Is(err, ErrInvalidSemver),
			"BUG-085: IsNewer must wrap ErrInvalidSemver for invalid current, got %v", err)
	})

	// BUG-086: Pre-release rejection — tag with -beta.N has a prerelease
	// component; IsPreRelease returns true.
	// Spec: version.go:50-55 + AGENTS.md updater table.
	t.Run("BUG_086_PreReleaseRejected", func(t *testing.T) {
		t.Log("BUG-086")
		assert.True(t, IsPreRelease("v2.0.0-beta.1"),
			"BUG-086: tag with -beta.N must be classified as pre-release")
		assert.True(t, IsPreRelease("v1.0.0-rc.1"),
			"BUG-086: tag with -rc.N must be classified as pre-release")
		assert.False(t, IsPreRelease("v2.0.0"),
			"BUG-086: tag without prerelease component must NOT be a pre-release")
	})

	// BUG-087: Network failure during fetch — CheckCached MUST be fail-silent.
	// Spec: cache.go:137-145 + AGENTS.md ("CheckCached: fail-silent network").
	t.Run("BUG_087_NetworkFailureFailSilent", func(t *testing.T) {
		t.Log("BUG-087")
		now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		fetcher := &mockedFetcher{err: errors.New("connection refused")}
		u := newMockedUpdater(t, fetcher, &mockedClock{now: now}, &mockedLocker{})

		require.NoError(t, saveCache(u.opts.CacheDir, &cacheFile{
			LatestVersion:  "v1.0.0",
			LastChecked:    now.Add(-48 * time.Hour),
			CurrentVersion: "v1.0.0",
		}))

		res, err := u.CheckCached(context.Background(), "v1.0.0")
		require.NoError(t, err, "BUG-087: network error must NOT bubble to caller")
		require.NotNil(t, res)
		assert.False(t, res.UpdateAvailable,
			"BUG-087: fail-silent must report UpdateAvailable=false")
		assert.Equal(t, 1, fetcher.calls,
			"BUG-087: stale-path must hit the fetcher exactly once")
	})

	// BUG-088: Atomic swap rollback — when the swapper fails mid-way, the
	// running binary remains in pre-swap state, the lock is released, and the
	// error is propagated.
	// Spec: swap.go:54-58 + AGENTS.md ("Atomic binary swap; never re-execs").
	t.Run("BUG_088_AtomicSwapRollback", func(t *testing.T) {
		t.Log("BUG-088")
		dir := t.TempDir()
		fakeBinary := filepath.Join(dir, "easyrice-fake")
		require.NoError(t, os.WriteFile(fakeBinary, []byte("OLD-BYTES"), 0o755))

		newSibling := fakeBinary + ".new"
		sw := &mockedSwapper{
			err: errors.New("swap interrupted: io error"),
			preHook: func() {
				_ = os.WriteFile(newSibling, []byte("PARTIAL"), 0o644)
			},
		}

		u, err := New(Options{
			Owner:    "x",
			Repo:     "y",
			CacheDir: t.TempDir(),
		})
		require.NoError(t, err)
		u.swapper = sw

		rel := &Release{Version: "v2.0.0", AssetURL: "https://example.com/asset.tar.gz"}
		err = u.Apply(context.Background(), rel)
		require.Error(t, err, "BUG-088: mid-swap failure must propagate an error")
		assert.Contains(t, err.Error(), "swap interrupted",
			"BUG-088: underlying swapper error must be wrapped, not swallowed")

		got, _ := os.ReadFile(fakeBinary)
		assert.Equal(t, "OLD-BYTES", string(got),
			"BUG-088: binary must remain in pre-swap state on error")

		_, statErr := os.Stat(filepath.Join(u.opts.CacheDir, lockFileName))
		assert.True(t, os.IsNotExist(statErr),
			"BUG-088: lockfile must be released after a failed swap")

		_ = newSibling
	})

	// BUG-089: CleanupOrphanArtifacts removes .new (all OSes) and .old
	// (non-Windows) siblings of the resolved binary path.
	// Spec: cleanup.go:14-40 + AGENTS.md ("Removes .new/.old siblings").
	t.Run("BUG_089_CleanupOrphanArtifacts", func(t *testing.T) {
		t.Log("BUG-089")
		dir := t.TempDir()
		bin := filepath.Join(dir, "easyrice-bin")
		require.NoError(t, os.WriteFile(bin, []byte("bin"), 0o755))

		newOrphan := bin + ".new"
		oldOrphan := bin + ".old"
		require.NoError(t, os.WriteFile(newOrphan, []byte("partial"), 0o644))
		require.NoError(t, os.WriteFile(oldOrphan, []byte("backup"), 0o644))

		require.NoError(t, CleanupOrphanArtifacts(bin),
			"BUG-089: CleanupOrphanArtifacts must succeed when orphans exist")

		_, errNew := os.Stat(newOrphan)
		assert.True(t, os.IsNotExist(errNew),
			"BUG-089: .new sibling must be removed")

		_, errOld := os.Stat(oldOrphan)
		if runtime.GOOS == "windows" {
			assert.NoError(t, errOld, "BUG-089: .old must remain on Windows")
		} else {
			assert.True(t, os.IsNotExist(errOld),
				"BUG-089: .old sibling must be removed on non-Windows")
		}

		assert.NoError(t, CleanupOrphanArtifacts(bin),
			"BUG-089: cleanup must be idempotent (no orphans => no error)")

		got, _ := os.ReadFile(bin)
		assert.Equal(t, "bin", string(got),
			"BUG-089: cleanup must NEVER touch the real binary")
	})

	// BUG-090: 24h TTL boundary — within TTL use cache, beyond TTL re-fetch.
	// Spec: cache.go:19 (cacheTTL = 24h) + cache.go:122-135 (staleness).
	t.Run("BUG_090_TTLBoundary", func(t *testing.T) {
		t.Log("BUG-090")
		t0 := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
		fetcher := &mockedFetcher{release: &Release{Version: "v2.0.0"}}
		clock := &mockedClock{now: t0}
		u := newMockedUpdater(t, fetcher, clock, &mockedLocker{})

		require.NoError(t, saveCache(u.opts.CacheDir, &cacheFile{
			LatestVersion:  "v1.0.0",
			LastChecked:    t0,
			CurrentVersion: "v1.0.0",
		}))

		clock.now = t0.Add(23*time.Hour + 59*time.Minute)
		res, err := u.CheckCached(context.Background(), "v1.0.0")
		require.NoError(t, err)
		assert.False(t, res.UpdateAvailable,
			"BUG-090: within TTL, cache says v1.0.0 == current, no update")
		assert.Equal(t, "v1.0.0", res.Latest)
		assert.Equal(t, 0, fetcher.calls,
			"BUG-090: within TTL (23h59m), fetcher MUST NOT be called")

		clock.now = t0.Add(24*time.Hour + 1*time.Minute)
		res, err = u.CheckCached(context.Background(), "v1.0.0")
		require.NoError(t, err)
		assert.Equal(t, 1, fetcher.calls,
			"BUG-090: at TTL+1m (24h01m), fetcher MUST be called exactly once")
		assert.Equal(t, "v2.0.0", res.Latest,
			"BUG-090: after re-fetch, Latest reflects the new release")
		assert.True(t, res.UpdateAvailable,
			"BUG-090: v1.0.0 to v2.0.0 must report UpdateAvailable=true")
	})

	// BUG-091: FormatReminder output must contain current version, latest
	// version, and the canonical GitHub releases URL.
	// Spec: reminder.go:10-17.
	t.Run("BUG_091_FormatReminder", func(t *testing.T) {
		t.Log("BUG-091")
		got := FormatReminder("v1.0.0", "v2.0.0", "guneet-xyz", "easyrice")
		assert.Contains(t, got, "v1.0.0",
			"BUG-091: reminder must contain current version")
		assert.Contains(t, got, "v2.0.0",
			"BUG-091: reminder must contain latest version")
		assert.Contains(t, got, "https://github.com/guneet-xyz/easyrice/releases/latest",
			"BUG-091: reminder must contain the canonical releases URL")
		lines := strings.Split(got, "\n")
		assert.Equal(t, 2, len(lines), "BUG-091: reminder must be exactly 2 lines")
		assert.NotEqual(t, "", lines[1], "BUG-091: second line must not be empty")
	})

	// BUG-092: IsTerminal / ShouldShowReminder — non-TTY stderr suppresses
	// the reminder regardless of other inputs.
	// Spec: reminder.go:26-43.
	t.Run("BUG_092_NonTTYNoReminder", func(t *testing.T) {
		t.Log("BUG-092")
		f, err := tempNonTTYFile(t)
		require.NoError(t, err)
		defer f.Close()
		assert.False(t, IsTerminal(f),
			"BUG-092: a regular file MUST NOT be classified as a terminal")

		assert.False(t, ShouldShowReminder(false, "v1.0.0", false),
			"BUG-092: non-TTY stderr must suppress reminder for tagged build")
		assert.False(t, ShouldShowReminder(false, "dev", true),
			"BUG-092: dev build must suppress reminder even on a TTY")
		assert.False(t, ShouldShowReminder(true, "v1.0.0", true),
			"BUG-092: explicit disable must suppress reminder even on TTY")
		assert.True(t, ShouldShowReminder(false, "v1.0.0", true),
			"BUG-092: tagged build + TTY + not-disabled must show reminder")
	})

	t.Run("HappyPath", func(t *testing.T) {
		now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
		release := &Release{
			Version:  "v3.0.0",
			URL:      "https://example.com/release/v3.0.0",
			AssetURL: "https://example.com/easyrice_linux_amd64.tar.gz",
		}
		fetcher := &mockedFetcher{release: release}
		clock := &mockedClock{now: now}
		u := newMockedUpdater(t, fetcher, clock, &mockedLocker{})

		got, err := u.FetchLatest(context.Background())
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "v3.0.0", got.Version)
		assert.Equal(t, release.AssetURL, got.AssetURL)
		assert.Equal(t, 1, fetcher.calls)

		sw := &mockedSwapper{preHook: func() {}}
		u.swapper = sw
		require.NoError(t, u.Apply(context.Background(), got))
		assert.Equal(t, 1, sw.calls, "HappyPath: swapper invoked exactly once")
		assert.NotEmpty(t, sw.gotRealExe,
			"HappyPath: swapper must receive the resolved exe path")

		require.NoError(t, saveCache(u.opts.CacheDir, &cacheFile{
			LatestVersion:  "v1.0.0",
			LastChecked:    now.Add(-48 * time.Hour),
			CurrentVersion: "v1.0.0",
		}))
		fetcher.calls = 0
		res, err := u.CheckCached(context.Background(), "v1.0.0")
		require.NoError(t, err)
		assert.True(t, res.UpdateAvailable, "HappyPath: v1.0.0 to v3.0.0 update")
		assert.Equal(t, "v3.0.0", res.Latest)
		assert.Equal(t, 1, fetcher.calls)

		data, rerr := os.ReadFile(filepath.Join(u.opts.CacheDir, cacheFileName))
		require.NoError(t, rerr)
		var c cacheFile
		require.NoError(t, json.Unmarshal(data, &c))
		assert.Equal(t, "v3.0.0", c.LatestVersion,
			"HappyPath: cache file must record the new latest version")
		assert.Equal(t, "v1.0.0", c.CurrentVersion)
	})
}
