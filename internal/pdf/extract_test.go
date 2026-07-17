package pdf

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pdflib "github.com/ledongthuc/pdf"

	"github.com/go-pdf/fpdf"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"golang.org/x/image/tiff"
)

// The page counter used to report the number of *emitted* pages under a "with text" label,
// so a scan with no text layer at all announced "Pages: 6 (with text: 6)" directly beneath
// the warning that no text had been extracted. Each kind of page is now counted separately
// and named for what it is.
func TestPageCountSummary(t *testing.T) {
	tests := []struct {
		name                            string
		totalPages, withText, generated int
		want                            string
	}{
		{"prose: every page has text", 300, 300, 300, "with text: 300"},
		{"a scan: no text layer anywhere", 6, 0, 6, "with text: 0, image-only: 6"},
		{"mixed: text, plates and blanks", 300, 280, 295, "with text: 280, image-only: 15, empty: 5"},
		{"blank pages are named, not hidden", 10, 4, 4, "with text: 4, empty: 6"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pageCountSummary(tt.totalPages, tt.withText, tt.generated); got != tt.want {
				t.Errorf("pageCountSummary(%d, %d, %d)\n got: %q\nwant: %q",
					tt.totalPages, tt.withText, tt.generated, got, tt.want)
			}
		})
	}
}

// selectPageImages must (1) drop /Thumb previews, which pdfcpu returns among a page's
// images and which render as a postage stamp where the page should be, and (2) collapse
// the same picture embedded at two resolutions down to the largest, so a scanned page is
// not shown twice - while leaving a genuinely composed page (differently-shaped images)
// fully intact.
func TestSelectPageImages(t *testing.T) {
	// The measured duplicate from pdf-1page-blackletter_Plague-Proclamation-1625.pdf:
	// the same page as 1455x2065 and 4363x6193 (exactly 3x).
	small := model.Image{ObjNr: 1, Width: 1455, Height: 2065}
	big := model.Image{ObjNr: 2, Width: 4363, Height: 6193}
	thumb := model.Image{ObjNr: 3, Width: 128, Height: 104, Thumb: true}

	t.Run("drops thumbnails", func(t *testing.T) {
		kept, thumbs, dups := selectPageImages(map[int]model.Image{3: thumb, 2: big})
		if thumbs != 1 || dups != 0 {
			t.Fatalf("thumbs=%d dups=%d, want thumbs=1 dups=0", thumbs, dups)
		}
		if len(kept) != 1 || kept[0].ObjNr != 2 {
			t.Fatalf("kept=%v, want only the page raster (objNr 2)", kept)
		}
	})

	t.Run("collapses a proportional-scale duplicate to the largest", func(t *testing.T) {
		kept, thumbs, dups := selectPageImages(map[int]model.Image{1: small, 2: big})
		if thumbs != 0 || dups != 1 {
			t.Fatalf("thumbs=%d dups=%d, want thumbs=0 dups=1", thumbs, dups)
		}
		if len(kept) != 1 || kept[0].ObjNr != 2 {
			t.Fatalf("kept=%v, want only the largest raster (objNr 2)", kept)
		}
	})

	t.Run("keeps every image of a composed page", func(t *testing.T) {
		portrait := model.Image{ObjNr: 1, Width: 600, Height: 900}
		landscape := model.Image{ObjNr: 2, Width: 900, Height: 600}
		square := model.Image{ObjNr: 3, Width: 500, Height: 500}
		kept, thumbs, dups := selectPageImages(map[int]model.Image{1: portrait, 2: landscape, 3: square})
		if thumbs != 0 || dups != 0 {
			t.Fatalf("thumbs=%d dups=%d, want both 0", thumbs, dups)
		}
		if len(kept) != 3 {
			t.Fatalf("kept %d images, want 3 (composed page left intact)", len(kept))
		}
	})

	t.Run("unknown dimensions are never treated as duplicates", func(t *testing.T) {
		// The real (non-stub) extraction can leave Width/Height at zero; such images
		// must pass through rather than collapse into each other.
		a := model.Image{ObjNr: 1}
		b := model.Image{ObjNr: 2}
		kept, _, dups := selectPageImages(map[int]model.Image{1: a, 2: b})
		if dups != 0 || len(kept) != 2 {
			t.Fatalf("kept=%d dups=%d, want kept=2 dups=0", len(kept), dups)
		}
	})
}

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

func TestParsePDFLayoutPage_DoubleSpacedMerge(t *testing.T) {
	// Simulate the pdftotext artifact where a blank line follows every wrapped
	// line, so each physical line is its own block. Paragraph openings carry a
	// leading indent (zero-width-space marker or spaces); continuations sit at the
	// left margin and must be glued back on.
	const zwsp = "\u200b"
	page := strings.Join([]string{
		"Body line one that wraps and",
		"",
		"continues here at the margin",
		"",
		zwsp + "   A brand new indented paragraph",
		"",
		"that also wraps onto a second line",
		"",
		"            Centered Heading",
	}, "\n")

	items := parsePDFLayoutPage(page)
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3:\n%#v", len(items), items)
	}
	if items[0].tag != "p" || items[0].text != "Body line one that wraps and continues here at the margin" {
		t.Errorf("item 0 = %+v, want merged body paragraph", items[0])
	}
	if items[1].tag != "p" || items[1].text != "A brand new indented paragraph that also wraps onto a second line" {
		t.Errorf("item 1 = %+v, want merged indented paragraph", items[1])
	}
	if items[2].tag != "h2" || items[2].text != "Centered Heading" {
		t.Errorf("item 2 = %+v, want centered h2 kept separate", items[2])
	}
	for _, it := range items {
		if strings.Contains(it.text, zwsp) {
			t.Errorf("zero-width-space marker leaked into output: %q", it.text)
		}
	}
}

