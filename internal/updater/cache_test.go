package updater

import (
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
