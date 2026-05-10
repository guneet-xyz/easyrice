package doctor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/guneet-xyz/easyrice/internal/state"
)

// CheckLegacyState detects the presence of a legacy rice state file at
// ~/.config/rice/state.json and prints a warning to w if it exists while
// the new easyrice state file does not.
//
// It does not migrate the file; it only informs the user.
func CheckLegacyState(w io.Writer) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	legacyPath := filepath.Join(home, ".config", "rice", "state.json")
	newPath := state.DefaultPath()

	if !fileExists(legacyPath) {
		return
	}
	if fileExists(newPath) {
		return
	}

	fmt.Fprintf(w, "Warning: Legacy state found at %s.\n", legacyPath)
	fmt.Fprintf(w, "Run: mv %s %s\n", legacyPath, newPath)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
