// Multi-remote bug-hunting tests (BUG-100..BUG-102).
//
// These tests use the internal/testutil/multiremote helper to build a parent
// repo with file:// submodules and exercise install paths that span remotes.
// Subtests are prefixed BUG-NNN so the pre-commit gate's failure filter
// (grep -vE 'BUG[-_]') can distinguish expected catalogued failures from
// real regressions.
package installer_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/installer"
	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/profile"
	"github.com/guneet-xyz/easyrice/internal/testutil/goldenfs"
	"github.com/guneet-xyz/easyrice/internal/testutil/multiremote"
)

func installPackage(t *testing.T, repoRoot, pkgName, profileName string) (*installer.InstallResult, string, error) {
	t.Helper()
	mf, err := manifest.LoadFile(filepath.Join(repoRoot, "rice.toml"))
	require.NoError(t, err, "load parent rice.toml")
	pkg, ok := mf.Packages[pkgName]
	require.True(t, ok, "package %q present in parent manifest", pkgName)

	specs, err := profile.ResolveSpecs(repoRoot, &pkg, pkgName, profileName)
	if err != nil {
		return nil, "", err
	}

	home := t.TempDir()
	statePath := filepath.Join(t.TempDir(), "state.json")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	res, err := installer.Install(installer.InstallRequest{
		RepoRoot:    repoRoot,
		PackageName: pkgName,
		ProfileName: profileName,
		Pkg:         &pkg,
		Specs:       specs,
		CurrentOS:   runtime.GOOS,
		HomeDir:     home,
		StatePath:   statePath,
	})
	return res, home, err
}

