package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/guneet-xyz/easyrice/internal/installer"
	"github.com/guneet-xyz/easyrice/internal/logger"
	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/prompt"
	"github.com/guneet-xyz/easyrice/internal/repo"
	"github.com/guneet-xyz/easyrice/internal/state"
)

var switchCmd = &cobra.Command{
	Use:   "switch <package> <new-profile>",
	Short: "Switch a package to a different profile",
	Args:  cobra.ExactArgs(2),
	RunE:  runSwitch,
}

var (
	flagSwitchProfile  string
	flagSwitchSkipDeps bool
)

func init() {
	switchCmd.Flags().StringVar(&flagSwitchProfile, "profile", "", "profile to switch to")
	switchCmd.Flags().BoolVar(&flagSwitchSkipDeps, "skip-deps", false, "skip dependency check and install")
	rootCmd.AddCommand(switchCmd)
}

func runSwitch(cmd *cobra.Command, args []string) error {
	pkg := args[0]
	newProfile := args[1]

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

	pkgDef, ok := mf.Packages[pkg]
	if !ok {
		return repo.ErrPackageNotDeclared(pkg)
	}

	if err := manifest.CheckOS(pkg, &pkgDef, runtime.GOOS); err != nil {
		return fmt.Errorf("os check: %w", err)
	}

	// Ensure dependencies before switch
	if flagSwitchSkipDeps {
		logger.L.Warn("skipping dependency check")
	} else {
		s, err := state.Load(flagState)
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}
		updated, err := installer.EnsureDependencies(cmd.Context(), DepsRunner, *mf, pkg, flagYes, s)
		if err != nil {
			return fmt.Errorf("ensure dependencies: %w", err)
		}
		if len(updated[pkg].InstalledDependencies) > len(s[pkg].InstalledDependencies) {
			if err := state.Save(flagState, updated); err != nil {
				return fmt.Errorf("save state: %w", err)
			}
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home dir: %w", err)
	}

	req := installer.SwitchRequest{
		RepoRoot:    repoRoot,
		PackageName: pkg,
		NewProfile:  newProfile,
		CurrentOS:   runtime.GOOS,
		HomeDir:     home,
		StatePath:   flagState,
	}

	sp, err := installer.BuildSwitchPlan(req)
	if sp != nil {
		prompt.RenderSwitchPlan(cmd.OutOrStdout(), sp.Uninstall, sp.Install)
	}
	if err != nil {
		return fmt.Errorf("build plan: %w", err)
	}

	if !flagYes {
		ok, err := prompt.Confirm(cmd.InOrStdin(), cmd.OutOrStdout(), "Proceed?")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
			return nil
		}
	}

	if err := installer.ExecuteSwitchPlan(sp, flagState); err != nil {
		return fmt.Errorf("execute plan: %w", err)
	}
	return nil
}
