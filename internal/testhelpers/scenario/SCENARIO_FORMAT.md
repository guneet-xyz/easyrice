# Scenario Format

The canonical reference for YAML-driven scenarios consumed by the easyrice test runner in `internal/testhelpers/scenario`.

## 1. Overview

A scenario is a directory containing a `steps.yaml` file plus optional seed and expected-snapshot subdirectories. Each scenario maps to a single Go test invocation: the runner loads `steps.yaml`, iterates the listed steps, and for each step executes a CLI command via a caller-supplied `Runner`, applies filesystem assertions, and reports results through the standard `testing.T` API. Steps run sequentially inside `t.Run` subtests so failures in one step do not prevent later steps from executing their assertions.

## 2. Directory Layout

```
<scenario-name>/
  steps.yaml          # required
  repo/               # optional seed repo content (copied into the sandbox repo root)
  home/               # optional seed home content (copied into the sandbox $HOME)
  expected/
    step-0/
      home.txt        # optional expected home snapshot
      repo.txt        # optional expected repo snapshot
      state.txt       # optional expected state.json snapshot
    step-1/
      ...
```

The names `step-0`, `home.txt`, `repo.txt`, `state.txt` are conventions. The actual file paths are whatever each step's `expect.home`, `expect.repo`, and `expect.state` point at, resolved relative to the scenario directory.

## 3. steps.yaml Schema

### Top-level

| Field   | Type       | Required | Description                                                                 |
|---------|------------|----------|-----------------------------------------------------------------------------|
| `mocks` | `[]string` | no       | Names looked up in `Config.MockRegistry`. Each setup runs once, before any step. |
| `steps` | `[]Step`   | yes      | Ordered list of steps to execute.                                           |

### Step

| Field    | Type                | Required | Description                                                                                |
|----------|---------------------|----------|--------------------------------------------------------------------------------------------|
| `name`   | `string`            | yes      | Human-readable label. Used as the `t.Run` subtest name. Must be non-empty.                 |
| `args`   | `[]string`          | yes\*    | CLI args passed to the `Runner`. Required unless the step contains only `mutate` ops.      |
| `stdin`  | `string`            | no       | Text piped to the command's stdin.                                                         |
| `env`    | `map[string]string` | no       | Extra env vars set for the duration of the step (via `t.Setenv`).                          |
| `mutate` | `[]MutateOp`        | no       | Filesystem mutations applied before the `Runner` is invoked.                               |
| `expect` | `Expect`            | no       | Assertions checked after the `Runner` returns.                                             |

### Expect

| Field                  | Type       | Default | Description                                                                          |
|------------------------|------------|---------|--------------------------------------------------------------------------------------|
| `exit_code`            | `int`      | `0`     | Expected exit. `0` means `Runner` returns `nil`; any non-zero means it returns an error. The runner does not compare specific non-zero codes. |
| `stdout_contains`      | `[]string` | nil     | Every substring must appear in normalized stdout.                                    |
| `stdout_not_contains`  | `[]string` | nil     | None of these substrings may appear in normalized stdout.                            |
| `stdout_equals`        | `string`   | `""`    | If set, normalized stdout must match exactly.                                        |
| `home`                 | `string`   | `""`    | Path (relative to scenario dir) of expected home-tree snapshot file.                 |
| `repo`                 | `string`   | `""`    | Path (relative to scenario dir) of expected repo-tree snapshot file.                 |
| `state`                | `string`   | `""`    | Path (relative to scenario dir) of expected state.json snapshot file.                |

Referenced snapshot files must exist on disk at load time, otherwise `LoadScenario` returns `ErrInvalidYAML`.

Stdout normalization strips ANSI escape sequences, converts CRLF to LF, trims trailing whitespace per line, and removes trailing blank lines.

### MutateOp

| Field     | Type          | Description                                                                                       |
|-----------|---------------|---------------------------------------------------------------------------------------------------|
| `op`      | `string`      | One of `remove`, `write_file`, `replace_symlink`, `mkdir`, `chmod`. Unknown values are rejected.  |
| `path`    | `string`      | Target path. Supports `<HOME>` and `<REPO>` placeholders. Must resolve inside the home or repo sandbox. |
| `content` | `string`      | File body for `write_file`.                                                                       |
| `target`  | `string`      | Symlink target for `replace_symlink`. Also expands `<HOME>` and `<REPO>`.                         |
| `mode`    | `os.FileMode` | Permission bits for `mkdir` and `chmod`. Accepts integer, float, or octal string (e.g. `"0755"`). `mkdir` defaults to `0o755` when omitted. |

Supported `op` semantics:

