//go:build !windows

package dialog

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ConfirmYesNo prints a Y/N prompt to stdout. Returns true if user answers yes.
func ConfirmYesNo(title, message string) bool {
	fmt.Printf("\n=== %s ===\n%s\n\nProceed? [y/N]: ", title, message)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	ans := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return ans == "y" || ans == "yes"
}

// ShowWarning prints a warning message to stderr.
func ShowWarning(title, message string) {
	fmt.Fprintf(os.Stderr, "\n⚠ WARNING: %s\n%s\n", title, message)
}
