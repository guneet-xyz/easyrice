package deps

import (
	"fmt"

	semver "github.com/Masterminds/semver/v3"
)

// MatchVersion reports whether installed satisfies constraint.
// If constraint is empty, any version is accepted (returns true, nil).
// installed may have a leading "v" prefix (stripped automatically by Masterminds).
func MatchVersion(installed string, constraint string) (bool, error) {
	if constraint == "" {
		return true, nil
	}
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return false, fmt.Errorf("invalid constraint %q: %w", constraint, err)
	}
	v, err := semver.NewVersion(installed)
	if err != nil {
		return false, fmt.Errorf("invalid version %q: %w", installed, err)
	}
	return c.Check(v), nil
}

// IsValidConstraint returns nil if s is a valid semver constraint string.
func IsValidConstraint(s string) error {
	if s == "" {
		return nil
	}
	_, err := semver.NewConstraint(s)
	if err != nil {
		return fmt.Errorf("invalid semver constraint %q: %w", s, err)
	}
	return nil
}
