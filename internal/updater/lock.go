package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// staleLockAge is the threshold above which an existing lockfile is treated as
// abandoned (e.g. crashed previous upgrade) and may be reclaimed.
const staleLockAge = time.Hour

// lockFileName is the basename of the upgrade lockfile inside the cache dir.
const lockFileName = "upgrade.lock"

// acquireLock creates an exclusive upgrade lockfile in cacheDir.
//
// Behavior:
//   - Creates <cacheDir>/upgrade.lock with O_CREATE|O_EXCL|O_WRONLY and writes
//     the current PID into it.
//   - If the lockfile already exists and is fresh (mtime within staleLockAge),
//     returns ErrLockBusy.
//   - If the lockfile already exists but is stale (mtime older than
//     staleLockAge), removes it and retries acquisition exactly once.
//   - On success, returns a release closure that closes the file handle and
//     removes the lockfile. The closure is safe to call exactly once.
func acquireLock(cacheDir string) (release func(), err error) {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("updater: create cache dir: %w", err)
	}

	lockPath := filepath.Join(cacheDir, lockFileName)

	f, err := tryCreateLock(lockPath)
	if err == nil {
		return makeReleaser(f, lockPath), nil
	}
	if !os.IsExist(err) {
		return nil, fmt.Errorf("updater: open lockfile: %w", err)
	}

	// Existing lockfile present — check whether it is stale.
	info, statErr := os.Stat(lockPath)
	if statErr != nil {
		// Vanished between create and stat — treat as busy to be safe.
		return nil, ErrLockBusy
	}
	if time.Since(info.ModTime()) < staleLockAge {
		return nil, ErrLockBusy
	}

	// Stale: remove and retry exactly once.
	if rmErr := os.Remove(lockPath); rmErr != nil && !os.IsNotExist(rmErr) {
		return nil, fmt.Errorf("updater: remove stale lockfile: %w", rmErr)
	}

	f, err = tryCreateLock(lockPath)
	if err != nil {
		if os.IsExist(err) {
			return nil, ErrLockBusy
		}
		return nil, fmt.Errorf("updater: open lockfile after stale reclaim: %w", err)
	}
	return makeReleaser(f, lockPath), nil
}

// tryCreateLock attempts a single O_EXCL create+PID-write.
func tryCreateLock(lockPath string) (*os.File, error) {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	if _, werr := fmt.Fprintf(f, "%d\n", os.Getpid()); werr != nil {
		_ = f.Close()
		_ = os.Remove(lockPath)
		return nil, fmt.Errorf("updater: write pid to lockfile: %w", werr)
	}
	return f, nil
}

// makeReleaser builds an idempotent release closure for an acquired lockfile.
func makeReleaser(f *os.File, lockPath string) func() {
	var done bool
	return func() {
		if done {
			return
		}
		done = true
		_ = f.Close()
		_ = os.Remove(lockPath)
	}
}
