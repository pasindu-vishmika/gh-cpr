package utils

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// safeBranchName accepts only conservative Git branch names: it must start with
// an alphanumeric character (which also rejects a leading '-', the vector for
// argument/option injection into git) and otherwise contain only letters,
// digits, and the separators . _ / -.
var safeBranchName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*$`)

// ValidateBranchName rejects names that are empty, malformed, or that Git would
// reject as a ref — and, importantly, anything that could be interpreted as a
// command-line option. Callers must validate before passing a name to git.
func ValidateBranchName(name string) error {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return fmt.Errorf("branch name cannot be empty")
	case len(name) > 255:
		return fmt.Errorf("branch name is too long")
	case !safeBranchName.MatchString(name):
		return fmt.Errorf("invalid branch name %q: use letters, digits, and . _ / - (must not start with '-')", name)
	case strings.Contains(name, ".."), strings.Contains(name, "//"):
		return fmt.Errorf("invalid branch name %q: cannot contain '..' or '//'", name)
	case strings.HasSuffix(name, "/"), strings.HasSuffix(name, ".lock"):
		return fmt.Errorf("invalid branch name %q: cannot end with '/' or '.lock'", name)
	}
	return nil
}

func CreateBranchFromMain(newBranchName string) error {
	newBranchName = strings.TrimSpace(newBranchName)
	// Validate up front: the name is user-supplied free text and is passed as a
	// positional argument to git, so it must not be able to pose as an option.
	if err := ValidateBranchName(newBranchName); err != nil {
		return err
	}

	// This function creates a branch based on the updated origin/main without checking out main locally.
	cmds := [][]string{
		{"git", "fetch", "origin"},
		{"git", "branch", newBranchName, "origin/main"},
		{"git", "push", "-u", "origin", newBranchName},
	}

	for _, c := range cmds {
		// #nosec G204 -- no shell is invoked (argv form); the only variable
		// argument is newBranchName, validated by ValidateBranchName above.
		cmd := exec.Command(c[0], c[1:]...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("command failed: %s\nError: %v", string(out), err)
		}
	}

	return nil
}
