package manifest

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"

	"github.com/BurntSushi/toml"
)

const currentSchemaVersion = 1

// duplicateKeyRE captures the key name from BurntSushi/toml's
// "Key 'a.b.c' has already been defined" error so we can rephrase it.
var duplicateKeyRE = regexp.MustCompile(`Key '([^']+)' has already been defined`)

func LoadFile(path string) (*Manifest, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("rice.toml not found at %q", path)
		}
		if errors.Is(err, fs.ErrPermission) {
			return nil, fmt.Errorf("rice.toml at %q: permission denied: %w", path, err)
		}
		return nil, fmt.Errorf("failed to stat rice.toml at %q: %w", path, err)
	}

	if _, err := os.ReadFile(path); err != nil {
		if errors.Is(err, fs.ErrPermission) {
			return nil, fmt.Errorf("rice.toml at %q: permission denied: %w", path, err)
		}
		return nil, fmt.Errorf("failed to read rice.toml at %q: %w", path, err)
	}

	var m Manifest
	meta, err := toml.DecodeFile(path, &m)
	if err != nil {
		if match := duplicateKeyRE.FindStringSubmatch(err.Error()); match != nil {
			return nil, fmt.Errorf("rice.toml: duplicate key %q: %w", match[1], err)
		}
		return nil, fmt.Errorf("failed to parse rice.toml: %w", err)
	}

	if !meta.IsDefined("schema_version") {
		return nil, fmt.Errorf("manifest validation failed: schema_version is required")
	}

	if m.SchemaVersion != currentSchemaVersion {
		return nil, fmt.Errorf("manifest validation failed: unsupported schema_version %d (this binary supports %d)", m.SchemaVersion, currentSchemaVersion)
	}

	if err := Validate(&m); err != nil {
		return nil, fmt.Errorf("manifest validation failed: %w", err)
	}

	return &m, nil
}
