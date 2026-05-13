# internal/manifest/

Single-file root `rice.toml` parsing, validation, OS gating, and profile-import spec parsing. Schema definitions live here.

## STRUCTURE

```
internal/manifest/
├── schema.go       # Manifest, PackageDef, ProfileDef (incl. Import), SourceSpec + SourceSpec.UnmarshalTOML
├── load.go         # LoadFile(path) - parse + validate the root rice.toml
├── validate.go     # Validate(*Manifest) - semantic checks beyond TOML decode
├── osgating.go     # CheckOS(*PackageDef, currentOS) - package-level OS gate
├── import_spec.go  # ImportSpec + ParseImportSpec - "remotes/<name>#<pkg>.<profile>"
└── *_test.go
```

## WHERE TO LOOK

| Task | File |
|------|------|
| Add a new schema field | `schema.go` (struct tag) + `validate.go` (semantic check) |
| Add new validation rule | `validate.go` |
| Bump `schema_version` | `schema.go` const + `validate.go` rejection of older versions |
| Add OS to allowlist | `osgating.go` (currently `linux`, `darwin`, `windows`) |
| Add new source `mode` value | `schema.go` doc + `validate.go` allowlist + `installer/install.go` handler |
| Change profile `import` syntax | `import_spec.go` (`ParseImportSpec`) |

## CONTRACT

- `LoadFile(path)` reads the file at `path` (typically `repo.RepoTomlPath(repoRoot)`), parses, validates. Returns `(*Manifest, error)`. NEVER returns a partially-valid manifest.
- The single root `rice.toml` declares ALL packages under `[packages.<name>]` tables.
- `PackageDef.Root` is optional; when empty, callers MUST default it to the package name. The manifest layer does NOT mutate it.
- `ProfileDef` has two fields: `Sources []SourceSpec` and `Import string` (TOML key `import`, omitempty). Both may be present at once.
  - When `Import` is non-empty, `internal/profile.ResolveSpecs` resolves the imported profile's sources first, then overlays the local `Sources` (file-mode last-wins).
  - Imports are recursive and cycle-detected at resolution time (in `internal/profile`, not here).
- `SourceSpec.UnmarshalTOML` accepts ONLY the inline table form `{path=..., mode=..., target=...}`. Bare strings or other shapes → error. All three fields required.
- `CheckOS` returns nil if `currentOS` is in `supported_os`, else descriptive error. Empty `supported_os` is rejected by `Validate`.

### ImportSpec (`import_spec.go`)

- `ImportSpec` is `{Remote, Package, Profile string}`.
- `ParseImportSpec(s)` parses `"remotes/<name>#<pkg>.<profile>"` into an `ImportSpec`. Rules:
  - Must start with `remotes/`.
  - Must contain exactly one `#` separator.
  - Remote name must be non-empty and contain no slashes.
  - The `<pkg>.<profile>` part must contain exactly one `.` separator with both halves non-empty.
- Cycle detection and the actual cross-remote resolution live in `internal/profile`, NOT here. This package only parses syntax.

## CONVENTIONS

- Use `BurntSushi/toml` (already in go.mod). Do not switch decoders.
- Validation errors should name the offending field: `fmt.Errorf("packages.%s.profiles.%s.sources[%d]: ...", pkg, profile, i)`.
- Keep `schema.go` as data-only. All semantics → `validate.go`.

## ANTI-PATTERNS

- DO NOT reintroduce `Discover()` - it has been deleted. The single-file manifest model means there is nothing to discover. Use `LoadFile(repo.RepoTomlPath(repoRoot))`.
- DO NOT scan subdirectories looking for nested `rice.toml` files - by design there is exactly one.
- DO NOT relax `SourceSpec.UnmarshalTOML` to accept bare strings - the table form is intentional and documented.
- DO NOT silently default missing fields (other than `PackageDef.Root` which is documented optional and resolved at the call site).
- DO NOT introduce schema changes without bumping `schema_version` and updating `Validate` to reject older versions.
- DO NOT do cross-remote resolution or cycle detection here. `ParseImportSpec` is purely syntactic; resolution lives in `internal/profile`.
- DO NOT accept `import` formats other than `remotes/<name>#<pkg>.<profile>`. The single shape is intentional.
