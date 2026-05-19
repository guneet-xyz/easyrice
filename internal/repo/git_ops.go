package repo

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// SubmoduleState represents the initialization/modification state of a submodule
// as reported by `git submodule status`.
type SubmoduleState int

const (
	// SubmoduleInitialized means the submodule is initialized and at the indexed SHA.
	SubmoduleInitialized SubmoduleState = iota
	// SubmoduleNotInitialized means `git submodule status` reported a leading '-'.
	SubmoduleNotInitialized
	// SubmoduleModified means the working SHA differs from the index ('+' prefix),
	// or the submodule has merge conflicts ('U' prefix).
	SubmoduleModified
)

// Submodule describes a single entry from `git submodule status`.
type Submodule struct {
	Name  string
	Path  string
	SHA   string
	State SubmoduleState
}

// IsGitRepo reports whether repoPath contains a .git entry (file or directory).
func IsGitRepo(repoPath string) (bool, error) {
	_, err := os.Stat(filepath.Join(repoPath, ".git"))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// IsClean reports whether the working tree has no uncommitted changes.
// Runs: git -C <repoPath> status --porcelain
func IsClean(ctx context.Context, repoPath string) (bool, error) {
	if err := ensureRepoExists(repoPath); err != nil {
		return false, err
	}
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "status", "--porcelain")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("git status: %w: %s", err, out)
	}
	return len(strings.TrimSpace(string(out))) == 0, nil
}

// HasUncommittedChanges is the logical inverse of IsClean.
func HasUncommittedChanges(ctx context.Context, repoPath string) (bool, error) {
	clean, err := IsClean(ctx, repoPath)
	if err != nil {
		return false, err
	}
	return !clean, nil
}

// CurrentBranch returns the name of the currently checked-out branch.
// Returns ErrDetachedHEAD when HEAD is detached.
// Runs: git -C <repoPath> branch --show-current
func CurrentBranch(ctx context.Context, repoPath string) (string, error) {
	if err := ensureRepoExists(repoPath); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "branch", "--show-current")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git branch --show-current: %w: %s", err, out)
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" {
		return "", ErrDetachedHEAD
	}
	return branch, nil
}

func ensureRepoExists(repoPath string) error {
	if _, err := os.Stat(repoPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrRepoNotInitialized, repoPath)
		}
		return err
	}
	return nil
}

// CommitPaths stages the given paths and creates a commit with the given message.
// Returns an error if paths is empty or any path is absolute. NEVER uses `git add -A` or `git add .`.
func CommitPaths(ctx context.Context, repoPath string, paths []string, message string) error {
	if len(paths) == 0 {
		return fmt.Errorf("CommitPaths: paths must not be empty")
	}
	for _, p := range paths {
		if filepath.IsAbs(p) {
			return fmt.Errorf("CommitPaths: paths must be relative to the repo root, got absolute path %q", p)
		}
	}
	addArgs := append([]string{"-C", repoPath, "add", "--"}, paths...)
	cmd := exec.CommandContext(ctx, "git", addArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git add: %w: %s", err, out)
	}
	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "commit", "-m", message)
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit: %w: %s", err, out)
	}
	return nil
}

// SubmoduleAdd adds a git submodule at relPath pointing to url.
// Refuses to add when the working tree is dirty (ErrRepoDirty) or when a
// submodule already exists at relPath (ErrRemoteAlreadyExists).
// Runs: git -C <repoPath> submodule add -- <url> <relPath>
func SubmoduleAdd(ctx context.Context, repoPath, url, relPath string) error {
	if dirty, derr := HasUncommittedChanges(ctx, repoPath); derr == nil && dirty {
		return ErrRepoDirty
	}
	if _, err := os.Stat(filepath.Join(repoPath, relPath)); err == nil {
		return fmt.Errorf("%w: %s", ErrRemoteAlreadyExists, relPath)
	}
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "submodule", "add", "--", url, relPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		outStr := string(out)
		if strings.Contains(outStr, "already exists in the index") {
			return fmt.Errorf("%w: %s: %s", ErrRemoteAlreadyExists, relPath, outStr)
		}
		return fmt.Errorf("git submodule add: %w: %s", err, out)
	}
	return nil
}

