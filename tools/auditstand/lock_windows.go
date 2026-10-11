//go:build windows

// lock_windows.go holds the file the stand keeps open with no share modes - the unreadable-file
// member of S16 is made at scene time by locking a regular input against readers.

package main

import (
	"io"

	"golang.org/x/sys/windows"
)

// lockFileExclusive opens path denying every share: a reader's CreateFile fails for as long as
// the returned closer stays open.
func lockFileExclusive(path string) (io.Closer, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, 0 /*no share*/, nil,
		windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, err
	}
	return closerFunc(func() error { return windows.CloseHandle(h) }), nil
}

type closerFunc func() error

func (f closerFunc) Close() error { return f() }
