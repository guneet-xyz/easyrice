// Helpers for inspecting manifest packages.
package manifest

import "sort"

// SortedProfileNames returns the profile names of pkg in lexicographic order.
// The manifest validator guarantees len(pkg.Profiles) >= 1 for any successfully-loaded
// manifest (see validate.go), so callers can rely on a non-empty slice.
func SortedProfileNames(pkg PackageDef) []string {
	names := make([]string, 0, len(pkg.Profiles))
	for name := range pkg.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
