package pipeline

import (
	"errors"
	"runtime/debug"

	"doc-html-translate/internal/epub"
	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/logging"
)

// ExitInternal is the code for a panic that reached the top of the pipeline. It reuses the
// parse code rather than adding one, because the exit codes are a published contract
// (OCR-INVOCATION) and a new code is an amendment there first; and a panic that escapes every
// narrower guard is, in practice, a third-party parser failing on this document, which is what
// the parse code already tells a caller: this input could not be converted.
const ExitInternal = ExitParse

// Run executes the pipeline behind a last-resort panic guard, so a bug or a library panic ends
// as a logged internal error and a clean exit code instead of a Go crash dump on the console.
// The stack still goes to the run log: a recovered panic must stay diagnosable.
func (r Runner) Run() (code int, err error) {
	defer func() {
		if p := recover(); p != nil {
			logging.RunLogf("internal error: %v\n%s\n", p, debug.Stack())
			code, err = ExitInternal, errors.New(i18n.S("internal error while converting (details in the run log): %v", p))
		}
	}()
	return r.run()
}

// overlayImagesSafe keeps the OCR stage best-effort even against a panic outside the per-image
// guard (the page rewriting, the colour sampling): the pages keep whatever overlay was written
// and the conversion goes on without the rest.
func (r Runner) overlayImagesSafe(book *epub.Book, outputDir string) {
	defer func() {
		if p := recover(); p != nil {
			logging.Printf("  WARNING: OCR overlay stopped by an internal error: %v\n", p)
			logging.RunLogf("%s\n", debug.Stack())
		}
	}()
	r.overlayImages(book, outputDir)
}
