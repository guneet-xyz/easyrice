package utils

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// go:embed ../assets/README.md
var NewSetupReadme []byte

func FirstTimeSetup() error {
	_, err := GetConfig()
	if os.IsNotExist(err) {
		repo, err := TextInput("First time setup. Enter URL for your rice repository.", "Leave empty for initializing new repository", "")
		if err != nil {
			slog.Error("Error while getting text input", "error", err)
			return err
		}

		repositoryDir := filepath.Join(ConfigDir, "repository")
		if IsEmpty(repo) {
			err = InitGitWorktree(repositoryDir)
			if err != nil {
				slog.Error("Error while initializing rice repository", "error", err)
				return err
			}

			err = CreateOrphanWorktree(repositoryDir, WorktreePath("default"))
			if err != nil {
				slog.Warn("Couldn't create default worktree", "error", err)
				return err
			}

			worktreePath := WorktreePath("default")
			readmePath := filepath.Join(worktreePath, "README.md")
			err = WriteBytes(readmePath, NewSetupReadme)
			if err != nil {
				slog.Warn("Failed to write readme", "error", err)
				return err
			}

			err = GitStage(worktreePath, "README.md")
			if err != nil {
				slog.Warn("Couldn't stage readme", "error", err)
				return err
			}

			err = Commit(worktreePath, "Initialize rice repository")
			if err != nil {
				slog.Warn("Couldn't commit", "error", err)
				return err
			}

		} else {
			fmt.Println("Not supported yet.")
			return errors.New("Not supported")
		}

		config := Config{
			RepositoryDir: repositoryDir,
			Profile:       "default",
		}

		err = UpdateConfig(config)
		if err != nil {
			slog.Error("Error while updating config", "error", err)
			return err
		}
	} else {
		fmt.Println("Already initialized.")
	}

	return nil
}
