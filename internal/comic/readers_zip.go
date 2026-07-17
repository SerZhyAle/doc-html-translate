package comic

import (
	"archive/zip"
	"fmt"
	"io"

	"doc-html-translate/internal/logging"
)

// collector accumulates page entries while enforcing the archive-hardening caps
// (page count and inflated total). Per-entry size is checked by each reader
// before the bytes are read, so a declared-oversize entry is skipped without ever
// being inflated.
type collector struct {
	pages []page
	total int64
}

func (c *collector) add(name string, data []byte) error {
	if len(c.pages) >= maxPages {
		return fmt.Errorf("comic archive has more than %d pages - refusing to extract", maxPages)
	}
	c.total += int64(len(data))
	if c.total > maxTotalBytes {
		return fmt.Errorf("comic archive inflates to more than %d bytes - refusing to extract", maxTotalBytes)
	}
	c.pages = append(c.pages, page{name: name, data: data})
	return nil
}

// readCBZ reads a CBZ (ZIP container) and returns its page entries. Non-page
// entries (ComicInfo.xml, directories, OS cruft) are filtered by isPageEntry; a
// single entry declaring more than maxPageBytes is skipped with a warning rather
// than inflated.
func readCBZ(path string) ([]page, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open cbz: %w", err)
	}
	defer r.Close()

	var c collector
	for _, f := range r.File {
		if !isPageEntry(f.Name) {
			continue
		}
		if f.UncompressedSize64 > maxPageBytes {
			logging.Printf("  WARNING: comic page %s is larger than %d bytes, skipped\n", f.Name, maxPageBytes)
			continue
		}
		rc, oerr := f.Open()
		if oerr != nil {
			logging.Printf("  WARNING: comic page %s could not be opened, skipped: %v\n", f.Name, oerr)
			continue
		}
		data, rerr := io.ReadAll(io.LimitReader(rc, maxPageBytes+1))
		rc.Close()
		if rerr != nil {
			logging.Printf("  WARNING: comic page %s could not be read, skipped: %v\n", f.Name, rerr)
			continue
		}
		if int64(len(data)) > maxPageBytes {
			logging.Printf("  WARNING: comic page %s exceeded %d bytes while reading, skipped\n", f.Name, maxPageBytes)
			continue
		}
		if err := c.add(f.Name, data); err != nil {
			return nil, err
		}
	}
	return c.pages, nil
}
