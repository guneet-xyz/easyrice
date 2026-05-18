package multiremote

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMulti_Empty(t *testing.T) {
	m := New(t)
	fixture := m.Build()
	defer fixture.Cleanup()

	if fixture.ParentRepoPath == "" {
		t.Fatal("ParentRepoPath should not be empty")
	}
	if len(fixture.RemotePaths) != 0 {
		t.Fatalf("expected 0 remotes, got %d", len(fixture.RemotePaths))
	}

	if !isGitRepo(t, fixture.ParentRepoPath) {
		t.Fatal("parent repo should be a git repo")
	}
}

func TestMulti_SingleRemote(t *testing.T) {
	manifest := `schema_version = 1

[packages.test]
description = "Test package"
supported_os = ["linux", "darwin"]

[packages.test.profiles.default]
sources = [{path = ".", mode = "file", target = "$HOME/.config/test"}]
`

	m := New(t).AddRemote("remote1", manifest)
	fixture := m.Build()
	defer fixture.Cleanup()

	if len(fixture.RemotePaths) != 1 {
		t.Fatalf("expected 1 remote, got %d", len(fixture.RemotePaths))
	}

	if _, ok := fixture.RemotePaths["remote1"]; !ok {
		t.Fatal("remote1 not found in RemotePaths")
	}

	submodules := listSubmodules(t, fixture.ParentRepoPath)
	if len(submodules) != 1 {
		t.Fatalf("expected 1 submodule, got %d", len(submodules))
	}

	if submodules[0].Name != "remote1" {
		t.Fatalf("expected submodule name 'remote1', got '%s'", submodules[0].Name)
	}
}

func TestMulti_ThreeRemotes(t *testing.T) {
	m := New(t).
		AddRemote("remote1", defaultRemoteManifest("remote1")).
		AddRemote("remote2", defaultRemoteManifest("remote2")).
		AddRemote("remote3", defaultRemoteManifest("remote3"))

	fixture := m.Build()
	defer fixture.Cleanup()

	if len(fixture.RemotePaths) != 3 {
		t.Fatalf("expected 3 remotes, got %d", len(fixture.RemotePaths))
	}

	submodules := listSubmodules(t, fixture.ParentRepoPath)
	if len(submodules) != 3 {
		t.Fatalf("expected 3 submodules, got %d", len(submodules))
	}

	names := make(map[string]bool)
	for _, sm := range submodules {
		names[sm.Name] = true
	}

	for _, name := range []string{"remote1", "remote2", "remote3"} {
		if !names[name] {
			t.Fatalf("submodule %s not found", name)
		}
	}
}

func TestMulti_CircularImport(t *testing.T) {
	manifestA := `schema_version = 1

[packages.pkgA]
description = "Package A"
supported_os = ["linux", "darwin"]

[packages.pkgA.profiles.default]
sources = [{path = ".", mode = "file", target = "$HOME/.config/a"}]
`

	manifestB := `schema_version = 1

[packages.pkgB]
description = "Package B"
supported_os = ["linux", "darwin"]

[packages.pkgB.profiles.default]
sources = [{path = ".", mode = "file", target = "$HOME/.config/b"}]
`

	m := New(t).
		AddRemote("a", manifestA).
		AddRemote("b", manifestB).
		WithCircularImport(
			ImportCycle{From: "a", To: "b"},
			ImportCycle{From: "b", To: "a"},
		)

	fixture := m.Build()
	defer fixture.Cleanup()

	aPath := fixture.RemotePaths["a"]
	bPath := fixture.RemotePaths["b"]

	aContent := readFile(t, filepath.Join(aPath, "rice.toml"))
	bContent := readFile(t, filepath.Join(bPath, "rice.toml"))

	if !strings.Contains(aContent, "remotes/b#") {
		t.Fatal("remote a should import from remote b")
	}
	if !strings.Contains(bContent, "remotes/a#") {
		t.Fatal("remote b should import from remote a")
	}
}

func TestMulti_UninitSubmodule(t *testing.T) {
	m := New(t).
		AddRemote("remote1", defaultRemoteManifest("remote1")).
		WithUninitSubmodule("remote1")

	fixture := m.Build()
	defer fixture.Cleanup()

	submodules := listSubmodules(t, fixture.ParentRepoPath)
	if len(submodules) != 1 {
		t.Fatalf("expected 1 submodule, got %d", len(submodules))
	}

	if submodules[0].State != SubmoduleNotInitialized {
		t.Fatalf("expected submodule to be uninitialized, got state %d", submodules[0].State)
	}
}

func TestMulti_InvalidManifestInRemote(t *testing.T) {
	invalidManifest := `this is not valid toml
[broken
`

	m := New(t).AddRemote("remote1", invalidManifest)
	fixture := m.Build()
	defer fixture.Cleanup()

	if len(fixture.RemotePaths) != 1 {
		t.Fatalf("expected 1 remote, got %d", len(fixture.RemotePaths))
	}

	remotePath := fixture.RemotePaths["remote1"]
	content := readFile(t, filepath.Join(remotePath, "rice.toml"))
	if content != invalidManifest {
		t.Fatal("invalid manifest should be written as-is")
	}
}

func TestMulti_AddRemoteRaw(t *testing.T) {
	files := map[string]string{
		"rice.toml": `schema_version = 1

[packages.test]
description = "Test"
supported_os = ["linux"]

[packages.test.profiles.default]
sources = [{path = "config", mode = "file", target = "$HOME/.config/test"}]
`,
		"config/file1.txt": "content1",
		"config/file2.txt": "content2",
	}

	m := New(t).AddRemoteRaw("remote1", files)
	fixture := m.Build()
	defer fixture.Cleanup()

	remotePath := fixture.RemotePaths["remote1"]
	for filePath, expectedContent := range files {
		fullPath := filepath.Join(remotePath, filePath)
		content := readFile(t, fullPath)
		if content != expectedContent {
			t.Fatalf("file %s content mismatch", filePath)
		}
	}
}

type Submodule struct {
	Name  string
	Path  string
	SHA   string
	State SubmoduleState
}

type SubmoduleState int

const (
	SubmoduleInitialized SubmoduleState = iota
	SubmoduleNotInitialized
	SubmoduleModified
)

func isGitRepo(t *testing.T, path string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = path
	err := cmd.Run()
	return err == nil
}

func listSubmodules(t *testing.T, repoPath string) []Submodule {
	cmd := exec.Command("git", "submodule", "status")
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git submodule status failed: %v", err)
	}

	var submodules []Submodule
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		var state SubmoduleState
		var rest string

		if strings.HasPrefix(line, "-") {
			state = SubmoduleNotInitialized
			rest = line[1:]
		} else if strings.HasPrefix(line, "+") {
			state = SubmoduleModified
			rest = line[1:]
		} else if strings.HasPrefix(line, "U") {
			state = SubmoduleModified
			rest = line[1:]
		} else {
			state = SubmoduleInitialized
			rest = line
		}

		parts := strings.Fields(rest)
		if len(parts) < 2 {
			continue
		}

		sha := parts[0]
		path := parts[1]
		name := filepath.Base(path)

		submodules = append(submodules, Submodule{
			Name:  name,
			Path:  path,
			SHA:   sha,
			State: state,
		})
	}

	return submodules
}

func readFile(t *testing.T, path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}
	return string(data)
}
