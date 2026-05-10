package installer

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/guneet-xyz/easyrice/internal/logger"
	"github.com/guneet-xyz/easyrice/internal/plan"
	"github.com/guneet-xyz/easyrice/internal/state"
)

// SwitchRequest captures all inputs needed to compute and execute a profile switch.
type SwitchRequest struct {
	RepoRoot    string
	PackageName string
	NewProfile  string
	CurrentOS   string
	HomeDir     string
	StatePath   string
}

// SwitchPlan holds both the uninstall and install plans for a profile switch.
type SwitchPlan struct {
	Uninstall *plan.Plan
	Install   *plan.Plan
}

// BuildSwitchPlan builds the switch plan WITHOUT touching the filesystem.
// Returns error if:
//   - package not currently installed
//   - new profile doesn't exist in manifest
//   - pre-flight conflict detected (plan returned with conflicts populated)
func BuildSwitchPlan(req SwitchRequest) (*SwitchPlan, error) {
	logger.Debug("BuildSwitchPlan: start",
		zap.String("package", req.PackageName),
		zap.String("newProfile", req.NewProfile),
	)

	// 1. Load state to verify package is installed and discover current profile.
	st, err := state.Load(req.StatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}
	pkgState, ok := st[req.PackageName]
	if !ok {
		return nil, fmt.Errorf("package %q not installed; use `rice install` instead", req.PackageName)
	}
	oldProfile := pkgState.Profile

	logger.Info("Switching package profile",
		zap.String("package", req.PackageName),
		zap.String("from", oldProfile),
		zap.String("to", req.NewProfile),
	)

	// 2. Build uninstall plan from current state.
	uninstallPlan, err := BuildUninstallPlan(UninstallRequest{
		PackageName: req.PackageName,
		StatePath:   req.StatePath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build uninstall plan: %w", err)
	}

	// 3. Build install plan for new profile (without conflict suppression).
	// BuildInstallPlan internally calls DetectConflicts(planned, nil), so it may
	// return an error if old links would conflict. We re-run conflict detection
	// below with ignoreTargets to filter those out.
	installPlan, installErr := BuildInstallPlan(InstallRequest{
		RepoRoot:    req.RepoRoot,
		PackageName: req.PackageName,
		Profile:     req.NewProfile,
		CurrentOS:   req.CurrentOS,
		HomeDir:     req.HomeDir,
		StatePath:   req.StatePath,
	})
	if installPlan == nil {
		// True build failure (manifest, profile, etc.) — installPlan is nil
		return nil, fmt.Errorf("failed to build install plan: %w", installErr)
	}

	// 4. Pre-flight conflict re-check: ignore targets being uninstalled.
	ignoreTargets := make(map[string]struct{}, len(uninstallPlan.Ops))
	for _, op := range uninstallPlan.Ops {
		ignoreTargets[op.Target] = struct{}{}
	}

	planned := make([]PlannedLink, 0, len(installPlan.Ops))
	for _, op := range installPlan.Ops {
		if op.Kind != plan.OpCreate {
			continue
		}
		planned = append(planned, PlannedLink{Source: op.Source, Target: op.Target, IsDir: op.IsDir})
	}
	conflicts := DetectConflicts(planned, ignoreTargets)

	installPlan.Conflicts = nil
	for _, c := range conflicts {
		installPlan.Conflicts = append(installPlan.Conflicts, plan.Conflict{
			Target: c.Target,
			Source: c.Source,
			Reason: c.Reason,
			IsDir:  c.IsDir,
		})
	}

	logger.Debug("BuildSwitchPlan: pre-flight summary",
		zap.Int("oldLinksToRemove", len(uninstallPlan.Ops)),
		zap.Int("newLinksToCreate", len(installPlan.Ops)),
		zap.Int("conflictsAfterIgnore", len(installPlan.Conflicts)),
	)

	sp := &SwitchPlan{Uninstall: uninstallPlan, Install: installPlan}

	if len(installPlan.Conflicts) > 0 {
		logger.Error("Pre-flight conflicts detected for switch",
			zap.String("package", req.PackageName),
			zap.Int("conflicts", len(installPlan.Conflicts)),
		)
		return sp, fmt.Errorf("pre-flight conflicts detected: %d", len(installPlan.Conflicts))
	}

	return sp, nil
}

// ExecuteSwitchPlan executes: uninstall then install.
// If install fails after uninstall succeeds, logs ERROR with recovery message.
func ExecuteSwitchPlan(p *SwitchPlan, statePath string) error {
	if p == nil || p.Uninstall == nil || p.Install == nil {
		return fmt.Errorf("invalid switch plan")
	}

	logger.Info("Executing switch: uninstalling old profile",
		zap.String("package", p.Uninstall.PackageName),
		zap.String("oldProfile", p.Uninstall.Profile),
	)

	if err := ExecuteUninstallPlan(p.Uninstall, statePath); err != nil {
		return fmt.Errorf("switch uninstall phase failed: %w", err)
	}

	logger.Info("Executing switch: installing new profile",
		zap.String("package", p.Install.PackageName),
		zap.String("newProfile", p.Install.Profile),
	)

	if _, err := ExecuteInstallPlan(p.Install, statePath); err != nil {
		logger.Error("Switch install phase failed after uninstall succeeded",
			zap.String("package", p.Install.PackageName),
			zap.String("recovery", fmt.Sprintf("run `rice install %s --profile %s` to recover",
				p.Install.PackageName, p.Install.Profile)),
			zap.Error(err),
		)
		return fmt.Errorf("switch left package %q uninstalled; run `rice install %s --profile %s` to recover: %w",
			p.Install.PackageName, p.Install.PackageName, p.Install.Profile, err)
	}

	logger.Info("Switch complete",
		zap.String("package", p.Install.PackageName),
		zap.String("profile", p.Install.Profile),
	)
	return nil
}

// Switch is a convenience wrapper for tests.
// CLI layer should call BuildSwitchPlan and ExecuteSwitchPlan separately
// to insert a confirmation prompt between them.
func Switch(req SwitchRequest) error {
	sp, err := BuildSwitchPlan(req)
	if err != nil {
		return err
	}
	return ExecuteSwitchPlan(sp, req.StatePath)
}
