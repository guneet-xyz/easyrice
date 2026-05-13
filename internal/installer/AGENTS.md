# internal/installer/

Plan/execute orchestrator. Owns overlay rules, folder-mode invariants, conflict semantics, and converge (install-as-converge) flow. Largest internal package.

## STRUCTURE

```
internal/installer/
├── install.go      # InstallRequest → BuildInstallPlan → ExecuteInstallPlan → Install (one-shot)
├── uninstall.go    # UninstallRequest → BuildUninstallPlan → ExecuteUninstallPlan → Uninstall
├── converge.go     # ConvergeRequest → BuildConvergePlan → ExecuteConvergePlan; ConvergeAllRequest → ConvergeAll
├── conflict.go     # DetectConflicts() + PlannedLink, Conflict types
├── deps.go         # dependency resolution helpers
└── *_test.go       # heavy fixture-driven; helpers: fixtureRepo, copyDir, newRequest
```

`switch.go` has been DELETED. The standalone `Switch` / `BuildSwitchPlan` / `ExecuteSwitchPlan` API no longer exists. Profile changes flow through `BuildConvergePlan` with outcome `OutcomeProfileSwitched`.

## CONTRACT (every Build/Execute pair)

- `Build*Plan` is **read-only on the filesystem** (stat, walk, readlink). Returns a `*plan.Plan`. On conflicts: returns plan AND error so callers can render details.
- `Execute*Plan` mutates: creates/removes symlinks, writes state.json. MUST be idempotent for already-correct symlinks (`IsSymlinkTo` returns true → skip).
- `Install`/`Uninstall` (one-shot) = `Build` then `Execute`.

## CONVERGE (`converge.go`)

`Converge*` is the single entry point used by `rice install [pkg]`. It collapses install / profile-switch / repair / no-op into one path, deciding the outcome from current state plus the requested profile.

- `ConvergeRequest{RepoRoot, PackageName, RequestedProfile, CurrentOS, HomeDir, StatePath, Pkg, Manifest}` - inputs for a single-package converge.
- `ConvergeResult{PackageName, Outcome, OldProfile, NewProfile, Plan, LinksAfter}` - what happened. `Outcome` is one of:
  - `OutcomeNoOp` - package already installed at the requested profile and every link is correct.
  - `OutcomeInstalled` - package is not yet in state; plan installs it fresh.
  - `OutcomeProfileSwitched` - package is installed under a different profile; plan uninstalls old, installs new (with `ignoreTargets` so old links don't trigger conflicts against new ones).
  - `OutcomeRepaired` - profile matches but links have drifted; plan uninstalls then re-installs the same profile.
- `BuildConvergePlan(req)` is read-only on the FS. Resolves profile → specs (which transparently follows `import` for remote rices), computes a plan, runs `DetectConflicts` for fresh installs and switch transitions, returns the result. Pre-flight conflicts → returns nil plan and a wrapped error.
- `ExecuteConvergePlan(req, cr)` mutates. `OutcomeNoOp` is a no-op. `OutcomeProfileSwitched` runs uninstall first, then install. `OutcomeRepaired` also uninstalls (best-effort) then re-installs. After install, `cr.LinksAfter` is populated.
- `ConvergeAll(req ConvergeAllRequest)` iterates every package in the manifest, builds + executes a converge for each, accumulates errors with `errors.Join`, and returns all results even if some packages fail. `RequestedProfile` is the optional default profile; per-package, the stored profile wins when the request is empty.

## OVERLAY & FOLDER-MODE RULES (THE non-obvious bits)

- **File-mode**: walk each source dir, build planned links, **last source wins** on identical relative path (overlay). Skip `rice.toml` and any symlinks inside source trees.
- **Folder-mode**: emit ONE op for the entire dir, do NOT walk in. Folder ops do **NOT** participate in file-mode last-wins overlay.
- Mixing folder-mode + any other source touching the same target subtree = error (caught at plan time).
- `withinHome(target, home)` rejects any target escaping `$HOME` (defense in depth). NEVER bypass.
- `target` is `os.ExpandEnv`-expanded before use.
- Cross-remote `import = "remotes/<name>#<pkg>.<profile>"` is resolved by `internal/profile.ResolveSpecs` BEFORE the install plan is built; the installer sees a flat `[]SourceSpec` and does not need to know about remotes.

## CONFLICT SEMANTICS (`conflict.go`)

| Target state | Outcome |
|--------------|---------|
| Doesn't exist | not a conflict |
| Exists, not a symlink (file or dir) | conflict: "existing file" / "existing directory" |
| Exists, symlink → some other path | conflict: "symlink points to <other>" |
| Exists, symlink → expected source | NOT a conflict (idempotent) |
| Stat fails (perm, etc.) | conflict with stat error message |

`DetectConflicts(planned, ignoreTargets)` - `ignoreTargets` lets `BuildConvergePlan` (for `OutcomeProfileSwitched`) exclude old links being removed in the same operation.

## TESTING CONVENTIONS

- `fixtureRepo(t, name)` copies `testdata/install/<name>` into `t.TempDir()` for isolation.
- `newRequest(t, ...)` factory for `InstallRequest` with temp `HomeDir` + `StatePath`.
- All helpers MUST call `t.Helper()`.
- NEVER write to real `$HOME` - always `t.TempDir()`.
- This package NEVER calls `git`. Tests must not require a git binary.

## ANTI-PATTERNS

- DO NOT call `os.Symlink` / `os.Remove` directly - go through `internal/symlink`.
- DO NOT mutate filesystem inside `Build*Plan` (read-only contract).
- DO NOT remove the `withinHome` check from install.
- DO NOT treat folder-mode ops as overlay candidates.
- DO NOT reintroduce `Switch` / `BuildSwitchPlan` / `ExecuteSwitchPlan` - it was deleted by design. All profile transitions go through `BuildConvergePlan` (`OutcomeProfileSwitched`).
- DO NOT auto-commit from `Install` / `Uninstall` / `Converge*`. The installer NEVER touches git. Only `cli/remote.go` issues commits, via `repo.CommitPaths`.
- DO NOT call `exec.Command("git", ...)` from this package - it stays git-free.
