package main

import (
	"fmt"
	"os"

	"github.com/guneet-xyz/easyrice/internal/logger"
	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/spf13/cobra"
)

const Version = "0.1.0"

var (
	flagRepo     string
	flagState    string
	flagLogLevel string
	flagYes      bool
)

var rootCmd = &cobra.Command{
	Use:   "easyrice",
	Short: "Cross-platform dotfile manager",
	Long:  `rice installs dotfile packages from a rice repo onto your machine using symlinks.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Resolve log level: flag wins over env var
		levelStr := flagLogLevel
		if levelStr == "" {
			levelStr = os.Getenv("EASYRICE_LOG_LEVEL")
		}
		if levelStr == "" {
			levelStr = "warn"
		}
		lvl, err := logger.ParseLevel(levelStr)
		if err != nil {
			return fmt.Errorf("--log-level: %w", err)
		}
		return logger.Init(lvl, logger.DefaultLogPath())
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		logger.Sync()
	},
}

// Execute is the entry point for the CLI.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagRepo, "repo", ".", "path to rice repo (default: current directory)")
	rootCmd.PersistentFlags().StringVar(&flagState, "state", state.DefaultPath(), "path to state file")
	rootCmd.PersistentFlags().StringVar(&flagLogLevel, "log-level", "", "log level: debug|info|warn|error|critical (default: warn, env: EASYRICE_LOG_LEVEL)")
	rootCmd.PersistentFlags().BoolVarP(&flagYes, "yes", "y", false, "bypass confirmation prompts")
}
