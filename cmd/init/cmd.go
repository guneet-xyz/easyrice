package cmd_init

import (
	"easyrice/utils"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize easyrice",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Debug("Calling init")
		err := utils.FirstTimeSetup()
		if err != nil {
			slog.Error("Error in first time setup", "error", err)
			fmt.Println("Failed to perform first time setup. Run with 'DEBUG=1' for more information.")
		}
		slog.Debug("Ended init")
	},
}

// create repo, create commit, create ~/.config/easyrice.toml
