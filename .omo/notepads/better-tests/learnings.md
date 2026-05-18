## [2026-05-18] Task 2: goldenfs snapshot helper

### Implementation Summary

Created `internal/testutil/goldenfs/` package with three core functions:

1. **`Snapshot(t *testing.T, root string) string`**
   - Walks directory tree using `filepath.WalkDir`
   - Returns deterministic text representation: relative paths + types (file/dir/symlink)
   - Symlink targets normalized: relative if inside root, `<TEMP>` placeholder if outside
   - Lexical sort by relative path ensures determinism across runs

2. **`AssertGolden(t interface{...}, got string, goldenPath string)`**
   - Compares snapshot to golden file
   - Accepts interface with `Helper()`, `Fatalf()`, `Errorf()` for testability
   - `-update` flag support: rewrites golden files when flag is set
   - Clear error message: "run with -update to create" for missing files

3. **`SnapshotState(t *testing.T, statePath string) string`**
   - Loads state via `internal/state.Load()`
   - Returns stable JSON via `json.MarshalIndent()` with 2-space indent
   - Handles non-existent state files (returns empty state `{}`)

### Key Design Decisions

- **Interface-based testing**: `AssertGolden` accepts interface instead of `*testing.T` to enable mock testing
- **Symlink normalization**: Simple approach - any absolute path outside root becomes `<TEMP>` placeholder
- **Flag registration**: Used `flag.Bool("update", false, ...)` at package level for `-update` flag
- **Determinism**: Explicit `sort.Strings()` on all entries before joining

### Test Coverage

8 tests implemented:
- `TestSnapshot_EmptyDir` - empty directory returns empty string
- `TestSnapshot_WithFiles` - files listed in lexical order
- `TestSnapshot_WithSubdirs` - directories marked with trailing `/`
- `TestSnapshot_WithInternalSymlink` - symlinks inside root shown as relative paths
- `TestSnapshot_WithExternalSymlink` - symlinks outside root shown as `<TEMP>`
- `TestSnapshot_Deterministic` - 3 consecutive snapshots are identical
- `TestAssertGolden_MissingFile` - missing golden file fails with hint message
- `TestUpdate_RewritesGolden` - `-update` flag rewrites golden files
- `TestSnapshotState_EmptyState` - empty state file returns `{}`

### Evidence Captured

- `.omo/evidence/task-2-determinism.log` - 3 runs of `TestSnapshot_Deterministic` all PASS
- `.omo/evidence/task-2-missing-golden.log` - `TestAssertGolden_MissingFile` PASS (verifies error handling)
- `.omo/evidence/task-2-update.log` - `TestUpdate_RewritesGolden` PASS with `-update` flag

### Lessons Learned

1. **Testing.TB interface**: Go's testing package doesn't export a complete interface; had to define custom interface with required methods
2. **Flag handling**: `flag.Bool()` works at package level; tests can check `*updateFlag` directly
3. **Symlink normalization**: Simple `<TEMP>` placeholder is sufficient; no need to preserve path structure for external symlinks
4. **Determinism**: Must sort all collections before output; even `filepath.WalkDir` order is not guaranteed across runs

### Blocks Unblocked

Task 18 (E2E scenarios) can now use `goldenfs` for `$HOME` + state.json assertions.

## [2026-05-18] Task 4: multiremote submodule builder

### Implementation Summary

Created `internal/testutil/multiremote/` package with a builder pattern for multi-remote test fixtures:

1. **`Multi` builder** - fluent API for constructing fixtures:
   - `New(t *testing.T) *Multi` - create builder
   - `.AddRemote(name, manifestContent) *Multi` - add remote with manifest
   - `.AddRemoteRaw(name, files) *Multi` - add remote with raw files
   - `.WithParentManifest(content) *Multi` - set parent repo manifest
   - `.WithCircularImport(...ImportCycle) *Multi` - add circular import relationships
   - `.WithUninitSubmodule(name) *Multi` - mark remote to be uninitialized
   - `.Build() *MultiFixture` - create fixture

2. **`MultiFixture` result** - exposes:
   - `ParentRepoPath string` - path to parent repo
   - `RemotePaths map[string]string` - name → remote repo path
   - `Cleanup func()` - cleanup function

3. **`ImportCycle` struct** - represents circular imports:
   - `From string` - remote that imports
   - `To string` - remote being imported

### Key Design Decisions

