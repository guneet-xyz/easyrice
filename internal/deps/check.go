package deps

import (
	"context"
	"fmt"
)

// DepStatus represents the status of a dependency check.
type DepStatus int

const (
	DepOK DepStatus = iota
	DepMissing
	DepVersionMismatch
	DepProbeUnknownVersion
)

// String returns a human-readable representation of the status.
func (s DepStatus) String() string {
	switch s {
	case DepOK:
		return "OK"
	case DepMissing:
		return "Missing"
	case DepVersionMismatch:
		return "VersionMismatch"
	case DepProbeUnknownVersion:
		return "ProbeUnknownVersion"
	default:
		return "Unknown"
	}
}

// DepReportEntry represents the result of checking a single dependency.
type DepReportEntry struct {
	Dep              ResolvedDependency
	Installed        bool
	InstalledVersion string
	Status           DepStatus
	Methods          []InstallMethod // pre-filtered to platform
}

// DepReport represents the result of checking all dependencies.
type DepReport struct {
	Entries []DepReportEntry
}

// Missing returns all entries with DepMissing status.
func (r DepReport) Missing() []DepReportEntry {
	var result []DepReportEntry
	for _, entry := range r.Entries {
		if entry.Status == DepMissing {
			result = append(result, entry)
		}
	}
	return result
}

// Mismatched returns all entries with DepVersionMismatch status.
func (r DepReport) Mismatched() []DepReportEntry {
	var result []DepReportEntry
	for _, entry := range r.Entries {
		if entry.Status == DepVersionMismatch {
			result = append(result, entry)
		}
	}
	return result
}

// NeedsAction returns true if any entry is DepMissing or DepVersionMismatch.
func (r DepReport) NeedsAction() bool {
	for _, entry := range r.Entries {
		if entry.Status == DepMissing || entry.Status == DepVersionMismatch {
			return true
		}
	}
	return false
}

// Check orchestrates the dependency check process.
// It resolves dependencies, filters by platform, probes each one, and returns a report.
// Returns an error if:
//   - Resolve fails
//   - FilterByPlatform fails
//   - Probe returns an error (aborts the whole check)
func Check(ctx context.Context, runner Runner, refs []DependencyRef, custom map[string]CustomDependencyDef, platform Platform) (DepReport, error) {
	// Resolve dependencies
	resolved, err := Resolve(refs, custom)
	if err != nil {
		return DepReport{}, fmt.Errorf("check: resolve: %w", err)
	}

	// Filter by platform
	filtered, err := FilterByPlatform(resolved, platform)
	if err != nil {
		return DepReport{}, fmt.Errorf("check: filter by platform: %w", err)
	}

	// Probe each dependency
	var entries []DepReportEntry
	for _, dep := range filtered {
		installed, version, err := Probe(ctx, runner, dep)
		if err != nil {
			return DepReport{}, fmt.Errorf("check: probe %q: %w", dep.Name, err)
		}

		// Determine status
		var status DepStatus
		if !installed {
			status = DepMissing
		} else if version == "" {
			status = DepProbeUnknownVersion
		} else {
			// Version captured; check constraint
			matches, err := MatchVersion(version, dep.Version)
			if err != nil {
				return DepReport{}, fmt.Errorf("check: match version for %q: %w", dep.Name, err)
			}
			if !matches {
				status = DepVersionMismatch
			} else {
				status = DepOK
			}
		}

		entry := DepReportEntry{
			Dep:              dep,
			Installed:        installed,
			InstalledVersion: version,
			Status:           status,
			Methods:          dep.Methods,
		}
		entries = append(entries, entry)
	}

	return DepReport{Entries: entries}, nil
}
