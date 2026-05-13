// Package style holds a process-wide toggle for plain-text (ASCII-only) output.
// Set once at startup from cli/root.go and read by renderers in internal/prompt
// and cli/status.go. NOT thread-safe; written exactly once before any reads.
package style

var plain bool

// SetPlain enables or disables plain (ASCII-only) output mode.
func SetPlain(v bool) { plain = v }

// Plain reports whether plain output mode is active.
func Plain() bool { return plain }
