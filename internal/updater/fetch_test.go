package updater

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchLatest_DelegatesWhenFetcherSet(t *testing.T) {
	want := &Release{Version: "v1.2.3", URL: "https://example/release", AssetURL: "https://example/asset"}
	u, err := New(Options{Owner: "guneet-xyz", Repo: "easyrice", CacheDir: t.TempDir()})
	require.NoError(t, err)
	fake := &fakeFetcher{release: want}
	u.fetcher = fake

	got, err := u.FetchLatest(context.Background())
	require.NoError(t, err)
	assert.Same(t, want, got, "FetchLatest must return the exact *Release from the stub fetcher")
	assert.Equal(t, 1, fake.calls)
}

func TestFetchLatest_PropagatesStubError(t *testing.T) {
	wantErr := errors.New("boom")
	u, err := New(Options{Owner: "guneet-xyz", Repo: "easyrice", CacheDir: t.TempDir()})
	require.NoError(t, err)
	u.fetcher = &fakeFetcher{err: wantErr}

	got, err := u.FetchLatest(context.Background())
	assert.Nil(t, got)
	// Same exact error, not double-wrapped.
	assert.Same(t, wantErr, err, "FetchLatest must return the exact stub error without wrapping")
}

func TestFetchLatest_PropagatesSentinels(t *testing.T) {
	cases := []struct {
		name     string
		sentinel error
	}{
		{"already-latest", ErrAlreadyLatest},
		{"no-checksum", ErrNoChecksum},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u, err := New(Options{Owner: "guneet-xyz", Repo: "easyrice", CacheDir: t.TempDir()})
			require.NoError(t, err)
			u.fetcher = &fakeFetcher{err: tc.sentinel}

			got, err := u.FetchLatest(context.Background())
			assert.Nil(t, got)
			assert.True(t, errors.Is(err, tc.sentinel), "expected errors.Is to match sentinel %v, got %v", tc.sentinel, err)
		})
	}
}

func TestFetchLatest_PropagatesContext(t *testing.T) {
	captured := &ctxCapturingFetcher{}
	u, err := New(Options{Owner: "guneet-xyz", Repo: "easyrice", CacheDir: t.TempDir()})
	require.NoError(t, err)
	u.fetcher = captured

	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "marker")
	_, _ = u.FetchLatest(ctx)
	// The fetcher must receive the exact context the caller passed in.
	assert.Equal(t, ctx, captured.gotCtx, "FetchLatest must pass the caller's context unchanged to the fetcher")
	assert.Equal(t, "marker", captured.gotCtx.Value(ctxKey{}))
}

type ctxCapturingFetcher struct {
	gotCtx context.Context
}

func (f *ctxCapturingFetcher) FetchLatest(ctx context.Context) (*Release, error) {
	f.gotCtx = ctx
	return &Release{Version: "v0"}, nil
}

// --- fake selfupdate.Source plumbing for fetchLatestFromGitHub coverage ---

type fakeSourceAsset struct {
	id   int64
	name string
	size int
	url  string
}

func (a *fakeSourceAsset) GetID() int64                  { return a.id }
func (a *fakeSourceAsset) GetName() string               { return a.name }
func (a *fakeSourceAsset) GetSize() int                  { return a.size }
func (a *fakeSourceAsset) GetBrowserDownloadURL() string { return a.url }

type fakeSourceRelease struct {
	id          int64
	tag         string
	url         string
	name        string
	notes       string
	publishedAt time.Time
	prerelease  bool
	draft       bool
	assets      []selfupdate.SourceAsset
}

func (r *fakeSourceRelease) GetID() int64                        { return r.id }
func (r *fakeSourceRelease) GetTagName() string                  { return r.tag }
func (r *fakeSourceRelease) GetDraft() bool                      { return r.draft }
func (r *fakeSourceRelease) GetPrerelease() bool                 { return r.prerelease }
func (r *fakeSourceRelease) GetPublishedAt() time.Time           { return r.publishedAt }
func (r *fakeSourceRelease) GetReleaseNotes() string             { return r.notes }
func (r *fakeSourceRelease) GetName() string                     { return r.name }
func (r *fakeSourceRelease) GetURL() string                      { return r.url }
func (r *fakeSourceRelease) GetAssets() []selfupdate.SourceAsset { return r.assets }

type fakeSource struct {
	releases []selfupdate.SourceRelease
	listErr  error
}

func (s *fakeSource) ListReleases(ctx context.Context, _ selfupdate.Repository) ([]selfupdate.SourceRelease, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.releases, nil
}

func (s *fakeSource) DownloadReleaseAsset(ctx context.Context, _ *selfupdate.Release, _ int64) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

// goReleaseAsset returns a binary asset name shaped like the production release.
func goReleaseAsset(name string, id int64, url string) selfupdate.SourceAsset {
	return &fakeSourceAsset{id: id, name: name, size: 100, url: url}
}

