package deps

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheck_DepOK(t *testing.T) {
	t.Helper()

	refs := []DependencyRef{
		{Name: "neovim", Version: ">=0.9.0"},
	}
	custom := map[string]CustomDependencyDef{}
	platform := Platform{OS: "darwin"}

	runner := &MockRunner{
		Expectations: []MockExpectation{
			{
				Argv: []string{"nvim", "--version"},
				Result: RunResult{
					ExitCode: 0,
					Combined: []byte("NVIM v0.10.0"),
				},
			},
		},
	}

	report, err := Check(context.Background(), runner, refs, custom, platform)
	require.NoError(t, err)
	require.Len(t, report.Entries, 1)

	entry := report.Entries[0]
	assert.Equal(t, "neovim", entry.Dep.Name)
	assert.True(t, entry.Installed)
	assert.Equal(t, "0.10.0", entry.InstalledVersion)
	assert.Equal(t, DepOK, entry.Status)
	assert.False(t, report.NeedsAction())
}

func TestCheck_DepMissing(t *testing.T) {
	t.Helper()

	refs := []DependencyRef{
		{Name: "neovim"},
	}
	custom := map[string]CustomDependencyDef{}
	platform := Platform{OS: "darwin"}

	runner := &MockRunner{
		Expectations: []MockExpectation{
			{
				Argv: []string{"nvim", "--version"},
				Result: RunResult{
					ExitCode: 127,
				},
			},
		},
	}

	report, err := Check(context.Background(), runner, refs, custom, platform)
	require.NoError(t, err)
	require.Len(t, report.Entries, 1)

	entry := report.Entries[0]
	assert.Equal(t, "neovim", entry.Dep.Name)
	assert.False(t, entry.Installed)
	assert.Equal(t, "", entry.InstalledVersion)
	assert.Equal(t, DepMissing, entry.Status)
	assert.True(t, report.NeedsAction())
	assert.Len(t, report.Missing(), 1)
}

func TestCheck_DepVersionMismatch(t *testing.T) {
	t.Helper()

	refs := []DependencyRef{
		{Name: "neovim", Version: ">=1.0.0"},
	}
	custom := map[string]CustomDependencyDef{}
	platform := Platform{OS: "darwin"}

	runner := &MockRunner{
		Expectations: []MockExpectation{
			{
				Argv: []string{"nvim", "--version"},
				Result: RunResult{
					ExitCode: 0,
					Combined: []byte("NVIM v0.10.0"),
				},
			},
		},
	}

	report, err := Check(context.Background(), runner, refs, custom, platform)
	require.NoError(t, err)
	require.Len(t, report.Entries, 1)

	entry := report.Entries[0]
	assert.Equal(t, "neovim", entry.Dep.Name)
	assert.True(t, entry.Installed)
	assert.Equal(t, "0.10.0", entry.InstalledVersion)
	assert.Equal(t, DepVersionMismatch, entry.Status)
	assert.True(t, report.NeedsAction())
	assert.Len(t, report.Mismatched(), 1)
}

func TestCheck_DepProbeUnknownVersion(t *testing.T) {
	t.Helper()

	refs := []DependencyRef{
		{Name: "neovim"},
	}
	custom := map[string]CustomDependencyDef{}
	platform := Platform{OS: "darwin"}

	runner := &MockRunner{
		Expectations: []MockExpectation{
			{
				Argv: []string{"nvim", "--version"},
				Result: RunResult{
					ExitCode: 0,
					Combined: []byte("neovim unknown"),
				},
			},
		},
	}

	report, err := Check(context.Background(), runner, refs, custom, platform)
	require.NoError(t, err)
	require.Len(t, report.Entries, 1)

	entry := report.Entries[0]
	assert.Equal(t, "neovim", entry.Dep.Name)
	assert.True(t, entry.Installed)
	assert.Equal(t, "", entry.InstalledVersion)
	assert.Equal(t, DepProbeUnknownVersion, entry.Status)
	assert.False(t, report.NeedsAction())
}

func TestCheck_ProbeError(t *testing.T) {
	t.Helper()

	refs := []DependencyRef{
		{Name: "neovim"},
	}
	custom := map[string]CustomDependencyDef{}
	platform := Platform{OS: "darwin"}

	runner := &MockRunner{
		Expectations: []MockExpectation{
			{
				Argv: []string{"nvim", "--version"},
				Err:  errors.New("runner failed"),
			},
		},
	}

	_, err := Check(context.Background(), runner, refs, custom, platform)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check: probe")
}

func TestCheck_ResolveError(t *testing.T) {
	t.Helper()

	refs := []DependencyRef{
		{Name: "unknown-dep"},
	}
	custom := map[string]CustomDependencyDef{}
	platform := Platform{OS: "linux"}

	runner := &MockRunner{}

	_, err := Check(context.Background(), runner, refs, custom, platform)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check: resolve")
	assert.Contains(t, err.Error(), "unknown dependency")
}

