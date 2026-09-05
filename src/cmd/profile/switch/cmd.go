package cmd_profile_switch

import (
	"easyrice/utils"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "switch",
	Short: "switch",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Debug("Called profiles switch")

		err := utils.SetActiveProfile(name)
		if err != nil {
			slog.Error("Failed to change active profile")
			fmt.Println("Couldn't change active profile. Run with 'DEBUG=1' for more information.")
			return
		}

		slog.Debug("Switched profile.")
	},
}

var name string

func init() {
	Cmd.Flags().StringVarP(&name, "name", "n", "", "Name of the profile you want to switch to.")
}
