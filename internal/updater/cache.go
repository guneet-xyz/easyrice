package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	"github.com/guneet-xyz/easyrice/internal/logger"
)

// cacheTTL is the freshness window for the update-check cache.
// Within this window, CheckCached returns cached data without hitting the network.
const cacheTTL = 24 * time.Hour

// cacheFile is the on-disk format for update-check.json.
type cacheFile struct {
	LatestVersion  string    `json:"latest_version"`
	LastChecked    time.Time `json:"last_checked"`
	CurrentVersion string    `json:"current_version"`
}

const cacheFileName = "update-check.json"

// loadCache reads <dir>/update-check.json.
// Returns (nil, nil) if the file does not exist.
// Returns (nil, ErrCacheCorrupt) wrapped if the JSON is invalid.
func loadCache(dir string) (*cacheFile, error) {
	path := filepath.Join(dir, cacheFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("updater: cache: read: %w", err)
	}

	var c cacheFile
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("updater: cache: %w", ErrCacheCorrupt)
	}
	return &c, nil
}

// saveCache writes the cache atomically to <dir>/update-check.json.
// Creates the directory if needed.
func saveCache(dir string, c *cacheFile) error {
	if c == nil {
		return fmt.Errorf("updater: cache: save: nil cache")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("updater: cache: mkdir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("updater: cache: marshal: %w", err)
	}

	final := filepath.Join(dir, cacheFileName)
	tmp := final + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("updater: cache: write tmp: %w", err)
	}
	if err := os.Rename(tmp, final); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("updater: cache: rename: %w", err)
	}
	return nil
}

// CheckCached returns the cached update-check result, refreshing it from GitHub
// when the cache is missing, stale (>24h), or has a future-dated LastChecked
// (clock skew). Fetch errors are logged at debug level and swallowed so the
// reminder path never bubbles a network failure to the caller.
//
// First-run behavior: when no cache exists and current is a tagged release, a
// sentinel cache entry is seeded (LatestVersion = current) WITHOUT touching
// the network. Dev builds short-circuit to UpdateAvailable=false without
// touching the cache or the network.
func (u *Updater) CheckCached(ctx context.Context, current string) (*CheckResult, error) {
	now := time.Now()

	if IsDevBuild(current) {
		return &CheckResult{
			Current:         current,
			UpdateAvailable: false,
			CheckedAt:       now,
		}, nil
	}

	c, err := loadCache(u.opts.CacheDir)
	if err != nil && !errors.Is(err, ErrCacheCorrupt) {
		return nil, err
	}
	if errors.Is(err, ErrCacheCorrupt) {
		c = nil
	}

	if c == nil {
		seed := &cacheFile{
			LatestVersion:  current,
			LastChecked:    now,
			CurrentVersion: current,
		}
		if saveErr := saveCache(u.opts.CacheDir, seed); saveErr != nil {
			logger.Debug("updater: seed cache failed", zap.Error(saveErr))
		}
		return &CheckResult{
			Current:         current,
			Latest:          current,
			UpdateAvailable: false,
			CheckedAt:       now,
		}, nil
	}

	stale := now.Sub(c.LastChecked) > cacheTTL || c.LastChecked.After(now)

	if !stale {
		updateAvailable, cmpErr := IsNewer(current, c.LatestVersion)
		if cmpErr != nil {
			updateAvailable = false
		}
		return &CheckResult{
			Current:         current,
			Latest:          c.LatestVersion,
			UpdateAvailable: updateAvailable,
			CheckedAt:       c.LastChecked,
		}, nil
	}

	release, fetchErr := u.FetchLatest(ctx)
	if fetchErr != nil {
		logger.Debug("updater: fetch latest failed (fail-silent)", zap.Error(fetchErr))
		return &CheckResult{
			Current:         current,
			UpdateAvailable: false,
			CheckedAt:       now,
		}, nil
	}

	updated := &cacheFile{
		LatestVersion:  release.Version,
		LastChecked:    now,
		CurrentVersion: current,
	}
	if saveErr := saveCache(u.opts.CacheDir, updated); saveErr != nil {
		logger.Debug("updater: save cache failed", zap.Error(saveErr))
	}

	updateAvailable, cmpErr := IsNewer(current, release.Version)
	if cmpErr != nil {
		updateAvailable = false
	}
	return &CheckResult{
		Current:         current,
		Latest:          release.Version,
		UpdateAvailable: updateAvailable,
		CheckedAt:       now,
	}, nil
}
