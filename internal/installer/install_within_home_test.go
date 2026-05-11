package installer

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet-xyz/easyrice/internal/manifest"
)

func buildWithinHomeRequest(t *testing.T, target string) (InstallRequest, string) {
	t.Helper()

	repoRoot := t.TempDir()
	homeDir := t.TempDir()

	pkgRoot := filepath.Join(repoRoot, "pkg", "src")
	require.NoError(t, os.MkdirAll(pkgRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkgRoot, "file.txt"), []byte("x"), 0o644))

	pkg := &manifest.PackageDef{
		Description: "withinHome probe",
		SupportedOS: []string{runtime.GOOS},
		Root:        "pkg",
		Profiles: map[string]manifest.ProfileDef{
			"default": {
				Sources: []manifest.SourceSpec{
					{Path: "src", Mode: "file", Target: target},
				},
			},
		},
	}

	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	req := InstallRequest{
		RepoRoot:    repoRoot,
		PackageName: "pkg",
		ProfileName: "default",
		Pkg:         pkg,
		Specs:       pkg.Profiles["default"].Sources,
		CurrentOS:   runtime.GOOS,
		HomeDir:     homeDir,
		StatePath:   filepath.Join(t.TempDir(), "state.json"),
	}
	return req, homeDir
}

func TestBuildInstallPlan_WithinHome(t *testing.T) {
	cases := []struct {
		name      string
		target    string
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "target equal to HOME is allowed",
			target:  "$HOME",
			wantErr: false,
		},
		{
			name:    "target nested deep under HOME is allowed",
			target:  "$HOME/.config/app",
			wantErr: false,
		},
		{
			name:    "target with dotdot resolving inside HOME is allowed",
			target:  "$HOME/foo/../bar",
			wantErr: false,
		},
		{
			name:      "target with dotdot escaping HOME is rejected",
			target:    "$HOME/../etc/passwd",
			wantErr:   true,
			errSubstr: "escapes home",
		},
		{
			name:      "absolute target outside HOME is rejected",
			target:    "/etc/passwd",
			wantErr:   true,
			errSubstr: "escapes home",
		},
		{
			name:      "absolute tmp target is rejected",
			target:    "/tmp/something",
			wantErr:   true,
			errSubstr: "escapes home",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := buildWithinHomeRequest(t, tc.target)
			req.Specs = []manifest.SourceSpec{
				{Path: "src", Mode: "file", Target: tc.target},
			}
			req.Pkg.Profiles["default"] = manifest.ProfileDef{Sources: req.Specs}

			_, err := BuildInstallPlan(req)
			if tc.wantErr {
				require.Error(t, err)
				assert.True(t,
					strings.Contains(err.Error(), tc.errSubstr) ||
						strings.Contains(err.Error(), "outside"),
					"error %q should mention boundary violation (substring %q)", err.Error(), tc.errSubstr,
				)
				return
			}
			require.NoError(t, err)
		})
	}
}
