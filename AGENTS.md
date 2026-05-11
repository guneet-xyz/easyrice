# AGENTS.md

Guide for AI agents working on `easyrice`. Conventions, schema, architecture.

**Generated:** 2026-05-11 | **Branch:** main

## OVERVIEW

`easyrice` is a cross-platform Go CLI dotfile manager. Reads a single root `rice.toml` manifest from a managed repo, resolves a **profile** for the requested package, composes **sources**, installs files into `$HOME` via symlinks, and tracks every link in a JSON **state file** for exact uninstall.

The dotfile repo is cloned to a fixed path (`~/.config/easyrice/repos/default/` on Linux/macOS, `%APPDATA%/easyrice/repos/default/` on Windows) by `rice init <url>` and refreshed by `rice update`. There is no `--repo` flag.

Module: `github.com/guneet-xyz/easyrice` | Go 1.26.2 | Deps: cobra, BurntSushi/toml, zap, testify.

## STRUCTURE

```
easyrice/
├── cli/             # cobra commands, package main (see cli/AGENTS.md)
├── internal/
│   ├── repo/        # managed repo lifecycle: DefaultRepoPath, Clone, Pull, Exists (see internal/repo/AGENTS.md)
│   ├── manifest/    # single-file rice.toml parsing/validation/OS gating (see internal/manifest/AGENTS.md)
│   ├── profile/     # ResolveSpecs(): profile name → []SourceSpec
│   ├── plan/        # pure data types: Op, Conflict, Plan
│   ├── installer/   # plan→execute for install/uninstall/switch (see internal/installer/AGENTS.md)
│   ├── symlink/     # low-level FS ops (see internal/symlink/AGENTS.md)
│   ├── state/       # state.json read/write
│   ├── logger/      # zap tee: console + file (debug always to file)
│   ├── doctor/      # health checks (legacy state detection)
│   ├── updater/     # self-update: GitHub release polling + binary swap (see internal/updater/AGENTS.md)
│   └── prompt/      # RenderPlan, RenderSwitchPlan, RenderConflicts, Confirm
├── testdata/        # fixtures (testdata/manifest_valid_v2, testdata/manifest_invalid_v2, testdata/install_v2)
├── Makefile         # build / install / test / vet / fmt / clean
├── go.mod
└── AGENTS.md        # this file
```

## WHERE TO LOOK

| Task | File |
|------|------|
| Add a CLI command | `cli/<name>.go` + register in `init()` via `rootCmd.AddCommand` |
| Change persistent flags | `cli/root.go` |
| Change `rice init` clone behavior | `cli/init.go` + `internal/repo/repo.go` (`Clone`) |
| Change `rice update` pull behavior | `cli/update.go` + `internal/repo/repo.go` (`Pull`) |
| Change managed repo location | `internal/repo/repo.go` (`DefaultRepoPath`) |
| Change rice.toml schema | `internal/manifest/schema.go` + `internal/manifest/validate.go` |
| Change manifest loading | `internal/manifest/load.go` (`LoadFile`) |
| Change install/overlay/folder-mode logic | `internal/installer/install.go` (BuildInstallPlan) |
| Change conflict semantics | `internal/installer/conflict.go` (DetectConflicts) |
| Change uninstall behavior | `internal/installer/uninstall.go` |
| Change switch atomicity | `internal/installer/switch.go` |
| Change state.json shape | `internal/state/state.go` (`InstalledLink`, `PackageState`, `State`) |
| Change log levels / output | `internal/logger/logger.go` |
| Add health check | `internal/doctor/` |
| Change self-update behavior | `internal/updater/` (+ `cli/upgrade.go`) |
| Change prompt rendering | `internal/prompt/prompt.go` |

## CODE MAP

