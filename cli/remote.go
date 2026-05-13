package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/repo"
)

var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Manage remote rice repos (git submodules)",
}

var remoteAddNameFlag string

var remoteNameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

var remoteAddCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Add a remote rice repo as a git submodule",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemoteAdd,
}

var remoteRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a remote rice repo",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemoteRemove,
}

var remoteUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Update one remote rice repo, or all of them",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRemoteUpdate,
}

var remoteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured remote rice repos",
	Args:  cobra.NoArgs,
	RunE:  runRemoteList,
}

func init() {
	rootCmd.AddCommand(remoteCmd)
	remoteCmd.AddCommand(remoteAddCmd)
	remoteCmd.AddCommand(remoteRemoveCmd)
	remoteCmd.AddCommand(remoteUpdateCmd)
	remoteCmd.AddCommand(remoteListCmd)
	remoteAddCmd.Flags().StringVar(&remoteAddNameFlag, "name", "", "name for the remote rice repo (required)")
	_ = remoteAddCmd.MarkFlagRequired("name")
}

func validateRemoteName(name string) error {
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	if !remoteNameRe.MatchString(name) {
		return fmt.Errorf("--name %q is invalid; use only letters, numbers, underscores, and hyphens", name)
	}
	return nil
}

func ensureRepoReady(repoRoot string) error {
	exists, err := repo.Exists(repoRoot)
	if err != nil {
		return fmt.Errorf("check repo: %w", err)
	}
	if !exists {
		return repo.ErrRepoNotInitialized
	}
	isGit, err := repo.IsGitRepo(repoRoot)
	if err != nil {
		return fmt.Errorf("check git repo: %w", err)
	}
	if !isGit {
		return repo.ErrRepoNotInitialized
	}
	return nil
}

func runRemoteAdd(cmd *cobra.Command, args []string) error {
	url := args[0]
	name := remoteAddNameFlag
	if err := validateRemoteName(name); err != nil {
		return err
	}

	repoRoot := repo.DefaultRepoPath()
	if err := ensureRepoReady(repoRoot); err != nil {
		return err
	}

	clean, err := repo.IsClean(cmd.Context(), repoRoot)
	if err != nil {
		return fmt.Errorf("check repo state: %w", err)
	}
	if !clean {
		return repo.ErrRepoDirty
	}

	relPath := "remotes/" + name
	destPath := filepath.Join(repoRoot, "remotes", name)
	if _, err := os.Stat(destPath); err == nil {
		return repo.ErrRemoteAlreadyExists
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat remote dest: %w", err)
	}

	if err := repo.SubmoduleAdd(cmd.Context(), repoRoot, url, relPath); err != nil {
		return fmt.Errorf("add submodule: %w", err)
	}

	tomlPath := repo.RemoteTomlPath(repoRoot, name)
	if _, err := os.Stat(tomlPath); err != nil {
		_ = repo.SubmoduleRemove(cmd.Context(), repoRoot, relPath)
		return fmt.Errorf("remote %q does not contain a rice.toml", name)
	}

	if _, err := manifest.LoadFile(tomlPath); err != nil {
		_ = repo.SubmoduleRemove(cmd.Context(), repoRoot, relPath)
		return fmt.Errorf("remote %q has an invalid rice.toml: %w", name, err)
	}

	if err := repo.CommitPaths(cmd.Context(), repoRoot, []string{".gitmodules", relPath}, fmt.Sprintf("Add remote rice %s", name)); err != nil {
		return fmt.Errorf("commit submodule: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Added remote %q from %s.\nNext: edit rice.toml and import profiles with remotes/%s#<package>.<profile>.\n", name, url, name)
	return nil
}

func runRemoteRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	if err := validateRemoteName(name); err != nil {
		return err
	}

	repoRoot := repo.DefaultRepoPath()
	if err := ensureRepoReady(repoRoot); err != nil {
		return err
	}

	clean, err := repo.IsClean(cmd.Context(), repoRoot)
	if err != nil {
		return fmt.Errorf("check repo state: %w", err)
	}
	if !clean {
		return repo.ErrRepoDirty
	}

	relPath := "remotes/" + name
	destPath := filepath.Join(repoRoot, "remotes", name)
	if _, err := os.Stat(destPath); err != nil {
		if os.IsNotExist(err) {
			return repo.ErrRemoteNotFound
		}
		return fmt.Errorf("stat remote: %w", err)
	}

	tomlPath := repo.RepoTomlPath(repoRoot)
	if _, err := os.Stat(tomlPath); err == nil {
		m, err := manifest.LoadFile(tomlPath)
		if err == nil {
			needle := "remotes/" + name + "#"
			var refs []string
			for pkgName, pkg := range m.Packages {
				for profName, prof := range pkg.Profiles {
					if strings.Contains(prof.Import, needle) {
						refs = append(refs, fmt.Sprintf("%s.%s", pkgName, profName))
					}
				}
			}
			if len(refs) > 0 {
				return fmt.Errorf("%w: %s", repo.ErrRemoteInUse, strings.Join(refs, ", "))
			}
		}
	}

	if err := repo.SubmoduleRemove(cmd.Context(), repoRoot, relPath); err != nil {
		return fmt.Errorf("remove submodule: %w", err)
	}

	if err := repo.CommitPaths(cmd.Context(), repoRoot, []string{".gitmodules"}, fmt.Sprintf("Remove remote rice %s", name)); err != nil {
		return fmt.Errorf("commit removal: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Removed remote %q.\n", name)
	return nil
}

func runRemoteUpdate(cmd *cobra.Command, args []string) error {
	repoRoot := repo.DefaultRepoPath()
	if err := ensureRepoReady(repoRoot); err != nil {
		return err
	}

	if len(args) == 1 {
		name := args[0]
		if err := validateRemoteName(name); err != nil {
			return err
		}
		destPath := filepath.Join(repoRoot, "remotes", name)
		if _, err := os.Stat(destPath); err != nil {
			if os.IsNotExist(err) {
				return repo.ErrRemoteNotFound
			}
			return fmt.Errorf("stat remote: %w", err)
		}
		if err := repo.SubmoduleUpdate(cmd.Context(), repoRoot, "remotes/"+name); err != nil {
			return fmt.Errorf("update submodule: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Updated remote %q.\n", name)
		return nil
	}

	if err := repo.SubmoduleUpdate(cmd.Context(), repoRoot, ""); err != nil {
		return fmt.Errorf("update submodules: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Updated all remotes.")
	return nil
}

func runRemoteList(cmd *cobra.Command, args []string) error {
	repoRoot := repo.DefaultRepoPath()
	if err := ensureRepoReady(repoRoot); err != nil {
		return err
	}

	subs, err := repo.SubmoduleList(cmd.Context(), repoRoot)
	if err != nil {
		return fmt.Errorf("list submodules: %w", err)
	}
	if len(subs) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No remotes configured.")
		return nil
	}

	tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tPATH\tSTATE\tSHA")
	for _, s := range subs {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", s.Name, s.Path, submoduleStateLabel(s.State), s.SHA)
	}
	return tw.Flush()
}

func submoduleStateLabel(s repo.SubmoduleState) string {
	switch s {
	case repo.SubmoduleInitialized:
		return "initialized"
	case repo.SubmoduleNotInitialized:
		return "uninitialized"
	case repo.SubmoduleModified:
		return "modified"
	default:
		return "unknown"
	}
}
