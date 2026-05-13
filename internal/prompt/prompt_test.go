package prompt

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/guneet-xyz/easyrice/internal/deps"
	"github.com/guneet-xyz/easyrice/internal/plan"
	"github.com/guneet-xyz/easyrice/internal/style"
	"github.com/stretchr/testify/assert"
)

func TestRenderPlan_EmptyInstall(t *testing.T) {
	p := &plan.Plan{
		PackageName: "test",
		Profile:     "default",
		Ops:         []plan.Op{},
	}

	var buf bytes.Buffer
	RenderPlan(&buf, p)
	output := buf.String()

	assert.Contains(t, output, "Plan: install test using profile default")
	assert.Contains(t, output, "Total: 0 link(s) to create.")
}

func TestRenderPlan_CreateOps(t *testing.T) {
	p := &plan.Plan{
		PackageName: "test",
		Profile:     "default",
		Ops: []plan.Op{
			{Kind: plan.OpCreate, Source: "src/file1", Target: "/home/user/.config/file1"},
			{Kind: plan.OpCreate, Source: "src/file2", Target: "/home/user/.config/file2"},
		},
	}

	var buf bytes.Buffer
	RenderPlan(&buf, p)
	output := buf.String()

	assert.Contains(t, output, "Plan: install test using profile default")
	assert.Contains(t, output, "CREATE")
	assert.Contains(t, output, "/home/user/.config/file1")
	assert.Contains(t, output, "src/file1")
	assert.Contains(t, output, "/home/user/.config/file2")
	assert.Contains(t, output, "src/file2")
	assert.Contains(t, output, "Total: 2 link(s) to create.")
}

func TestRenderPlan_RemoveOps(t *testing.T) {
	p := &plan.Plan{
		PackageName: "test",
		Ops: []plan.Op{
			{Kind: plan.OpRemove, Target: "/home/user/.config/file1"},
			{Kind: plan.OpRemove, Target: "/home/user/.config/file2"},
		},
	}

	var buf bytes.Buffer
	RenderPlan(&buf, p)
	output := buf.String()

	assert.Contains(t, output, "Plan: uninstall test")
	assert.Contains(t, output, "REMOVE")
	assert.Contains(t, output, "/home/user/.config/file1")
	assert.Contains(t, output, "/home/user/.config/file2")
	assert.Contains(t, output, "Total: 2 link(s) to remove.")
}

func TestRenderPlan_ManyOps(t *testing.T) {
	ops := make([]plan.Op, 100)
	for i := 0; i < 100; i++ {
		ops[i] = plan.Op{
			Kind:   plan.OpCreate,
			Source: "src/file",
			Target: "/home/user/.config/file",
		}
	}

	p := &plan.Plan{
		PackageName: "test",
		Profile:     "default",
		Ops:         ops,
	}

	var buf bytes.Buffer
	RenderPlan(&buf, p)
	output := buf.String()

	assert.Contains(t, output, "Total: 100 link(s) to create.")
	count := strings.Count(output, "CREATE")
	assert.Equal(t, 100, count)
}

func TestRenderConflicts(t *testing.T) {
	conflicts := []plan.Conflict{
		{Target: "/home/user/.config/file1", Reason: "already exists"},
		{Target: "/home/user/.config/file2", Reason: "is a directory"},
	}

	var buf bytes.Buffer
	RenderConflicts(&buf, conflicts)
	output := buf.String()

	assert.Contains(t, output, "CONFLICT  /home/user/.config/file1: already exists")
	assert.Contains(t, output, "CONFLICT  /home/user/.config/file2: is a directory")
}

func TestConfirm_Yes(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"lowercase y", "y\n"},
		{"uppercase Y", "Y\n"},
		{"lowercase yes", "yes\n"},
		{"uppercase YES", "YES\n"},
		{"mixed case Yes", "Yes\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := strings.NewReader(tt.input)
			var out bytes.Buffer

			result, err := Confirm(in, &out, "Continue")
			assert.NoError(t, err)
			assert.True(t, result)
			assert.Contains(t, out.String(), "Continue [y/N]: ")
		})
	}
}

