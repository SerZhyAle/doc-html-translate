package pdf

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pdflib "github.com/ledongthuc/pdf"

	"github.com/go-pdf/fpdf"
	"golang.org/x/image/tiff"
)

// createTestPDF generates a minimal PDF with the given pages of text.
// Each element in pages is the text content for one page.
func createTestPDF(t *testing.T, filePath string, pages []string) {
	t.Helper()
	doc := fpdf.New("P", "mm", "A4", "")
	doc.SetFont("Helvetica", "", 12)
	for _, text := range pages {
		doc.AddPage()
		// Write text line by line
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			doc.CellFormat(0, 6, line, "", 1, "", false, 0, "")
		}
	}
	if err := doc.OutputFileAndClose(filePath); err != nil {
		t.Fatalf("create test PDF: %v", err)
	}
}

func TestPdfTitle(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{`C:\Books\My Report.pdf`, "My Report"},
		{"/home/user/document.pdf", "document"},
		{"simple.pdf", "simple"},
		{"path/to/Report 2024.pdf", "Report 2024"},
	}
	for _, tt := range tests {
		got := pdfTitle(tt.path)
		if got != tt.want {
			t.Errorf("pdfTitle(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestRowsToText(t *testing.T) {
	// Empty rows
	got := rowsToText(nil)
	if got != "" {
		t.Errorf("rowsToText(nil) = %q, want empty", got)
	}

	// Single row with content
	rows := pdflib.Rows{
		{Content: pdflib.TextHorizontal{
			{S: "Hello ", X: 10, Y: 100},
			{S: "World", X: 30, Y: 100},
		}},
	}
	got = rowsToText(rows)
	if strings.TrimSpace(got) != "Hello World" {
		t.Errorf("rowsToText single row = %q, want 'Hello World'", strings.TrimSpace(got))
	}

	// Multiple rows
	rows = pdflib.Rows{
		{Content: pdflib.TextHorizontal{{S: "Line one", X: 10, Y: 100}}},
		{Content: pdflib.TextHorizontal{{S: "Line two", X: 30, Y: 70}}},
		{Content: pdflib.TextHorizontal{{S: "Line three", X: 50, Y: 40}}},
	}
	got = rowsToText(rows)
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 3 {
		t.Errorf("rowsToText multi-row: got %d lines, want 3", len(lines))
	}

	// Row with only whitespace is skipped
	rows = pdflib.Rows{
		{Content: pdflib.TextHorizontal{{S: "  ", X: 10, Y: 100}}},
		{Content: pdflib.TextHorizontal{{S: "Real content", X: 10, Y: 80}}},
	}
	got = rowsToText(rows)
	if strings.TrimSpace(got) != "Real content" {
		t.Errorf("rowsToText whitespace row = %q, want 'Real content'", strings.TrimSpace(got))
	}
}

func TestBuildPageHTML(t *testing.T) {
	html := buildPageHTML("Test Book", 3, 10, "Hello World\nSecond line", nil)

	// Check structure
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("missing DOCTYPE")
	}
	if !strings.Contains(html, "<title>Test Book — Page 3</title>") {
		t.Error("missing/wrong title")
	}
	if !strings.Contains(html, "Page 3 / 10") {
		t.Error("missing page header")
	}
	if !strings.Contains(html, "<p>Hello World</p>") {
		t.Error("missing first paragraph")
	}
	if !strings.Contains(html, "<p>Second line</p>") {
		t.Error("missing second paragraph")
	}
}

func TestBuildPageHTML_EscapesHTML(t *testing.T) {
	html := buildPageHTML("Book <script>", 1, 1, "<b>bold</b>", nil)
	if strings.Contains(html, "<script>") {
		t.Error("title not escaped")
	}
	if strings.Contains(html, "<b>bold</b>") {
		t.Error("content not escaped")
	}
	if !strings.Contains(html, "&lt;b&gt;bold&lt;/b&gt;") {
		t.Error("content should be HTML-escaped")
	}
}

func TestExtract_ValidPDF(t *testing.T) {
	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, "test_book.pdf")

	createTestPDF(t, pdfPath, []string{
		"Chapter one content\nWith two lines",
		"Chapter two content\nAnother line here",
	})

	outputDir := filepath.Join(tmpDir, "output")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}

	book, err := Extract(pdfPath, outputDir)
	if err != nil {
		t.Fatalf("Extract() error: %v", err)
	}

	// Title should come from filename
	if book.Title != "test_book" {
		t.Errorf("Title = %q, want 'test_book'", book.Title)
	}

	// BasePath should be empty for PDF
	if book.BasePath != "" {
		t.Errorf("BasePath = %q, want empty", book.BasePath)
	}

	// Should have generated pages
	if len(book.Manifest) == 0 {
		t.Fatal("no manifest items generated")
	}
	if len(book.Spine) == 0 {
		t.Fatal("no spine items generated")
	}
	if len(book.Manifest) != len(book.Spine) {
		t.Errorf("manifest (%d) != spine (%d) count", len(book.Manifest), len(book.Spine))
	}

	// SpineHrefs should return page files
	hrefs := book.SpineHrefs()
	if len(hrefs) == 0 {
		t.Fatal("SpineHrefs() returned empty")
	}

	// All generated HTML files should exist
	for _, href := range hrefs {
		pagePath := filepath.Join(outputDir, href)
		if _, err := os.Stat(pagePath); os.IsNotExist(err) {
			t.Errorf("page file not found: %s", pagePath)
		}
	}

	// ContentFiles should return all pages (they have text/html media type)
	contentFiles := book.ContentFiles()
	if len(contentFiles) != len(book.Manifest) {
		t.Errorf("ContentFiles() = %d, want %d", len(contentFiles), len(book.Manifest))
	}

	// Verify HTML content of first page
	firstPage := filepath.Join(outputDir, hrefs[0])
	data, err := os.ReadFile(firstPage)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "<!DOCTYPE html>") {
		t.Error("page missing DOCTYPE")
	}
	if !strings.Contains(content, "<body>") {
		t.Error("page missing body tag")
	}
}

