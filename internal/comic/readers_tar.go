package comic

import (
	"archive/tar"
	"fmt"
	"io"
	"os"

	"doc-html-translate/internal/logging"
)

// readCBT reads a CBT (TAR container) and returns its page entries. TAR is a
// sequential format with no central directory, so entries are streamed in order;
// natural sorting happens later in Extract, identically to CBZ.
func readCBT(path string) ([]page, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open cbt: %w", err)
	}
	defer f.Close()

	tr := tar.NewReader(f)
	var c collector
	for {
		hdr, terr := tr.Next()
		if terr == io.EOF {
			break
		}
		if terr != nil {
			return nil, fmt.Errorf("read cbt: %w", terr)
		}
		if !hdr.FileInfo().Mode().IsRegular() {
			continue
		}
		if !isPageEntry(hdr.Name) {
			continue
		}
		if hdr.Size > maxPageBytes {
			logging.Printf("  WARNING: comic page %s is larger than %d bytes, skipped\n", hdr.Name, maxPageBytes)
			continue
		}
		data, rerr := io.ReadAll(io.LimitReader(tr, maxPageBytes+1))
		if rerr != nil {
			logging.Printf("  WARNING: comic page %s could not be read, skipped: %v\n", hdr.Name, rerr)
			continue
		}
		if int64(len(data)) > maxPageBytes {
			logging.Printf("  WARNING: comic page %s exceeded %d bytes while reading, skipped\n", hdr.Name, maxPageBytes)
			continue
		}
		if err := c.add(hdr.Name, data); err != nil {
			return nil, err
		}
	}
	return c.pages, nil
}
