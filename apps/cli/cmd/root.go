package cmd

import (
	"fmt"
	"os"

	"github.com/guneet/easyrice/apps/cli/internal/log"
	"github.com/guneet/easyrice/apps/cli/internal/tui"
	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags:
//
//	go build -ldflags "-X github.com/guneet/easyrice/apps/cli/cmd.Version=1.0.0"
var Version = "dev"

var (
	verbosity int
	showSteps bool
	hideSteps bool
)

var rootCmd = &cobra.Command{
	Use:   "easyrice",
	Short: "Manage, share, and discover Linux desktop rices",
	Long: `easyrice manages your desktop rice configs as a git repo with symlinks.

Track dotfiles, switch between rices, share them on easyrice.sh,
and try other people's rices with a single command.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log.Init(verbosity)

		switch {
		case showSteps:
			tui.SetStepsMode(tui.StepsAlways)
		case hideSteps:
			tui.SetStepsMode(tui.StepsNever)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	rootCmd.PersistentFlags().CountVarP(&verbosity, "verbose", "v", "increase verbosity (-v for debug, -vv for trace)")
	rootCmd.PersistentFlags().BoolVar(&showSteps, "steps", false, "keep step details visible after completion")
	rootCmd.PersistentFlags().BoolVar(&hideSteps, "no-steps", false, "hide step details entirely")
	rootCmd.MarkFlagsMutuallyExclusive("steps", "no-steps")

	rootCmd.SetVersionTemplate("easyrice {{.Version}}\n")
	rootCmd.Version = Version
}

// Execute runs the root command and exits on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
