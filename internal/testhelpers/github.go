package testhelpers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// FakeReleaseOpts configures a fake GitHub release for testing.
type FakeReleaseOpts struct {
	Version          string // e.g. "v1.2.3"
	AssetBytes       []byte // binary content for the fake asset
	IncludeChecksums bool   // whether to include checksums.txt asset
	PreRelease       bool   // whether to mark as pre-release
}

// FakeGitHubServer starts an httptest.Server that serves GitHub Releases API JSON.
// It returns the server so callers can get server.URL and use it as the GitHub API endpoint.
// The server is automatically cleaned up via t.Cleanup.
func FakeGitHubServer(t *testing.T, opts FakeReleaseOpts) *httptest.Server {
	t.Helper()

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/test-owner/test-repo/releases/latest" {
			w.Header().Set("Content-Type", "application/json")

			assets := []map[string]interface{}{
				{
					"id":                   1,
					"name":                 "easyrice_linux_amd64.tar.gz",
					"browser_download_url": fmt.Sprintf("%s/asset/easyrice_linux_amd64.tar.gz", server.URL),
				},
			}

			if opts.IncludeChecksums {
				assets = append(assets, map[string]interface{}{
					"id":                   2,
					"name":                 "checksums.txt",
					"browser_download_url": fmt.Sprintf("%s/asset/checksums.txt", server.URL),
				})
			}

			release := map[string]interface{}{
				"tag_name":   opts.Version,
				"prerelease": opts.PreRelease,
				"assets":     assets,
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(release)
			return
		}

		if r.URL.Path == "/asset/easyrice_linux_amd64.tar.gz" {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			w.Write(opts.AssetBytes)
			return
		}

		if r.URL.Path == "/asset/checksums.txt" {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("sha256sum content"))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))

	t.Cleanup(server.Close)
	return server
}
