package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/guneet-xyz/easyrice/internal/logger"
	"github.com/guneet-xyz/easyrice/internal/updater"
	"go.uber.org/zap"
)

// reminderTTYFn returns whether stderr is a TTY. Overridable in tests.
var reminderTTYFn = func() bool { return updater.IsTerminal(os.Stderr) }

// reminderCheckFn returns the cached update-check result. Overridable in tests
// to avoid touching the network.
var reminderCheckFn = func(ctx context.Context, current string) (*updater.CheckResult, error) {
	u, err := updater.New(updater.Options{
		Owner:    "guneet-xyz",
		Repo:     "easyrice",
		Timeout:  5 * time.Second,
		CacheDir: updater.DefaultCacheDir(),
	})
	if err != nil {
		return nil, err
	}
	return u.CheckCached(ctx, current)
}

// maybePrintUpdateReminder checks the update cache and prints a reminder to
// stderr if a newer version is available, the user hasn't opted out, and
// stderr is a TTY. It is fail-silent: errors are logged at Debug only.
func maybePrintUpdateReminder() {
	if UpdateCheckDisabled() {
		return
	}
	tty := reminderTTYFn()
	if !updater.ShouldShowReminder(false, Version, tty) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := reminderCheckFn(ctx, Version)
	if err != nil {
		logger.L.Debug("update check: cache error", zap.Error(err))
		return
	}
	if result == nil || !result.UpdateAvailable {
		return
	}
	newer, err := updater.IsNewer(Version, result.Latest)
	if err != nil || !newer {
		return
	}
	fmt.Fprintln(os.Stderr, updater.FormatReminder(Version, result.Latest, "guneet-xyz", "easyrice"))
}
