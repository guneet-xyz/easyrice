package cmd

import (
	"github.com/guneet/easyrice/apps/cli/internal/rice"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize easyrice in your home directory",
	Long: `Create the easyrice config directory and initialize a git repository
to track your desktop rice configurations.

This sets up ~/.config/easyrice/rice/ as a managed git repo and creates
a default easyrice.toml file.`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	_, err := rice.Initialize()
	return err
}
