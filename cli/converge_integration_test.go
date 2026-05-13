package main

// Cross-package converge + remote scenarios.
//
// Each TestIntegration_* runs in its own t.TempDir()-isolated HOME, sets up a
// managed git repo via the existing helpers in remote_test.go / init_test.go,
// and exercises the rice CLI through rootCmd.Execute (no built binary needed).

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/repo"
	"github.com/guneet-xyz/easyrice/internal/state"
	"github.com/guneet-xyz/easyrice/internal/symlink"
)

// makeManagedRepoWithFiles initializes a fresh managed git repo containing
// rice.toml plus an arbitrary set of (relative path, content) extra files,
// then commits everything with a single explicit `git add --` (no `git add .`).
func makeManagedRepoWithFiles(t *testing.T, manifestContent string, extra map[string]string) string {
	t.Helper()
	gitOnPath(t)
	root := repo.DefaultRepoPath()
	require.NoError(t, os.MkdirAll(root, 0o755))
	gitRunRemote(t, root, "init", "-b", "main")
	gitRunRemote(t, root, "config", "user.email", "test@test.com")
	gitRunRemote(t, root, "config", "user.name", "Test")

	require.NoError(t, os.WriteFile(filepath.Join(root, "rice.toml"), []byte(manifestContent), 0o644))
	addArgs := []string{"add", "--", "rice.toml"}
	for rel, content := range extra {
		full := filepath.Join(root, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
		addArgs = append(addArgs, rel)
	}
	gitRunRemote(t, root, addArgs...)
	gitRunRemote(t, root, "commit", "-m", "init managed")
	return root
}

// setupBareUpstream initializes a bare git repo from a local fixture directory
// (every file under fixtureDir is copied into a working tree, committed, and
// pushed to a bare remote). Returns a file:// URL suitable for `git submodule add`.
func setupBareUpstream(t *testing.T, fixtureDir string) string {
	t.Helper()
	gitOnPath(t)

	bareDir := t.TempDir()
	bare := filepath.Join(bareDir, "upstream.git")
	require.NoError(t, os.MkdirAll(bare, 0o755))
	gitRunRemote(t, bareDir, "init", "--bare", "-b", "main", "upstream.git")

	wt := t.TempDir()
	gitRunRemote(t, wt, "init", "-b", "main")
	gitRunRemote(t, wt, "config", "user.email", "test@test.com")
	gitRunRemote(t, wt, "config", "user.name", "Test")

	// Copy fixtureDir contents into the worktree.
	require.NoError(t, filepath.Walk(fixtureDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(fixtureDir, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		dest := filepath.Join(wt, rel)
		if info.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dest, data, 0o644)
	}))

	// Stage every file we just copied (no `git add .`).
	addArgs := []string{"add", "--"}
	require.NoError(t, filepath.Walk(wt, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if filepath.Base(p) == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(wt, p)
		if err != nil {
			return err
		}
		addArgs = append(addArgs, rel)
		return nil
	}))
	require.Greater(t, len(addArgs), 2, "no fixture files copied into worktree")

	gitRunRemote(t, wt, addArgs...)
	gitRunRemote(t, wt, "commit", "-m", "init upstream")
	gitRunRemote(t, wt, "remote", "add", "origin", bare)
	gitRunRemote(t, wt, "push", "origin", "main")

	return "file://" + bare
}

// fixtureAbs returns the absolute path to a testdata/<name> directory by
// walking up from the cli package working directory.
func fixtureAbs(t *testing.T, rel string) string {
	t.Helper()
	wd, err := os.Getwd() // .../cli
	require.NoError(t, err)
	return filepath.Join(filepath.Dir(wd), "testdata", rel)
}

// threePackageManifest returns a manifest with three packages, each with a
// "default" profile pointing at a single-file source under <pkg>/data.
func threePackageManifest() (manifest string, files map[string]string) {
	manifest = `schema_version = 1

[packages.alpha]
description = "alpha"
supported_os = ["linux", "darwin", "windows"]
[packages.alpha.profiles.default]
sources = [{path = "data", mode = "file", target = "$HOME/.config/alpha"}]

[packages.beta]
description = "beta"
supported_os = ["linux", "darwin", "windows"]
[packages.beta.profiles.default]
sources = [{path = "data", mode = "file", target = "$HOME/.config/beta"}]

[packages.gamma]
description = "gamma"
supported_os = ["linux", "darwin", "windows"]
[packages.gamma.profiles.default]
sources = [{path = "data", mode = "file", target = "$HOME/.config/gamma"}]
`
	files = map[string]string{
		"alpha/data/a.conf": "alpha=true\n",
		"beta/data/b.conf":  "beta=true\n",
		"gamma/data/c.conf": "gamma=true\n",
	}
	return
}

