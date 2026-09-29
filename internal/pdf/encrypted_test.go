package pdf

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pdflib "github.com/ledongthuc/pdf"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// encryptAES256 encrypts src with an empty user password - a file any reader opens without asking,
// the shape of the corpus scan ja-scanpdf-senryu-manga.
func encryptAES256(t *testing.T, src, dst string) {
	t.Helper()
	conf := model.NewAESConfiguration("", "owner-secret", 256)
	if err := api.EncryptFile(src, dst, conf); err != nil {
		t.Fatalf("encrypt fixture: %v", err)
	}
}

// Ticket 72: an AES-256 scan failed the whole conversion. pdftotext read it and found no text, the
// pure-Go reader could not open it ("256-bit encryption key"), and nothing tried the page images
// pdfcpu extracts without trouble.
func TestImageOnlyFallbackBuildsAnEncryptedScan(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "plain.pdf")
	buildFixturePDF(t, plain, 3, nil, allPages(3))
	enc := filepath.Join(dir, "scan.pdf")
	encryptAES256(t, plain, enc)

	// The premise: the reader the pipeline falls back to cannot open the file.
	if f, _, err := pdflib.Open(enc); err == nil {
		_ = f.Close()
		t.Skip("ledongthuc/pdf opens AES-256 files now; the fallback is no longer reachable this way")
	}

	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	cause := errors.New("open pdf: malformed PDF: 256-bit encryption key")
	book, err := imageOnlyFallback(context.Background(), true, enc, out, cause)
	if err != nil {
		t.Fatalf("image-only fallback failed: %v", err)
	}
	if len(book.Spine) != 3 {
		t.Fatalf("got %d pages, want 3", len(book.Spine))
	}
	page, err := os.ReadFile(filepath.Join(out, book.Manifest[0].Href))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), "<img") {
		t.Fatalf("the first page carries no image:\n%s", page)
	}
}

// A file pdftotext did not find textless keeps the reader's own error: the fallback is for scans,
// not a way to hide a broken text PDF behind whatever pictures it holds.
func TestImageOnlyFallbackKeepsTheCauseForATextPDF(t *testing.T) {
	cause := errors.New("open pdf: malformed PDF")
	if _, err := imageOnlyFallback(context.Background(), false, "unused.pdf", t.TempDir(), cause); !errors.Is(err, cause) {
		t.Fatalf("got %v, want the original cause", err)
	}
}
