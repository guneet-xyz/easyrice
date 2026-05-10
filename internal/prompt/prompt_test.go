package prompt

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/guneet-xyz/easyrice/internal/plan"
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

	assert.Contains(t, output, "Plan: install test (profile: default)")
	assert.Contains(t, output, "Total: 0 symlinks to create.")
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

	assert.Contains(t, output, "Plan: install test (profile: default)")
	assert.Contains(t, output, "CREATE")
	assert.Contains(t, output, "/home/user/.config/file1")
	assert.Contains(t, output, "src/file1")
	assert.Contains(t, output, "/home/user/.config/file2")
	assert.Contains(t, output, "src/file2")
	assert.Contains(t, output, "Total: 2 symlinks to create.")
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
	assert.Contains(t, output, "Total: 2 symlinks to remove.")
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

	assert.Contains(t, output, "Total: 100 symlinks to create.")
	count := strings.Count(output, "CREATE")
	assert.Equal(t, 100, count)
}

func TestRenderSwitchPlan(t *testing.T) {
	uninstall := &plan.Plan{
		PackageName: "test",
		Ops: []plan.Op{
			{Kind: plan.OpRemove, Target: "/home/user/.config/old"},
		},
	}

	install := &plan.Plan{
		PackageName: "test",
		Profile:     "new",
		Ops: []plan.Op{
			{Kind: plan.OpCreate, Source: "src/new", Target: "/home/user/.config/new"},
		},
	}

	var buf bytes.Buffer
	RenderSwitchPlan(&buf, uninstall, install)
	output := buf.String()

	assert.Contains(t, output, "Plan: uninstall test")
	assert.Contains(t, output, "REMOVE")
	assert.Contains(t, output, "/home/user/.config/old")
	assert.Contains(t, output, "Plan: install test (profile: new)")
	assert.Contains(t, output, "CREATE")
	assert.Contains(t, output, "/home/user/.config/new")
	assert.Contains(t, output, "Total: 2 symlinks (1 remove, 1 create).")
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
