package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the easyrice version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.OutOrStdout(), "easyrice "+Version)
		maybePrintUpdateReminder()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
