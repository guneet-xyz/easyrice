package repo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func DefaultRepoPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		homeDir, _ := os.UserHomeDir()
		configDir = filepath.Join(homeDir, ".config")
	}
	return filepath.Join(configDir, "easyrice", "repos", "default")
}

func RepoTomlPath(repoRoot string) string {
	return filepath.Join(repoRoot, "rice.toml")
}

func Exists(repoPath string) (bool, error) {
	_, err := os.Stat(repoPath)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func Clone(ctx context.Context, url, dest string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", url, dest)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone: %w: %s", err, out)
	}
	return nil
}

func Pull(ctx context.Context, repoPath string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "pull")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull: %w: %s", err, out)
	}
	return nil
}

func GitOnPath() bool {
	_, err := exec.LookPath("git")
	return err == nil
}
