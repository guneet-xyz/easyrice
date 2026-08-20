package cmd_profile_switch

import (
	"log/slog"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "switch",
	Short: "switch",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Debug("Called profiles switch")
	},
}
