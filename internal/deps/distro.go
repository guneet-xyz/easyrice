package deps

import (
	"bufio"
	"io"
	"os"
	"runtime"
	"strings"
)

// Detect returns the current platform by reading /etc/os-release on Linux.
func Detect() Platform {
	switch runtime.GOOS {
	case "darwin":
		return Platform{OS: "darwin"}
	case "windows":
		return Platform{OS: "windows"}
	case "linux":
		f, err := os.Open("/etc/os-release")
		if err != nil {
			return Platform{OS: "linux", DistroFamily: "unknown"}
		}
		defer f.Close()
		return DetectFrom(f)
	default:
		return Platform{OS: runtime.GOOS}
	}
}

// DetectFrom parses an os-release formatted reader and returns the Platform.
// This is the testable seam — Detect() calls this with the real /etc/os-release.
func DetectFrom(r io.Reader) Platform {
	distroFamilyMap := map[string]string{
		"debian":      "debian",
		"ubuntu":      "debian",
		"linuxmint":   "debian",
		"pop":         "debian",
		"elementary":  "debian",
		"kali":        "debian",
		"raspbian":    "debian",
		"arch":        "arch",
		"manjaro":     "arch",
		"endeavouros": "arch",
		"garuda":      "arch",
		"artix":       "arch",
		"fedora":      "fedora",
		"rhel":        "fedora",
		"centos":      "fedora",
		"rocky":       "fedora",
		"almalinux":   "fedora",
		"ol":          "fedora",
		"alpine":      "alpine",
	}

	var id, idLike string

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Strip surrounding quotes
		value = strings.Trim(value, `"'`)

		switch key {
		case "ID":
			id = value
		case "ID_LIKE":
			idLike = value
		}
	}

	// Try ID first
	if id != "" {
		if family, ok := distroFamilyMap[id]; ok {
			return Platform{OS: "linux", DistroFamily: family}
		}
	}

	// Try ID_LIKE (space-separated list)
	if idLike != "" {
		for _, candidate := range strings.Fields(idLike) {
			if family, ok := distroFamilyMap[candidate]; ok {
				return Platform{OS: "linux", DistroFamily: family}
			}
		}
	}

	// Default to unknown
	return Platform{OS: "linux", DistroFamily: "unknown"}
}
