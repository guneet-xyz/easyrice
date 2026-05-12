package manifest

import (
	"fmt"
	"strings"
)

// ImportSpec represents a parsed profile import reference.
// The raw string format is: "remotes/<name>#<pkg>.<profile>"
type ImportSpec struct {
	Remote  string // e.g. "kick"
	Package string // e.g. "nvim"
	Profile string // e.g. "default"
}

// ParseImportSpec parses an import string of the form "remotes/<name>#<pkg>.<profile>".
// Returns an error if the string does not match the expected format.
func ParseImportSpec(s string) (ImportSpec, error) {
	// Must start with "remotes/"
	if !strings.HasPrefix(s, "remotes/") {
		return ImportSpec{}, fmt.Errorf("import must start with \"remotes/\"")
	}

	// Strip the prefix
	rest := strings.TrimPrefix(s, "remotes/")

	// Must contain exactly one "#" separator
	parts := strings.Split(rest, "#")
	if len(parts) != 2 {
		return ImportSpec{}, fmt.Errorf("import must contain exactly one \"#\" separator")
	}

	remoteName := parts[0]
	pkgProfile := parts[1]

	// Remote name must be non-empty and contain no slashes
	if strings.TrimSpace(remoteName) == "" {
		return ImportSpec{}, fmt.Errorf("remote name must not be empty")
	}
	if strings.Contains(remoteName, "/") {
		return ImportSpec{}, fmt.Errorf("remote name must not contain slashes")
	}

	// pkg.profile part must contain exactly one "." separator
	pkgProfileParts := strings.Split(pkgProfile, ".")
	if len(pkgProfileParts) != 2 {
		return ImportSpec{}, fmt.Errorf("import must contain exactly one \".\" separator in package.profile part")
	}

	pkgName := pkgProfileParts[0]
	profileName := pkgProfileParts[1]

	// Package name must be non-empty
	if strings.TrimSpace(pkgName) == "" {
		return ImportSpec{}, fmt.Errorf("package name must not be empty")
	}

	// Profile name must be non-empty
	if strings.TrimSpace(profileName) == "" {
		return ImportSpec{}, fmt.Errorf("profile name must not be empty")
	}

	return ImportSpec{
		Remote:  remoteName,
		Package: pkgName,
		Profile: profileName,
	}, nil
}
