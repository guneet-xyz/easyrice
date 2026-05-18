// Package scenario provides a YAML-driven multi-step CLI scenario runner for
// integration tests. It is a leaf package and MUST NOT import cli/.
//
// Each scenario lives in a directory containing:
//   - steps.yaml: ordered step definitions
//   - repo/: initial managed-repo tree
//   - home/: initial home-dir tree
//   - expected/step-N/: per-step snapshot files
//
// Consuming test packages supply a Runner callback that wraps their CLI entry
// point (e.g. cobra's rootCmd.Execute), keeping this package cobra-agnostic.
package scenario
