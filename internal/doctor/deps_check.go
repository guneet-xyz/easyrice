package doctor

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/guneet-xyz/easyrice/internal/deps"
	"github.com/guneet-xyz/easyrice/internal/manifest"
)

// CheckDeclaredDeps checks all declared dependencies in the manifest and reports
// missing or mismatched versions. It is an informational check that does not fail
// the doctor; missing/mismatched deps are a normal pre-install state.
//
// For each package with dependencies, it runs deps.Check and reports:
//   - [WARN] for DepMissing or DepVersionMismatch (increments warnings count)
//   - [INFO] for DepProbeUnknownVersion (does not increment count)
//   - [WARN] for check errors (increments warnings count)
//
// Returns the total warning count.
func CheckDeclaredDeps(ctx context.Context, w io.Writer, runner deps.Runner, m manifest.Manifest) int {
	warnings := 0

	// Sort package names for stable output
	var pkgNames []string
	for name := range m.Packages {
		pkgNames = append(pkgNames, name)
	}
	sort.Strings(pkgNames)

	for _, pkgName := range pkgNames {
		pkg := m.Packages[pkgName]

		// Skip packages with no dependencies
		if len(pkg.Dependencies) == 0 {
			continue
		}

		// Run the dependency check
		report, err := deps.Check(ctx, runner, pkg.Dependencies, m.CustomDependencies, deps.Detect())
		if err != nil {
			fmt.Fprintf(w, "[WARN] %s: dep check failed: %v\n", pkgName, err)
			warnings++
			continue
		}

		// Report each entry
		for _, entry := range report.Entries {
			depName := entry.Dep.Name
			switch entry.Status {
			case deps.DepMissing:
				fmt.Fprintf(w, "[WARN] %s.%s — missing (installed=%v constraint=%s)\n",
					pkgName, depName, entry.Installed, entry.Dep.Version)
				warnings++
			case deps.DepVersionMismatch:
				fmt.Fprintf(w, "[WARN] %s.%s — version mismatch (installed=%s constraint=%s)\n",
					pkgName, depName, entry.InstalledVersion, entry.Dep.Version)
				warnings++
			case deps.DepProbeUnknownVersion:
				fmt.Fprintf(w, "[INFO] %s.%s — installed (version unknown)\n",
					pkgName, depName)
			case deps.DepOK:
				// No output for OK status
			}
		}
	}

	return warnings
}
