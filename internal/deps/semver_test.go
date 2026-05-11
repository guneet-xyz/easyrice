package deps

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchVersion(t *testing.T) {
	cases := []struct {
		name       string
		installed  string
		constraint string
		want       bool
		wantErr    bool
	}{
		{
			name:       "empty constraint accepts any version",
			installed:  "1.2.3",
			constraint: "",
			want:       true,
			wantErr:    false,
		},
		{
			name:       "version matches >=0.10",
			installed:  "0.10.4",
			constraint: ">=0.10",
			want:       true,
			wantErr:    false,
		},
		{
			name:       "version does not match >=0.10",
			installed:  "0.9.5",
			constraint: ">=0.10",
			want:       false,
			wantErr:    false,
		},
		{
			name:       "exact version match",
			installed:  "1.2.3",
			constraint: "1.2.3",
			want:       true,
			wantErr:    false,
		},
		{
			name:       "caret constraint matches",
			installed:  "1.3.0",
			constraint: "^1.2",
			want:       true,
			wantErr:    false,
		},
		{
			name:       "tilde constraint does not match",
			installed:  "2.0.0",
			constraint: "~2.3",
			want:       false,
			wantErr:    false,
		},
		{
			name:       "invalid version string",
			installed:  "garbage",
			constraint: ">=0.10",
			want:       false,
			wantErr:    true,
		},
		{
			name:       "version with v prefix",
			installed:  "v1.2.3",
			constraint: "1.2.3",
			want:       true,
			wantErr:    false,
		},
		{
			name:       "invalid constraint string",
			installed:  "1.2.3",
			constraint: "not-a-constraint",
			want:       false,
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := MatchVersion(tc.installed, tc.constraint)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsValidConstraint(t *testing.T) {
	cases := []struct {
		name      string
		constraint string
		wantErr   bool
	}{
		{
			name:       "empty constraint is valid",
			constraint: "",
			wantErr:    false,
		},
		{
			name:       ">=0.10 is valid",
			constraint: ">=0.10",
			wantErr:    false,
		},
		{
			name:       "^1.2 is valid",
			constraint: "^1.2",
			wantErr:    false,
		},
		{
			name:       "~2.3 is valid",
			constraint: "~2.3",
			wantErr:    false,
		},
		{
			name:       "1.2.3 is valid",
			constraint: "1.2.3",
			wantErr:    false,
		},
		{
			name:       "not-a-constraint is invalid",
			constraint: "not-a-constraint",
			wantErr:    true,
		},
		{
			name:       "invalid operator",
			constraint: ">>1.0",
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := IsValidConstraint(tc.constraint)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
