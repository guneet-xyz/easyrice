package prompt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/guneet-xyz/easyrice/internal/deps"
	"github.com/guneet-xyz/easyrice/internal/plan"
	"github.com/guneet-xyz/easyrice/internal/style"
)

// ErrUserDeclined is returned when a user declines a confirmation prompt.
var ErrUserDeclined = errors.New("user declined")

// RenderPlan writes the human-readable plan to w.
// Format for install:
//
//	Plan: install <pkg> (profile: <name>)
//	  CREATE  <target>  →  <source>
//	  ...
//	Total: N symlinks to create.
//
// Format for uninstall:
//
//	Plan: uninstall <pkg>
//	  REMOVE  <target>
//	  ...
//	Total: N symlinks to remove.
//
// NEVER truncates — prints every op.
func RenderPlan(w io.Writer, p *plan.Plan) {
	if p == nil {
		return
	}

	// Determine operation type from first op (if any)
	var opType string
	if len(p.Ops) > 0 {
		if p.Ops[0].Kind == plan.OpCreate {
			opType = "install"
		} else {
			opType = "uninstall"
		}
	} else {
		opType = "install" // default
	}

	if opType == "install" {
		fmt.Fprintf(w, "Plan: install %s using profile %s\n", p.PackageName, p.Profile)
	} else {
		fmt.Fprintf(w, "Plan: uninstall %s\n", p.PackageName)
	}

	// Operations table
	if len(p.Ops) > 0 {
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		arrow := "→"
		if style.Plain() {
			arrow = "->"
		}
		for _, op := range p.Ops {
			if op.Kind == plan.OpCreate {
				label := "CREATE"
				if op.IsDir {
					label = "CREATE-DIR"
				}
				fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n", label, op.Target, arrow, op.Source)
			} else {
				label := "REMOVE"
				if op.IsDir {
					label = "REMOVE-DIR"
				}
				fmt.Fprintf(tw, "  %s\t%s\n", label, op.Target)
			}
		}
		tw.Flush()
	}

	// Total line
	count := len(p.Ops)
	if opType == "install" {
		fmt.Fprintf(w, "Total: %d link(s) to create.\n", count)
	} else {
		fmt.Fprintf(w, "Total: %d link(s) to remove.\n", count)
	}

	// Conflicts (if any)
	if len(p.Conflicts) > 0 {
		fmt.Fprintf(w, "\nConflicts (%d):\n", len(p.Conflicts))
		RenderConflicts(w, p.Conflicts)
	}
}

// RenderConflicts writes conflict lines to w.
//
//	CONFLICT  <target>: <reason>
func RenderConflicts(w io.Writer, conflicts []plan.Conflict) {
	for _, c := range conflicts {
		reason := c.Reason
		if c.IsDir {
			reason = reason + " (directory)"
		}
		fmt.Fprintf(w, "CONFLICT  %s: %s\n", c.Target, reason)
	}
}

// Confirm writes "<message> [y/N]: " to out, reads one line from in.
// Returns true ONLY if input is "y" or "yes" (case-insensitive, trimmed).
// Returns false for empty input (bare Enter), "n", "no", or anything else.
// Returns (false, nil) on EOF.
func Confirm(in io.Reader, out io.Writer, message string) (bool, error) {
	fmt.Fprintf(out, "%s [y/N]: ", message)

	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err == io.EOF {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	line = strings.TrimSpace(line)
	line = strings.ToLower(line)

	return line == "y" || line == "yes", nil
}

// SelectOption is one choice in a Select prompt.
type SelectOption struct {
	Label       string
	Description string // shown as secondary info; may be empty
}

// Select writes a numbered menu to out, reads a 1-based choice from in.
// Returns the 0-based index of the chosen option.
// On empty input, returns defaultIdx.
// Re-prompts up to 3 times on out-of-range input; returns error after 3 failures.
// Returns error on EOF when autoAccept is false.
// Signature mirrors Confirm(in io.Reader, out io.Writer, ...) exactly.
func Select(in io.Reader, out io.Writer, label string, options []SelectOption, defaultIdx int) (int, error) {
	return SelectWithDefault(in, out, label, options, defaultIdx, false)
}

// SelectWithDefault is like Select but when autoAccept is true, returns defaultIdx
// immediately without reading from in (in may be nil).
func SelectWithDefault(in io.Reader, out io.Writer, label string, options []SelectOption, defaultIdx int, autoAccept bool) (int, error) {
	if autoAccept {
		// Print "Using: <label>" so user knows what's running
		if defaultIdx >= 0 && defaultIdx < len(options) {
			fmt.Fprintf(out, "Using default: %s\n", options[defaultIdx].Label)
		}
		return defaultIdx, nil
	}

	// Print menu
	fmt.Fprintf(out, "%s\n", label)
	sep := "—"
	if style.Plain() {
		sep = "--"
	}
	for i, opt := range options {
		if opt.Description != "" {
			fmt.Fprintf(out, "  %d) %s %s %s\n", i+1, opt.Label, sep, opt.Description)
		} else {
			fmt.Fprintf(out, "  %d) %s\n", i+1, opt.Label)
		}
	}
	fmt.Fprintf(out, "Choose an option [default: %d]: ", defaultIdx+1)

	reader := bufio.NewReader(in)
	const maxRetries = 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			return 0, fmt.Errorf("prompt: EOF before selection")
		}
		if err != nil {
			return 0, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return defaultIdx, nil
		}
		n, parseErr := strconv.Atoi(line)
		if parseErr == nil && n >= 1 && n <= len(options) {
			return n - 1, nil
		}
		if attempt < maxRetries-1 {
			fmt.Fprintf(out, "Invalid choice %q. Choose 1-%d [default: %d]: ", line, len(options), defaultIdx+1)
		}
	}
	return 0, fmt.Errorf("prompt: too many invalid choices")
}

