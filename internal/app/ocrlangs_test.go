package app

import (
	"io"
	"os"
	"testing"

	"doc-html-translate/internal/config"
	"doc-html-translate/internal/ocr"
)

// -ocr-langs is a frozen spelling (OCR-INVOCATION section F): it prints the listing of internal/ocr, the very text the
// command `ocr langs` prints without --json, so the two cannot differ (ticket E0146).
func TestOCRLangsFlagPrintsTheListingOfTheCommand(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	code, runErr := New(config.Config{OCRList: true, UILang: "en"}).Run()
	os.Stdout = old
	_ = w.Close()
	out, _ := io.ReadAll(r)
	if code != 0 || runErr != nil {
		t.Fatalf("Run = %d, %v", code, runErr)
	}
	if string(out) != ocr.LangsText() {
		t.Errorf("-ocr-langs printed\n%q\nwant the listing of the command\n%q", out, ocr.LangsText())
	}
}
