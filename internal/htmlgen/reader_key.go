package htmlgen

import (
	"fmt"
	"hash/fnv"

	"doc-html-translate/internal/epub"
)

// ReaderKey derives the id that namespaces a book's saved reading position in
// localStorage. It is built from what the conversion cannot change - the
// source file's name and size, plus the title and page count as extracted,
// before any translation - so chapter pages and index.html, written before and
// after translation, agree on it, while two books that merely share a title
// and page count do not. The pipeline stores it on the book once
// (epub.Book.ReaderKey) right after extraction.
func ReaderKey(sourceName string, sourceSize int64, title string, pages int) string {
	h := fnv.New64a()
	fmt.Fprintf(h, "%s\x00%d\x00%s\x00%d", sourceName, sourceSize, title, pages)
	return fmt.Sprintf("%016x", h.Sum64())
}

// readerKey returns the book's reading-position key, deriving it on first use
// for a caller that did not set one. It is stored on the book either way, so a
// later call - after the title was translated - still returns the same key.
func readerKey(book *epub.Book) string {
	if book.ReaderKey == "" {
		book.ReaderKey = ReaderKey("", 0, book.Title, len(book.Spine))
	}
	return book.ReaderKey
}
