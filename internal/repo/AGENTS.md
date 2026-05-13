# internal/repo/

Managed dotfile repo lifecycle plus remote-rice (submodule) management. Owns the fixed clone path, all `git` invocations, working-tree state queries, and scoped commits. The ONLY package allowed to invoke `git`.

## STRUCTURE

```
internal/repo/
├── repo.go         # DefaultRepoPath, RepoTomlPath, Exists, Clone, Pull, GitOnPath
├── git_ops.go      # IsGitRepo, IsClean, HasUncommittedChanges, CurrentBranch, CommitPaths,
│                   # SubmoduleAdd, SubmoduleRemove, SubmoduleUpdate, SubmoduleList,
│                   # Submodule, SubmoduleState
├── errors.go       # sentinel errors (see CONTRACT)
└── *_test.go
```

## WHERE TO LOOK

| Task | File |
|------|------|
| Change managed-repo path | `repo.go` (`DefaultRepoPath`) |
| Change `rice.toml` location within the repo | `repo.go` (`RepoTomlPath`) |
| Change clone semantics (depth, branch, etc.) | `repo.go` (`Clone`) |
| Change pull semantics (rebase, ff-only, etc.) | `repo.go` (`Pull`) |
| Change submodule add/remove/update/list | `git_ops.go` |
| Change working-tree cleanliness rule | `git_ops.go` (`IsClean`) |
| Change scoped-commit policy | `git_ops.go` (`CommitPaths`) |
| Add a new sentinel error | `errors.go` |

## CONTRACT

### Repo lifecycle (`repo.go`)

- `DefaultRepoPath()` returns `<UserConfigDir>/easyrice/repos/default` (POSIX: `~/.config/...`, Windows: `%APPDATA%/...`). Falls back to `~/.config/easyrice/repos/default` if `UserConfigDir` fails.
- `RepoTomlPath(repoRoot)` returns `<repoRoot>/rice.toml`.
- `Exists(repoPath)` returns `(true, nil)` if the path exists, `(false, nil)` if it does not, `(false, err)` only on stat errors that are NOT `os.ErrNotExist`.
- `Clone(ctx, url, dest)` shells out to `git clone <url> <dest>`. Returns wrapped error including stderr/stdout on failure. Caller is expected to ensure `dest` does not already exist.
- `Pull(ctx, repoPath)` shells out to `git -C <repoPath> pull`. Returns wrapped error including output on failure.
- `GitOnPath()` returns true if `git` is on `$PATH`. CLI commands `init`, `update`, and any `remote` subcommand MUST check this before invoking git ops and surface a clear error.

### Working-tree state (`git_ops.go`)

- `IsGitRepo(repoPath)` returns true if `<repoPath>/.git` exists (file or directory). Pure stat check, no `git` call.
- `IsClean(ctx, repoPath)` runs `git -C <repoPath> status --porcelain` and returns true when output is empty.
- `HasUncommittedChanges(ctx, repoPath)` is the logical inverse of `IsClean`.
- `CurrentBranch(ctx, repoPath)` runs `git -C <repoPath> branch --show-current` and returns the trimmed branch name.

### Scoped commits (`git_ops.go`)

- `CommitPaths(ctx, repoPath, paths, message)` stages ONLY the given paths and commits with `message`. Returns an error if `paths` is empty. NEVER uses `git add -A` or `git add .`. This is the single entry point for any auto-commit done on the user's behalf.

### Submodule management (`git_ops.go`)

- `SubmoduleAdd(ctx, repoPath, url, name)` runs `git submodule add <url> remotes/<name>`. Caller is responsible for committing `.gitmodules` and `remotes/<name>` via `CommitPaths`.
- `SubmoduleRemove(ctx, repoPath, name)` deinitialises and removes `remotes/<name>` (`git submodule deinit -f` + `git rm -f`). Caller commits the result via `CommitPaths`.
- `SubmoduleUpdate(ctx, repoPath, name)` runs `git submodule update --remote remotes/<name>`. When `name` is empty, updates all submodules.
- `SubmoduleList(ctx, repoPath)` parses `git submodule status` and returns `[]Submodule`.
- `Submodule` is `{Name, Path, SHA string; State SubmoduleState}`.
- `SubmoduleState` is one of `SubmoduleInitialized`, `SubmoduleNotInitialized`, `SubmoduleModified` (covers the `+`/`-`/`U` prefixes from `git submodule status`).

### Sentinel errors (`errors.go`)

- `ErrRepoNotInitialized` - returned upstream when callers attempt to read the manifest before `rice init` has been run.
- `ErrRepoDirty` - working tree has uncommitted changes; BLOCKS `remote add/remove`. `install`/`status`/`doctor` only WARN.
- `ErrRemoteAlreadyExists` - `remote add` was called with a name that already maps to a submodule.
- `ErrRemoteNotFound` - `remote remove`/`remote update <name>` was called with an unknown name.
- `ErrRemoteInUse` - `remote remove` was called for a remote referenced by some profile's `import` field; user must drop the import first.
- `ErrSubmoduleNotInitialized` - submodule entry exists but has not been checked out.
- `ErrPackageNotDeclared(name)` constructs an error for "package N not declared in rice.toml".

## CONVENTIONS

- All `exec.Command("git", ...)` calls live in this package - NOWHERE else in the codebase.
- Use `exec.CommandContext` so callers can cancel long-running clones/pulls.
- Always wrap errors with command output: `fmt.Errorf("git X: %w: %s", err, out)`.
- Keep this package free of business logic - no manifest parsing, no symlinking, no state I/O.
- Auto-commit policy: this package NEVER decides whether to commit. The CLI layer decides; this package only exposes `CommitPaths` so commits are always scoped to specific paths. NEVER add a `CommitAll` helper or any function that runs `git add -A` / `git add .`.

## ANTI-PATTERNS

- DO NOT add multi-repo support (multiple named clone slots, etc.) - one repo, one location, by design.
- DO NOT accept a `--repo` override - the path is fixed.
- DO NOT call `git` from any other package.
- DO NOT swallow git stderr - always include it in the wrapped error so users can debug auth/network failures.
- DO NOT add a `CommitAll` / `git add -A` / `git add .` helper. All commits MUST go through `CommitPaths` with an explicit path list.
- DO NOT auto-commit from `Clone`/`Pull`/`SubmoduleUpdate`. Only `SubmoduleAdd`/`SubmoduleRemove` are followed by a commit, and that commit is issued by the CLI layer (`cli/remote.go`) via `CommitPaths`.
