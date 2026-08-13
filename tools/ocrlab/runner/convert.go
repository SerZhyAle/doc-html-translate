package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"doc-html-translate/internal/img"
	"doc-html-translate/internal/ocr"
	"doc-html-translate/tools/ocrlab/evidence"
)

// convertScene runs one image through the shipped standalone-image path: img.Extract wraps it in
// a one-page document exactly as the app does for a user's picture, then ocr.OverlayFile applies
// the real recognition and the real plate rendering.
//
// Calling the packages rather than shelling out to a built CLI keeps the run honest in the way
// that matters (it is the same code the reader gets) while not requiring a build first. The
// diagnostics sidecar is pointed into the run directory so the geometry the app computed is
// recorded even for a scene whose render later fails.
func convertScene(bin, imgPath, workDir string, opt Options) (pagePath string, res ocr.Result, err error) {
	book, err := img.Extract(imgPath, workDir)
	if err != nil {
		return "", res, err
	}
	if len(book.Manifest) == 0 {
		return "", res, fmt.Errorf("no page produced")
	}
	pagePath = filepath.Join(workDir, book.Manifest[0].Href)

	diagPath := filepath.Join(opt.OutDir, DiagFile)
	old := os.Getenv("DOCHT_OCR_DIAG")
	if err := os.Setenv("DOCHT_OCR_DIAG", diagPath); err != nil {
		return "", res, err
	}
	defer func() { _ = os.Setenv("DOCHT_OCR_DIAG", old) }()

	stats, err := ocr.OverlayFile(bin, pagePath, opt.Lang, ocr.DataDir(), nil)
	if err != nil {
		return "", res, err
	}
	if len(stats.Failed) > 0 {
		return "", res, fmt.Errorf("overlay: %v", stats.Failed[0].Err)
	}

	// Geometry comes from the recognizer's own report of the page, which is what the plates were
	// positioned against.
	res, err = ocr.Recognize(bin, imgPath, opt.Lang, ocr.DataDir())
	if err != nil {
		return "", res, err
	}
	if stats.Overlaid == 0 {
		// Not an error: a scene may genuinely hold no recognizable text, and the metrics will
		// report that as zero recall rather than as a broken run.
		return pagePath, res, nil
	}
	return pagePath, res, nil
}

// tesseractVersion is the engine's own first line of `--version`, e.g.
// "tesseract 5.3.3". Unknown rather than fatal: a run whose engine cannot be named is still a
// run, and the name is reporting metadata rather than a dependency.
func tesseractVersion(bin string) string {
	out, err := exec.Command(bin, "--version").CombinedOutput()
	if err != nil {
		return "unknown"
	}
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return "unknown"
}

// tessdataFingerprint identifies the language data by the bytes actually loaded rather than by a
// version string.
//
// A version string would have to be mirrored from internal/ocr into the lab, and a mirrored
// magic number is the drift the parity document exists to prevent. The file's hash is
// observable, is strictly more precise (it distinguishes two builds that both call themselves
// 4.0.0) and needs no second source of truth.
func tessdataFingerprint(lang string) string {
	dir := ocr.DataDir()
	if dir == "" {
		return ""
	}
	code := strings.Split(lang, "+")[0]
	f, err := os.Open(filepath.Join(dir, code+".traineddata"))
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))[:16]
}

// engineFor fills the evidence's engine record from what can be observed.
func engineFor(bin, lang string) evidence.Engine {
	return evidence.Engine{
		Tesseract:       tesseractVersion(bin),
		TessdataVersion: tessdataFingerprint(lang),
		Lang:            lang,
	}
}

// peakRSS reports the Go runtime's total memory obtained from the OS.
//
// Conversion happens in this process, so this is the honest in-process proxy for the cost
// dimension: it is not the operating system's RSS and does not pretend to be, but it moves when
// a scene makes the converter allocate and that is what the measurement is for.
func peakRSS() int64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.Sys)
}
