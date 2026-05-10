package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/guneet-xyz/easyrice/internal/doctor"
	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/guneet-xyz/easyrice/internal/symlink"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system health and report issues",
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	issues := 0

	doctor.CheckLegacyState(out)

	st, err := state.Load(flagState)
	if err != nil {
		fmt.Fprintf(out, "[ERROR] Cannot read state file %s: %v\n", flagState, err)
		issues++
		st = state.State{}
	}

	for pkgName, pkgState := range st {
		for _, link := range pkgState.InstalledLinks {
			ok, _ := symlink.IsSymlinkTo(link.Target, link.Source)
			if ok {
				continue
			}
			if _, statErr := os.Lstat(link.Target); os.IsNotExist(statErr) {
				fmt.Fprintf(out, "[ERROR] %s: missing symlink %s -> %s\n", pkgName, link.Target, link.Source)
			} else {
				fmt.Fprintf(out, "[ERROR] %s: symlink replaced %s (expected -> %s)\n", pkgName, link.Target, link.Source)
			}
			issues++
		}
	}

	if flagRepo != "." {
		if _, err := os.Stat(flagRepo); err != nil {
			fmt.Fprintf(out, "[ERROR] Repo directory not accessible: %s\n", flagRepo)
			issues++
		}
	}

	if issues == 0 {
		fmt.Fprintln(out, "All checks passed.")
		return nil
	}
	fmt.Fprintf(out, "\n%d issue(s) found.\n", issues)
	return fmt.Errorf("%d issue(s) found", issues)
}
