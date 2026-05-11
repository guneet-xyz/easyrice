package deps

import (
	"testing"

	"github.com/guneet-xyz/easyrice/internal/testhelpers"
	"github.com/stretchr/testify/assert"
)

func TestDetectFrom_UsingFakeOSReleaseHelpers(t *testing.T) {
	cases := []struct {
		distro string
		want   Platform
	}{
		{"ubuntu", Platform{OS: "linux", DistroFamily: "debian"}},
		{"debian", Platform{OS: "linux", DistroFamily: "debian"}},
		{"fedora", Platform{OS: "linux", DistroFamily: "fedora"}},
		{"arch", Platform{OS: "linux", DistroFamily: "arch"}},
		{"alpine", Platform{OS: "linux", DistroFamily: "alpine"}},
		{"rhel", Platform{OS: "linux", DistroFamily: "fedora"}},
		{"centos", Platform{OS: "linux", DistroFamily: "fedora"}},
		{"unknown-distro", Platform{OS: "linux", DistroFamily: "unknown"}},
	}
	for _, tc := range cases {
		t.Run(tc.distro, func(t *testing.T) {
			got := DetectFrom(testhelpers.FakeOSRelease(tc.distro))
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDetectFrom_MalformedFakeOSRelease(t *testing.T) {
	got := DetectFrom(testhelpers.FakeOSReleaseMalformed())
	assert.Equal(t, Platform{OS: "linux", DistroFamily: "unknown"}, got)
}
