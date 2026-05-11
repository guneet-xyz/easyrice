package deps

// registry holds the hardcoded, well-known dependencies that easyrice can probe
// and install without the user declaring a custom_dependency entry in rice.toml.
//
// Each entry MUST:
//   - Use explicit argv in Probe.Command and InstallMethod.Command (no sh -c).
//   - Order native package managers before brew on linux.
//   - Set RequiresRoot=true for system package managers (apt/dnf/pacman/apk).
//   - Set RequiresRoot=false for brew (Homebrew) and pip3 --user-style installs.
//   - Never include "sudo" in argv: the install runner adds privilege handling.
var registry = map[string]ResolvedDependency{
	"neovim": {
		Name: "neovim",
		Probe: ProbeSpec{
			Command:      []string{"nvim", "--version"},
			VersionRegex: `NVIM v(\d+\.\d+\.\d+)`,
		},
		Methods: []InstallMethod{
			{
				ID:             "apt",
				Description:    "Install neovim via apt-get (Debian/Ubuntu)",
				OS:             "linux",
				DistroFamilies: []string{"debian"},
				Command:        []string{"apt-get", "install", "-y", "neovim"},
				RequiresRoot:   true,
			},
			{
				ID:             "pacman",
				Description:    "Install neovim via pacman (Arch)",
				OS:             "linux",
				DistroFamilies: []string{"arch"},
				Command:        []string{"pacman", "-S", "--noconfirm", "neovim"},
				RequiresRoot:   true,
			},
			{
				ID:             "dnf",
				Description:    "Install neovim via dnf (Fedora/RHEL)",
				OS:             "linux",
				DistroFamilies: []string{"fedora"},
				Command:        []string{"dnf", "install", "-y", "neovim"},
				RequiresRoot:   true,
			},
			{
				ID:             "apk",
				Description:    "Install neovim via apk (Alpine)",
				OS:             "linux",
				DistroFamilies: []string{"alpine"},
				Command:        []string{"apk", "add", "neovim"},
				RequiresRoot:   true,
			},
			{
				ID:           "brew",
				Description:  "Install neovim via Homebrew (macOS)",
				OS:           "darwin",
				Command:      []string{"brew", "install", "neovim"},
				RequiresRoot: false,
			},
		},
	},
	"ripgrep": {
		Name: "ripgrep",
		Probe: ProbeSpec{
			Command:      []string{"rg", "--version"},
			VersionRegex: `ripgrep (\d+\.\d+\.\d+)`,
		},
		Methods: []InstallMethod{
			{
				ID:             "apt",
				Description:    "Install ripgrep via apt-get (Debian/Ubuntu)",
				OS:             "linux",
				DistroFamilies: []string{"debian"},
				Command:        []string{"apt-get", "install", "-y", "ripgrep"},
				RequiresRoot:   true,
			},
			{
				ID:             "pacman",
				Description:    "Install ripgrep via pacman (Arch)",
				OS:             "linux",
				DistroFamilies: []string{"arch"},
				Command:        []string{"pacman", "-S", "--noconfirm", "ripgrep"},
				RequiresRoot:   true,
			},
			{
				ID:             "dnf",
				Description:    "Install ripgrep via dnf (Fedora/RHEL)",
				OS:             "linux",
				DistroFamilies: []string{"fedora"},
				Command:        []string{"dnf", "install", "-y", "ripgrep"},
				RequiresRoot:   true,
			},
			{
				ID:             "apk",
				Description:    "Install ripgrep via apk (Alpine)",
				OS:             "linux",
				DistroFamilies: []string{"alpine"},
				Command:        []string{"apk", "add", "ripgrep"},
				RequiresRoot:   true,
			},
			{
				ID:           "brew",
				Description:  "Install ripgrep via Homebrew (macOS)",
				OS:           "darwin",
				Command:      []string{"brew", "install", "ripgrep"},
				RequiresRoot: false,
			},
		},
	},
	"node": {
		Name: "node",
		Probe: ProbeSpec{
			Command:      []string{"node", "--version"},
			VersionRegex: `v(\d+\.\d+\.\d+)`,
		},
		Methods: []InstallMethod{
			{
				ID:             "apt",
				Description:    "Install Node.js via apt-get (Debian/Ubuntu)",
				OS:             "linux",
				DistroFamilies: []string{"debian"},
				Command:        []string{"apt-get", "install", "-y", "nodejs"},
				RequiresRoot:   true,
			},
			{
				ID:             "pacman",
				Description:    "Install Node.js via pacman (Arch)",
				OS:             "linux",
				DistroFamilies: []string{"arch"},
				Command:        []string{"pacman", "-S", "--noconfirm", "nodejs"},
				RequiresRoot:   true,
			},
			{
				ID:             "dnf",
				Description:    "Install Node.js via dnf (Fedora/RHEL)",
				OS:             "linux",
				DistroFamilies: []string{"fedora"},
				Command:        []string{"dnf", "install", "-y", "nodejs"},
				RequiresRoot:   true,
			},
			{
				ID:             "apk",
				Description:    "Install Node.js via apk (Alpine)",
				OS:             "linux",
				DistroFamilies: []string{"alpine"},
				Command:        []string{"apk", "add", "nodejs"},
				RequiresRoot:   true,
			},
			{
				ID:           "brew",
				Description:  "Install Node.js via Homebrew (macOS)",
				OS:           "darwin",
				Command:      []string{"brew", "install", "node"},
				RequiresRoot: false,
			},
		},
	},
	// nvm has no install methods: its official installer uses curl|bash which
	// violates the argv-only policy. Users must install nvm manually.
	"nvm": {
		Name: "nvm",
		Probe: ProbeSpec{
			Command:      []string{"bash", "-c", "command -v nvm"},
			VersionRegex: ``,
		},
		Methods: nil,
	},
	"mdformat": {
		Name: "mdformat",
		Probe: ProbeSpec{
			Command:      []string{"mdformat", "--version"},
			VersionRegex: `(\d+\.\d+\.\d+)`,
		},
		Methods: []InstallMethod{
			{
				ID:           "pip3-linux",
				Description:  "Install mdformat via pip3 (user site)",
				OS:           "linux",
				Command:      []string{"pip3", "install", "mdformat"},
				RequiresRoot: false,
			},
			{
				ID:           "pip3-darwin",
				Description:  "Install mdformat via pip3 (user site)",
				OS:           "darwin",
				Command:      []string{"pip3", "install", "mdformat"},
				RequiresRoot: false,
			},
			{
				ID:           "brew",
				Description:  "Install mdformat via Homebrew (macOS)",
				OS:           "darwin",
				Command:      []string{"brew", "install", "mdformat"},
				RequiresRoot: false,
			},
		},
	},
	"zsh": {
		Name: "zsh",
		Probe: ProbeSpec{
			Command:      []string{"zsh", "--version"},
			VersionRegex: `zsh (\d+\.\d+)`,
		},
		Methods: []InstallMethod{
			{
				ID:             "apt",
				Description:    "Install zsh via apt-get (Debian/Ubuntu)",
				OS:             "linux",
				DistroFamilies: []string{"debian"},
				Command:        []string{"apt-get", "install", "-y", "zsh"},
				RequiresRoot:   true,
			},
			{
				ID:             "pacman",
				Description:    "Install zsh via pacman (Arch)",
				OS:             "linux",
				DistroFamilies: []string{"arch"},
				Command:        []string{"pacman", "-S", "--noconfirm", "zsh"},
				RequiresRoot:   true,
			},
			{
				ID:             "dnf",
				Description:    "Install zsh via dnf (Fedora/RHEL)",
				OS:             "linux",
				DistroFamilies: []string{"fedora"},
				Command:        []string{"dnf", "install", "-y", "zsh"},
				RequiresRoot:   true,
			},
			{
				ID:             "apk",
				Description:    "Install zsh via apk (Alpine)",
				OS:             "linux",
				DistroFamilies: []string{"alpine"},
				Command:        []string{"apk", "add", "zsh"},
				RequiresRoot:   true,
			},
			{
				ID:           "brew",
				Description:  "Install zsh via Homebrew (macOS)",
				OS:           "darwin",
				Command:      []string{"brew", "install", "zsh"},
				RequiresRoot: false,
			},
		},
	},
	"tmux": {
		Name: "tmux",
		Probe: ProbeSpec{
			Command:      []string{"tmux", "-V"},
			VersionRegex: `tmux (\d+\.\d+)`,
		},
		Methods: []InstallMethod{
			{
				ID:             "apt",
				Description:    "Install tmux via apt-get (Debian/Ubuntu)",
				OS:             "linux",
				DistroFamilies: []string{"debian"},
				Command:        []string{"apt-get", "install", "-y", "tmux"},
				RequiresRoot:   true,
			},
			{
				ID:             "pacman",
				Description:    "Install tmux via pacman (Arch)",
				OS:             "linux",
				DistroFamilies: []string{"arch"},
				Command:        []string{"pacman", "-S", "--noconfirm", "tmux"},
				RequiresRoot:   true,
			},
			{
				ID:             "dnf",
				Description:    "Install tmux via dnf (Fedora/RHEL)",
				OS:             "linux",
				DistroFamilies: []string{"fedora"},
				Command:        []string{"dnf", "install", "-y", "tmux"},
				RequiresRoot:   true,
			},
			{
				ID:             "apk",
				Description:    "Install tmux via apk (Alpine)",
				OS:             "linux",
				DistroFamilies: []string{"alpine"},
				Command:        []string{"apk", "add", "tmux"},
				RequiresRoot:   true,
			},
			{
				ID:           "brew",
				Description:  "Install tmux via Homebrew (macOS)",
				OS:           "darwin",
				Command:      []string{"brew", "install", "tmux"},
				RequiresRoot: false,
			},
		},
	},
	"fzf": {
		Name: "fzf",
		Probe: ProbeSpec{
			Command:      []string{"fzf", "--version"},
			VersionRegex: `(\d+\.\d+\.\d+)`,
		},
		Methods: []InstallMethod{
			{
				ID:             "apt",
				Description:    "Install fzf via apt-get (Debian/Ubuntu)",
				OS:             "linux",
				DistroFamilies: []string{"debian"},
				Command:        []string{"apt-get", "install", "-y", "fzf"},
				RequiresRoot:   true,
			},
			{
				ID:             "pacman",
				Description:    "Install fzf via pacman (Arch)",
				OS:             "linux",
				DistroFamilies: []string{"arch"},
				Command:        []string{"pacman", "-S", "--noconfirm", "fzf"},
				RequiresRoot:   true,
			},
			{
				ID:             "dnf",
				Description:    "Install fzf via dnf (Fedora/RHEL)",
				OS:             "linux",
				DistroFamilies: []string{"fedora"},
				Command:        []string{"dnf", "install", "-y", "fzf"},
				RequiresRoot:   true,
			},
			{
				ID:             "apk",
				Description:    "Install fzf via apk (Alpine)",
				OS:             "linux",
				DistroFamilies: []string{"alpine"},
				Command:        []string{"apk", "add", "fzf"},
				RequiresRoot:   true,
			},
			{
				ID:           "brew",
				Description:  "Install fzf via Homebrew (macOS)",
				OS:           "darwin",
				Command:      []string{"brew", "install", "fzf"},
				RequiresRoot: false,
			},
		},
	},
	"fd": {
		Name: "fd",
		Probe: ProbeSpec{
			Command:      []string{"fd", "--version"},
			VersionRegex: `fd (\d+\.\d+\.\d+)`,
		},
		Methods: []InstallMethod{
			{
				ID:             "apt",
				Description:    "Install fd via apt-get (Debian/Ubuntu, package: fd-find)",
				OS:             "linux",
				DistroFamilies: []string{"debian"},
				Command:        []string{"apt-get", "install", "-y", "fd-find"},
				RequiresRoot:   true,
			},
			{
				ID:             "pacman",
				Description:    "Install fd via pacman (Arch)",
				OS:             "linux",
				DistroFamilies: []string{"arch"},
				Command:        []string{"pacman", "-S", "--noconfirm", "fd"},
				RequiresRoot:   true,
			},
			{
				ID:             "dnf",
				Description:    "Install fd via dnf (Fedora/RHEL, package: fd-find)",
				OS:             "linux",
				DistroFamilies: []string{"fedora"},
				Command:        []string{"dnf", "install", "-y", "fd-find"},
				RequiresRoot:   true,
			},
			{
				ID:             "apk",
				Description:    "Install fd via apk (Alpine)",
				OS:             "linux",
				DistroFamilies: []string{"alpine"},
				Command:        []string{"apk", "add", "fd"},
				RequiresRoot:   true,
			},
			{
				ID:           "brew",
				Description:  "Install fd via Homebrew (macOS)",
				OS:           "darwin",
				Command:      []string{"brew", "install", "fd"},
				RequiresRoot: false,
			},
		},
	},
	"bat": {
		Name: "bat",
		Probe: ProbeSpec{
			Command:      []string{"bat", "--version"},
			VersionRegex: `bat (\d+\.\d+\.\d+)`,
		},
		Methods: []InstallMethod{
			{
				ID:             "apt",
				Description:    "Install bat via apt-get (Debian/Ubuntu)",
				OS:             "linux",
				DistroFamilies: []string{"debian"},
				Command:        []string{"apt-get", "install", "-y", "bat"},
				RequiresRoot:   true,
			},
			{
				ID:             "pacman",
				Description:    "Install bat via pacman (Arch)",
				OS:             "linux",
				DistroFamilies: []string{"arch"},
				Command:        []string{"pacman", "-S", "--noconfirm", "bat"},
				RequiresRoot:   true,
			},
			{
				ID:             "dnf",
				Description:    "Install bat via dnf (Fedora/RHEL)",
				OS:             "linux",
				DistroFamilies: []string{"fedora"},
				Command:        []string{"dnf", "install", "-y", "bat"},
				RequiresRoot:   true,
			},
			{
				ID:             "apk",
				Description:    "Install bat via apk (Alpine)",
				OS:             "linux",
				DistroFamilies: []string{"alpine"},
				Command:        []string{"apk", "add", "bat"},
				RequiresRoot:   true,
			},
			{
				ID:           "brew",
				Description:  "Install bat via Homebrew (macOS)",
				OS:           "darwin",
				Command:      []string{"brew", "install", "bat"},
				RequiresRoot: false,
			},
		},
	},
	"eza": {
		Name: "eza",
		Probe: ProbeSpec{
			Command:      []string{"eza", "--version"},
			VersionRegex: `v(\d+\.\d+\.\d+)`,
		},
		Methods: []InstallMethod{
			{
				ID:             "apt",
				Description:    "Install eza via apt-get (Debian/Ubuntu)",
				OS:             "linux",
				DistroFamilies: []string{"debian"},
				Command:        []string{"apt-get", "install", "-y", "eza"},
				RequiresRoot:   true,
			},
			{
				ID:             "pacman",
				Description:    "Install eza via pacman (Arch)",
				OS:             "linux",
				DistroFamilies: []string{"arch"},
				Command:        []string{"pacman", "-S", "--noconfirm", "eza"},
				RequiresRoot:   true,
			},
			{
				ID:             "dnf",
				Description:    "Install eza via dnf (Fedora/RHEL)",
				OS:             "linux",
				DistroFamilies: []string{"fedora"},
				Command:        []string{"dnf", "install", "-y", "eza"},
				RequiresRoot:   true,
			},
			{
				ID:             "apk",
				Description:    "Install eza via apk (Alpine)",
				OS:             "linux",
				DistroFamilies: []string{"alpine"},
				Command:        []string{"apk", "add", "eza"},
				RequiresRoot:   true,
			},
			{
				ID:           "brew",
				Description:  "Install eza via Homebrew (macOS)",
				OS:           "darwin",
				Command:      []string{"brew", "install", "eza"},
				RequiresRoot: false,
			},
		},
	},
	"git": {
		Name: "git",
		Probe: ProbeSpec{
			Command:      []string{"git", "--version"},
			VersionRegex: `git version (\d+\.\d+\.\d+)`,
		},
		Methods: []InstallMethod{
			{
				ID:             "apt",
				Description:    "Install git via apt-get (Debian/Ubuntu)",
				OS:             "linux",
				DistroFamilies: []string{"debian"},
				Command:        []string{"apt-get", "install", "-y", "git"},
				RequiresRoot:   true,
			},
			{
				ID:             "pacman",
				Description:    "Install git via pacman (Arch)",
				OS:             "linux",
				DistroFamilies: []string{"arch"},
				Command:        []string{"pacman", "-S", "--noconfirm", "git"},
				RequiresRoot:   true,
			},
			{
				ID:             "dnf",
				Description:    "Install git via dnf (Fedora/RHEL)",
				OS:             "linux",
				DistroFamilies: []string{"fedora"},
				Command:        []string{"dnf", "install", "-y", "git"},
				RequiresRoot:   true,
			},
			{
				ID:             "apk",
				Description:    "Install git via apk (Alpine)",
				OS:             "linux",
				DistroFamilies: []string{"alpine"},
				Command:        []string{"apk", "add", "git"},
				RequiresRoot:   true,
			},
			{
				ID:           "brew",
				Description:  "Install git via Homebrew (macOS)",
				OS:           "darwin",
				Command:      []string{"brew", "install", "git"},
				RequiresRoot: false,
			},
		},
	},
}

// RegistryDep returns the hardcoded ResolvedDependency for the given name and
// true if it exists in the registry, or the zero value and false otherwise.
func RegistryDep(name string) (ResolvedDependency, bool) {
	dep, ok := registry[name]
	return dep, ok
}
