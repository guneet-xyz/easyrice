package manifest

import (
	"testing"

	"github.com/guneet-xyz/easyrice/internal/deps"
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
			errSubstr: "must have at least one source or an import",
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

		// Dependency validation: empty dependency name
		{
			name: "empty dependency name",
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
						Dependencies: []deps.DependencyRef{
							{Name: "", Version: ""},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "dependency name is required",
		},

		// Dependency validation: reserved self-dependency
		{
			name: "reserved self-dependency",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages: map[string]PackageDef{
					"neovim": {
						SupportedOS: []string{"linux"},
						Profiles: map[string]ProfileDef{
							"default": {
								Sources: []SourceSpec{
									{Path: "config", Mode: "file", Target: "$HOME/.config/nvim"},
								},
							},
						},
						Dependencies: []deps.DependencyRef{
							{Name: "neovim", Version: ""},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "reserved name",
		},

		// Dependency validation: invalid semver constraint
		{
			name: "invalid semver constraint",
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
						Dependencies: []deps.DependencyRef{
							{Name: "ripgrep", Version: "not-semver"},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "invalid semver constraint",
		},

		// Dependency validation: valid dependency with semver constraint
		{
			name: "valid dependency with semver constraint",
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
						Dependencies: []deps.DependencyRef{
							{Name: "ripgrep", Version: ">=1.0.0"},
						},
					},
				},
			},
			wantErr: false,
		},

		// Custom dependency validation: empty version_probe and install
		{
			name: "custom dependency with empty version_probe and install",
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
				},
				CustomDependencies: map[string]deps.CustomDependencyDef{
					"mycustom": {
						VersionProbe: []string{},
						Install:      map[string]deps.CustomInstallMethod{},
					},
				},
			},
			wantErr:   true,
			errSubstr: "must have at least one of version_probe or install methods",
		},

		// Custom dependency validation: invalid version_regex
		{
			name: "custom dependency with invalid version_regex",
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
				},
				CustomDependencies: map[string]deps.CustomDependencyDef{
					"mycustom": {
						VersionProbe: []string{"echo", "1.0.0"},
						VersionRegex: "[invalid(regex",
					},
				},
			},
			wantErr:   true,
			errSubstr: "version_regex does not compile",
		},

		// Custom dependency validation: empty shell_payload
		{
			name: "custom dependency with empty shell_payload",
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
				},
				CustomDependencies: map[string]deps.CustomDependencyDef{
					"mycustom": {
						Install: map[string]deps.CustomInstallMethod{
							"linux": {
								Description:  "Install on Linux",
								ShellPayload: "",
							},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "shell_payload must not be empty",
		},

		// Custom dependency validation: reserved name collision
		{
			name: "custom dependency with reserved name",
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
				},
				CustomDependencies: map[string]deps.CustomDependencyDef{
					"neovim": {
						VersionProbe: []string{"nvim", "--version"},
					},
				},
			},
			wantErr:   true,
			errSubstr: "name is reserved",
		},

		// Custom dependency validation: registry name collision
		{
			name: "custom dependency with registry name",
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
				},
				CustomDependencies: map[string]deps.CustomDependencyDef{
					"nvm": {
						VersionProbe: []string{"nvm", "--version"},
					},
				},
			},
			wantErr:   true,
			errSubstr: "name is already in the registry",
		},

		// Custom dependency validation: valid custom dependency
		{
			name: "valid custom dependency",
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
				},
				CustomDependencies: map[string]deps.CustomDependencyDef{
					"mycustom": {
						VersionProbe: []string{"mycustom", "--version"},
						VersionRegex: `v(\d+\.\d+\.\d+)`,
						Install: map[string]deps.CustomInstallMethod{
							"linux": {
								Description:  "Install on Linux",
								ShellPayload: "curl -fsSL https://example.com/install.sh | sh",
							},
						},
					},
				},
			},
			wantErr: false,
		},

		// Custom dependency edge case: only-probe (no install methods)
		{
			name: "custom dependency with only version_probe (no install)",
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
				},
				CustomDependencies: map[string]deps.CustomDependencyDef{
					"mycustom": {
						VersionProbe: []string{"mycustom", "--version"},
						VersionRegex: `v(\d+\.\d+\.\d+)`,
						Install:      map[string]deps.CustomInstallMethod{},
					},
				},
			},
			wantErr: false,
		},

		// Custom dependency edge case: only-install (no version_probe)
		{
			name: "custom dependency with only install methods (no version_probe)",
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
				},
				CustomDependencies: map[string]deps.CustomDependencyDef{
					"mycustom": {
						VersionProbe: []string{},
						Install: map[string]deps.CustomInstallMethod{
							"linux": {
								Description:  "Install on Linux",
								ShellPayload: "curl -fsSL https://example.com/install.sh | sh",
							},
						},
					},
				},
			},
			wantErr: false,
		},

		// Custom dependency edge case: multiple install methods
		{
			name: "custom dependency with multiple install methods",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages: map[string]PackageDef{
					"nvim": {
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
				CustomDependencies: map[string]deps.CustomDependencyDef{
					"mycustom": {
						VersionProbe: []string{"mycustom", "--version"},
						Install: map[string]deps.CustomInstallMethod{
							"linux": {
								Description:  "Install on Linux",
								ShellPayload: "apt-get install mycustom",
							},
							"darwin": {
								Description:  "Install on macOS",
								ShellPayload: "brew install mycustom",
							},
						},
					},
				},
			},
			wantErr: false,
		},

		// Custom dependency edge case: valid complex regex pattern
		{
			name: "custom dependency with complex valid regex",
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
				},
				CustomDependencies: map[string]deps.CustomDependencyDef{
					"mycustom": {
						VersionProbe: []string{"mycustom", "--version"},
						VersionRegex: `^v?(\d+)\.(\d+)\.(\d+)(?:-([a-zA-Z0-9]+))?$`,
						Install: map[string]deps.CustomInstallMethod{
							"linux": {
								Description:  "Install on Linux",
								ShellPayload: "curl -fsSL https://example.com/install.sh | sh",
							},
						},
					},
				},
			},
			wantErr: false,
		},

		// Custom dependency edge case: invalid regex with special chars
		{
			name: "custom dependency with invalid regex (unclosed bracket)",
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
				},
				CustomDependencies: map[string]deps.CustomDependencyDef{
					"mycustom": {
						VersionProbe: []string{"mycustom", "--version"},
						VersionRegex: `(\d+\.\d+\.\d+`,
						Install: map[string]deps.CustomInstallMethod{
							"linux": {
								Description:  "Install on Linux",
								ShellPayload: "curl -fsSL https://example.com/install.sh | sh",
							},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "version_regex does not compile",
		},

		// Custom dependency edge case: empty shell_payload in one of multiple methods
		{
			name: "custom dependency with empty shell_payload in second method",
			manifest: &Manifest{
				SchemaVersion: 1,
				Packages: map[string]PackageDef{
					"nvim": {
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
				CustomDependencies: map[string]deps.CustomDependencyDef{
					"mycustom": {
						VersionProbe: []string{"mycustom", "--version"},
						Install: map[string]deps.CustomInstallMethod{
							"linux": {
								Description:  "Install on Linux",
								ShellPayload: "apt-get install mycustom",
							},
							"darwin": {
								Description:  "Install on macOS",
								ShellPayload: "",
							},
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "shell_payload must not be empty",
		},

		// Custom dependency edge case: no version_regex but with version_probe
		{
			name: "custom dependency with version_probe but no version_regex",
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
				},
				CustomDependencies: map[string]deps.CustomDependencyDef{
					"mycustom": {
						VersionProbe: []string{"mycustom", "--version"},
						VersionRegex: "",
						Install: map[string]deps.CustomInstallMethod{
							"linux": {
								Description:  "Install on Linux",
								ShellPayload: "curl -fsSL https://example.com/install.sh | sh",
							},
						},
					},
				},
			},
			wantErr: false,
		},

		// Custom dependency edge case: install method with distro_families
		{
			name: "custom dependency with install method having distro_families",
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
				},
				CustomDependencies: map[string]deps.CustomDependencyDef{
					"mycustom": {
						VersionProbe: []string{"mycustom", "--version"},
						Install: map[string]deps.CustomInstallMethod{
							"debian": {
								Description:    "Install on Debian-based systems",
								ShellPayload:   "apt-get install mycustom",
								DistroFamilies: []string{"debian", "ubuntu"},
							},
						},
					},
				},
			},
			wantErr: false,
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
