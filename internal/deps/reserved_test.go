package deps

import (
	"slices"
	"testing"
)

func TestIsReserved(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected bool
	}{
		{"neovim reserved", "neovim", true},
		{"zsh reserved", "zsh", true},
		{"tmux reserved", "tmux", true},
		{"ripgrep reserved", "ripgrep", true},
		{"fzf reserved", "fzf", true},
		{"git reserved", "git", true},
		{"node reserved", "node", true},
		{"fd reserved", "fd", true},
		{"bat reserved", "bat", true},
		{"eza reserved", "eza", true},
		{"mypkg not reserved", "mypkg", false},
		{"custom not reserved", "custom", false},
		{"empty string not reserved", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsReserved(tc.input)
			if got != tc.expected {
				t.Errorf("IsReserved(%q) = %v, want %v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestReservedNames(t *testing.T) {
	got := ReservedNames()

	expected := []string{"bat", "eza", "fd", "fzf", "git", "node", "ripgrep", "tmux", "zsh", "neovim"}

	if len(got) != len(expected) {
		t.Errorf("ReservedNames() returned %d names, want %d", len(got), len(expected))
	}

	for _, name := range expected {
		if !slices.Contains(got, name) {
			t.Errorf("ReservedNames() missing %q", name)
		}
	}

	// Verify it's sorted
	for i := 0; i < len(got)-1; i++ {
		if got[i] > got[i+1] {
			t.Errorf("ReservedNames() not sorted: %v", got)
			break
		}
	}
}
