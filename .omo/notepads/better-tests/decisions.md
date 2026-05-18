
## [2026-05-18T17:19:05Z] Task 7: catalog convention

Locked the BUG-NNN template at `.omo/known-bugs.md`. Tasks 8-18 must append entries using this template verbatim:

```
## BUG-NNN — short title
**Status**: failing | passing | wont-fix
**Severity**: S1 | S2 | S3
**Package**: internal/<pkg>
**Test**: <path/to/test.go>:TestFunctionName
**Spec source**: <citation>
**Expected**: <intended behavior>
**Actual**: <current behavior>
**Repro**: <minimal commands or test seed>
**How we know test is correct**: <pointer or argument>
```

Block reservations (zero-padded 3-digit NNN):
- 001-019: state
- 020-039: manifest
- 040-059: profile
- 060-079: installer (incl. concurrency)
- 080-099: updater
- 100-119: multi-remote
- 120-139: symlink/path
- 140-159: git
- 160-179: CLI/UX
- 180-199: E2E scenarios

Severity: S1=data loss, S2=feature broken, S3=UX wart. Status: failing | passing | wont-fix.

Catalog freshness is enforced by the bash one-liner under `## Verification` in the catalog file. Every test marker `BUG-NNN` must have a matching `## BUG-NNN` header in the catalog.

Placeholder `BUG-000` is present and must be removed before Task 18 is marked complete.

## [2026-05-18T17:23:53Z] Task 6: updater interfaces

- Defined exported interfaces in `internal/updater/types.go` (alongside Options/Release/CheckResult), not a new file — keeps the package surface compact.
  - `ReleaseFetcher` — `FetchLatest(ctx) (*Release, error)`
  - `Clock` — `Now() time.Time`
  - `Locker` — `Acquire(path) (release func() error, err error)`
- Default wiring in `New(opts)`:
  - `opts.Clock` defaults to `realClock{}` (calls `time.Now()`).
  - `opts.Locker` defaults to `flockLocker{}`, a thin adapter around the existing unexported `acquireLock(cacheDir)` whose returned closure is wrapped to satisfy `func() error` (acquireLock returns `func()`).
  - `opts.Fetcher` is **not** assigned a non-nil default; instead the existing dispatch in `FetchLatest` (`if u.fetcher != nil ... else fetchLatestFromGitHub`) is preserved. This keeps behavior identical to pre-refactor and avoids breaking the fetch tests that explicitly assign `u.fetcher = nil` to exercise the go-selfupdate code path via `u.sourceFactory`.
  - `opts.Fetcher` is still copied into the existing unexported `u.fetcher` field at construction time, so external callers can inject a `ReleaseFetcher` via the public Options field.
- Internal unexported field `fetcher` retypes from `releaseFetcher` to `ReleaseFetcher` (the exported alias) so existing test assignments (`u.fetcher = fakeFetcher`) keep compiling.
- `cache.go` swaps the single `time.Now()` call for `u.opts.Clock.Now()`.
- `swap.go` swaps `acquireLock(u.opts.CacheDir)` for `u.opts.Locker.Acquire(u.opts.CacheDir)` and wraps the deferred release in `func() { _ = releaseLock() }` to discard the new error return.