func TestMultiRemoteInstallBugs(t *testing.T) {
	t.Run("BUG-100-ThreeRemoteComposition", func(t *testing.T) {
		t.Log("BUG-100")
		// Parent declares three packages, each importing from a different remote.
		// Installing each must produce non-conflicting link sets per remote.
		alphaToml := `schema_version = 1

[packages.alpha]
description = "alpha remote"
supported_os = ["linux", "darwin", "windows"]
root = "."

[packages.alpha.profiles.default]
sources = [{path = "files", mode = "file", target = "$HOME/alpha"}]
`
		betaToml := `schema_version = 1

[packages.beta]
description = "beta remote"
supported_os = ["linux", "darwin", "windows"]
root = "."

[packages.beta.profiles.default]
sources = [{path = "files", mode = "file", target = "$HOME/beta"}]
`
		gammaToml := `schema_version = 1

[packages.gamma]
description = "gamma remote"
supported_os = ["linux", "darwin", "windows"]
root = "."

[packages.gamma.profiles.default]
sources = [{path = "files", mode = "file", target = "$HOME/gamma"}]
`
		fx := multiremote.New(t).
			AddRemoteRaw("alpha", map[string]string{
				"rice.toml":   alphaToml,
				"files/a.txt": "alpha\n",
			}).
			AddRemoteRaw("beta", map[string]string{
				"rice.toml":   betaToml,
				"files/b.txt": "beta\n",
			}).
			AddRemoteRaw("gamma", map[string]string{
				"rice.toml":   gammaToml,
				"files/c.txt": "gamma\n",
			}).
			WithParentManifest(`schema_version = 1

[packages.alpha_local]
description = "wraps alpha"
supported_os = ["linux", "darwin", "windows"]

[packages.alpha_local.profiles.default]
import = "remotes/alpha#alpha.default"

[packages.beta_local]
description = "wraps beta"
supported_os = ["linux", "darwin", "windows"]

[packages.beta_local.profiles.default]
import = "remotes/beta#beta.default"

[packages.gamma_local]
description = "wraps gamma"
supported_os = ["linux", "darwin", "windows"]

[packages.gamma_local.profiles.default]
import = "remotes/gamma#gamma.default"
`).Build()
		defer fx.Cleanup()

		_, home1, err := installPackage(t, fx.ParentRepoPath, "alpha_local", "default")
		require.NoError(t, err, "BUG-100: install alpha_local must succeed")
		_, err1 := os.Lstat(filepath.Join(home1, "alpha", "a.txt"))
		assert.NoError(t, err1, "BUG-100: alpha link must exist")

		_, home2, err := installPackage(t, fx.ParentRepoPath, "beta_local", "default")
		require.NoError(t, err, "BUG-100: install beta_local must succeed")
		_, err2 := os.Lstat(filepath.Join(home2, "beta", "b.txt"))
		assert.NoError(t, err2, "BUG-100: beta link must exist")

		_, home3, err := installPackage(t, fx.ParentRepoPath, "gamma_local", "default")
		require.NoError(t, err, "BUG-100: install gamma_local must succeed")
		_, err3 := os.Lstat(filepath.Join(home3, "gamma", "c.txt"))
		assert.NoError(t, err3, "BUG-100: gamma link must exist")
	})

	t.Run("BUG-101-CrossRemoteOverlay", func(t *testing.T) {
		t.Log("BUG-101")
		// Remote profile has two file-mode sources targeting the same dir;
		// last-source-wins must hold across the imported spec list.
		remoteToml := `schema_version = 1

[packages.overlay]
description = "remote with internal overlay"
supported_os = ["linux", "darwin", "windows"]
root = "."

[packages.overlay.profiles.default]
sources = [
  {path = "base",     mode = "file", target = "$HOME/overlay"},
  {path = "override", mode = "file", target = "$HOME/overlay"},
]
`
		fx := multiremote.New(t).
			AddRemoteRaw("ov", map[string]string{
				"rice.toml":       remoteToml,
				"base/config":     "base-version\n",
				"override/config": "override-version\n",
			}).
			WithParentManifest(`schema_version = 1

[packages.wrap]
description = "imports overlay"
supported_os = ["linux", "darwin", "windows"]

[packages.wrap.profiles.default]
import = "remotes/ov#overlay.default"
`).Build()
		defer fx.Cleanup()

		_, home, err := installPackage(t, fx.ParentRepoPath, "wrap", "default")
		require.NoError(t, err, "BUG-101: cross-remote overlay install should succeed")

		linkPath := filepath.Join(home, "overlay", "config")
		target, err := os.Readlink(linkPath)
		require.NoError(t, err, "BUG-101: overlay link must exist at %s", linkPath)
		assert.Contains(t, target, "override", "BUG-101: last source wins - link should point into override (got %q)", target)
	})

	t.Run("BUG-102-DeepOverlayGolden", func(t *testing.T) {
		t.Log("BUG-102")
		// Three-layer overlay inside a single remote; snapshot the resulting tree.
		remoteToml := `schema_version = 1

[packages.deep]
description = "three-layer overlay"
supported_os = ["linux", "darwin", "windows"]
root = "."

[packages.deep.profiles.default]
sources = [
  {path = "l1", mode = "file", target = "$HOME/deep"},
  {path = "l2", mode = "file", target = "$HOME/deep"},
  {path = "l3", mode = "file", target = "$HOME/deep"},
]
`
		fx := multiremote.New(t).
			AddRemoteRaw("dp", map[string]string{
				"rice.toml": remoteToml,
				"l1/a":      "l1-a\n",
				"l1/b":      "l1-b\n",
				"l2/b":      "l2-b\n",
				"l2/c":      "l2-c\n",
				"l3/c":      "l3-c\n",
				"l3/d":      "l3-d\n",
			}).
			WithParentManifest(`schema_version = 1

[packages.wrap]
description = "imports deep"
supported_os = ["linux", "darwin", "windows"]

[packages.wrap.profiles.default]
import = "remotes/dp#deep.default"
`).Build()
		defer fx.Cleanup()

		_, home, err := installPackage(t, fx.ParentRepoPath, "wrap", "default")
		require.NoError(t, err, "BUG-102: deep overlay install should succeed")

		snap := goldenfs.Snapshot(t, filepath.Join(home, "deep"))
		goldenPath := filepath.Join("testdata", "multiremote_deep_overlay.golden")
		goldenfs.AssertGolden(t, snap, goldenPath)
	})
}