func TestParsePDFLayoutPage_NormalNotMerged(t *testing.T) {
	// A normally-spaced page keeps whole paragraphs in multi-line blocks separated
	// by blank lines. These must stay distinct paragraphs, never merged together.
	blocks := make([]string, 0, 6)
	for i := 0; i < 6; i++ {
		blocks = append(blocks, "Paragraph line one here\nand its wrapped second line")
	}
	page := strings.Join(blocks, "\n\n")

	items := parsePDFLayoutPage(page)
	if len(items) != 6 {
		t.Fatalf("got %d items, want 6 (no merging on normal layout)", len(items))
	}
	for i, it := range items {
		if it.text != "Paragraph line one here and its wrapped second line" {
			t.Errorf("item %d = %q, want unmerged multi-line paragraph", i, it.text)
		}
	}
}

func TestBuildPageHTML(t *testing.T) {
	html := buildPageHTML(t.TempDir(), "Test Book", 3, 10, "Hello World\nSecond line", nil)

	// Check structure
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("missing DOCTYPE")
	}
	if !strings.Contains(html, "<title>Test Book — Page 3</title>") {
		t.Error("missing/wrong title")
	}
	// Page number is shown in the injected navbar, not in a body header.
	if strings.Contains(html, "pdf-page-header") {
		t.Error("body page header should be removed (navbar shows the page number)")
	}
	if !strings.Contains(html, "<p>Hello World</p>") {
		t.Error("missing first paragraph")
	}
	if !strings.Contains(html, "<p>Second line</p>") {
		t.Error("missing second paragraph")
	}
}

// A page holding one image and no text is a scan, and it gets a box sized to the window
// rather than to the text measure. The three caps are what make "one page, one screen"
// work without upscaling the image into mush.
func TestBuildPageHTMLSizesAFullPageScan(t *testing.T) {
	dir := t.TempDir()
	writeTestPNG(t, filepath.Join(dir, "scan.png"), 1600, 1200)

	html := buildPageHTML(dir, "Book", 1, 1, "", []string{"scan.png"})
	if !strings.Contains(html, `class="pdf-images pdf-page-scan"`) {
		t.Errorf("text-less single-image page not marked as a scan:\n%s", html)
	}
	// 92vh of height at 1600/1200 allows 122.7vh of width.
	if !strings.Contains(html, `style="width:min(1600px,96vw,122.7vh)"`) {
		t.Errorf("scan box not sized from the image:\n%s", html)
	}
}

// A page with text alongside its image is a normal page: it keeps the reading measure,
// because widening it would be widening a column of text.
func TestBuildPageHTMLLeavesTextPagesAlone(t *testing.T) {
	dir := t.TempDir()
	writeTestPNG(t, filepath.Join(dir, "fig.png"), 400, 300)

	html := buildPageHTML(dir, "Book", 1, 1, "Some text", []string{"fig.png"})
	if strings.Contains(html, "pdf-page-scan") {
		t.Errorf("a page with text was treated as a scan:\n%s", html)
	}
}

func writeTestPNG(t *testing.T, path string, w, h int) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatalf("encode: %v", err)
	}
}

func TestBuildPageHTML_EscapesHTML(t *testing.T) {
	html := buildPageHTML(t.TempDir(), "Book <script>", 1, 1, "<b>bold</b>", nil)
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

func TestClassifyBlock(t *testing.T) {
	cases := []struct {
		name          string
		text          string
		leadingSpaces int
		want          string
	}{
		{"latin all-caps short", "LITTLE TOKYO", 0, "h2"},
		{"cyrillic all-caps short", "ГЛАВА ПЕРВАЯ", 0, "h2"},
		{"cyrillic all-caps with digit", "ГЛАВА 1", 0, "h2"},
		{"mixed case not heading", "Little Tokyo", 0, "p"},
		{"cyrillic mixed case not heading", "Глава первая", 0, "p"},
		{"digits only not heading", "12345", 0, "p"},
		{"centered short heading", "A Quiet Chapter", 12, "h2"},
		{"centered medium heading", "A Somewhat Longer Centered Title Line That Runs Here Now", 12, "h3"},
		{"plain body", "This is an ordinary body line of text.", 0, "p"},
		{"all-caps but long stays body", "THIS ALL CAPS LINE HAS FAR TOO MANY WORDS TO BE A HEADING LINE", 0, "p"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := classifyBlock(c.text, c.leadingSpaces); got != c.want {
				t.Errorf("classifyBlock(%q, %d) = %q, want %q", c.text, c.leadingSpaces, got, c.want)
			}
		})
	}
}
