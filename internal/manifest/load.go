package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Load reads and parses the rice.toml at dir/rice.toml.
// Validates the manifest after parsing.
// Returns error if file missing, parse fails, or validation fails.
func Load(dir string) (*Manifest, error) {
	filePath := filepath.Join(dir, "rice.toml")

	// Check if file exists
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("rice.toml not found in %q", dir)
		}
		return nil, fmt.Errorf("failed to stat rice.toml: %w", err)
	}

	// Decode TOML file
	var m Manifest
	if _, err := toml.DecodeFile(filePath, &m); err != nil {
		return nil, fmt.Errorf("failed to parse rice.toml: %w", err)
	}

	// Validate the manifest
	if err := Validate(&m); err != nil {
		return nil, fmt.Errorf("manifest validation failed: %w", err)
	}

	return &m, nil
}

// Discover walks repoRoot looking for rice.toml files (one level deep only —
// each package is a direct subdirectory of repoRoot).
// Returns a map of packageName -> *Manifest for all valid manifests found.
// Skips directories that have no rice.toml (silently).
// Returns error only if a rice.toml is found but fails to parse/validate.
func Discover(repoRoot string) (map[string]*Manifest, error) {
	result := make(map[string]*Manifest)

	// List direct subdirectories of repoRoot
	entries, err := os.ReadDir(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to read repoRoot: %w", err)
	}

	for _, entry := range entries {
		// Skip non-directory entries
		if !entry.IsDir() {
			continue
		}

		packageName := entry.Name()
		packageDir := filepath.Join(repoRoot, packageName)

		// Check if rice.toml exists in this directory
		ricePath := filepath.Join(packageDir, "rice.toml")
		if _, err := os.Stat(ricePath); err != nil {
			if os.IsNotExist(err) {
				// Silently skip directories without rice.toml
				continue
			}
			// Return error for other stat errors
			return nil, fmt.Errorf("failed to stat rice.toml in %q: %w", packageDir, err)
		}

		// Load and validate the manifest
		manifest, err := Load(packageDir)
		if err != nil {
			return nil, fmt.Errorf("failed to load manifest from %q: %w", packageDir, err)
		}

		result[packageName] = manifest
	}

	return result, nil
}
