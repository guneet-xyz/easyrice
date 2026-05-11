package manifest

import (
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchema_RejectsNonStringFields covers the type-mismatch branches in
// SourceSpec.UnmarshalTOML for path, mode, and target.
func TestSchema_RejectsNonStringFields(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantSub string
	}{
		{
			name: "path is integer",
			body: `schema_version = 1
[packages.p]
description = ""
supported_os = ["darwin"]
[packages.p.profiles.default]
sources = [{path = 7, mode = "file", target = "$HOME"}]
`,
			wantSub: `"path" must be a string`,
		},
		{
			name: "mode is bool",
			body: `schema_version = 1
[packages.p]
description = ""
supported_os = ["darwin"]
[packages.p.profiles.default]
sources = [{path = "x", mode = true, target = "$HOME"}]
`,
			wantSub: `"mode" must be a string`,
		},
		{
			name: "target is integer",
			body: `schema_version = 1
[packages.p]
description = ""
supported_os = ["darwin"]
[packages.p.profiles.default]
sources = [{path = "x", mode = "file", target = 42}]
`,
			wantSub: `"target" must be a string`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m Manifest
			_, err := toml.Decode(tc.body, &m)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantSub)
		})
	}
}
