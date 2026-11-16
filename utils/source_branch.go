package utils

import (
	"errors"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true)

// GetCurrentBranch is exported so main.go (or other packages) can call it.
func GetCurrentBranch() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", errors.New(errorStyle.Render("Error: Unable to determine the current branch.\nMake sure you are inside a Git repository with valid branches."))
	}
	return strings.TrimSpace(string(out)), nil
}
