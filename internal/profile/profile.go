package profile

import (
	"fmt"
	"sort"
	"strings"

	"github.com/guneet-xyz/easyrice/internal/manifest"
)

// ResolveSpecs returns the ordered list of SourceSpec entries for the given
// profile name, preserving Mode and Target fields needed for folder-mode
// installs.
func ResolveSpecs(m *manifest.Manifest, profileName string) ([]manifest.SourceSpec, error) {
	profile, exists := m.Profiles[profileName]
	if !exists {
		available := make([]string, 0, len(m.Profiles))
		for name := range m.Profiles {
			available = append(available, name)
		}
		sort.Strings(available)

		availableStr := strings.Join(available, ", ")
		return nil, fmt.Errorf("profile %q not defined in package %q; available: %s", profileName, m.Name, availableStr)
	}

	specs := make([]manifest.SourceSpec, len(profile.Sources))
	copy(specs, profile.Sources)
	return specs, nil
}
