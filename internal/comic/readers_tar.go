package comic

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"strings"

	"doc-html-translate/internal/limits"
)

// openCBT lists a TAR container. TAR has no central directory, so the headers are
// walked once and each regular file's data offset is recorded; a page's open() is
// then a section of the file, read only when that page is written. The walk seeks
// over the data, so listing a 2 GB archive reads its headers, not its pages.
func openCBT(path string) (*archive, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open cbt: %w", err)
	}
	arc := &archive{close: func() { _ = f.Close() }}
	tr := tar.NewReader(f)
	for {
		hdr, terr := tr.Next()
		if terr == io.EOF {
			break
		}
		if terr != nil {
			arc.close()
			return nil, fmt.Errorf("read cbt: %w", terr)
		}
		arc.count++
		if arc.count > limits.MaxArchiveEntries {
			break // over budget already; selectPages refuses it without walking the rest
		}
		if hdr.Typeflag != tar.TypeReg || isSparse(hdr) {
			continue
		}
		// tar.Reader reads whole header blocks and nothing ahead, so after Next the
		// file position is the first byte of this entry's data.
		off, serr := f.Seek(0, io.SeekCurrent)
		if serr != nil {
			arc.close()
			return nil, fmt.Errorf("read cbt: %w", serr)
		}
		size := hdr.Size
		arc.entries = append(arc.entries, entry{
			name: hdr.Name,
			size: size,
			open: func() (io.ReadCloser, error) { return io.NopCloser(io.NewSectionReader(f, off, size)), nil },
		})
	}
	return arc, nil
}

// isSparse reports a GNU sparse entry, whose stored bytes are not its content laid
// out in order - a section read would return the packed map, not the page.
func isSparse(hdr *tar.Header) bool {
	for k := range hdr.PAXRecords {
		if strings.HasPrefix(k, "GNU.sparse.") {
			return true
		}
	}
	return false
}
