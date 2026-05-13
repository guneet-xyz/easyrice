package installer

import (
	"errors"
	"fmt"

	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/plan"
	"github.com/guneet-xyz/easyrice/internal/profile"
	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/guneet-xyz/easyrice/internal/symlink"
)

// ConvergeOutcome describes the result of a converge operation.
type ConvergeOutcome int

const (
	OutcomeNoOp ConvergeOutcome = iota
	OutcomeInstalled
	OutcomeProfileSwitched
	OutcomeRepaired
)

// ConvergeRequest captures all inputs for a single-package converge.
type ConvergeRequest struct {
	RepoRoot         string
	PackageName      string
	RequestedProfile string
	CurrentOS        string
	HomeDir          string
	StatePath        string
	Pkg              *manifest.PackageDef
	Manifest         *manifest.Manifest
}

// ConvergeResult describes what happened during a converge.
type ConvergeResult struct {
	PackageName string
	Outcome     ConvergeOutcome
	OldProfile  string
	NewProfile  string
	Plan        *plan.Plan
	LinksAfter  []state.InstalledLink
}

// BuildConvergePlan computes what needs to happen for a package to reach its
// desired state without touching the filesystem.
func BuildConvergePlan(req ConvergeRequest) (*ConvergeResult, error) {
	if req.Pkg == nil {
		return nil, fmt.Errorf("converge request: Pkg must not be nil for package %q", req.PackageName)
	}

	st, err := state.Load(req.StatePath)
	if err != nil {
		return nil, fmt.Errorf("load state: %w", err)
	}

	pkgState, installed := st[req.PackageName]

	targetProfile := req.RequestedProfile
	if targetProfile == "" && installed {
		targetProfile = pkgState.Profile
	}
	if targetProfile == "" {
		return nil, fmt.Errorf("package %q: no profile specified and no stored profile", req.PackageName)
	}

	if !installed {
		specs, err := profile.ResolveSpecs(req.RepoRoot, req.Pkg, req.PackageName, targetProfile)
		if err != nil {
			return nil, fmt.Errorf("resolve profile %q: %w", targetProfile, err)
		}
		installPlan, err := BuildInstallPlan(InstallRequest{
			RepoRoot:    req.RepoRoot,
			PackageName: req.PackageName,
			ProfileName: targetProfile,
			Pkg:         req.Pkg,
			Specs:       specs,
			CurrentOS:   req.CurrentOS,
			HomeDir:     req.HomeDir,
			StatePath:   req.StatePath,
		})
		if err != nil {
			return nil, fmt.Errorf("build install plan: %w", err)
		}
		return &ConvergeResult{
			PackageName: req.PackageName,
			Outcome:     OutcomeInstalled,
			NewProfile:  targetProfile,
			Plan:        installPlan,
		}, nil
	}

	if targetProfile != pkgState.Profile {
		// Profile switch: uninstall old, install new
		uninstallPlan, err := BuildUninstallPlan(UninstallRequest{
			PackageName: req.PackageName,
			StatePath:   req.StatePath,
		})
		if err != nil {
			return nil, fmt.Errorf("build uninstall plan: %w", err)
		}

		specs, err := profile.ResolveSpecs(req.RepoRoot, req.Pkg, req.PackageName, targetProfile)
		if err != nil {
			return nil, fmt.Errorf("resolve profile %q: %w", targetProfile, err)
		}

		// Build install plan ignoring targets that will be freed by uninstall
		ignoreTargets := make(map[string]struct{}, len(uninstallPlan.Ops))
		for _, op := range uninstallPlan.Ops {
			ignoreTargets[op.Target] = struct{}{}
		}

		installPlan, err := BuildInstallPlan(InstallRequest{
			RepoRoot:      req.RepoRoot,
			PackageName:   req.PackageName,
			ProfileName:   targetProfile,
			Pkg:           req.Pkg,
			Specs:         specs,
			CurrentOS:     req.CurrentOS,
			HomeDir:       req.HomeDir,
			StatePath:     req.StatePath,
			IgnoreTargets: ignoreTargets,
		})
		if err != nil {
			return nil, fmt.Errorf("build install plan: %w", err)
		}

		if len(installPlan.Conflicts) > 0 {
			return nil, fmt.Errorf("pre-flight conflicts detected: %d", len(installPlan.Conflicts))
		}

		return &ConvergeResult{
			PackageName: req.PackageName,
			Outcome:     OutcomeProfileSwitched,
			OldProfile:  pkgState.Profile,
			NewProfile:  targetProfile,
			Plan:        installPlan,
		}, nil
	}

	allOK := true
	for _, link := range pkgState.InstalledLinks {
		ok, err := symlink.IsSymlinkTo(link.Target, link.Source)
		if err != nil || !ok {
			allOK = false
			break
		}
	}
	if allOK {
		return &ConvergeResult{
			PackageName: req.PackageName,
			Outcome:     OutcomeNoOp,
			OldProfile:  targetProfile,
			NewProfile:  targetProfile,
			LinksAfter:  pkgState.InstalledLinks,
		}, nil
	}

	specs, err := profile.ResolveSpecs(req.RepoRoot, req.Pkg, req.PackageName, targetProfile)
	if err != nil {
		return nil, fmt.Errorf("resolve profile %q: %w", targetProfile, err)
	}
	repairPlan, err := BuildInstallPlan(InstallRequest{
		RepoRoot:    req.RepoRoot,
		PackageName: req.PackageName,
		ProfileName: targetProfile,
		Pkg:         req.Pkg,
		Specs:       specs,
		CurrentOS:   req.CurrentOS,
		HomeDir:     req.HomeDir,
		StatePath:   req.StatePath,
	})
	if err != nil {
		return nil, fmt.Errorf("build repair plan: %w", err)
	}
	return &ConvergeResult{
		PackageName: req.PackageName,
		Outcome:     OutcomeRepaired,
		OldProfile:  targetProfile,
		NewProfile:  targetProfile,
		Plan:        repairPlan,
	}, nil
}

