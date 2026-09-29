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
