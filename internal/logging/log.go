// Package logging provides simple timestamp-prefixed console output.
package logging

import (
	"fmt"
	"os"
	"time"
)

// AppVersion is set by main before any pipeline output so it appears in the first log line.
var AppVersion = "dev"

var stdoutIsTerminal = detectStdoutTerminal()

func ts() string {
	return time.Now().Format("[15:04:05] ")
}

// Printf prints a timestamped message to stdout.
func Printf(format string, args ...any) {
	fmt.Printf(ts()+format, args...)
}

// Println prints a timestamped line to stdout.
func Println(s string) {
	fmt.Printf(ts()+"%s\n", s)
}

// Errorf prints a timestamped message to stderr.
func Errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, ts()+format, args...)
}

// Progress prints an in-place progress update line (uses \r to overwrite, no trailing newline).
func Progress(format string, args ...any) {
	if stdoutIsTerminal {
		fmt.Printf("\r"+ts()+format, args...)
		return
	}
	// In non-interactive outputs (pipes/UI log capture), carriage return creates
	// broken lines. Emit normal timestamped lines instead.
	fmt.Printf(ts()+format+"\n", args...)
}

func detectStdoutTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
