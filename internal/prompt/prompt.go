package prompt

import (
	"bufio"
	"fmt"
	"io"
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
