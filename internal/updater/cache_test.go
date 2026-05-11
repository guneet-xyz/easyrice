package updater

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCache_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	in := &cacheFile{
		LatestVersion:  "v1.2.3",
		LastChecked:    time.Now().UTC().Truncate(time.Second),
		CurrentVersion: "v1.0.0",
	}
	if err := saveCache(dir, in); err != nil {
		t.Fatalf("saveCache: %v", err)
	}
	out, err := loadCache(dir)
	if err != nil {
		t.Fatalf("loadCache: %v", err)
	}
	if out == nil {
		t.Fatal("loadCache returned nil")
	}
	if out.LatestVersion != in.LatestVersion ||
		out.CurrentVersion != in.CurrentVersion ||
		!out.LastChecked.Equal(in.LastChecked) {
		t.Fatalf("round-trip mismatch: in=%+v out=%+v", in, out)
	}
}

func TestCache_MissingFile(t *testing.T) {
	dir := t.TempDir()
	c, err := loadCache(dir)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if c != nil {
		t.Fatalf("expected nil cache, got %+v", c)
	}
}

func TestCache_CorruptJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, cacheFileName), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := loadCache(dir)
	if c != nil {
		t.Fatalf("expected nil cache, got %+v", c)
	}
	if !errors.Is(err, ErrCacheCorrupt) {
		t.Fatalf("expected ErrCacheCorrupt, got %v", err)
	}
}

func TestCache_AtomicWrite_NoTmpLeftover(t *testing.T) {
	dir := t.TempDir()
	in := &cacheFile{LatestVersion: "v1.0.0", LastChecked: time.Now().UTC(), CurrentVersion: "v1.0.0"}
	if err := saveCache(dir, in); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, cacheFileName+".tmp")); !os.IsNotExist(err) {
		t.Fatalf("expected no .tmp leftover, stat err=%v", err)
	}
}

func newTestCheckCachedUpdater(t *testing.T) *Updater {
	t.Helper()
	u, err := New(Options{
		Owner:    "guneet-xyz",
		Repo:     "easyrice",
		CacheDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return u
}

func readTestCache(t *testing.T, dir string) *cacheFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, cacheFileName))
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}
	var c cacheFile
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("unmarshal cache: %v", err)
	}
	return &c
}

func TestCheckCached_DevBuildShortCircuits(t *testing.T) {
	u := newTestCheckCachedUpdater(t)
	res, err := u.CheckCached(context.Background(), "dev")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.UpdateAvailable {
		t.Fatalf("dev build must not signal update")
	}
	if _, err := os.Stat(filepath.Join(u.opts.CacheDir, cacheFileName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dev build must not touch cache, got err=%v", err)
	}
}

func TestCheckCached_FirstRunSeedsSentinel(t *testing.T) {
	u := newTestCheckCachedUpdater(t)
	res, err := u.CheckCached(context.Background(), "v1.2.3")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.UpdateAvailable {
		t.Fatalf("first run must return UpdateAvailable=false")
	}
	c := readTestCache(t, u.opts.CacheDir)
	if c.LatestVersion != "v1.2.3" || c.CurrentVersion != "v1.2.3" {
		t.Fatalf("sentinel mismatch: %+v", c)
	}
}

func TestCheckCached_FreshCacheUpdateAvailable(t *testing.T) {
	u := newTestCheckCachedUpdater(t)
	if err := saveCache(u.opts.CacheDir, &cacheFile{
		LatestVersion:  "v2.0.0",
		LastChecked:    time.Now().Add(-1 * time.Hour),
		CurrentVersion: "v1.0.0",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	res, err := u.CheckCached(context.Background(), "v1.0.0")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.UpdateAvailable || res.Latest != "v2.0.0" {
		t.Fatalf("want UpdateAvailable=true Latest=v2.0.0, got %+v", res)
	}
}

func TestCheckCached_FreshCacheNoUpdate(t *testing.T) {
	u := newTestCheckCachedUpdater(t)
	if err := saveCache(u.opts.CacheDir, &cacheFile{
		LatestVersion:  "v1.0.0",
		LastChecked:    time.Now().Add(-1 * time.Hour),
		CurrentVersion: "v1.0.0",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	res, err := u.CheckCached(context.Background(), "v1.0.0")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.UpdateAvailable {
		t.Fatalf("expected UpdateAvailable=false")
	}
}

func TestCheckCached_StaleFailSilent(t *testing.T) {
	u := newTestCheckCachedUpdater(t)
	if err := saveCache(u.opts.CacheDir, &cacheFile{
		LatestVersion:  "v1.0.0",
		LastChecked:    time.Now().Add(-48 * time.Hour),
		CurrentVersion: "v1.0.0",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := u.CheckCached(ctx, "v1.0.0")
	if err != nil {
		t.Fatalf("fail-silent: err must be nil, got %v", err)
	}
	if res.UpdateAvailable {
		t.Fatalf("fail-silent: UpdateAvailable must be false")
	}
}

func TestCheckCached_ClockSkewTriggersRefresh(t *testing.T) {
	u := newTestCheckCachedUpdater(t)
	if err := saveCache(u.opts.CacheDir, &cacheFile{
		LatestVersion:  "v1.0.0",
		LastChecked:    time.Now().Add(48 * time.Hour),
		CurrentVersion: "v1.0.0",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := u.CheckCached(ctx, "v1.0.0")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.UpdateAvailable {
		t.Fatalf("expected fail-silent UpdateAvailable=false on cancelled fetch")
	}
}

func TestCheckCached_CorruptCacheTreatedAsMiss(t *testing.T) {
	u := newTestCheckCachedUpdater(t)
	if err := os.MkdirAll(u.opts.CacheDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(u.opts.CacheDir, cacheFileName), []byte("not json"), 0o600); err != nil {
		t.Fatalf("write corrupt: %v", err)
	}
	res, err := u.CheckCached(context.Background(), "v1.0.0")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.UpdateAvailable {
		t.Fatalf("corrupt-as-miss must seed sentinel and return false")
	}
	c := readTestCache(t, u.opts.CacheDir)
	if c.LatestVersion != "v1.0.0" {
		t.Fatalf("sentinel LatestVersion=%q want v1.0.0", c.LatestVersion)
	}
}
