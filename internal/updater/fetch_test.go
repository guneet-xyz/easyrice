package updater

import (
	"context"
	"errors"
	"testing"

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
