//go:build windows

package ocr

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// shortPath returns the 8.3 short name of an existing path, or p itself when the call fails or
// the volume keeps no short names (then the long name comes back, and the ASCII check rejects it).
func shortPath(p string) string {
	from, err := windows.UTF16PtrFromString(p)
	if err != nil {
		return p
	}
	buf := make([]uint16, windows.MAX_PATH)
	for {
		n, err := windows.GetShortPathName(from, &buf[0], uint32(len(buf)))
		if err != nil || n == 0 {
			return p
		}
		// A buffer that was too small gets back the size it needs, terminator included.
		if int(n) < len(buf) {
			return windows.UTF16ToString(buf[:n])
		}
		buf = make([]uint16, n)
	}
}

// platformStagingRoots are the per-machine folders whose paths do not depend on the user's
// name: %PUBLIC% (C:\Users\Public) is writable by every user, and %ProgramData% lets a user
// create a folder of their own.
func platformStagingRoots() []string {
	var out []string
	for _, env := range []string{"PUBLIC", "ProgramData"} {
		if v := os.Getenv(env); v != "" {
			out = append(out, filepath.Join(v, stagingDirName))
		}
	}
	return out
}
