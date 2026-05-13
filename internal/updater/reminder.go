package updater

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// FormatReminder returns a formatted reminder message for a new release.
// It returns a 2-line string with no leading or trailing newlines:
//
//	Update available: <current> → <latest>
//	Run `rice upgrade` or visit https://github.com/<owner>/<repo>/releases/latest
func FormatReminder(current, latest, owner, repo string) string {
	return fmt.Sprintf("Update available: %s → %s\nRun `rice upgrade` or visit https://github.com/%s/%s/releases/latest", current, latest, owner, repo)
}

// ShouldShowReminder determines whether the update reminder should be displayed.
// It returns false if:
//   - disabled is true (user opted out)
//   - current is a dev build (checked via IsDevBuild)
//   - isStderrTTY is false (not a terminal)
//
// Otherwise, it returns true.
func ShouldShowReminder(disabled bool, current string, isStderrTTY bool) bool {
	if disabled {
		return false
	}
	if IsDevBuild(current) {
		return false
	}
	if !isStderrTTY {
		return false
	}
	return true
}

// IsTerminal returns true if f is a terminal (TTY).
// It uses golang.org/x/term.IsTerminal to check if the file descriptor
// is connected to a terminal.
func IsTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}
