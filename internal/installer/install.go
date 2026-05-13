package installer

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/guneet-xyz/easyrice/internal/logger"
	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/plan"
	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/guneet-xyz/easyrice/internal/symlink"
)

// InstallRequest captures all inputs needed to compute and execute an install.
// The caller is responsible for loading the manifest, looking up the requested
// PackageDef, performing the OS gate check, and resolving the profile to specs
// BEFORE invoking BuildInstallPlan/Install.
type InstallRequest struct {
	RepoRoot      string
	PackageName   string
	ProfileName   string
	Pkg           *manifest.PackageDef
	Specs         []manifest.SourceSpec
	CurrentOS     string
	HomeDir       string
	StatePath     string
	IgnoreTargets map[string]struct{}
}

// InstallResult is returned from a successful (or partial) execution.
type InstallResult struct {
	LinksCreated []state.InstalledLink
}

// withinHome reports whether target is contained in home (defense in depth).
func withinHome(target, home string) bool {
	absHome, err := filepath.Abs(home)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absHome, absTarget)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return false
	}
	return true
}

// BuildInstallPlan computes the plan WITHOUT touching the filesystem (other than reads).
// On conflicts, returns the plan AND an error so callers can render details.
func BuildInstallPlan(req InstallRequest) (*plan.Plan, error) {
	logger.Debug("building install plan",
		zap.String("repoRoot", req.RepoRoot),
		zap.String("package", req.PackageName),
		zap.String("os", req.CurrentOS),
	)

	if req.Pkg == nil {
		return nil, fmt.Errorf("install request: Pkg must not be nil for package %q", req.PackageName)
	}
	specs := req.Specs

	sourcePaths := make([]string, len(specs))
	for i, s := range specs {
		sourcePaths[i] = s.Path
	}
	logger.Info("building install plan",
		zap.String("package", req.PackageName),
		zap.Strings("sources", sourcePaths),
	)

	// 5. Walk each source dir and build planned links.
	// Later sources OVERRIDE earlier ones (last wins) for the same relative path.
	type pendingOp struct {
		Source string
		Target string
		IsDir  bool
	}
	indexByTarget := make(map[string]int)
	var ops []pendingOp
	// Track folder-mode op origin for overlay validation error messages.
	type folderOrigin struct {
		SourcePath string // spec.Path
		Target     string
		OpIndex    int
	}
	var folderOps []folderOrigin

	for _, spec := range specs {
		packageRoot := req.Pkg.Root
		if packageRoot == "" {
			packageRoot = req.PackageName
		}
		var sourceDir string
		if filepath.IsAbs(spec.Path) {
			sourceDir = spec.Path
		} else {
			sourceDir = filepath.Join(req.RepoRoot, packageRoot, spec.Path)
		}
		fi, err := os.Stat(sourceDir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, fmt.Errorf("source directory %q does not exist for package %q", spec.Path, req.PackageName)
			}
			return nil, fmt.Errorf("failed to stat source dir %q: %w", sourceDir, err)
		}
		if !fi.IsDir() {
			return nil, fmt.Errorf("source %q is not a directory", sourceDir)
		}

		// Folder-mode: emit ONE op for the entire directory; do NOT walk in.
		if spec.Mode == "folder" {
			absSource, err := filepath.Abs(sourceDir)
			if err != nil {
				return nil, fmt.Errorf("failed to abs source dir %q: %w", sourceDir, err)
			}
			target := os.ExpandEnv(spec.Target)
			if !withinHome(target, req.HomeDir) {
				return nil, fmt.Errorf("target %q escapes home directory %q", target, req.HomeDir)
			}
			logger.Debug("planned folder op",
				zap.String("source", absSource),
				zap.String("target", target),
			)
			folderOps = append(folderOps, folderOrigin{SourcePath: spec.Path, Target: target, OpIndex: len(ops)})
			ops = append(ops, pendingOp{Source: absSource, Target: target, IsDir: true})
			// Folder-mode ops do NOT participate in file-mode last-wins overlay.
			continue
		}

		sourceName := spec.Path
		walkErr := filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			// Skip rice.toml files anywhere in tree
			if d.Name() == "rice.toml" {
				logger.Warn("skipping rice.toml in source tree", zap.String("path", path))
				return nil
			}
			// Skip symlinks (we only manage real files)
			if d.Type()&fs.ModeSymlink != 0 {
				logger.Warn("skipping symlink in source tree", zap.String("path", path))
				return nil
			}

			rel, err := filepath.Rel(sourceDir, path)
			if err != nil {
				return fmt.Errorf("failed to compute relative path: %w", err)
			}

			sourceTarget := os.ExpandEnv(spec.Target)
			target := filepath.Join(sourceTarget, rel)

			// Defense in depth: ensure target is within HomeDir
			if !withinHome(target, req.HomeDir) {
				return fmt.Errorf("target %q escapes home directory %q", target, req.HomeDir)
			}

			absSource, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("failed to abs source: %w", err)
			}

			logger.Debug("planned op",
				zap.String("source", absSource),
				zap.String("target", target),
			)

			if idx, exists := indexByTarget[target]; exists {
				// Override: later source wins
				ops[idx] = pendingOp{Source: absSource, Target: target}
			} else {
				indexByTarget[target] = len(ops)
				ops = append(ops, pendingOp{Source: absSource, Target: target})
			}
			return nil
		})
		if walkErr != nil {
			return nil, fmt.Errorf("failed to walk source %q: %w", sourceName, walkErr)
		}
	}

	// 6. Overlay validation: folder-mode targets must not collide with any
	// other op's target (file or folder). Check same path, prefix-of, and
	// is-prefix relationships.
	for _, fop := range folderOps {
		folderTarget := fop.Target
		folderPrefix := folderTarget + string(os.PathSeparator)
		for i, other := range ops {
			if i == fop.OpIndex {
				continue
			}
			otherTarget := other.Target
			otherPrefix := otherTarget + string(os.PathSeparator)
			collision := otherTarget == folderTarget ||
				strings.HasPrefix(otherTarget, folderPrefix) ||
				strings.HasPrefix(folderTarget, otherPrefix)
			if collision {
				return nil, fmt.Errorf("planning error: sources %q and %q both target overlapping paths %q and %q",
					fop.SourcePath, other.Source, folderTarget, otherTarget)
			}
		}
	}

	// 7. Build planned-links list for conflict detection. Folder-mode ops are
	// included so DetectConflicts can apply directory-symlink-aware rules.
	planned := make([]PlannedLink, 0, len(ops))
	for _, op := range ops {
		planned = append(planned, PlannedLink{Source: op.Source, Target: op.Target, IsDir: op.IsDir})
	}
	conflicts := DetectConflicts(planned, req.IgnoreTargets)

	// 8. Build plan
	p := &plan.Plan{
		PackageName: req.PackageName,
		Profile:     req.ProfileName,
	}
	for _, op := range ops {
		p.Ops = append(p.Ops, plan.Op{
			Kind:   plan.OpCreate,
			Source: op.Source,
			Target: op.Target,
			IsDir:  op.IsDir,
		})
	}
	for _, c := range conflicts {
		p.Conflicts = append(p.Conflicts, plan.Conflict{
			Target: c.Target,
			Source: c.Source,
			Reason: c.Reason,
			IsDir:  c.IsDir,
		})
	}

	if len(conflicts) > 0 {
		return p, fmt.Errorf("conflicts detected: %d", len(conflicts))
	}

	logger.Debug("install plan built", zap.Int("ops", len(p.Ops)))
	return p, nil
}

