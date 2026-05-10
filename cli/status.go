package main

import (
	"fmt"

	"github.com/spf13/cobra"

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
	}
	return nil
}
