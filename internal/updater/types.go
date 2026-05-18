package updater

import (
	"context"
	"net/http"
	"path/filepath"
	"time"

	"github.com/guneet-xyz/easyrice/internal/xdgpath"
)

// ReleaseFetcher abstracts release lookup (default: GitHub via go-selfupdate).
type ReleaseFetcher interface {
	FetchLatest(ctx context.Context) (*Release, error)
}

// Clock abstracts wall-clock time for deterministic tests.
type Clock interface {
	Now() time.Time
}

// Locker abstracts upgrade-lock acquisition.
type Locker interface {
	Acquire(path string) (release func() error, err error)
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

type flockLocker struct{}

func (flockLocker) Acquire(path string) (func() error, error) {
	rel, err := acquireLock(path)
	if err != nil {
		return nil, err
	}
	return func() error { rel(); return nil }, nil
}

// Release represents a GitHub release.
type Release struct {
	Version  string
	URL      string
	AssetURL string
}

// CheckResult represents the result of a version check.
type CheckResult struct {
	Current         string
	Latest          string
	UpdateAvailable bool
	CheckedAt       time.Time
}

// Options configures an Updater.
type Options struct {
	Owner      string
	Repo       string
	Timeout    time.Duration
	CacheDir   string
	HTTPClient *http.Client
	Fetcher    ReleaseFetcher
	Clock      Clock
	Locker     Locker
}

// DefaultCacheDir returns the platform-appropriate cache directory for updates.
// POSIX: ~/.config/easyrice/update-check.json
// Windows: %APPDATA%/easyrice/update-check.json
func DefaultCacheDir() string {
	return filepath.Join(xdgpath.ConfigDir(), "easyrice")
}
