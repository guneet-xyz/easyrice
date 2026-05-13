package installer

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/guneet-xyz/easyrice/internal/deps"
	"github.com/guneet-xyz/easyrice/internal/logger"
	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/prompt"
	"github.com/guneet-xyz/easyrice/internal/state"
)

// EnsureDependencies checks and installs dependencies for a package.
// It performs the following steps:
//  1. Look up the package definition
//  2. Detect the current platform
//  3. Check all dependencies
//  4. If all OK, log and return unchanged state
//  5. If version mismatch, abort with error
//  6. For missing dependencies, prompt user to select install method
//  7. If user accepts, execute installs and update state
//
// Returns the updated state (caller persists to state.json).
// On error, returns the original state unchanged.
func EnsureDependencies(ctx context.Context, runner deps.Runner, m manifest.Manifest, pkgName string, autoAccept bool, s state.State) (state.State, error) {
	// Look up package definition
	pkg, ok := m.Packages[pkgName]
	if !ok {
		return s, fmt.Errorf("ensure dependencies: package %q not found in manifest", pkgName)
	}

	// Detect platform
	platform := deps.Detect()

	// Check all dependencies
	report, err := deps.Check(ctx, runner, pkg.Dependencies, m.CustomDependencies, platform)
	if err != nil {
		return s, fmt.Errorf("ensure dependencies: check failed: %w", err)
	}

	// If no action needed, log and return
	if !report.NeedsAction() {
		logger.L.Info("all dependencies OK",
			zap.String("package", pkgName),
		)
		return s, nil
	}

	// Render the dependency report
	prompt.RenderDepReport(os.Stdout, report)

	// Check for version mismatches — abort if found
	if len(report.Mismatched()) > 0 {
		fmt.Fprintf(os.Stdout, "Cannot continue because one or more dependencies have the wrong version. Upgrade them manually, then run easyrice again.\n")
		return s, fmt.Errorf("ensure dependencies: version mismatch detected for package %q", pkgName)
	}

	// Collect install choices for missing dependencies
	var choices []deps.InstallChoice
	for _, entry := range report.Missing() {
		method, err := prompt.SelectInstallMethod(os.Stdin, os.Stdout, entry, autoAccept)
		if err != nil {
			if err == prompt.ErrUserDeclined {
				return s, fmt.Errorf("ensure dependencies: user declined to install %q", entry.Dep.Name)
			}
			return s, fmt.Errorf("ensure dependencies: select install method for %q: %w", entry.Dep.Name, err)
		}
		choices = append(choices, deps.InstallChoice{
			Dep:    entry.Dep,
			Method: method,
		})
	}

	// If we have choices and not auto-accepting, confirm with user
	if len(choices) > 0 && !autoAccept {
		ok, err := prompt.Confirm(os.Stdin, os.Stdout, fmt.Sprintf("Install %d dependency(ies) listed above?", len(choices)))
		if err != nil {
			return s, fmt.Errorf("ensure dependencies: confirm prompt: %w", err)
		}
		if !ok {
			return s, fmt.Errorf("ensure dependencies: user declined to install dependencies for package %q", pkgName)
		}
	}

	// Execute installs
	outcomes, err := deps.Install(ctx, runner, choices)
	if err != nil {
		return s, fmt.Errorf("ensure dependencies: install failed: %w", err)
	}

	if s == nil {
		s = make(state.State)
	}

	pkgState, ok := s[pkgName]
	if !ok {
		pkgState = state.PackageState{}
	}

	existingByName := make(map[string]int)
	for i, dep := range pkgState.InstalledDependencies {
		existingByName[dep.Name] = i
	}

	for _, outcome := range outcomes {
		if idx, found := existingByName[outcome.Installed.Name]; found {
			pkgState.InstalledDependencies[idx] = outcome.Installed
		} else {
			pkgState.InstalledDependencies = append(pkgState.InstalledDependencies, outcome.Installed)
		}
	}

	s[pkgName] = pkgState

	// Log DepProbeUnknownVersion entries (installed but version unknown)
	for _, entry := range report.Entries {
		if entry.Status == deps.DepProbeUnknownVersion {
			logger.L.Warn("dependency installed but version unknown",
				zap.String("package", pkgName),
				zap.String("dependency", entry.Dep.Name),
			)
		}
	}

	return s, nil
}
