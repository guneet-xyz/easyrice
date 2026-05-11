package deps

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstall_SuccessfulRegistryMethod(t *testing.T) {
	t.Helper()

	dep := ResolvedDependency{
		Name:    "test-dep",
		Version: "1.0.0",
		Probe: ProbeSpec{
			Command:      []string{"test-dep", "--version"},
			VersionRegex: `(\d+\.\d+\.\d+)`,
		},
		Methods: []InstallMethod{
			{
				ID:           "apt",
				Description:  "Install via apt",
				OS:           "linux",
				Command:      []string{"apt-get", "install", "-y", "test-dep"},
				RequiresRoot: false,
			},
		},
	}

	runner := &MockRunner{
		Expectations: []MockExpectation{
			{
				Argv:   []string{"apt-get", "install", "-y", "test-dep"},
				Result: RunResult{ExitCode: 0, Stdout: []byte(""), Stderr: []byte("")},
				Err:    nil,
			},
			{
				Argv:   []string{"test-dep", "--version"},
				Result: RunResult{ExitCode: 0, Combined: []byte("test-dep version 1.0.0\n")},
				Err:    nil,
			},
		},
	}

	choices := []InstallChoice{
		{
			Dep:    dep,
			Method: dep.Methods[0],
		},
	}

	outcomes, err := Install(context.Background(), runner, choices)

	require.NoError(t, err)
	require.Len(t, outcomes, 1)

	outcome := outcomes[0]
	assert.Equal(t, "test-dep", outcome.Installed.Name)
	assert.Equal(t, "1.0.0", outcome.Installed.Version)
	assert.Equal(t, "apt", outcome.Installed.Method)
	assert.True(t, outcome.Installed.ManagedByEasyrice)
	assert.WithinDuration(t, time.Now(), outcome.Installed.InstalledAt, 1*time.Second)
}

func TestInstall_SuccessfulShellPayload(t *testing.T) {
	t.Helper()

	dep := ResolvedDependency{
		Name:    "custom-dep",
		Version: "2.0.0",
		Probe: ProbeSpec{
			Command:      []string{"custom-dep", "--version"},
			VersionRegex: `(\d+\.\d+\.\d+)`,
		},
		Methods: []InstallMethod{
			{
				ID:           "custom",
				Description:  "Custom install",
				OS:           "linux",
				ShellPayload: "curl -fsSL https://example.com/install.sh | sh",
				RequiresRoot: false,
			},
		},
	}

	runner := &MockRunner{
		Expectations: []MockExpectation{
			{
				Argv:   []string{"sh", "-c", "curl -fsSL https://example.com/install.sh | sh"},
				Result: RunResult{ExitCode: 0, Combined: []byte("Installing...\n")},
				Err:    nil,
			},
			{
				Argv:   []string{"custom-dep", "--version"},
				Result: RunResult{ExitCode: 0, Combined: []byte("custom-dep 2.0.0\n")},
				Err:    nil,
			},
		},
	}

	choices := []InstallChoice{
		{
			Dep:    dep,
			Method: dep.Methods[0],
		},
	}

	outcomes, err := Install(context.Background(), runner, choices)

	require.NoError(t, err)
	require.Len(t, outcomes, 1)

	outcome := outcomes[0]
	assert.Equal(t, "custom-dep", outcome.Installed.Name)
	assert.Equal(t, "2.0.0", outcome.Installed.Version)
	assert.Equal(t, "custom", outcome.Installed.Method)
}

func TestInstall_RequiresRootButNotRoot(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping test when running as root")
	}

	dep := ResolvedDependency{
		Name:    "root-dep",
		Version: "1.0.0",
		Probe: ProbeSpec{
			Command: []string{"root-dep", "--version"},
		},
		Methods: []InstallMethod{
			{
				ID:           "apt",
				Description:  "Install via apt",
				OS:           "linux",
				Command:      []string{"apt-get", "install", "-y", "root-dep"},
				RequiresRoot: true,
			},
		},
	}

	runner := &MockRunner{
		Expectations: []MockExpectation{},
	}

	choices := []InstallChoice{
		{
			Dep:    dep,
			Method: dep.Methods[0],
		},
	}

	outcomes, err := Install(context.Background(), runner, choices)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires root")
	assert.Contains(t, err.Error(), "sudo")
	assert.Len(t, outcomes, 0)
}

