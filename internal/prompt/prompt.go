package prompt

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/guneet-xyz/easyrice/internal/plan"
)

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

	// Header
	if opType == "install" {
		fmt.Fprintf(w, "Plan: install %s (profile: %s)\n", p.PackageName, p.Profile)
	} else {
		fmt.Fprintf(w, "Plan: uninstall %s\n", p.PackageName)
	}

	// Operations table
	if len(p.Ops) > 0 {
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for _, op := range p.Ops {
			if op.Kind == plan.OpCreate {
				label := "CREATE"
				if op.IsDir {
					label = "CREATE-DIR"
				}
				fmt.Fprintf(tw, "  %s\t%s\t→\t%s\n", label, op.Target, op.Source)
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
		fmt.Fprintf(w, "Total: %d symlinks to create.\n", count)
	} else {
		fmt.Fprintf(w, "Total: %d symlinks to remove.\n", count)
	}

	// Conflicts (if any)
	if len(p.Conflicts) > 0 {
		fmt.Fprintf(w, "\nConflicts (%d):\n", len(p.Conflicts))
		RenderConflicts(w, p.Conflicts)
	}
}

// RenderSwitchPlan writes the combined switch plan (uninstall + install phases).
func RenderSwitchPlan(w io.Writer, uninstall *plan.Plan, install *plan.Plan) {
	if uninstall == nil || install == nil {
		return
	}

	// Uninstall phase
	fmt.Fprintf(w, "Plan: uninstall %s\n", uninstall.PackageName)
	if len(uninstall.Ops) > 0 {
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for _, op := range uninstall.Ops {
			label := "REMOVE"
			if op.IsDir {
				label = "REMOVE-DIR"
			}
			fmt.Fprintf(tw, "  %s\t%s\n", label, op.Target)
		}
		tw.Flush()
	}

	// Install phase
	fmt.Fprintf(w, "Plan: install %s (profile: %s)\n", install.PackageName, install.Profile)
	if len(install.Ops) > 0 {
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for _, op := range install.Ops {
			label := "CREATE"
			if op.IsDir {
				label = "CREATE-DIR"
			}
			fmt.Fprintf(tw, "  %s\t%s\t→\t%s\n", label, op.Target, op.Source)
		}
		tw.Flush()
	}

	// Combined total
	totalOps := len(uninstall.Ops) + len(install.Ops)
	fmt.Fprintf(w, "Total: %d symlinks (%d remove, %d create).\n", totalOps, len(uninstall.Ops), len(install.Ops))

	// Conflicts from install phase (if any)
	if len(install.Conflicts) > 0 {
		fmt.Fprintf(w, "\nConflicts (%d):\n", len(install.Conflicts))
		RenderConflicts(w, install.Conflicts)
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
			fmt.Fprintf(out, "Using: %s\n", options[defaultIdx].Label)
		}
		return defaultIdx, nil
	}

	// Print menu
	fmt.Fprintf(out, "%s\n", label)
	for i, opt := range options {
		if opt.Description != "" {
			fmt.Fprintf(out, "  %d) %s — %s\n", i+1, opt.Label, opt.Description)
		} else {
			fmt.Fprintf(out, "  %d) %s\n", i+1, opt.Label)
		}
	}
	fmt.Fprintf(out, "Enter choice [default: %d]: ", defaultIdx+1)

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
			fmt.Fprintf(out, "Invalid choice %q. Enter 1-%d [default: %d]: ", line, len(options), defaultIdx+1)
		}
	}
	return 0, fmt.Errorf("prompt: too many invalid inputs")
}
