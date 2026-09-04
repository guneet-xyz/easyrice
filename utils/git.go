package utils

import (
	"errors"
	"log/slog"
	"strings"
)

var ArgumentError = errors.New("Argument error")
var ErrorBranchDoesNotExist = errors.New("Branch does not exist")

func CloneGitWorktree(remote string, path string) error {
	slog.Debug("Cloning git worktree", "remote", remote, "path", path)
	if IsEmpty(remote) || IsEmpty(path) {
		return ArgumentError
	}

	err := ExecNoOutput("git", "clone", "--bare", remote, path)
	if err != nil {
		slog.Warn("Failed to clone worktree", "error", err)
		return err
	}

	slog.Debug("Cloned git worktree")
	return nil
}

func InitGitWorktree(path string) error {
	slog.Debug("Initializing git worktree", "path", path)
	if IsEmpty(path) {
		return ArgumentError
	}

	_, err := Exec("git", "init", "--bare", path)
	if err != nil {
		slog.Warn("Failed to init worktree", "error", err)
		return err
	}

	slog.Debug("Initialized git worktree")
	return nil
}

func GetBranches(path string) ([]string, error) {
	slog.Debug("Getting branches", "path", path)
	if IsEmpty(path) {
		return nil, ArgumentError
	}
	out, err := Exec("git", "-C", path, "branch", "--no-color")
	if err != nil {
		slog.Warn("Failed to get branches", "error", err)
		return nil, err
	}

	var branches []string
	for line := range strings.SplitSeq(out, "\n") {
		if IsEmpty(line) {
			continue
		}
		branches = append(branches, line[2:])
	}

	return branches, nil
}

func CreateBranchWithBase(path string, newBranch string, baseBranch string) error {
	slog.Debug("Creating branch")
	if IsEmpty(path) {
		slog.Warn("Argument error : path")
		return ArgumentError
	}
	if IsEmpty(newBranch) {
		slog.Warn("Argument error : newBranch")
		return ArgumentError
	}
	if IsEmpty(baseBranch) {
		slog.Warn("Argument error : baseBranch")
		return ArgumentError
	}

	err := ExecNoOutput("git", "-C", path, "branch", newBranch, baseBranch)

	if err != nil {
		slog.Warn("Failed to create branch", "error", err)
		return err
	}
	slog.Debug("Created branch")
	return nil
}

func CreateBranch(path string, branch string) error {
	slog.Debug("Creating branch")
	if IsEmpty(path) {
		slog.Warn("Argument error : path")
		return ArgumentError
	}
	if IsEmpty(branch) {
		slog.Warn("Argument error : branch")
		return ArgumentError
	}

	err := ExecNoOutput("git", "-C", path, "branch", branch)

	if err != nil {
		slog.Warn("Failed to create branch", "error", err)
		return err
	}
	slog.Debug("Created branch")
	return nil
}

func IsBranchExist(path string, branch string) (bool, error) {
	slog.Debug("Checking if branch exists", "path", path, "branch", branch)
	if IsEmpty(path) || IsEmpty(branch) {
		return false, ArgumentError
	}

	out, err := Exec("git", "-C", path, "branch", "--list", branch)
	if err != nil {
		slog.Warn("Failed to check if branch exists", "error", err)
		return false, err
	}

	exists := len(strings.TrimSpace(out)) > 0
	slog.Debug("Checked if branch exists", "exists", exists)
	return exists, nil
}

func RemoveBranch(path string, branch string) error {
	slog.Debug("Removing branch", "path", path, "branch", branch)
	if IsEmpty(path) || IsEmpty(branch) {
		return ArgumentError
	}

	err := ExecNoOutput("git", "-C", path, "branch", "-d", branch)
	if err != nil {
		slog.Warn("Failed to remove branch", "error", err)
		return nil
	}
	slog.Debug("Removed branch")
	return nil
}

func Commit(workingDir string, message string) error {
	slog.Debug("Committing", "workingDir", workingDir, "message", message)
	if IsEmpty(workingDir) {
		slog.Warn("Argument error : workingDir")
		return ArgumentError
	}
	if IsEmpty(message) {
		slog.Warn("Argument error : message")
		return ArgumentError
	}

	err := ExecNoOutput("git", "-C", workingDir, "commit", "-m", message, "-q")

	if err != nil {
		slog.Warn("Failed to run command", "error", err)
		return err
	}

	slog.Debug("Committed")
	return nil
}

func GitStage(workingDir string, path string) error {
	slog.Debug("Staging file", "workingDir", workingDir, "path", path)
	if IsEmpty(workingDir) {
		slog.Warn("Argument error : workingDir")
		return ArgumentError
	}
	if IsEmpty(path) {
		slog.Warn("Argument error : path")
		return ArgumentError
	}

	err := ExecNoOutput("git", "-C", workingDir, "stage", path)

	if err != nil {
		slog.Warn("Failed to run command", "error", err)
		return err
	}

	slog.Debug("Staged file")
	return nil
}

func CreateWorktree(repoDir string, path string, branch string) error {
	slog.Debug("Creating worktree", "repoDir", repoDir, "path", path, "branch", branch)
	if IsEmpty(repoDir) {
		slog.Warn("Argument error : repoDir")
		return ArgumentError
	}
	if IsEmpty(path) {
		slog.Warn("Argument error : path")
		return ArgumentError
	}
	if IsEmpty(branch) {
		slog.Warn("Argument error : branch")
		return ArgumentError
	}

	exists, err := IsBranchExist(repoDir, branch)
	if err != nil {
		slog.Warn("Couldn't check if branch exists", "error", err)
		return err
	}

	if !exists {
		slog.Warn("Branch does not exist")
		return ErrorBranchDoesNotExist
	}

	err = ExecNoOutput("git", "-C", repoDir, "worktree", "add", path, branch, "-q")
	if err != nil {
		slog.Warn("Couldn't create worktree", "error", err)
		return err
	}

	slog.Debug("Created worktree")
	return nil
}

func CreateOrphanWorktree(repoDir string, path string) error {
	slog.Debug("Creating orphan worktree", "repoDir", repoDir, "path", path)
	if IsEmpty(repoDir) {
		slog.Warn("Argument error : repoDir")
		return ArgumentError
	}
	if IsEmpty(path) {
		slog.Warn("Argument error : path")
		return ArgumentError
	}

	err := ExecNoOutput("git", "-C", repoDir, "worktree", "add", path, "--orphan", "-q")
	if err != nil {
		slog.Warn("Couldn't create orphan worktree", "error", err)
		return err
	}

	slog.Debug("Created worktree")
	return nil
}
