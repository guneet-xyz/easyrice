package deps

import (
	"fmt"
	"strings"
)

// Resolve maps DependencyRef entries to ResolvedDependency entries.
// For each ref:
//   - If RegistryDep(ref.Name) exists → use registry entry; override Version with ref.Version
//   - Else if custom[ref.Name] exists → build ResolvedDependency from CustomDependencyDef
//   - Else → return error "unknown dependency %q (not in registry, not declared in custom_dependencies)"
//
// Preserves input order in output slice.
func Resolve(refs []DependencyRef, custom map[string]CustomDependencyDef) ([]ResolvedDependency, error) {
	result := make([]ResolvedDependency, 0, len(refs))
	for _, ref := range refs {
		if reg, ok := RegistryDep(ref.Name); ok {
			reg.Version = ref.Version // override constraint
			result = append(result, reg)
			continue
		}
		if customDef, ok := custom[ref.Name]; ok {
			resolved := buildFromCustom(ref.Name, ref.Version, customDef)
			result = append(result, resolved)
			continue
		}
		return nil, fmt.Errorf("unknown dependency %q (not in registry, not declared in custom_dependencies)", ref.Name)
	}
	return result, nil
}

// buildFromCustom converts a CustomDependencyDef into a ResolvedDependency.
func buildFromCustom(name, version string, def CustomDependencyDef) ResolvedDependency {
	probe := ProbeSpec{
		Command:      def.VersionProbe,
		VersionRegex: def.VersionRegex,
	}
	methods := make([]InstallMethod, 0, len(def.Install))
	for key, m := range def.Install {
		methods = append(methods, InstallMethod{
			ID:             key,
			Description:    m.Description,
			ShellPayload:   m.ShellPayload,
			DistroFamilies: m.DistroFamilies,
			// OS is derived from the key convention (e.g., "linux_debian", "darwin")
			// Parse: if key starts with "linux" → OS="linux"; "darwin" → OS="darwin"; "windows" → OS="windows"
			OS: parseOSFromKey(key),
		})
	}
	return ResolvedDependency{
		Name:    name,
		Version: version,
		Probe:   probe,
		Methods: methods,
	}
}

// parseOSFromKey extracts OS from a method key like "linux_debian", "darwin", "windows".
func parseOSFromKey(key string) string {
	if strings.HasPrefix(key, "linux") {
		return "linux"
	}
	if strings.HasPrefix(key, "darwin") {
		return "darwin"
	}
	if strings.HasPrefix(key, "windows") {
		return "windows"
	}
	return ""
}

// FilterByPlatform returns only the deps that have at least one install method
// matching the given platform. Methods that don't match are dropped from each dep.
// If a dep has zero methods after filtering (and had methods to begin with),
// returns an error.
func FilterByPlatform(deps []ResolvedDependency, p Platform) ([]ResolvedDependency, error) {
	result := make([]ResolvedDependency, 0, len(deps))
	for _, dep := range deps {
		filtered := filterMethods(dep.Methods, p)
		if len(dep.Methods) > 0 && len(filtered) == 0 {
			return nil, fmt.Errorf("dependency %q has no install method for %s", dep.Name, p)
		}
		dep.Methods = filtered
		result = append(result, dep)
	}
	return result, nil
}

// filterMethods returns methods matching the platform.
func filterMethods(methods []InstallMethod, p Platform) []InstallMethod {
	var out []InstallMethod
	for _, m := range methods {
		if m.OS != p.OS {
			continue
		}
		if p.OS == "linux" && len(m.DistroFamilies) > 0 {
			if !containsString(m.DistroFamilies, p.DistroFamily) {
				continue
			}
		}
		out = append(out, m)
	}
	return out
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