- **file:// URLs only**: All remotes are local temp dirs with `file://` URLs (no network)
- **Git protocol config**: Used `-c protocol.file.allow=always` flag on `git submodule add` to enable local file:// cloning (git 2.42+ blocks this by default for security)
- **Separate remote repos**: Each remote is its own git repo initialized with `.gitkeep` + initial commit
- **Submodule path convention**: All submodules added under `remotes/<name>` in parent repo
- **Circular import handling**: Rewrites remote manifests to add import directives; helper itself does NOT resolve (that's Task 10's job)
- **Uninitialized submodules**: Uses `git submodule deinit -f` to mark submodules as uninitialized (produces `-` prefix in `git submodule status`)
- **Conditional commits**: Only commits when there are staged changes (avoids "nothing to commit" errors)

### Test Coverage

7 tests implemented:
- `TestMulti_Empty` - zero remotes, parent repo only
- `TestMulti_SingleRemote` - one remote, verifies submodule listing
- `TestMulti_ThreeRemotes` - three remotes simultaneously
- `TestMulti_CircularImport` - A→B→A circular imports in manifests
- `TestMulti_UninitSubmodule` - submodule marked as uninitialized (deinit'd)
- `TestMulti_InvalidManifestInRemote` - remote with broken TOML
- `TestMulti_AddRemoteRaw` - raw files instead of manifest

### Evidence Captured

- `.omo/evidence/task-4-three-remotes.log` - `TestMulti_ThreeRemotes` PASS
- `.omo/evidence/task-4-circular.log` - `TestMulti_CircularImport` PASS
- `.omo/evidence/task-4-uninit.log` - `TestMulti_UninitSubmodule` PASS

### Lessons Learned

1. **Git file:// protocol security**: Modern git (2.42+) blocks `file://` URLs by default. Must use `-c protocol.file.allow=always` flag on submodule add.
2. **Submodule deinit recipe**: `git submodule deinit -f <path>` marks submodule as uninitialized without removing `.gitmodules` entry. Produces `-` prefix in `git submodule status`.
3. **Import spec format**: `remotes/<name>#<pkg>.<profile>` is the canonical format (see `internal/manifest/import_spec.go`). Helper appends new packages with imports rather than modifying existing ones.
4. **Conditional commits**: Must check `git status --porcelain` before committing to avoid "nothing to commit" errors when rewriting manifests.
5. **Test helper exception**: This package is a documented exception to AGENTS.md anti-pattern (no raw git in production). Test helpers are permitted to use `exec.Command("git", ...)` with clear documentation.

### Blocks Unblocked

Task 10 (profile circular imports) can now use `multiremote` to build aggressive test scenarios.
Task 14 (multi-remote integration) can use this builder for integration tests.

## [2026-05-18T17:24:07Z] Task 6: updater interface refactor

- **Final LOC**: production net diff = 49 (added=37, removed=12), within ≤50 budget.
- **Behavior preservation**:
  - flock contention semantics unchanged: `flockLocker.Acquire` delegates to the existing `acquireLock` which retains `O_CREATE|O_EXCL|O_WRONLY` create + 1h staleLockAge reclaim + single retry. The error-returning closure is a no-op wrapper around the original `func()` releaser.
  - go-selfupdate config wiring (ChecksumValidator, Source factory, Prerelease=false) is untouched in `fetch.go`. The `u.sourceFactory` test seam survives.
  - Cache `time.Now()` replaced exactly once (`cache.go:87`) — no other clock reads exist in the package.
- **Subtle tradeoff**: `Options.Fetcher` does NOT auto-populate a non-nil default in `New`. The "default fetcher" is the implicit GitHub code path inside `FetchLatest` that fires when `u.fetcher == nil`. Wiring a concrete wrapper would either need a self-reference back to `*Updater` (circular) or duplicate the github logic outside FetchLatest — both bust the LOC budget. The current shape preserves the existing nil-fallback seam that fetch tests depend on.
- **Test impact**: zero existing tests modified. Added one new test (`TestNew_ZeroOptions`, 15 LOC, excluded from production budget) verifying `Clock`/`Locker` defaults wire up and `Clock.Now()` returns non-zero.
- **All updater tests still green** under `-race -count=1`.

## [2026-05-18T23:00] Task 10: profile cycles

### Bug results (FAIL = real bug demonstrated; PASS = production matches contract)

- **FAIL** BUG-040 A→B→A cycle: production emits raw cache key `import cycle detected: <repoRoot>|<remote>|<pkg>|<profile>` instead of arrow path `a -> b -> a`. Each wrapping layer adds `package %q profile %q import: remote %q: ...` so the chain DOES name the hops, but the final terminus is a path-with-pipe, not an arrow path.
- **FAIL** BUG-041 A→B→C→A 3-hop cycle: same root cause as BUG-040; error chain shows all three hops via wrapping but cycle terminator is still the cache key.
- **PASS** BUG-042 self-import via remote: cycle correctly detected.
- **PASS** BUG-043 same-name profile cycle: cycle correctly detected.
- **FAIL** BUG-044 missing remote hint: production error reads `remote is not initialized; run 'rice remote update <name>'`. Hint points at wrong command (`update` requires existing submodule). Spec mandates `rice remote add`. The remote name "ghost" appears AFTER the hint as part of the `%w: %s` wrap, not interpolated into the hint.
- **PASS** BUG-045 missing package: production emits `package %q not found in remote rice %q` — names both.
- **PASS** BUG-046 missing profile: error chain `... import: remote "kick": profile "ghostprofile" not defined in package "realpkg"` — names all three.
- **PASS** BUG-047 space in spec: rejected because `#` separator missing.
- **PASS** BUG-048 empty parts: production emits `remote name must not be empty` / `package name must not be empty` / `profile name must not be empty` — names which part.
- **PASS** BUG-049 overlay ordering: `ResolveSpecs` appends imported specs first, local sources last — matches AGENTS.md file-mode last-wins.

### Exact cycle-error format production currently emits

For `top.start -> a.p.x -> b.p.x -> a.p.x`, the message is:

```
package "top" profile "start" import: remote "a": package "p" profile "x" import: remote "b": package "p" profile "x" import: remote "a": import cycle detected: <repoRoot>|a|p|x
```

The fix path (future Fix task): change `internal/profile/profile.go:33-35` to build an ordered slice of `remote.pkg.profile` triples (or just remote names) as visited keys advance, then render as `a -> b -> a` when cycle hits. Currently `visited` is `map[string]bool` (unordered) keyed by the cache string.

### Gate caveat

`go test -race -count=1 ./internal/profile/... 2>&1 | grep -v 'BUG-' | grep -q FAIL` exits 1 because Go's `--- FAIL: TestProfile_CircularImports` parent-test summary and the package-level `FAIL` / `FAIL\tpkg\t...s` lines do NOT contain `BUG-`. Same precedent in `internal/manifest/validate_bugs_test.go` (Task 8). The individual subtest FAIL lines are correctly tagged and skippable. The gate as written cannot pass when ANY bug test fails — accepted across the project as a known pattern issue.

### Multi-remote helper integration

`multiremote.New(t).AddRemote(name, fullToml).Build()` is sufficient. Bypassed `WithCircularImport` because it auto-injects `imported_<name>` packages with hardcoded import specs — too coarse for these tests. Wrote full remote rice.tomls inline via `remoteWithImport`/`remoteWithSources` helpers.

The multiremote builder DOES execute `git submodule add file://...` per remote, so the on-disk layout is realistic (the parent has `remotes/<name>/rice.toml` as a real submodule checkout). `ResolveSpecs` only reads via `repo.RemoteTomlPath(parentRoot, name)`, so this works.


## [2026-05-18T17:34:10Z] Task 11: concurrency — confirmed real production races

Real bugs surfaced by `go test -race -count=5 -run TestInstaller_Concurrency`:

- **BUG-063 (deterministic, every run)**: `state.Save` is *Load → in-memory mutate → WriteFile*; no synchronization. 8 disjoint installs reliably reduce to ~1 entry. The "last-writer-wins" pattern is exactly the predicted BUG-007 territory.
- **BUG-065 (deterministic, every run)**: `os.WriteFile` truncates THEN writes; a concurrent `state.Load` sees an empty file and json.Unmarshal returns "unexpected end of JSON input". `BuildConvergePlan` is supposed to be read-only-safe but cannot survive an in-flight writer.
- **BUG-061 (intermittent)**: under contention `os.WriteFile`'s two-step truncate-then-write can leave bytes from a previous payload past the new payload's end → "invalid character 'C' after top-level value" (a torn write, not just last-writer-wins). This is more severe than BUG-063 because the FILE itself is invalid JSON, not just missing entries.
- **BUG-060 (intermittent)**: ConvergeAll iterates the manifest's `map[string]PackageDef`; map iteration order is non-deterministic, so the goroutine that wins state.Save varies — sometimes losing a package even though it was installed.
- **BUG-062, BUG-064**: did not reproduce in this run. State payload here is small (<2KB) so `os.WriteFile` fits in a single syscall on Linux ext4; on slower disks or larger states these will surface. Kept as guards.

Synchronization pattern that worked: `chan struct{} start barrier + sync.WaitGroup`. No sleeps. Goroutines block on `<-start` then `close(start)` releases all at once. Determinism over `-count=5` is excellent for BUG-063/065; the others are 30-80% per run.

No race detector hits in test code — only production races, as intended.

The fix (out of scope for tests-only T11) is straightforward: `state.Save` must (a) write to a temp file in the same directory, (b) `os.Rename` to the final path (atomic on POSIX), (c) hold a file lock around Load+mutate+Save sequences. The temp+rename pattern alone fixes BUG-061/065 immediately; the lock fixes BUG-060/063.