// shorten truncates s to n characters, appending "..." if longer.
func shorten(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// RenderDepReport writes a human-readable dependency check report to w.
// Format:
//
//	Dependency check:
//	  ✓ ripgrep (14.1.0) — installed
//	  ✗ neovim — missing
//	  ! node (18.20.0) — version mismatch, need >=20
//	  ? mdformat — installed (version unknown)
func RenderDepReport(w io.Writer, report deps.DepReport) {
	fmt.Fprintf(w, "Dependency check:\n")
	for _, entry := range report.Entries {
		var glyph string
		var status string

		switch entry.Status {
		case deps.DepOK:
			glyph = "✓"
			status = "installed"
		case deps.DepMissing:
			glyph = "✗"
			status = "missing"
		case deps.DepVersionMismatch:
			glyph = "!"
			status = fmt.Sprintf("version mismatch; required %s", entry.Dep.Version)
		case deps.DepProbeUnknownVersion:
			glyph = "?"
			status = "installed (version unknown)"
		default:
			glyph = "?"
			status = "unknown"
		}

		// Apply plain-mode glyph substitution
		if style.Plain() {
			switch glyph {
			case "✓":
				glyph = "OK"
			case "✗":
				glyph = "FAIL"
			case "!":
				glyph = "WARN"
				// "?" stays "?"
			}
		}

		// Format version info
		var versionStr string
		if entry.InstalledVersion != "" {
			versionStr = fmt.Sprintf(" (%s)", entry.InstalledVersion)
		}

		// Compute separator
		sep := "—"
		if style.Plain() {
			sep = "--"
		}

		fmt.Fprintf(w, "  %s %s%s %s %s\n", glyph, entry.Dep.Name, versionStr, sep, status)
	}
}

// SelectInstallMethod prompts the user to choose an install method for a dependency.
// If autoAccept is true, selects the first method without prompting (but still prints "Using: ...").
// For custom methods (ShellPayload != ""), prompts for confirmation before returning.
// Returns ErrUserDeclined if user declines the custom method confirmation.
// Returns an error if no methods are available.
func SelectInstallMethod(in io.Reader, out io.Writer, entry deps.DepReportEntry, autoAccept bool) (deps.InstallMethod, error) {
	if len(entry.Methods) == 0 {
		return deps.InstallMethod{}, fmt.Errorf("no install methods available for %q", entry.Dep.Name)
	}

	// Build options for Select
	options := make([]SelectOption, len(entry.Methods))
	for i, m := range entry.Methods {
		options[i].Label = m.Description
		if m.Command != nil {
			options[i].Description = strings.Join(m.Command, " ")
		} else if m.ShellPayload != "" {
			options[i].Description = shorten(m.ShellPayload, 60)
		}
	}

	// Select method
	idx, err := SelectWithDefault(in, out, fmt.Sprintf("Choose how to install %s:", entry.Dep.Name), options, 0, autoAccept)
	if err != nil {
		return deps.InstallMethod{}, err
	}

	selected := entry.Methods[idx]

	// For custom methods, prompt for confirmation
	if selected.ShellPayload != "" {
		ok, err := Confirm(in, out, fmt.Sprintf("Run this command from rice.toml? %s", selected.ShellPayload))
		if err != nil {
			return deps.InstallMethod{}, err
		}
		if !ok {
			return deps.InstallMethod{}, ErrUserDeclined
		}
	}

	return selected, nil
}