// ExecuteInstallPlan applies the plan to the filesystem.
// On partial failure, the partial state is saved and the error returned.
func ExecuteInstallPlan(p *plan.Plan, statePath string) (*InstallResult, error) {
	logger.Info("installing package",
		zap.String("package", p.PackageName),
		zap.String("profile", p.Profile),
		zap.Int("ops", len(p.Ops)),
	)

	// Load existing state
	st, err := state.Load(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	created := make([]state.InstalledLink, 0, len(p.Ops))

	saveAndReturn := func(execErr error) (*InstallResult, error) {
		// InstalledDependencies is populated by installer.EnsureDependencies before plan execution; preserve it here.
		existing := st[p.PackageName]
		st[p.PackageName] = state.PackageState{
			Profile:               p.Profile,
			InstalledLinks:        created,
			InstalledAt:           time.Now(),
			InstalledDependencies: existing.InstalledDependencies,
		}
		if saveErr := state.Save(statePath, st); saveErr != nil {
			logger.Error("failed to save partial install state",
				zap.String("path", statePath),
				zap.Error(saveErr),
			)
			return &InstallResult{LinksCreated: created}, fmt.Errorf("%w; additionally failed to save state: %v", execErr, saveErr)
		}
		return &InstallResult{LinksCreated: created}, execErr
	}

	for _, op := range p.Ops {
		if op.Kind != plan.OpCreate {
			continue
		}
		// Ensure parent directory of target exists
		if err := os.MkdirAll(filepath.Dir(op.Target), 0o755); err != nil {
			logger.Error("failed to create parent directory",
				zap.String("target", op.Target),
				zap.Error(err),
			)
			return saveAndReturn(fmt.Errorf("failed to create parent directory for %s: %w", op.Target, err))
		}
		err := symlink.CreateSymlink(op.Source, op.Target)
		if err != nil {
			// Idempotency: if target already a symlink to our source, treat as success.
			isOurs, checkErr := symlink.IsSymlinkTo(op.Target, op.Source)
			if checkErr == nil && isOurs {
				created = append(created, state.InstalledLink{Source: op.Source, Target: op.Target, IsDir: op.IsDir})
				continue
			}
			logger.Error("failed to create symlink",
				zap.String("source", op.Source),
				zap.String("target", op.Target),
				zap.Error(err),
			)
			return saveAndReturn(fmt.Errorf("failed to create symlink %s -> %s: %w", op.Target, op.Source, err))
		}
		created = append(created, state.InstalledLink{Source: op.Source, Target: op.Target, IsDir: op.IsDir})
	}

	// Success: save full state
	// InstalledDependencies is populated by installer.EnsureDependencies before plan execution; preserve it here.
	existing := st[p.PackageName]
	st[p.PackageName] = state.PackageState{
		Profile:               p.Profile,
		InstalledLinks:        created,
		InstalledAt:           time.Now(),
		InstalledDependencies: existing.InstalledDependencies,
	}
	if err := state.Save(statePath, st); err != nil {
		return &InstallResult{LinksCreated: created}, fmt.Errorf("failed to save state: %w", err)
	}

	logger.Info("installed package",
		zap.String("package", p.PackageName),
		zap.Int("symlinks", len(created)),
	)

	return &InstallResult{LinksCreated: created}, nil
}

// Install is a convenience wrapper combining Build and Execute (used by tests).
// CLI layer should call BuildInstallPlan and ExecuteInstallPlan separately
// to insert a confirmation prompt between them.
func Install(req InstallRequest) (*InstallResult, error) {
	p, err := BuildInstallPlan(req)
	if err != nil {
		return nil, err
	}
	return ExecuteInstallPlan(p, req.StatePath)
}
