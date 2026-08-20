package cmd_profile_create

import (
	"log/slog"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "create",
	Short: "create",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Debug("Called profiles create")
	},
}
