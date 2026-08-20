package utils

import (
	"log/slog"
	"os/exec"
	"strings"
)

func CloneGitWorktree(remote string, path string) error {
	slog.Debug("Cloning git worktree", "remote", remote, "path", path)
	cmd := exec.Command("git", "clone", "--bare", remote, path)
	err := cmd.Run()
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			slog.Warn("Cloning git worktree failed", "error", err)
			return err
		}

		if exitErr.ExitCode() == 127 {
			slog.Warn("Cloning git worktree failed because git not found", "exitError", exitErr, "stdout", exitErr.String(), "stderr", exitErr.Stderr)
			return err
		}

		slog.Warn("Cloning git worktree failed because of unknown error", "exitError", exitErr, "stdout", exitErr.String(), "stderr", exitErr.Stderr)
		return err
	}

	slog.Debug("Cloned git worktree")
	return nil
}

func InitGitWorktree(path string) error {
	slog.Debug("Initializing git worktree", "path", path)
	cmd := exec.Command("git", "init", "--bare", path)
	err := cmd.Run()
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			slog.Warn("Initializing git worktree failed", "error", err)
			return err
		}

		if exitErr.ExitCode() == 127 {
			slog.Warn("Initializing git worktree failed because git not found", "exitError", exitErr)
			return err
		}

		slog.Warn("Initializing git worktree failed because of unknown error", "exitError", exitErr)
		return err
	}

	slog.Debug("Initialized git worktree")
	return nil
}

func GetBranches(path string) ([]string, error) {
	slog.Debug("Getting branches", "path", path)
	cmd := exec.Command("git", "branch", "--no-color")

	bytes, err := cmd.Output()
	if err != nil {
		slog.Warn("Listing branches failed", "error", err)
		return nil, err
	}

	out := string(bytes)

	var branches []string
	for line := range strings.SplitSeq(out, "\n") {
		if IsEmpty(line) {
			continue
		}
		branches = append(branches, line[2:])
	}

	return branches, nil
}
