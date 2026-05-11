package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/guneet-xyz/easyrice/internal/installer"
	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/profile"
	"github.com/guneet-xyz/easyrice/internal/prompt"
	"github.com/guneet-xyz/easyrice/internal/repo"
)

var installCmd = &cobra.Command{
	Use:   "install <package>",
	Short: "Install a dotfile package",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstall,
}

var flagProfile string

func init() {
	installCmd.Flags().StringVar(&flagProfile, "profile", "", "profile to install (default: auto-detected from hostname)")
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	pkgName := args[0]

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

	pkgDef, ok := mf.Packages[pkgName]
	if !ok {
		return repo.ErrPackageNotDeclared(pkgName)
	}

	if err := manifest.CheckOS(pkgName, &pkgDef, runtime.GOOS); err != nil {
		return fmt.Errorf("os check: %w", err)
	}

	specs, err := profile.ResolveSpecs(&pkgDef, pkgName, flagProfile)
	if err != nil {
		return fmt.Errorf("resolve profile: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home dir: %w", err)
	}

	req := installer.InstallRequest{
		RepoRoot:    repoRoot,
		PackageName: pkgName,
		ProfileName: flagProfile,
		Pkg:         &pkgDef,
		Specs:       specs,
		CurrentOS:   runtime.GOOS,
		HomeDir:     home,
		StatePath:   flagState,
	}

	p, err := installer.BuildInstallPlan(req)
	if p != nil {
		prompt.RenderPlan(cmd.OutOrStdout(), p)
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

	if _, err := installer.ExecuteInstallPlan(p, flagState); err != nil {
		return fmt.Errorf("execute plan: %w", err)
	}
	return nil
}
