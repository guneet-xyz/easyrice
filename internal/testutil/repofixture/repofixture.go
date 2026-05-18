// Package repofixture provides a builder for constructing test fixtures of managed rice repos.
//
// This package uses exec.Command("git", ...) directly as a permitted exception per the
// better-tests plan (line 250). Production code MUST still go through internal/repo/;
// this helper is test-only and couples to git for fixture construction, not to the SUT.
package repofixture

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"

	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/state"
)

// Profile represents a profile definition for a package.
type Profile struct {
	Sources []manifest.SourceSpec
	Import  string // optional
}

// Builder constructs a managed rice repo fixture with optional packages, submodules, and state.
type Builder struct {
	t              *testing.T
	repoPath       string
	statePath      string
	homePath       string
	packages       map[string]manifest.PackageDef
	submodules     map[string]string // name -> manifestContent
	staleState     string             // raw state.json content
	rawManifest    string             // raw rice.toml content (overrides packages)
	root           string             // optional root override
}

// New creates a new Builder for a managed repo fixture.
func New(t *testing.T) *Builder {
	t.Helper()
	repoPath := filepath.Join(t.TempDir(), "repo")
	statePath := filepath.Join(t.TempDir(), "state.json")
	homePath := filepath.Join(t.TempDir(), "home")

	return &Builder{
		t:          t,
		repoPath:   repoPath,
		statePath:  statePath,
		homePath:   homePath,
		packages:   make(map[string]manifest.PackageDef),
		submodules: make(map[string]string),
	}
}

// WithPackage adds a package definition to the fixture.
// profiles is a map of profile name -> Profile (sources and optional import).
func (b *Builder) WithPackage(name string, profiles map[string]Profile) *Builder {
	b.t.Helper()
	profileDefs := make(map[string]manifest.ProfileDef)
	for pname, p := range profiles {
		profileDefs[pname] = manifest.ProfileDef{
			Sources: p.Sources,
			Import:  p.Import,
		}
	}
	b.packages[name] = manifest.PackageDef{
		Description: fmt.Sprintf("Test package %s", name),
		SupportedOS: []string{"linux", "darwin", "windows"},
		Profiles:    profileDefs,
	}
	return b
}

// WithSubmodule adds a git submodule with the given manifest content.
// The submodule is created as a bare repo and added via git submodule add.
func (b *Builder) WithSubmodule(name, manifestContent string) *Builder {
	b.t.Helper()
	b.submodules[name] = manifestContent
	return b
}

// WithStaleState sets the raw state.json content to be written to the state file.
func (b *Builder) WithStaleState(stateContent string) *Builder {
	b.t.Helper()
	b.staleState = stateContent
	return b
}

// WithRoot sets an optional root directory override for the repo.
func (b *Builder) WithRoot(root string) *Builder {
	b.t.Helper()
	b.root = root
	return b
}

// WithRawManifest sets raw rice.toml content (passthrough, no validation).
// This overrides any packages added via WithPackage.
func (b *Builder) WithRawManifest(raw string) *Builder {
	b.t.Helper()
	b.rawManifest = raw
	return b
}

