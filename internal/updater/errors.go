package updater

import "errors"

var (
	ErrDevBuild      = errors.New("updater: cannot check for updates in dev build")
	ErrAlreadyLatest = errors.New("updater: already running the latest version")
	ErrLockBusy      = errors.New("updater: another update is in progress")
	ErrNoChecksum    = errors.New("updater: release has no checksums.txt")
	ErrCacheCorrupt  = errors.New("updater: cache file is corrupted")
	ErrInvalidSemver = errors.New("updater: invalid semantic version")
)