// ExecuteConvergePlan executes the plan computed by BuildConvergePlan and
// updates state. For OutcomeNoOp it is a no-op.
func ExecuteConvergePlan(req ConvergeRequest, cr *ConvergeResult) error {
	if cr == nil {
		return fmt.Errorf("nil converge result")
	}
	if cr.Outcome == OutcomeNoOp {
		return nil
	}

	switch cr.Outcome {
	case OutcomeProfileSwitched:
		uninstallPlan, err := BuildUninstallPlan(UninstallRequest{
			PackageName: req.PackageName,
			StatePath:   req.StatePath,
		})
		if err != nil {
			return fmt.Errorf("build uninstall plan: %w", err)
		}
		if err := ExecuteUninstallPlan(uninstallPlan, req.StatePath); err != nil {
			return fmt.Errorf("execute uninstall: %w", err)
		}
		fallthrough
	case OutcomeInstalled, OutcomeRepaired:
		if cr.Outcome == OutcomeRepaired {
			uninstallPlan, err := BuildUninstallPlan(UninstallRequest{
				PackageName: req.PackageName,
				StatePath:   req.StatePath,
			})
			if err == nil {
				_ = ExecuteUninstallPlan(uninstallPlan, req.StatePath)
			}
		}
		specs, err := profile.ResolveSpecs(req.RepoRoot, req.Pkg, req.PackageName, cr.NewProfile)
		if err != nil {
			return fmt.Errorf("resolve profile %q: %w", cr.NewProfile, err)
		}
		result, err := Install(InstallRequest{
			RepoRoot:    req.RepoRoot,
			PackageName: req.PackageName,
			ProfileName: cr.NewProfile,
			Pkg:         req.Pkg,
			Specs:       specs,
			CurrentOS:   req.CurrentOS,
			HomeDir:     req.HomeDir,
			StatePath:   req.StatePath,
		})
		if err != nil {
			return fmt.Errorf("execute install: %w", err)
		}
		if result != nil {
			cr.LinksAfter = result.LinksCreated
		}
	}
	return nil
}

// ConvergeAllRequest captures inputs for converging all packages.
type ConvergeAllRequest struct {
	RepoRoot       string
	DefaultProfile string
	CurrentOS      string
	HomeDir        string
	StatePath      string
	Manifest       *manifest.Manifest
}

// ConvergeAll converges every package declared in the manifest. Errors are
// accumulated; all packages are attempted even if some fail.
func ConvergeAll(req ConvergeAllRequest) ([]ConvergeResult, error) {
	if req.Manifest == nil {
		return nil, fmt.Errorf("converge-all request: Manifest must not be nil")
	}
	var results []ConvergeResult
	var errs []error
	for pkgName, pkg := range req.Manifest.Packages {
		pkgCopy := pkg
		creq := ConvergeRequest{
			RepoRoot:         req.RepoRoot,
			PackageName:      pkgName,
			RequestedProfile: req.DefaultProfile,
			CurrentOS:        req.CurrentOS,
			HomeDir:          req.HomeDir,
			StatePath:        req.StatePath,
			Pkg:              &pkgCopy,
			Manifest:         req.Manifest,
		}
		cr, err := BuildConvergePlan(creq)
		if err != nil {
			errs = append(errs, fmt.Errorf("package %q: %w", pkgName, err))
			continue
		}
		if err := ExecuteConvergePlan(creq, cr); err != nil {
			errs = append(errs, fmt.Errorf("package %q execute: %w", pkgName, err))
			continue
		}
		results = append(results, *cr)
	}
	if len(errs) > 0 {
		return results, errors.Join(errs...)
	}
	return results, nil
}
