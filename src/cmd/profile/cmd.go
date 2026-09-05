package cmd_profile

import (
	cmd_profile_create "easyrice/cmd/profile/create"
	cmd_profile_list "easyrice/cmd/profile/list"
	cmd_profile_remove "easyrice/cmd/profile/remove"
	cmd_profile_switch "easyrice/cmd/profile/switch"
	"easyrice/utils"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "profile",
	Short: "profile",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Debug("Called profile")

		profile, err := utils.GetCurrentProfile()
		if err != nil {
			slog.Error("Failed to get current profile", "error", err)
			fmt.Println("Couldn't get active profile. Run with 'DEBUG=1' for more information.")
		}

		fmt.Printf("Current active profile : %s\n", profile)
		slog.Debug("Ended profile")
	},
}

func init() {
	Cmd.AddCommand(cmd_profile_list.Cmd)
	Cmd.AddCommand(cmd_profile_create.Cmd)
	Cmd.AddCommand(cmd_profile_remove.Cmd)
	Cmd.AddCommand(cmd_profile_switch.Cmd)
}