func TestExtract_EmptyPDF(t *testing.T) {
	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, "empty.pdf")

	// Create PDF with one empty page
	doc := fpdf.New("P", "mm", "A4", "")
	doc.AddPage() // empty page, no text
	if err := doc.OutputFileAndClose(pdfPath); err != nil {
		t.Fatal(err)
	}

	outputDir := filepath.Join(tmpDir, "output")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}

	book, err := Extract(pdfPath, outputDir)
	if err != nil {
		t.Fatalf("expected fallback HTML for empty PDF, got error: %v", err)
	}
	if len(book.Spine) != 1 {
		t.Fatalf("expected 1 fallback page, got %d", len(book.Spine))
	}
	if _, statErr := os.Stat(filepath.Join(outputDir, "original.pdf")); statErr != nil {
		t.Fatalf("expected original.pdf fallback copy, got error: %v", statErr)
	}
}

func TestExtract_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, "notapdf.pdf")

	// Write garbage data
	if err := os.WriteFile(pdfPath, []byte("this is not a PDF"), 0o644); err != nil {
		t.Fatal(err)
	}

	outputDir := filepath.Join(tmpDir, "output")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := Extract(pdfPath, outputDir)
	if err == nil {
		t.Error("expected error for invalid PDF, got nil")
	}
}

func TestExtract_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := Extract(filepath.Join(tmpDir, "nope.pdf"), outputDir)
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestNeedsPDFToTextPathStaging(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{`C:\Books\plain.pdf`, false},
		{`C:\Books\with space.pdf`, false},
		{`C:\Books\Auntie’s Story.pdf`, true},
		{`C:\Books\M502e – Story.pdf`, true},
	}

	for _, tt := range tests {
		if got := needsPDFToTextPathStaging(tt.path); got != tt.want {
			t.Errorf("needsPDFToTextPathStaging(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestStagePDFForPDFToText(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "M502e – Story.pdf")
	content := []byte("fake pdf content")
	if err := os.WriteFile(srcPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	stagedPath, cleanup, err := stagePDFForPDFToText(srcPath)
	if err != nil {
		t.Fatalf("stagePDFForPDFToText() error: %v", err)
	}
	defer cleanup()

	if filepath.Base(stagedPath) != "input.pdf" {
		t.Fatalf("expected staged filename input.pdf, got %q", filepath.Base(stagedPath))
	}
	for _, r := range stagedPath {
		if r > 127 {
			t.Fatalf("expected ASCII-only staged path, got %q", stagedPath)
		}
	}

	stagedContent, err := os.ReadFile(stagedPath)
	if err != nil {
		t.Fatalf("read staged file: %v", err)
	}
	if string(stagedContent) != string(content) {
		t.Fatalf("staged file content mismatch: got %q want %q", stagedContent, content)
	}
}

func TestFlipImageFileVertically(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "sample.tif")

	img := image.NewNRGBA(image.Rect(0, 0, 1, 2))
	img.Set(0, 0, color.NRGBA{R: 255, A: 255})
	img.Set(0, 1, color.NRGBA{B: 255, A: 255})

	file, err := os.Create(imgPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := tiff.Encode(file, img, nil); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	if err := flipImageFileVertically(imgPath); err != nil {
		t.Fatalf("flipImageFileVertically() error: %v", err)
	}

	flippedFile, err := os.Open(imgPath)
	if err != nil {
		t.Fatal(err)
	}
	defer flippedFile.Close()
	flipped, err := tiff.Decode(flippedFile)
	if err != nil {
		t.Fatal(err)
	}

	top := color.NRGBAModel.Convert(flipped.At(0, 0)).(color.NRGBA)
	bottom := color.NRGBAModel.Convert(flipped.At(0, 1)).(color.NRGBA)
	if top.B != 255 || bottom.R != 255 {
		t.Fatalf("expected image to be vertically flipped, got top=%v bottom=%v", top, bottom)
	}
}

func TestImageHTMLClassAttr(t *testing.T) {
	if got := imageHTMLClassAttr("pdf_images/sample.png"); got != "" {
		t.Fatalf("expected no class for png, got %q", got)
	}
	if got := imageHTMLClassAttr("pdf_images/sample.tif"); got != ` class="pdf-flip-y"` {
		t.Fatalf("expected tif flip class, got %q", got)
	}
}
