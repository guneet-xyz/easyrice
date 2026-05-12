package manifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseImportSpec(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    ImportSpec
		wantErr bool
		errMsg  string
	}{
		// Valid cases
		{
			name:    "valid import with simple names",
			input:   "remotes/kick#nvim.default",
			want:    ImportSpec{Remote: "kick", Package: "nvim", Profile: "default"},
			wantErr: false,
		},
		{
			name:    "valid import with hyphenated remote",
			input:   "remotes/my-remote#zsh.workmac",
			want:    ImportSpec{Remote: "my-remote", Package: "zsh", Profile: "workmac"},
			wantErr: false,
		},
		{
			name:    "valid import with underscores",
			input:   "remotes/my_remote#nvim_config.dev",
			want:    ImportSpec{Remote: "my_remote", Package: "nvim_config", Profile: "dev"},
			wantErr: false,
		},

		// Invalid cases
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
			errMsg:  "remotes/",
		},
		{
			name:    "missing remotes/ prefix",
			input:   "kick#nvim.default",
			wantErr: true,
			errMsg:  "remotes/",
		},
		{
			name:    "missing # separator",
			input:   "remotes/kicknvim.default",
			wantErr: true,
			errMsg:  "#",
		},
		{
			name:    "missing . separator in pkg.profile",
			input:   "remotes/kick#nvimdefault",
			wantErr: true,
			errMsg:  ".",
		},
		{
			name:    "empty remote name",
			input:   "remotes/#nvim.default",
			wantErr: true,
			errMsg:  "remote name",
		},
		{
			name:    "empty package name",
			input:   "remotes/kick#.default",
			wantErr: true,
			errMsg:  "package name",
		},
		{
			name:    "empty profile name",
			input:   "remotes/kick#nvim.",
			wantErr: true,
			errMsg:  "profile name",
		},
		{
			name:    "multiple # separators",
			input:   "remotes/kick#nvim#default",
			wantErr: true,
			errMsg:  "#",
		},
		{
			name:    "multiple . separators",
			input:   "remotes/kick#nvim.default.extra",
			wantErr: true,
			errMsg:  ".",
		},
		{
			name:    "slash in remote name",
			input:   "remotes/my/remote#nvim.default",
			wantErr: true,
			errMsg:  "slash",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseImportSpec(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}
