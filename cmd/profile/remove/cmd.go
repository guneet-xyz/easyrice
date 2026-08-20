package cmd_profile_remove

import (
	"log/slog"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "remove",
	Short: "remove",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Debug("Called profiles remove")
	},
}
