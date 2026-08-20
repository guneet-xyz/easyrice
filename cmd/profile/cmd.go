package cmd_profile

import (
	cmd_profile_create "easyrice/cmd/profile/create"
	cmd_profile_list "easyrice/cmd/profile/list"
	cmd_profile_remove "easyrice/cmd/profile/remove"
	cmd_profile_switch "easyrice/cmd/profile/switch"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "profile",
	Short: "profile",
}

func init() {
	Cmd.AddCommand(cmd_profile_list.Cmd)
	Cmd.AddCommand(cmd_profile_create.Cmd)
	Cmd.AddCommand(cmd_profile_remove.Cmd)
	Cmd.AddCommand(cmd_profile_switch.Cmd)
}
