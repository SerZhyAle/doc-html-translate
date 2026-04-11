package htmlproc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testHTML = `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<h1>Hello World</h1>
<p>This is a paragraph.</p>
<script>var x = 1;</script>
<style>body { color: red; }</style>
<pre>code block should not translate</pre>
<p>Another <code>inline_code</code> paragraph text.</p>
</body>
</html>`

func TestExtractTexts(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.html")
	if err := os.WriteFile(filePath, []byte(testHTML), 0o644); err != nil {
		t.Fatal(err)
	}

	segments, doc, err := ExtractTexts(filePath)
	if err != nil {
		t.Fatalf("ExtractTexts failed: %v", err)
	}

	if doc == nil {
		t.Fatal("doc is nil")
	}

	// Collect texts
	var texts []string
	for _, seg := range segments {
		texts = append(texts, seg.Text)
	}

	// "Test" (title), "Hello World", "This is a paragraph.", "Another", "paragraph text."
	// Script, style, pre, code content should be EXCLUDED
	for _, s := range texts {
		if s == "var x = 1;" {
			t.Error("script content should be excluded")
		}
		if s == "body { color: red; }" {
			t.Error("style content should be excluded")
		}
		if s == "code block should not translate" {
			t.Error("pre content should be excluded")
		}
		if s == "inline_code" {
			t.Error("code content should be excluded")
		}
	}

	// Check that translatable text IS present
	found := map[string]bool{}
	for _, s := range texts {
		found[s] = true
	}
	if !found["Hello World"] {
		t.Error("expected 'Hello World' in extracted texts")
	}
	if !found["This is a paragraph."] {
		t.Error("expected 'This is a paragraph.' in extracted texts")
	}
}

func TestReplaceTextsAndRender(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.html")
	if err := os.WriteFile(filePath, []byte(testHTML), 0o644); err != nil {
		t.Fatal(err)
	}

	segments, doc, err := ExtractTexts(filePath)
	if err != nil {
		t.Fatal(err)
	}

	// Create fake translations
	translated := make([]string, len(segments))
	for i, seg := range segments {
		translated[i] = "T:" + seg.Text
	}

	ReplaceTexts(segments, translated)

	outPath := filepath.Join(tmpDir, "out.html")
	if err := RenderToFile(doc, outPath); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "T:Hello World") {
		t.Error("expected translated 'T:Hello World' in output")
	}
	if !strings.Contains(content, "T:This is a paragraph.") {
		t.Error("expected translated paragraph in output")
	}
	// Script content should remain unchanged
	if !strings.Contains(content, "var x = 1;") {
		t.Error("script content should remain unchanged")
	}
}