| Symbol | Type | Location | Role |
|--------|------|----------|------|
| `Manifest`, `PackageDef`, `ProfileDef`, `SourceSpec` | struct | `internal/manifest/schema.go` | Single-file rice.toml schema |
| `manifest.LoadFile` | func | `internal/manifest/load.go` | Parse + validate the root `rice.toml` |
| `manifest.Validate`, `manifest.CheckOS` | func | `internal/manifest/{validate,osgating}.go` | Schema + OS gate |
| `repo.DefaultRepoPath`, `repo.RepoTomlPath` | func | `internal/repo/repo.go` | Fixed managed-repo paths |
| `repo.Exists`, `repo.Clone`, `repo.Pull`, `repo.GitOnPath` | func | `internal/repo/repo.go` | Lifecycle ops on the managed repo |
| `repo.ErrRepoNotInitialized`, `repo.ErrPackageNotDeclared` | var/func | `internal/repo/errors.go` | Sentinel errors for CLI mapping |
| `profile.ResolveSpecs` | func | `internal/profile/profile.go` | profile name → ordered `[]SourceSpec` |
| `plan.Op`, `plan.Plan`, `plan.Conflict` | struct | `internal/plan/plan.go` | Dry-run model (data only) |
| `installer.Install` / `Uninstall` / `Switch` | func | `internal/installer/` | High-level orchestration |
| `installer.BuildInstallPlan` / `ExecuteInstallPlan` | func | `internal/installer/install.go` | Plan/execute split |
| `installer.DetectConflicts` | func | `internal/installer/conflict.go` | Idempotent conflict check |
| `symlink.{Create,Remove,IsSymlinkTo,ReadLink}` | func | `internal/symlink/symlink.go` | FS primitives |
| `state.{Load,Save,DefaultPath}` | func | `internal/state/state.go` | state.json I/O |
| `state.State` (= `map[string]PackageState`) | type | `internal/state/state.go` | Source of truth for uninstall |
| `logger.{Init,L,Sync,ParseLevel,Debug,Info,Warn,Error,Critical}` | func/var | `internal/logger/logger.go` | Global zap logger `L` |
| `prompt.{RenderPlan,RenderSwitchPlan,RenderConflicts,Confirm}` | func | `internal/prompt/prompt.go` | TTY rendering + y/n |
| `doctor.CheckLegacyState` | func | `internal/doctor/legacy_state.go` | Drift detection |
| `updater.Updater`, `updater.New` | struct/func | `internal/updater/updater.go` | Self-update boundary; constructor validates `Owner`/`Repo` and sets defaults |
| `updater.Options`, `updater.Release`, `updater.CheckResult` | struct | `internal/updater/types.go` | Configuration and result types for the updater |
| `updater.DefaultCacheDir` | func | `internal/updater/types.go` | Returns `~/.config/easyrice` (POSIX) / `%APPDATA%/easyrice` (Windows) |
| `updater.IsDevBuild`, `updater.IsNewer`, `updater.IsPreRelease` | func | `internal/updater/version.go` | Semver helpers (normalize + `golang.org/x/mod/semver`) |
| `(*Updater).FetchLatest` | method | `internal/updater/fetch.go` | Anonymous GitHub release lookup via `go-selfupdate`; rejects pre-releases and assets without checksums.txt |
| `(*Updater).Apply` | method | `internal/updater/swap.go` | Atomic binary swap; resolves symlinks, holds upgrade lock, never re-execs |
| `(*Updater).CheckCached` | method | `internal/updater/cache.go` | 24h-TTL cached check; fail-silent on network errors |
| `updater.FormatReminder`, `updater.ShouldShowReminder`, `updater.IsTerminal` | func | `internal/updater/reminder.go` | Pure helpers for the post-command update reminder |
| `updater.CleanupOrphanArtifacts` | func | `internal/updater/cleanup.go` | Removes `.new`/`.old` siblings left by interrupted swaps |
| `updater.ErrDevBuild`, `updater.ErrAlreadyLatest`, `updater.ErrLockBusy`, `updater.ErrNoChecksum`, `updater.ErrCacheCorrupt`, `updater.ErrInvalidSemver` | var | `internal/updater/errors.go` | Sentinel errors for CLI mapping |

Dependency direction: `cli/` → {`installer/`, `repo/`} → {`manifest`, `profile`, `plan`, `symlink`, `state`, `logger`}. Never the reverse. `prompt`, `doctor`, `logger` are leaf packages.

