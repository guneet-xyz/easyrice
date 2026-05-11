package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/guneet-xyz/easyrice/internal/prompt"
	"github.com/guneet-xyz/easyrice/internal/updater"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade easyrice to the latest release",
	Args:  cobra.NoArgs,
	RunE:  runUpgrade,
}

var flagUpgradeCheck bool

func init() {
	upgradeCmd.Flags().BoolVar(&flagUpgradeCheck, "check", false, "check for a new release without installing it")
	rootCmd.AddCommand(upgradeCmd)
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	if updater.IsDevBuild(Version) {
		fmt.Fprintln(os.Stderr, "easyrice is a dev build; cannot self-upgrade. Reinstall via `go install github.com/guneet-xyz/easyrice/cli@latest` or download a release from https://github.com/guneet-xyz/easyrice/releases")
		return fmt.Errorf("upgrade: %w", updater.ErrDevBuild)
	}

	u, err := updater.New(updater.Options{
		Owner:    "guneet-xyz",
		Repo:     "easyrice",
		Timeout:  30 * time.Second,
		CacheDir: updater.DefaultCacheDir(),
	})
	if err != nil {
		return fmt.Errorf("upgrade: %w", err)
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	release, err := u.FetchLatest(ctx)
	if err != nil {
		if errors.Is(err, updater.ErrAlreadyLatest) {
			fmt.Fprintf(cmd.OutOrStdout(), "easyrice is up to date (%s)\n", Version)
			return nil
		}
		return fmt.Errorf("could not check for updates: %w", err)
	}

	newer, nerr := updater.IsNewer(Version, release.Version)
	if nerr != nil || !newer {
		fmt.Fprintf(cmd.OutOrStdout(), "easyrice is up to date (%s)\n", Version)
		return nil
	}

	if flagUpgradeCheck {
		fmt.Fprintf(cmd.OutOrStdout(), "new version available: %s → %s\nhttps://github.com/guneet-xyz/easyrice/releases/latest\n", Version, release.Version)
		return nil
	}

	if !flagYes {
		ok, err := prompt.Confirm(cmd.InOrStdin(), cmd.OutOrStdout(), fmt.Sprintf("Upgrade easyrice from %s to %s?", Version, release.Version))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(cmd.OutOrStdout(), "cancelled")
			return nil
		}
	}

	if err := u.Apply(ctx, release); err != nil {
		return fmt.Errorf("upgrade failed: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Upgraded easyrice to %s. Restart easyrice to use the new version.\n", release.Version)
	return nil
}