func TestConfirm_No(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"bare enter", "\n"},
		{"lowercase n", "n\n"},
		{"uppercase N", "N\n"},
		{"lowercase no", "no\n"},
		{"uppercase NO", "NO\n"},
		{"random input", "asdf\n"},
		{"spaces", "   \n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := strings.NewReader(tt.input)
			var out bytes.Buffer

			result, err := Confirm(in, &out, "Continue")
			assert.NoError(t, err)
			assert.False(t, result)
			assert.Contains(t, out.String(), "Continue [y/N]: ")
		})
	}
}

func TestConfirm_EOF(t *testing.T) {
	in := strings.NewReader("")
	var out bytes.Buffer

	result, err := Confirm(in, &out, "Continue")
	assert.NoError(t, err)
	assert.False(t, result)
}

func TestConfirm_Error(t *testing.T) {
	in := &errorReader{}
	var out bytes.Buffer

	result, err := Confirm(in, &out, "Continue")
	assert.Error(t, err)
	assert.False(t, result)
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestRenderPlan_NilPlan(t *testing.T) {
	var buf bytes.Buffer
	RenderPlan(&buf, nil)
	output := buf.String()
	assert.Empty(t, output)
}

func TestRenderPlan_WithConflicts(t *testing.T) {
	p := &plan.Plan{
		PackageName: "test",
		Profile:     "default",
		Ops: []plan.Op{
			{Kind: plan.OpCreate, Source: "src/file1", Target: "/home/user/.config/file1"},
		},
		Conflicts: []plan.Conflict{
			{Target: "/home/user/.config/conflict1", Reason: "already exists"},
			{Target: "/home/user/.config/conflict2", Reason: "is a directory"},
		},
	}

	var buf bytes.Buffer
	RenderPlan(&buf, p)
	output := buf.String()

	assert.Contains(t, output, "Plan: install test using profile default")
	assert.Contains(t, output, "CREATE")
	assert.Contains(t, output, "Total: 1 link(s) to create.")
	assert.Contains(t, output, "Conflicts (2):")
	assert.Contains(t, output, "CONFLICT  /home/user/.config/conflict1: already exists")
	assert.Contains(t, output, "CONFLICT  /home/user/.config/conflict2: is a directory")
}

func TestRenderPlan_DirectoryOps(t *testing.T) {
	p := &plan.Plan{
		PackageName: "test",
		Profile:     "default",
		Ops: []plan.Op{
			{Kind: plan.OpCreate, Source: "src/dir", Target: "/home/user/.config/dir", IsDir: true},
			{Kind: plan.OpRemove, Target: "/home/user/.config/olddir", IsDir: true},
		},
	}

	var buf bytes.Buffer
	RenderPlan(&buf, p)
	output := buf.String()

	assert.Contains(t, output, "CREATE-DIR")
	assert.Contains(t, output, "REMOVE-DIR")
	assert.Contains(t, output, "/home/user/.config/dir")
	assert.Contains(t, output, "/home/user/.config/olddir")
}

func TestRenderPlan_MultipleOps(t *testing.T) {
	ops := []plan.Op{
		{Kind: plan.OpCreate, Source: "src/file1", Target: "/home/user/.config/file1"},
		{Kind: plan.OpCreate, Source: "src/file2", Target: "/home/user/.config/file2"},
		{Kind: plan.OpCreate, Source: "src/file3", Target: "/home/user/.config/file3"},
	}

	p := &plan.Plan{
		PackageName: "test",
		Profile:     "default",
		Ops:         ops,
	}

	var buf bytes.Buffer
	RenderPlan(&buf, p)
	output := buf.String()

	assert.Contains(t, output, "Plan: install test using profile default")
	assert.Contains(t, output, "src/file1")
	assert.Contains(t, output, "src/file2")
	assert.Contains(t, output, "src/file3")
	assert.Contains(t, output, "/home/user/.config/file1")
	assert.Contains(t, output, "/home/user/.config/file2")
	assert.Contains(t, output, "/home/user/.config/file3")
	assert.Contains(t, output, "Total: 3 link(s) to create.")
}

func TestRenderConflicts_NoConflicts(t *testing.T) {
	var buf bytes.Buffer
	RenderConflicts(&buf, []plan.Conflict{})
	output := buf.String()
	assert.Empty(t, output)
}

func TestRenderConflicts_DirectoryConflicts(t *testing.T) {
	conflicts := []plan.Conflict{
		{Target: "/home/user/.config/dir1", Reason: "already exists", IsDir: true},
		{Target: "/home/user/.config/dir2", Reason: "is a file", IsDir: true},
	}

	var buf bytes.Buffer
	RenderConflicts(&buf, conflicts)
	output := buf.String()

	assert.Contains(t, output, "CONFLICT  /home/user/.config/dir1: already exists (directory)")
	assert.Contains(t, output, "CONFLICT  /home/user/.config/dir2: is a file (directory)")
}

func TestRenderConflicts_MixedConflicts(t *testing.T) {
	conflicts := []plan.Conflict{
		{Target: "/home/user/.config/file", Reason: "already exists", IsDir: false},
		{Target: "/home/user/.config/dir", Reason: "is a directory", IsDir: true},
	}

	var buf bytes.Buffer
	RenderConflicts(&buf, conflicts)
	output := buf.String()

	assert.Contains(t, output, "CONFLICT  /home/user/.config/file: already exists")
	assert.NotContains(t, output, "already exists (directory)")
	assert.Contains(t, output, "CONFLICT  /home/user/.config/dir: is a directory (directory)")
}

func TestSelect_ValidChoice(t *testing.T) {
	in := strings.NewReader("1\n")
	var out bytes.Buffer

	options := []SelectOption{
		{Label: "Option A", Description: "First option"},
		{Label: "Option B", Description: "Second option"},
	}

	idx, err := Select(in, &out, "Choose one", options, 0)
	assert.NoError(t, err)
	assert.Equal(t, 0, idx)
	assert.Contains(t, out.String(), "Choose one")
	assert.Contains(t, out.String(), "1) Option A — First option")
	assert.Contains(t, out.String(), "2) Option B — Second option")
}

func TestSelect_EmptyInputUsesDefault(t *testing.T) {
	in := strings.NewReader("\n")
	var out bytes.Buffer

	options := []SelectOption{
		{Label: "Option A"},
		{Label: "Option B"},
		{Label: "Option C"},
	}

	idx, err := Select(in, &out, "Choose", options, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, idx)
}

func TestSelect_OutOfRangeThenValid(t *testing.T) {
	in := strings.NewReader("9\n1\n")
	var out bytes.Buffer

	options := []SelectOption{
		{Label: "Option A"},
		{Label: "Option B"},
	}

	idx, err := Select(in, &out, "Choose", options, 0)
	assert.NoError(t, err)
	assert.Equal(t, 0, idx)
	assert.Contains(t, out.String(), "Invalid choice")
}

func TestSelect_ThreeInvalidInputsReturnsError(t *testing.T) {
	in := strings.NewReader("9\n10\n11\n")
	var out bytes.Buffer

	options := []SelectOption{
		{Label: "Option A"},
		{Label: "Option B"},
	}

	idx, err := Select(in, &out, "Choose", options, 0)
	assert.Error(t, err)
	assert.Equal(t, 0, idx)
	assert.Contains(t, err.Error(), "too many invalid choices")
}

func TestSelect_EOFReturnsError(t *testing.T) {
	in := strings.NewReader("")
	var out bytes.Buffer

	options := []SelectOption{
		{Label: "Option A"},
	}

	_, err := Select(in, &out, "Choose", options, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "EOF before selection")
}

func TestSelect_NoDescription(t *testing.T) {
	in := strings.NewReader("1\n")
	var out bytes.Buffer

	options := []SelectOption{
		{Label: "Option A"},
		{Label: "Option B"},
	}

	idx, err := Select(in, &out, "Choose", options, 0)
	assert.NoError(t, err)
	assert.Equal(t, 0, idx)
	assert.Contains(t, out.String(), "1) Option A\n")
	assert.NotContains(t, out.String(), "—")
}

func TestSelectWithDefault_AutoAcceptTrue(t *testing.T) {
	var out bytes.Buffer

	options := []SelectOption{
		{Label: "Option A"},
		{Label: "Option B"},
		{Label: "Option C"},
	}

	idx, err := SelectWithDefault(nil, &out, "Choose", options, 1, true)
	assert.NoError(t, err)
	assert.Equal(t, 1, idx)
	assert.Contains(t, out.String(), "Using default: Option B")
}

func TestSelectWithDefault_AutoAcceptFalse(t *testing.T) {
	in := strings.NewReader("2\n")
	var out bytes.Buffer

	options := []SelectOption{
		{Label: "Option A"},
		{Label: "Option B"},
	}

	idx, err := SelectWithDefault(in, &out, "Choose", options, 0, false)
	assert.NoError(t, err)
	assert.Equal(t, 1, idx)
	assert.NotContains(t, out.String(), "Using:")
}

func TestSelect_APISignature(t *testing.T) {
	var _ func(io.Reader, io.Writer, string, []SelectOption, int) (int, error) = Select
}

func TestRenderDepReport_AllStatuses(t *testing.T) {
	report := deps.DepReport{
		Entries: []deps.DepReportEntry{
			{
				Dep:              deps.ResolvedDependency{Name: "ripgrep"},
				Status:           deps.DepOK,
				InstalledVersion: "14.1.0",
			},
			{
				Dep:    deps.ResolvedDependency{Name: "neovim"},
				Status: deps.DepMissing,
			},
			{
				Dep:              deps.ResolvedDependency{Name: "node", Version: ">=20"},
				Status:           deps.DepVersionMismatch,
				InstalledVersion: "18.20.0",
			},
			{
				Dep:              deps.ResolvedDependency{Name: "mdformat"},
				Status:           deps.DepProbeUnknownVersion,
				InstalledVersion: "",
			},
		},
	}

	var buf bytes.Buffer
	RenderDepReport(&buf, report)
	output := buf.String()

	assert.Contains(t, output, "Dependency check:")
	assert.Contains(t, output, "✓ ripgrep (14.1.0) — installed")
	assert.Contains(t, output, "✗ neovim — missing")
	assert.Contains(t, output, "! node (18.20.0) — version mismatch; required >=20")
	assert.Contains(t, output, "? mdformat — installed (version unknown)")
}

func TestRenderDepReport_Empty(t *testing.T) {
	report := deps.DepReport{Entries: []deps.DepReportEntry{}}

	var buf bytes.Buffer
	RenderDepReport(&buf, report)
	output := buf.String()

	assert.Contains(t, output, "Dependency check:")
	assert.Equal(t, "Dependency check:\n", output)
}

func TestRenderDepReport_PlainMode(t *testing.T) {
	style.SetPlain(true)
	t.Cleanup(func() { style.SetPlain(false) })

	report := deps.DepReport{
		Entries: []deps.DepReportEntry{
			{
				Dep:              deps.ResolvedDependency{Name: "ripgrep"},
				Status:           deps.DepOK,
				InstalledVersion: "14.1.0",
			},
			{
				Dep:    deps.ResolvedDependency{Name: "neovim"},
				Status: deps.DepMissing,
			},
			{
				Dep:              deps.ResolvedDependency{Name: "node", Version: ">=20"},
				Status:           deps.DepVersionMismatch,
				InstalledVersion: "18.20.0",
			},
			{
				Dep:              deps.ResolvedDependency{Name: "mdformat"},
				Status:           deps.DepProbeUnknownVersion,
				InstalledVersion: "",
			},
		},
	}

	var buf bytes.Buffer
	RenderDepReport(&buf, report)
	output := buf.String()

	assert.Contains(t, output, "OK ripgrep")
	assert.Contains(t, output, "FAIL neovim")
	assert.Contains(t, output, "WARN node")
	assert.Contains(t, output, "? mdformat")
	assert.Contains(t, output, "--")
	assert.NotContains(t, output, "✓")
	assert.NotContains(t, output, "✗")
	assert.NotContains(t, output, "—")
}

func TestSelectInstallMethod_AutoAcceptTrue(t *testing.T) {
	entry := deps.DepReportEntry{
		Dep: deps.ResolvedDependency{Name: "ripgrep"},
		Methods: []deps.InstallMethod{
			{ID: "apt", Description: "apt install ripgrep", Command: []string{"apt", "install", "ripgrep"}},
			{ID: "brew", Description: "brew install ripgrep", Command: []string{"brew", "install", "ripgrep"}},
		},
	}

	in := strings.NewReader("")
	var out bytes.Buffer

	method, err := SelectInstallMethod(in, &out, entry, true)
	assert.NoError(t, err)
	assert.Equal(t, "apt", method.ID)
	assert.Contains(t, out.String(), "Using default: apt install ripgrep")
}

func TestSelectInstallMethod_UserSelection(t *testing.T) {
	entry := deps.DepReportEntry{
		Dep: deps.ResolvedDependency{Name: "ripgrep"},
		Methods: []deps.InstallMethod{
			{ID: "apt", Description: "apt install ripgrep", Command: []string{"apt", "install", "ripgrep"}},
			{ID: "brew", Description: "brew install ripgrep", Command: []string{"brew", "install", "ripgrep"}},
		},
	}

	in := strings.NewReader("2\n")
	var out bytes.Buffer

	method, err := SelectInstallMethod(in, &out, entry, false)
	assert.NoError(t, err)
	assert.Equal(t, "brew", method.ID)
	assert.Contains(t, out.String(), "Choose how to install ripgrep:")
}

func TestSelectInstallMethod_CustomMethodConfirmYes(t *testing.T) {
	entry := deps.DepReportEntry{
		Dep: deps.ResolvedDependency{Name: "custom-tool"},
		Methods: []deps.InstallMethod{
			{ID: "custom", Description: "custom install", ShellPayload: "curl https://example.com/install.sh | sh"},
		},
	}

	in := strings.NewReader("y\n")
	var out bytes.Buffer

	method, err := SelectInstallMethod(in, &out, entry, true)
	assert.NoError(t, err)
	assert.Equal(t, "custom", method.ID)
	assert.Contains(t, out.String(), "Run this command from rice.toml?")
}

func TestSelectInstallMethod_CustomMethodConfirmNo(t *testing.T) {
	entry := deps.DepReportEntry{
		Dep: deps.ResolvedDependency{Name: "custom-tool"},
		Methods: []deps.InstallMethod{
			{ID: "custom", Description: "custom install", ShellPayload: "curl https://example.com/install.sh | sh"},
		},
	}

	in := strings.NewReader("n\n")
	var out bytes.Buffer

	method, err := SelectInstallMethod(in, &out, entry, true)
	assert.Equal(t, ErrUserDeclined, err)
	assert.Equal(t, deps.InstallMethod{}, method)
}

func TestSelectInstallMethod_NoMethods(t *testing.T) {
	entry := deps.DepReportEntry{
		Dep:     deps.ResolvedDependency{Name: "unknown-tool"},
		Methods: []deps.InstallMethod{},
	}

	in := strings.NewReader("")
	var out bytes.Buffer

	method, err := SelectInstallMethod(in, &out, entry, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no install methods available for")
	assert.Equal(t, deps.InstallMethod{}, method)
}

func TestShorten(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncate", "hello world", 5, "hello..."},
		{"empty string", "", 5, ""},
		{"single char truncate", "a", 0, "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shorten(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSelectWithDefault_AutoAcceptWithInvalidDefaultIdx(t *testing.T) {
	var out bytes.Buffer

	options := []SelectOption{
		{Label: "Option A"},
		{Label: "Option B"},
	}

	idx, err := SelectWithDefault(nil, &out, "Choose", options, 2, true)
	assert.NoError(t, err)
	assert.Equal(t, 2, idx)
	assert.NotContains(t, out.String(), "Using:")
}

func TestSelectWithDefault_AutoAcceptWithNegativeDefaultIdx(t *testing.T) {
	var out bytes.Buffer

	options := []SelectOption{
		{Label: "Option A"},
		{Label: "Option B"},
	}

	idx, err := SelectWithDefault(nil, &out, "Choose", options, -1, true)
	assert.NoError(t, err)
	assert.Equal(t, -1, idx)
	assert.NotContains(t, out.String(), "Using:")
}

func TestSelectWithDefault_EmptyOptions(t *testing.T) {
	in := strings.NewReader("\n")
	var out bytes.Buffer

	options := []SelectOption{}

	idx, err := SelectWithDefault(in, &out, "Choose", options, 0, false)
	assert.NoError(t, err)
	assert.Equal(t, 0, idx)
}

func TestSelectWithDefault_InvalidInputThenEOF(t *testing.T) {
	in := strings.NewReader("invalid\ninvalid\ninvalid\n")
	var out bytes.Buffer

	options := []SelectOption{
		{Label: "Option A"},
		{Label: "Option B"},
	}

	idx, err := SelectWithDefault(in, &out, "Choose", options, 0, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many invalid choices")
	assert.Equal(t, 0, idx)
}

func TestSelect_PlainMode(t *testing.T) {
	style.SetPlain(true)
	t.Cleanup(func() { style.SetPlain(false) })

	in := strings.NewReader("1\n")
	var out bytes.Buffer

	options := []SelectOption{
		{Label: "Option A", Description: "First option"},
		{Label: "Option B", Description: "Second option"},
	}

	idx, err := SelectWithDefault(in, &out, "Choose", options, 0, false)
	assert.NoError(t, err)
	assert.Equal(t, 0, idx)
	output := out.String()
	assert.Contains(t, output, "1) Option A -- First option")
	assert.NotContains(t, output, "—")
}

func TestRenderDepReport_UnknownStatus(t *testing.T) {
	report := deps.DepReport{
		Entries: []deps.DepReportEntry{
			{
				Dep:              deps.ResolvedDependency{Name: "unknown-status"},
				Status:           999,
				InstalledVersion: "1.0.0",
			},
		},
	}

	var buf bytes.Buffer
	RenderDepReport(&buf, report)
	output := buf.String()

	assert.Contains(t, output, "Dependency check:")
	assert.Contains(t, output, "? unknown-status (1.0.0) — unknown")
}

func TestRenderDepReport_NoVersionInfo(t *testing.T) {
	report := deps.DepReport{
		Entries: []deps.DepReportEntry{
			{
				Dep:              deps.ResolvedDependency{Name: "tool"},
				Status:           deps.DepOK,
				InstalledVersion: "",
			},
		},
	}

	var buf bytes.Buffer
	RenderDepReport(&buf, report)
	output := buf.String()

	assert.Contains(t, output, "Dependency check:")
	assert.Contains(t, output, "✓ tool — installed")
	assert.NotContains(t, output, "()")
}

func TestRenderDepReport_VersionMismatchWithoutVersion(t *testing.T) {
	report := deps.DepReport{
		Entries: []deps.DepReportEntry{
			{
				Dep:              deps.ResolvedDependency{Name: "tool", Version: ""},
				Status:           deps.DepVersionMismatch,
				InstalledVersion: "1.0.0",
			},
		},
	}

	var buf bytes.Buffer
	RenderDepReport(&buf, report)
	output := buf.String()

	assert.Contains(t, output, "Dependency check:")
	assert.Contains(t, output, "! tool (1.0.0) — version mismatch; required ")
}

func TestSelectInstallMethod_SelectWithDefaultError(t *testing.T) {
	entry := deps.DepReportEntry{
		Dep: deps.ResolvedDependency{Name: "ripgrep"},
		Methods: []deps.InstallMethod{
			{ID: "apt", Description: "apt install ripgrep", Command: []string{"apt", "install", "ripgrep"}},
		},
	}

	in := strings.NewReader("")
	var out bytes.Buffer

	method, err := SelectInstallMethod(in, &out, entry, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "EOF before selection")
	assert.Equal(t, deps.InstallMethod{}, method)
}

func TestSelectInstallMethod_MultipleMethodsWithCommand(t *testing.T) {
	entry := deps.DepReportEntry{
		Dep: deps.ResolvedDependency{Name: "ripgrep"},
		Methods: []deps.InstallMethod{
			{ID: "apt", Description: "apt install ripgrep", Command: []string{"apt", "install", "ripgrep"}},
			{ID: "brew", Description: "brew install ripgrep", Command: []string{"brew", "install", "ripgrep"}},
			{ID: "cargo", Description: "cargo install ripgrep", Command: []string{"cargo", "install", "ripgrep"}},
		},
	}

	in := strings.NewReader("3\n")
	var out bytes.Buffer

	method, err := SelectInstallMethod(in, &out, entry, false)
	assert.NoError(t, err)
	assert.Equal(t, "cargo", method.ID)
	assert.Contains(t, out.String(), "Choose how to install ripgrep:")
	assert.Contains(t, out.String(), "1) apt install ripgrep")
	assert.Contains(t, out.String(), "2) brew install ripgrep")
	assert.Contains(t, out.String(), "3) cargo install ripgrep")
}

func TestRenderConflicts_DirectoryConflictWithoutIsDir(t *testing.T) {
	conflicts := []plan.Conflict{
		{Target: "/home/user/.config/file", Reason: "already exists", IsDir: false},
	}

	var buf bytes.Buffer
	RenderConflicts(&buf, conflicts)
	output := buf.String()

	assert.Contains(t, output, "CONFLICT  /home/user/.config/file: already exists")
	assert.NotContains(t, output, "(directory)")
}

func TestRenderConflicts_SingleConflict(t *testing.T) {
	conflicts := []plan.Conflict{
		{Target: "/home/user/.config/file", Reason: "already exists", IsDir: false},
	}

	var buf bytes.Buffer
	RenderConflicts(&buf, conflicts)
	output := buf.String()

	assert.Contains(t, output, "CONFLICT  /home/user/.config/file: already exists")
}

func TestRenderPlan_CreateDirOps(t *testing.T) {
	p := &plan.Plan{
		PackageName: "test",
		Profile:     "default",
		Ops: []plan.Op{
			{Kind: plan.OpCreate, Source: "src/dir", Target: "/home/user/.config/dir", IsDir: true},
		},
	}

	var buf bytes.Buffer
	RenderPlan(&buf, p)
	output := buf.String()

	assert.Contains(t, output, "CREATE-DIR")
	assert.Contains(t, output, "/home/user/.config/dir")
	assert.Contains(t, output, "src/dir")
}

func TestRenderPlan_RemoveDirOps(t *testing.T) {
	p := &plan.Plan{
		PackageName: "test",
		Ops: []plan.Op{
			{Kind: plan.OpRemove, Target: "/home/user/.config/dir", IsDir: true},
		},
	}

	var buf bytes.Buffer
	RenderPlan(&buf, p)
	output := buf.String()

	assert.Contains(t, output, "REMOVE-DIR")
	assert.Contains(t, output, "/home/user/.config/dir")
	assert.Contains(t, output, "Total: 1 link(s) to remove.")
}

func TestRenderPlan_PlainMode(t *testing.T) {
	style.SetPlain(true)
	t.Cleanup(func() { style.SetPlain(false) })

	p := &plan.Plan{
		PackageName: "test",
		Profile:     "default",
		Ops: []plan.Op{
			{Kind: plan.OpCreate, Source: "src/file1", Target: "/home/user/.config/file1"},
		},
	}

	var buf bytes.Buffer
	RenderPlan(&buf, p)
	output := buf.String()

	assert.Contains(t, output, "->")
	assert.NotContains(t, output, "→")
}
