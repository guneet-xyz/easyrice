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

		profiles, err := getProfiles()
		if err != nil {
			slog.Error("Failed to get profiles list", "error", err)
			panic(err)
		}

		fmt.Printf("Got profiles list %s", profiles)
	},
}

func getProfiles() ([]string, error) {
	slog.Debug("Getting profiles")

	branches, err := utils.GetBranches(utils.RepoDir)
	if err != nil {
		slog.Warn("Failed to get branches", "error", err)
		return nil, err
	}

	slog.Debug("Got profiles", "profiles", branches)
	return branches, nil
}
