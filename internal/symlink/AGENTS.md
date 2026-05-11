# internal/symlink/

The ONLY package allowed to call `os.Symlink`, `os.Readlink`, and symlink-removing `os.Remove`. Thin wrappers with strict pre/post conditions.

## API

```
CreateSymlink(source, target string) error      // fails if target exists (any kind); mkdir -p parents
RemoveSymlink(target string) error              // fails if target missing OR not a symlink
IsSymlinkTo(target, source string) (bool, err)  // false (not error) if missing/not-a-symlink
ReadLink(target string) (string, error)         // wraps os.Readlink with context
```

## CONTRACT

- `CreateSymlink` is **strict**: any pre-existing target → error. Caller must check first via `IsSymlinkTo` for idempotency.
- `RemoveSymlink` refuses non-symlinks (safety net against deleting real files).
- `IsSymlinkTo` returns `(false, nil)` for missing target - convenience for idempotency checks; reserve errors for actual stat failures.
- All paths SHOULD be absolute. Callers are responsible (no normalization here).
- `os.Lstat` (NOT `os.Stat`) is used everywhere - never follow the link when checking existence.

## WINDOWS NOTE

`os.Symlink` on Windows requires Developer Mode or Administrator. This package does NOT check at runtime - that responsibility lives in `internal/doctor/`. Keep it that way (separation of concerns).

## ANTI-PATTERNS

- DO NOT add caching - filesystem state can change between calls.
- DO NOT add retry logic - callers decide policy.
- DO NOT use `os.Stat` (follows links) where `os.Lstat` is needed.
- DO NOT relax `RemoveSymlink` to delete real files - that defeats its purpose.
- DO NOT add Windows runtime checks here - belongs in `doctor/`.
- Other packages MUST NOT import `os.Symlink`/`os.Readlink` - go through this package.
