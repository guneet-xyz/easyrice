# AGENTS.md

Guide for AI agents working on `easyrice`. Conventions, schema, architecture.

**Generated:** 2026-05-11 | **Commit:** aba7c0c | **Branch:** main

## OVERVIEW

`easyrice` is a cross-platform Go CLI dotfile manager. Reads per-package `rice.toml` manifests, resolves a **profile**, composes **sources**, installs files into `$HOME` via symlinks, tracks every link in a JSON **state file** for exact uninstall.

Module: `github.com/guneet-xyz/easyrice` | Go 1.26.2 | Deps: cobra, BurntSushi/toml, zap, testify.

## STRUCTURE

```
easyrice/
├── cli/             # cobra commands, package main (see cli/AGENTS.md)
├── internal/
│   ├── manifest/    # rice.toml parsing/validation/OS gating (see internal/manifest/AGENTS.md)
│   ├── profile/     # ResolveSpecs(): profile name → []SourceSpec
│   ├── plan/        # pure data types: Op, Conflict, Plan
│   ├── installer/   # plan→execute for install/uninstall/switch (see internal/installer/AGENTS.md)
│   ├── symlink/     # low-level FS ops (see internal/symlink/AGENTS.md)
│   ├── state/       # state.json read/write
│   ├── logger/      # zap tee: console + file (debug always to file)
│   ├── doctor/      # health checks (legacy state detection)
│   └── prompt/      # RenderPlan, RenderSwitchPlan, RenderConflicts, Confirm
├── testdata/        # fixtures (testdata/manifest, testdata/manifest_valid, testdata/install)
├── Makefile         # build / install / test / vet / fmt / clean
├── go.mod
└── AGENTS.md        # this file
```

## WHERE TO LOOK

| Task | File |
|------|------|
| Add a CLI command | `cli/<name>.go` + register in `init()` via `rootCmd.AddCommand` |
| Change persistent flags | `cli/root.go` |
| Change rice.toml schema | `internal/manifest/schema.go` + `internal/manifest/validate.go` |
| Change install/overlay/folder-mode logic | `internal/installer/install.go` (BuildInstallPlan) |
| Change conflict semantics | `internal/installer/conflict.go` (DetectConflicts) |
| Change uninstall behavior | `internal/installer/uninstall.go` |
| Change switch atomicity | `internal/installer/switch.go` |
| Change state.json shape | `internal/state/state.go` (`InstalledLink`, `PackageState`, `State`) |
| Change log levels / output | `internal/logger/logger.go` |
| Add health check | `internal/doctor/` |
| Change prompt rendering | `internal/prompt/prompt.go` |

## CODE MAP

| Symbol | Type | Location | Role |
|--------|------|----------|------|
| `Manifest`, `ProfileDef`, `SourceSpec` | struct | `internal/manifest/schema.go` | rice.toml schema |
| `manifest.Load`, `manifest.Discover` | func | `internal/manifest/load.go` | Parse one package / scan repo |
| `manifest.Validate`, `manifest.CheckOS` | func | `internal/manifest/{validate,osgating}.go` | Schema + OS gate |
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

Dependency direction: `cli/` → `installer/` → {`manifest`, `profile`, `plan`, `symlink`, `state`, `logger`}. Never the reverse. `prompt`, `doctor`, `logger` are leaf packages.

## rice.toml Schema

Every dotfile package has `rice.toml` at its root. Schema in `internal/manifest/schema.go`.

```toml
schema_version = 1
name = "ghostty"
description = "Ghostty terminal emulator configuration"
supported_os = ["linux", "darwin"]

[profiles.common]
sources = [{path = "common", mode = "file", target = "$HOME"}]

[profiles.macbook]
sources = [
  {path = "common",  mode = "file", target = "$HOME"},
  {path = "macbook", mode = "file", target = "$HOME"},
]
```

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `schema_version` | int | yes | Currently `1`. Bump only on breaking changes. |
| `name` | string | yes | Should match the directory name. |
| `description` | string | no | Short human description. |
| `supported_os` | []string | yes | Package-level OS gate. Values: `linux`, `darwin`, `windows`. |
| `profile_key` | string | no | Reserved for future per-package overrides. |
| `profiles.<name>.sources` | []table | yes | Inline table form ONLY. Each: `path`, `mode`, `target` (all required). |

