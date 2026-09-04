package cmd_profile_list

import (
	"easyrice/utils"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "list",
	Short: "list",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Debug("Called profiles list")

		profiles, err := utils.GetProfiles()
		if err != nil {
			fmt.Println("Failed to list profiles. Run with 'DEBUG=1' for more information.")
		}

		fmt.Printf("Got profiles list %s", profiles)
	},
}
