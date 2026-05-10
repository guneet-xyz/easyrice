# AGENTS.md

Guide for AI agents working in this repo. Covers conventions, schemas, and workflows for `easyrice`.

## Project Overview

`easyrice` is a cross-platform Go CLI dotfile manager. It replaces ad-hoc setups (GNU `stow` plus shell scripts) with a single binary that:

- Reads per-package `rice.toml` manifests
- Resolves the active **profile** (e.g. `macbook`, `devstick`)
- Composes file **sources** into a flat tree
- Installs files into `$HOME` (or any target) via symlinks
- Tracks every link in a JSON **state file** so uninstall is exact and safe

If you are adding a feature, prefer extending an existing `internal/` package over adding a new top-level dir.

## Repo Structure

```
easyrice/
├── cli/                 # CLI entrypoint and cobra commands (package main)
│   └── main.go, root.go, install.go, switch.go, uninstall.go, status.go, doctor.go, version.go
├── internal/
│   ├── manifest/        # rice.toml parsing + schema
│   ├── profile/         # profile resolution + source composition
│   ├── plan/            # planned link operations (dry-run model)
│   ├── installer/       # apply install/uninstall plans
│   ├── symlink/         # low-level symlink ops (create, verify, remove)
│   ├── state/           # state.json read/write
│   ├── logger/          # zap-backed leveled logger
│   ├── doctor/          # health checks (broken links, drift)
│   └── prompt/          # interactive yes/no confirmation
├── testdata/            # fixtures for tests (mirrors real package layouts)
├── go.mod
└── AGENTS.md            # you are here
```

Go module path: `github.com/guneet-xyz/easyrice`.

## rice.toml Schema

Every dotfile package directory consumed by `easyrice` has a `rice.toml` at its root. Schema lives in `internal/manifest/schema.go`.

```toml
schema_version = 1
name = "ghostty"
description = "Ghostty terminal emulator configuration"
supported_os = ["linux", "darwin"]

[profiles.common]
sources = [{path = "common", mode = "file", target = "$HOME"}]

[profiles.macbook]
sources = [
  {path = "common", mode = "file", target = "$HOME"},
  {path = "macbook", mode = "file", target = "$HOME"},
]
```

### Fields

| Field             | Type                | Required | Notes                                                            |
|-------------------|---------------------|----------|------------------------------------------------------------------|
| `schema_version`  | int                 | yes      | Currently `1`. Bump only on breaking schema changes.             |
| `name`            | string              | yes      | Package name. Should match the directory name.                   |
| `description`     | string              | no       | Short human-readable description.                                |
| `supported_os`    | []string            | yes      | OS gate at package level. Values: `linux`, `darwin`, `windows`.  |
| `profile_key`     | string              | no       | Reserved for future per-package profile overrides.               |
| `profiles.<name>` | table               | yes      | One or more profiles. At least `common` is conventional.         |
| `profiles.<name>.sources` | []table | yes      | List of source tables. Each entry requires `path` (relative subdir), `mode` (`"file"` or `"folder"`), and `target` (absolute destination root, env vars expanded). |

`sources` are relative to the package directory. Each source table specifies how files are installed.

## Source Spec

Each source entry in the `sources` list is a table with three required fields:

- **`path`**: Relative path to the source directory within the package (e.g., `"common"`, `".config/nvim"`).
- **`mode`**: Installation mode:
  - `"file"`: Walk the source directory and symlink each file individually under `target`. Files from multiple sources are overlaid.
  - `"folder"`: Symlink the entire source directory as a single unit to `target`. Cannot be overlaid by other sources in the same profile.
- **`target`**: Absolute destination root where files are installed. Supports environment variable expansion (e.g., `"$HOME"`, `"$HOME/.config"`).

### File-mode example (default, overlayable):

```toml
[profiles.common]
sources = [
  {path = "common", mode = "file", target = "$HOME"},
  {path = "macbook", mode = "file", target = "$HOME"},
]
```

This installs files from `common/` and `macbook/` into `$HOME`, with `macbook/` overlaying `common/`.

### Folder-mode example (single symlink, not overlayable):

```toml
[profiles.common]
sources = [{path = ".config/nvim", mode = "folder", target = "$HOME/.config/nvim"}]
```

