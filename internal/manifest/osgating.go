package manifest

import (
	"fmt"
	"strings"
)

// CheckOS returns nil if currentOS is in m.SupportedOS, or a descriptive error.
// currentOS should be runtime.GOOS (e.g., "linux", "darwin", "windows").
func CheckOS(m *Manifest, currentOS string) error {
	// Defensive: empty SupportedOS shouldn't happen after Validate, but check anyway
	if len(m.SupportedOS) == 0 {
		return fmt.Errorf("package %q does not support %s; supported: (none)", m.Name, currentOS)
	}

	// Check if currentOS is in the supported list
	for _, os := range m.SupportedOS {
		if os == currentOS {
			return nil
		}
	}

	// OS not supported; return error with supported list
	supported := strings.Join(m.SupportedOS, ", ")
	return fmt.Errorf("package %q does not support %s; supported: %s", m.Name, currentOS, supported)
}
