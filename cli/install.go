package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/guneet-xyz/easyrice/internal/deps"
	"github.com/guneet-xyz/easyrice/internal/installer"
	"github.com/guneet-xyz/easyrice/internal/logger"
	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/profile"
	"github.com/guneet-xyz/easyrice/internal/prompt"
	"github.com/guneet-xyz/easyrice/internal/repo"
	"github.com/guneet-xyz/easyrice/internal/state"
)

// DepsRunner is the runner used for dependency checks and installs.
// Tests may swap this to a deps.MockRunner before calling Execute().
var DepsRunner deps.Runner = &deps.ExecRunner{}

var installCmd = &cobra.Command{
	Use:   "install [package]",
	Short: "Converge a dotfile package to its desired state (or all packages)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInstall,
}

var (
	flagProfile  string
	flagSkipDeps bool
)

func init() {
	installCmd.Flags().StringVar(&flagProfile, "profile", "", "profile to install (default: stored profile, then hostname, then 'default', then sole profile)")
	installCmd.Flags().BoolVar(&flagSkipDeps, "skip-deps", false, "skip dependency check and install")
	rootCmd.AddCommand(installCmd)
}

func hostnameProfile() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	h = strings.TrimSpace(h)
	if i := strings.IndexByte(h, '.'); i >= 0 {
		h = h[:i]
	}
	return strings.ToLower(h)
}

func resolveDefaultProfile(pkg *manifest.PackageDef, pkgName, flag string, st state.State) string {
	if flag != "" {
		return flag
	}
	if ps, ok := st[pkgName]; ok && ps.Profile != "" {
		return ps.Profile
	}
	if pkg != nil {
		if h := hostnameProfile(); h != "" {
			if _, ok := pkg.Profiles[h]; ok {
				return h
			}
		}
		if _, ok := pkg.Profiles["default"]; ok {
			return "default"
		}
		if len(pkg.Profiles) == 1 {
			for name := range pkg.Profiles {
				return name
			}
		}
	}
	return ""
}

func formatOutcome(cr installer.ConvergeResult) string {
	switch cr.Outcome {
	case installer.OutcomeInstalled:
		return fmt.Sprintf("Installed: %s (profile: %s)", cr.PackageName, cr.NewProfile)
	case installer.OutcomeProfileSwitched:
		return fmt.Sprintf("Switched: %s from %s to %s", cr.PackageName, cr.OldProfile, cr.NewProfile)
	case installer.OutcomeRepaired:
		return fmt.Sprintf("Repaired: %s (%d links)", cr.PackageName, len(cr.LinksAfter))
	case installer.OutcomeNoOp:
		return fmt.Sprintf("No-op: %s already converged", cr.PackageName)
	default:
		return fmt.Sprintf("Unknown outcome for %s", cr.PackageName)
	}
}

func emitDirtyWarning(cmd *cobra.Command, repoRoot string) {
	dirty, err := repo.HasUncommittedChanges(cmd.Context(), repoRoot)
	if err != nil {
		return
	}
	if dirty {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: rice repo at %s has uncommitted changes; commit to preserve history (cd %s && git status).\n", repoRoot, repoRoot)
	}
}

func runInstall(cmd *cobra.Command, args []string) error {
	repoRoot := repo.DefaultRepoPath()
	exists, err := repo.Exists(repoRoot)
	if err != nil {
		return fmt.Errorf("check repo: %w", err)
	}
	if !exists {
		return repo.ErrRepoNotInitialized
	}

	mf, err := manifest.LoadFile(repo.RepoTomlPath(repoRoot))
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	emitDirtyWarning(cmd, repoRoot)

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home dir: %w", err)
	}

	if len(args) == 0 {
		return runInstallAll(cmd, repoRoot, home, mf)
	}
	return runInstallOne(cmd, repoRoot, home, mf, args[0])
}

func runInstallAll(cmd *cobra.Command, repoRoot, home string, mf *manifest.Manifest) error {
	results, cerr := installer.ConvergeAll(installer.ConvergeAllRequest{
		RepoRoot:       repoRoot,
		DefaultProfile: flagProfile,
		CurrentOS:      runtime.GOOS,
		HomeDir:        home,
		StatePath:      flagState,
		Manifest:       mf,
	})

	out := cmd.OutOrStdout()
	sort.Slice(results, func(i, j int) bool {
		return results[i].PackageName < results[j].PackageName
	})
	for _, r := range results {
		switch r.Outcome {
		case installer.OutcomeInstalled:
			fmt.Fprintf(out, "Installed: %s (profile: %s)\n", r.PackageName, r.NewProfile)
		case installer.OutcomeProfileSwitched:
			fmt.Fprintf(out, "Switched: %s from %s to %s\n", r.PackageName, r.OldProfile, r.NewProfile)
		case installer.OutcomeRepaired:
			fmt.Fprintf(out, "Repaired: %s (%d links)\n", r.PackageName, len(r.Plan.Ops))
		case installer.OutcomeNoOp:
			fmt.Fprintf(out, "No-op: %s already converged\n", r.PackageName)
		}
	}
	if cerr != nil {
		return fmt.Errorf("converge-all: %w", cerr)
	}
	return nil
}

