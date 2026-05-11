package updater

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoSelfupdateSwapper_Swap_Integration exercises the production swapper
// against a local httptest.Server serving a raw (uncompressed) asset. This
// lifts swap_seam.go:Swap from 0% coverage by driving the real go-selfupdate
// UpdateTo path end-to-end without touching GitHub or the live network.
//
// The asset URL/filename use no recognized archive extension so go-selfupdate's
// DecompressCommand returns the body as-is (see its fileTypes table) — meaning
// the bytes served are written verbatim to the target path.
func TestGoSelfupdateSwapper_Swap_Integration(t *testing.T) {
	if runtime.GOOS == "windows" {
		// go-selfupdate's swap on Windows hides the .old sibling rather than
		// removing it, and symlink/rename semantics differ. Out of scope here.
		t.Skip("symlink/permission semantics differ on windows")
	}

	const (
		assetName = "easyrice"
		newBody   = "NEW_BINARY_CONTENT_v2"
		oldBody   = "OLD_BINARY_CONTENT_v1"
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/asset/"+assetName {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(newBody))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	target := filepath.Join(dir, assetName)
	require.NoError(t, os.WriteFile(target, []byte(oldBody), 0o755))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sw := &goSelfupdateSwapper{}
	err := sw.Swap(ctx, srv.URL+"/asset/"+assetName, assetName, target)
	require.NoError(t, err)

	got, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, newBody, string(got), "target file must contain the swapped-in bytes")

	// No orphan .new / .old siblings should remain after a successful swap.
	_, errNew := os.Stat(target + ".new")
	assert.True(t, os.IsNotExist(errNew), ".new sibling must not be left behind, got: %v", errNew)
	_, errOld := os.Stat(target + ".old")
	assert.True(t, os.IsNotExist(errOld), ".old sibling must not be left behind, got: %v", errOld)
}
