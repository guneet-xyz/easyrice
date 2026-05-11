package doctor

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/guneet-xyz/easyrice/internal/deps"
	"github.com/guneet-xyz/easyrice/internal/manifest"
)

func TestCheckDeclaredDeps(t *testing.T) {
	cases := []struct {
		name            string
		manifest        manifest.Manifest
		runner          deps.Runner
		wantWarnings    int
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "no packages -> no output",
			manifest: manifest.Manifest{
				Packages: map[string]manifest.PackageDef{},
			},
			runner:       &deps.MockRunner{},
			wantWarnings: 0,
		},
		{
			name: "package with no dependencies -> skipped silently",
			manifest: manifest.Manifest{
				Packages: map[string]manifest.PackageDef{
					"pkg1": {
						Dependencies: []deps.DependencyRef{},
					},
				},
			},
			runner:       &deps.MockRunner{},
			wantWarnings: 0,
		},
		{
			name: "package with all-OK deps -> no output, 0 warnings",
			manifest: manifest.Manifest{
				Packages: map[string]manifest.PackageDef{
					"pkg1": {
						Dependencies: []deps.DependencyRef{
							{Name: "neovim", Version: ">=0.1"},
						},
					},
				},
			},
			runner: &deps.MockRunner{
				Expectations: []deps.MockExpectation{
					{
						Argv: []string{"nvim", "--version"},
						Result: deps.RunResult{
							ExitCode: 0,
							Combined: []byte("NVIM v0.9.0\n"),
						},
					},
				},
			},
			wantWarnings: 0,
		},
		{
			name: "package with missing dep -> [WARN] line, warnings=1",
			manifest: manifest.Manifest{
				Packages: map[string]manifest.PackageDef{
					"pkg1": {
						Dependencies: []deps.DependencyRef{
							{Name: "custom_missing", Version: ">=1.0"},
						},
					},
				},
				CustomDependencies: map[string]deps.CustomDependencyDef{
					"custom_missing": {
						VersionProbe: []string{"which", "custom_missing"},
					},
				},
			},
			runner: &deps.MockRunner{
				Expectations: []deps.MockExpectation{
					{
						Argv: []string{"which", "custom_missing"},
						Result: deps.RunResult{
							ExitCode: 1,
						},
					},
				},
			},
			wantWarnings: 1,
			wantContains: []string{"[WARN] pkg1.custom_missing — missing"},
		},
		{
			name: "package with version mismatch -> [WARN] line, warnings=1",
			manifest: manifest.Manifest{
				Packages: map[string]manifest.PackageDef{
					"pkg1": {
						Dependencies: []deps.DependencyRef{
							{Name: "custom_mismatch", Version: ">=2.0"},
						},
					},
				},
				CustomDependencies: map[string]deps.CustomDependencyDef{
					"custom_mismatch": {
						VersionProbe: []string{"echo", "1.0.0"},
						VersionRegex: `(\d+\.\d+\.\d+)`,
					},
				},
			},
			runner: &deps.MockRunner{
				Expectations: []deps.MockExpectation{
					{
						Argv: []string{"echo", "1.0.0"},
						Result: deps.RunResult{
							ExitCode: 0,
							Combined: []byte("1.0.0\n"),
						},
					},
				},
			},
			wantWarnings: 1,
			wantContains: []string{"[WARN] pkg1.custom_mismatch — version mismatch"},
		},
		{
			name: "package with unknown version -> [INFO] line, warnings=0",
			manifest: manifest.Manifest{
				Packages: map[string]manifest.PackageDef{
					"pkg1": {
						Dependencies: []deps.DependencyRef{
							{Name: "custom_unknown", Version: ">=1.0"},
						},
					},
				},
				CustomDependencies: map[string]deps.CustomDependencyDef{
					"custom_unknown": {
						VersionProbe: []string{"echo", "installed"},
					},
				},
			},
			runner: &deps.MockRunner{
				Expectations: []deps.MockExpectation{
					{
						Argv: []string{"echo", "installed"},
						Result: deps.RunResult{
							ExitCode: 0,
							Combined: []byte("installed\n"),
						},
					},
				},
			},
			wantWarnings: 0,
			wantContains: []string{"[INFO] pkg1.custom_unknown — installed (version unknown)"},
		},
		{
			name: "multiple packages sorted by name -> output in alphabetical order",
			manifest: manifest.Manifest{
				Packages: map[string]manifest.PackageDef{
					"zebra": {
						Dependencies: []deps.DependencyRef{
							{Name: "custom_z", Version: ">=1.0"},
						},
					},
					"apple": {
						Dependencies: []deps.DependencyRef{
							{Name: "custom_a", Version: ">=1.0"},
						},
					},
				},
				CustomDependencies: map[string]deps.CustomDependencyDef{
					"custom_a": {
						VersionProbe: []string{"which", "custom_a"},
					},
					"custom_z": {
						VersionProbe: []string{"which", "custom_z"},
					},
				},
			},
			runner: &deps.MockRunner{
				Expectations: []deps.MockExpectation{
					{
						Argv: []string{"which", "custom_a"},
						Result: deps.RunResult{
							ExitCode: 1,
						},
					},
					{
						Argv: []string{"which", "custom_z"},
						Result: deps.RunResult{
							ExitCode: 1,
						},
					},
				},
			},
			wantWarnings: 2,
			wantContains: []string{"[WARN] apple.custom_a", "[WARN] zebra.custom_z"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := context.Background()
			got := CheckDeclaredDeps(ctx, &buf, tc.runner, tc.manifest)

			if got != tc.wantWarnings {
				t.Errorf("warnings: want %d, got %d", tc.wantWarnings, got)
			}

			output := buf.String()
			for _, sub := range tc.wantContains {
				if !strings.Contains(output, sub) {
					t.Errorf("output missing %q; got:\n%s", sub, output)
				}
			}

			for _, sub := range tc.wantNotContains {
				if strings.Contains(output, sub) {
					t.Errorf("output should not contain %q; got:\n%s", sub, output)
				}
			}
		})
	}
}
