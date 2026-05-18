package scenario

import "errors"

var (
	// ErrInvalidYAML is returned when scenario YAML parsing fails.
	ErrInvalidYAML = errors.New("invalid scenario YAML")
	// ErrMissingExpected is returned when an expected snapshot file is missing.
	ErrMissingExpected = errors.New("missing expected snapshot file")
)
