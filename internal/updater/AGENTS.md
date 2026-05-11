# internal/updater/

Self-update for the `easyrice` binary. Owns GitHub release polling, the local update-check cache, the upgrade lock, and the binary-swap path. The ONLY package allowed to talk to GitHub or run `go-selfupdate`.

## OVERVIEW

(placeholder — filled in T18)

Wraps the external "is there a newer release?" question and the "swap the running binary" mechanic behind a single internal API. Keeps networking, file-format, and self-replacement concerns inside one boundary so the rest of the codebase stays offline and pure.

## PUBLIC API

(placeholder — filled in T18)

| Symbol | Type | Location | Role |
|--------|------|----------|------|
| `Updater` | struct | `updater.go` | (placeholder — filled in T18) |
| `CheckLatest` | func | `updater.go` | (placeholder — filled in T18) |
| `Apply` | func | `updater.go` | (placeholder — filled in T18) |
| `LoadCache`, `SaveCache` | func | `types.go` | (placeholder — filled in T18) |
| `CleanupOrphans` | func | `cleanup.go` | (placeholder — filled in T18) |
| `ErrUpToDate`, `ErrLocked` | var | `errors.go` | (placeholder — filled in T18) |

## BOUNDARY RULES

1. **Only this package may call the GitHub API.** No `github.com/...` HTTP calls or `gh` shell-outs anywhere else in the codebase.
2. **Only this package may read or write `update-check.json` and `upgrade.lock`.** Other packages MUST go through the exported API.
3. **Only this package may invoke binary-swap or call `go-selfupdate`.** No `os.Rename` of `os.Executable()` outside this package.
4. **Forbidden inside this package:**
   - `os.Symlink` (use `internal/symlink/` if ever needed — but it should not be needed here)
   - `exec.Command("git", ...)` (git lives in `internal/repo/` only)
   - `panic` outside `*_test.go`
   - `http://` literals (HTTPS only for GitHub API and asset downloads)
   - Reading `GITHUB_TOKEN` from env (this is a public-release flow; no auth)

## FILES

(placeholder — filled in T18)

```
internal/updater/
├── updater.go       # Updater, CheckLatest, Apply
├── types.go         # cache schema, lock schema, LoadCache/SaveCache
├── cleanup.go       # CleanupOrphans (stale .old binaries)
├── errors.go        # ErrUpToDate, ErrLocked, sentinel errors
└── *_test.go
```
