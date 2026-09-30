package ocr

import (
	"os"
	"path/filepath"
)

// executablePath is replaceable so the winget portable alias can be exercised in tests.
var executablePath = os.Executable

// runningExecutableDir returns the directory holding the real binary. Winget launches its
// portable executables through links in WinGet/Links, which contain no bundled OCR files.
func runningExecutableDir() string {
	exe, err := executablePath()
	if err != nil {
		return ""
	}
	if target, err := filepath.EvalSymlinks(exe); err == nil {
		exe = target
	}
	return filepath.Dir(exe)
}
