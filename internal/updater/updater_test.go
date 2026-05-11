package updater

import (
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_RequiresOwner(t *testing.T) {
	u, err := New(Options{Repo: "x"})
	assert.Nil(t, u)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Owner")
}

func TestNew_RequiresRepo(t *testing.T) {
	u, err := New(Options{Owner: "x"})
	assert.Nil(t, u)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Repo")
}

func TestNew_AppliesDefaults(t *testing.T) {
	u, err := New(Options{Owner: "o", Repo: "r"})
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, u.opts.Timeout)
	assert.Equal(t, http.DefaultClient, u.opts.HTTPClient)
	assert.NotEmpty(t, u.opts.CacheDir)
}

func TestNew_RespectsExplicitOptions(t *testing.T) {
	customClient := &http.Client{Timeout: 5 * time.Second}
	u, err := New(Options{
		Owner:      "o",
		Repo:       "r",
		Timeout:    7 * time.Second,
		HTTPClient: customClient,
		CacheDir:   "/tmp/custom",
	})
	require.NoError(t, err)
	assert.Equal(t, 7*time.Second, u.opts.Timeout)
	assert.Same(t, customClient, u.opts.HTTPClient)
	assert.Equal(t, "/tmp/custom", u.opts.CacheDir)
}

func TestDefaultCacheDir_EndsInEasyrice(t *testing.T) {
	got := DefaultCacheDir()
	assert.Equal(t, "easyrice", filepath.Base(got))
	if runtime.GOOS != "windows" {
		assert.True(t, strings.Contains(got, "easyrice"), "got %q", got)
	}
}
