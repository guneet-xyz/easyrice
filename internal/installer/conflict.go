package installer

import (
	"fmt"
	"github.com/guneet-xyz/easyrice/internal/symlink"
	"os"
)

// Conflict describes a symlink target that cannot be created.
type Conflict struct {
	Target string // the path that would be the symlink
	Source string // the intended symlink source
	Reason string // human-readable reason
	IsDir  bool   // true if the planned link is a directory symlink (folder-mode)
}

func (c Conflict) Error() string {
	return fmt.Sprintf("conflict at %s: %s", c.Target, c.Reason)
}

// PlannedLink is a (source, target) pair to be created as a symlink.
// IsDir indicates folder-mode (directory symlink); zero value (false) is file-mode.
type PlannedLink struct {
	Source string
	Target string
	IsDir  bool
}

// DetectConflicts checks each planned link for conflicts.
// A conflict exists if:
//   - target exists and is NOT a symlink → "existing file/directory"
//   - target exists and IS a symlink but does NOT point to source → "symlink points to <other>"
//   - target exists and IS a symlink pointing to source → NOT a conflict (idempotent)
//
// ignoreTargets is a set of target paths to skip (used by switch pre-flight to exclude old links).
// Returns the list of conflicts found (empty slice = no conflicts).
func DetectConflicts(planned []PlannedLink, ignoreTargets map[string]struct{}) []Conflict {
	var conflicts []Conflict

	for _, link := range planned {
		// Skip if target is in ignoreTargets
		if _, shouldIgnore := ignoreTargets[link.Target]; shouldIgnore {
			continue
		}

		// Check if target exists
		fi, err := os.Lstat(link.Target)
		if err != nil {
			if os.IsNotExist(err) {
				// Target doesn't exist, no conflict
				continue
			}
			// Some other error (permission denied, etc.) — treat as conflict
			conflicts = append(conflicts, Conflict{
				Target: link.Target,
				Source: link.Source,
				Reason: fmt.Sprintf("failed to check target: %v", err),
				IsDir:  link.IsDir,
			})
			continue
		}

		// Target exists. Check if it's a symlink.
		if fi.Mode()&os.ModeSymlink == 0 {
			// Not a symlink — it's a regular file or directory
			if fi.IsDir() {
				reason := "existing directory"
				if link.IsDir {
					reason = "existing directory (folder-mode requires symlink or absent path)"
				}
				conflicts = append(conflicts, Conflict{
					Target: link.Target,
					Source: link.Source,
					Reason: reason,
					IsDir:  link.IsDir,
				})
			} else {
				conflicts = append(conflicts, Conflict{
					Target: link.Target,
					Source: link.Source,
					Reason: "existing file",
					IsDir:  link.IsDir,
				})
			}
			continue
		}

		// Target is a symlink. Check if it points to our source.
		isOurs, err := symlink.IsSymlinkTo(link.Target, link.Source)
		if err != nil {
			// Error reading symlink — treat as conflict
			conflicts = append(conflicts, Conflict{
				Target: link.Target,
				Source: link.Source,
				Reason: fmt.Sprintf("failed to read symlink: %v", err),
				IsDir:  link.IsDir,
			})
			continue
		}

		if isOurs {
			// Symlink already points to our source — idempotent, no conflict
			continue
		}

		// Symlink points to something else
		otherDest, err := os.Readlink(link.Target)
		if err != nil {
			conflicts = append(conflicts, Conflict{
				Target: link.Target,
				Source: link.Source,
				Reason: fmt.Sprintf("failed to read symlink: %v", err),
				IsDir:  link.IsDir,
			})
			continue
		}

		conflicts = append(conflicts, Conflict{
			Target: link.Target,
			Source: link.Source,
			Reason: fmt.Sprintf("symlink points to %s", otherDest),
			IsDir:  link.IsDir,
		})
	}

	return conflicts
}
