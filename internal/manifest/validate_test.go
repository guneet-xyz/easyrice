package manifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	cases := []struct {
		name      string
		manifest  *Manifest
		wantErr   bool
		errSubstr string
	}{
		// Happy path: valid manifest with one package and one profile
		{
			name: "valid manifest with one package",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages: map[string]PackageDef{
					"nvim": {
						Description: "Neovim config",
						SupportedOS: []string{"linux", "darwin"},
						Profiles: map[string]ProfileDef{
							"default": {
								Sources: []SourceSpec{
									{Path: "config", Mode: "file", Target: "$HOME/.config/nvim"},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},

		// Happy path: multiple packages
		{
			name: "valid manifest with multiple packages",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages: map[string]PackageDef{
					"nvim": {
						SupportedOS: []string{"linux"},
						Profiles: map[string]ProfileDef{
							"default": {
								Sources: []SourceSpec{
									{Path: "config", Mode: "file", Target: "$HOME/.config/nvim"},
								},
							},
						},
					},
					"zsh": {
						SupportedOS: []string{"darwin"},
						Profiles: map[string]ProfileDef{
							"work": {
								Sources: []SourceSpec{
									{Path: "zshrc", Mode: "file", Target: "$HOME/.zshrc"},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},

		// Happy path: folder mode
		{
			name: "valid manifest with folder mode",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages: map[string]PackageDef{
					"ghostty": {
						SupportedOS: []string{"darwin"},
						Profiles: map[string]ProfileDef{
							"default": {
								Sources: []SourceSpec{
									{Path: "config", Mode: "folder", Target: "$HOME/.config/ghostty"},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},

		// Happy path: with optional Root field
		{
			name: "valid manifest with root field",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages: map[string]PackageDef{
					"nvim": {
						Root:        "nvim-config",
						SupportedOS: []string{"linux"},
						Profiles: map[string]ProfileDef{
							"default": {
								Sources: []SourceSpec{
									{Path: "config", Mode: "file", Target: "$HOME/.config/nvim"},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},

		// Negative: bad schema version
		{
			name: "invalid schema version 0",
			manifest: &Manifest{
				SchemaVersion: 0,
				Packages: map[string]PackageDef{
					"nvim": {
						SupportedOS: []string{"linux"},
						Profiles: map[string]ProfileDef{
							"default": {
								Sources: []SourceSpec{
									{Path: "config", Mode: "file", Target: "$HOME/.config/nvim"},
								},
							},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "unsupported schema_version",
		},

		// Negative: no packages
		{
			name: "no packages declared",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages:      map[string]PackageDef{},
			},
			wantErr:   true,
			errSubstr: "no packages declared",
		},

		// Negative: empty package name
		{
			name: "empty package name",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages: map[string]PackageDef{
					"": {
						SupportedOS: []string{"linux"},
						Profiles: map[string]ProfileDef{
							"default": {
								Sources: []SourceSpec{
									{Path: "config", Mode: "file", Target: "$HOME/.config/nvim"},
								},
							},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "package name must not be empty",
		},

		// Negative: package name with slash
		{
			name: "package name with slash",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages: map[string]PackageDef{
					"nvim/config": {
						SupportedOS: []string{"linux"},
						Profiles: map[string]ProfileDef{
							"default": {
								Sources: []SourceSpec{
									{Path: "config", Mode: "file", Target: "$HOME/.config/nvim"},
								},
							},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "must not contain slashes",
		},

		// Negative: package name with whitespace
		{
			name: "package name with whitespace",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages: map[string]PackageDef{
					"nvim config": {
						SupportedOS: []string{"linux"},
						Profiles: map[string]ProfileDef{
							"default": {
								Sources: []SourceSpec{
									{Path: "config", Mode: "file", Target: "$HOME/.config/nvim"},
								},
							},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "must not contain whitespace",
		},

		// Negative: empty supported_os
		{
			name: "empty supported_os",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages: map[string]PackageDef{
					"nvim": {
						SupportedOS: []string{},
						Profiles: map[string]ProfileDef{
							"default": {
								Sources: []SourceSpec{
									{Path: "config", Mode: "file", Target: "$HOME/.config/nvim"},
								},
							},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "supported_os must not be empty",
		},

		// Negative: invalid OS
		{
			name: "invalid OS",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages: map[string]PackageDef{
					"nvim": {
						SupportedOS: []string{"freebsd"},
						Profiles: map[string]ProfileDef{
							"default": {
								Sources: []SourceSpec{
									{Path: "config", Mode: "file", Target: "$HOME/.config/nvim"},
								},
							},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "unsupported OS",
		},

		// Negative: root starts with /
		{
			name: "root starts with /",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages: map[string]PackageDef{
					"nvim": {
						Root:        "/absolute/path",
						SupportedOS: []string{"linux"},
						Profiles: map[string]ProfileDef{
							"default": {
								Sources: []SourceSpec{
									{Path: "config", Mode: "file", Target: "$HOME/.config/nvim"},
								},
							},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "root must not start with /",
		},

		// Negative: root contains ..
		{
			name: "root contains ..",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages: map[string]PackageDef{
					"nvim": {
						Root:        "config/../other",
						SupportedOS: []string{"linux"},
						Profiles: map[string]ProfileDef{
							"default": {
								Sources: []SourceSpec{
									{Path: "config", Mode: "file", Target: "$HOME/.config/nvim"},
								},
							},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "root must not contain .. segments",
		},

		// Negative: no profiles
		{
			name: "no profiles",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages: map[string]PackageDef{
					"nvim": {
						SupportedOS: []string{"linux"},
						Profiles:    map[string]ProfileDef{},
					},
				},
			},
			wantErr:   true,
			errSubstr: "at least one profile must be defined",
		},

		// Negative: empty profile name
		{
			name: "empty profile name",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages: map[string]PackageDef{
					"nvim": {
						SupportedOS: []string{"linux"},
						Profiles: map[string]ProfileDef{
							"": {
								Sources: []SourceSpec{
									{Path: "config", Mode: "file", Target: "$HOME/.config/nvim"},
								},
							},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "profile name must not be empty",
		},

		// Negative: no sources in profile
		{
			name: "no sources in profile",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages: map[string]PackageDef{
					"nvim": {
						SupportedOS: []string{"linux"},
						Profiles: map[string]ProfileDef{
							"default": {
								Sources: []SourceSpec{},
							},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "has no sources",
		},

		// Negative: empty source path
		{
			name: "empty source path",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages: map[string]PackageDef{
					"nvim": {
						SupportedOS: []string{"linux"},
						Profiles: map[string]ProfileDef{
							"default": {
								Sources: []SourceSpec{
									{Path: "", Mode: "file", Target: "$HOME/.config/nvim"},
								},
							},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "path must not be empty",
		},

		// Negative: source path with ..
		{
			name: "source path with ..",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages: map[string]PackageDef{
					"nvim": {
						SupportedOS: []string{"linux"},
						Profiles: map[string]ProfileDef{
							"default": {
								Sources: []SourceSpec{
									{Path: "config/../other", Mode: "file", Target: "$HOME/.config/nvim"},
								},
							},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "must not contain .. segments",
		},

		// Negative: source path with leading /
		{
			name: "source path with leading /",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages: map[string]PackageDef{
					"nvim": {
						SupportedOS: []string{"linux"},
						Profiles: map[string]ProfileDef{
							"default": {
								Sources: []SourceSpec{
									{Path: "/absolute/path", Mode: "file", Target: "$HOME/.config/nvim"},
								},
							},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "must be relative",
		},

		// Negative: invalid mode
		{
			name: "invalid mode",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages: map[string]PackageDef{
					"nvim": {
						SupportedOS: []string{"linux"},
						Profiles: map[string]ProfileDef{
							"default": {
								Sources: []SourceSpec{
									{Path: "config", Mode: "symlink", Target: "$HOME/.config/nvim"},
								},
							},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "mode must be \"file\" or \"folder\"",
		},

		// Negative: empty target
		{
			name: "empty target",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages: map[string]PackageDef{
					"nvim": {
						SupportedOS: []string{"linux"},
						Profiles: map[string]ProfileDef{
							"default": {
								Sources: []SourceSpec{
									{Path: "config", Mode: "file", Target: ""},
								},
							},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "target must not be empty",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.manifest)
			if tc.wantErr {
				assert.Error(t, err, "expected error but got nil")
				if tc.errSubstr != "" {
					assert.Contains(t, err.Error(), tc.errSubstr, "error message should contain substring")
				}
			} else {
				assert.NoError(t, err, "expected no error but got: %v", err)
			}
		})
	}
}
