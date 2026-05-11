package deps

import (
	"regexp"
	"strings"
	"testing"
)

func TestRegistryAllEntriesExist(t *testing.T) {
	t.Parallel()
	want := []string{
		"neovim", "ripgrep", "node", "nvm", "mdformat",
		"zsh", "tmux", "fzf", "fd", "bat", "eza", "git",
	}
	for _, name := range want {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dep, ok := RegistryDep(name)
			if !ok {
				t.Fatalf("RegistryDep(%q): not found", name)
			}
			if dep.Name != name {
				t.Errorf("RegistryDep(%q).Name = %q, want %q", name, dep.Name, name)
			}
		})
	}
}

func TestRegistryUnknownReturnsFalse(t *testing.T) {
	t.Parallel()
	if _, ok := RegistryDep("definitely-not-a-real-tool"); ok {
		t.Fatal("RegistryDep(unknown): expected ok=false")
	}
}

func TestRegistryProbeIsValid(t *testing.T) {
	t.Parallel()
	for name, dep := range registry {
		name, dep := name, dep
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if len(dep.Probe.Command) == 0 {
				t.Errorf("%s: Probe.Command is empty", name)
			}
			if dep.Probe.VersionRegex != "" {
				if _, err := regexp.Compile(dep.Probe.VersionRegex); err != nil {
					t.Errorf("%s: Probe.VersionRegex %q does not compile: %v",
						name, dep.Probe.VersionRegex, err)
				}
			}
		})
	}
}

func TestRegistryNvmHasNoInstallMethods(t *testing.T) {
	t.Parallel()
	dep, ok := RegistryDep("nvm")
	if !ok {
		t.Fatal("nvm: not in registry")
	}
	if len(dep.Methods) != 0 {
		t.Errorf("nvm: expected 0 install methods (probe-only), got %d", len(dep.Methods))
	}
}

func TestRegistryNonNvmHasMethodsPerSupportedOS(t *testing.T) {
	t.Parallel()
	for name, dep := range registry {
		if name == "nvm" {
			continue
		}
		name, dep := name, dep
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if len(dep.Methods) == 0 {
				t.Fatalf("%s: expected at least one install method", name)
			}
			byOS := map[string]int{}
			for _, m := range dep.Methods {
				byOS[m.OS]++
			}
			if byOS["linux"] == 0 {
				t.Errorf("%s: no linux install method", name)
			}
			if byOS["darwin"] == 0 {
				t.Errorf("%s: no darwin install method", name)
			}
		})
	}
}

func TestRegistryNoSudoInArgv(t *testing.T) {
	t.Parallel()
	for name, dep := range registry {
		for i, m := range dep.Methods {
			for _, arg := range m.Command {
				if strings.EqualFold(arg, "sudo") {
					t.Errorf("%s: method[%d] (%s) Command contains 'sudo': %v",
						name, i, m.ID, m.Command)
				}
			}
		}
	}
}

func TestRegistryMethodsHaveCommandAndID(t *testing.T) {
	t.Parallel()
	for name, dep := range registry {
		for i, m := range dep.Methods {
			if m.ID == "" {
				t.Errorf("%s: method[%d] has empty ID", name, i)
			}
			if len(m.Command) == 0 {
				t.Errorf("%s: method[%d] (%s) has empty Command", name, i, m.ID)
			}
			if m.OS == "" {
				t.Errorf("%s: method[%d] (%s) has empty OS", name, i, m.ID)
			}
		}
	}
}
