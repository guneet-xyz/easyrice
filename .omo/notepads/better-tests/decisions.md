
## [2026-05-18T17:19:05Z] Task 7: catalog convention

Locked the BUG-NNN template at `.omo/known-bugs.md`. Tasks 8-18 must append entries using this template verbatim:

```
## BUG-NNN — short title
**Status**: failing | passing | wont-fix
**Severity**: S1 | S2 | S3
**Package**: internal/<pkg>
**Test**: <path/to/test.go>:TestFunctionName
**Spec source**: <citation>
**Expected**: <intended behavior>
**Actual**: <current behavior>
**Repro**: <minimal commands or test seed>
**How we know test is correct**: <pointer or argument>
```

Block reservations (zero-padded 3-digit NNN):
- 001-019: state
- 020-039: manifest
- 040-059: profile
- 060-079: installer (incl. concurrency)
- 080-099: updater
- 100-119: multi-remote
- 120-139: symlink/path
- 140-159: git
- 160-179: CLI/UX
- 180-199: E2E scenarios

Severity: S1=data loss, S2=feature broken, S3=UX wart. Status: failing | passing | wont-fix.

Catalog freshness is enforced by the bash one-liner under `## Verification` in the catalog file. Every test marker `BUG-NNN` must have a matching `## BUG-NNN` header in the catalog.

Placeholder `BUG-000` is present and must be removed before Task 18 is marked complete.