## rice.toml Schema

A single `rice.toml` lives at the **root** of the managed dotfile repo and declares **every** package. Schema lives in `internal/manifest/schema.go`.

```toml
schema_version = 1

[packages.ghostty]
description = "Ghostty terminal emulator configuration"
supported_os = ["linux", "darwin"]

[packages.ghostty.profiles.common]
sources = [{path = "common", mode = "file", target = "$HOME/.config/ghostty"}]

[packages.ghostty.profiles.macbook]
sources = [
  {path = "common",  mode = "file", target = "$HOME/.config/ghostty"},
  {path = "macbook", mode = "file", target = "$HOME/.config/ghostty"},
]

[packages.nvim]
description = "Neovim configuration"
supported_os = ["linux", "darwin"]
root = "nvim-custom"   # optional; defaults to package name

[packages.nvim.profiles.default]
sources = [{path = "config", mode = "folder", target = "$HOME/.config/nvim"}]
```

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `schema_version` | int | yes | Currently `1`. Bump only on breaking changes. |
| `packages` | table | yes | Map of package name → `PackageDef`. Keyed by package name. |
| `packages.<name>.description` | string | no | Short human description. |
| `packages.<name>.supported_os` | []string | yes | Package-level OS gate. Values: `linux`, `darwin`, `windows`. |
| `packages.<name>.root` | string | no | Subdirectory holding the package files. Defaults to the package name. |
| `packages.<name>.profiles.<name>.sources` | []table | yes | Inline table form ONLY. Each: `path`, `mode`, `target` (all required). |

`SourceSpec.UnmarshalTOML` rejects non-table forms - DO NOT accept bare strings.

`path` is interpreted relative to the package root (`<repoRoot>/<root or name>/<path>`).

### Source modes

- **`file`** (overlayable): walk the source dir, symlink each file under `target`. Later sources override earlier ones (last-wins) on identical relative paths.
- **`folder`** (single symlink, NOT overlayable): symlink the entire source dir as one unit to `target`. Cannot be combined with another source touching the same target subtree.

`target` supports `os.ExpandEnv` (e.g. `"$HOME"`, `"$HOME/.config/nvim"`).

### Profile conventions

`common`, `macbook`, `devstick`, `personal`, `work`. Compose by listing `common` first, then overlay:

```toml
[packages.zsh.profiles.workmac]
sources = [
  {path = "common",  mode = "file", target = "$HOME"},
  {path = "macbook", mode = "file", target = "$HOME"},
  {path = "work",    mode = "file", target = "$HOME"},
]
```

## Managed Repo

| OS | Path |
|----|------|
| linux/darwin | `~/.config/easyrice/repos/default/` |
| windows | `%APPDATA%/easyrice/repos/default/` |

- `rice init <url>` clones the dotfile repo to that path. Errors if the directory already exists.
- `rice update` runs `git -C <path> pull`.
- All other commands (`install`, `uninstall`, `switch`, `status`) read `rice.toml` from that path. If the path doesn't exist, they return `repo.ErrRepoNotInitialized`.
- `git` MUST be on `PATH` for `init` and `update`. Check via `repo.GitOnPath()`.

## State File

| OS | Path |
|----|------|
| linux/darwin | `~/.config/easyrice/state.json` |
| windows | `%APPDATA%/easyrice/state.json` |

Override with `--state /path`. Format: JSON object keyed by package name (`map[string]PackageState`). NEVER hand-edit; use the CLI.

```json
{
  "ghostty": {
    "profile": "macbook",
    "installed_links": [
      { "source": "/abs/repo/ghostty/common/config", "target": "/abs/$HOME/.config/ghostty/config" }
    ],
    "installed_at": "2025-05-10T12:34:56Z"
  }
}
```

State is the source of truth for `uninstall` and `switch`. Symlinks are absolute, pointing back into the managed repo.

## CLI Surface

Persistent flags (defined in `cli/root.go`):

