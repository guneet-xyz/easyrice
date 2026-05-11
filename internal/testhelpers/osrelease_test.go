package testhelpers

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeOSRelease_KnownDistros(t *testing.T) {
	for _, distro := range []string{"ubuntu", "debian", "fedora", "arch", "alpine", "rhel", "centos"} {
		t.Run(distro, func(t *testing.T) {
			r := FakeOSRelease(distro)
			data, err := io.ReadAll(r)
			require.NoError(t, err)
			assert.Contains(t, string(data), "ID=")
		})
	}
}

func TestFakeOSRelease_UnknownDistro(t *testing.T) {
	r := FakeOSRelease("totally-not-real")
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Contains(t, string(data), "ID=unknown")
}

func TestFakeOSReleaseMalformed(t *testing.T) {
	r := FakeOSReleaseMalformed()
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(data), "not valid"))
}
