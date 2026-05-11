package testhelpers

import (
	"io"
	"strings"
)

// FakeOSRelease returns an io.Reader with valid /etc/os-release content for the given distro.
// Supported distros: "ubuntu", "debian", "fedora", "arch", "alpine", "rhel", "centos".
func FakeOSRelease(distro string) io.Reader {
	content := ""
	switch distro {
	case "ubuntu":
		content = `ID=ubuntu
VERSION_ID="22.04"
NAME="Ubuntu"
ID_LIKE=debian
`
	case "debian":
		content = `ID=debian
VERSION_ID="11"
NAME="Debian GNU/Linux"
`
	case "fedora":
		content = `ID=fedora
VERSION_ID="38"
NAME="Fedora Linux"
`
	case "arch":
		content = `ID=arch
NAME="Arch Linux"
ID_LIKE=archlinux
`
	case "alpine":
		content = `ID=alpine
VERSION_ID="3.18"
NAME="Alpine Linux"
`
	case "rhel":
		content = `ID=rhel
VERSION_ID="8"
NAME="Red Hat Enterprise Linux"
ID_LIKE=fedora
`
	case "centos":
		content = `ID=centos
VERSION_ID="7"
NAME="CentOS Linux"
ID_LIKE=rhel
`
	default:
		// Unknown distro — return minimal valid content
		content = `ID=unknown
NAME="Unknown Linux"
`
	}
	return strings.NewReader(content)
}

// FakeOSReleaseMalformed returns an io.Reader with invalid/unparseable /etc/os-release content.
func FakeOSReleaseMalformed() io.Reader {
	content := `this is not valid os-release format
no equals signs here
just random text
`
	return strings.NewReader(content)
}
