package updater

import (
	"fmt"

	"golang.org/x/mod/semver"
)

// normalize ensures the version string starts with "v".
// If v already starts with "v", it returns v unchanged.
// Otherwise, it prepends "v" to v.
func normalize(v string) string {
	if len(v) > 0 && v[0] == 'v' {
		return v
	}
	return "v" + v
}

// IsDevBuild returns true if v is a development build.
// A version is considered a dev build if:
// - v is empty string, OR
// - v is "dev", OR
// - v is not a valid semantic version (after normalization)
func IsDevBuild(v string) bool {
	if v == "" || v == "dev" {
		return true
	}
	normalized := normalize(v)
	return !semver.IsValid(normalized)
}

// IsNewer returns true if latest is a newer version than current.
// Both versions are normalized before comparison.
// Returns an error if either version is invalid.
func IsNewer(current, latest string) (bool, error) {
	normCurrent := normalize(current)
	normLatest := normalize(latest)

	if !semver.IsValid(normCurrent) {
		return false, fmt.Errorf("updater: invalid semver: %w", ErrInvalidSemver)
	}
	if !semver.IsValid(normLatest) {
		return false, fmt.Errorf("updater: invalid semver: %w", ErrInvalidSemver)
	}

	cmp := semver.Compare(semver.Canonical(normCurrent), semver.Canonical(normLatest))
	return cmp < 0, nil
}

// IsPreRelease returns true if v is a pre-release version.
// A version is a pre-release if it has a prerelease component (e.g., "v1.0.0-beta.1").
func IsPreRelease(v string) bool {
	normalized := normalize(v)
	return semver.Prerelease(normalized) != ""
}
