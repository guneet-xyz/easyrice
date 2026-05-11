package main

import (
	"fmt"
	"os"

	"github.com/guneet-xyz/easyrice/internal/logger"
	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/spf13/cobra"
)

var Version = "dev"

// Repo path is fixed at internal/repo.DefaultRepoPath(); commands resolve it via that helper.
var (
	flagState           string
	flagLogLevel        string
	flagYes             bool
	flagNoUpdateCheck   bool
)

var updateCheckDisabled bool

var rootCmd = &cobra.Command{
	Use:   "easyrice",
	Short: "Cross-platform dotfile manager",
	Long:  `rice installs dotfile packages from a rice repo onto your machine using symlinks.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
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
		if err := logger.Init(lvl, logger.DefaultLogPath()); err != nil {
			return err
		}
		// Set updateCheckDisabled from flag or env var (strict: only "1")
		updateCheckDisabled = flagNoUpdateCheck || os.Getenv("EASYRICE_NO_UPDATE_CHECK") == "1"
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		logger.Sync()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagState, "state", state.DefaultPath(), "path to state file")
	rootCmd.PersistentFlags().StringVar(&flagLogLevel, "log-level", "", "log level: debug|info|warn|error|critical (default: warn, env: EASYRICE_LOG_LEVEL)")
	rootCmd.PersistentFlags().BoolVarP(&flagYes, "yes", "y", false, "bypass confirmation prompts")
	rootCmd.PersistentFlags().BoolVar(&flagNoUpdateCheck, "no-update-check", false, "skip update reminder (env: EASYRICE_NO_UPDATE_CHECK=1)")
}

// UpdateCheckDisabled returns true if update checks should be skipped.
func UpdateCheckDisabled() bool {
	return updateCheckDisabled
}

