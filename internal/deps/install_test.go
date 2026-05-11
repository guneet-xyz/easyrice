package deps

import (
	"context"
	"errors"
	"os"
	"strings"
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

func TestFormatCommand(t *testing.T) {
	cases := []struct {
		name   string
		method InstallMethod
		want   string
	}{
		{
			name:   "command with args",
			method: InstallMethod{Command: []string{"apt-get", "install", "-y", "vim"}},
			want:   "[apt-get install -y vim]",
		},
		{
			name:   "single command no args",
			method: InstallMethod{Command: []string{"true"}},
			want:   "[true]",
		},
		{
			name:   "shell payload",
			method: InstallMethod{ShellPayload: "curl -fsSL https://example.com | sh"},
			want:   `sh -c "curl -fsSL https://example.com | sh"`,
		},
		{
			name:   "no command and no shell payload",
			method: InstallMethod{ID: "broken"},
			want:   "(no command)",
		},
		{
			name:   "command takes precedence over shell payload",
			method: InstallMethod{Command: []string{"echo", "hi"}, ShellPayload: "ignored"},
			want:   "[echo hi]",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := formatCommand(tc.method)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestInstall_RequiresRoot_ErrorMessageIncludesFormattedShellPayload(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping test when running as root")
	}

	dep := ResolvedDependency{
		Name:  "shell-root-dep",
		Probe: ProbeSpec{Command: []string{"shell-root-dep", "--version"}},
		Methods: []InstallMethod{
			{
				ID:           "custom",
				ShellPayload: "curl -fsSL https://example.com/install.sh | sh",
				RequiresRoot: true,
			},
		},
	}

	runner := &MockRunner{}
	outcomes, err := Install(context.Background(), runner, []InstallChoice{{Dep: dep, Method: dep.Methods[0]}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires root")
	assert.Contains(t, err.Error(), `sh -c "curl -fsSL https://example.com/install.sh | sh"`)
	assert.Empty(t, outcomes)
	assert.Empty(t, runner.Calls, "no commands should run when root check fails")
}

func TestInstall_RequiresRoot_ErrorMessageIncludesNoCommandSentinel(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping test when running as root")
	}

	dep := ResolvedDependency{
		Name:  "empty-root-dep",
		Probe: ProbeSpec{Command: []string{"empty-root-dep", "--version"}},
		Methods: []InstallMethod{
			{
				ID:           "weird",
				RequiresRoot: true,
			},
		},
	}

	runner := &MockRunner{}
	outcomes, err := Install(context.Background(), runner, []InstallChoice{{Dep: dep, Method: dep.Methods[0]}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires root")
	assert.Contains(t, err.Error(), "(no command)")
	assert.Empty(t, outcomes)
}

func TestInstall_MethodHasNeitherCommandNorShellPayload(t *testing.T) {
	dep := ResolvedDependency{
		Name:  "no-cmd-dep",
		Probe: ProbeSpec{Command: []string{"no-cmd-dep", "--version"}},
		Methods: []InstallMethod{
			{ID: "broken", RequiresRoot: false},
		},
	}

	runner := &MockRunner{}
	outcomes, err := Install(context.Background(), runner, []InstallChoice{{Dep: dep, Method: dep.Methods[0]}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "neither command nor shell_payload")
	assert.Contains(t, err.Error(), "no-cmd-dep")
	assert.Empty(t, outcomes)
	assert.Empty(t, runner.Calls)
}

func TestInstall_RunnerErrOnCommandMethod(t *testing.T) {
	dep := ResolvedDependency{
		Name:  "exec-fail",
		Probe: ProbeSpec{Command: []string{"exec-fail", "--version"}},
		Methods: []InstallMethod{
			{
				ID:      "apt",
				Command: []string{"apt-get", "install", "-y", "exec-fail"},
			},
		},
	}

	runner := &MockRunner{
		Expectations: []MockExpectation{
			{
				Argv: []string{"apt-get", "install", "-y", "exec-fail"},
				Err:  errors.New("fork/exec failed"),
			},
		},
	}

	outcomes, err := Install(context.Background(), runner, []InstallChoice{{Dep: dep, Method: dep.Methods[0]}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), `installing "exec-fail" via "apt"`)
	assert.Contains(t, err.Error(), "fork/exec failed")
	assert.Empty(t, outcomes)
}

func TestInstall_ShellPayload_NonZeroExit(t *testing.T) {
	dep := ResolvedDependency{
		Name:  "shell-exit",
		Probe: ProbeSpec{Command: []string{"shell-exit", "--version"}},
		Methods: []InstallMethod{
			{
				ID:           "custom",
				ShellPayload: "false",
			},
		},
	}

	runner := &MockRunner{
		Expectations: []MockExpectation{
			{
				Argv:   []string{"sh", "-c", "false"},
				Result: RunResult{ExitCode: 1, Combined: []byte("script bombed\n")},
			},
		},
	}

	outcomes, err := Install(context.Background(), runner, []InstallChoice{{Dep: dep, Method: dep.Methods[0]}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed (exit 1)")
	assert.Contains(t, err.Error(), "script bombed")
	assert.Empty(t, outcomes)
}

func TestInstall_ShellPayload_RunnerErr(t *testing.T) {
	dep := ResolvedDependency{
		Name:  "shell-err",
		Probe: ProbeSpec{Command: []string{"shell-err", "--version"}},
		Methods: []InstallMethod{
			{
				ID:           "custom",
				ShellPayload: "irrelevant",
			},
		},
	}

	runner := &MockRunner{
		Expectations: []MockExpectation{
			{
				Argv: []string{"sh", "-c", "irrelevant"},
				Err:  errors.New("sh not found"),
			},
		},
	}

	outcomes, err := Install(context.Background(), runner, []InstallChoice{{Dep: dep, Method: dep.Methods[0]}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), `installing "shell-err" via "custom"`)
	assert.Contains(t, err.Error(), "sh not found")
	assert.Empty(t, outcomes)
}

func TestInstall_PostInstallProbeErrors(t *testing.T) {
	dep := ResolvedDependency{
		Name:  "probe-err",
		Probe: ProbeSpec{Command: []string{"probe-err", "--version"}},
		Methods: []InstallMethod{
			{
				ID:      "apt",
				Command: []string{"apt-get", "install", "-y", "probe-err"},
			},
		},
	}

	runner := &MockRunner{
		Expectations: []MockExpectation{
			{
				Argv:   []string{"apt-get", "install", "-y", "probe-err"},
				Result: RunResult{ExitCode: 0},
			},
			{
				Argv: []string{"probe-err", "--version"},
				Err:  errors.New("probe blew up"),
			},
		},
	}

	outcomes, err := Install(context.Background(), runner, []InstallChoice{{Dep: dep, Method: dep.Methods[0]}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "post-install probe")
	assert.Contains(t, err.Error(), "probe-err")
	assert.True(t, strings.Contains(err.Error(), "probe blew up"), "expected wrapped runner error, got: %v", err)
	assert.Empty(t, outcomes)
}

func TestInstall_PartialOutcomes_ThirdFails(t *testing.T) {
	mkDep := func(name string) ResolvedDependency {
		return ResolvedDependency{
			Name:  name,
			Probe: ProbeSpec{Command: []string{name, "--version"}, VersionRegex: `(\d+\.\d+\.\d+)`},
			Methods: []InstallMethod{
				{ID: "apt", Command: []string{"apt-get", "install", "-y", name}},
			},
		}
	}

	dep1 := mkDep("alpha")
	dep2 := mkDep("beta")
	dep3 := mkDep("gamma")

	runner := &MockRunner{
		Expectations: []MockExpectation{
			{Argv: []string{"apt-get", "install", "-y", "alpha"}, Result: RunResult{ExitCode: 0}},
			{Argv: []string{"alpha", "--version"}, Result: RunResult{ExitCode: 0, Combined: []byte("alpha 1.0.0\n")}},
			{Argv: []string{"apt-get", "install", "-y", "beta"}, Result: RunResult{ExitCode: 0}},
			{Argv: []string{"beta", "--version"}, Result: RunResult{ExitCode: 0, Combined: []byte("beta 2.0.0\n")}},
			{Argv: []string{"apt-get", "install", "-y", "gamma"}, Result: RunResult{ExitCode: 2, Stderr: []byte("kaboom\n")}},
		},
	}

	choices := []InstallChoice{
		{Dep: dep1, Method: dep1.Methods[0]},
		{Dep: dep2, Method: dep2.Methods[0]},
		{Dep: dep3, Method: dep3.Methods[0]},
	}

	outcomes, err := Install(context.Background(), runner, choices)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "gamma")
	require.Len(t, outcomes, 2, "first two outcomes must be preserved when third fails")
	assert.Equal(t, "alpha", outcomes[0].Installed.Name)
	assert.Equal(t, "1.0.0", outcomes[0].Installed.Version)
	assert.Equal(t, "beta", outcomes[1].Installed.Name)
	assert.Equal(t, "2.0.0", outcomes[1].Installed.Version)
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
