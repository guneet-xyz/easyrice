package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/guneet-xyz/easyrice/internal/repo"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Pull the latest changes from the rice repo origin",
	Args:  cobra.NoArgs,
	RunE:  runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	dest := repo.DefaultRepoPath()
	exists, err := repo.Exists(dest)
	if err != nil {
		return fmt.Errorf("check repo: %w", err)
	}
	if !exists {
		return repo.ErrRepoNotInitialized
	}
	if err := repo.Pull(cmd.Context(), dest); err != nil {
		return fmt.Errorf("pull: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Pulled latest from origin")
	return nil
}