`SourceSpec.UnmarshalTOML` rejects non-table forms - DO NOT accept bare strings.

### Source modes

- **`file`** (overlayable): walk source dir, symlink each file under `target`. Later sources override earlier ones (last-wins) on identical relative paths.
- **`folder`** (single symlink, NOT overlayable): symlink the entire source dir as one unit to `target`. Cannot be combined with another source touching the same target subtree.

`target` supports `os.ExpandEnv` (e.g. `"$HOME"`, `"$HOME/.config/nvim"`).

### Profile conventions

`common`, `macbook`, `devstick`, `personal`, `work`. Compose by listing `common` first, then overlay:

```toml
[profiles.workmac]
sources = [
  {path = "common",  mode = "file", target = "$HOME"},
  {path = "macbook", mode = "file", target = "$HOME"},
  {path = "work",    mode = "file", target = "$HOME"},
]
```

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

State is the source of truth for `uninstall` and `switch`. Symlinks are absolute, pointing back into the dotfile repo.

## CLI Surface

Persistent flags (defined in `cli/root.go`):

| Flag | Default | Purpose |
|------|---------|---------|
| `--repo` | `.` | Path to dotfile repo |
| `--state` | `state.DefaultPath()` | Path to state.json |
| `--log-level` | `warn` | `debug`/`info`/`warn`/`error`/`critical` |
| `--yes`, `-y` | `false` | Skip confirmation prompts |

Env: `EASYRICE_LOG_LEVEL` sets log level. Flag wins over env.

```sh
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
- Symlinks are **absolute**, pointing into the dotfile repo.
- All FS ops MUST go through `internal/symlink` - never `os.Symlink` directly outside that package.
- Logger is global (`logger.L`); call `logger.Init()` once in `PersistentPreRunE`.
- Tests: `t.TempDir()` for isolation, table-driven default, fixtures under `testdata/`.

## ANTI-PATTERNS (forbidden in this repo)

- Writing into real `$HOME` from tests - use `t.TempDir()` + `--state` to a temp file.
- Hand-editing `state.json` - use the CLI.
- Adding a new top-level dir before extending an existing `internal/` package.
- Calling `os.Symlink` / `os.Readlink` outside `internal/symlink/`.
- Accepting non-table forms for `sources` entries (e.g. bare string paths).
- Bypassing `withinHome()` defense-in-depth check in installer.
- Adding `golangci-lint` config or CI workflows without explicit request - intentionally minimal tooling.

## TESTING

- ALWAYS `go test -race ./...` (Makefile already enforces).
- Table-driven default: `for _, tc := range cases { t.Run(tc.name, ...) }`.
- Fixtures mirror real package layout under `testdata/<scenario>/<pkg>/rice.toml`. `testdata/` is ignored by Go toolchain.
- Pass `--state` to a temp file in tests; never the real default path.
- Use `t.Helper()` on test helpers; `t.TempDir()` auto-cleans.

## NOTES / GOTCHAS

- `withinHome(target, home)` enforces install targets stay inside `$HOME` (defense in depth in `internal/installer/install.go`).
- `rice.toml` files inside source trees are skipped during walk - by design.
- Symlinks inside source trees are skipped during walk - we only manage real files.
- `folder` mode op does NOT participate in file-mode last-wins overlay.
- Windows `os.Symlink` requires Developer Mode or admin - runtime check belongs in `doctor/`, NOT in `symlink/`.
- `logger.L` is a `zap.NewNop()` until `Init()` is called - safe to call before init, just silent.
- File logger ALWAYS at DebugLevel regardless of console level (tee).
