//go:build windows

package pdf

import (
	"context"
	"doc-html-translate/internal/dialog"
	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/logging"
)

var pdftotextKnownPaths = []string{
	`C:\Program Files\poppler\bin\pdftotext.exe`,
	`C:\Program Files (x86)\poppler\bin\pdftotext.exe`,
	`C:\Program Files\Git\mingw64\bin\pdftotext.exe`,
	`C:\Program Files (x86)\Git\mingw64\bin\pdftotext.exe`,
	`C:\Program Files\Xpdf\bin64\pdftotext.exe`,
}

var pdftotextKnownGlobs = []string{
	`C:\Program Files\poppler*\bin\pdftotext.exe`,
	`C:\Program Files (x86)\poppler*\bin\pdftotext.exe`,
}

// pdftotextMissingAdvice is empty on Windows: the binary ships inside the app, so it is
// only ever blocked, never simply absent, and the blocked case has its own warning.
func pdftotextMissingAdvice() string { return "" }

// retryBlockedPDFToText handles a bundled pdftotext that exists but could not be started,
// typically quarantined by antivirus. A Poppler the user installed is tried; otherwise the
// user is told how to install one. Nothing is installed from here: an unrequested system-wide
// install in the middle of a conversion is not the converter's call to make.
func retryBlockedPDFToText(ctx context.Context, pdfPath, outputDir string) *epub.Book {
	if p := findSystemPDFToText(); p != "" {
		logging.Printf("  Retrying with system pdftotext: %s\n", p)
		if book, err := extractWithPDFToText(ctx, p, pdfPath, outputDir); err == nil {
			return book
		}
	}
	// The advice deliberately stops at installing Poppler: telling a user to exempt a folder
	// from antivirus scanning is weakening a protection, which INSTALL-TRUST rules out.
	advice := i18n.S("pdftotext could not run (possibly blocked by antivirus), so the text was read with a less accurate method - ligatures and complex fonts may look wrong. Nothing is installed automatically. To restore full quality, install Poppler yourself, for example: winget install ossia.poppler")
	logging.Printf("  %s\n", advice)
	dialog.ShowWarning(i18n.S("PDF quality reduced - pdftotext unavailable"), advice)
	return nil
}
