package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/guneet-xyz/easyrice/internal/installer"
	"github.com/guneet-xyz/easyrice/internal/prompt"
	"github.com/guneet-xyz/easyrice/internal/repo"
)

var switchCmd = &cobra.Command{
	Use:   "switch <package> <new-profile>",
	Short: "Switch a package to a different profile",
	Args:  cobra.ExactArgs(2),
	RunE:  runSwitch,
}

func init() {
	rootCmd.AddCommand(switchCmd)
}

func runSwitch(cmd *cobra.Command, args []string) error {
	pkg := args[0]
	newProfile := args[1]

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home dir: %w", err)
	}

	req := installer.SwitchRequest{
		RepoRoot:    repo.DefaultRepoPath(),
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
