package readiness

import (
	"context"
	"errors"
	"regexp"
	"time"

	"doc-html-translate/internal/comic"
	"doc-html-translate/internal/mobi"
	"doc-html-translate/internal/ocr"
	"doc-html-translate/internal/pdf"
	"doc-html-translate/internal/procrun"
)

// versionDeadline bounds one tool's own version query: a version is a courtesy of the report, never worth a wait.
const versionDeadline = 3 * time.Second

// versionNumber is the first dotted number of what a tool prints: "tesseract 5.3.3" gives 5.3.3, "ebook-convert (calibre
// 7.5.1)" gives 7.5.1, "7-Zip 23.01 (x64)" gives 23.01. Nothing else is kept, so no path can enter a report through it.
var versionNumber = regexp.MustCompile(`\d+(?:\.\d+)+`)

// parseVersion returns the version number in the text a tool printed, or "".
func parseVersion(text string) string {
	if len(text) > 4096 {
		text = text[:4096]
	}
	v := versionNumber.FindString(text)
	if len(v) > 32 {
		return ""
	}
	return v
}

// readVersion runs path with args and returns the version number it prints on standard output or standard error ("" when
// it cannot start, is not done within the deadline, or prints no number). A tool that exits non-zero after printing its
// banner still counts: `7z` with no arguments does.
func readVersion(ctx context.Context, tool, path string, args ...string) string {
	res, err := procrun.Run(ctx, procrun.Cmd{
		Tool: tool, Path: path, Args: args, Timeout: versionDeadline, MaxStdout: 4096, MaxStderr: 4096,
	})
	var pe *procrun.Error
	if errors.As(err, &pe) && (pe.TimedOut || pe.Canceled) {
		return ""
	}
	return parseVersion(string(res.Stdout) + "\n" + string(res.Stderr))
}

// toolVersion is the Probes.Version the program uses: it locates the tool the way a conversion does and asks it for its
// version. The bundled pdftotext is not asked (it would have to be unpacked), and FileDO has no version query.
func toolVersion(ctx context.Context, tool string) string {
	switch tool {
	case ToolTesseract:
		if path, err := ocr.Locate(); err == nil {
			return readVersion(ctx, tool, path, "--version")
		}
	case ToolPDFToText:
		if path := pdf.SystemPDFToTextPath(); path != "" {
			return readVersion(ctx, tool, path, "-v")
		}
	case ToolCalibre:
		if path := mobi.Path(); path != "" {
			return readVersion(ctx, tool, path, "--version")
		}
	case ToolSevenZip:
		if path := comic.SevenZipPath(); path != "" {
			return readVersion(ctx, tool, path)
		}
	}
	return ""
}
