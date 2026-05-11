// Package manifest provides rice.toml schema definitions and parsing.
package manifest

import (
	"fmt"

	"github.com/guneet-xyz/easyrice/internal/deps"
)

// Manifest represents the root-level rice.toml file structure.
// It contains a schema version and a map of packages, each with their own
// configuration, profiles, and sources.
type Manifest struct {
	SchemaVersion      int                                `toml:"schema_version"`
	Packages           map[string]PackageDef              `toml:"packages"`
	CustomDependencies map[string]deps.CustomDependencyDef `toml:"custom_dependencies"`
}

// PackageDef represents a single package definition within the root rice.toml.
// Description and SupportedOS are package-level metadata.
// Root is optional; if empty, callers MUST default it to the package name.
type PackageDef struct {
	Description  string                `toml:"description"`
	SupportedOS  []string              `toml:"supported_os"`
	Root         string                `toml:"root"`
	Profiles     map[string]ProfileDef `toml:"profiles"`
	Dependencies []deps.DependencyRef  `toml:"dependencies"`
}

// ProfileDef represents a single profile within a PackageDef.
type ProfileDef struct {
	Sources []SourceSpec `toml:"sources"`
}

// SourceSpec describes a single source entry under a profile. It accepts only
// the inline table form:
//
//	sources = [{path = "foo", mode = "folder", target = ".config/nvim"}]
//	-> SourceSpec{Path: "foo", Mode: "folder", Target: ".config/nvim"}
//
// All three fields (path, mode, target) are required. "folder" mode opts the
// entire source directory into being symlinked as a single directory link rather
// than walked file-by-file; in that case Target is interpreted relative to the
// package's target root.
type SourceSpec struct {
	Path   string `toml:"path"`
	Mode   string `toml:"mode"`
	Target string `toml:"target"`
}

// UnmarshalTOML implements the toml.Unmarshaler interface from
// github.com/BurntSushi/toml. It accepts only the table form.
func (s *SourceSpec) UnmarshalTOML(data interface{}) error {
	switch v := data.(type) {
	case map[string]interface{}:
		if raw, ok := v["path"]; ok {
			str, ok := raw.(string)
			if !ok {
				return fmt.Errorf("source: \"path\" must be a string, got %T", raw)
			}
			s.Path = str
		}
		if raw, ok := v["mode"]; ok {
			str, ok := raw.(string)
			if !ok {
				return fmt.Errorf("source: \"mode\" must be a string, got %T", raw)
			}
			s.Mode = str
		}
		if raw, ok := v["target"]; ok {
			str, ok := raw.(string)
			if !ok {
				return fmt.Errorf("source: \"target\" must be a string, got %T", raw)
			}
			s.Target = str
		}
		return nil
	default:
		return fmt.Errorf("source: expected a table with path, mode, and target fields, got %T", data)
	}
}
