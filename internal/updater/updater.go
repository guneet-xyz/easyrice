package updater

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// Package updater owns all version-check, GitHub-release fetch, binary swap, cache, and lockfile logic.
// No other package may call the GitHub API or write the cache.
// It provides a single boundary for self-update concerns: checking for new releases,
// validating checksums, managing the update cache with TTL, and coordinating atomic binary swaps.
// The Updater type is injectable and carries state for HTTP client, cache directory, and GitHub credentials.
type Updater struct {
	opts    Options
	fetcher releaseFetcher
	swapper swapper
}

// releaseFetcher abstracts release lookup behind an interface so callers can
// substitute alternative sources (default: GitHub via go-selfupdate).
type releaseFetcher interface {
	FetchLatest(ctx context.Context) (*Release, error)
}

// New constructs a new Updater with the given options.
// Returns an error if Owner or Repo is empty.
func New(opts Options) (*Updater, error) {
	if opts.Owner == "" {
		return nil, errors.New("updater: Owner is required")
	}
	if opts.Repo == "" {
		return nil, errors.New("updater: Repo is required")
	}

	// Set defaults if not provided
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = http.DefaultClient
	}
	if opts.CacheDir == "" {
		opts.CacheDir = DefaultCacheDir()
	}

	return &Updater{opts: opts, swapper: &goSelfupdateSwapper{}}, nil
}
