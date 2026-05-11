package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/guneet-xyz/easyrice/internal/deps"
	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/prompt"
	"github.com/guneet-xyz/easyrice/internal/repo"
	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/guneet-xyz/easyrice/internal/symlink"
)

var statusCmd = &cobra.Command{
	Use:   "status [package]",
	Short: "Show installed packages and symlink health",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	st, err := state.Load(flagState)
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	if len(st) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No packages installed.")
		return nil
	}

	filter := ""
	if len(args) == 1 {
		filter = args[0]
	}

	for pkgName, pkgState := range st {
		if filter != "" && pkgName != filter {
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Package: %s (profile: %s)\n", pkgName, pkgState.Profile)
		for _, link := range pkgState.InstalledLinks {
			ok, lerr := symlink.IsSymlinkTo(link.Target, link.Source)
			status := "OK"
			if lerr != nil || !ok {
				status = "BROKEN"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  %s %s -> %s\n", status, link.Target, link.Source)
		}

		if len(pkgState.InstalledDependencies) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Installed dependencies:\n")
			for _, dep := range pkgState.InstalledDependencies {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s (%s) — via %s\n", dep.Name, dep.Version, dep.Method)
			}
		}

		if filter != "" {
			if err := showDeclaredDependencies(cmd, pkgName); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "Warning: dependency check failed: %v\n", err)
			}
		}
	}
	return nil
}

// showDeclaredDependencies runs a live dependency check for a package and renders the report.
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

	fmt.Fprintf(cmd.OutOrStdout(), "Declared dependencies:\n")
	prompt.RenderDepReport(cmd.OutOrStdout(), report)

	return nil
}
