package htmlsplit

import (
	"strings"
	"testing"

	"doc-html-translate/internal/epub"
)

// A nav document may percent-encode a fragment ("#%D0%B3" for id="г"); the TOC
// keeps it as written, so the lookup must decode it to find the moved anchor.
func TestSplitRewritesPercentEncodedTOCFragment(t *testing.T) {
	dir := t.TempDir()
	chapter := strings.Replace(wrappedChapter(50000), `id="last"`, `id="глава"`, 1)
	book := testBook(t, dir, map[string]string{"text/chapter.html": chapter}, []string{"text/chapter.html"})
	book.TOC = []epub.TOCEntry{{Title: "Last", Href: "text/chapter.html#%D0%B3%D0%BB%D0%B0%D0%B2%D0%B0"}}

	if _, err := SplitIfNeeded(book, dir, 5000); err != nil {
		t.Fatal(err)
	}
	holder := partHolding(t, dir, book, "глава")
	if holder == "text/chapter.html" {
		t.Fatalf("the anchor stayed in the first part; test chapter too small")
	}
	if got, want := book.TOC[0].Href, holder+"#%D0%B3%D0%BB%D0%B0%D0%B2%D0%B0"; got != want {
		t.Errorf("TOC href = %q, want %q", got, want)
	}
}
