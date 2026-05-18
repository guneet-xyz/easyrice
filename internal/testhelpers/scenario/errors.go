package scenario

import "errors"

var (
	// ErrScenarioNotFound is returned when a scenario directory cannot be found.
	ErrScenarioNotFound = errors.New("scenario not found")
	// ErrInvalidYAML is returned when scenario YAML parsing fails.
	ErrInvalidYAML = errors.New("invalid scenario YAML")
	// ErrSnapshotMismatch is returned when a snapshot comparison fails.
	ErrSnapshotMismatch = errors.New("snapshot mismatch")
	// ErrMissingExpected is returned when an expected snapshot file is missing.
	ErrMissingExpected = errors.New("missing expected snapshot file")
)
