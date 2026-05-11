package doctor

import (
	"fmt"
	"os"
)

// CheckRepoInitialized returns nil if repoPath exists, or a descriptive error
// telling the user to run `rice init`.
func CheckRepoInitialized(repoPath string) error {
	if _, err := os.Stat(repoPath); err != nil {
		return fmt.Errorf("rice repo not found at %q — run `rice init <url>` to initialize", repoPath)
	}
	return nil
}
