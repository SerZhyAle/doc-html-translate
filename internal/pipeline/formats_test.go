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

func TestConvertible(t *testing.T) {
	for _, ext := range []string{".epub", ".pdf", ".txt", ".md", ".fb2", ".rtf", ".html", ".htm", ".mobi", ".azw3", ".png", ".jpg", ".cbz", ".cbr"} {
		if !Convertible(ext) {
			t.Errorf("%s is converted by build", ext)
		}
	}
	for _, ext := range []string{".docx", ".fd-sec", "", ".zip", ".exe"} {
		if Convertible(ext) {
			t.Errorf("%q is not converted by build", ext)
		}
	}
}
