package updater

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/guneet-xyz/easyrice/internal/logger"
)

// Apply downloads the asset for release and atomically replaces the running
// binary with it.
//
// Preconditions:
//   - release must be the result of a successful (*Updater).FetchLatest call.
//   - The caller is responsible for the IsDevBuild() gate; Apply does not
//     re-check.
//
// Behavior:
//   - Acquires the upgrade lockfile in u.opts.CacheDir; returns ErrLockBusy
//     when another upgrade is in progress.
//   - Resolves os.Executable() through filepath.EvalSymlinks so that the swap
//     targets the real binary, not a wrapper symlink (e.g. the `rice` alias
//     installed by `make install`). On EvalSymlinks failure, falls back to
//     the unresolved path and logs at debug.
//   - Delegates the actual atomic swap (and rollback on failure) to
//     go-selfupdate.
//   - Does NOT re-exec the new binary; the caller prints a restart hint.
func (u *Updater) Apply(ctx context.Context, release *Release) error {
	if release == nil {
		return fmt.Errorf("updater: apply: release is nil")
	}

	releaseLock, err := acquireLock(u.opts.CacheDir)
	if err != nil {
		return fmt.Errorf("updater: acquire lock: %w", err)
	}
	defer releaseLock()

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("updater: resolve executable: %w", err)
	}

	realExe, err := filepath.EvalSymlinks(exe)
	if err != nil {
		logger.L.Debug(fmt.Sprintf("updater: EvalSymlinks failed for %q, falling back: %v", exe, err))
		realExe = exe
	}

	// SECURITY: HTTPS only — release.AssetURL originates from go-selfupdate's
	// GitHub source which uses HTTPS exclusively (see fetch.go).
	assetFileName := filepath.Base(release.AssetURL)
	if err := u.swapper.Swap(ctx, release.AssetURL, assetFileName, realExe); err != nil {
		return fmt.Errorf("updater: apply update to %q: %w", realExe, err)
	}

	return nil
}
