//go:build windows

package browser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// shellExecute is indirected so a test can see exactly what reaches the shell
// without opening a browser window.
var shellExecute = windows.ShellExecute

// Open opens the given file or URL with its default handler.
//
// The target goes to ShellExecute as data, never through cmd.exe: a book named
// "a&calc&b.epub" or "50%.epub" used to be parsed as a command line by
// `cmd /c start`, which ran whatever followed the "&" and failed to open "Q&A".
// ShellExecute starts no helper process, so nothing is left to wait on or
// release. It is available to MSIX full-trust apps, so the Store build is covered.
func Open(target string) error {
	target = normalizeTarget(target)
	file, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	verb, _ := windows.UTF16PtrFromString("open")
	if err := shellExecute(0, verb, file, nil, nil, windows.SW_SHOWNORMAL); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}

// normalizeTarget enforces opening index.html when the target points to a
// legacy XHTML chapter file. It lives on the Windows side because Open is its only caller.
func normalizeTarget(target string) string {
	t := strings.TrimSpace(target)
	if t == "" {
		return target
	}

	// URLs should be opened as-is.
	if strings.HasPrefix(strings.ToLower(t), "http://") || strings.HasPrefix(strings.ToLower(t), "https://") {
		return target
	}

	ext := strings.ToLower(filepath.Ext(t))
	if ext != ".xhtml" && ext != ".xhtm" {
		return target
	}

	abs := t
	if !filepath.IsAbs(abs) {
		if p, err := filepath.Abs(abs); err == nil {
			abs = p
		}
	}

	for dir := filepath.Dir(abs); ; {
		idx := filepath.Join(dir, "index.html")
		if st, err := os.Stat(idx); err == nil && !st.IsDir() {
			return idx
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return target
}
