// Package browser opens URLs/files in the default system browser.
package browser

import (
	"os"
	"path/filepath"
	"strings"
)

// normalizeTarget enforces opening index.html when the target points to a
// legacy XHTML chapter file.
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
