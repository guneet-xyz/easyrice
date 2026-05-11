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
