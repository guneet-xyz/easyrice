package plan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		plan     *Plan
		expected bool
	}{
		{
			name:     "empty plan",
			plan:     &Plan{},
			expected: true,
		},
		{
			name: "plan with ops",
			plan: &Plan{
				Ops: []Op{
					{Kind: OpCreate, Source: "src", Target: "tgt"},
				},
			},
			expected: false,
		},
		{
			name: "plan with conflicts",
			plan: &Plan{
				Conflicts: []Conflict{
					{Target: "tgt", Reason: "exists"},
				},
			},
			expected: false,
		},
		{
			name: "plan with both ops and conflicts",
			plan: &Plan{
				Ops: []Op{
					{Kind: OpCreate, Source: "src", Target: "tgt"},
				},
				Conflicts: []Conflict{
					{Target: "tgt2", Reason: "exists"},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.plan.IsEmpty()
			assert.Equal(t, tt.expected, result)
		})
	}
}
