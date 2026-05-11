package deps

import (
	"runtime"
	"strings"
	"testing"

	"github.com/guneet-xyz/easyrice/internal/testhelpers"
)

func TestDetectFrom(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected Platform
	}{
		{
			name:     "ID=ubuntu",
			input:    "ID=ubuntu",
			expected: Platform{OS: "linux", DistroFamily: "debian"},
		},
		{
			name:     "ID=debian",
			input:    "ID=debian",
			expected: Platform{OS: "linux", DistroFamily: "debian"},
		},
		{
			name:     "ID=linuxmint with ID_LIKE fallback",
			input:    "ID=linuxmint\nID_LIKE=\"ubuntu debian\"",
			expected: Platform{OS: "linux", DistroFamily: "debian"},
		},
		{
			name:     "ID=arch",
			input:    "ID=arch",
			expected: Platform{OS: "linux", DistroFamily: "arch"},
		},
		{
			name:     "ID=manjaro with ID_LIKE",
			input:    "ID=manjaro\nID_LIKE=arch",
			expected: Platform{OS: "linux", DistroFamily: "arch"},
		},
		{
			name:     "ID=fedora",
			input:    "ID=fedora",
			expected: Platform{OS: "linux", DistroFamily: "fedora"},
		},
		{
			name:     "ID=rhel with ID_LIKE",
			input:    "ID=rhel\nID_LIKE=fedora",
			expected: Platform{OS: "linux", DistroFamily: "fedora"},
		},
		{
			name:     "ID=alpine",
			input:    "ID=alpine",
			expected: Platform{OS: "linux", DistroFamily: "alpine"},
		},
		{
			name:     "empty reader",
			input:    "",
			expected: Platform{OS: "linux", DistroFamily: "unknown"},
		},
		{
			name:     "quoted values",
			input:    "ID=\"ubuntu\"",
			expected: Platform{OS: "linux", DistroFamily: "debian"},
		},
		{
			name:     "comments skipped",
			input:    "# comment\nID=arch",
			expected: Platform{OS: "linux", DistroFamily: "arch"},
		},
		{
			name:     "ID=pop (debian family)",
			input:    "ID=pop",
			expected: Platform{OS: "linux", DistroFamily: "debian"},
		},
		{
			name:     "ID=elementary (debian family)",
			input:    "ID=elementary",
			expected: Platform{OS: "linux", DistroFamily: "debian"},
		},
		{
			name:     "ID=kali (debian family)",
			input:    "ID=kali",
			expected: Platform{OS: "linux", DistroFamily: "debian"},
		},
		{
			name:     "ID=raspbian (debian family)",
			input:    "ID=raspbian",
			expected: Platform{OS: "linux", DistroFamily: "debian"},
		},
		{
			name:     "ID=endeavouros (arch family)",
			input:    "ID=endeavouros",
			expected: Platform{OS: "linux", DistroFamily: "arch"},
		},
		{
			name:     "ID=garuda (arch family)",
			input:    "ID=garuda",
			expected: Platform{OS: "linux", DistroFamily: "arch"},
		},
		{
			name:     "ID=artix (arch family)",
			input:    "ID=artix",
			expected: Platform{OS: "linux", DistroFamily: "arch"},
		},
		{
			name:     "ID=centos (fedora family)",
			input:    "ID=centos",
			expected: Platform{OS: "linux", DistroFamily: "fedora"},
		},
		{
			name:     "ID=rocky (fedora family)",
			input:    "ID=rocky",
			expected: Platform{OS: "linux", DistroFamily: "fedora"},
		},
		{
			name:     "ID=almalinux (fedora family)",
			input:    "ID=almalinux",
			expected: Platform{OS: "linux", DistroFamily: "fedora"},
		},
		{
			name:     "ID=ol (fedora family)",
			input:    "ID=ol",
			expected: Platform{OS: "linux", DistroFamily: "fedora"},
		},
		{
			name:     "ID_LIKE with multiple candidates",
			input:    "ID=unknown\nID_LIKE=\"ubuntu debian arch\"",
			expected: Platform{OS: "linux", DistroFamily: "debian"},
		},
		{
			name:     "single-quoted values",
			input:    "ID='ubuntu'",
			expected: Platform{OS: "linux", DistroFamily: "debian"},
		},
		{
			name:     "whitespace handling",
			input:    "  ID  =  ubuntu  ",
			expected: Platform{OS: "linux", DistroFamily: "debian"},
		},
		{
			name:     "unknown distro",
			input:    "ID=unknowndistro",
			expected: Platform{OS: "linux", DistroFamily: "unknown"},
		},
		{
			name:     "ID_LIKE fallback to unknown",
			input:    "ID=custom\nID_LIKE=\"custom other\"",
			expected: Platform{OS: "linux", DistroFamily: "unknown"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reader := strings.NewReader(tc.input)
			result := DetectFrom(reader)
			if result.OS != tc.expected.OS || result.DistroFamily != tc.expected.DistroFamily {
				t.Errorf("DetectFrom(%q) = %+v, want %+v", tc.input, result, tc.expected)
			}
		})
	}
}