This symlinks `<repo>/nvim/.config/nvim` as a single unit to `$HOME/.config/nvim`. Folder-mode sources are ideal for tools like nvim and opencode that manage their entire config directory.

## Profile Conventions

Standard profile names (use these unless you have a strong reason not to):

- `common`   — shared baseline used by every machine
- `macbook`  — personal MacBook overlay
- `devstick` — Linux dev box / portable USB rig
- `personal` — personal-only tweaks (cross-machine)
- `work`     — work-only tweaks

Profiles compose by listing sources. To make a new machine variant, add a new profile that lists `common` first, then your overlay:

```toml
[profiles.workmac]
sources = [
  {path = "common", mode = "file", target = "$HOME"},
  {path = "macbook", mode = "file", target = "$HOME"},
  {path = "work", mode = "file", target = "$HOME"},
]
```

## OS Gating

Two layers:

1. **Package level** via `supported_os`. If the current OS is not in the list, the package is skipped entirely (with a warning).
2. **Profile level** via `os` on a profile (reserved field, see schema). Currently profiles inherit from the package-level gate.

Valid OS values: `linux`, `darwin`, `windows`. Detected via Go's `runtime.GOOS`.

## State File

Location (resolved by `state.DefaultPath()`):

- POSIX (`linux`, `darwin`): `~/.config/easyrice/state.json`
- Windows: `%APPDATA%/easyrice/state.json`

Override with `--state /path/to/state.json` on any command.

Format: a JSON object keyed by package name.

```json
{
  "ghostty": {
    "profile": "macbook",
    "installed_links": [
      {
        "source": "/Users/me/code/rice/ghostty/common/config",
        "target": "/Users/me/.config/ghostty/config"
      }
    ],
    "installed_at": "2025-05-10T12:34:56Z"
  }
}
```

The state file is the source of truth for `uninstall` and `switch`. Never hand-edit it; use the CLI.

## CLI Commands

All commands accept the persistent flags below.

### Persistent flags

| Flag           | Default                | Purpose                                          |
|----------------|------------------------|--------------------------------------------------|
| `--repo`       | `.`                    | Path to the dotfile repo                         |
| `--state`      | `state.DefaultPath()`  | Path to state.json                               |
| `--log-level`  | `warn`                 | `debug` / `info` / `warn` / `error` / `critical` |
| `--yes`, `-y`  | `false`                | Skip interactive confirmation prompts            |

Env var: `EASYRICE_LOG_LEVEL` sets the log level. The `--log-level` flag wins over the env var.

### Commands

```sh
easyrice install <package> --profile <name>   # install a package under a profile
easyrice uninstall <package>                  # remove all links recorded in state
easyrice switch <package> --profile <name>    # uninstall current profile, install new
easyrice status                               # show installed packages, profiles, drift
easyrice doctor                               # detect broken links, missing sources
```

Examples:

```sh
easyrice install ghostty --profile macbook --repo ~/code/rice
easyrice switch nvim --profile work -y
easyrice status --log-level info
EASYRICE_LOG_LEVEL=debug easyrice doctor
```

## Logging

Five levels, in order of verbosity:

`debug` < `info` < `warn` (default) < `error` < `critical`

Set with `--log-level` or `EASYRICE_LOG_LEVEL`. Logs are written via `internal/logger` (zap). A persistent log file lives at `logger.DefaultLogPath()` (typically `~/.config/easyrice/logs/easyrice.log` on POSIX).

## Testing Conventions

- **Always run with the race detector**: `go test -race ./...`
- **Table-driven tests** are the default style. One `for _, tc := range cases` loop per behavior.
- **Fixtures** live under `testdata/`. Mirror the real package layout (`testdata/install/mypkg/rice.toml`, etc.). `testdata/` is ignored by the Go toolchain, so it's safe for arbitrary files.
- **Temp dirs**: use `t.TempDir()`. Never write into the real `$HOME` from tests.
- **State paths in tests**: pass `--state` explicitly to a temp file. Same for `--repo`.

Run a single package's tests:

```sh
go test -race ./internal/installer/...
```

## Conventions Summary

- Go module: `github.com/guneet-xyz/easyrice`
- Go version: see `go.mod`
- All exported types live under `internal/` (the binary is the only consumer)
- Errors wrap with `fmt.Errorf("context: %w", err)`
- No `panic` outside `main`; return errors
- Symlinks are absolute, pointing back into the dotfile repo
