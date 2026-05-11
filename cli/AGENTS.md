# cli/

`package main` - cobra CLI surface. One file per command, plus `root.go` (persistent flags + logger init) and `main.go` (3-line entry).

## STRUCTURE

```
cli/
├── main.go               # func main() { Execute() }
├── root.go               # rootCmd + persistent flags + PersistentPreRunE (logger.Init)
├── init.go               # easyrice init <git-url>  (clone repo into managed location)
├── update.go             # easyrice update          (git pull on managed repo)
├── install.go            # easyrice install <pkg> --profile <name>
├── uninstall.go          # easyrice uninstall <pkg>
├── switch.go             # easyrice switch <pkg> --profile <name>
├── status.go             # easyrice status [pkg]
├── doctor.go             # easyrice doctor
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
| Bump CLI version string | `cli/root.go` `const Version` |

## CONVENTIONS

- Each command file owns: `var <name>Cmd`, command-local flags, `init()`, `run<Name>(cmd, args)`.
- Commands MUST delegate work to `internal/installer` or `internal/repo` (or other internal pkgs). NO business logic in `cli/`.
- Read persistent flag values via the package-level vars in `root.go` (`flagState`, `flagYes`). There is NO `flagRepo` - the repo path is fixed via `repo.DefaultRepoPath()`.
- For interactive y/n: call `prompt.Confirm()`; respect `flagYes` to bypass.
- Render plans via `prompt.RenderPlan` / `RenderSwitchPlan` / `RenderConflicts` BEFORE executing.
- Errors from `runX` propagate to cobra → exit 1 via `Execute()` in `root.go`.
- `install`/`uninstall`/`switch`/`status` MUST check `repo.Exists(repo.DefaultRepoPath())` and return `repo.ErrRepoNotInitialized` if missing.

## ANTI-PATTERNS

- DO NOT call `os.Exit` inside command bodies - return error, let `Execute()` handle it.
- DO NOT touch the filesystem here beyond what `installer`/`repo`/`state` exposes.
- DO NOT initialize the logger anywhere except `PersistentPreRunE` in `root.go`.
- DO NOT add commands without a `_test.go` file alongside.
- DO NOT reintroduce a `--repo` flag - the managed-repo path is fixed.
- DO NOT call `exec.Command("git", ...)` - all git ops go through `internal/repo`.

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