// Build constructs the fixture and registers cleanup with t.Cleanup.
func (b *Builder) Build() *Fixture {
	b.t.Helper()

	// Create directories
	if err := os.MkdirAll(b.repoPath, 0o755); err != nil {
		b.t.Fatalf("failed to create repo path: %v", err)
	}
	if err := os.MkdirAll(b.homePath, 0o755); err != nil {
		b.t.Fatalf("failed to create home path: %v", err)
	}

	// Initialize git repo
	b.gitRun("init", "-b", "main")
	b.gitRun("config", "user.email", "test@test.com")
	b.gitRun("config", "user.name", "Test User")

	// Write rice.toml
	var manifestContent string
	if b.rawManifest != "" {
		// Raw passthrough: write exactly as provided
		manifestContent = b.rawManifest
	} else {
		// Build manifest from packages
		mf := manifest.Manifest{
			SchemaVersion: 1,
			Packages:      b.packages,
		}
		data, err := toml.Marshal(mf)
		if err != nil {
			b.t.Fatalf("failed to marshal manifest: %v", err)
		}
		manifestContent = string(data)
	}

	if err := os.WriteFile(filepath.Join(b.repoPath, "rice.toml"), []byte(manifestContent), 0o644); err != nil {
		b.t.Fatalf("failed to write rice.toml: %v", err)
	}

	// Create package directories and files
	if b.rawManifest == "" {
		for pkgName, pkgDef := range b.packages {
			root := pkgName
			if b.root != "" {
				root = b.root
			}
			if pkgDef.Root != "" {
				root = pkgDef.Root
			}
			pkgPath := filepath.Join(b.repoPath, root)
			if err := os.MkdirAll(pkgPath, 0o755); err != nil {
				b.t.Fatalf("failed to create package path: %v", err)
			}

			// Create profile directories
			for profileName := range pkgDef.Profiles {
				profilePath := filepath.Join(pkgPath, profileName)
				if err := os.MkdirAll(profilePath, 0o755); err != nil {
					b.t.Fatalf("failed to create profile path: %v", err)
				}
				// Create a dummy file in each profile
				dummyFile := filepath.Join(profilePath, "dummy.txt")
				if err := os.WriteFile(dummyFile, []byte("test content"), 0o644); err != nil {
					b.t.Fatalf("failed to write dummy file: %v", err)
				}
			}
		}
	}

	// Stage and commit initial files
	b.gitRun("add", "--", "rice.toml")
	b.gitRun("commit", "-m", "init managed repo")

	// Create submodules
	for submoduleName, submoduleManifest := range b.submodules {
		b.createSubmodule(submoduleName, submoduleManifest)
	}

	// Write state.json if provided
	if b.staleState != "" {
		if err := os.WriteFile(b.statePath, []byte(b.staleState), 0o644); err != nil {
			b.t.Fatalf("failed to write state.json: %v", err)
		}
	}

	// Register cleanup
	fixture := &Fixture{
		RepoPath:  b.repoPath,
		StatePath: b.statePath,
		HomePath:  b.homePath,
		Cleanup: func() {
			// Cleanup is handled by t.TempDir() auto-cleanup
		},
	}

	b.t.Cleanup(func() {
		// t.TempDir() handles cleanup automatically
	})

	return fixture
}

