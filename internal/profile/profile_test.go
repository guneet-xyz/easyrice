package profile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/manifest"
	"github.com/guneet-xyz/easyrice/internal/repo"
)

func TestResolveSpecs(t *testing.T) {
	tests := []struct {
		name        string
		pkg         *manifest.PackageDef
		pkgName     string
		profileName string
		want        []manifest.SourceSpec
		wantErr     bool
		errContains []string
	}{
		{
			name: "single source",
			pkg: &manifest.PackageDef{
				Profiles: map[string]manifest.ProfileDef{
					"default": {
						Sources: []manifest.SourceSpec{{Path: ".", Mode: "file", Target: "$HOME"}},
					},
				},
			},
			pkgName:     "nvim",
			profileName: "default",
			want:        []manifest.SourceSpec{{Path: ".", Mode: "file", Target: "$HOME"}},
			wantErr:     false,
		},
		{
			name: "multiple sources in order",
			pkg: &manifest.PackageDef{
				Profiles: map[string]manifest.ProfileDef{
					"macbook": {
						Sources: []manifest.SourceSpec{
							{Path: "common", Mode: "file", Target: "$HOME"},
							{Path: "macbook", Mode: "file", Target: "$HOME"},
						},
					},
				},
			},
			pkgName:     "ghostty",
			profileName: "macbook",
			want: []manifest.SourceSpec{
				{Path: "common", Mode: "file", Target: "$HOME"},
				{Path: "macbook", Mode: "file", Target: "$HOME"},
			},
			wantErr: false,
		},
		{
			name: "unknown profile with available profiles",
			pkg: &manifest.PackageDef{
				Profiles: map[string]manifest.ProfileDef{
					"default": {
						Sources: []manifest.SourceSpec{{Path: ".", Mode: "file", Target: "$HOME"}},
					},
					"minimal": {
						Sources: []manifest.SourceSpec{{Path: "minimal", Mode: "file", Target: "$HOME"}},
					},
				},
			},
			pkgName:     "nvim",
			profileName: "unknown",
			want:        nil,
			wantErr:     true,
			errContains: []string{"unknown", "nvim", "default", "minimal"},
		},
		{
			name: "preserves source order",
			pkg: &manifest.PackageDef{
				Profiles: map[string]manifest.ProfileDef{
					"work": {
						Sources: []manifest.SourceSpec{
							{Path: "base", Mode: "file", Target: "$HOME"},
							{Path: "work", Mode: "file", Target: "$HOME"},
							{Path: "secrets", Mode: "file", Target: "$HOME"},
						},
					},
				},
			},
			pkgName:     "zsh",
			profileName: "work",
			want: []manifest.SourceSpec{
				{Path: "base", Mode: "file", Target: "$HOME"},
				{Path: "work", Mode: "file", Target: "$HOME"},
				{Path: "secrets", Mode: "file", Target: "$HOME"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveSpecs("", tt.pkg, tt.pkgName, tt.profileName)

			if tt.wantErr {
				require.Error(t, err)
				errMsg := err.Error()
				for _, substr := range tt.errContains {
					assert.Contains(t, errMsg, substr, "error message should contain %q", substr)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// writeRiceToml writes a rice.toml file at the given path, creating parent dirs.
func writeRiceToml(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestResolveSpecs_Import(t *testing.T) {
	t.Run("import only", func(t *testing.T) {
		root := t.TempDir()
		remoteRoot := filepath.Join(repo.RemotesDir(root), "kick")
		writeRiceToml(t, repo.RemoteTomlPath(root, "kick"), `
schema_version = 1

[packages.nvim]
description = "nvim"
supported_os = ["linux", "darwin", "windows"]

[packages.nvim.profiles.default]
sources = [{path = "config", mode = "folder", target = "$HOME/.config/nvim"}]
`)

		localPkg := &manifest.PackageDef{
			Profiles: map[string]manifest.ProfileDef{
				"default": {Import: "remotes/kick#nvim.default"},
			},
		}
		got, err := ResolveSpecs(root, localPkg, "nvim", "default")
		require.NoError(t, err)
		require.Len(t, got, 1)
		expectedPath := filepath.Join(remoteRoot, "nvim", "config")
		assert.Equal(t, expectedPath, got[0].Path)
		assert.True(t, filepath.IsAbs(got[0].Path))
		assert.Equal(t, "folder", got[0].Mode)
		assert.Equal(t, "$HOME/.config/nvim", got[0].Target)
	})

	t.Run("import plus local overlay", func(t *testing.T) {
		root := t.TempDir()
		writeRiceToml(t, repo.RemoteTomlPath(root, "kick"), `
schema_version = 1

[packages.nvim]
supported_os = ["linux", "darwin", "windows"]

[packages.nvim.profiles.default]
sources = [{path = "config", mode = "file", target = "$HOME/.config/nvim"}]
`)

		localPkg := &manifest.PackageDef{
			Profiles: map[string]manifest.ProfileDef{
				"default": {
					Import: "remotes/kick#nvim.default",
					Sources: []manifest.SourceSpec{
						{Path: "local", Mode: "file", Target: "$HOME/.config/nvim"},
					},
				},
			},
		}
		got, err := ResolveSpecs(root, localPkg, "nvim", "default")
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.True(t, filepath.IsAbs(got[0].Path), "imported spec must be absolute")
		assert.Equal(t, "local", got[1].Path, "local spec must be appended after, unchanged")
	})

	t.Run("cycle detected", func(t *testing.T) {
		root := t.TempDir()
		// remote A imports remote B; remote B imports remote A.
		writeRiceToml(t, repo.RemoteTomlPath(root, "a"), `
schema_version = 1

[packages.p]
supported_os = ["linux", "darwin", "windows"]

[packages.p.profiles.x]
import = "remotes/b#p.x"
`)
		writeRiceToml(t, repo.RemoteTomlPath(root, "b"), `
schema_version = 1

[packages.p]
supported_os = ["linux", "darwin", "windows"]

[packages.p.profiles.x]
import = "remotes/a#p.x"
`)

		localPkg := &manifest.PackageDef{
			Profiles: map[string]manifest.ProfileDef{
				"start": {Import: "remotes/a#p.x"},
			},
		}
		_, err := ResolveSpecs(root, localPkg, "top", "start")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cycle detected")
	})

	t.Run("missing remote", func(t *testing.T) {
		root := t.TempDir()
		localPkg := &manifest.PackageDef{
			Profiles: map[string]manifest.ProfileDef{
				"default": {Import: "remotes/ghost#nvim.default"},
			},
		}
		_, err := ResolveSpecs(root, localPkg, "nvim", "default")
		require.Error(t, err)
		assert.True(t, errors.Is(err, repo.ErrSubmoduleNotInitialized),
			"error chain must include ErrSubmoduleNotInitialized; got: %v", err)
	})
}
