// Package multiremote provides a builder for creating multi-remote test fixtures.
//
// This package uses exec.Command("git", ...) directly for operations not covered by
// internal/repo (e.g., creating separate remote repos, deinit). This is a documented
// exception for test helpers per the better-tests plan (line 553). Production code
// must use internal/repo for all git operations.
package multiremote

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// ImportCycle represents a circular import relationship between two remotes.
type ImportCycle struct {
	From string
	To   string
}

// Multi is a builder for multi-remote test fixtures.
type Multi struct {
	t                *testing.T
	remotes          map[string]string
	remoteRaws       map[string]map[string]string
	parentManifest   string
	circularImports  []ImportCycle
	uninitSubmodules map[string]bool
}

// MultiFixture is the result of building a multi-remote fixture.
type MultiFixture struct {
	ParentRepoPath string
	RemotePaths    map[string]string
	Cleanup        func()
}

// New creates a new Multi builder.
func New(t *testing.T) *Multi {
	return &Multi{
		t:                t,
		remotes:          make(map[string]string),
		remoteRaws:       make(map[string]map[string]string),
		parentManifest:   defaultParentManifest(),
		circularImports:  []ImportCycle{},
		uninitSubmodules: make(map[string]bool),
	}
}

// AddRemote adds a remote with the given name and manifest content.
func (m *Multi) AddRemote(name string, manifestContent string) *Multi {
	m.remotes[name] = manifestContent
	return m
}

// AddRemoteRaw adds a remote with the given name and raw files.
func (m *Multi) AddRemoteRaw(name string, files map[string]string) *Multi {
	m.remoteRaws[name] = files
	return m
}

// WithParentManifest sets the parent repo's rice.toml content.
func (m *Multi) WithParentManifest(content string) *Multi {
	m.parentManifest = content
	return m
}

// WithCircularImport adds a circular import relationship.
func (m *Multi) WithCircularImport(cycles ...ImportCycle) *Multi {
	m.circularImports = append(m.circularImports, cycles...)
	return m
}

// WithUninitSubmodule marks a remote to be uninitialized after adding.
func (m *Multi) WithUninitSubmodule(name string) *Multi {
	m.uninitSubmodules[name] = true
	return m
}

// Build creates the fixture and returns it.
func (m *Multi) Build() *MultiFixture {
	parentPath := m.t.TempDir()
	m.initGitRepo(parentPath)
	m.gitConfig(parentPath, "protocol.file.allow", "always")

	remotePaths := make(map[string]string)
	for name := range m.remotes {
		remotePath := m.t.TempDir()
		m.initGitRepo(remotePath)
		remotePaths[name] = remotePath
	}
	for name := range m.remoteRaws {
		remotePath := m.t.TempDir()
		m.initGitRepo(remotePath)
		remotePaths[name] = remotePath
	}

	for name, manifestContent := range m.remotes {
		remotePath := remotePaths[name]
		m.writeFile(remotePath, "rice.toml", manifestContent)
		m.gitAdd(remotePath, "rice.toml")
		m.gitCommit(remotePath, fmt.Sprintf("Add rice.toml for %s", name))
	}

	for name, files := range m.remoteRaws {
		remotePath := remotePaths[name]
		for filePath, content := range files {
			m.writeFile(remotePath, filePath, content)
		}
		m.gitAddAll(remotePath)
		m.gitCommit(remotePath, fmt.Sprintf("Add files for %s", name))
	}

	if m.parentManifest != "" {
		m.writeFile(parentPath, "rice.toml", m.parentManifest)
		m.gitAdd(parentPath, "rice.toml")
		m.gitCommit(parentPath, "Add parent manifest")
	}

	for name, remotePath := range remotePaths {
		submodulePath := filepath.Join("remotes", name)
		m.gitSubmoduleAdd(parentPath, fmt.Sprintf("file://%s", remotePath), submodulePath)
	}

	if len(remotePaths) > 0 {
		m.gitAddAll(parentPath)
		m.gitCommit(parentPath, "Add submodules")
	}

	for _, cycle := range m.circularImports {
		fromPath := remotePaths[cycle.From]
		toPath := remotePaths[cycle.To]

		fromManifest := m.remotes[cycle.From]
		if fromManifest == "" {
			fromManifest = defaultRemoteManifest(cycle.From)
		}
		fromManifest = m.addImportToManifest(fromManifest, cycle.To)
		m.writeFile(fromPath, "rice.toml", fromManifest)

		toManifest := m.remotes[cycle.To]
		if toManifest == "" {
			toManifest = defaultRemoteManifest(cycle.To)
		}
		toManifest = m.addImportToManifest(toManifest, cycle.From)
		m.writeFile(toPath, "rice.toml", toManifest)

		m.gitAdd(fromPath, "rice.toml")
		if m.hasChanges(fromPath) {
			m.gitCommit(fromPath, fmt.Sprintf("Add circular import to %s", cycle.To))
		}
		m.gitAdd(toPath, "rice.toml")
		if m.hasChanges(toPath) {
			m.gitCommit(toPath, fmt.Sprintf("Add circular import to %s", cycle.From))
		}
	}

	for name := range m.uninitSubmodules {
		submodulePath := filepath.Join("remotes", name)
		m.gitSubmoduleDeinit(parentPath, submodulePath)
	}

	cleanup := func() {
	}

	return &MultiFixture{
		ParentRepoPath: parentPath,
		RemotePaths:    remotePaths,
		Cleanup:        cleanup,
	}
}

