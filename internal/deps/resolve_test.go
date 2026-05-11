package deps

import (
	"testing"
)

func TestResolve(t *testing.T) {
	cases := []struct {
		name      string
		refs      []DependencyRef
		custom    map[string]CustomDependencyDef
		wantErr   bool
		wantCount int
		check     func(t *testing.T, result []ResolvedDependency)
	}{
		{
			name: "registry hit with version override",
			refs: []DependencyRef{
				{Name: "neovim", Version: "0.9.0"},
			},
			custom:    map[string]CustomDependencyDef{},
			wantErr:   false,
			wantCount: 1,
			check: func(t *testing.T, result []ResolvedDependency) {
				if result[0].Name != "neovim" {
					t.Errorf("Name = %q, want neovim", result[0].Name)
				}
				if result[0].Version != "0.9.0" {
					t.Errorf("Version = %q, want 0.9.0", result[0].Version)
				}
				if len(result[0].Methods) == 0 {
					t.Error("Methods should not be empty for registry entry")
				}
			},
		},
		{
			name: "custom dependency used when not in registry",
			refs: []DependencyRef{
				{Name: "custom-tool", Version: "1.0.0"},
			},
			custom: map[string]CustomDependencyDef{
				"custom-tool": {
					VersionProbe: []string{"custom-tool", "--version"},
					VersionRegex: `v(\d+\.\d+\.\d+)`,
					Install: map[string]CustomInstallMethod{
						"linux_debian": {
							Description:    "Install via apt",
							ShellPayload:   "apt-get install custom-tool",
							DistroFamilies: []string{"debian"},
						},
						"darwin": {
							Description:  "Install via brew",
							ShellPayload: "brew install custom-tool",
						},
					},
				},
			},
			wantErr:   false,
			wantCount: 1,
			check: func(t *testing.T, result []ResolvedDependency) {
				if result[0].Name != "custom-tool" {
					t.Errorf("Name = %q, want custom-tool", result[0].Name)
				}
				if result[0].Version != "1.0.0" {
					t.Errorf("Version = %q, want 1.0.0", result[0].Version)
				}
				if len(result[0].Methods) != 2 {
					t.Errorf("Methods count = %d, want 2", len(result[0].Methods))
				}
				// Check OS parsing
				hasLinux := false
				hasDarwin := false
				for _, m := range result[0].Methods {
					if m.OS == "linux" {
						hasLinux = true
					}
					if m.OS == "darwin" {
						hasDarwin = true
					}
				}
				if !hasLinux || !hasDarwin {
					t.Error("Expected both linux and darwin methods")
				}
			},
		},
		{
			name: "unknown dependency returns error",
			refs: []DependencyRef{
				{Name: "unknown-tool", Version: "1.0.0"},
			},
			custom:  map[string]CustomDependencyDef{},
			wantErr: true,
		},
		{
			name: "preserves input order",
			refs: []DependencyRef{
				{Name: "ripgrep", Version: "13.0.0"},
				{Name: "neovim", Version: "0.9.0"},
				{Name: "ripgrep", Version: "14.0.0"},
			},
			custom:    map[string]CustomDependencyDef{},
			wantErr:   false,
			wantCount: 3,
			check: func(t *testing.T, result []ResolvedDependency) {
				if result[0].Name != "ripgrep" || result[0].Version != "13.0.0" {
					t.Errorf("First: Name=%q Version=%q, want ripgrep 13.0.0", result[0].Name, result[0].Version)
				}
				if result[1].Name != "neovim" || result[1].Version != "0.9.0" {
					t.Errorf("Second: Name=%q Version=%q, want neovim 0.9.0", result[1].Name, result[1].Version)
				}
				if result[2].Name != "ripgrep" || result[2].Version != "14.0.0" {
					t.Errorf("Third: Name=%q Version=%q, want ripgrep 14.0.0", result[2].Name, result[2].Version)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Resolve(tc.refs, tc.custom)
			if (err != nil) != tc.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr {
				if len(result) != tc.wantCount {
					t.Errorf("Resolve() count = %d, want %d", len(result), tc.wantCount)
				}
				if tc.check != nil {
					tc.check(t, result)
				}
			}
		})
	}
}

func TestFilterByPlatform(t *testing.T) {
	cases := []struct {
		name      string
		deps      []ResolvedDependency
		platform  Platform
		wantErr   bool
		wantCount int
		check     func(t *testing.T, result []ResolvedDependency)
	}{
		{
			name: "filters methods not matching OS",
			deps: []ResolvedDependency{
				{
					Name: "test-tool",
					Methods: []InstallMethod{
						{ID: "apt", OS: "linux", DistroFamilies: []string{"debian"}},
						{ID: "brew", OS: "darwin"},
						{ID: "choco", OS: "windows"},
					},
				},
			},
			platform:  Platform{OS: "linux", DistroFamily: "debian"},
			wantErr:   false,
			wantCount: 1,
			check: func(t *testing.T, result []ResolvedDependency) {
				if len(result[0].Methods) != 1 {
					t.Errorf("Methods count = %d, want 1", len(result[0].Methods))
				}
				if result[0].Methods[0].ID != "apt" {
					t.Errorf("Method ID = %q, want apt", result[0].Methods[0].ID)
				}
			},
		},
		{
			name: "filters linux methods not matching distro family",
			deps: []ResolvedDependency{
				{
					Name: "test-tool",
					Methods: []InstallMethod{
						{ID: "apt", OS: "linux", DistroFamilies: []string{"debian"}},
						{ID: "pacman", OS: "linux", DistroFamilies: []string{"arch"}},
						{ID: "dnf", OS: "linux", DistroFamilies: []string{"fedora"}},
					},
				},
			},
			platform:  Platform{OS: "linux", DistroFamily: "arch"},
			wantErr:   false,
			wantCount: 1,
			check: func(t *testing.T, result []ResolvedDependency) {
				if len(result[0].Methods) != 1 {
					t.Errorf("Methods count = %d, want 1", len(result[0].Methods))
				}
				if result[0].Methods[0].ID != "pacman" {
					t.Errorf("Method ID = %q, want pacman", result[0].Methods[0].ID)
				}
			},
		},
		{
			name: "error when all methods filtered out",
			deps: []ResolvedDependency{
				{
					Name: "test-tool",
					Methods: []InstallMethod{
						{ID: "apt", OS: "linux", DistroFamilies: []string{"debian"}},
					},
				},
			},
			platform: Platform{OS: "darwin"},
			wantErr:  true,
		},
		{
			name: "no error for dep with zero methods initially",
			deps: []ResolvedDependency{
				{
					Name:    "nvm",
					Methods: []InstallMethod{},
				},
			},
			platform:  Platform{OS: "linux", DistroFamily: "debian"},
			wantErr:   false,
			wantCount: 1,
			check: func(t *testing.T, result []ResolvedDependency) {
				if len(result[0].Methods) != 0 {
					t.Errorf("Methods count = %d, want 0", len(result[0].Methods))
				}
			},
		},
		{
			name: "preserves multiple deps",
			deps: []ResolvedDependency{
				{
					Name: "tool1",
					Methods: []InstallMethod{
						{ID: "apt", OS: "linux", DistroFamilies: []string{"debian"}},
					},
				},
				{
					Name: "tool2",
					Methods: []InstallMethod{
						{ID: "apt2", OS: "linux", DistroFamilies: []string{"debian"}},
					},
				},
			},
			platform:  Platform{OS: "linux", DistroFamily: "debian"},
			wantErr:   false,
			wantCount: 2,
			check: func(t *testing.T, result []ResolvedDependency) {
				if result[0].Name != "tool1" {
					t.Errorf("First dep name = %q, want tool1", result[0].Name)
				}
				if result[1].Name != "tool2" {
					t.Errorf("Second dep name = %q, want tool2", result[1].Name)
				}
				if len(result[0].Methods) != 1 {
					t.Errorf("First dep methods = %d, want 1", len(result[0].Methods))
				}
				if len(result[1].Methods) != 1 {
					t.Errorf("Second dep methods = %d, want 1", len(result[1].Methods))
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := FilterByPlatform(tc.deps, tc.platform)
			if (err != nil) != tc.wantErr {
				t.Errorf("FilterByPlatform() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr {
				if len(result) != tc.wantCount {
					t.Errorf("FilterByPlatform() count = %d, want %d", len(result), tc.wantCount)
				}
				if tc.check != nil {
					tc.check(t, result)
				}
			}
		})
	}
}
