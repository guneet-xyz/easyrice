package testhelpers

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

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
