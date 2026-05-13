package doctor

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/repo"
)

// CheckGitOnPath returns nil if the git binary is available, or a descriptive error.
func CheckGitOnPath() error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git binary not found on PATH — install git to use easyrice")
	}
	return nil
}

// CheckRepoClean checks if the managed repo has uncommitted changes.
// Prints a [WARN] line if the working tree is dirty. Returns the number of issues
// counted (always 0 — a dirty repo is a warning, not a hard error).
func CheckRepoClean(ctx context.Context, out io.Writer, repoRoot string) int {
	dirty, err := repo.HasUncommittedChanges(ctx, repoRoot)
	if err != nil {
		fmt.Fprintf(out, "[WARN] could not check repo cleanliness: %v\n", err)
		return 0
	}
	if dirty {
		fmt.Fprintln(out, "[WARN] rice repo has uncommitted changes; commit to preserve history")
	}
	return 0
}

// CheckSubmodules inspects every git submodule in the managed repo.
// Uninitialized submodules are reported as [ERROR] and counted as issues.
// Modified submodules are reported as [WARN] and do NOT count as issues.
func CheckSubmodules(ctx context.Context, out io.Writer, repoRoot string) int {
	submodules, err := repo.SubmoduleList(ctx, repoRoot)
	if err != nil {
		fmt.Fprintf(out, "[WARN] could not list submodules: %v\n", err)
		return 0
	}
	issues := 0
	for _, sub := range submodules {
		switch sub.State {
		case repo.SubmoduleNotInitialized:
			fmt.Fprintf(out, "[ERROR] submodule %s not initialized; run: rice remote update %s\n", sub.Name, sub.Name)
			issues++
		case repo.SubmoduleModified:
			fmt.Fprintf(out, "[WARN] submodule %s has local changes; commit or update\n", sub.Name)
		}
	}
	return issues
}

// CheckDanglingImports scans every profile in the manifest and verifies that
// any "import" directive resolves to a remote rice that is initialized on disk.
// Each dangling import is reported as [ERROR] and counted as an issue.
func CheckDanglingImports(out io.Writer, repoRoot string, mf manifest.Manifest) int {
	issues := 0
	for pkgName, pkg := range mf.Packages {
		for profileName, profile := range pkg.Profiles {
			if profile.Import == "" {
				continue
			}
			spec, err := manifest.ParseImportSpec(profile.Import)
			if err != nil {
				fmt.Fprintf(out, "[WARN] package %s profile %s: invalid import %q: %v\n", pkgName, profileName, profile.Import, err)
				continue
			}
			tomlPath := repo.RemoteTomlPath(repoRoot, spec.Remote)
			if _, err := os.Stat(tomlPath); os.IsNotExist(err) {
				fmt.Fprintf(out, "[ERROR] package %s profile %s imports remotes/%s but it is not initialized\n", pkgName, profileName, spec.Remote)
				issues++
			}
		}
	}
	return issues
}
