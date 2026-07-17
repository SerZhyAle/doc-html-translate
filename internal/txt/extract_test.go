package txt_test

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

	"golang.org/x/text/encoding/charmap"

	"doc-html-translate/internal/txt"
)

// encodeUTF16 builds the bytes Notepad writes for its "Unicode" and "Unicode big endian"
// options: a BOM followed by 2-byte code units.
func encodeUTF16(t *testing.T, s string, order binary.ByteOrder) []byte {
	t.Helper()
	units := append([]uint16{0xFEFF}, utf16.Encode([]rune(s))...)
	buf := make([]byte, len(units)*2)
	for i, u := range units {
		order.PutUint16(buf[i*2:], u)
	}
	return buf
}

// extractToPage runs a conversion over raw bytes and returns page_001.html.
func extractToPage(t *testing.T, raw []byte) string {
	t.Helper()
	dir := t.TempDir()
	txtPath := filepath.Join(dir, "book.txt")
	if err := os.WriteFile(txtPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := txt.Extract(txtPath, outDir); err != nil {
		t.Fatalf("Extract: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	if err != nil {
		t.Fatalf("page_001.html: %v", err)
	}
	return string(data)
}

// UTF-16 was read as if every byte were a character, so "Это обычное" came out as
// "-B> >1KG=>5" - the low byte of each code unit. ASCII hid the bug: ASCII-in-UTF-16 is
// exactly "character, NUL, character, NUL", and the single-page merge strips control
// characters on the way past, so only Cyrillic (and -multipage) ever showed it. Both
// alphabets and both byte orders are checked here, on the page the extractor writes,
// before any merge can launder it.
func TestExtractDecodesUTF16(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		order binary.ByteOrder
	}{
		{"LE, Cyrillic - the case that was mojibake", "Это обычное предложение на русском языке.", binary.LittleEndian},
		{"BE, Cyrillic", "Это обычное предложение на русском языке.", binary.BigEndian},
		{"LE, ASCII - only ever correct by accident", "The Project Gutenberg eBook of something.", binary.LittleEndian},
		{"BE, ASCII", "The Project Gutenberg eBook of something.", binary.BigEndian},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := extractToPage(t, encodeUTF16(t, tt.text, tt.order))
			if !strings.Contains(page, tt.text) {
				t.Errorf("decoded text missing from the page.\nwant: %q", tt.text)
			}
			if strings.ContainsRune(page, 0) {
				t.Error("raw UTF-16 code units reached the page: it carries NUL bytes")
			}
			if strings.ContainsRune(page, 0xFFFD) {
				t.Error("page carries U+FFFD: the bytes were decoded as the wrong encoding")
			}
		})
	}
}

// A UTF-8 BOM used to survive into the first paragraph as an invisible character. Cosmetic,
// but the same root gap - nothing looked at the leading bytes. Note the extension never had
// this one: TextDecoder strips a BOM unless asked not to.
func TestExtractStripsUTF8BOM(t *testing.T) {
	page := extractToPage(t, append([]byte{0xEF, 0xBB, 0xBF}, []byte("Первый абзац.\n\nВторой абзац.")...))
	if strings.ContainsRune(page, 0xFEFF) {
		t.Error("the UTF-8 BOM leaked into the page")
	}
	if !strings.Contains(page, "<p>Первый абзац.</p>") {
		t.Error("the first paragraph should start at its first real character")
	}
}

// A .txt in a pre-Unicode Cyrillic code page - a DOS-era archive, a pre-UTF-8 Russian web
// page - was decoded as UTF-8 and came out as mojibake. detectLegacy now picks the code page
// from the byte distribution. cp1251 and koi8-r remap the same byte range, so choosing between
// them is the crux: a naive letter count can't (they yield nearly the same number of Cyrillic
// letters), which is why the fit is frequency-weighted.
func TestExtractDecodesLegacyCyrillic(t *testing.T) {
	const text = "Лицензионное соглашение на использование программы. Все права защищены."
	for _, enc := range []struct {
		name string
		cm   *charmap.Charmap
	}{
		{"windows-1251", charmap.Windows1251},
		{"koi8-r", charmap.KOI8R},
		{"cp866", charmap.CodePage866},
		{"iso-8859-5", charmap.ISO8859_5},
	} {
		t.Run(enc.name, func(t *testing.T) {
			raw, err := enc.cm.NewEncoder().Bytes([]byte(text))
			if err != nil {
				t.Fatalf("encode fixture: %v", err)
			}
			page := extractToPage(t, raw)
			if !strings.Contains(page, text) {
				t.Errorf("%s text not decoded back.\nwant: %q", enc.name, text)
			}
		})
	}
}

