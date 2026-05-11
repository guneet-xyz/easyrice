package manifest

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Validate checks that a Manifest conforms to all schema rules.
// Returns an error if any rule is violated.
func Validate(m *Manifest) error {
	if m.SchemaVersion != 1 {
		return fmt.Errorf("validate: unsupported schema_version: %d", m.SchemaVersion)
	}

	if len(m.Packages) == 0 {
		return fmt.Errorf("validate: no packages declared")
	}

	validOS := map[string]bool{"linux": true, "darwin": true, "windows": true}

	for name, pkg := range m.Packages {
		// Validate package name
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("validate: package name must not be empty")
		}
		if strings.Contains(name, "/") {
			return fmt.Errorf("validate: package %q: name must not contain slashes", name)
		}
		if strings.ContainsAny(name, " \t\n\r") {
			return fmt.Errorf("validate: package %q: name must not contain whitespace", name)
		}

		// Validate SupportedOS
		if len(pkg.SupportedOS) == 0 {
			return fmt.Errorf("validate: package %q: supported_os must not be empty", name)
		}
		for _, os := range pkg.SupportedOS {
			if !validOS[os] {
				return fmt.Errorf("validate: package %q: unsupported OS: %q (must be one of: linux, darwin, windows)", name, os)
			}
		}

		// Validate Root (optional, but if present must not start with / or contain ..)
		if pkg.Root != "" {
			if strings.HasPrefix(pkg.Root, "/") {
				return fmt.Errorf("validate: package %q: root must not start with /", name)
			}
			if strings.Contains(pkg.Root, "..") {
				return fmt.Errorf("validate: package %q: root must not contain .. segments", name)
			}
		}

		// Validate Profiles
		if len(pkg.Profiles) == 0 {
			return fmt.Errorf("validate: package %q: at least one profile must be defined", name)
		}

		for profileName, profile := range pkg.Profiles {
			// Validate profile name
			if strings.TrimSpace(profileName) == "" {
				return fmt.Errorf("validate: package %q: profile name must not be empty", name)
			}

			// Validate Sources
			if len(profile.Sources) == 0 {
				return fmt.Errorf("validate: package %q: profile %q has no sources", name, profileName)
			}

			for i, source := range profile.Sources {
				// Validate Path
				if strings.TrimSpace(source.Path) == "" {
					return fmt.Errorf("validate: package %q: profile %q source[%d]: path must not be empty", name, profileName, i)
				}
				if strings.Contains(source.Path, "..") {
					return fmt.Errorf("validate: package %q: profile %q source[%d]: path %q must not contain .. segments", name, profileName, i, source.Path)
				}
				// Clean the path to check for issues
				cleanPath := filepath.Clean(source.Path)
				if strings.HasPrefix(cleanPath, "/") {
					return fmt.Errorf("validate: package %q: profile %q source[%d]: path %q must be relative", name, profileName, i, source.Path)
				}

				// Validate Mode
				if source.Mode != "file" && source.Mode != "folder" {
					return fmt.Errorf("validate: package %q: profile %q source[%d]: mode must be \"file\" or \"folder\", got %q", name, profileName, i, source.Mode)
				}

				// Validate Target
				if strings.TrimSpace(source.Target) == "" {
					return fmt.Errorf("validate: package %q: profile %q source[%d]: target must not be empty", name, profileName, i)
				}
			}
		}
	}

	return nil
}
