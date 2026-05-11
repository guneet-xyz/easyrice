package profile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/manifest"
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
			got, err := ResolveSpecs(tt.pkg, tt.pkgName, tt.profileName)

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
