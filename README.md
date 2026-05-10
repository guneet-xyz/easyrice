# easyrice

A cross-platform Go CLI for installing, switching, and tracking dotfile packages across machines.

`easyrice` is extracted from [guneet-xyz/rice](https://github.com/guneet-xyz/rice), a personal dotfile manager. It's the same engine, packaged as a standalone tool you can point at any dotfile repo.

## What it does

One binary that replaces GNU `stow` plus ad-hoc `install.sh` scripts:

- Symlink-based installs into `$HOME` (or any target)
- Per-package `rice.toml` manifests with **profiles**, so one repo serves many machines
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

## Quick start

Point `easyrice` at any dotfile repo that uses the `rice.toml` schema:

```sh
# Install a package using a named profile
easyrice install ghostty --profile macbook --repo ~/code/dotfiles

# Check what's installed
easyrice status

# Switch profiles later
easyrice switch ghostty --profile devstick -y
```

The `rice` alias works identically:

```sh
rice install nvim --profile work --repo ~/code/dotfiles
```

## Commands

| Command     | Purpose                                          |
|-------------|--------------------------------------------------|
| `install`   | Install a package under a profile                |
| `uninstall` | Remove all links recorded in state               |
| `switch`    | Uninstall current profile, install a new one     |
| `status`    | Show installed packages, profiles, drift         |
| `doctor`    | Detect broken links and missing sources          |

Examples:

```sh
easyrice install nvim --profile macbook --repo ~/dotfiles
easyrice uninstall nvim -y
easyrice switch zsh --profile work --log-level info
easyrice status
easyrice doctor
```

## Persistent flags

| Flag           | Default                          | Purpose                                          |
|----------------|----------------------------------|--------------------------------------------------|
| `--profile`    | (required for `install`)         | Which profile to install or switch to            |
| `--repo`       | `.`                              | Path to the dotfile repo                         |
| `--state`      | `~/.config/easyrice/state.json`  | Path to state.json (Windows: `%APPDATA%/easyrice/`) |
| `--log-level`  | `warn`                           | `debug` / `info` / `warn` / `error` / `critical` |
| `--yes`, `-y`  | `false`                          | Skip interactive confirmation prompts            |

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

`easyrice` expects a repo with one directory per package, each containing a `rice.toml` manifest. A minimal example:

```
dotfiles/
├── ghostty/
│   ├── rice.toml
│   ├── common/
│   │   └── config
│   └── macbook/
│       └── config
└── nvim/
    ├── rice.toml
    └── .config/nvim/
        └── init.lua
```

A minimal `rice.toml`:

```toml
schema_version = 1
name = "ghostty"
supported_os = ["linux", "darwin"]

[profiles.common]
sources = [{path = "common", mode = "file", target = "$HOME"}]

[profiles.macbook]
sources = [
  {path = "common",  mode = "file", target = "$HOME"},
  {path = "macbook", mode = "file", target = "$HOME"},
]
```

Source modes:

- `file`: walk the source dir and symlink each file individually under `target`. Multiple sources overlay.
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