// runtimeAssetName produces an asset name matching the lib's OS/arch suffix matcher.
func runtimeAssetName() string {
	return fmt.Sprintf("easyrice_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
}

func newUpdaterWithSource(t *testing.T, src selfupdate.Source, srcErr error) *Updater {
	t.Helper()
	u, err := New(Options{Owner: "test-owner", Repo: "test-repo", CacheDir: t.TempDir()})
	require.NoError(t, err)
	u.fetcher = nil // ensure we exercise fetchLatestFromGitHub
	u.sourceFactory = func() (selfupdate.Source, error) {
		if srcErr != nil {
			return nil, srcErr
		}
		return src, nil
	}
	return u
}

func TestFetchLatestFromGitHub_NoReleaseFound(t *testing.T) {
	u := newUpdaterWithSource(t, &fakeSource{releases: nil}, nil)

	got, err := u.FetchLatest(context.Background())
	assert.Nil(t, got)
	assert.ErrorIs(t, err, ErrAlreadyLatest)
}

func TestFetchLatestFromGitHub_PreReleaseFiltered(t *testing.T) {
	// Pre-release with prerelease flag false on the SourceRelease (so the lib doesn't
	// drop it itself); the IsPreRelease defense-in-depth check inside fetchLatestFromGitHub
	// must reject it via the tag's pre-release component.
	rel := &fakeSourceRelease{
		id:  1,
		tag: "v1.0.0-rc.1",
		url: "https://example.com/release/v1.0.0-rc.1",
		assets: []selfupdate.SourceAsset{
			goReleaseAsset(runtimeAssetName(), 10, "https://example.com/asset.tar.gz"),
			goReleaseAsset("checksums.txt", 11, "https://example.com/checksums.txt"),
		},
	}
	u := newUpdaterWithSource(t, &fakeSource{releases: []selfupdate.SourceRelease{rel}}, nil)

	got, err := u.FetchLatest(context.Background())
	assert.Nil(t, got)
	assert.ErrorIs(t, err, ErrAlreadyLatest)
}

func TestFetchLatestFromGitHub_NoChecksumAsset(t *testing.T) {
	// Provide a checksums.txt asset whose ID is 0 AND URL is empty so the
	// fail-closed `ValidationAssetID == 0 && ValidationAssetURL == ""` check trips.
	rel := &fakeSourceRelease{
		id:  1,
		tag: "v1.0.0",
		url: "https://example.com/release/v1.0.0",
		assets: []selfupdate.SourceAsset{
			goReleaseAsset(runtimeAssetName(), 10, "https://example.com/asset.tar.gz"),
			// Zero-id, empty-url checksums.txt: the lib finds it (matches by name)
			// but our fail-closed check then rejects the release.
			&fakeSourceAsset{id: 0, name: "checksums.txt", size: 0, url: ""},
		},
	}
	u := newUpdaterWithSource(t, &fakeSource{releases: []selfupdate.SourceRelease{rel}}, nil)

	got, err := u.FetchLatest(context.Background())
	assert.Nil(t, got)
	assert.ErrorIs(t, err, ErrNoChecksum)
}

func TestFetchLatestFromGitHub_SourceFactoryError(t *testing.T) {
	factoryErr := errors.New("factory boom")
	u := newUpdaterWithSource(t, nil, factoryErr)

	got, err := u.FetchLatest(context.Background())
	assert.Nil(t, got)
	require.Error(t, err)
	assert.True(t, errors.Is(err, factoryErr), "expected wrapped factory error")
	assert.Contains(t, err.Error(), "updater: configure github source")
}

func TestFetchLatestFromGitHub_DetectLatestError(t *testing.T) {
	listErr := errors.New("api 500")
	u := newUpdaterWithSource(t, &fakeSource{listErr: listErr}, nil)

	got, err := u.FetchLatest(context.Background())
	assert.Nil(t, got)
	require.Error(t, err)
	assert.True(t, errors.Is(err, listErr), "expected wrapped list error")
	assert.Contains(t, err.Error(), "updater: detect latest release")
}

func TestFetchLatestFromGitHub_HappyPath(t *testing.T) {
	rel := &fakeSourceRelease{
		id:          42,
		tag:         "v2.3.4",
		url:         "https://example.com/release/v2.3.4",
		name:        "Release 2.3.4",
		publishedAt: time.Now(),
		assets: []selfupdate.SourceAsset{
			goReleaseAsset(runtimeAssetName(), 100, "https://example.com/easyrice_linux_amd64.tar.gz"),
			goReleaseAsset("checksums.txt", 101, "https://example.com/checksums.txt"),
		},
	}
	u := newUpdaterWithSource(t, &fakeSource{releases: []selfupdate.SourceRelease{rel}}, nil)

	got, err := u.FetchLatest(context.Background())
	require.NoError(t, err)
	require.NotNil(t, got)
	// Version is the parsed semver (the lib strips the leading "v" via Masterminds/semver).
	assert.True(t, strings.Contains(got.Version, "2.3.4"), "expected version to contain 2.3.4, got %q", got.Version)
	assert.Equal(t, "https://example.com/release/v2.3.4", got.URL)
	assert.Equal(t, "https://example.com/easyrice_linux_amd64.tar.gz", got.AssetURL)
}
