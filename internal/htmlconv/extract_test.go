package htmlconv_test

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"doc-html-translate/internal/htmlconv"
)

func TestExtract_BasicHTML(t *testing.T) {
	dir := t.TempDir()
	content := `<!DOCTYPE html>
<html>
<head><title>My Page</title></head>
<body>
<h1>Hello World</h1>
<p>Some content here.</p>
</body>
</html>`
	htmlPath := filepath.Join(dir, "test.html")
	if err := os.WriteFile(htmlPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	book, err := htmlconv.Extract(htmlPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if book.Title != "My Page" {
		t.Errorf("expected title 'My Page', got %q", book.Title)
	}
	if len(book.Spine) != 1 {
		t.Errorf("expected 1 page, got %d", len(book.Spine))
	}
	data, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	body := string(data)
	if !strings.Contains(body, "Hello World") {
		t.Error("expected 'Hello World' in output")
	}
	if !strings.Contains(body, "Some content here.") {
		t.Error("expected content in output")
	}
}

func TestExtract_HTMLNoTitle(t *testing.T) {
	dir := t.TempDir()
	content := "<p>Just a paragraph.</p>"
	htmlPath := filepath.Join(dir, "notitle.htm")
	_ = os.WriteFile(htmlPath, []byte(content), 0o644)
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	book, err := htmlconv.Extract(htmlPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	// Falls back to filename
	if book.Title != "notitle" {
		t.Errorf("expected fallback title 'notitle', got %q", book.Title)
	}
}

func TestExtract_HTMLEmpty(t *testing.T) {
	dir := t.TempDir()
	htmlPath := filepath.Join(dir, "empty.html")
	_ = os.WriteFile(htmlPath, []byte("   \n\n  "), 0o644)
	_, err := htmlconv.Extract(htmlPath, dir)
	if err == nil {
		t.Error("expected error for empty HTML, got nil")
	}
}

func TestExtract_HTMLFileNotFound(t *testing.T) {
	_, err := htmlconv.Extract("/nonexistent/file.html", t.TempDir())
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// A "Save page as" HTML with images next to it must carry those images into the output so
// they still display. Before this, the refs were kept but no file was copied, so every picture
// came out broken.
func TestExtract_HTMLLocalImagesCopied(t *testing.T) {
	dir := t.TempDir()
	imagesDir := filepath.Join(dir, "images")
	_ = os.MkdirAll(imagesDir, 0o755)
	writePNG(t, filepath.Join(imagesDir, "cover.png"))
	writePNG(t, filepath.Join(dir, "banner.png"))

	content := `<html><head><title>Illustrated</title></head><body>
<p>text</p>
<img src="images/cover.png">
<img src="banner.png">
<img src="images/cover.png">
</body></html>`
	htmlPath := filepath.Join(dir, "page.html")
	_ = os.WriteFile(htmlPath, []byte(content), 0o644)
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	if _, err := htmlconv.Extract(htmlPath, outDir); err != nil {
		t.Fatalf("Extract: %v", err)
	}
	page, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	body := string(page)

	// Both distinct files were copied and every ref resolves from the output folder.
	for _, ref := range refs(body) {
		if _, err := os.Stat(filepath.Join(outDir, ref)); err != nil {
			t.Errorf("output references %q which is not in the output dir: %v", ref, err)
		}
	}
	// The same source file referenced twice was copied once (dedup).
	pngs, _ := filepath.Glob(filepath.Join(outDir, "*.png"))
	if len(pngs) != 2 {
		t.Errorf("expected 2 copied image files (cover + banner, cover deduped), got %d: %v", len(pngs), pngs)
	}
}

// A src that escapes the source directory subtree must NOT copy an arbitrary disk file into the
// output; it is left as-is (and will simply not resolve).
func TestExtract_HTMLPathTraversalRefused(t *testing.T) {
	root := t.TempDir()
	secret := filepath.Join(root, "secret.png")
	writePNG(t, secret)

	srcDir := filepath.Join(root, "sub")
	_ = os.MkdirAll(srcDir, 0o755)
	content := `<html><body><img src="../secret.png"></body></html>`
	htmlPath := filepath.Join(srcDir, "page.html")
	_ = os.WriteFile(htmlPath, []byte(content), 0o644)
	outDir := filepath.Join(root, "out")
	_ = os.MkdirAll(outDir, 0o755)

	if _, err := htmlconv.Extract(htmlPath, outDir); err != nil {
		t.Fatalf("Extract: %v", err)
	}
	// secret.png must not have been copied out.
	if _, err := os.Stat(filepath.Join(outDir, "secret.png")); err == nil {
		t.Error("a ../ traversal copied a file from outside the source subtree into the output")
	}
	page, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	if !strings.Contains(string(page), "../secret.png") {
		t.Error("the escaping ref should be left untouched (visibly broken), not rewritten")
	}
}

// Remote and inline srcs are not ours to copy - they must pass through unchanged.
func TestExtract_HTMLRemoteAndDataLeftAlone(t *testing.T) {
	dir := t.TempDir()
	content := `<html><body>
<img src="https://example.com/a.png">
<img src="data:image/png;base64,iVBORw0KGgo=">
</body></html>`
	htmlPath := filepath.Join(dir, "page.html")
	_ = os.WriteFile(htmlPath, []byte(content), 0o644)
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	if _, err := htmlconv.Extract(htmlPath, outDir); err != nil {
		t.Fatalf("Extract: %v", err)
	}
	page, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	body := string(page)
	if !strings.Contains(body, "https://example.com/a.png") {
		t.Error("remote src should pass through unchanged")
	}
	if !strings.Contains(body, "data:image/png;base64,") {
		t.Error("data: src should pass through unchanged")
	}
}

// A local image that does not exist is left as a visibly broken ref, not dropped - the
// conversion still succeeds.
func TestExtract_HTMLDanglingImageLeftVisible(t *testing.T) {
	dir := t.TempDir()
	content := `<html><body><p>text</p><img src="missing.png"></body></html>`
	htmlPath := filepath.Join(dir, "page.html")
	_ = os.WriteFile(htmlPath, []byte(content), 0o644)
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	if _, err := htmlconv.Extract(htmlPath, outDir); err != nil {
		t.Fatalf("Extract: %v", err)
	}
	page, _ := os.ReadFile(filepath.Join(outDir, "page_001.html"))
	if !strings.Contains(string(page), "missing.png") {
		t.Error("a dangling ref should remain (visibly broken), not be dropped")
	}
}

// refs pulls the src values out of the output HTML.
func refs(body string) []string {
	var out []string
	for {
		i := strings.Index(body, `src="`)
		if i < 0 {
			break
		}
		body = body[i+5:]
		j := strings.IndexByte(body, '"')
		if j < 0 {
			break
		}
		out = append(out, body[:j])
		body = body[j:]
	}
	return out
}

// writePNG writes a tiny valid PNG file.
func writePNG(t *testing.T, path string) {
	t.Helper()
	// 1x1 PNG.
	const b64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+M8AAAMBAQBS5f4CAAAAAElFTkSuQmCC"
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestExtract_HTMExtension(t *testing.T) {
	dir := t.TempDir()
	content := `<html><head><title>HTM Test</title></head><body><p>HTM content</p></body></html>`
	htmlPath := filepath.Join(dir, "test.htm")
	_ = os.WriteFile(htmlPath, []byte(content), 0o644)
	outDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(outDir, 0o755)

	book, err := htmlconv.Extract(htmlPath, outDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if book.Title != "HTM Test" {
		t.Errorf("expected title 'HTM Test', got %q", book.Title)
	}
}
