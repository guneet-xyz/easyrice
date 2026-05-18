//go:build !windows

package symlink

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSymlink_Edges pins edge-case behavior of the symlink package per
// AGENTS.md "NOTES / GOTCHAS" + symlink.go docstrings + first-principles FS safety.
// Spec source: .omo/plans/better-tests.md lines 1586-1588.
func TestSymlink_Edges(t *testing.T) {
	t.Run("BUG-128-ReadLinkNonSymlink", func(t *testing.T) {
		t.Log("BUG-128")
		// Spec: ReadLink on a non-symlink (regular file) must return a clear error.
		tmp := t.TempDir()
		regular := filepath.Join(tmp, "regular.txt")
		require.NoError(t, os.WriteFile(regular, []byte("not a symlink"), 0o644), "BUG-128: setup failed")

		got, err := ReadLink(regular)
		require.Error(t, err, "BUG-128: ReadLink must error on a regular file, got=%q", got)
		assert.NotEmpty(t, err.Error(), "BUG-128: error message must not be empty")
		assert.Empty(t, got, "BUG-128: ReadLink on non-symlink must return empty string on error")
	})

	t.Run("BUG-129-IsSymlinkToBrokenSymlink", func(t *testing.T) {
		t.Log("BUG-129")
		// Spec: a function named IsSymlinkTo(path, target) must answer the LITERAL
		// question — is path a symlink whose stored target equals the queried target?
		// Must use os.Readlink semantics, NOT os.Stat (which follows the link).
		tmp := t.TempDir()
		link := filepath.Join(tmp, "broken_link")
		// Pointee deliberately does not exist.
		stored := filepath.Join(tmp, "does_not_exist", "target.txt")
		require.NoError(t, os.Symlink(stored, link), "BUG-129: setup failed")

		// Sanity: pointee must NOT exist (otherwise this isn't a broken symlink).
		_, statErr := os.Stat(stored)
		require.True(t, os.IsNotExist(statErr), "BUG-129: stored target must be missing for broken-symlink scenario, got %v", statErr)

		ok, err := IsSymlinkTo(link, stored)
		require.NoError(t, err, "BUG-129: IsSymlinkTo must not error on a broken symlink whose stored target matches")
		assert.True(t, ok, "BUG-129: IsSymlinkTo must return true for broken symlink whose stored target equals query (uses Readlink, not Stat)")
	})

	t.Run("BUG-130-ConcurrentCreateRemove", func(t *testing.T) {
		t.Log("BUG-130")
		// Spec: spawning N goroutines that race Create vs Remove on the same target
		// must end in a deterministic final state — either present-with-correct-target
		// or absent. No torn intermediate state (e.g., a regular file, or a symlink
		// to garbage). Note: low-level test; installer-level concurrency is Task 11.
		tmp := t.TempDir()
		target := filepath.Join(tmp, "concurrent_link")
		source := filepath.Join(tmp, "source_file")
		require.NoError(t, os.WriteFile(source, []byte("source"), 0o644), "BUG-130: setup failed")

		const goroutines = 50
		var wg sync.WaitGroup
		var creates, removes int64
		// Seed: start with the symlink present so Remove has something to remove
		// half the time. CreateSymlink itself is strict and fails on existing
		// target — we accept errors and only check the FINAL state.
		_ = CreateSymlink(source, target)

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			even := i%2 == 0
			go func(create bool) {
				defer wg.Done()
				if create {
					if err := CreateSymlink(source, target); err == nil {
						atomic.AddInt64(&creates, 1)
					}
				} else {
					if err := RemoveSymlink(target); err == nil {
						atomic.AddInt64(&removes, 1)
					}
				}
			}(even)
		}
		wg.Wait()

		// Deterministic final state: either the file does not exist, or it is a
		// symlink to exactly `source`. ANY other state is torn.
		fi, err := os.Lstat(target)
		if err != nil {
			assert.True(t, os.IsNotExist(err), "BUG-130: torn state — Lstat returned non-NotExist error: %v (creates=%d removes=%d)", err, creates, removes)
			return
		}
		assert.NotZero(t, fi.Mode()&os.ModeSymlink, "BUG-130: torn state — target exists but is NOT a symlink (creates=%d removes=%d)", creates, removes)
		dest, rlErr := os.Readlink(target)
		require.NoError(t, rlErr, "BUG-130: Readlink failed on final state")
		assert.Equal(t, source, dest, "BUG-130: torn state — symlink points to %q, expected %q (creates=%d removes=%d)", dest, source, creates, removes)
	})
}
