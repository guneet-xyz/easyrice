# Usage

`easyrice` installs packages from one managed dotfile repository. The binary is named `easyrice`; the installer also creates a `rice` symlink, and the examples below use `rice`.

## Managed paths

`rice init` clones your dotfile repository into a fixed location:

| OS | Managed repo |
|---|---|
| Linux/macOS | `~/.config/easyrice/repos/default/` |
| Windows | `%APPDATA%/easyrice/repos/default/` |

Runtime state is stored separately:

| OS | State file |
|---|---|
| Linux/macOS | `~/.config/easyrice/state.json` |
| Windows | `%APPDATA%/easyrice/state.json` |

There is no `--repo` flag. Update or replace the managed repository with `rice update` or by removing the managed repo directory and running `rice init` again.

## First-time setup

1. Add a `rice.toml` to the root of your dotfile repo. See [`rice-toml.md`](./rice-toml.md).
2. Clone that repo into easyrice's managed location.
3. Install one package, or converge every package.

```sh
rice init https://github.com/you/dotfiles.git
rice install nvim --profile macbook
```

Install every declared package:

```sh
rice install --profile macbook
```

If you omit `--profile`, easyrice chooses a profile in this order:

1. The package's currently installed profile from state.
2. A profile matching the machine hostname.
3. A `default` profile.
4. The only profile, when the package declares exactly one.

If no default can be inferred, pass `--profile` explicitly.

## Commands

| Command | Purpose |
|---|---|
| `rice init <repo-url>` | Clone your dotfile repo into the fixed managed location. |
| `rice update` | Run `git pull` in the managed repo. |
| `rice install [package]` | Converge one package, or every package when omitted. Handles first install, profile switch, repair, and no-op. |
| `rice uninstall <package>` | Remove all symlinks recorded for a package in state. |
| `rice status [package]` | Show repo state, installed packages, drift, dependencies, and remotes. |
| `rice doctor` | Run health checks. |
| `rice remote add <url> --name <name>` | Add another rice as a git submodule under `remotes/<name>`. |
| `rice remote remove <name>` | Remove a remote rice submodule. |
| `rice remote update [name]` | Update one remote rice, or all remotes when omitted. |
| `rice remote list` | List configured remote rices. |
| `rice upgrade` | Upgrade the easyrice binary from GitHub releases. |
| `rice version` | Print the current version. |

## Common workflows

### Install or switch a profile

`install` is converge-shaped. The same command installs a new package, switches an existing package to another profile, repairs drift, or reports a no-op.

```sh
rice install ghostty --profile macbook
rice install ghostty --profile work
```

### Repair links after editing dotfiles

Run install again. easyrice compares desired links to state and fixes missing or drifted links.

```sh
rice install nvim
```

### Install without dependency checks

```sh
rice install nvim --skip-deps
```

### Skip confirmation prompts

```sh
rice install nvim --profile macbook -y
rice uninstall nvim -y
```

### Use a temporary state file

Useful for tests, dry experiments, and isolated machines.

```sh
rice install nvim --profile default --state /tmp/easyrice-state.json
rice status --state /tmp/easyrice-state.json
```

### Pull repo changes

```sh
rice update
rice install
```

### Use remote rices

Add a remote rice:

```sh
rice remote add https://github.com/someone/base-rice.git --name base
```

Import a profile from it in `rice.toml`:

```toml
[packages.nvim.profiles.work]
import = "remotes/base#nvim.default"
sources = [{path = "work", mode = "file", target = "$HOME/.config/nvim"}]
```

Update remotes later:

```sh
rice remote update
rice remote list
```

## Flags and environment

| Flag | Scope | Purpose |
|---|---|---|
| `--profile <name>` | `install` | Desired profile. Optional when easyrice can infer one. |
| `--skip-deps` | `install` | Skip dependency checks and dependency installation. |
| `--state <path>` | global | Override the state file path. |
| `--log-level <level>` | global | `debug`, `info`, `warn`, `error`, or `critical`. |
| `--yes`, `-y` | global | Bypass confirmation prompts. |
| `--no-update-check` | global | Skip update reminder checks. |

| Environment variable | Effect |
|---|---|
| `EASYRICE_LOG_LEVEL` | Sets the default log level when `--log-level` is not passed. |
| `EASYRICE_NO_UPDATE_CHECK=1` | Skips update reminder checks. |

## Safety model

- easyrice creates absolute symlinks from your home directory back into the managed repo.
- Install targets must stay inside `$HOME`.
- Uninstall removes only links recorded in the state file for that package.
- `install` and `uninstall` never commit changes to your dotfile repo.
- `remote add` and `remote remove` are the only commands that auto-commit, and they only commit scoped submodule changes.
- Do not hand-edit `state.json`; use the CLI.
