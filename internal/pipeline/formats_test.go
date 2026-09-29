package pipeline

import "testing"

func TestSupportsImageOCR(t *testing.T) {
	for _, ext := range []string{".epub", ".pdf", ".mobi", ".azw3", ".fb2", ".md", ".html", ".htm", ".cbz", ".cbr", ".cb7", ".cbt", ".png", ".webp"} {
		if !SupportsImageOCR(ext) {
			t.Errorf("%s can contain images", ext)
		}
	}
	for _, ext := range []string{".txt", ".rtf", ".bin"} {
		if SupportsImageOCR(ext) {
			t.Errorf("%s cannot contain images", ext)
		}
	}
}
