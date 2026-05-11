package deps

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProbe(t *testing.T) {
	tests := []struct {
		name          string
		dep           ResolvedDependency
		runner        *MockRunner
		wantInstalled bool
		wantVersion   string
		wantErr       bool
		errContains   string
	}{
		{
			name: "exit 0 with regex match",
			dep: ResolvedDependency{
				Name: "git",
				Probe: ProbeSpec{
					Command:      []string{"git", "--version"},
					VersionRegex: `version (\d+\.\d+\.\d+)`,
				},
			},
			runner: &MockRunner{
				Expectations: []MockExpectation{
					{
						Argv: []string{"git", "--version"},
						Result: RunResult{
							ExitCode: 0,
							Combined: []byte("git version 2.40.1"),
						},
					},
				},
			},
			wantInstalled: true,
			wantVersion:   "2.40.1",
			wantErr:       false,
		},
		{
			name: "exit 0 without version regex",
			dep: ResolvedDependency{
				Name: "curl",
				Probe: ProbeSpec{
					Command:      []string{"curl", "--version"},
					VersionRegex: "",
				},
			},
			runner: &MockRunner{
				Expectations: []MockExpectation{
					{
						Argv: []string{"curl", "--version"},
						Result: RunResult{
							ExitCode: 0,
							Combined: []byte("curl 7.85.0"),
						},
					},
				},
			},
			wantInstalled: true,
			wantVersion:   "",
			wantErr:       false,
		},
		{
			name: "exit 0 with regex but no match",
			dep: ResolvedDependency{
				Name: "node",
				Probe: ProbeSpec{
					Command:      []string{"node", "--version"},
					VersionRegex: `release (\d+\.\d+\.\d+)`,
				},
			},
			runner: &MockRunner{
				Expectations: []MockExpectation{
					{
						Argv: []string{"node", "--version"},
						Result: RunResult{
							ExitCode: 0,
							Combined: []byte("v18.0.0-rc.1"),
						},
					},
				},
			},
			wantInstalled: true,
			wantVersion:   "",
			wantErr:       false,
		},
		{
			name: "exit non-zero (not installed)",
			dep: ResolvedDependency{
				Name: "missing-tool",
				Probe: ProbeSpec{
					Command:      []string{"missing-tool", "--version"},
					VersionRegex: `(\d+\.\d+)`,
				},
			},
			runner: &MockRunner{
				Expectations: []MockExpectation{
					{
						Argv: []string{"missing-tool", "--version"},
						Result: RunResult{
							ExitCode: 127,
							Combined: []byte("command not found"),
						},
					},
				},
			},
			wantInstalled: false,
			wantVersion:   "",
			wantErr:       false,
		},
		{
			name: "empty probe command",
			dep: ResolvedDependency{
				Name: "bad-dep",
				Probe: ProbeSpec{
					Command:      []string{},
					VersionRegex: `(\d+)`,
				},
			},
			runner:        &MockRunner{},
			wantInstalled: false,
			wantVersion:   "",
			wantErr:       true,
			errContains:   "no probe command",
		},
		{
			name: "context cancelled",
			dep: ResolvedDependency{
				Name: "timeout-dep",
				Probe: ProbeSpec{
					Command:      []string{"sleep", "10"},
					VersionRegex: `(\d+)`,
				},
			},
			runner: &MockRunner{
				Expectations: []MockExpectation{
					{
						Argv: []string{"sleep", "10"},
						Err:  context.Canceled,
					},
				},
			},
			wantInstalled: false,
			wantVersion:   "",
			wantErr:       true,
			errContains:   "context canceled",
		},
		{
			name: "invalid version regex",
			dep: ResolvedDependency{
				Name: "bad-regex",
				Probe: ProbeSpec{
					Command:      []string{"echo", "version 1.0"},
					VersionRegex: `[invalid(regex`,
				},
			},
			runner: &MockRunner{
				Expectations: []MockExpectation{
					{
						Argv: []string{"echo", "version 1.0"},
						Result: RunResult{
							ExitCode: 0,
							Combined: []byte("version 1.0"),
						},
					},
				},
			},
			wantInstalled: true,
			wantVersion:   "",
			wantErr:       true,
			errContains:   "invalid version regex",
		},
		{
			name: "regex with multiple capture groups (uses first)",
			dep: ResolvedDependency{
				Name: "python",
				Probe: ProbeSpec{
					Command:      []string{"python3", "--version"},
					VersionRegex: `Python (\d+\.\d+\.\d+)`,
				},
			},
			runner: &MockRunner{
				Expectations: []MockExpectation{
					{
						Argv: []string{"python3", "--version"},
						Result: RunResult{
							ExitCode: 0,
							Combined: []byte("Python 3.11.2"),
						},
					},
				},
			},
			wantInstalled: true,
			wantVersion:   "3.11.2",
			wantErr:       false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			installed, version, err := Probe(ctx, tc.runner, tc.dep)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.wantInstalled, installed)
			assert.Equal(t, tc.wantVersion, version)
		})
	}
}
