# `rice.toml` specification

`rice.toml` is the manifest at the root of your dotfile repository. It declares every package easyrice can install, which operating systems it supports, which profiles exist, and which files should be linked into `$HOME`.

```toml
schema_version = 1
```

## Repository layout

Each package stores its source files in a package root directory. By default, that directory has the same name as the package. Set `root` to use a different directory name.

```text
dotfiles/
├── rice.toml
├── ghostty/
│   ├── common/
│   │   └── config
│   └── macbook/
│       └── config
├── nvim-custom/
│   └── config/
│       └── init.lua
└── zsh/
    ├── common/
    │   └── .zshrc
    └── work/
        └── .zshrc
```

## Complete example

```toml
schema_version = 1

[custom_dependencies.starship]
version_probe = ["starship", "--version"]
version_regex = "starship (\\d+\\.\\d+\\.\\d+)"

[custom_dependencies.starship.install.darwin]
description = "Install starship with Homebrew"
shell_payload = "brew install starship"

[custom_dependencies.starship.install.linux]
description = "Install starship with the official installer"
shell_payload = "curl -sS https://starship.rs/install.sh | sh -s -- -y"

[packages.ghostty]
description = "Ghostty terminal configuration"
supported_os = ["linux", "darwin"]
dependencies = [
  {name = "starship"},
]

[packages.ghostty.profiles.default]
sources = [{path = "common", mode = "file", target = "$HOME/.config/ghostty"}]

[packages.ghostty.profiles.macbook]
sources = [
  {path = "common", mode = "file", target = "$HOME/.config/ghostty"},
  {path = "macbook", mode = "file", target = "$HOME/.config/ghostty"},
]

[packages.nvim]
description = "Neovim configuration"
supported_os = ["linux", "darwin"]
root = "nvim-custom"
dependencies = [
  {name = "neovim", version = ">=0.9.0"},
  {name = "ripgrep"},
]

[packages.nvim.profiles.default]
sources = [{path = "config", mode = "folder", target = "$HOME/.config/nvim"}]

[packages.zsh]
description = "Z shell configuration"
supported_os = ["linux", "darwin"]

[packages.zsh.profiles.work]
sources = [
  {path = "common", mode = "file", target = "$HOME"},
  {path = "work", mode = "file", target = "$HOME"},
]
```

## Top-level fields

| Field | Type | Required | Description |
|---|---:|:---:|---|
| `schema_version` | integer | yes | Must be `1`. |
| `packages` | table | yes | Map of package name to package definition. At least one package is required. |
| `custom_dependencies` | table | no | Custom dependency probes and install methods used by package `dependencies`. |

## Package fields

Defined under `[packages.<name>]`.

| Field | Type | Required | Description |
|---|---:|:---:|---|
| `description` | string | no | Human-readable package description. |
| `supported_os` | string array | yes | Any of `linux`, `darwin`, `windows`. |
| `root` | string | no | Package source directory relative to the repo root. Defaults to the package name. |
| `dependencies` | inline table array | no | Dependencies to check before install. Each entry is `{name = "...", version = "..."}`; `version` is optional. |
| `profiles` | table | yes | Map of profile name to install sources and/or an imported remote profile. |

Package names must be non-empty and must not contain slashes or whitespace. `root` must be relative and must not contain `..` segments.

## Profile fields

Defined under `[packages.<package>.profiles.<profile>]`.

| Field | Type | Required | Description |
|---|---:|:---:|---|
| `sources` | inline table array | if no `import` | Local source entries to install. |
| `import` | string | if no `sources` | Import sources from a remote rice profile using `remotes/<name>#<package>.<profile>`. |

A profile must define at least one local `source` or one `import`. If both are present, imported sources resolve first and local sources overlay after them.

Profile names are arbitrary non-empty strings. Common conventions are `default`, `common`, a machine hostname such as `macbook`, or context names such as `work` and `personal`.

## Source entries

Each source must use inline table form. Bare strings are invalid.

```toml
sources = [{path = "common", mode = "file", target = "$HOME/.config/ghostty"}]
```

| Field | Type | Required | Description |
|---|---:|:---:|---|
| `path` | string | yes | Source path relative to the package root. Must be relative and must not contain `..` segments. |
| `mode` | string | yes | Either `file` or `folder`. |
| `target` | string | yes | Destination path. Environment variables such as `$HOME` are expanded. |

### `file` mode

`file` mode walks the source directory and symlinks each regular file under `target`.

```toml
[packages.zsh.profiles.work]
sources = [
  {path = "common", mode = "file", target = "$HOME"},
  {path = "work", mode = "file", target = "$HOME"},
]
```

If two `file` sources produce the same target path, later sources win. This makes profile layering straightforward: put shared files first, then machine- or context-specific overrides.

### `folder` mode

`folder` mode symlinks the entire source directory to `target` as one unit.

```toml
[packages.nvim.profiles.default]
sources = [{path = "config", mode = "folder", target = "$HOME/.config/nvim"}]
```

Use `folder` when a complete directory should be managed as one link. `folder` sources are not overlayable and cannot be combined with another source that touches the same target subtree.

## Remote profile imports

Remote rices are git submodules under `remotes/<name>`, managed by `rice remote` commands.

```sh
rice remote add https://github.com/someone/base-rice.git --name base
```

Then import a profile from that remote:

```toml
[packages.nvim.profiles.work]
import = "remotes/base#nvim.default"
sources = [{path = "work", mode = "file", target = "$HOME/.config/nvim"}]
```

The import format is:

```text
remotes/<remote-name>#<package-name>.<profile-name>
```

Imports can be recursive. Cycles are rejected.

## Dependencies

Package dependencies are checked before installing a named package unless you pass `--skip-deps`.

```toml
[packages.nvim]
supported_os = ["linux", "darwin"]
dependencies = [
  {name = "neovim", version = ">=0.9.0"},
  {name = "ripgrep"},
]
```

Known dependencies are built into easyrice. Unknown dependencies must be declared under `custom_dependencies`.

```toml
[custom_dependencies.starship]
version_probe = ["starship", "--version"]
version_regex = "starship (\\d+\\.\\d+\\.\\d+)"

[custom_dependencies.starship.install.darwin]
description = "Install starship with Homebrew"
shell_payload = "brew install starship"

[custom_dependencies.starship.install.linux]
description = "Install starship with the official installer"
shell_payload = "curl -sS https://starship.rs/install.sh | sh -s -- -y"
```

Custom dependency install method keys should start with `linux`, `darwin`, or `windows`. Linux methods can include distro family suffixes such as `linux_debian`; add `distro_families = ["debian"]` when the method should only apply to specific Linux families.

## Validation rules

- `schema_version` must be `1`.
- At least one package must be declared.
- Package names cannot be empty, contain whitespace, or contain `/`.
- `supported_os` is required and may only contain `linux`, `darwin`, or `windows`.
- `root` and source `path` values must be relative and must not contain `..`.
- Every package needs at least one profile.
- Every profile needs at least one `source` or one `import`.
- Every source needs `path`, `mode`, and `target`.
- Source `mode` must be `file` or `folder`.
- Dependency entries must be inline tables, not bare strings.
- Source entries must be inline tables, not bare strings.
