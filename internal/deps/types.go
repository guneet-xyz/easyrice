package deps

import (
	"fmt"
	"time"
)

// DependencyRef represents a dependency reference as it appears in rice.toml
// under packages.<name>.dependencies. It accepts only the table form.
type DependencyRef struct {
	Name    string `toml:"name"`
	Version string `toml:"version"` // semver constraint, optional
}

// UnmarshalTOML implements the toml.Unmarshaler interface from
// github.com/BurntSushi/toml. It accepts only the table form and rejects
// bare strings or other non-table forms.
func (d *DependencyRef) UnmarshalTOML(data any) error {
	switch v := data.(type) {
	case map[string]any:
		if raw, ok := v["name"]; ok {
			str, ok := raw.(string)
			if !ok {
				return fmt.Errorf("dependency: \"name\" must be a string, got %T", raw)
			}
			d.Name = str
		}
		if raw, ok := v["version"]; ok {
			str, ok := raw.(string)
			if !ok {
				return fmt.Errorf("dependency: \"version\" must be a string, got %T", raw)
			}
			d.Version = str
		}
		return nil
	default:
		return fmt.Errorf("dependency: expected a table with name and optional version fields, got %T", data)
	}
}

// ProbeSpec describes how to detect if a dependency is installed.
type ProbeSpec struct {
	Command      []string `toml:"command"`
	VersionRegex string   `toml:"version_regex"`
}

// InstallMethod describes one way to install a dependency.
type InstallMethod struct {
	ID             string   `toml:"id"`
	Description    string   `toml:"description"`
	OS             string   `toml:"os"`              // "linux", "darwin", "windows"
	DistroFamilies []string `toml:"distro_families"` // empty = all distros for this OS
	Command        []string `toml:"command"`         // explicit argv (registry)
	ShellPayload   string   `toml:"shell_payload"`   // sh -c payload (custom only)
	RequiresRoot   bool     `toml:"requires_root"`
}

// ResolvedDependency represents a fully resolved dependency ready for probe/install.
type ResolvedDependency struct {
	Name    string
	Version string // semver constraint
	Probe   ProbeSpec
	Methods []InstallMethod
}

// Platform represents the current OS and distro family.
type Platform struct {
	OS           string
	DistroFamily string
}

// String returns a human-readable representation of the platform.
func (p Platform) String() string {
	if p.DistroFamily == "" {
		return p.OS
	}
	return p.OS + "/" + p.DistroFamily
}

// InstalledDependency represents a dependency recorded in state.json after install.
type InstalledDependency struct {
	Name              string    `json:"name"`
	Version           string    `json:"version"`
	Method            string    `json:"method"`
	InstalledAt       time.Time `json:"installed_at"`
	ManagedByEasyrice bool      `json:"managed_by_easyrice"`
}

// CustomInstallMethod represents the shape of [custom_dependencies.<name>.install.<key>]
// in rice.toml.
type CustomInstallMethod struct {
	Description    string   `toml:"description"`
	ShellPayload   string   `toml:"shell_payload"`
	DistroFamilies []string `toml:"distro_families"`
}

// CustomDependencyDef represents the shape of [custom_dependencies.<name>] in rice.toml.
// Defined here (not in manifest) to avoid import cycle: manifest→deps is allowed;
// deps→manifest is FORBIDDEN.
type CustomDependencyDef struct {
	VersionProbe []string                       `toml:"version_probe"`
	VersionRegex string                         `toml:"version_regex"`
	Install      map[string]CustomInstallMethod `toml:"install"`
}
