package installer

import (
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/guneet-xyz/easyrice/internal/logger"
	"github.com/guneet-xyz/easyrice/internal/plan"
	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/guneet-xyz/easyrice/internal/symlink"
)

// UninstallRequest captures all inputs needed to compute and execute an uninstall.
type UninstallRequest struct {
	PackageName string
	StatePath   string
}

// BuildUninstallPlan builds the uninstall plan WITHOUT touching the filesystem.
// Loads state, finds package's InstalledLinks, builds plan with OpRemove ops.
// Returns error if package not in state.
func BuildUninstallPlan(req UninstallRequest) (*plan.Plan, error) {
	logger.Debug("building uninstall plan",
		zap.String("package", req.PackageName),
	)

	// 1. Load state from req.StatePath
	s, err := state.Load(req.StatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	// 2. Find req.PackageName in state (error if not found)
	pkgState, ok := s[req.PackageName]
	if !ok {
		return nil, fmt.Errorf("package %q not installed", req.PackageName)
	}

	// 3. Build []plan.Op{Kind: plan.OpRemove, Target: link.Target} for each InstalledLink
	ops := make([]plan.Op, 0, len(pkgState.InstalledLinks))
	for _, link := range pkgState.InstalledLinks {
		ops = append(ops, plan.Op{
			Kind:   plan.OpRemove,
			Target: link.Target,
			Source: link.Source, // Keep source for reference during execution
		})
	}

	// 4. Return plan.Plan{PackageName: req.PackageName, Profile: pkgState.Profile, Ops: ops}
	p := &plan.Plan{
		PackageName: req.PackageName,
		Profile:     pkgState.Profile,
		Ops:         ops,
		Conflicts:   []plan.Conflict{},
	}

	logger.Debug("uninstall plan built",
		zap.String("package", req.PackageName),
		zap.Int("ops", len(ops)),
	)

	return p, nil
}

// ExecuteUninstallPlan removes symlinks per the plan.
// For each op: verify it's still our symlink (IsSymlinkTo), then remove.
// If link is missing or replaced: log WARN and continue (don't error).
// After processing: remove package entry from state and save.
func ExecuteUninstallPlan(p *plan.Plan, statePath string) error {
	logger.Info("uninstalling package",
		zap.String("package", p.PackageName),
	)

	// 1. Load state from statePath
	s, err := state.Load(statePath)
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// 2. For each Op (OpRemove):
	removed := 0
	skipped := 0

	for _, op := range p.Ops {
		if op.Kind != plan.OpRemove {
			continue
		}

		// Find the corresponding InstalledLink to get Source and IsDir
		pkgState, ok := s[p.PackageName]
		if !ok {
			// Package already removed from state (shouldn't happen, but be defensive)
			logger.Warn("package not in state during execution",
				zap.String("package", p.PackageName),
				zap.String("target", op.Target),
			)
			skipped++
			continue
		}

		var source string
		var isDir bool
		for _, link := range pkgState.InstalledLinks {
			if link.Target == op.Target {
				source = link.Source
				isDir = link.IsDir
				break
			}
		}

		// Safety check for folder symlinks: verify target is still a symlink
		if isDir {
			fi, err := os.Lstat(op.Target)
			if err != nil {
				// Error checking symlink (permission denied, etc.)
				logger.Warn("failed to check symlink",
					zap.String("target", op.Target),
					zap.Error(err),
				)
				skipped++
				continue
			}

			// If it's NOT a symlink (i.e., replaced with a real directory), skip
			if fi.Mode()&os.ModeSymlink == 0 {
				logger.Warn("folder symlink replaced with real directory, skipping removal",
					zap.String("target", op.Target),
				)
				skipped++
				continue
			}
		}

		// Call symlink.IsSymlinkTo(op.Target, source)
		isOurs, err := symlink.IsSymlinkTo(op.Target, source)
		if err != nil {
			// Error checking symlink (permission denied, etc.)
			logger.Warn("failed to check symlink",
				zap.String("target", op.Target),
				zap.Error(err),
			)
			skipped++
			continue
		}

		if isOurs {
			// Call symlink.RemoveSymlink(op.Target)
			if err := symlink.RemoveSymlink(op.Target); err != nil {
				logger.Warn("failed to remove symlink",
					zap.String("target", op.Target),
					zap.Error(err),
				)
				skipped++
				continue
			}
			logger.Debug("removed symlink",
				zap.String("target", op.Target),
			)
			removed++
		} else {
			// Link is missing, replaced by file, or points elsewhere
			logger.Warn("symlink drift detected",
				zap.String("target", op.Target),
				zap.String("expected_source", source),
				zap.String("reason", "link missing, replaced, or points elsewhere"),
			)
			skipped++
		}
	}

	// 3. Remove package entry from state: delete(s, p.PackageName)
	// Deps are NOT removed from the system; only the state tracking is cleared.
	delete(s, p.PackageName)

	// 4. Save state
	if err := state.Save(statePath, s); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	// 5. Log summary and return nil
	logger.Info("uninstalled package",
		zap.String("package", p.PackageName),
		zap.Int("removed", removed),
		zap.Int("skipped", skipped),
	)

	return nil
}

// Uninstall is a convenience wrapper for tests.
// Builds and executes the uninstall plan in one call.
func Uninstall(req UninstallRequest) error {
	p, err := BuildUninstallPlan(req)
	if err != nil {
		return err
	}

	return ExecuteUninstallPlan(p, req.StatePath)
}