func (m *Multi) initGitRepo(path string) {
	cmd := exec.Command("git", "init")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		m.t.Fatalf("git init failed: %v", err)
	}

	m.gitConfig(path, "user.email", "test@example.com")
	m.gitConfig(path, "user.name", "Test User")
	m.gitConfig(path, "protocol.file.allow", "always")

	m.writeFile(path, ".gitkeep", "")
	m.gitAdd(path, ".gitkeep")
	m.gitCommit(path, "Initial commit")
}

func (m *Multi) gitConfig(path, key, value string) {
	cmd := exec.Command("git", "config", key, value)
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		m.t.Fatalf("git config failed: %v", err)
	}
}

func (m *Multi) gitAdd(path, file string) {
	cmd := exec.Command("git", "add", file)
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		m.t.Fatalf("git add failed: %v", err)
	}
}

func (m *Multi) gitAddAll(path string) {
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		m.t.Fatalf("git add -A failed: %v", err)
	}
}

func (m *Multi) gitCommit(path, message string) {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = path
	out, err := cmd.CombinedOutput()
	if err != nil {
		m.t.Fatalf("git commit failed: %v: %s", err, out)
	}
}

func (m *Multi) hasChanges(path string) bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = path
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return len(out) > 0
}

func (m *Multi) gitSubmoduleAdd(path, url, submodulePath string) {
	cmd := exec.Command("git", "-c", "protocol.file.allow=always", "submodule", "add", "--", url, submodulePath)
	cmd.Dir = path
	out, err := cmd.CombinedOutput()
	if err != nil {
		m.t.Fatalf("git submodule add failed: %v: %s", err, out)
	}
}

func (m *Multi) gitSubmoduleDeinit(path, submodulePath string) {
	cmd := exec.Command("git", "submodule", "deinit", "-f", submodulePath)
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		m.t.Fatalf("git submodule deinit failed: %v", err)
	}
}

func (m *Multi) writeFile(path, file, content string) {
	fullPath := filepath.Join(path, file)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		m.t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		m.t.Fatalf("write file failed: %v", err)
	}
}

func (m *Multi) addImportToManifest(manifest, remoteName string) string {
	if manifest == "" {
		return manifest
	}

	importSpec := fmt.Sprintf("remotes/%s#%s.default", remoteName, remoteName)
	return manifest + fmt.Sprintf("\n\n[packages.imported_%s.profiles.default]\nimport = \"%s\"\n", remoteName, importSpec)
}

func defaultParentManifest() string {
	return `schema_version = 1
`
}

func defaultRemoteManifest(name string) string {
	return fmt.Sprintf(`schema_version = 1

[packages.%s]
description = "Package %s"
supported_os = ["linux", "darwin", "windows"]

[packages.%s.profiles.default]
sources = [{path = ".", mode = "file", target = "$HOME/.config/%s"}]
`, name, name, name, name)
}
