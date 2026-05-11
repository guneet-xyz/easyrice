package manifest

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

func LoadFile(path string) (*Manifest, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("rice.toml not found at %q", path)
		}
		return nil, fmt.Errorf("failed to stat rice.toml: %w", err)
	}

	var m Manifest
	if _, err := toml.DecodeFile(path, &m); err != nil {
		return nil, fmt.Errorf("failed to parse rice.toml: %w", err)
	}

	if err := Validate(&m); err != nil {
		return nil, fmt.Errorf("manifest validation failed: %w", err)
	}

	return &m, nil
}
