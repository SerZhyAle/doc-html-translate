//go:build !windows

package main

import (
	"io"
	"os"
)

// openConsole is where the commands front writes: on a platform whose executables have a console of their own it is
// standard output and standard error (the Windows side attaches to the parent's console, console_windows.go).
func openConsole() (io.Writer, io.Writer) { return os.Stdout, os.Stderr }
