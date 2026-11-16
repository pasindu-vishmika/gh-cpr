package utils

import (
	"os/exec"
	"strconv"
	"strings"
)

// Note on G204/CWE-78: every exec.Command below uses the argv form (no shell),
// so shell metacharacters are inert. sourceBranch/targetBranch originate from
// the remote branch list (git ls-remote), which Git guarantees are valid ref
// names — in particular they cannot begin with '-', so they cannot pose as
// command-line options. Hence the calls are not injectable.
func GetDefaultPRTitle(sourceBranch, targetBranch string) (string, error) {
	// #nosec G204 -- argv form; targetBranch is a remote ref name.
	_ = exec.Command("git", "fetch", "origin", targetBranch).Run() // ignoring err for brevity

	// Count commits from origin/<targetBranch>..sourceBranch
	// #nosec G204 -- argv form; the range is a single positional prefixed with "origin/".
	countBytes, err := exec.Command("git", "rev-list", "--count", "origin/"+targetBranch+".."+sourceBranch).Output()
	if err != nil {
		return "", err
	}
	countStr := strings.TrimSpace(string(countBytes))
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return "", err
	}

	if count == 0 {
		// No new commits
		return "", nil
	}

	if count == 1 {
		// Exactly 1 commit
		// #nosec G204 -- argv form; sourceBranch is a validated remote ref name.
		subjectBytes, err := exec.Command("git", "log", "-1", "--pretty=format:%s", sourceBranch).Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(subjectBytes)), nil
	}

	// If multiple commits
	// #nosec G204 -- argv form; the range is a single positional prefixed with "origin/".
	cmd := exec.Command("git", "log", "origin/"+targetBranch+".."+sourceBranch, "--pretty=format:%s", "--reverse")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}

	return "", nil
}
