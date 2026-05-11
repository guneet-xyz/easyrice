# internal/updater/

Self-update for the `easyrice` binary. Owns GitHub release polling, the local update-check cache, the upgrade lock, and the binary-swap path. The ONLY package allowed to talk to GitHub or run `go-selfupdate`.

## OVERVIEW

Wraps the external "is there a newer release?" question and the "swap the running binary" mechanic behind a single internal API. Keeps networking, file-format, and self-replacement concerns inside one boundary so the rest of the codebase stays offline and pure.

The package has two consumers in `cli/`:

1. `cli/upgrade.go` — interactive `easyrice upgrade` command. Calls `New` → `FetchLatest` → `IsNewer` → `Apply`.
2. `cli/reminder.go` — non-blocking post-command reminder. Calls `New` → `CheckCached` → `FormatReminder`.

All network and disk I/O for self-update goes through this package. No other package imports `creativeprojects/go-selfupdate` or touches `update-check.json` / `upgrade.lock`.

## PUBLIC API

| Symbol | Type | Location | Role |
|--------|------|----------|------|
| `Updater` | struct | `updater.go` | Carries `Options` for one updater instance |
| `New(Options) (*Updater, error)` | func | `updater.go` | Constructor; validates `Owner`/`Repo`, fills defaults |
| `Options` | struct | `types.go` | `Owner`, `Repo`, `Timeout`, `CacheDir`, `HTTPClient` |
| `Release` | struct | `types.go` | `Version`, `URL`, `AssetURL` returned by `FetchLatest` |
| `CheckResult` | struct | `types.go` | `Current`, `Latest`, `UpdateAvailable`, `CheckedAt` |
| `DefaultCacheDir() string` | func | `types.go` | `~/.config/easyrice` (POSIX) / `%APPDATA%/easyrice` (Windows) |
| `IsDevBuild(v string) bool` | func | `version.go` | True for `""`, `"dev"`, or non-semver tags |
| `IsNewer(current, latest string) (bool, error)` | func | `version.go` | Semver comparison via `golang.org/x/mod/semver` |
| `IsPreRelease(v string) bool` | func | `version.go` | True if the tag has a pre-release component |
| `(*Updater).FetchLatest(ctx) (*Release, error)` | method | `fetch.go` | Anonymous GitHub release lookup; rejects pre-releases and missing checksums |
| `(*Updater).Apply(ctx, *Release) error` | method | `swap.go` | Atomic binary swap; resolves symlinks; holds upgrade lock |
| `(*Updater).CheckCached(ctx, current string) (*CheckResult, error)` | method | `cache.go` | 24h-TTL cached check; fail-silent network |
| `FormatReminder(current, latest, owner, repo string) string` | func | `reminder.go` | 2-line stderr reminder string |
| `ShouldShowReminder(disabled bool, current string, isStderrTTY bool) bool` | func | `reminder.go` | Pure gating predicate |
| `IsTerminal(*os.File) bool` | func | `reminder.go` | Wraps `golang.org/x/term.IsTerminal` |
| `CleanupOrphanArtifacts(execPath string) error` | func | `cleanup.go` | Removes `.new`/`.old` siblings left by interrupted swaps |
| `ErrDevBuild`, `ErrAlreadyLatest`, `ErrLockBusy`, `ErrNoChecksum`, `ErrCacheCorrupt`, `ErrInvalidSemver` | var | `errors.go` | Sentinel errors for CLI mapping |

## BOUNDARY RULES

1. **Only this package may call the GitHub API.** No `github.com/...` HTTP calls or `gh` shell-outs anywhere else in the codebase.
2. **Only this package may read or write `update-check.json` and `upgrade.lock`.** Other packages MUST go through the exported API.
3. **Only this package may invoke binary-swap or call `go-selfupdate`.** No `os.Rename` of `os.Executable()` outside this package.
4. **Forbidden inside this package:**
   - `os.Symlink` (use `internal/symlink/` if ever needed, but it should not be needed here)
   - `exec.Command("git", ...)` (git lives in `internal/repo/` only)
   - `panic` outside `*_test.go`
   - `http://` literals (HTTPS only for GitHub API and asset downloads)
   - Reading `GITHUB_TOKEN` from env (this is a public-release flow; no auth)

## FILES

