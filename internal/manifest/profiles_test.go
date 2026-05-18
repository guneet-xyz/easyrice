package manifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortedProfileNames(t *testing.T) {
	cases := []struct {
		name     string
		pkg      PackageDef
		expected []string
	}{
		{
			name: "multiple_profiles",
			pkg: PackageDef{
				Profiles: map[string]ProfileDef{
					"work":    {},
					"common":  {},
					"macbook": {},
				},
			},
			expected: []string{"common", "macbook", "work"},
		},
		{
			name: "single_profile",
			pkg: PackageDef{
				Profiles: map[string]ProfileDef{
					"default": {},
				},
			},
			expected: []string{"default"},
		},
		{
			name: "comma_in_name",
			pkg: PackageDef{
				Profiles: map[string]ProfileDef{
					"foo,bar": {},
					"alpha":   {},
				},
			},
			expected: []string{"alpha", "foo,bar"},
		},
		{
			name: "asterisk_in_name",
			pkg: PackageDef{
				Profiles: map[string]ProfileDef{
					"*starred": {},
					"plain":    {},
				},
			},
			expected: []string{"*starred", "plain"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := SortedProfileNames(tc.pkg)
			assert.Equal(t, tc.expected, result)
		})
	}
}
