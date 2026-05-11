package doctor

import (
	"fmt"
	"os/exec"
)

// CheckGitOnPath returns nil if the git binary is available, or a descriptive error.
func CheckGitOnPath() error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git binary not found on PATH — install git to use easyrice")
	}
	return nil
}