// twoProfileManifest returns a manifest with one package "demo" exposing two
// profiles: "alpha" -> a.conf and "beta" -> b.conf, both targeting the same dir.
func twoProfileManifest() (string, map[string]string) {
	m := `schema_version = 1

[packages.demo]
description = "two-profile demo"
supported_os = ["linux", "darwin", "windows"]

[packages.demo.profiles.alpha]
sources = [{path = "alpha", mode = "file", target = "$HOME/.config/demo"}]

[packages.demo.profiles.beta]
sources = [{path = "beta", mode = "file", target = "$HOME/.config/demo"}]
`
	files := map[string]string{
		"demo/alpha/a.conf": "alpha=true\n",
		"demo/beta/b.conf":  "beta=true\n",
	}
	return m, files
}

// ─────────────────────────────────────────────────────────────────────────────

func TestIntegration_InstallNoArgs_ConvergesAll(t *testing.T) {
	setIsolatedHome(t)
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	mf, files := threePackageManifest()
	makeManagedRepoWithFiles(t, mf, files)

	statePath := filepath.Join(t.TempDir(), "state.json")
	out, err := runRemoteCmd(t, "--state", statePath, "--yes", "install", "--profile", "default")
	require.NoError(t, err, "out=%s", out)

	assert.Contains(t, out, "Installed: alpha")
	assert.Contains(t, out, "Installed: beta")
	assert.Contains(t, out, "Installed: gamma")

	st, err := state.Load(statePath)
	require.NoError(t, err)
	for _, name := range []string{"alpha", "beta", "gamma"} {
		ps, ok := st[name]
		require.True(t, ok, "state must contain %q (have: %v)", name, keys(st))
		assert.Equal(t, "default", ps.Profile)
		require.Len(t, ps.InstalledLinks, 1, "package %s", name)
	}
}

func TestIntegration_InstallWithProfile_Switches(t *testing.T) {
	setIsolatedHome(t)
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	mf, files := twoProfileManifest()
	makeManagedRepoWithFiles(t, mf, files)

	statePath := filepath.Join(t.TempDir(), "state.json")
	home := os.Getenv("HOME")
	tgtA := filepath.Join(home, ".config", "demo", "a.conf")
	tgtB := filepath.Join(home, ".config", "demo", "b.conf")

	out, err := runRemoteCmd(t, "--state", statePath, "--yes",
		"install", "demo", "--profile", "alpha")
	require.NoError(t, err, "out=%s", out)
	_, err = os.Lstat(tgtA)
	require.NoError(t, err, "alpha link missing")

	resetInstallFlags()
	out, err = runRemoteCmd(t, "--state", statePath, "--yes",
		"install", "demo", "--profile", "beta")
	require.NoError(t, err, "out=%s", out)

	_, err = os.Lstat(tgtA)
	assert.True(t, os.IsNotExist(err), "alpha link should be removed after switch, got err=%v", err)
	_, err = os.Lstat(tgtB)
	require.NoError(t, err, "beta link missing after switch")

	st, err := state.Load(statePath)
	require.NoError(t, err)
	ps, ok := st["demo"]
	require.True(t, ok)
	assert.Equal(t, "beta", ps.Profile)
	require.Len(t, ps.InstalledLinks, 1)
	assert.Equal(t, tgtB, ps.InstalledLinks[0].Target)
}

func TestIntegration_InstallSameProfile_NoOp(t *testing.T) {
	setIsolatedHome(t)
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	mf, files := twoProfileManifest()
	makeManagedRepoWithFiles(t, mf, files)
	statePath := filepath.Join(t.TempDir(), "state.json")

	_, err := runRemoteCmd(t, "--state", statePath, "--yes",
		"install", "demo", "--profile", "alpha")
	require.NoError(t, err)

	resetInstallFlags()
	out, err := runRemoteCmd(t, "--state", statePath, "--yes",
		"install", "demo", "--profile", "alpha")
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "already converged")
}

