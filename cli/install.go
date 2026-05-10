package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/guneet-xyz/easyrice/internal/installer"
	"github.com/guneet-xyz/easyrice/internal/prompt"
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
	pkg := args[0]

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home dir: %w", err)
	}

	req := installer.InstallRequest{
		RepoRoot:    flagRepo,
		PackageName: pkg,
		Profile:     flagProfile,
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
