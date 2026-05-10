package installer

import (
	"fmt"
	"os"
	"github.com/guneet/rice/internal/symlink"
)

// Conflict describes a symlink target that cannot be created.
type Conflict struct {
	Target string // the path that would be the symlink
	Source string // the intended symlink source
	Reason string // human-readable reason
}

func (c Conflict) Error() string {
	return fmt.Sprintf("conflict at %s: %s", c.Target, c.Reason)
}

// PlannedLink is a (source, target) pair to be created as a symlink.
type PlannedLink struct {
	Source string
	Target string
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
			})
			continue
		}

		// Target exists. Check if it's a symlink.
		if fi.Mode()&os.ModeSymlink == 0 {
			// Not a symlink — it's a regular file or directory
			if fi.IsDir() {
				conflicts = append(conflicts, Conflict{
					Target: link.Target,
					Source: link.Source,
					Reason: "existing directory",
				})
			} else {
				conflicts = append(conflicts, Conflict{
					Target: link.Target,
					Source: link.Source,
					Reason: "existing file",
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
			})
			continue
		}

		conflicts = append(conflicts, Conflict{
			Target: link.Target,
			Source: link.Source,
			Reason: fmt.Sprintf("symlink points to %s", otherDest),
		})
	}

	return conflicts
}