func TestCheck_MultipleEntries(t *testing.T) {
	t.Helper()

	refs := []DependencyRef{
		{Name: "neovim", Version: ">=0.9.0"},
		{Name: "ripgrep"},
	}
	custom := map[string]CustomDependencyDef{}
	platform := Platform{OS: "darwin"}

	runner := &MockRunner{
		Expectations: []MockExpectation{
			{
				Argv: []string{"nvim", "--version"},
				Result: RunResult{
					ExitCode: 0,
					Combined: []byte("NVIM v0.10.0"),
				},
			},
			{
				Argv: []string{"rg", "--version"},
				Result: RunResult{
					ExitCode: 127,
				},
			},
		},
	}

	report, err := Check(context.Background(), runner, refs, custom, platform)
	require.NoError(t, err)
	require.Len(t, report.Entries, 2)

	assert.Equal(t, DepOK, report.Entries[0].Status)
	assert.Equal(t, DepMissing, report.Entries[1].Status)
	assert.True(t, report.NeedsAction())
	assert.Len(t, report.Missing(), 1)
}

func TestCheck_NoConstraint(t *testing.T) {
	t.Helper()

	refs := []DependencyRef{
		{Name: "neovim"},
	}
	custom := map[string]CustomDependencyDef{}
	platform := Platform{OS: "darwin"}

	runner := &MockRunner{
		Expectations: []MockExpectation{
			{
				Argv: []string{"nvim", "--version"},
				Result: RunResult{
					ExitCode: 0,
					Combined: []byte("NVIM v0.5.0"),
				},
			},
		},
	}

	report, err := Check(context.Background(), runner, refs, custom, platform)
	require.NoError(t, err)
	require.Len(t, report.Entries, 1)

	entry := report.Entries[0]
	assert.Equal(t, DepOK, entry.Status)
	assert.False(t, report.NeedsAction())
}

func TestDepReport_Missing(t *testing.T) {
	t.Helper()

	report := DepReport{
		Entries: []DepReportEntry{
			{Status: DepOK},
			{Status: DepMissing},
			{Status: DepVersionMismatch},
			{Status: DepMissing},
		},
	}

	missing := report.Missing()
	assert.Len(t, missing, 2)
	assert.Equal(t, DepMissing, missing[0].Status)
	assert.Equal(t, DepMissing, missing[1].Status)
}

func TestDepReport_Mismatched(t *testing.T) {
	t.Helper()

	report := DepReport{
		Entries: []DepReportEntry{
			{Status: DepOK},
			{Status: DepMissing},
			{Status: DepVersionMismatch},
			{Status: DepVersionMismatch},
		},
	}

	mismatched := report.Mismatched()
	assert.Len(t, mismatched, 2)
	assert.Equal(t, DepVersionMismatch, mismatched[0].Status)
	assert.Equal(t, DepVersionMismatch, mismatched[1].Status)
}

func TestDepReport_NeedsAction(t *testing.T) {
	t.Helper()

	tests := []struct {
		name        string
		entries     []DepReportEntry
		needsAction bool
	}{
		{
			name:        "all OK",
			entries:     []DepReportEntry{{Status: DepOK}, {Status: DepOK}},
			needsAction: false,
		},
		{
			name:        "has missing",
			entries:     []DepReportEntry{{Status: DepOK}, {Status: DepMissing}},
			needsAction: true,
		},
		{
			name:        "has mismatched",
			entries:     []DepReportEntry{{Status: DepOK}, {Status: DepVersionMismatch}},
			needsAction: true,
		},
		{
			name:        "has unknown version",
			entries:     []DepReportEntry{{Status: DepOK}, {Status: DepProbeUnknownVersion}},
			needsAction: false,
		},
		{
			name:        "empty",
			entries:     []DepReportEntry{},
			needsAction: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := DepReport{Entries: tc.entries}
			assert.Equal(t, tc.needsAction, report.NeedsAction())
		})
	}
}

func TestDepStatus_String(t *testing.T) {
	t.Helper()

	tests := []struct {
		status   DepStatus
		expected string
	}{
		{DepOK, "OK"},
		{DepMissing, "Missing"},
		{DepVersionMismatch, "VersionMismatch"},
		{DepProbeUnknownVersion, "ProbeUnknownVersion"},
		{DepStatus(999), "Unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.status.String())
		})
	}
}

func TestMatchVersion_InvalidVersion(t *testing.T) {
	t.Helper()

	_, err := MatchVersion("not-a-valid-version", ">=1.0.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid version")
}