// SubmoduleRemove removes a git submodule at relPath.
// Refuses to remove when relPath does not exist (ErrRemoteNotFound) or when
// any profile in rice.toml still imports from this remote (ErrRemoteInUse).
// Runs: git -C <repoPath> submodule deinit -f -- <relPath>
//
//	git -C <repoPath> rm -f -- <relPath>
//	os.RemoveAll(<repoPath>/.git/modules/<relPath>)
func SubmoduleRemove(ctx context.Context, repoPath, relPath string) error {
	if _, err := os.Stat(filepath.Join(repoPath, relPath)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrRemoteNotFound, relPath)
		}
		return err
	}
	if used, name, err := remoteImportedByManifest(repoPath, relPath); err == nil && used {
		return fmt.Errorf("%w: %s", ErrRemoteInUse, name)
	}
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "submodule", "deinit", "-f", "--", relPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git submodule deinit: %w: %s", err, out)
	}
	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "rm", "-f", "--", relPath)
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git rm: %w: %s", err, out)
	}
	if err := os.RemoveAll(filepath.Join(repoPath, ".git", "modules", relPath)); err != nil {
		return fmt.Errorf("remove submodule metadata: %w", err)
	}
	return nil
}

// remoteImportedByManifest does a text scan of <repoPath>/rice.toml looking
// for an `import = "remotes/<name>#..."` reference that matches relPath
// (expected form "remotes/<name>"). Avoids importing manifest to keep this
// package free of toml parsing.
func remoteImportedByManifest(repoPath, relPath string) (bool, string, error) {
	name := strings.TrimPrefix(relPath, "remotes/")
	tomlPath := filepath.Join(repoPath, "rice.toml")
	data, err := os.ReadFile(tomlPath)
	if err != nil {
		return false, name, err
	}
	needle := "remotes/" + name + "#"
	if bytes.Contains(data, []byte(needle)) {
		return true, name, nil
	}
	return false, name, nil
}

// SubmoduleUpdate initializes and updates submodules.
// If relPath is empty, updates all submodules.
// Runs: git -C <repoPath> submodule update --init --remote [-- <relPath>]
func SubmoduleUpdate(ctx context.Context, repoPath, relPath string) error {
	submoduleUpdateMu.Lock()
	defer submoduleUpdateMu.Unlock()
	args := []string{"-C", repoPath, "submodule", "update", "--init", "--remote"}
	if relPath != "" {
		args = append(args, "--", relPath)
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git submodule update: %w: %s", err, out)
	}
	return nil
}

var submoduleUpdateMu sync.Mutex

// SubmoduleList returns the list of submodules in the repo.
// Parses output of: git -C <repoPath> submodule status
//
// Each line: `[+-U ]<sha1> <path> [(<describe>)]`
//   - '-' prefix => SubmoduleNotInitialized
//   - '+' prefix => SubmoduleModified
//   - 'U' prefix => SubmoduleModified (merge conflict)
//   - ' ' prefix => SubmoduleInitialized
func SubmoduleList(ctx context.Context, repoPath string) ([]Submodule, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "submodule", "status")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git submodule status: %w: %s", err, out)
	}
	var subs []Submodule
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if len(line) < 2 {
			continue
		}
		var state SubmoduleState
		switch line[0] {
		case '-':
			state = SubmoduleNotInitialized
		case '+':
			state = SubmoduleModified
		case 'U':
			state = SubmoduleModified
		case ' ':
			state = SubmoduleInitialized
		default:
			// Unknown prefix - skip.
			continue
		}
		rest := line[1:]
		fields := strings.Fields(rest)
		if len(fields) < 2 {
			continue
		}
		sha := fields[0]
		path := fields[1]
		name := path
		if idx := strings.LastIndex(path, "/"); idx >= 0 {
			name = path[idx+1:]
		}
		subs = append(subs, Submodule{
			Name:  name,
			Path:  path,
			SHA:   sha,
			State: state,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan submodule status: %w", err)
	}
	return subs, nil
}