// TestDetectFrom_Ubuntu tests DetectFrom with testhelpers.FakeOSRelease("ubuntu")
func TestDetectFrom_Ubuntu(t *testing.T) {
	reader := testhelpers.FakeOSRelease("ubuntu")
	result := DetectFrom(reader)
	expected := Platform{OS: "linux", DistroFamily: "debian"}
	if result.OS != expected.OS || result.DistroFamily != expected.DistroFamily {
		t.Errorf("DetectFrom(FakeOSRelease(\"ubuntu\")) = %+v, want %+v", result, expected)
	}
}

// TestDetectFrom_Malformed tests DetectFrom with testhelpers.FakeOSReleaseMalformed
func TestDetectFrom_Malformed(t *testing.T) {
	reader := testhelpers.FakeOSReleaseMalformed()
	result := DetectFrom(reader)
	expected := Platform{OS: "linux", DistroFamily: "unknown"}
	if result.OS != expected.OS || result.DistroFamily != expected.DistroFamily {
		t.Errorf("DetectFrom(FakeOSReleaseMalformed()) = %+v, want %+v", result, expected)
	}
}

// TestDetect_RuntimeBranch tests Detect() runtime branches
func TestDetect_RuntimeBranch(t *testing.T) {
	cases := []struct {
		name     string
		goos     string
		expected Platform
	}{
		{
			name:     "darwin branch",
			goos:     "darwin",
			expected: Platform{OS: "darwin"},
		},
		{
			name:     "windows branch",
			goos:     "windows",
			expected: Platform{OS: "windows"},
		},
		{
			name:     "unknown OS branch",
			goos:     "freebsd",
			expected: Platform{OS: "freebsd"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Only test the branch that matches the current runtime.GOOS
			if runtime.GOOS != tc.goos {
				t.Skipf("skipping %s test on %s", tc.goos, runtime.GOOS)
			}

			result := Detect()
			if result.OS != tc.expected.OS {
				t.Errorf("Detect() = %+v, want %+v", result, tc.expected)
			}
		})
	}
}

// TestDetect_LinuxFileOpen tests Detect() on Linux
func TestDetect_LinuxFileOpen(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skipf("skipping Linux-specific test on %s", runtime.GOOS)
	}

	result := Detect()
	if result.OS != "linux" {
		t.Errorf("Detect() on Linux should return OS=linux, got %+v", result)
	}
	if result.DistroFamily == "" {
		t.Errorf("Detect() on Linux should set DistroFamily, got empty string")
	}
}

// TestDetect_LinuxFileOpenError verifies Detect() returns valid Platform on Linux
func TestDetect_LinuxFileOpenError(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skipf("skipping Linux-specific test on %s", runtime.GOOS)
	}

	_ = t.TempDir()

	result := Detect()
	if result.OS != "linux" {
		t.Errorf("Detect() on Linux should return OS=linux, got %+v", result)
	}
}