func runInstallOne(cmd *cobra.Command, repoRoot, home string, mf *manifest.Manifest, pkgName string) error {
	pkgDef, ok := mf.Packages[pkgName]
	if !ok {
		return repo.ErrPackageNotDeclared(pkgName)
	}

	if err := manifest.CheckOS(pkgName, &pkgDef, runtime.GOOS); err != nil {
		return fmt.Errorf("os check: %w", err)
	}

	if flagSkipDeps {
		logger.L.Warn("skipping dependency check")
	}

	var st state.State
	var pendingDeps state.State
	if !flagSkipDeps {
		var err error
		st, err = state.Load(flagState)
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}
		prevDeps := len(st[pkgName].InstalledDependencies)
		updated, err := installer.EnsureDependencies(cmd.Context(), DepsRunner, *mf, pkgName, flagYes, st)
		if err != nil {
			return fmt.Errorf("ensure dependencies: %w", err)
		}
		if len(updated[pkgName].InstalledDependencies) > prevDeps {
			pendingDeps = updated
		}
	}

	chosen := flagProfile
	if chosen == "" {
		if st == nil {
			loaded, err := state.Load(flagState)
			if err != nil {
				return fmt.Errorf("load state: %w", err)
			}
			st = loaded
		}
		chosen = resolveDefaultProfile(&pkgDef, pkgName, flagProfile, st)
	}
	if chosen == "" {
		return fmt.Errorf("package %q: cannot resolve profile (use --profile)", pkgName)
	}

	req := installer.ConvergeRequest{
		RepoRoot:         repoRoot,
		PackageName:      pkgName,
		RequestedProfile: chosen,
		CurrentOS:        runtime.GOOS,
		HomeDir:          home,
		StatePath:        flagState,
		Pkg:              &pkgDef,
		Manifest:         mf,
	}

	out := cmd.OutOrStdout()

	cr, err := installer.BuildConvergePlan(req)
	if err != nil {
		if strings.Contains(err.Error(), "conflicts detected") {
			renderConflictsPlan(out, req)
		}
		return fmt.Errorf("build converge plan: %w", err)
	}

	renderConvergePlan(out, cr)

	if cr.Plan != nil && len(cr.Plan.Conflicts) > 0 {
		return errors.New("install aborted due to conflicts")
	}

	switch cr.Outcome {
	case installer.OutcomeProfileSwitched:
		fmt.Fprintf(out, "Switching %s: %s → %s\n", pkgName, cr.OldProfile, cr.NewProfile)
	case installer.OutcomeRepaired:
		fmt.Fprintf(out, "Repairing %s (%d broken links)\n", pkgName, len(cr.Plan.Ops))
	case installer.OutcomeNoOp:
		fmt.Fprintf(out, "%s already converged\n", pkgName)
		return nil
	case installer.OutcomeInstalled:
		fmt.Fprintf(out, "Installing %s (profile: %s)\n", pkgName, cr.NewProfile)
	}

	if cr.Outcome != installer.OutcomeNoOp && !flagYes {
		ok, err := prompt.Confirm(cmd.InOrStdin(), out, "Proceed?")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(out, "Aborted.")
			return nil
		}
	}

	// Emit outcome-specific message before confirm prompt
	switch cr.Outcome {
	case installer.OutcomeProfileSwitched:
		fmt.Fprintf(out, "Switching %s: %s → %s\n", pkgName, cr.OldProfile, cr.NewProfile)
	case installer.OutcomeRepaired:
		fmt.Fprintf(out, "Repairing %s (%d broken links)\n", pkgName, len(cr.Plan.Ops))
	case installer.OutcomeNoOp:
		fmt.Fprintf(out, "%s already converged\n", pkgName)
		return nil // skip confirm + execute
	case installer.OutcomeInstalled:
		fmt.Fprintf(out, "Installing %s (profile: %s)\n", pkgName, cr.NewProfile)
	}

	if cr.Outcome != installer.OutcomeNoOp && !flagYes {
		ok, err := prompt.Confirm(cmd.InOrStdin(), out, "Proceed?")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(out, "Aborted.")
			return nil
		}
	}

	if err := installer.ExecuteConvergePlan(req, cr); err != nil {
		return fmt.Errorf("execute plan: %w", err)
	}

	if pendingDeps != nil {
		finalSt, err := state.Load(flagState)
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}
		fps := finalSt[pkgName]
		fps.InstalledDependencies = pendingDeps[pkgName].InstalledDependencies
		finalSt[pkgName] = fps
		if err := state.Save(flagState, finalSt); err != nil {
			return fmt.Errorf("save state: %w", err)
		}
	}

	fmt.Fprintln(out, formatOutcome(*cr))
	return nil
}

func renderConvergePlan(w io.Writer, cr *installer.ConvergeResult) {
	if cr == nil || cr.Plan == nil {
		return
	}
	prompt.RenderPlan(w, cr.Plan)
}

func renderConflictsPlan(w io.Writer, req installer.ConvergeRequest) {
	specs, err := profile.ResolveSpecs(req.RepoRoot, req.Pkg, req.PackageName, req.RequestedProfile)
	if err != nil {
		return
	}
	p, _ := installer.BuildInstallPlan(installer.InstallRequest{
		RepoRoot:    req.RepoRoot,
		PackageName: req.PackageName,
		ProfileName: req.RequestedProfile,
		Pkg:         req.Pkg,
		Specs:       specs,
		CurrentOS:   req.CurrentOS,
		HomeDir:     req.HomeDir,
		StatePath:   req.StatePath,
	})
	if p != nil {
		prompt.RenderPlan(w, p)
	}
}