| Flag | Default | Purpose |
|------|---------|---------|
| `--state` | `state.DefaultPath()` | Path to state.json |
| `--log-level` | `warn` | `debug`/`info`/`warn`/`error`/`critical` |
| `--yes`, `-y` | `false` | Skip confirmation prompts |

There is **no `--repo` flag**. The repo path is fixed (`repo.DefaultRepoPath()`).

Env: `EASYRICE_LOG_LEVEL` sets log level. Flag wins over env.

```sh
easyrice init      <git-url>
easyrice update
easyrice install   <package> --profile <name>
easyrice uninstall <package>
easyrice switch    <package> --profile <name>
easyrice status    [package]
easyrice doctor
easyrice version
```

## COMMANDS

```sh
make build      # ./easyrice
make install    # $(GOPATH)/bin/easyrice + rice symlink
make test       # go test -race -count=1 ./...
make vet
make fmt
```

## CONVENTIONS

- Errors wrap with `fmt.Errorf("context: %w", err)` - always.
- No `panic` outside `main`; return errors.
- Exported types live under `internal/` (binary is the only consumer).
- Symlinks are **absolute**, pointing into the managed dotfile repo.
- All FS ops MUST go through `internal/symlink` - never `os.Symlink` directly outside that package.
- All `git` ops MUST go through `internal/repo` - never `exec.Command("git", ...)` outside that package.
- Logger is global (`logger.L`); call `logger.Init()` once in `PersistentPreRunE`.
- Tests: `t.TempDir()` for isolation, table-driven default, fixtures under `testdata/`.

## ANTI-PATTERNS (forbidden in this repo)

- Adding back the `--repo` flag - the managed-repo path is fixed by design.
- Calling `manifest.Discover` - it has been deleted. Use `manifest.LoadFile(repo.RepoTomlPath(repoRoot))`.
- Reading `rice.toml` from anywhere other than the managed repo root.
- Promising multi-repo support - one repo, one location, intentionally.
- Writing into real `$HOME` from tests - use `t.TempDir()` + `--state` to a temp file.
- Hand-editing `state.json` - use the CLI.
- Adding a new top-level dir before extending an existing `internal/` package.
- Calling `os.Symlink` / `os.Readlink` outside `internal/symlink/`.
- Calling `exec.Command("git", ...)` outside `internal/repo/`.
- Accepting non-table forms for `sources` entries (e.g. bare string paths).
- Bypassing `withinHome()` defense-in-depth check in installer.
- Adding `golangci-lint` config or CI workflows without explicit request - intentionally minimal tooling.
- Calling the GitHub API or `go-selfupdate` outside `internal/updater/` - the updater package owns ALL network I/O for releases.
- Reading or writing `update-check.json` / `upgrade.lock` outside `internal/updater/` - go through the exported API.
- Auto-restarting the binary after `upgrade` succeeds - we print a restart hint and exit; the user re-runs.

## TESTING

- ALWAYS `go test -race ./...` (Makefile already enforces).
- Table-driven default: `for _, tc := range cases { t.Run(tc.name, ...) }`.
- Fixtures live under `testdata/<scenario>/rice.toml` for v2 (single-file) layout. `testdata/` is ignored by Go toolchain.
- Pass `--state` to a temp file in tests; never the real default path.
- For tests that need a managed repo, point at a `t.TempDir()` (the `repo` package functions accept the path explicitly).
- Use `t.Helper()` on test helpers; `t.TempDir()` auto-cleans.

## NOTES / GOTCHAS

- `withinHome(target, home)` enforces install targets stay inside `$HOME` (defense in depth in `internal/installer/install.go`).
- Symlinks inside source trees are skipped during walk - we only manage real files.
- `folder` mode op does NOT participate in file-mode last-wins overlay.
- Windows `os.Symlink` requires Developer Mode or admin - runtime check belongs in `doctor/`, NOT in `symlink/`.
- `logger.L` is a `zap.NewNop()` until `Init()` is called - safe to call before init, just silent.
- File logger ALWAYS at DebugLevel regardless of console level (tee).
- `PackageDef.Root` defaults to the package name when empty; resolve this at the call site, not in the schema decoder.
