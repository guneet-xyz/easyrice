package manifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckOS(t *testing.T) {
	cases := []struct {
		name        string
		pkgName     string
		pkgDef      *PackageDef
		currentOS   string
		wantErr     bool
		errSubstr   string
	}{
		// Happy path: single OS matches
		{
			name:      "supported_os=[linux] on linux",
			pkgName:   "nvim",
			pkgDef:    &PackageDef{SupportedOS: []string{"linux"}},
			currentOS: "linux",
			wantErr:   false,
		},
		{
			name:      "supported_os=[darwin] on darwin",
			pkgName:   "ghostty",
			pkgDef:    &PackageDef{SupportedOS: []string{"darwin"}},
			currentOS: "darwin",
			wantErr:   false,
		},
		{
			name:      "supported_os=[windows] on windows",
			pkgName:   "powershell",
			pkgDef:    &PackageDef{SupportedOS: []string{"windows"}},
			currentOS: "windows",
			wantErr:   false,
		},

		// Happy path: multiple OS, one matches
		{
			name:      "supported_os=[linux,darwin] on darwin",
			pkgName:   "zsh",
			pkgDef:    &PackageDef{SupportedOS: []string{"linux", "darwin"}},
			currentOS: "darwin",
			wantErr:   false,
		},
		{
			name:      "supported_os=[linux,darwin] on linux",
			pkgName:   "zsh",
			pkgDef:    &PackageDef{SupportedOS: []string{"linux", "darwin"}},
			currentOS: "linux",
			wantErr:   false,
		},

		// Happy path: all OS supported
		{
			name:      "supported_os=[linux,darwin,windows] on linux",
			pkgName:   "universal",
			pkgDef:    &PackageDef{SupportedOS: []string{"linux", "darwin", "windows"}},
			currentOS: "linux",
			wantErr:   false,
		},
		{
			name:      "supported_os=[linux,darwin,windows] on darwin",
			pkgName:   "universal",
			pkgDef:    &PackageDef{SupportedOS: []string{"linux", "darwin", "windows"}},
			currentOS: "darwin",
			wantErr:   false,
		},
		{
			name:      "supported_os=[linux,darwin,windows] on windows",
			pkgName:   "universal",
			pkgDef:    &PackageDef{SupportedOS: []string{"linux", "darwin", "windows"}},
			currentOS: "windows",
			wantErr:   false,
		},

		// Error path: OS not supported
		{
			name:      "supported_os=[linux] on darwin",
			pkgName:   "nvim",
			pkgDef:    &PackageDef{SupportedOS: []string{"linux"}},
			currentOS: "darwin",
			wantErr:   true,
			errSubstr: "does not support darwin",
		},
		{
			name:      "supported_os=[darwin] on linux",
			pkgName:   "ghostty",
			pkgDef:    &PackageDef{SupportedOS: []string{"darwin"}},
			currentOS: "linux",
			wantErr:   true,
			errSubstr: "does not support linux",
		},
		{
			name:      "supported_os=[linux,darwin] on windows",
			pkgName:   "zsh",
			pkgDef:    &PackageDef{SupportedOS: []string{"linux", "darwin"}},
			currentOS: "windows",
			wantErr:   true,
			errSubstr: "does not support windows",
		},

		// Error path: empty supported_os
		{
			name:      "supported_os=[] (empty)",
			pkgName:   "broken",
			pkgDef:    &PackageDef{SupportedOS: []string{}},
			currentOS: "linux",
			wantErr:   true,
			errSubstr: "does not support linux",
		},

		// Edge case: unknown OS string in supported_os (should not match)
		{
			name:      "supported_os=[freebsd] on linux",
			pkgName:   "bsd-only",
			pkgDef:    &PackageDef{SupportedOS: []string{"freebsd"}},
			currentOS: "linux",
			wantErr:   true,
			errSubstr: "does not support linux",
		},
		{
			name:      "supported_os=[freebsd] on freebsd",
			pkgName:   "bsd-only",
			pkgDef:    &PackageDef{SupportedOS: []string{"freebsd"}},
			currentOS: "freebsd",
			wantErr:   false,
		},

		// Error message includes package name
		{
			name:      "error message includes package name",
			pkgName:   "my-package",
			pkgDef:    &PackageDef{SupportedOS: []string{"darwin"}},
			currentOS: "linux",
			wantErr:   true,
			errSubstr: "my-package",
		},

		// Error message includes supported OS list
		{
			name:      "error message includes supported OS list",
			pkgName:   "pkg",
			pkgDef:    &PackageDef{SupportedOS: []string{"linux", "darwin"}},
			currentOS: "windows",
			wantErr:   true,
			errSubstr: "linux, darwin",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckOS(tc.pkgName, tc.pkgDef, tc.currentOS)
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
