//go:build !windows

// lock_other.go: the unreadable-file member only exists on the Windows stand.

package main

import "io"

// lockFileExclusive refuses off Windows; the error text comes from probes_other.go.
func lockFileExclusive(path string) (io.Closer, error) { return nil, errNotWindows }
