package cmd_profile_remove

import (
	"easyrice/utils"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "remove",
	Short: "remove",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Debug("Called profiles remove", "name", name)
		err := utils.RemoveProfile(name)
		if err != nil {
			slog.Error("Could not remove profile", "error", err)
			fmt.Println("Could not remove profile. Run with 'DEBUG=1' for more information.")
			panic(err)
		}
		slog.Debug("Removed profile")
		fmt.Println("Removed profile.")
	},
}

var name string

func init() {
	Cmd.Flags().StringVarP(&name, "name", "n", "", "Name of the profile that you want to remove")
}
