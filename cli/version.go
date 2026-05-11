package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the rice version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("easyrice version " + Version)
		maybePrintUpdateReminder()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
