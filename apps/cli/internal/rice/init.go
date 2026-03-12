package rice

import (
	"os"
	"path/filepath"

	"github.com/guneet/easyrice/apps/cli/internal/config"
	"github.com/guneet/easyrice/apps/cli/internal/git"
	"github.com/guneet/easyrice/apps/cli/internal/log"
	"github.com/guneet/easyrice/apps/cli/internal/tui"
)

func isInitialized() (bool, error) {
	riceDir, err := config.RiceDir()
	if err != nil {
		return false, err
	}

	gitDir := filepath.Join(riceDir, ".git")
	log.Get().Log(log.TraceLevel, "checking for existing init", "path", gitDir)

	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		return true, nil
	}

	return false, nil
}

// Initialize sets up the easyrice config directory, git repo, and default config.
// Safe to call multiple times — returns (false, nil) if already initialized.
// Returns (true, nil) if initialization was performed successfully.
func Initialize() (bool, error) {
	already, err := isInitialized()
	if err != nil {
		return false, err
	}
	if already {
		riceDir, _ := config.RiceDir()
		log.Get().Info("easyrice is already initialized", "path", riceDir)
		return false, nil
	}

	riceDir, err := config.RiceDir()
	if err != nil {
		return false, err
	}

	tomlPath, err := config.TomlPath()
	if err != nil {
		return false, err
	}

	steps := []tui.Step{
		{
			Title: "Creating directories",
			Run: func() error {
				log.Get().Log(log.TraceLevel, "creating directory", "path", riceDir)
				return os.MkdirAll(riceDir, 0o755)
			},
		},
		{
			Title: "Initializing git repo",
			Run: func() error {
				return git.Init(riceDir)
			},
		},
		{
			Title: "Writing easyrice.toml",
			Run: func() error {
				content := []byte(config.DefaultToml())
				log.Get().Log(log.TraceLevel, "writing file", "path", tomlPath, "bytes", len(content))
				return os.WriteFile(tomlPath, content, 0o644)
			},
		},
		{
			Title: "Creating initial commit",
			Run: func() error {
				if err := git.Add(riceDir, "."); err != nil {
					return err
				}
				if err := git.Commit(riceDir, "init: initialize easyrice"); err != nil {
					return err
				}
				log.Get().Info("initialized easyrice", "path", riceDir)
				return nil
			},
		},
	}

	if err := tui.RunSteps("Initializing easyrice", steps); err != nil {
		return false, err
	}

	return true, nil
}
