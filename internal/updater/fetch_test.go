package updater

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchLatest_DelegatesToFetcher_Success(t *testing.T) {
	want := &Release{Version: "v1.2.3", URL: "https://example/release", AssetURL: "https://example/asset"}
	u, err := New(Options{Owner: "guneet-xyz", Repo: "easyrice", CacheDir: t.TempDir()})
	require.NoError(t, err)
	u.fetcher = &fakeFetcher{release: want}

	got, err := u.FetchLatest(context.Background())
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestFetchLatest_DelegatesToFetcher_AlreadyLatest(t *testing.T) {
	u, err := New(Options{Owner: "guneet-xyz", Repo: "easyrice", CacheDir: t.TempDir()})
	require.NoError(t, err)
	u.fetcher = &fakeFetcher{err: ErrAlreadyLatest}

	got, err := u.FetchLatest(context.Background())
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, ErrAlreadyLatest))
}

func TestFetchLatest_DelegatesToFetcher_NoChecksum(t *testing.T) {
	u, err := New(Options{Owner: "guneet-xyz", Repo: "easyrice", CacheDir: t.TempDir()})
	require.NoError(t, err)
	u.fetcher = &fakeFetcher{err: ErrNoChecksum}

	got, err := u.FetchLatest(context.Background())
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, ErrNoChecksum))
}

func TestFetchLatest_DelegatesToFetcher_GenericError(t *testing.T) {
	u, err := New(Options{Owner: "guneet-xyz", Repo: "easyrice", CacheDir: t.TempDir()})
	require.NoError(t, err)
	wantErr := errors.New("http 500")
	u.fetcher = &fakeFetcher{err: wantErr}

	got, err := u.FetchLatest(context.Background())
	assert.Nil(t, got)
	assert.ErrorIs(t, err, wantErr)
}

func TestFetchLatest_ContextPassedThrough(t *testing.T) {
	captured := &ctxCapturingFetcher{}
	u, err := New(Options{Owner: "guneet-xyz", Repo: "easyrice", CacheDir: t.TempDir()})
	require.NoError(t, err)
	u.fetcher = captured

	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "marker")
	_, _ = u.FetchLatest(ctx)
	assert.Equal(t, "marker", captured.gotCtx.Value(ctxKey{}))
}

type ctxCapturingFetcher struct {
	gotCtx context.Context
}

func (f *ctxCapturingFetcher) FetchLatest(ctx context.Context) (*Release, error) {
	f.gotCtx = ctx
	return &Release{Version: "v0"}, nil
}
