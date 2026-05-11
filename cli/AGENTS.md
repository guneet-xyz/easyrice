# cli/

`package main` - cobra CLI surface. One file per command, plus `root.go` (persistent flags + logger init) and `main.go` (3-line entry).

## STRUCTURE

```
cli/
├── main.go               # func main() { Execute() }
├── root.go               # rootCmd + persistent flags + PersistentPreRunE (logger.Init)
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
| Bump CLI version string | `cli/root.go` `const Version` |

## CONVENTIONS

- Each command file owns: `var <name>Cmd`, command-local flags, `init()`, `run<Name>(cmd, args)`.
- Commands MUST delegate work to `internal/installer` (or other internal pkgs). NO business logic in `cli/`.
- Read persistent flag values via the package-level vars in `root.go` (`flagRepo`, `flagState`, `flagYes`). Don't redeclare.
- For interactive y/n: call `prompt.Confirm()`; respect `flagYes` to bypass.
- Render plans via `prompt.RenderPlan` / `RenderSwitchPlan` / `RenderConflicts` BEFORE executing.
- Errors from `runX` propagate to cobra → exit 1 via `Execute()` in `root.go`.

## ANTI-PATTERNS

- DO NOT call `os.Exit` inside command bodies - return error, let `Execute()` handle it.
- DO NOT touch the filesystem here beyond what `installer`/`state` exposes.
- DO NOT initialize the logger anywhere except `PersistentPreRunE` in `root.go`.
- DO NOT add commands without a `_test.go` file alongside.
