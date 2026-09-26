//go:build !windows

package dialog

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Confirm prints a Y/N prompt to stdout. Returns true only on an explicit yes, so Enter declines.
// Under the GUI the question is asked in the GUI's window.
func Confirm(title, message string) bool {
	if hostedByGUI() {
		return askHost(title, message)
	}
	fmt.Printf("\n=== %s ===\n%s\n\nProceed? [y/N]: ", title, message)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	ans := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return ans == "y" || ans == "yes"
}

// ShowWarning prints a warning message to stderr, or hands it to the GUI that runs the converter.
// An unattended run also keeps it in the run log.
func ShowWarning(title, message string) {
	if hostedByGUI() {
		noteHost(title, message)
		return
	}
	if unattended.Load() {
		logWarning(title, message)
		return
	}
	fmt.Fprintf(os.Stderr, "\n⚠ WARNING: %s\n%s\n", title, message)
}
