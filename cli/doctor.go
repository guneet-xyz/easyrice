package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/guneet-xyz/easyrice/internal/deps"
	"github.com/guneet-xyz/easyrice/internal/doctor"
	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/repo"
	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/guneet-xyz/easyrice/internal/symlink"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system health and report issues",
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	issues := 0

	if err := doctor.CheckGitOnPath(); err != nil {
		fmt.Fprintf(out, "[ERROR] Git: %v\n", err)
		issues++
	} else {
		fmt.Fprintln(out, "[OK] Git is available.")
	}

	if err := doctor.CheckRepoInitialized(repo.DefaultRepoPath()); err != nil {
		fmt.Fprintf(out, "[ERROR] Repo: %v\n", err)
		issues++
	} else {
		fmt.Fprintln(out, "[OK] Rice repo is initialized.")
	}

	doctor.CheckLegacyState(out)

	st, err := state.Load(flagState)
	if err != nil {
		fmt.Fprintf(out, "[ERROR] State: could not read %s: %v\n", flagState, err)
		issues++
		st = state.State{}
	}

	repoPath := repo.DefaultRepoPath()
	mf, err := manifest.LoadFile(repo.RepoTomlPath(repoPath))
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(out, "[WARN] Manifest: could not load rice.toml: %v\n", err)
		}
	} else {
		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
		defer cancel()
		runner := &deps.ExecRunner{}
		warnings := doctor.CheckDeclaredDeps(ctx, out, runner, *mf)
		issues += warnings
	}

	gitCtx, gitCancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer gitCancel()
	issues += doctor.CheckRepoClean(gitCtx, out, repoPath)
	issues += doctor.CheckSubmodules(gitCtx, out, repoPath)
	if mf != nil {
		issues += doctor.CheckDanglingImports(out, repoPath, *mf)
	}

	for pkgName, pkgState := range st {
		for _, link := range pkgState.InstalledLinks {
			ok, _ := symlink.IsSymlinkTo(link.Target, link.Source)
			if ok {
				continue
			}
			if _, statErr := os.Lstat(link.Target); os.IsNotExist(statErr) {
				fmt.Fprintf(out, "[ERROR] %s: missing symlink %s -> %s\n", pkgName, link.Target, link.Source)
			} else {
				fmt.Fprintf(out, "[ERROR] %s: symlink was replaced at %s (expected -> %s)\n", pkgName, link.Target, link.Source)
			}
			issues++
		}
	}

	if issues == 0 {
		fmt.Fprintln(out, "All checks passed.")
		maybePrintUpdateReminder()
		return nil
	}
	fmt.Fprintf(out, "\nFound %d issue(s).\n", issues)
	return fmt.Errorf("doctor found %d issue(s)", issues)
}
