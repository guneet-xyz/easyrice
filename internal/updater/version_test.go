package updater

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsDevBuild(t *testing.T) {
	cases := []struct {
		name string
		v    string
		want bool
	}{
		{"empty", "", true},
		{"dev literal", "dev", true},
		{"garbage", "not-a-version", true},
		{"valid semver no v", "1.2.3", false},
		{"valid semver with v", "v1.2.3", false},
		{"valid prerelease", "v1.2.3-beta.1", false},
		{"valid build metadata", "v1.2.3+build.5", false},
		{"only v", "v", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsDevBuild(tc.v))
		})
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		name    string
		current string
		latest  string
		want    bool
		wantErr bool
	}{
		{"latest is newer", "v1.0.0", "v1.0.1", true, false},
		{"latest is older", "v1.0.1", "v1.0.0", false, false},
		{"equal versions", "v1.2.3", "v1.2.3", false, false},
		{"major bump", "v1.99.99", "v2.0.0", true, false},
		{"unprefixed both", "1.0.0", "1.0.1", true, false},
		{"mixed prefix", "1.0.0", "v1.0.1", true, false},
		{"prerelease vs release", "v1.0.0-beta", "v1.0.0", true, false},
		{"invalid current", "garbage", "v1.0.0", false, true},
		{"invalid latest", "v1.0.0", "garbage", false, true},
		{"both invalid", "abc", "def", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := IsNewer(tc.current, tc.latest)
			if tc.wantErr {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, ErrInvalidSemver), "expected ErrInvalidSemver")
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsPreRelease(t *testing.T) {
	cases := []struct {
		name string
		v    string
		want bool
	}{
		{"plain release", "v1.0.0", false},
		{"plain release no prefix", "1.0.0", false},
		{"beta prerelease", "v1.0.0-beta.1", true},
		{"rc prerelease", "v2.5.0-rc.3", true},
		{"alpha prerelease no prefix", "1.0.0-alpha", true},
		{"build metadata only", "v1.0.0+build.5", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsPreRelease(tc.v))
		})
	}
}
