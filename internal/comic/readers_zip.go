package comic

import (
	"archive/zip"
	"fmt"
	"io"
	"math"
)

// openCBZ lists a ZIP container from its central directory. Nothing is inflated
// here; each page's open() inflates it later, straight into its page file.
// Symlink entries are left out: a page is a regular file or nothing.
func openCBZ(path string) (*archive, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open cbz: %w", err)
	}
	arc := &archive{count: len(r.File), close: func() { _ = r.Close() }}
	for _, f := range r.File {
		if !f.Mode().IsRegular() {
			continue
		}
		arc.entries = append(arc.entries, entry{
			name: f.Name,
			size: int64(min(f.UncompressedSize64, math.MaxInt64)),
			open: func() (io.ReadCloser, error) { return f.Open() },
		})
	}
	return arc, nil
}
