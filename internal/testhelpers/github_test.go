package testhelpers

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeGitHubServer_AssetEndpoints(t *testing.T) {
	srv := FakeGitHubServer(t, FakeReleaseOpts{
		Version:          "v1.0.0",
		AssetBytes:       []byte("binary-payload"),
		IncludeChecksums: true,
	})

	resp, err := http.Get(srv.URL + "/asset/easyrice_linux_amd64.tar.gz")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "binary-payload", string(body))

	resp2, err := http.Get(srv.URL + "/asset/checksums.txt")
	require.NoError(t, err)
	defer resp2.Body.Close()
	body2, err := io.ReadAll(resp2.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body2), "sha256")

	resp3, err := http.Get(srv.URL + "/unknown")
	require.NoError(t, err)
	defer resp3.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp3.StatusCode)
}

func TestFakeGitHubServer_ReturnsValidJSON(t *testing.T) {
	opts := FakeReleaseOpts{
		Version:          "v1.2.3",
		AssetBytes:       []byte("fake binary content"),
		IncludeChecksums: true,
		PreRelease:       false,
	}

	server := FakeGitHubServer(t, opts)

	// Make a request to the fake server
	resp, err := http.Get(server.URL + "/repos/test-owner/test-repo/releases/latest")
	if err != nil {
		t.Fatalf("failed to GET fake server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	// Parse the JSON response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	var release map[string]interface{}
	if err := json.Unmarshal(body, &release); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	// Verify expected fields
	if release["tag_name"] != "v1.2.3" {
		t.Errorf("expected tag_name v1.2.3, got %v", release["tag_name"])
	}

	if release["prerelease"] != false {
		t.Errorf("expected prerelease false, got %v", release["prerelease"])
	}

	assets, ok := release["assets"].([]interface{})
	if !ok {
		t.Fatalf("assets is not a list")
	}

	if len(assets) != 2 {
		t.Errorf("expected 2 assets (with checksums), got %d", len(assets))
	}
}

func TestFakeGitHubServer_WithoutChecksums(t *testing.T) {
	opts := FakeReleaseOpts{
		Version:          "v2.0.0",
		AssetBytes:       []byte("binary"),
		IncludeChecksums: false,
		PreRelease:       false,
	}

	server := FakeGitHubServer(t, opts)

	resp, err := http.Get(server.URL + "/repos/test-owner/test-repo/releases/latest")
	if err != nil {
		t.Fatalf("failed to GET fake server: %v", err)
	}
	defer resp.Body.Close()

	var release map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &release)

	assets := release["assets"].([]interface{})
	if len(assets) != 1 {
		t.Errorf("expected 1 asset (no checksums), got %d", len(assets))
	}
}

func TestFakeGitHubServer_PreRelease(t *testing.T) {
	opts := FakeReleaseOpts{
		Version:          "v1.0.0-rc1",
		AssetBytes:       []byte("binary"),
		IncludeChecksums: true,
		PreRelease:       true,
	}

	server := FakeGitHubServer(t, opts)

	resp, err := http.Get(server.URL + "/repos/test-owner/test-repo/releases/latest")
	if err != nil {
		t.Fatalf("failed to GET fake server: %v", err)
	}
	defer resp.Body.Close()

	var release map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &release)

	if release["prerelease"] != true {
		t.Errorf("expected prerelease true, got %v", release["prerelease"])
	}
}
