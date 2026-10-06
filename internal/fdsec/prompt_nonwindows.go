//go:build !windows

package fdsec

// stdinIsConsole is false on other systems: an echoing prompt is never offered for a password.
func stdinIsConsole() bool { return false }

// readConsoleSecret always refuses: without a no-echo console the password has no safe prompt.
func readConsoleSecret(string) ([]byte, error) { return nil, errNoConsole }
