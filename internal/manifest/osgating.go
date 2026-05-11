package manifest

import (
	"fmt"
	"strings"
)

// CheckOS returns nil if currentOS is in pkgDef.SupportedOS, or a descriptive error.
// currentOS should be runtime.GOOS (e.g., "linux", "darwin", "windows").
// pkgName is used for error messages.
func CheckOS(pkgName string, pkgDef *PackageDef, currentOS string) error {
	// Defensive: empty SupportedOS shouldn't happen after Validate, but check anyway
	if len(pkgDef.SupportedOS) == 0 {
		return fmt.Errorf("package %q does not support %s; supported: (none)", pkgName, currentOS)
	}

	// Check if currentOS is in the supported list
	for _, os := range pkgDef.SupportedOS {
		if os == currentOS {
			return nil
		}
	}

	// OS not supported; return error with supported list
	supported := strings.Join(pkgDef.SupportedOS, ", ")
	return fmt.Errorf("package %q does not support %s; supported: %s", pkgName, currentOS, supported)
}
