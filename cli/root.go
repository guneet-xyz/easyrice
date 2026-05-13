package main

import (
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/guneet-xyz/easyrice/internal/logger"
	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/guneet-xyz/easyrice/internal/style"
	"github.com/guneet-xyz/easyrice/internal/updater"
	"github.com/spf13/cobra"
)

var Version = "dev"

// Repo path is fixed at internal/repo.DefaultRepoPath(); commands resolve it via that helper.
var (
	flagState         string
	flagLogLevel      string
	flagYes           bool
	flagNoUpdateCheck bool
	flagPlain         bool
)

var updateCheckDisabled bool

var rootCmd = &cobra.Command{
	Use:          "easyrice",
	Short:        "Cross-platform dotfile manager",
	Long:         `easyrice installs dotfile packages from your managed rice repo using symlinks.`,
	SilenceUsage: true,
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
		// Cleanup orphan artifacts (best-effort, never block command execution)
		if exe, err := os.Executable(); err == nil {
			if cleanupErr := updater.CleanupOrphanArtifacts(exe); cleanupErr != nil {
				logger.L.Debug("could not clean up leftover upgrade files", zap.Error(cleanupErr))
			}
		}
		// Set updateCheckDisabled from flag or env var (strict: only "1")
		updateCheckDisabled = flagNoUpdateCheck || os.Getenv("EASYRICE_NO_UPDATE_CHECK") == "1"
		style.SetPlain(flagPlain)
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
	rootCmd.PersistentFlags().StringVar(&flagState, "state", state.DefaultPath(), "path to the state file")
	rootCmd.PersistentFlags().StringVar(&flagLogLevel, "log-level", "", "log level: debug, info, warn, error, or critical (default: warn; env: EASYRICE_LOG_LEVEL)")
	rootCmd.PersistentFlags().BoolVarP(&flagYes, "yes", "y", false, "bypass confirmation prompts")
	rootCmd.PersistentFlags().BoolVar(&flagNoUpdateCheck, "no-update-check", false, "skip the update reminder (env: EASYRICE_NO_UPDATE_CHECK=1)")
	rootCmd.PersistentFlags().BoolVar(&flagPlain, "plain", false, "ASCII-only output (no glyphs); useful for scripts and CI")
}

// UpdateCheckDisabled returns true if update checks should be skipped.
func UpdateCheckDisabled() bool {
	return updateCheckDisabled
}