func TestInstall_NonZeroExitFromCommand(t *testing.T) {
	t.Helper()

	dep := ResolvedDependency{
		Name:    "failing-dep",
		Version: "1.0.0",
		Probe: ProbeSpec{
			Command: []string{"failing-dep", "--version"},
		},
		Methods: []InstallMethod{
			{
				ID:           "apt",
				Description:  "Install via apt",
				OS:           "linux",
				Command:      []string{"apt-get", "install", "-y", "failing-dep"},
				RequiresRoot: false,
			},
		},
	}

	runner := &MockRunner{
		Expectations: []MockExpectation{
			{
				Argv: []string{"apt-get", "install", "-y", "failing-dep"},
				Result: RunResult{
					ExitCode: 1,
					Stderr:   []byte("E: Unable to locate package failing-dep\n"),
				},
				Err: nil,
			},
		},
	}

	choices := []InstallChoice{
		{
			Dep:    dep,
			Method: dep.Methods[0],
		},
	}

	outcomes, err := Install(context.Background(), runner, choices)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed (exit 1)")
	assert.Contains(t, err.Error(), "failing-dep")
	assert.Len(t, outcomes, 0)
}

func TestInstall_PostInstallProbeShowsNotInstalled(t *testing.T) {
	t.Helper()

	dep := ResolvedDependency{
		Name:    "broken-dep",
		Version: "1.0.0",
		Probe: ProbeSpec{
			Command: []string{"broken-dep", "--version"},
		},
		Methods: []InstallMethod{
			{
				ID:           "apt",
				Description:  "Install via apt",
				OS:           "linux",
				Command:      []string{"apt-get", "install", "-y", "broken-dep"},
				RequiresRoot: false,
			},
		},
	}

	runner := &MockRunner{
		Expectations: []MockExpectation{
			{
				Argv:   []string{"apt-get", "install", "-y", "broken-dep"},
				Result: RunResult{ExitCode: 0, Stdout: []byte("")},
				Err:    nil,
			},
			{
				Argv:   []string{"broken-dep", "--version"},
				Result: RunResult{ExitCode: 1, Combined: []byte("command not found\n")},
				Err:    nil,
			},
		},
	}

	choices := []InstallChoice{
		{
			Dep:    dep,
			Method: dep.Methods[0],
		},
	}

	outcomes, err := Install(context.Background(), runner, choices)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "probe still reports missing")
	assert.Len(t, outcomes, 0)
}

func TestInstall_MultipleChoices_FirstSucceedsSecondFails(t *testing.T) {
	t.Helper()

	dep1 := ResolvedDependency{
		Name:    "dep1",
		Version: "1.0.0",
		Probe: ProbeSpec{
			Command:      []string{"dep1", "--version"},
			VersionRegex: `(\d+\.\d+\.\d+)`,
		},
		Methods: []InstallMethod{
			{
				ID:           "apt",
				Command:      []string{"apt-get", "install", "-y", "dep1"},
				RequiresRoot: false,
			},
		},
	}

	dep2 := ResolvedDependency{
		Name:    "dep2",
		Version: "2.0.0",
		Probe: ProbeSpec{
			Command: []string{"dep2", "--version"},
		},
		Methods: []InstallMethod{
			{
				ID:           "apt",
				Command:      []string{"apt-get", "install", "-y", "dep2"},
				RequiresRoot: false,
			},
		},
	}

	runner := &MockRunner{
		Expectations: []MockExpectation{
			{
				Argv:   []string{"apt-get", "install", "-y", "dep1"},
				Result: RunResult{ExitCode: 0},
				Err:    nil,
			},
			{
				Argv:   []string{"dep1", "--version"},
				Result: RunResult{ExitCode: 0, Combined: []byte("dep1 1.0.0\n")},
				Err:    nil,
			},
			{
				Argv: []string{"apt-get", "install", "-y", "dep2"},
				Result: RunResult{
					ExitCode: 1,
					Stderr:   []byte("E: Unable to locate package dep2\n"),
				},
				Err: nil,
			},
		},
	}

	choices := []InstallChoice{
		{
			Dep:    dep1,
			Method: dep1.Methods[0],
		},
		{
			Dep:    dep2,
			Method: dep2.Methods[0],
		},
	}

	outcomes, err := Install(context.Background(), runner, choices)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "dep2")
	assert.Len(t, outcomes, 1)
	assert.Equal(t, "dep1", outcomes[0].Installed.Name)
}

func TestInstall_EmptyChoices(t *testing.T) {
	t.Helper()

	runner := &MockRunner{
		Expectations: []MockExpectation{},
	}

	outcomes, err := Install(context.Background(), runner, []InstallChoice{})

	require.NoError(t, err)
	assert.Len(t, outcomes, 0)
}
