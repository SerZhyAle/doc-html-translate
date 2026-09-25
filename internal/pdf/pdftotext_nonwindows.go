//go:build !windows

package pdf

import (
	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/logging"
)

var pdftotextKnownPaths = []string{
	"/usr/bin/pdftotext",
	"/usr/local/bin/pdftotext",
	"/opt/homebrew/bin/pdftotext",
}

var pdftotextKnownGlobs []string

// pdftotextMissingAdvice tells the user how to get the better extractor. Nothing is bundled
// for this platform, so an absent pdftotext is the normal state of a fresh machine.
func pdftotextMissingAdvice() string {
	return i18n.S("pdftotext not found - using the built-in PDF reader. For better text, install Poppler (package poppler-utils, or brew install poppler).")
}

// retryBlockedPDFToText only advises: the system package manager is the user's to drive,
// so nothing is installed or retried here.
func retryBlockedPDFToText(_, _ string) *epub.Book {
	logging.Printf("  %s\n", pdftotextMissingAdvice())
	return nil
}
