package cmd_profile_create

import (
	"easyrice/utils"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "create",
	Short: "create",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Debug("Called profiles create", "newProfile", newProfile, "baseProfile", baseProfile)

		err := utils.CreateProfile(newProfile, baseProfile)
		if err != nil {
			switch err {
			case utils.ArgumentError:
				fmt.Println("Profile name is required.")
			case utils.ErrorProfileNameNotValid:
				fmt.Println("Profile name is not valid.")
			case utils.ErrorProfileAlreadyExsists:
				fmt.Println("Profile already exists.")
			case utils.ErrorBaseProfileDoesNotExist:
				fmt.Println("Base profile does not exist.")
			default:
				fmt.Println("Profile creation failed. Run with 'DEBUG=1' for more information.")
			}
		}
	},
}

var newProfile string
var baseProfile string

func init() {
	Cmd.Flags().StringVarP(&newProfile, "name", "n", "", "New profile name")
	Cmd.Flags().StringVarP(&baseProfile, "base", "b", "", "Base profile name")
}
