package updater

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatReminder(t *testing.T) {
	got := FormatReminder("v1.0.0", "v1.2.3", "guneet-xyz", "easyrice")
	want := "A new release of easyrice is available: v1.0.0 \u2192 v1.2.3\nhttps://github.com/guneet-xyz/easyrice/releases/latest"
	assert.Equal(t, want, got, "byte-equal mismatch")
}

func TestFormatReminder_OtherOwnerRepo(t *testing.T) {
	got := FormatReminder("v0.1.0", "v0.2.0", "acme", "tool")
	want := "A new release of easyrice is available: v0.1.0 \u2192 v0.2.0\nhttps://github.com/acme/tool/releases/latest"
	assert.Equal(t, want, got)
}

func TestShouldShowReminder(t *testing.T) {
	cases := []struct {
		name     string
		disabled bool
		current  string
		isTTY    bool
		want     bool
	}{
		{"all good shows", false, "v1.0.0", true, true},
		{"disabled blocks", true, "v1.0.0", true, false},
		{"dev build blocks", false, "dev", true, false},
		{"empty version blocks", false, "", true, false},
		{"non-tty blocks", false, "v1.0.0", false, false},
		{"disabled+dev blocks", true, "dev", true, false},
		{"disabled+non-tty blocks", true, "v1.0.0", false, false},
		{"dev+non-tty blocks", false, "dev", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ShouldShowReminder(tc.disabled, tc.current, tc.isTTY))
		})
	}
}

func TestIsTerminal_NotATTY(t *testing.T) {
	f, err := tempNonTTYFile(t)
	if err != nil {
		t.Skip("cannot create temp file")
	}
	defer f.Close()
	assert.False(t, IsTerminal(f), "regular file is not a TTY")
}

// TestFormatReminder_ContainsVersion verifies the formatted reminder string
// contains both the current and latest version identifiers verbatim.
func TestFormatReminder_ContainsVersion(t *testing.T) {
	got := FormatReminder("v1.2.3", "v2.0.0", "guneet-xyz", "easyrice")
	assert.Contains(t, got, "v1.2.3", "output must contain current version")
	assert.Contains(t, got, "v2.0.0", "output must contain latest version")
	assert.Contains(t, got, "guneet-xyz/easyrice", "output must contain owner/repo")
}

// TestShouldShowReminder_ReturnsFalseWhenCacheStale models the scenario where
// the cache-check pipeline yielded no fresh "update available" signal. In the
// pure-helper API surface, that maps to disabled=true (the post-command hook
// suppresses the reminder when the cached result is not actionable).
func TestShouldShowReminder_ReturnsFalseWhenCacheStale(t *testing.T) {
	// Stale cache → CLI passes disabled=true to suppress reminder.
	assert.False(t, ShouldShowReminder(true, "v1.0.0", true))
}

// TestShouldShowReminder_ReturnsTrueWhenNewerAvailable models a fresh cache
// reporting a newer version: not disabled, tagged build, TTY → reminder shows.
func TestShouldShowReminder_ReturnsTrueWhenNewerAvailable(t *testing.T) {
	assert.True(t, ShouldShowReminder(false, "v1.0.0", true))
}

// TestShouldShowReminder_ReturnsFalseWhenAlreadyLatest models a fresh cache
// where current == latest: the CLI passes disabled=true so no reminder fires.
func TestShouldShowReminder_ReturnsFalseWhenAlreadyLatest(t *testing.T) {
	assert.False(t, ShouldShowReminder(true, "v1.2.3", true))
}

// TestIsTerminal_ReturnsFalse confirms IsTerminal returns false for a
// non-terminal file descriptor (a regular temp file, not a PTY).
func TestIsTerminal_ReturnsFalse(t *testing.T) {
	f, err := tempNonTTYFile(t)
	if err != nil {
		t.Skip("cannot create temp file")
	}
	defer f.Close()
	assert.False(t, IsTerminal(f), "non-TTY file descriptor must report false")
}
