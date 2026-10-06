package pipeline

import (
	"doc-html-translate/internal/comic"
	"doc-html-translate/internal/img"
)

// SupportsImageOCR says which input readers can put images in generated HTML.
// Text and RTF readers produce only text, so selecting -ocr adds no prerequisite for them.
func SupportsImageOCR(ext string) bool {
	if img.IsImage(ext) || comic.IsComic(ext) {
		return true
	}
	switch ext {
	case ".epub", ".pdf", ".mobi", ".azw3", ".fb2", ".md", ".html", ".htm":
		return true
	}
	return false
}

// Convertible says which extensions build converts: exactly the arms of its format switch,
// without the plain-text fallback for an unknown extension. A FileDO secret file is unwrapped
// first, and what comes out is dispatched by this list, so a type that only the fallback would
// swallow (a .docx, a .zip) is refused instead of being read as garbage.
func Convertible(ext string) bool {
	if img.IsImage(ext) || comic.IsComic(ext) {
		return true
	}
	switch ext {
	case ".epub", ".pdf", ".txt", ".md", ".fb2", ".rtf", ".html", ".htm", ".mobi", ".azw3":
		return true
	}
	return false
}
