package log

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	charmlog "github.com/charmbracelet/log"
)

// TraceLevel is a custom level below DebugLevel for internal diagnostics.
const TraceLevel = charmlog.Level(-8)

var logger *charmlog.Logger

// Init initializes the global logger based on verbosity count.
//
//	0 = info (default)
//	1 = debug (-v)
//	2+ = trace (-vv)
func Init(verbosity int) {
	level := charmlog.InfoLevel
	switch {
	case verbosity >= 2:
		level = TraceLevel
	case verbosity == 1:
		level = charmlog.DebugLevel
	}

	logger = charmlog.NewWithOptions(os.Stderr, charmlog.Options{
		Level:           level,
		ReportTimestamp: false,
	})

	styles := charmlog.DefaultStyles()

	// Register custom TRACE level style
	styles.Levels[TraceLevel] = styles.Levels[charmlog.DebugLevel].Copy().SetString("TRAC")

	// At INFO verbosity, hide the INFO prefix for clean user-facing output.
	// At DEBUG/TRACE verbosity, show it so levels are distinguishable.
	if verbosity == 0 {
		styles.Levels[charmlog.InfoLevel] = lipgloss.NewStyle().SetString("")
	}

	logger.SetStyles(styles)
}

// Get returns the global logger instance.
func Get() *charmlog.Logger {
	if logger == nil {
		Init(0)
	}
	return logger
}

// IsDebug returns true if verbosity is debug or higher.
func IsDebug() bool {
	return Get().GetLevel() <= charmlog.DebugLevel
}

// IsTrace returns true if verbosity is trace.
func IsTrace() bool {
	return Get().GetLevel() <= TraceLevel
}