func TestIntegration_InstallRepairsBrokenLink(t *testing.T) {
	setIsolatedHome(t)
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	mf, files := threePackageManifest()
	makeManagedRepoWithFiles(t, mf, files)
	statePath := filepath.Join(t.TempDir(), "state.json")

	_, err := runRemoteCmd(t, "--state", statePath, "--yes", "install", "--profile", "default")
	require.NoError(t, err)

	home := os.Getenv("HOME")
	link := filepath.Join(home, ".config", "alpha", "a.conf")
	require.NoError(t, os.Remove(link), "could not remove link to simulate damage")

	resetInstallFlags()
	out, err := runRemoteCmd(t, "--state", statePath, "--yes", "install", "alpha")
	require.NoError(t, err, "out=%s", out)

	src := filepath.Join(repo.DefaultRepoPath(), "alpha", "data", "a.conf")
	ok, err := symlink.IsSymlinkTo(link, src)
	require.NoError(t, err)
	assert.True(t, ok, "expected repaired symlink %s -> %s", link, src)
}

func TestIntegration_RemoteAddRemoveLifecycle(t *testing.T) {
	setIsolatedHome(t)
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	root := makeManagedRepo(t, "")
	upstream := setupBareUpstream(t, fixtureAbs(t, "remote_rice"))

	out, err := runRemoteCmd(t, "remote", "add", upstream, "--name", "kick")
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Added remote rice")

	out, err = runRemoteCmd(t, "remote", "list")
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "kick")
	assert.Contains(t, out, "remotes/kick")

	out, err = runRemoteCmd(t, "remote", "remove", "kick")
	require.NoError(t, err, "out=%s", out)
	assert.Contains(t, out, "Removed remote rice")

	_, statErr := os.Stat(filepath.Join(root, "remotes", "kick"))
	assert.True(t, os.IsNotExist(statErr), "remotes/kick must be gone")

	logOut, lerr := exec.Command("git", "-C", root, "log", "--oneline").CombinedOutput()
	require.NoError(t, lerr, "git log: %s", logOut)
	assert.Contains(t, string(logOut), "Remove remote rice kick")
}

func TestIntegration_ImportProfileResolves(t *testing.T) {
	setIsolatedHome(t)
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	root := makeManagedRepo(t, "")
	upstream := setupBareUpstream(t, fixtureAbs(t, "remote_rice"))

	_, err := runRemoteCmd(t, "remote", "add", upstream, "--name", "kick")
	require.NoError(t, err)

	importManifest := `schema_version = 1

[packages.nvim]
description = "Neovim configuration (local with import)"
supported_os = ["linux", "darwin", "windows"]

[packages.nvim.profiles.default]
import = "remotes/kick#nvim.default"
`
	require.NoError(t, os.WriteFile(filepath.Join(root, "rice.toml"), []byte(importManifest), 0o644))
	gitRunRemote(t, root, "add", "--", "rice.toml")
	gitRunRemote(t, root, "commit", "-m", "add import")

	statePath := filepath.Join(t.TempDir(), "state.json")
	resetInstallFlags()
	out, err := runRemoteCmd(t, "--state", statePath, "--yes",
		"install", "nvim", "--profile", "default")
	require.NoError(t, err, "out=%s", out)

	home := os.Getenv("HOME")
	target := filepath.Join(home, ".config", "nvim")
	dst, err := os.Readlink(target)
	require.NoError(t, err, "target should be a folder-mode symlink: %s", target)
	expectedSrc := filepath.Join(root, "remotes", "kick", "nvim", "config")
	assert.Equal(t, expectedSrc, dst, "link must point under remotes/<name>/...")
}

func TestIntegration_ImportPlusOverlay(t *testing.T) {
	setIsolatedHome(t)
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	root := makeManagedRepo(t, "")
	upstream := setupBareUpstream(t, fixtureAbs(t, "remote_rice"))

	_, err := runRemoteCmd(t, "remote", "add", upstream, "--name", "kick")
	require.NoError(t, err)

	// Switch the remote nvim package to file mode by rewriting the rice.toml
	// to compose: import (file mode) + local overlay (file mode), so last-wins
	// can take effect. The remote_rice fixture has init.lua under nvim/config/.
	// We add a local overlay nvim/local/init.lua and verify it overrides.
	overlayManifest := `schema_version = 1

[packages.nvim]
description = "import + overlay"
supported_os = ["linux", "darwin", "windows"]
root = "."

[packages.nvim.profiles.default]
sources = [
  {path = "remotes/kick/nvim/config", mode = "file", target = "$HOME/.config/nvim"},
  {path = "nvim_overlay",             mode = "file", target = "$HOME/.config/nvim"},
]
`
	require.NoError(t, os.WriteFile(filepath.Join(root, "rice.toml"), []byte(overlayManifest), 0o644))
	overlayDir := filepath.Join(root, "nvim_overlay")
	require.NoError(t, os.MkdirAll(overlayDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(overlayDir, "init.lua"),
		[]byte("-- local override\n"), 0o644))
	gitRunRemote(t, root, "add", "--", "rice.toml", "nvim_overlay/init.lua")
	gitRunRemote(t, root, "commit", "-m", "add overlay")

	statePath := filepath.Join(t.TempDir(), "state.json")
	resetInstallFlags()
	out, err := runRemoteCmd(t, "--state", statePath, "--yes",
		"install", "nvim", "--profile", "default")
	require.NoError(t, err, "out=%s", out)

	home := os.Getenv("HOME")
	link := filepath.Join(home, ".config", "nvim", "init.lua")
	dst, err := os.Readlink(link)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(overlayDir, "init.lua"), dst,
		"local overlay must win last-wins over imported source")
}

