//go:build windows

package fdsec

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows"
)

// stdinIsConsole reports whether standard input is a console a person can type into.
func stdinIsConsole() bool {
	var mode uint32
	return windows.GetConsoleMode(windows.Handle(os.Stdin.Fd()), &mode) == nil
}

// readConsoleSecret reads one line from the console with echo off. Nothing is trimmed: a space
// is part of a password. The console mode is restored on every path.
func readConsoleSecret(prompt string) ([]byte, error) {
	h := windows.Handle(os.Stdin.Fd())
	var saved uint32
	if err := windows.GetConsoleMode(h, &saved); err != nil {
		return nil, errNoConsole
	}
	if err := windows.SetConsoleMode(h, saved&^windows.ENABLE_ECHO_INPUT); err != nil {
		return nil, errNoConsole
	}
	defer func() { _ = windows.SetConsoleMode(h, saved) }()

	fmt.Fprint(os.Stderr, prompt)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	fmt.Fprintln(os.Stderr)
	if err != nil && line == "" {
		return nil, err
	}
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return []byte(line), nil
}