| File | Symbols | Role |
|------|---------|------|
| `updater.go` | package doc, `Updater`, `New` | Constructor; validates `Owner`/`Repo`, applies default `Timeout`, `HTTPClient`, `CacheDir` |
| `types.go` | `Options`, `Release`, `CheckResult`, `DefaultCacheDir` | Plain data types and the platform-aware cache-dir helper |
| `errors.go` | `ErrDevBuild`, `ErrAlreadyLatest`, `ErrLockBusy`, `ErrNoChecksum`, `ErrCacheCorrupt`, `ErrInvalidSemver` | Sentinel errors used by CLI for exit-code and message mapping |
| `version.go` | `IsDevBuild`, `IsNewer`, `IsPreRelease`, unexported `normalize` | Pure semver helpers built on `golang.org/x/mod/semver` |
| `fetch.go` | `(*Updater).FetchLatest(ctx)` | GitHub release lookup via `creativeprojects/go-selfupdate`; pre-release filter; checksum-required gate |
| `swap.go` | `(*Updater).Apply(ctx, release)` | Holds the lock, resolves `os.Executable()` through `EvalSymlinks`, delegates atomic swap to `go-selfupdate`; never re-execs |
| `lock.go` | unexported `acquireLock(cacheDir)`, `tryCreateLock`, `makeReleaser` | `O_CREATE\|O_EXCL` lockfile with PID, 1h stale reclaim, single retry |
| `cache.go` | `cacheTTL`, `loadCache`, `saveCache`, `(*Updater).CheckCached` | 24h-TTL `update-check.json` with atomic write; fail-silent on network |
| `reminder.go` | `FormatReminder`, `ShouldShowReminder`, `IsTerminal` | Pure helpers for the post-command stderr reminder |
| `cleanup.go` | `CleanupOrphanArtifacts` | Idempotent removal of `.new` (all OS) and `.old` (non-Windows) siblings |

Test files (`*_test.go`) live next to each implementation file.

## CONVENTIONS

- HTTPS only. No `http://` string literals anywhere in this package; `go-selfupdate` is configured to use the GitHub API client which is HTTPS-only.
- No `GITHUB_TOKEN` env reads. The v1 flow is anonymous; rate-limit pressure is mitigated by the 24h cache and dev-build short-circuit.
- No HTTP response bodies in logs. Network errors are wrapped with `fmt.Errorf("updater: ...: %w", err)` and surfaced; raw bodies never reach the logger.
- All FS writes go to `DefaultCacheDir()` (`update-check.json` and `upgrade.lock`). The single exception is `Apply`, which writes the new binary to the resolved path of `os.Executable()`.
- Lockfile semantics: created with `os.O_CREATE|os.O_EXCL|os.O_WRONLY` and the current PID, treated as stale after 1h (`staleLockAge`), reclaimed exactly once.
- Cache writes are atomic: write `update-check.json.tmp` (mode `0o600`) then `os.Rename`.
- Errors wrap with `fmt.Errorf("updater: <context>: %w", err)`. Sentinels in `errors.go` are matched by callers via `errors.Is`.
- Pure helpers (`IsDevBuild`, `IsNewer`, `IsPreRelease`, `FormatReminder`, `ShouldShowReminder`, `IsTerminal`) are deliberately free of I/O so they can be unit-tested without mocks.

## ANTI-PATTERNS (forbidden in this package)

- Auto-restarting the binary after a successful swap. `Apply` returns; the CLI prints a restart hint; the user re-runs.
- Adding `GITHUB_TOKEN` (or any auth) support. v1 is anonymous-only; introducing tokens would require config plumbing, secret-handling docs, and a second test surface for no first-iteration win.
- Making a blocking network call on first run. The reminder path uses `CheckCached` with seeded sentinel cache so first-run users never wait on the network for a non-essential message.
- Writing outside `~/.config/easyrice/` (POSIX) / `%APPDATA%/easyrice/` (Windows). The single exception is the binary-swap path inside `Apply`, which writes to `os.Executable()`.
- Modifying `cli/update.go` semantics from this package, or aliasing `update`/`upgrade`. They are deliberately distinct: `update` pulls the dotfile repo, `upgrade` replaces the binary.
- Bypassing the `internal/updater` boundary: no other package may import `creativeprojects/go-selfupdate`, call the GitHub API, or touch `update-check.json`/`upgrade.lock` directly.
