# internal/manifest/

Single-file root `rice.toml` parsing, validation, and OS gating. Schema definitions live here.

## STRUCTURE

```
internal/manifest/
├── schema.go       # Manifest, PackageDef, ProfileDef, SourceSpec + SourceSpec.UnmarshalTOML
├── load.go         # LoadFile(path) - parse + validate the root rice.toml
├── validate.go     # Validate(*Manifest) - semantic checks beyond TOML decode
├── osgating.go     # CheckOS(*PackageDef, currentOS) - package-level OS gate
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

## CONTRACT

- `LoadFile(path)` reads the file at `path` (typically `repo.RepoTomlPath(repoRoot)`), parses, validates. Returns `(*Manifest, error)`. NEVER returns a partially-valid manifest.
- The single root `rice.toml` declares ALL packages under `[packages.<name>]` tables.
- `PackageDef.Root` is optional; when empty, callers MUST default it to the package name. The manifest layer does NOT mutate it.
- `SourceSpec.UnmarshalTOML` accepts ONLY the inline table form `{path=..., mode=..., target=...}`. Bare strings or other shapes → error. All three fields required.
- `CheckOS` returns nil if `currentOS` is in `supported_os`, else descriptive error. Empty `supported_os` is rejected by `Validate`.

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