// createSubmodule creates a bare git repo and adds it as a submodule.
func (b *Builder) createSubmodule(name, manifestContent string) {
	b.t.Helper()

	// Create remotes directory if it doesn't exist
	remotesDir := filepath.Join(b.repoPath, "remotes")
	if err := os.MkdirAll(remotesDir, 0o755); err != nil {
		b.t.Fatalf("failed to create remotes dir: %v", err)
	}

	// Create submodule directory
	submoduleDir := filepath.Join(remotesDir, name)
	if err := os.MkdirAll(submoduleDir, 0o755); err != nil {
		b.t.Fatalf("failed to create submodule dir: %v", err)
	}

	// Write rice.toml to submodule
	if err := os.WriteFile(filepath.Join(submoduleDir, "rice.toml"), []byte(manifestContent), 0o644); err != nil {
		b.t.Fatalf("failed to write submodule rice.toml: %v", err)
	}

	// Create a temporary directory for the bare repo (for git operations)
	tempDir := filepath.Join(b.repoPath, "..", "submodule_"+name)
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		b.t.Fatalf("failed to create submodule temp dir: %v", err)
	}

	// Initialize a bare repo
	cmd := exec.Command("git", "init", "--bare", "-b", "main", tempDir)
	if err := cmd.Run(); err != nil {
		b.t.Fatalf("failed to init bare submodule repo: %v", err)
	}

	// Create a working tree to commit the manifest
	wtDir := filepath.Join(b.repoPath, "..", "submodule_"+name+"_wt")
	if err := os.MkdirAll(wtDir, 0o755); err != nil {
		b.t.Fatalf("failed to create submodule wt dir: %v", err)
	}

	// Initialize working tree
	cmd = exec.Command("git", "init", "-b", "main")
	cmd.Dir = wtDir
	if err := cmd.Run(); err != nil {
		b.t.Fatalf("failed to init submodule wt: %v", err)
	}

	// Configure git
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = wtDir
	if err := cmd.Run(); err != nil {
		b.t.Fatalf("failed to config submodule wt: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = wtDir
	if err := cmd.Run(); err != nil {
		b.t.Fatalf("failed to config submodule wt: %v", err)
	}

	// Write rice.toml
	if err := os.WriteFile(filepath.Join(wtDir, "rice.toml"), []byte(manifestContent), 0o644); err != nil {
		b.t.Fatalf("failed to write submodule rice.toml: %v", err)
	}

	// Commit and push
	cmd = exec.Command("git", "add", "--", "rice.toml")
	cmd.Dir = wtDir
	if err := cmd.Run(); err != nil {
		b.t.Fatalf("failed to add submodule rice.toml: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "init submodule")
	cmd.Dir = wtDir
	if err := cmd.Run(); err != nil {
		b.t.Fatalf("failed to commit submodule: %v", err)
	}

	cmd = exec.Command("git", "remote", "add", "origin", tempDir)
	cmd.Dir = wtDir
	if err := cmd.Run(); err != nil {
		b.t.Fatalf("failed to add submodule remote: %v", err)
	}

	cmd = exec.Command("git", "push", "-u", "origin", "main")
	cmd.Dir = wtDir
	if err := cmd.Run(); err != nil {
		b.t.Fatalf("failed to push submodule: %v", err)
	}

	// Create .gitmodules file manually
	gitmodulesPath := filepath.Join(b.repoPath, ".gitmodules")
	gitmodulesContent := fmt.Sprintf("[submodule \"remotes/%s\"]\n\tpath = remotes/%s\n\turl = ../%s\n", name, name, filepath.Base(tempDir))
	if err := os.WriteFile(gitmodulesPath, []byte(gitmodulesContent), 0o644); err != nil {
		b.t.Fatalf("failed to write .gitmodules: %v", err)
	}

	// Stage and commit submodule changes
	cmd = exec.Command("git", "add", "--", ".gitmodules", "remotes/"+name)
	cmd.Dir = b.repoPath
	if err := cmd.Run(); err != nil {
		b.t.Fatalf("failed to stage submodule: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "add submodule "+name)
	cmd.Dir = b.repoPath
	if err := cmd.Run(); err != nil {
		b.t.Fatalf("failed to commit submodule: %v", err)
	}
}

// gitRun executes a git command in the repo directory.
func (b *Builder) gitRun(args ...string) {
	b.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = b.repoPath
	if err := cmd.Run(); err != nil {
		b.t.Fatalf("git %v failed: %v", args, err)
	}
}

// Fixture represents a ready-to-use managed repo fixture.
type Fixture struct {
	RepoPath  string
	StatePath string
	HomePath  string
	Cleanup   func()
}

// LoadState loads the state.json from the fixture.
func (f *Fixture) LoadState() (state.State, error) {
	return state.Load(f.StatePath)
}

// SaveState saves state to the fixture's state.json.
func (f *Fixture) SaveState(s state.State) error {
	return state.Save(f.StatePath, s)
}

// WriteState writes raw state.json content to the fixture.
func (f *Fixture) WriteState(content string) error {
	return os.WriteFile(f.StatePath, []byte(content), 0o644)
}

// ReadManifest reads the rice.toml from the repo.
func (f *Fixture) ReadManifest() (string, error) {
	data, err := os.ReadFile(filepath.Join(f.RepoPath, "rice.toml"))
	return string(data), err
}

// WriteManifest writes rice.toml to the repo.
func (f *Fixture) WriteManifest(content string) error {
	return os.WriteFile(filepath.Join(f.RepoPath, "rice.toml"), []byte(content), 0o644)
}

// CreateFile creates a file in the repo at the given relative path.
func (f *Fixture) CreateFile(relPath, content string) error {
	fullPath := filepath.Join(f.RepoPath, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, []byte(content), 0o644)
}

// ReadFile reads a file from the repo.
func (f *Fixture) ReadFile(relPath string) (string, error) {
	data, err := os.ReadFile(filepath.Join(f.RepoPath, relPath))
	return string(data), err
}

// CreateHomeFile creates a file in the home directory.
func (f *Fixture) CreateHomeFile(relPath, content string) error {
	fullPath := filepath.Join(f.HomePath, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, []byte(content), 0o644)
}

// ReadHomeFile reads a file from the home directory.
func (f *Fixture) ReadHomeFile(relPath string) (string, error) {
	data, err := os.ReadFile(filepath.Join(f.HomePath, relPath))
	return string(data), err
}

// GitRun executes a git command in the repo directory.
func (f *Fixture) GitRun(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = f.RepoPath
	return cmd.Run()
}

// GitRunOutput executes a git command and returns its output.
func (f *Fixture) GitRunOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = f.RepoPath
	output, err := cmd.CombinedOutput()
	return string(output), err
}
