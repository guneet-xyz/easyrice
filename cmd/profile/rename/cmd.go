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
		slog.Debug("Called profiles rename", "oldName", oldName, "newName", newName)
		err := utils.RenameProfile(oldName, newName)
		if err != nil {
			slog.Error("Could not rename profile", "error", err)
			fmt.Println("Could not rename profile. Run with 'DEBUG=1' for more information.")
			panic(err)
		}
		slog.Debug("Renamed profile")
		fmt.Println("Renamed profile.")
	},
}

var oldName string
var newName string

func init() {
	Cmd.Flags().StringVarP(&oldName, "from", "f", "", "Name of the profile")
	Cmd.Flags().StringVarP(&newName, "to", "t", "", "New name of the profile")
}
