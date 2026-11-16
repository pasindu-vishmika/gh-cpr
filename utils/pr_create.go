package utils

import (
	"fmt"
	"os/exec"
	"strings"
)

// ExecutePR creates the pull request via the GitHub CLI and returns the URL of
// the created PR. On failure it returns an error wrapping gh's own output so
// the caller can surface the real reason (e.g. not authenticated).
func ExecutePR(sBranch, tBranch, title, body string) (string, error) {
	// #nosec G204 -- no shell is invoked (argv form). sBranch/tBranch are chosen
	// from the remote branch list (git-ref constrained); title/body are passed
	// as values to the --title/--body flags, so they cannot pose as options.
	cmd := exec.Command("gh", "pr", "create",
		"--base", tBranch,
		"--head", sBranch,
		"--title", title,
		"--body", body,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s\n%w\n"+
			"Make sure GitHub CLI is installed and authenticated (run 'gh auth status')",
			strings.TrimSpace(string(out)), err)
	}

	// gh prints the created PR's URL to stdout; grab the last http(s) line.
	var url string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
			url = line
		}
	}
	return url, nil
}