// A non-Russian legacy file (French Latin-1) is not valid UTF-8 either, so it reaches the same
// path - but none of the Cyrillic candidates is right, so it must pass through unchanged rather
// than be Cyrillized into confident nonsense.
func TestExtractLeavesNonCyrillicLegacyAlone(t *testing.T) {
	raw, err := charmap.ISO8859_1.NewEncoder().Bytes([]byte("Élément très cher, à côté de l'hôtel où nous étions cet été."))
	if err != nil {
		t.Fatal(err)
	}
	page := extractToPage(t, raw)
	// The bytes pass through as-is; what matters is that we did not invent Cyrillic. The word
	// "hôtel" survives at least as "h" + something, never as Russian letters.
	if strings.ContainsAny(page, "абвгдежзийклмнопрстуфхцчшщъыьэюяАБВГДЕЖЗ") {
		t.Errorf("non-Russian legacy text was wrongly decoded into Cyrillic:\n%s", firstParagraph(page))
	}
}

// firstParagraph pulls the first <p> body out of a page, for readable failure messages.
func firstParagraph(page string) string {
	i := strings.Index(page, "<p>")
	if i < 0 {
		return page
	}
	j := strings.Index(page[i:], "</p>")
	if j < 0 {
		return page[i:]
	}
	return page[i : i+j]
}

