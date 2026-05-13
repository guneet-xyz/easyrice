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

### One-liner

**Linux / macOS:**

```sh
curl -fsSL https://raw.githubusercontent.com/guneet-xyz/easyrice/main/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/guneet-xyz/easyrice/main/install.ps1 | iex
```

Downloads the latest prebuilt binary for your OS/architecture from [GitHub Releases](https://github.com/guneet-xyz/easyrice/releases), verifies its SHA-256 checksum, installs to `~/.local/bin` (Linux/macOS) or `%LOCALAPPDATA%\Programs\easyrice` (Windows), creates a `rice` symlink, and optionally adds the install directory to your PATH. No Go toolchain required.

Supported platforms: Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64, arm64).

To install a specific version or to a custom directory:

```sh
curl -fsSL https://raw.githubusercontent.com/guneet-xyz/easyrice/main/install.sh | \
  EASYRICE_VERSION=v0.2.0 EASYRICE_INSTALL_DIR=/usr/local/bin sh
```

```powershell
$env:EASYRICE_VERSION = 'v0.2.0'; irm https://raw.githubusercontent.com/guneet-xyz/easyrice/main/install.ps1 | iex
```

### Manual download

Grab the appropriate binary from [Releases](https://github.com/guneet-xyz/easyrice/releases/latest), verify it against `checksums.txt`, then move it onto your PATH:

```sh
chmod +x easyrice-v0.2.0-linux-amd64
mv easyrice-v0.2.0-linux-amd64 ~/.local/bin/easyrice
ln -sf ~/.local/bin/easyrice ~/.local/bin/rice
```

### From source

Requires [Go 1.21+](https://go.dev/dl).

```sh
git clone https://github.com/guneet-xyz/easyrice.git
cd easyrice
make install
```

This compiles the binary to `$(go env GOPATH)/bin/easyrice` and creates a `rice` symlink alongside it. Make sure `$(go env GOPATH)/bin` is on your `PATH`.

> **macOS note**: On first run, macOS may block the binary as "unidentified developer". Run `xattr -d com.apple.quarantine ~/.local/bin/easyrice` to allow it. Binaries are currently unsigned.

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

| Command | Purpose |
|---|---|
| `init` | Clone a dotfile repo into the fixed managed location |
| `update` | `git pull` the managed repo |
| `install` | Converge one package, or every package when omitted |
| `uninstall` | Remove all links recorded in state for a package |
| `status` | Show repo state, installed packages, drift, dependencies, and remotes |
| `doctor` | Run health checks |
| `remote` | Manage remote rices as git submodules |
| `upgrade` | Upgrade the easyrice binary |
| `version` | Print the current version |

Examples:

```sh
rice init https://github.com/you/dotfiles.git
rice install nvim --profile macbook
rice uninstall nvim -y
rice install zsh --profile work --log-level info
rice update
rice status
rice doctor
```

## Persistent flags

| Flag | Default | Purpose |
|---|---|---|
| `--state` | `~/.config/easyrice/state.json` | Path to state.json (Windows: `%APPDATA%/easyrice/`) |
| `--log-level` | `warn` | `debug` / `info` / `warn` / `error` / `critical` |
| `--yes`, `-y` | `false` | Skip interactive confirmation prompts |
| `--no-update-check` | `false` | Skip update reminder checks |

`install` also accepts `--profile <name>` and `--skip-deps`. If `--profile` is omitted, easyrice uses the package's stored profile, then a hostname-matching profile, then `default`, then the sole profile if only one is declared.

There is no `--repo` flag. The repo path is fixed at `~/.config/easyrice/repos/default/` and managed by `rice init` / `rice update`.

## Environment

| Variable              | Effect                                                    |
|-----------------------|-----------------------------------------------------------|
| `EASYRICE_LOG_LEVEL`  | Sets the default log level (`--log-level` flag wins).     |
| `EASYRICE_NO_UPDATE_CHECK=1` | Skips update reminder checks.                    |

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

For the full manifest schema, see [`docs/rice-toml.md`](./docs/rice-toml.md). For command workflows, see [`docs/usage.md`](./docs/usage.md).

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