- `remove`: `os.RemoveAll(path)`. Safe on missing paths.
- `write_file`: Creates parent directories (`0o755`), then writes `content` at mode `0o644`.
- `replace_symlink`: Removes the existing symlink at `path`, then creates a new one pointing at `target`.
- `mkdir`: `os.MkdirAll(path, mode)`.
- `chmod`: `os.Chmod(path, mode)`.

Containment is enforced: each resolved path must sit under either the home or repo sandbox root.

## 4. Placeholder Expansion

Three placeholders are recognized:

- `<HOME>` expands to the sandbox home directory (the step's `env.HOME`).
- `<REPO>` expands to the sandbox repo root (the step's `env.EASYRICE_REPO`).
- `<STATE>` expands to `<HOME>/.config/easyrice/state.json`.

Expansion happens in two places:

1. **Mutate paths and targets** (`<HOME>`, `<REPO>`): substituted before applying the op.
2. **Expected snapshot file contents** (`<HOME>`, `<REPO>`, `<STATE>`): substituted before diffing against captured output.

Plain string replacement is used. Placeholders are case-sensitive.

## 5. Snapshot File Format

Home and repo snapshots are deterministic, sorted, one-entry-per-line listings of the directory tree. `.git/` subtrees are skipped. The root entry itself is omitted.

Per-entry format:

```
<relative-path> [file]
<relative-path> [dir]
<relative-path> -> <absolute-target>
```

- Regular files end with ` [file]`.
- Directories end with ` [dir]`.
- Symlinks render as `rel -> target`. The target is whatever `os.Readlink` returns. In expected files, write absolute targets using `<REPO>` (or `<HOME>`) so the snapshot stays portable across sandbox locations.

State snapshots are the contents of `state.json` re-marshaled through `encoding/json` with sorted keys and a two-space indent. If the state file does not exist when the snapshot is captured, the captured bytes are the literal string `<no state file>`.

Diffs are reported line-by-line: `-` for expected-only, `+` for actual-only, ` ` for shared lines.

## 6. Environment Variables

Three env vars in each step's `env` map drive what the runner inspects:

| Variable          | Required when                       | Used for                                                                 |
|-------------------|-------------------------------------|--------------------------------------------------------------------------|
| `HOME`            | `expect.home` is set, or any mutate path uses `<HOME>` | Root of the home snapshot and `<HOME>` expansion.            |
| `EASYRICE_REPO`   | `expect.repo` is set, or any mutate path uses `<REPO>` | Root of the repo snapshot and `<REPO>` expansion.            |
| `EASYRICE_STATE`  | `expect.state` is set               | Path to the state.json file the runner reads for the state snapshot.    |

If a snapshot is requested but the corresponding env var is empty, the step fails fatally. Set these vars in every step that performs assertions, even when they are identical across steps. They are not inherited from earlier steps for snapshot purposes.

## 7. Complete Worked Example

A two-step scenario that installs a package, then uninstalls it.

`testdata/install-then-uninstall/steps.yaml`:

```yaml
mocks:
  - hostname

steps:
  - name: install ghostty
    args: ["install", "ghostty", "--profile", "common", "-y"]
    env:
      HOME: /tmp/scenario-home
      EASYRICE_REPO: /tmp/scenario-repo
      EASYRICE_STATE: /tmp/scenario-home/.config/easyrice/state.json
    expect:
      exit_code: 0
      stdout_contains:
        - "installed ghostty"
      home: expected/step-0/home.txt
      state: expected/step-0/state.txt

  - name: uninstall ghostty
    args: ["uninstall", "ghostty", "-y"]
    env:
      HOME: /tmp/scenario-home
      EASYRICE_REPO: /tmp/scenario-repo
      EASYRICE_STATE: /tmp/scenario-home/.config/easyrice/state.json
    mutate:
      - op: write_file
        path: <HOME>/.config/ghostty/extra
        content: "stray file\n"
    expect:
      exit_code: 0
      stdout_contains:
        - "uninstalled ghostty"
      home: expected/step-1/home.txt
      state: expected/step-1/state.txt
```

`expected/step-0/home.txt`:

```
.config [dir]
.config/easyrice [dir]
.config/easyrice/state.json [file]
.config/ghostty [dir]
.config/ghostty/config -> <REPO>/ghostty/common/config
```

`expected/step-0/state.txt`:

```
{
  "ghostty": {
    "installed_at": "2025-05-10T12:34:56Z",
    "installed_links": [
      {
        "source": "<REPO>/ghostty/common/config",
        "target": "<HOME>/.config/ghostty/config"
      }
    ],
    "profile": "common"
  }
}
```

`expected/step-1/home.txt`:

```
.config [dir]
.config/easyrice [dir]
.config/easyrice/state.json [file]
.config/ghostty [dir]
.config/ghostty/extra [file]
```

`expected/step-1/state.txt`:

```
{}
```

The actual env values are set by the test harness to `t.TempDir()` paths; the literal `/tmp/...` strings above are illustrative. In a real scenario test, the Go-side wrapper computes the temp paths, writes them into a copy of `steps.yaml`, and then calls `scenario.Run`. Inside expected files the test relies on `<HOME>` and `<REPO>` placeholders so the snapshots stay portable.

## 8. Constraints

- Do not call `t.Parallel()` in scenario tests. Steps mutate process env via `t.Setenv` and share a single working directory inside the harness.
- Never point `HOME` or `EASYRICE_STATE` at the real `$HOME` of the developer's machine. Use `t.TempDir()` for both the home sandbox and the repo sandbox, then write those paths into the step `env`.
- Snapshots only see what the sandbox contains. Anything written outside `<HOME>` or `<REPO>` falls outside containment and will fail the mutate guard.
- Snapshot files are checked in as test fixtures. Regenerate them with care; the diff output is the source of truth when something breaks.

## 9. Placeholder Syntax (Two Forms)

Scenarios use two distinct placeholder syntaxes that operate at different lifecycle stages. They are not interchangeable.

| Form | Where it goes | Substituted by | When |
|------|---------------|----------------|------|
| `__HOME__`, `__REPO__`, `__STATE__` | `steps.yaml` | `renderScenario` (`cli/scenario_run_test.go`) | Render time, before the test runs |
| `<HOME>`, `<REPO>` | Expected snapshot files (`expected/step-N/*.txt`) | `expandPlaceholders` (`internal/testhelpers/scenario/snapshot.go`) | Compare time, just before diffing |

Why two forms exist: `steps.yaml` is rewritten on disk into a temp dir before the executor parses it, so the harness needs a syntax that survives copy and replace. Expected snapshots stay on disk as-is and are only expanded in memory when compared against captured output. Different lifecycle stages, different substitution machinery, different placeholder shapes.

Use `__HOME__`, `__REPO__`, `__STATE__` in `steps.yaml` for the `args`, `env`, `stdin`, and any field whose value must be a real absolute path before the CLI runs. Use `<HOME>` and `<REPO>` inside expected snapshot files so checked-in fixtures stay portable across sandbox locations. The `<STATE>` placeholder is also valid inside expected snapshots and resolves to `<HOME>/.config/easyrice/state.json`.

Mutate ops are the one exception inside `steps.yaml`: their `path` and `target` fields accept `<HOME>` and `<REPO>` because they are expanded by the executor at mutate time, not during render. See section 4.

Common mistake: writing `<HOME>` in `steps.yaml` outside a `mutate` op, or `__HOME__` inside an expected snapshot file. Both silently no-op. The CLI then receives the literal string `<HOME>` as an argument, or the diff fails because the expected file still contains `__HOME__` while the captured output contains a real path. If a step looks correct but the assertion diff shows raw placeholders on one side, you have crossed the streams.

## 10. Mutate-only Steps Pattern

The executor always invokes `cfg.Runner` once per step, even when the step's only real work is `mutate`. There is no pure-mutate step type. To express "mutate the sandbox, then assert without running a meaningful command", use `args: ["version"]` as a no-op carrier: it touches no state, exits 0, and produces deterministic stdout.

Worked example, taken from `cli/testdata/scenarios/converge-repair-broken-symlink/steps.yaml`:

```yaml
- name: "break symlink"
  args: ["version"]
  mutate:
    - op: remove
      path: "<HOME>/.config/demo/file1"
  env:
    HOME: "__HOME__"
    EASYRICE_REPO: "__REPO__"
    EASYRICE_STATE: "__STATE__"
  expect:
    exit_code: 0
```

The mutate runs first and corrupts the sandbox. The `version` invocation then exits cleanly without observing or modifying any easyrice state. The next step in the scenario can then assert that `install` repairs the drift introduced here.

Do not use `install`, `uninstall`, `update`, or any command that reads or writes state as the no-op carrier. They are not idempotent across mutates and will pollute the next step's expectations.

## 11. Glob Discovery

New scenarios are auto-discovered by `TestScenarios_AllDiscovered` in `cli/`; no per-scenario Go function is needed. Drop a directory under `cli/testdata/scenarios/` matching the layout below, and the next test run picks it up.

Required layout:

```
cli/testdata/scenarios/<name>/
  steps.yaml
  repo/
  expected/
    step-0/
    step-1/
    ...
```

`steps.yaml` is mandatory. `repo/` is the seed repo content (copied into the sandbox repo root before step 0). Each `step-N/` directory holds the snapshot files referenced by that step's `expect.home`, `expect.repo`, and `expect.state`. Step indices are zero-based and must match the order in `steps.yaml`.
