package deps

import "sort"

// reservedNames is the set of package names that easyrice officially recognizes.
// A rice package whose name matches a reserved name gets an implicit self-dependency
// and MUST NOT explicitly declare that same dependency.
var reservedNames = map[string]struct{}{
	"neovim":  {},
	"zsh":     {},
	"tmux":    {},
	"ripgrep": {},
	"fzf":     {},
	"git":     {},
	"node":    {},
	"fd":      {},
	"bat":     {},
	"eza":     {},
}

// IsReserved reports whether name is a reserved package name.
func IsReserved(name string) bool {
	_, ok := reservedNames[name]
	return ok
}

// ReservedNames returns a sorted slice of all reserved names (for display/docs).
func ReservedNames() []string {
	names := make([]string, 0, len(reservedNames))
	for name := range reservedNames {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
