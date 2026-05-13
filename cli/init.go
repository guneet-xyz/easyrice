package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/guneet-xyz/easyrice/internal/repo"
)

var initCmd = &cobra.Command{
	Use:   "init <repo-url>",
	Short: "Clone a dotfile repo into ~/.config/easyrice/repos/default/",
	Args:  cobra.ExactArgs(1),
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	dest := repo.DefaultRepoPath()
	exists, err := repo.Exists(dest)
	if err != nil {
		return fmt.Errorf("check repo: %w", err)
	}
	if exists {
		return fmt.Errorf("easyrice repo is already initialized at %q; run `rice update` to pull changes, or remove that directory and run `rice init <url>` again", dest)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}
	if err := repo.Clone(cmd.Context(), args[0], dest); err != nil {
		return fmt.Errorf("clone: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Cloned rice repo to %s\nNext: run `rice install <package>` or `rice install` to converge all packages.\n", dest)
	return nil
}
