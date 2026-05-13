# cli/

`package main` - cobra CLI surface. One file per command, plus `root.go` (persistent flags + logger init) and `main.go` (3-line entry).

## STRUCTURE

```
cli/
├── main.go               # func main() { Execute() }
├── root.go               # rootCmd + persistent flags + PersistentPreRunE (logger.Init)
├── init.go               # easyrice init <git-url>  (clone repo into managed location)
├── update.go             # easyrice update          (git pull on managed repo)
├── remote.go             # easyrice remote {add,remove,update,list}  (manage submodules under remotes/)
├── install.go            # easyrice install [pkg] [--profile <name>]  (converge: install/switch/repair/no-op)
├── uninstall.go          # easyrice uninstall <pkg>
├── status.go             # easyrice status [pkg]
├── doctor.go             # easyrice doctor
├── upgrade.go            # easyrice upgrade         (self-update binary; see upgrade section below)
├── reminder.go           # shared post-command update reminder helper
├── version.go            # easyrice version  (uses const Version in root.go)
└── *_test.go             # one per command file
```

## WHERE TO LOOK

| Task | File |
|------|------|
| Add new command | new `<name>.go`, define `var <name>Cmd = &cobra.Command{...}`, register in `init()` via `rootCmd.AddCommand(<name>Cmd)` |
| Change persistent flag default | `cli/root.go` `init()` |
| Change persistent pre-run (logger setup) | `cli/root.go` `PersistentPreRunE` |
| Change clone behavior | `cli/init.go` (delegates to `internal/repo.Clone`) |
| Change pull behavior | `cli/update.go` (delegates to `internal/repo.Pull`) |
| Change `rice remote ...` behavior | `cli/remote.go` (delegates to `internal/repo` `SubmoduleAdd/Remove/Update/List`) |
| Change converge (install) semantics | `cli/install.go` (delegates to `internal/installer.BuildConvergePlan` / `ConvergeAll`) |
| Bump CLI version string | `cli/root.go` `const Version` |

## CONVENTIONS

- Each command file owns: `var <name>Cmd`, command-local flags, `init()`, `run<Name>(cmd, args)`.
- Commands MUST delegate work to `internal/installer` or `internal/repo` (or other internal pkgs). NO business logic in `cli/`.
- Read persistent flag values via the package-level vars in `root.go` (`flagState`, `flagYes`). There is NO `flagRepo` - the repo path is fixed via `repo.DefaultRepoPath()`.
- For interactive y/n: call `prompt.Confirm()`; respect `flagYes` to bypass.
- Render plans via `prompt.RenderPlan` / `RenderConflicts` BEFORE executing.
- Errors from `runX` propagate to cobra → exit 1 via `Execute()` in `root.go`.
- `install`/`uninstall`/`status`/`remote` MUST check `repo.Exists(repo.DefaultRepoPath())` and return `repo.ErrRepoNotInitialized` if missing.
- `remote add`/`remote remove` MUST also check `repo.IsClean(ctx, repoRoot)` and return `repo.ErrRepoDirty` if uncommitted changes exist. `install`/`status`/`doctor` only WARN on dirty trees, never block.
- `install`/`uninstall` MUST NOT auto-commit. Only `remote add`/`remote remove` commit on the user's behalf, via `repo.CommitPaths(ctx, repoRoot, paths, msg)` (scope only).

## ANTI-PATTERNS

- DO NOT call `os.Exit` inside command bodies - return error, let `Execute()` handle it.
- DO NOT touch the filesystem here beyond what `installer`/`repo`/`state` exposes.
- DO NOT initialize the logger anywhere except `PersistentPreRunE` in `root.go`.
- DO NOT add commands without a `_test.go` file alongside.
- DO NOT reintroduce a `--repo` flag - the managed-repo path is fixed.
- DO NOT call `exec.Command("git", ...)` - all git ops go through `internal/repo`.
- DO NOT reintroduce a `switch` command - it was deleted by design. `install <pkg> --profile <name>` already handles install, profile change, repair, and no-op.
- DO NOT auto-commit from `install` / `uninstall` / `update`. Only `remote add` / `remote remove` commit, and they MUST scope `git add` to specific paths via `repo.CommitPaths`. NEVER `git add -A` / `git add .`.

## install (converge)

`easyrice install [package] [--profile <name>]` is converge-shaped:

- Zero args: converge every package declared in `rice.toml` (`installer.ConvergeAll`).
- One arg: converge that package (`installer.BuildConvergePlan` → `ExecuteConvergePlan`).
- For each package the outcome is one of: **installed** (not yet present), **profile-switched** (profile changed), **repaired** (links drifted), **no-op** (already correct).
- The deleted `rice switch` command is fully covered by `install <pkg> --profile <name>` - DO NOT reintroduce it.

## remote

`easyrice remote {add,remove,update,list}` manages git submodules under `remotes/<name>/` in the managed repo, used by profile-level `import = "remotes/<name>#<pkg>.<profile>"` references.

- `remote add <url> --name <name>` adds a submodule + auto-commits `.gitmodules` and `remotes/<name>` via `repo.CommitPaths`. Refuses if the working tree is dirty (`ErrRepoDirty`) or the name is taken (`ErrRemoteAlreadyExists`).
- `remote remove <name>` deinit/removes the submodule and auto-commits the result. Refuses if any profile in `rice.toml` still imports from this remote (`ErrRemoteInUse`).
- `remote update [name]` runs `git submodule update --remote` for the named remote (or all when omitted).
- `remote list` prints submodule name, path, SHA, and state.
- `--name` is validated against `^[a-zA-Z0-9_-]+$`.

## upgrade

`easyrice upgrade` self-updates the `easyrice` binary from the latest GitHub release. Delegates to `internal/updater`.

- Command name is `upgrade` (NOT `update`). `update` pulls the dotfile repo; `upgrade` replaces the binary. Do not confuse them, do not alias them.
- File: `cli/upgrade.go` (+ `cli/upgrade_test.go`). Shared reminder helper lives in `cli/reminder.go`.
- MUST delegate all network I/O, file I/O, and binary-swap to `internal/updater`. No HTTP calls in `cli/`.

### Flags

| Flag | Default | Behavior |
|------|---------|----------|
| `--check` | `false` | Performs a live API call to fetch the latest release, prints `current → latest` (or "up to date"), and exits 0. NEVER mutates the cache, NEVER applies the swap. |

### Behavior

- **Dev-build refusal:** if `updater.IsDevBuild(Version)` returns true, prints a message pointing the user at `go install` or the GitHub releases page and returns `updater.ErrDevBuild`. The binary is NEVER swapped in dev builds.
- **Up-to-date path:** when `FetchLatest` returns `ErrAlreadyLatest`, or when `IsNewer(Version, release.Version)` returns false, prints `easyrice is up to date (<version>)` and returns nil (exit 0).
- **Confirmation:** unless `--yes`/`-y` is set, prompts `Upgrade easyrice from <current> to <latest>?`. Cancellation prints `cancelled` and returns nil.
- **Success message:** on a successful swap, prints `Upgraded easyrice to <version>. Restart easyrice to use the new version.` and exits 0.
- **No auto-restart (decision):** the new binary is intentionally NOT re-execed. The user re-runs `easyrice` themselves. Reasoning: re-execing a freshly-swapped binary inside the same process invites argv/stdio/cwd surprises and complicates testing. The cost of one extra keystroke is worth the simplicity.
