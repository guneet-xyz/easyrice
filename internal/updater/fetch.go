package updater

import (
	"context"
	"fmt"

	"github.com/creativeprojects/go-selfupdate"
)

// FetchLatest queries the configured GitHub repository for the latest release.
//
// Behavior:
//   - Returns ErrAlreadyLatest when no release is found or the release is a pre-release.
//   - Returns ErrNoChecksum when the release has no SHA256 checksums asset (fail-closed).
//   - Returns a populated *Release on success.
//
// Networking:
//   - HTTPS only via the GitHub API client embedded in go-selfupdate.
//   - SECURITY: HTTPS only — see internal/updater/AGENTS.md
//   - No GITHUB_TOKEN is read; v1 is anonymous-only.
func (u *Updater) FetchLatest(ctx context.Context) (*Release, error) {
	if u.fetcher != nil {
		return u.fetcher.FetchLatest(ctx)
	}
	return u.fetchLatestFromGitHub(ctx)
}

func (u *Updater) fetchLatestFromGitHub(ctx context.Context) (*Release, error) {
	// SECURITY: HTTPS only — see internal/updater/AGENTS.md
	factory := u.sourceFactory
	if factory == nil {
		factory = func() (selfupdate.Source, error) {
			return selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
		}
	}
	source, err := factory()
	if err != nil {
		return nil, fmt.Errorf("updater: configure github source: %w", err)
	}

	up, err := selfupdate.NewUpdater(selfupdate.Config{
		Source: source,
		// Fail-closed checksum validator: download will be rejected unless a
		// matching SHA256 entry is found in checksums.txt.
		Validator: &selfupdate.ChecksumValidator{
			UniqueFilename: "checksums.txt",
		},
		// Pre-releases are filtered explicitly below; keep library default (false).
	})
	if err != nil {
		return nil, fmt.Errorf("updater: construct updater: %w", err)
	}

	repo := selfupdate.NewRepositorySlug(u.opts.Owner, u.opts.Repo)

	latest, found, err := up.DetectLatest(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("updater: detect latest release: %w", err)
	}
	if !found || latest == nil {
		return nil, ErrAlreadyLatest
	}

	// Defense-in-depth: skip pre-release tags even though Config.Prerelease=false.
	if IsPreRelease(latest.Version()) {
		return nil, ErrAlreadyLatest
	}

	// Fail-closed: reject release if no checksum asset is published.
	if latest.ValidationAssetID == 0 && latest.ValidationAssetURL == "" {
		return nil, ErrNoChecksum
	}

	return &Release{
		Version:  latest.Version(),
		URL:      latest.URL,
		AssetURL: latest.AssetURL,
	}, nil
}
