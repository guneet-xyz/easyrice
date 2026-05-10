package manifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name     string
		manifest *Manifest
		wantErr  bool
		errMsg   string
	}{
		// Rule 1: SchemaVersion must be 1
		{
			name: "valid schema version 1",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "file.txt", Mode: "file", Target: "$HOME"}}},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid schema version 0",
			manifest: &Manifest{
				SchemaVersion: 0,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "file.txt", Mode: "file", Target: "$HOME"}}},
				},
			},
			wantErr: true,
			errMsg:  "unsupported schema_version: 0",
		},
		{
			name: "invalid schema version 2",
			manifest: &Manifest{
				SchemaVersion: 2,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "file.txt", Mode: "file", Target: "$HOME"}}},
				},
			},
			wantErr: true,
			errMsg:  "unsupported schema_version: 2",
		},

		// Rule 2: Name must be non-empty
		{
			name: "valid non-empty name",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "mypackage",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "file.txt", Mode: "file", Target: "$HOME"}}},
				},
			},
			wantErr: false,
		},
		{
			name: "empty name",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "file.txt", Mode: "file", Target: "$HOME"}}},
				},
			},
			wantErr: true,
			errMsg:  "name is required and must not be empty",
		},
		{
			name: "whitespace-only name",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "   ",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "file.txt", Mode: "file", Target: "$HOME"}}},
				},
			},
			wantErr: true,
			errMsg:  "name is required and must not be empty",
		},

		// Rule 3: SupportedOS must be non-empty and each element must be in {linux, darwin, windows}
		{
			name: "valid single OS linux",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "file.txt", Mode: "file", Target: "$HOME"}}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid single OS darwin",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"darwin"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "file.txt", Mode: "file", Target: "$HOME"}}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid single OS windows",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"windows"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "file.txt", Mode: "file", Target: "$HOME"}}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid multiple OS",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux", "darwin", "windows"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "file.txt", Mode: "file", Target: "$HOME"}}},
				},
			},
			wantErr: false,
		},
		{
			name: "empty supported_os",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "file.txt", Mode: "file", Target: "$HOME"}}},
				},
			},
			wantErr: true,
			errMsg:  "supported_os must not be empty",
		},
		{
			name: "invalid OS freebsd",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"freebsd"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "file.txt", Mode: "file", Target: "$HOME"}}},
				},
			},
			wantErr: true,
			errMsg:  "unsupported OS: \"freebsd\"",
		},
		{
			name: "mixed valid and invalid OS",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux", "macos"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "file.txt", Mode: "file", Target: "$HOME"}}},
				},
			},
			wantErr: true,
			errMsg:  "unsupported OS: \"macos\"",
		},

		// Rule 4: At least one profile must be defined; each profile's Sources must be non-empty
		{
			name: "valid single profile",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "file.txt", Mode: "file", Target: "$HOME"}}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid multiple profiles",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"personal": {Sources: []SourceSpec{{Path: "file1.txt", Mode: "file", Target: "$HOME"}}},
					"work":     {Sources: []SourceSpec{{Path: "file2.txt", Mode: "file", Target: "$HOME"}}},
				},
			},
			wantErr: false,
		},
		{
			name: "no profiles",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles:      map[string]ProfileDef{},
			},
			wantErr: true,
			errMsg:  "at least one profile must be defined",
		},
		{
			name: "profile with empty sources",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{}},
				},
			},
			wantErr: true,
			errMsg:  "profile \"default\" has no sources",
		},
		{
			name: "one profile valid, one with empty sources",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"personal": {Sources: []SourceSpec{{Path: "file.txt", Mode: "file", Target: "$HOME"}}},
					"work":     {Sources: []SourceSpec{}},
				},
			},
			wantErr: true,
			errMsg:  "profile \"work\" has no sources",
		},

		// Rule 5: Each Sources entry must be a relative path (no leading /, no .. segments)
		{
			name: "valid relative path",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "config/file.txt", Mode: "file", Target: "$HOME"}}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid nested relative path",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "a/b/c/file.txt", Mode: "file", Target: "$HOME"}}},
				},
			},
			wantErr: false,
		},
		{
			name: "absolute path with leading slash",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "/etc/config", Mode: "file", Target: "$HOME"}}},
				},
			},
			wantErr: true,
			errMsg:  "must be a relative path (no leading /)",
		},
		{
			name: "path with .. segment",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "../config/file.txt", Mode: "file", Target: "$HOME"}}},
				},
			},
			wantErr: true,
			errMsg:  "must not contain .. segments",
		},
		{
			name: "path with .. in middle",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "a/../b/file.txt", Mode: "file", Target: "$HOME"}}},
				},
			},
			wantErr: true,
			errMsg:  "must not contain .. segments",
		},

		// Source mode and target tests
		{
			name: "folder-mode source with valid target",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "nvim", Mode: "folder", Target: "$HOME/.config/nvim"}}},
				},
			},
			wantErr: false,
		},
		{
			name: "folder-mode source missing target",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "nvim", Mode: "folder"}}},
				},
			},
			wantErr: true,
			errMsg:  "target is required",
		},
		{
			name: "file-mode source with valid target",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "config.txt", Mode: "file", Target: "$HOME/.config"}}},
				},
			},
			wantErr: false,
		},
		{
			name: "file-mode source missing target",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "config.txt", Mode: "file"}}},
				},
			},
			wantErr: true,
			errMsg:  "target is required",
		},
		{
			name: "empty mode is invalid",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "config.txt", Mode: "", Target: "$HOME/.config"}}},
				},
			},
			wantErr: true,
			errMsg:  "mode must be \"file\" or \"folder\"",
		},
		{
			name: "unknown mode value",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "nvim", Mode: "symlink"}}},
				},
			},
			wantErr: true,
			errMsg:  "mode must be \"file\" or \"folder\"",
		},
		{
			name: "source target with invalid prefix",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "config.txt", Mode: "file", Target: "/etc/config"}}},
				},
			},
			wantErr: true,
			errMsg:  "must start with one of",
		},

		// Integration tests: multiple rules
		{
			name: "comprehensive valid manifest",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "nvim",
				Description:   "Neovim configuration",
				SupportedOS:   []string{"linux", "darwin"},
				ProfileKey:    "nvim_profile",
				Profiles: map[string]ProfileDef{
					"personal": {Sources: []SourceSpec{{Path: "init.lua", Mode: "file", Target: "$HOME/.config/nvim"}, {Path: "lua/config.lua", Mode: "file", Target: "$HOME/.config/nvim"}}},
					"work":     {Sources: []SourceSpec{{Path: "init.lua", Mode: "file", Target: "$HOME/.config/nvim"}, {Path: "lua/work.lua", Mode: "file", Target: "$HOME/.config/nvim"}}},
				},
			},
			wantErr: false,
		},
		{
			name: "comprehensive invalid manifest (multiple errors, first caught)",
			manifest: &Manifest{
				SchemaVersion: 2,
				Name:          "",
				SupportedOS:   []string{},
				Profiles:      map[string]ProfileDef{},
			},
			wantErr: true,
			errMsg:  "unsupported schema_version: 2",
		},

		// Source mode and target tests
		{
			name: "folder-mode source with valid target",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "nvim", Mode: "folder", Target: "$HOME/.config/nvim"}}},
				},
			},
			wantErr: false,
		},
		{
			name: "folder-mode source missing target",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "nvim", Mode: "folder"}}},
				},
			},
			wantErr: true,
			errMsg:  "target is required",
		},
		{
			name: "file-mode source with valid target",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "config.txt", Mode: "file", Target: "$HOME/.config"}}},
				},
			},
			wantErr: false,
		},
		{
			name: "file-mode source missing target",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "config.txt", Mode: "file"}}},
				},
			},
			wantErr: true,
			errMsg:  "target is required",
		},
		{
			name: "empty mode is invalid",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "config.txt", Mode: "", Target: "$HOME/.config"}}},
				},
			},
			wantErr: true,
			errMsg:  "mode must be \"file\" or \"folder\"",
		},
		{
			name: "unknown mode value",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "nvim", Mode: "symlink"}}},
				},
			},
			wantErr: true,
			errMsg:  "mode must be \"file\" or \"folder\"",
		},
		{
			name: "source target with invalid prefix",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []SourceSpec{{Path: "config.txt", Mode: "file", Target: "/etc/config"}}},
				},
			},
			wantErr: true,
			errMsg:  "must start with one of",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.manifest)
			if tt.wantErr {
				assert.Error(t, err, "expected error but got none")
				assert.Contains(t, err.Error(), tt.errMsg, "error message mismatch")
			} else {
				assert.NoError(t, err, "expected no error but got: %v", err)
			}
		})
	}
}
