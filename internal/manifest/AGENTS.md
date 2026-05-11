# internal/manifest/

`rice.toml` parsing, validation, and OS gating. Schema definitions live here.

## STRUCTURE

```
internal/manifest/
├── schema.go       # Manifest, ProfileDef, SourceSpec + SourceSpec.UnmarshalTOML
├── load.go         # Load(dir) parse+validate single package / Discover(repoRoot) scan all
├── validate.go     # Validate(*Manifest) - semantic checks beyond TOML decode
├── osgating.go     # CheckOS(*Manifest, currentOS) - package-level OS gate
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

- `Load(dir)` reads `<dir>/rice.toml`, parses, validates. Returns `(*Manifest, error)`. NEVER returns a partially-valid manifest.
- `Discover(repoRoot)` scans direct children of `repoRoot` (one level deep ONLY). Silently skips dirs with no `rice.toml`. Returns first parse/validate error encountered.
- `SourceSpec.UnmarshalTOML` accepts ONLY the inline table form `{path=..., mode=..., target=...}`. Bare strings or other shapes → error. All three fields required.
- `CheckOS` returns nil if `currentOS` is in `supported_os`, else descriptive error. Empty `supported_os` is rejected by `Validate`.

## CONVENTIONS

- Use `BurntSushi/toml` (already in go.mod). Do not switch decoders.
- Validation errors should name the offending field: `fmt.Errorf("profiles.%s.sources[%d]: ...", name, i)`.
- Keep `schema.go` as data-only. All semantics → `validate.go`.

## ANTI-PATTERNS

- DO NOT relax `SourceSpec.UnmarshalTOML` to accept bare strings - the table form is intentional and documented.
- DO NOT silently default missing fields - return an error pointing at the field path.
- DO NOT walk deeper than one level in `Discover` - packages are direct children of the repo root, by spec.
- DO NOT introduce schema changes without bumping `schema_version` and updating `Validate` to reject older versions.
