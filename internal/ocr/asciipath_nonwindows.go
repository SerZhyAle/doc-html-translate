//go:build !windows

package ocr

import "path/filepath"

// shortPath has no counterpart here: there are no 8.3 names, so only an ASCII path or a
// fallback root passes the check.
func shortPath(p string) string { return p }

// platformStagingRoots are the standard system temp folders, which do not follow $TMPDIR.
func platformStagingRoots() []string {
	return []string{
		filepath.Join("/tmp", stagingDirName),
		filepath.Join("/var/tmp", stagingDirName),
	}
}
