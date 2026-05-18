# testdata/ Audit Report

**Date:** 2026-05-18  
**Branch:** better-tests  
**Scope:** All top-level fixtures under `testdata/` and immediate children of `testdata/manifest_invalid_v2/`

---

## Summary

| Status | Count | Details |
|--------|-------|---------|
| **KEEP** | 5 | Referenced in active tests |
| **DEAD** | 2 | No references found in codebase |
| **MIGRATE** | 1 | Inline-mirrored; fixture may be deletable |
| **TOTAL** | 8 | Top-level fixtures |

---

## Top-Level Fixtures Audit

| Fixture | Reference Type | file:line | Disposition | Notes |
|---------|---|---|---|---|
| `.gitkeep` | N/A | — | KEEP | Placeholder; required for git |
| `install` | DEAD | — | DEAD | v1 pre-v2 fixture; no references in codebase |
| `install_v2` | REFERENCED | `internal/installer/folder_test.go:15`<br/>`internal/installer/install_test.go:22` | KEEP | Active test fixture for install scenarios |
| `manifest_invalid_v2` | REFERENCED (partial) | `internal/manifest/load_test.go:45` | KEEP | Parent dir; only `missing_packages` child is referenced |
| `manifest_valid_v2` | REFERENCED | `internal/manifest/load_test.go:14` | KEEP | Valid manifest test fixture |
| `manifest_with_deps` | INLINE-MIRRORED | `cli/cli_e2e_test.go:80` | MIGRATE | Inline fixture in `installDepsFixture()`; directory may be deletable |
| `manifest_with_import` | REFERENCED | `cli/converge_integration_test.go` (via `fixtureAbs`) | KEEP | Used by import/overlay integration tests |
| `remote_rice` | REFERENCED | `cli/converge_integration_test.go:301,330,368,418,449,466` | KEEP | Remote submodule fixture for integration tests |
| `state_legacy` | DEAD | — | DEAD | Legacy state format; no references in codebase |

---

## manifest_invalid_v2/ Children Audit

| Child | Reference Type | file:line | Disposition | Notes |
|-------|---|---|---|---|
| `bad_mode` | DEAD | — | DEAD | Invalid manifest test case; not referenced |
| `bad_os` | DEAD | — | DEAD | Invalid manifest test case; not referenced |
| `custom_collides_registry` | DEAD | — | DEAD | Invalid manifest test case; not referenced |
| `dotdot_root` | DEAD | — | DEAD | Invalid manifest test case; not referenced |
| `empty_profile` | DEAD | — | DEAD | Invalid manifest test case; not referenced |
| `missing_packages` | REFERENCED | `internal/manifest/load_test.go:45` | KEEP | Referenced in load validation test |
| `reserved_self_dep` | DEAD | — | DEAD | Invalid manifest test case; not referenced |

---

## Migration Plan

### KEEP (No Action Required)
- **install_v2**: Used by `internal/installer/folder_test.go` and `internal/installer/install_test.go`
- **manifest_valid_v2**: Used by `internal/manifest/load_test.go`
- **manifest_invalid_v2/missing_packages**: Used by `internal/manifest/load_test.go`
- **manifest_with_import**: Used by `cli/converge_integration_test.go` (import/overlay scenarios)
- **remote_rice**: Used by `cli/converge_integration_test.go` (6 test functions)

### MIGRATE (Candidate for Deletion)
- **manifest_with_deps**: Currently mirrored inline in `cli/cli_e2e_test.go:83` via `installDepsFixture()`. The fixture directory exists but is not directly referenced. **Action**: Verify that `installDepsFixture()` fully covers the manifest content, then delete `testdata/manifest_with_deps/` if confirmed.

### DEAD (Candidates for Deletion)
- **testdata/install/**: v1 pre-v2 fixture; no references in any test file. Safe to delete.
- **testdata/state_legacy/**: Legacy state format; no references in any test file. Safe to delete.
- **testdata/manifest_invalid_v2/bad_mode/**: Invalid manifest test case; not referenced. Safe to delete.
- **testdata/manifest_invalid_v2/bad_os/**: Invalid manifest test case; not referenced. Safe to delete.
- **testdata/manifest_invalid_v2/custom_collides_registry/**: Invalid manifest test case; not referenced. Safe to delete.
- **testdata/manifest_invalid_v2/dotdot_root/**: Invalid manifest test case; not referenced. Safe to delete.
- **testdata/manifest_invalid_v2/empty_profile/**: Invalid manifest test case; not referenced. Safe to delete.
- **testdata/manifest_invalid_v2/reserved_self_dep/**: Invalid manifest test case; not referenced. Safe to delete.

---

## Recommendations

1. **Immediate**: Delete `testdata/install/` (v1 fixture, fully superseded by `install_v2`)
2. **Immediate**: Delete `testdata/state_legacy/` (legacy format, no active tests)
3. **Investigate**: Review `cli/cli_e2e_test.go:83` `installDepsFixture()` to confirm it fully covers `manifest_with_deps` content. If yes, delete `testdata/manifest_with_deps/`.
4. **Investigate**: Determine if the 6 dead `manifest_invalid_v2/` children are intentional (reserved for future tests) or truly dead. If dead, delete them.

---

## Audit Methodology

- **Grep search**: `grep -rn "testdata" --include="*.go" .` across entire codebase
- **Enumeration**: `ls -la testdata/` and `ls -la testdata/manifest_invalid_v2/`
- **File walk**: `find testdata/ -type f` to verify structure
- **Reference verification**: Each KEEP/MIGRATE entry verified with explicit grep output (see `.omo/evidence/task-1-references-verified.txt`)
