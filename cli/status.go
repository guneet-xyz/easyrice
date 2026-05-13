package main

import (
	"context"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/guneet-xyz/easyrice/internal/deps"
	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/prompt"
	"github.com/guneet-xyz/easyrice/internal/repo"
	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/guneet-xyz/easyrice/internal/style"
	"github.com/guneet-xyz/easyrice/internal/symlink"
)

var statusCmd = &cobra.Command{
	Use:   "status [package]",
	Short: "Show rice repo state, declared-vs-installed packages, and remotes",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

const (
	pkgStatusOK            = "OK"
	pkgStatusBroken        = "BROKEN"
	pkgStatusNotInstalled  = "NOT INSTALLED"
	pkgStatusUntrackedInst = "UNTRACKED"
)

func runStatus(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()

	filter := ""
	if len(args) == 1 {
		filter = args[0]
	}

	repoRoot := repo.DefaultRepoPath()
	st, err := state.Load(flagState)
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	fmt.Fprintf(out, "Rice repo: %s\n", repoRoot)

	exists, existsErr := repo.Exists(repoRoot)
	if existsErr != nil {
		fmt.Fprintf(out, "Warning: could not check rice repo: %v\n", existsErr)
	} else if !exists {
		fmt.Fprintln(out, "Rice repo is not initialized. Run `rice init <url>` first.")
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	repoUsable := existsErr == nil && exists
	if repoUsable {
		renderGitHeader(ctx, out, repoRoot)
	}

	var mf *manifest.Manifest
	if repoUsable {
		if mfLoaded, mfErr := manifest.LoadFile(repo.RepoTomlPath(repoRoot)); mfErr == nil {
			mf = mfLoaded
		} else {
			fmt.Fprintf(out, "Warning: could not load rice.toml: %v\n", mfErr)
		}
	}

	if repoUsable {
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "Packages:")

		pkgNames := collectPackageNames(mf, st, filter)
		if len(pkgNames) == 0 {
			fmt.Fprintln(out, "  (none)")
		} else {
			for _, name := range pkgNames {
				renderPackageLine(out, name, mf, st)
			}
			if filter == "" && len(pkgNames) > 0 {
				installed, notInstalled, broken, untracked := summarizePackages(pkgNames, mf, st)
				sep := "—"
				if style.Plain() {
					sep = "--"
				}
				untrackedStr := ""
				if untracked > 0 {
					untrackedStr = fmt.Sprintf(", %d untracked", untracked)
				}
				fmt.Fprintf(out, "\nTotal: %d packages %s %d installed, %d not installed, %d broken%s.\n",
					len(pkgNames), sep, installed, notInstalled, broken, untrackedStr)
			}
		}
	}

	if filter != "" {
		if depErr := showDeclaredDependencies(cmd, filter); depErr != nil {
			fmt.Fprintf(out, "Warning: could not check declared dependencies: %v\n", depErr)
		}
	}

	if repoUsable {
		fmt.Fprintln(out, "")
		renderRemotes(ctx, out, repoRoot)
	}

	return nil
}

func renderGitHeader(ctx context.Context, w io.Writer, repoRoot string) {
	branch, berr := repo.CurrentBranch(ctx, repoRoot)
	if berr != nil || branch == "" {
		branch = "unknown"
	}
	dirty, derr := repo.HasUncommittedChanges(ctx, repoRoot)
	if derr != nil {
		fmt.Fprintf(w, "Git: branch %s, status unknown\n", branch)
		return
	}
	if dirty {
		fmt.Fprintf(w, "Git: branch %s, uncommitted changes\n", branch)
		fmt.Fprintf(w, "Tip: commit your rice repo changes to preserve history. Repo: %s\n", repoRoot)
		return
	}
	fmt.Fprintf(w, "Git: branch %s, clean\n", branch)
}

func collectPackageNames(mf *manifest.Manifest, st state.State, filter string) []string {
	seen := map[string]struct{}{}
	if mf != nil {
		for name := range mf.Packages {
			seen[name] = struct{}{}
		}
	}
	for name := range st {
		seen[name] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		if filter != "" && name != filter {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func renderPackageLine(w io.Writer, name string, mf *manifest.Manifest, st state.State) {
	pkgState, installed := st[name]
	declared := false
	if mf != nil {
		_, declared = mf.Packages[name]
	}

	switch {
	case !installed:
		fmt.Fprintf(w, "  [%s]    %s\n", pkgStatusNotInstalled, name)
	case installed && !declared:
		fmt.Fprintf(w, "  [%s]    %s (profile: %s)\n", pkgStatusUntrackedInst, name, pkgState.Profile)
	default:
		broken := brokenLinks(pkgState)
		if len(broken) > 0 {
			fmt.Fprintf(w, "  [%s]    %s (profile: %s)\n", pkgStatusBroken, name, pkgState.Profile)
			for _, link := range broken {
				fmt.Fprintf(w, "    broken link: %s -> %s\n", link.Target, link.Source)
			}
		} else {
			fmt.Fprintf(w, "  [%s]    %s (profile: %s)\n", pkgStatusOK, name, pkgState.Profile)
		}
	}
}

func brokenLinks(pkgState state.PackageState) []state.InstalledLink {
	var broken []state.InstalledLink
	for _, link := range pkgState.InstalledLinks {
		ok, lerr := symlink.IsSymlinkTo(link.Target, link.Source)
		if lerr != nil || !ok {
			broken = append(broken, link)
		}
	}
	return broken
}

func summarizePackages(names []string, mf *manifest.Manifest, st state.State) (installed, notInstalled, broken, untracked int) {
	for _, name := range names {
		pkgState, isInstalled := st[name]
		declared := false
		if mf != nil {
			_, declared = mf.Packages[name]
		}
		switch {
		case !isInstalled:
			notInstalled++
		case isInstalled && !declared:
			untracked++
			installed++ // untracked IS installed
		default:
			if len(brokenLinks(pkgState)) > 0 {
				broken++
			}
			installed++
		}
	}
	return
}

func renderRemotes(ctx context.Context, w io.Writer, repoRoot string) {
	subs, err := repo.SubmoduleList(ctx, repoRoot)
	if err != nil || len(subs) == 0 {
		fmt.Fprintln(w, "Remotes: none")
		return
	}
	fmt.Fprintln(w, "Remotes:")
	for _, s := range subs {
		var label string
		switch s.State {
		case repo.SubmoduleInitialized:
			label = "OK"
		case repo.SubmoduleNotInitialized:
			label = "NOT INITIALIZED"
		case repo.SubmoduleModified:
			label = "MODIFIED"
		default:
			label = "?"
		}
		sha := s.SHA
		if len(sha) > 7 {
			sha = sha[:7]
		}
		fmt.Fprintf(w, "  [%s] %s  %s  %s\n", label, s.Name, sha, s.Path)
	}
}

func showDeclaredDependencies(cmd *cobra.Command, pkgName string) error {
	repoRoot := repo.DefaultRepoPath()
	exists, err := repo.Exists(repoRoot)
	if err != nil {
		return fmt.Errorf("check repo: %w", err)
	}
	if !exists {
		return nil
	}

	mf, err := manifest.LoadFile(repo.RepoTomlPath(repoRoot))
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	pkgDef, ok := mf.Packages[pkgName]
	if !ok {
		return nil
	}

	if len(pkgDef.Dependencies) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	var refs []deps.DependencyRef
	for _, depRef := range pkgDef.Dependencies {
		refs = append(refs, depRef)
	}

	platform := deps.Detect()
	report, err := deps.Check(ctx, DepsRunner, refs, mf.CustomDependencies, platform)
	if err != nil {
		return fmt.Errorf("check dependencies: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nDeclared dependencies for %s:\n", pkgName)
	prompt.RenderDepReport(cmd.OutOrStdout(), report)

	return nil
}