// The common case must be provably untouched by the decode path.
func TestExtractPlainUTF8Unchanged(t *testing.T) {
	const text = "Обычный UTF-8 без BOM.\n\nВторой абзац."
	page := extractToPage(t, []byte(text))
	for _, want := range []string{"<p>Обычный UTF-8 без BOM.</p>", "<p>Второй абзац.</p>"} {
		if !strings.Contains(page, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestExtract_Basic(t *testing.T) {
	dir := t.TempDir()
	content := "First paragraph line one\nFirst paragraph line two\n\nSecond paragraph\n\nThird paragraph"
	txtPath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(txtPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	book, err := txt.Extract(txtPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}

	if book.Title != "test" {
		t.Errorf("expected title 'test', got %q", book.Title)
	}
	// 3 paragraphs < paragraphsPerPage → 1 page
	if len(book.Spine) != 1 {
		t.Errorf("expected 1 page, got %d", len(book.Spine))
	}

	data, err := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	if err != nil {
		t.Fatalf("page_001.html not found: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "First paragraph line one First paragraph line two") {
		t.Errorf("expected merged paragraph in output")
	}
	if !strings.Contains(body, "Third paragraph") {
		t.Errorf("expected third paragraph in output")
	}
}

func TestExtract_Empty(t *testing.T) {
	dir := t.TempDir()
	txtPath := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(txtPath, []byte("   \n\n   \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := txt.Extract(txtPath, dir)
	if err == nil {
		t.Error("expected error for empty file, got nil")
	}
}

func TestExtract_Pagination(t *testing.T) {
	dir := t.TempDir()
	// 35 paragraphs separated by blank lines → 2 pages (>30)
	var lines []string
	for i := 1; i <= 35; i++ {
		lines = append(lines, fmt.Sprintf("Paragraph number %d with some text here.", i))
		lines = append(lines, "")
	}
	txtPath := filepath.Join(dir, "big.txt")
	if err := os.WriteFile(txtPath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	book, err := txt.Extract(txtPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if len(book.Spine) != 2 {
		t.Errorf("expected 2 pages, got %d", len(book.Spine))
	}
	if _, err := os.Stat(filepath.Join(outDir, "page_002.html")); err != nil {
		t.Error("page_002.html not created")
	}
}

func TestExtract_FileNotFound(t *testing.T) {
	_, err := txt.Extract("/nonexistent/path/file.txt", t.TempDir())
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestExtract_HtmlEscaping(t *testing.T) {
	dir := t.TempDir()
	content := "Hello <World> & \"quotes\""
	txtPath := filepath.Join(dir, "escape.txt")
	if err := os.WriteFile(txtPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	_, err := txt.Extract(txtPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	body := string(data)
	if strings.Contains(body, "<World>") {
		t.Error("HTML not escaped: raw <World> found in output")
	}
	if !strings.Contains(body, "&lt;World&gt;") {
		t.Error("expected &lt;World&gt; in HTML-escaped output")
	}
}

func TestExtract_LinuxNoBlankLines(t *testing.T) {
	// File with only single \n between lines and no blank lines at all.
	// Each line must become its own paragraph.
	dir := t.TempDir()
	content := "Line one\nLine two\nLine three\nLine four"
	txtPath := filepath.Join(dir, "linux.txt")
	if err := os.WriteFile(txtPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	book, err := txt.Extract(txtPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	// 4 lines → 4 paragraphs → 1 page
	if len(book.Spine) != 1 {
		t.Errorf("expected 1 page, got %d", len(book.Spine))
	}
	data, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	body := string(data)
	// Count <p> tags — must be 4 separate paragraphs
	count := strings.Count(body, "<p>")
	if count != 4 {
		t.Errorf("expected 4 <p> elements for 4 lines, got %d\n%s", count, body)
	}
	if strings.Contains(body, "Line one Line two") {
		t.Error("lines must not be merged into one paragraph")
	}
}

func TestExtract_CRLFLineEndings(t *testing.T) {
	// Windows \r\n with blank lines — must behave like the baseline test.
	dir := t.TempDir()
	content := "First paragraph\r\n\r\nSecond paragraph\r\n\r\nThird paragraph\r\n"
	txtPath := filepath.Join(dir, "windows.txt")
	_ = os.WriteFile(txtPath, []byte(content), 0o644)
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	book, err := txt.Extract(txtPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if len(book.Spine) != 1 {
		t.Errorf("expected 1 page, got %d", len(book.Spine))
	}
	data, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	count := strings.Count(string(data), "<p>")
	if count != 3 {
		t.Errorf("expected 3 paragraphs from CRLF file, got %d", count)
	}
}

func TestExtract_OldMacCRLineEndings(t *testing.T) {
	// Old Mac \r only — no blank lines, each \r-line becomes its own paragraph.
	dir := t.TempDir()
	content := "Alpha\rBeta\rGamma"
	txtPath := filepath.Join(dir, "mac.txt")
	_ = os.WriteFile(txtPath, []byte(content), 0o644)
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	book, err := txt.Extract(txtPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	_ = book
	data, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	count := strings.Count(string(data), "<p>")
	if count != 3 {
		t.Errorf("expected 3 paragraphs from old-Mac CR file, got %d", count)
	}
}

func TestExtract_UnicodeParagraphSeparators(t *testing.T) {
	dir := t.TempDir()
	content := "First paragraph\u2028\u2028Second paragraph\u2029\u2029Third paragraph"
	txtPath := filepath.Join(dir, "unicode-separators.txt")
	_ = os.WriteFile(txtPath, []byte(content), 0o644)
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	_, err := txt.Extract(txtPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	count := strings.Count(string(data), "<p>")
	if count != 3 {
		t.Errorf("expected 3 paragraphs from Unicode separators, got %d", count)
	}
	if !strings.Contains(string(data), "First paragraph") || !strings.Contains(string(data), "Third paragraph") {
		t.Errorf("expected Unicode-separated paragraphs to survive conversion")
	}
}

func TestExtract_ControlCharacterLineSeparators(t *testing.T) {
	dir := t.TempDir()
	content := "Alpha\u0085Beta\vGamma\fDelta"
	txtPath := filepath.Join(dir, "control-separators.txt")
	_ = os.WriteFile(txtPath, []byte(content), 0o644)
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	_, err := txt.Extract(txtPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	count := strings.Count(string(data), "<p>")
	if count != 4 {
		t.Errorf("expected 4 paragraphs from control-character separators, got %d", count)
	}
}
