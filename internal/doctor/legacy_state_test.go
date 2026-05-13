package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckLegacyState(t *testing.T) {
	cases := []struct {
		name         string
		createLegacy bool
		createNew    bool
		wantContains []string
		wantEmpty    bool
	}{
		{
			name:         "legacy exists, new missing -> warns",
			createLegacy: true,
			createNew:    false,
			wantContains: []string{"Legacy rice state found", "mv "},
		},
		{
			name:         "legacy exists, new exists -> silent",
			createLegacy: true,
			createNew:    true,
			wantEmpty:    true,
		},
		{
			name:         "neither exists -> silent",
			createLegacy: false,
			createNew:    false,
			wantEmpty:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("HOME", tmp)
			t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
			t.Setenv("APPDATA", filepath.Join(tmp, "AppData"))

			legacyPath := filepath.Join(tmp, ".config", "rice", "state.json")
			if tc.createLegacy {
				if err := os.MkdirAll(filepath.Dir(legacyPath), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(legacyPath, []byte("{}"), 0644); err != nil {
					t.Fatal(err)
				}
			}

			if tc.createNew {
				// Mirror state.DefaultPath() which uses os.UserConfigDir().
				// On linux: $XDG_CONFIG_HOME/easyrice/state.json
				// On darwin: $HOME/Library/Application Support/easyrice/state.json
				// We create both to be safe across platforms.
				newPaths := []string{
					filepath.Join(tmp, ".config", "easyrice", "state.json"),
					filepath.Join(tmp, "Library", "Application Support", "easyrice", "state.json"),
					filepath.Join(tmp, "AppData", "easyrice", "state.json"),
				}
				for _, p := range newPaths {
					if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(p, []byte("{}"), 0644); err != nil {
						t.Fatal(err)
					}
				}
			}

			var buf bytes.Buffer
			CheckLegacyState(&buf)
			got := buf.String()

			if tc.wantEmpty {
				if got != "" {
					t.Fatalf("expected no output, got: %q", got)
				}
				return
			}

			for _, sub := range tc.wantContains {
				if !strings.Contains(got, sub) {
					t.Errorf("output missing %q; got: %q", sub, got)
				}
			}
			if !strings.Contains(got, legacyPath) {
				t.Errorf("output should contain legacy path %q; got: %q", legacyPath, got)
			}
		})
	}
}
