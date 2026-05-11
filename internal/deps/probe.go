package deps

import (
	"context"
	"fmt"
	"regexp"

	"github.com/guneet-xyz/easyrice/internal/logger"
	"go.uber.org/zap"
)

// Probe checks whether dep is installed and returns its version.
// Returns:
//   - (false, "", nil) if the probe command exits non-zero (not installed)
//   - (true, "", nil) if installed but version regex doesn't match (unknown version)
//   - (true, version, nil) if installed and version captured
//   - (false, "", err) if probe command is empty or context cancelled
func Probe(ctx context.Context, runner Runner, dep ResolvedDependency) (installed bool, version string, err error) {
	if len(dep.Probe.Command) == 0 {
		return false, "", fmt.Errorf("probe: no probe command defined for %q", dep.Name)
	}

	logger.L.Debug("probing dependency", zap.String("dep", dep.Name), zap.Strings("cmd", dep.Probe.Command))

	result, err := runner.Run(ctx, dep.Probe.Command, RunOpts{CombinedOutput: true})
	if err != nil {
		return false, "", fmt.Errorf("probe %q: %w", dep.Name, err)
	}

	if result.ExitCode != 0 {
		logger.L.Debug("probe: not installed", zap.String("dep", dep.Name), zap.Int("exit", result.ExitCode))
		return false, "", nil
	}

	// Installed — try to extract version
	if dep.Probe.VersionRegex == "" {
		return true, "", nil
	}

	re, err := regexp.Compile(dep.Probe.VersionRegex)
	if err != nil {
		return true, "", fmt.Errorf("probe %q: invalid version regex %q: %w", dep.Name, dep.Probe.VersionRegex, err)
	}

	matches := re.FindSubmatch(result.Combined)
	if len(matches) < 2 {
		logger.L.Debug("probe: installed, version unknown", zap.String("dep", dep.Name))
		return true, "", nil
	}

	return true, string(matches[1]), nil
}
