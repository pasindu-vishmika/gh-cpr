package utils

import (
	"os"
	"path/filepath"
)

// IsGitRepository reports whether the current working directory is inside a Git
// repository. Like Git itself, it walks up the directory tree looking for a
// ".git" entry, so CPR can be run from any subdirectory of the repo, not just
// its root.
func IsGitRepository() bool {
	dir, err := os.Getwd()
	if err != nil {
		return false
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the filesystem root without finding ".git".
			return false
		}
		dir = parent
	}
}
