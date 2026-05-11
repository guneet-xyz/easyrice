# internal/installer/

Plan/execute orchestrator. Owns overlay rules, folder-mode invariants, conflict semantics, switch atomicity. Largest internal package.

## STRUCTURE

```
internal/installer/
├── install.go      # InstallRequest → BuildInstallPlan → ExecuteInstallPlan → Install (one-shot)
├── uninstall.go    # UninstallRequest → BuildUninstallPlan → ExecuteUninstallPlan → Uninstall
├── switch.go       # SwitchRequest → BuildSwitchPlan (uninstall+install) → ExecuteSwitchPlan → Switch
├── conflict.go     # DetectConflicts() + PlannedLink, Conflict types
└── *_test.go       # heavy fixture-driven; helpers: fixtureRepo, copyDir, newRequest, switchSetup
```

## CONTRACT (every Build/Execute pair)

- `Build*Plan` is **read-only on the filesystem** (stat, walk, readlink). Returns a `*plan.Plan`. On conflicts: returns plan AND error so callers can render details.
- `Execute*Plan` mutates: creates/removes symlinks, writes state.json. MUST be idempotent for already-correct symlinks (`IsSymlinkTo` returns true → skip).
- `Install`/`Uninstall`/`Switch` (one-shot) = `Build` then `Execute`.

## OVERLAY & FOLDER-MODE RULES (THE non-obvious bits)

- **File-mode**: walk each source dir, build planned links, **last source wins** on identical relative path (overlay). Skip `rice.toml` and any symlinks inside source trees.
- **Folder-mode**: emit ONE op for the entire dir, do NOT walk in. Folder ops do **NOT** participate in file-mode last-wins overlay.
- Mixing folder-mode + any other source touching the same target subtree = error (caught at plan time).
- `withinHome(target, home)` rejects any target escaping `$HOME` (defense in depth). NEVER bypass.
- `target` is `os.ExpandEnv`-expanded before use.

## CONFLICT SEMANTICS (`conflict.go`)

| Target state | Outcome |
|--------------|---------|
| Doesn't exist | not a conflict |
| Exists, not a symlink (file or dir) | conflict: "existing file" / "existing directory" |
| Exists, symlink → some other path | conflict: "symlink points to <other>" |
| Exists, symlink → expected source | NOT a conflict (idempotent) |
| Stat fails (perm, etc.) | conflict with stat error message |

`DetectConflicts(planned, ignoreTargets)` - `ignoreTargets` lets `BuildSwitchPlan` exclude old links being removed in the same operation.

## SWITCH ATOMICITY

`BuildSwitchPlan` produces a `SwitchPlan{Uninstall, Install}` pair. `ExecuteSwitchPlan` runs uninstall first, then install. If install fails after uninstall succeeds, state reflects partial - this is intentional (state.json is post-uninstall consistent). DO NOT add transactional rollback without discussion.

## TESTING CONVENTIONS

- `fixtureRepo(t, name)` copies `testdata/install/<name>` into `t.TempDir()` for isolation.
- `newRequest(t, ...)` factory for `InstallRequest` with temp `HomeDir` + `StatePath`.
- `switchSetup(t, ...)` chains an initial install, returns ready-to-switch state.
- All helpers MUST call `t.Helper()`.
- NEVER write to real `$HOME` - always `t.TempDir()`.

## ANTI-PATTERNS

- DO NOT call `os.Symlink` / `os.Remove` directly - go through `internal/symlink`.
- DO NOT mutate filesystem inside `Build*Plan` (read-only contract).
- DO NOT remove the `withinHome` check from install.
- DO NOT add rollback logic to `ExecuteSwitchPlan` without explicit discussion (see Switch Atomicity).
- DO NOT treat folder-mode ops as overlay candidates.
