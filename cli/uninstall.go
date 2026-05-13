package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/guneet-xyz/easyrice/internal/installer"
	"github.com/guneet-xyz/easyrice/internal/prompt"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <package>",
	Short: "Uninstall a dotfile package",
	Args:  cobra.ExactArgs(1),
	RunE:  runUninstall,
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall(cmd *cobra.Command, args []string) error {
	pkg := args[0]

	req := installer.UninstallRequest{
		PackageName: pkg,
		StatePath:   flagState,
	}

	p, err := installer.BuildUninstallPlan(req)
	if err != nil {
		return fmt.Errorf("build plan: %w", err)
	}

	prompt.RenderPlan(cmd.OutOrStdout(), p)

	if !flagYes {
		ok, err := prompt.Confirm(cmd.InOrStdin(), cmd.OutOrStdout(), "Apply this plan?")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(cmd.OutOrStdout(), "Cancelled. No changes were made.")
			return nil
		}
	}

	if err := installer.ExecuteUninstallPlan(p, flagState); err != nil {
		return fmt.Errorf("execute plan: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Uninstalled %s.\n", pkg)
	return nil
}
