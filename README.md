# easyrice

A cross-platform Go CLI for installing, switching, and tracking dotfile packages across machines.

`easyrice` is extracted from [guneet-xyz/rice](https://github.com/guneet-xyz/rice), a personal dotfile manager. It's the same engine, packaged as a standalone tool.

## What it does

One binary that replaces GNU `stow` plus ad-hoc `install.sh` scripts:

- A single `rice.toml` at your dotfile repo root declares every package
- Symlink-based installs into `$HOME` (or any target)
- **Profiles** so one repo serves many machines
- Cross-platform: `linux`, `darwin`, `windows`
- Exact, safe uninstall via a JSON state file
- Health checks for broken links and drift

## Installation

```sh
git clone https://github.com/guneet-xyz/easyrice.git
cd easyrice
make install
```

This compiles the binary to `$(go env GOPATH)/bin/easyrice` and creates a `rice` symlink alongside it (handy if you've got the muscle memory). Make sure `$(go env GOPATH)/bin` is on your `PATH`.

To build locally without installing:

```sh
make build      # produces ./easyrice
```

## Getting started

`easyrice` clones your dotfile repo to a fixed location (`~/.config/easyrice/repos/default/` on Linux/macOS, `%APPDATA%/easyrice/repos/default/` on Windows) and reads a single `rice.toml` from the repo root.

```sh
# 1. Clone your dotfile repo (one-time setup)
rice init https://github.com/you/dotfiles.git

# 2. Install a package using a named profile
rice install ghostty --profile macbook

# 3. Pull the latest changes from the remote later
rice update
```

The `easyrice` binary works identically:

```sh
easyrice install nvim --profile work
```

## Commands

| Command     | Purpose                                                  |
|-------------|----------------------------------------------------------|
| `init`      | Clone a dotfile repo into the fixed managed location     |
| `update`    | `git pull` the managed repo                              |
| `install`   | Install a package under a profile                        |
| `uninstall` | Remove all links recorded in state                       |
| `switch`    | Uninstall current profile, install a new one             |
| `status`    | Show installed packages, profiles, drift                 |
| `doctor`    | Detect broken links and missing sources                  |

Examples:

```sh
rice init https://github.com/you/dotfiles.git
rice install nvim --profile macbook
rice uninstall nvim -y
rice switch zsh --profile work --log-level info
rice update
rice status
rice doctor
```

## Persistent flags

| Flag           | Default                          | Purpose                                          |
|----------------|----------------------------------|--------------------------------------------------|
| `--profile`    | (required for `install`)         | Which profile to install or switch to            |
| `--state`      | `~/.config/easyrice/state.json`  | Path to state.json (Windows: `%APPDATA%/easyrice/`) |
| `--log-level`  | `warn`                           | `debug` / `info` / `warn` / `error` / `critical` |
| `--yes`, `-y`  | `false`                          | Skip interactive confirmation prompts            |

There is no `--repo` flag. The repo path is fixed at `~/.config/easyrice/repos/default/` and managed by `rice init` / `rice update`.

## Environment

| Variable              | Effect                                                    |
|-----------------------|-----------------------------------------------------------|
| `EASYRICE_LOG_LEVEL`  | Sets the default log level (`--log-level` flag wins).     |

## State file

`easyrice` tracks every symlink it creates so uninstalls are exact and safe.

- POSIX (`linux`, `darwin`): `~/.config/easyrice/state.json`
- Windows: `%APPDATA%/easyrice/state.json`

Override with `--state /path/to/state.json` on any command. Don't hand-edit it; use the CLI.

## Dotfile repo layout

A single `rice.toml` lives at the repo root and declares every package. Each package has its own subdirectory (named after the package by default, or whatever you set in `root`). A minimal example:

```
dotfiles/
├── rice.toml
├── ghostty/
│   ├── common/
│   │   └── config
│   └── macbook/
│       └── config
└── nvim-custom/
    └── config/
        └── init.lua
```

A minimal `rice.toml`:

```toml
schema_version = 1

[packages.ghostty]
description = "Ghostty terminal emulator"
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
root = "nvim-custom"   # optional; defaults to package name ("nvim")

[packages.nvim.profiles.default]
sources = [{path = "config", mode = "folder", target = "$HOME/.config/nvim"}]
```

Source modes:

- `file`: walk the source dir and symlink each file individually under `target`. Multiple sources overlay (last wins).
- `folder`: symlink the entire source dir as one unit. Not overlayable.

For the full schema, profile composition rules, and OS gating, see [`AGENTS.md`](./AGENTS.md).

## Development

```sh
make test       # go test -race -count=1 ./...
make vet
make fmt
make build
```

The module path is `github.com/guneet-xyz/easyrice`. All implementation lives under `internal/`; the binary in `cli/` is the only consumer.

## License

See [`LICENSE`](./LICENSE).
