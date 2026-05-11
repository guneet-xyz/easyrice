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
		return fmt.Errorf("easyrice repo already initialized at %q; run `rice update` to pull, or remove the directory to re-init", dest)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}
	if err := repo.Clone(cmd.Context(), args[0], dest); err != nil {
		return fmt.Errorf("clone: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Cloned to %s\nNext: rice install <package>\n", dest)
	return nil
}
