package cmd_init

import (
	"easyrice/utils"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize easyrice",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Debug("Calling init")
		err := firstTimeSetup()
		if err != nil {
			slog.Error("Error in first time setup", "error", err)
			panic(err)
		}
		slog.Debug("Ended init")
	},
}

func firstTimeSetup() error {
	_, err := utils.GetConfig()
	if os.IsNotExist(err) {
		repo, err := utils.TextInput("First time setup. Enter URL for your rice repository.", "Leave empty for initializing new repository", "")
		if err != nil {
			slog.Error("Error while getting text input", "error", err)
			return err
		}

		fmt.Printf("Repo %s", repo)
		repositoryDir := filepath.Join(utils.ConfigDir, "repository")
		if utils.IsEmpty(repo) {
			err = utils.InitGitWorktree(repositoryDir)
			if err != nil {
				slog.Error("Error while initializing rice repository", "error", err)
				return err
			}
		} else {
			fmt.Println("Not supported yet.")
			return errors.New("Not supported")
		}

		config := utils.Config{
			RepositoryDir: repositoryDir,
		}

		err = utils.UpdateConfig(config)
		if err != nil {
			slog.Error("Error while updating config", "error", err)
			return err
		}
	} else {
		fmt.Println("Already initialized.")
	}

	return nil
}
