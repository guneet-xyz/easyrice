# internal/repo/

Managed dotfile repo lifecycle. Owns the fixed clone path, `git clone`, `git pull`, and existence checks. The ONLY package allowed to invoke `git`.

## STRUCTURE

```
internal/repo/
├── repo.go         # DefaultRepoPath, RepoTomlPath, Exists, Clone, Pull, GitOnPath
├── errors.go       # ErrRepoNotInitialized, ErrPackageNotDeclared
└── repo_test.go
```

## WHERE TO LOOK

| Task | File |
|------|------|
| Change managed-repo path | `repo.go` (`DefaultRepoPath`) |
| Change `rice.toml` location within the repo | `repo.go` (`RepoTomlPath`) |
| Change clone semantics (depth, branch, etc.) | `repo.go` (`Clone`) |
| Change pull semantics (rebase, ff-only, etc.) | `repo.go` (`Pull`) |
| Add a new sentinel error | `errors.go` |

## CONTRACT

- `DefaultRepoPath()` returns `<UserConfigDir>/easyrice/repos/default` (POSIX: `~/.config/...`, Windows: `%APPDATA%/...`). Falls back to `~/.config/easyrice/repos/default` if `UserConfigDir` fails.
- `RepoTomlPath(repoRoot)` returns `<repoRoot>/rice.toml`.
- `Exists(repoPath)` returns `(true, nil)` if the path exists, `(false, nil)` if it does not, `(false, err)` only on stat errors that are NOT `os.ErrNotExist`.
- `Clone(ctx, url, dest)` shells out to `git clone <url> <dest>`. Returns wrapped error including stderr/stdout on failure. Caller is expected to ensure `dest` does not already exist.
- `Pull(ctx, repoPath)` shells out to `git -C <repoPath> pull`. Returns wrapped error including output on failure.
- `GitOnPath()` returns true if `git` is on `$PATH`. CLI commands `init` and `update` MUST check this before invoking `Clone`/`Pull` and surface a clear error.
- `ErrRepoNotInitialized` is the sentinel returned upstream when callers attempt to read the manifest before `rice init` has been run.
- `ErrPackageNotDeclared(name)` constructs an error for "package N not declared in rice.toml".

## CONVENTIONS

- All `exec.Command("git", ...)` calls live in this file - NOWHERE else in the codebase.
- Use `exec.CommandContext` so callers can cancel long-running clones/pulls.
- Always wrap errors with command output: `fmt.Errorf("git X: %w: %s", err, out)`.
- Keep this package free of business logic - no manifest parsing, no symlinking, no state I/O.

## ANTI-PATTERNS

- DO NOT add multi-repo support (multiple named clone slots, etc.) - one repo, one location, by design.
- DO NOT accept a `--repo` override - the path is fixed.
- DO NOT call `git` from any other package.
- DO NOT swallow git stderr - always include it in the wrapped error so users can debug auth/network failures.
