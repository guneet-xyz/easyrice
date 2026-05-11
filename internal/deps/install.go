package deps

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/guneet-xyz/easyrice/internal/logger"
	"go.uber.org/zap"
)

// InstallChoice represents a dependency and the chosen method to install it.
type InstallChoice struct {
	Dep    ResolvedDependency
	Method InstallMethod
}

// InstallOutcome represents the result of installing a single dependency.
type InstallOutcome struct {
	Dep       ResolvedDependency
	Method    InstallMethod
	Installed InstalledDependency // result for state recording
}

// Install executes a list of install choices sequentially.
// For each choice:
//   - Checks if the method requires root; aborts if not running as root
//   - Executes the install command (registry method) or shell payload (custom method)
//   - Re-probes to capture the installed version
//   - Returns outcomes accumulated so far PLUS the first error (if any)
//
// Non-zero exit from the install command is treated as fatal.
// If post-install probe shows not-installed, returns an error.
func Install(ctx context.Context, runner Runner, choices []InstallChoice) ([]InstallOutcome, error) {
	var outcomes []InstallOutcome

	for _, choice := range choices {
		dep := choice.Dep
		method := choice.Method

		logger.L.Info("installing dependency",
			zap.String("dep", dep.Name),
			zap.String("method", method.ID),
		)

		// Check if method requires root
		if method.RequiresRoot && os.Geteuid() != 0 {
			err := fmt.Errorf(
				"dependency %q requires root for method %q (command: %s) — re-run with sudo, install manually, or pick a non-root method",
				dep.Name,
				method.ID,
				formatCommand(method),
			)
			return outcomes, err
		}

		// Execute the install command
		var result RunResult
		var err error

		if method.Command != nil {
			logger.L.Debug("running install command",
				zap.String("dep", dep.Name),
				zap.Strings("cmd", method.Command),
			)
			result, err = runner.Run(ctx, method.Command, RunOpts{})
			if err != nil {
				return outcomes, fmt.Errorf("installing %q via %q: %w", dep.Name, method.ID, err)
			}
			if result.ExitCode != 0 {
				return outcomes, fmt.Errorf(
					"installing %q via %q failed (exit %d): %s",
					dep.Name,
					method.ID,
					result.ExitCode,
					string(result.Stderr),
				)
			}
		} else if method.ShellPayload != "" {
			logger.L.Debug("running shell payload",
				zap.String("dep", dep.Name),
			)
			result, err = RunShell(ctx, runner, method.ShellPayload)
			if err != nil {
				return outcomes, fmt.Errorf("installing %q via %q: %w", dep.Name, method.ID, err)
			}
			if result.ExitCode != 0 {
				return outcomes, fmt.Errorf(
					"installing %q via %q failed (exit %d): %s",
					dep.Name,
					method.ID,
					result.ExitCode,
					string(result.Combined),
				)
			}
		} else {
			return outcomes, fmt.Errorf("install method %q for %q has neither command nor shell_payload", method.ID, dep.Name)
		}

		// Re-probe to capture installed version
		installed, version, err := Probe(ctx, runner, dep)
		if err != nil {
			return outcomes, fmt.Errorf("post-install probe for %q: %w", dep.Name, err)
		}

		if !installed {
			return outcomes, fmt.Errorf(
				"install completed with exit 0 but probe still reports missing for %q",
				dep.Name,
			)
		}

		// Build outcome
		outcome := InstallOutcome{
			Dep:    dep,
			Method: method,
			Installed: InstalledDependency{
				Name:              dep.Name,
				Version:           version,
				Method:            method.ID,
				InstalledAt:       time.Now(),
				ManagedByEasyrice: true,
			},
		}
		outcomes = append(outcomes, outcome)

		logger.L.Info("dependency installed",
			zap.String("dep", dep.Name),
			zap.String("version", version),
			zap.String("method", method.ID),
		)
	}

	return outcomes, nil
}

// formatCommand returns a human-readable representation of the install command.
func formatCommand(method InstallMethod) string {
	if method.Command != nil {
		return fmt.Sprintf("%v", method.Command)
	}
	if method.ShellPayload != "" {
		return fmt.Sprintf("sh -c %q", method.ShellPayload)
	}
	return "(no command)"
}