func TestIntegration_RemoteRemoveBlocked_WhenImportExists(t *testing.T) {
	setIsolatedHome(t)
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	root := makeManagedRepo(t, "")
	upstream := setupBareUpstream(t, fixtureAbs(t, "remote_rice"))

	_, err := runRemoteCmd(t, "remote", "add", upstream, "--name", "kick")
	require.NoError(t, err)

	importManifest := `schema_version = 1

[packages.nvim]
description = "import only"
supported_os = ["linux", "darwin", "windows"]

[packages.nvim.profiles.default]
import = "remotes/kick#nvim.default"
`
	require.NoError(t, os.WriteFile(filepath.Join(root, "rice.toml"), []byte(importManifest), 0o644))
	gitRunRemote(t, root, "add", "--", "rice.toml")
	gitRunRemote(t, root, "commit", "-m", "add import")

	out, err := runRemoteCmd(t, "remote", "remove", "kick")
	require.Error(t, err, "out=%s", out)
	assert.ErrorIs(t, err, repo.ErrRemoteInUse)
	// Sanity: error message names the offending profile reference.
	assert.Contains(t, err.Error(), "nvim.default")
}

func TestIntegration_DirtyRepoBlocksRemoteAdd(t *testing.T) {
	setIsolatedHome(t)
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	root := makeManagedRepo(t, "")
	upstream := setupBareUpstream(t, fixtureAbs(t, "remote_rice"))

	// Modify the tracked rice.toml to dirty the working tree.
	require.NoError(t, os.WriteFile(filepath.Join(root, "rice.toml"),
		[]byte("schema_version = 1\n# dirty\n"), 0o644))

	out, err := runRemoteCmd(t, "remote", "add", upstream, "--name", "kick")
	require.Error(t, err, "out=%s", out)
	assert.ErrorIs(t, err, repo.ErrRepoDirty)
}

func TestIntegration_AutoCommitScopedToPaths(t *testing.T) {
	setIsolatedHome(t)
	resetInstallFlags()
	t.Cleanup(resetInstallFlags)

	root := makeManagedRepo(t, "")
	upstream := setupBareUpstream(t, fixtureAbs(t, "remote_rice"))

	out, err := runRemoteCmd(t, "remote", "add", upstream, "--name", "kick")
	require.NoError(t, err, "out=%s", out)

	// stray.txt MUST be created AFTER `remote add`: creating it before would
	// dirty the tree and trip the IsClean guard. Its presence here proves the
	// auto-commit's `git add --` scope did not silently widen.
	stray := filepath.Join(root, "stray.txt")
	require.NoError(t, os.WriteFile(stray, []byte("stray\n"), 0o644))

	logOut, lerr := exec.Command("git", "-C", root,
		"log", "-1", "--name-only", "--pretty=format:").CombinedOutput()
	require.NoError(t, lerr, "git log: %s", logOut)

	files := strings.Fields(strings.TrimSpace(string(logOut)))
	assert.ElementsMatch(t, []string{".gitmodules", "remotes/kick"}, files,
		"auto-commit must include EXACTLY .gitmodules + remotes/<name>; got %v", files)

	statusOut, sErr := exec.Command("git", "-C", root, "status", "--porcelain", "stray.txt").CombinedOutput()
	require.NoError(t, sErr)
	assert.Contains(t, string(statusOut), "?? stray.txt", "stray.txt must remain untracked")
}

// keys returns the sorted keys of a map[string]V for nicer assertion messages.
func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
