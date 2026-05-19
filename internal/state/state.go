package state

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/guneet-xyz/easyrice/internal/deps"
	"github.com/guneet-xyz/easyrice/internal/xdgpath"
)

// InstalledLink represents a single symlink installed by rice.
type InstalledLink struct {
	Source string `json:"source"`           // absolute path to the file in the rice repo
	Target string `json:"target"`           // absolute path to the symlink in $HOME
	IsDir  bool   `json:"is_dir,omitempty"` // true if the source is a directory
}

// PackageState represents the state of a single installed package.
type PackageState struct {
	Profile               string                     `json:"profile"`
	InstalledLinks        []InstalledLink            `json:"installed_links"`
	InstalledAt           time.Time                  `json:"installed_at"`
	InstalledDependencies []deps.InstalledDependency `json:"installed_dependencies,omitempty"`
}

// State is the top-level state file structure.
// Key = package name (e.g., "nvim", "ghostty")
type State map[string]PackageState

// DefaultPath returns the platform-appropriate state file path.
// POSIX: ~/.config/easyrice/state.json
// Windows: %APPDATA%/easyrice/state.json
func DefaultPath() string {
	return filepath.Join(xdgpath.ConfigDir(), "easyrice", "state.json")
}

// Load reads and parses the state file at path.
// If the file does not exist, returns an empty State (not an error).
// Errors are wrapped with the file path and include actionable guidance
// (the file may need to be repaired from a backup).
func Load(path string) (State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{}, nil
		}
		return nil, fmt.Errorf("read state file %s: %w", path, err)
	}

	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) {
		return nil, fmt.Errorf("state file %s is null; the file may need to be repaired from a backup or removed to recover", path)
	}

	if len(trimmed) > 0 && trimmed[0] != '{' {
		return nil, fmt.Errorf("state file %s must be a JSON object/map at the top level; got %q; the file may need to be repaired from a backup", path, firstByte(trimmed))
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || strings.Contains(err.Error(), "unexpected end of JSON input") {
			return nil, fmt.Errorf("state file %s appears truncated or partially written: %w; the file may need to be repaired from a backup or recovered from a previous version", path, err)
		}
		return nil, fmt.Errorf("parse state file %s: %w; the file may need to be repaired from a backup", path, err)
	}

	return s, nil
}

func firstByte(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return string(b[:1])
}

var (
	stateOpenFile  = os.OpenFile
	stateRename    = os.Rename
	stateWriteFile = atomicWriteFile
)

// atomicWriteFile writes data to name via a sibling tempfile + rename, so a
// concurrent reader never observes a partial write. Failures leave the
// destination untouched.
func atomicWriteFile(name string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(name)
	tmp, err := os.CreateTemp(dir, ".state.json.tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, name); err != nil {
		cleanup()
		return err
	}
	return nil
}

var saveMu sync.Mutex

var (
	pathLocksMu sync.Mutex
	pathLocks   = map[string]*sync.Mutex{}
)

// Lock acquires a process-wide mutex keyed by absolute path so a caller can
// safely run a Load + mutate + Save cycle without races against another
// goroutine performing the same cycle on the same state file. The returned
// function releases the lock.
func Lock(path string) func() {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	pathLocksMu.Lock()
	mu, ok := pathLocks[abs]
	if !ok {
		mu = &sync.Mutex{}
		pathLocks[abs] = mu
	}
	pathLocksMu.Unlock()
	mu.Lock()
	return mu.Unlock
}

// Save writes the state to path as pretty-printed JSON via an atomic
// temp-file + rename, so concurrent readers see the old bytes or the new
// bytes but never a torn write. A process-level mutex serialises Save calls
// so callers performing read-modify-write cycles via Load+Save do not need
// their own locking for write ordering.
func Save(path string, s State) error {
	saveMu.Lock()
	defer saveMu.Unlock()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create state directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	var prevBytes []byte
	prevExisted := false
	if existing, rerr := os.ReadFile(path); rerr == nil {
		prevBytes = existing
		prevExisted = true
	} else if !errors.Is(rerr, os.ErrNotExist) {
		return fmt.Errorf("read existing state file %s: %w", path, classifyFSError(rerr))
	}

	probe, err := stateOpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("open state file %s: %w", path, classifyFSError(err))
	}
	_ = probe.Close()
	if !prevExisted {
		_ = os.Remove(path)
	}

	if err := stateWriteFile(path, data, 0644); err != nil {
		if prevExisted {
			_ = os.WriteFile(path, prevBytes, 0644)
		} else {
			_ = os.Remove(path)
		}
		return fmt.Errorf("write state file %s: %w", path, classifyFSError(err))
	}

	_ = stateRename

	return nil
}

func classifyFSError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, os.ErrPermission) {
		return fmt.Errorf("permission denied: %w", err)
	}
	msg := err.Error()
	if strings.Contains(msg, "no space") || strings.Contains(strings.ToLower(msg), "enospc") {
		return fmt.Errorf("no space left on device: %w", err)
	}
	return err
}
