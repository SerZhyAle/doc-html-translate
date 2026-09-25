//go:build windows

package pdf

import (
	"os"
	"os/exec"

	"doc-html-translate/internal/dialog"
	"doc-html-translate/internal/epub"
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

// retryBlockedPDFToText handles a pdftotext that exists but could not be started.
// It returns the book when a retry succeeded, nil when the caller should fall back.
func retryBlockedPDFToText(pdfPath, outputDir string) *epub.Book {
	logging.Printf("  pdftotext blocked (possibly by antivirus) - attempting auto-install..\n")
	if p := tryInstallPoppler(); p != "" {
		logging.Printf("  Retrying with system pdftotext: %s\n", p)
		if book, err := extractWithPDFToText(p, pdfPath, outputDir); err == nil {
			return book
		}
	}
	logging.Printf("  Auto-install failed. To install manually, run:\n")
	logging.Printf("    winget install ossia.poppler\n")
	dialog.ShowWarning(
		"PDF Quality Reduced - pdftotext Unavailable",
		"pdftotext could not run (blocked by antivirus or unavailable)\n"+
			"and automatic installation failed.\n"+
			"Text extraction fell back to a less accurate method - ligatures\n"+
			"and complex fonts may not render correctly.\n\n"+
			"To restore full quality, run:\n\n"+
			"  winget install ossia.poppler\n\n"+
			"Or exclude the app cache from antivirus scans:\n"+
			"  %LOCALAPPDATA%\\doc-html-translate\\pdftotext\\",
	)
	return nil
}

// tryInstallPoppler runs "winget install ossia.poppler" and returns the path to
// pdftotext if installation succeeded and the binary can be located.
func tryInstallPoppler() string {
	winget, err := exec.LookPath("winget")
	if err != nil {
		return ""
	}
	logging.Printf("  Installing Poppler via winget..\n")
	cmd := exec.Command(winget, "install", "--id", "ossia.poppler",
		"--accept-package-agreements", "--accept-source-agreements")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return ""
	}
	return findSystemPDFToText()
}
