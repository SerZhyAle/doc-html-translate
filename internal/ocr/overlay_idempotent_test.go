package ocr

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	gohtml "golang.org/x/net/html"
)

// A page overlaid twice - a re-run, or a converted page fed back in - must come out the same as
// after one pass: no second stylesheet or script, no plates nested over plates.
func TestOverlayTwiceEqualsOnce(t *testing.T) {
	saved := recognizeImage
	t.Cleanup(func() { recognizeImage = saved })
	recognizeImage = func(_ context.Context, _, _, _, _ string) (Result, error) {
		return Result{Width: 100, Height: 200, Blocks: []Block{
			{Text: "Hello there, reader", X0: 1, Y0: 1, X1: 90, Y1: 20, LineH: 10},
		}}, nil
	}
	_, paths := writeBook(t, []string{"p1.png"}, `<p>text</p><img src="p1.png">`)

	run := func() []byte {
		t.Helper()
		stats := OverlayBook(context.Background(), "tesseract", paths, "eng", "", true, nil)
		if len(stats.Failed) != 0 {
			t.Fatalf("overlay failed: %+v", stats.Failed)
		}
		b, err := os.ReadFile(paths[0])
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	once := run()
	if !bytes.Contains(once, []byte("ocr-fig")) {
		t.Fatalf("the first pass added no overlay:\n%s", once)
	}
	twice := run()
	if !bytes.Equal(once, twice) {
		t.Errorf("a second pass changed the page:\nonce:  %s\ntwice: %s", once, twice)
	}
}

// A page written before the marker existed carries the stylesheet and script without it; they
// are recognized by their content and not stacked.
func TestEnsureAssetsRecognizesAnUnmarkedEarlierPass(t *testing.T) {
	page := "<html><head><style>" + ocrCSS + "</style></head><body><p>x</p><script>" + ocrScript + "</script></body></html>"
	doc, err := gohtml.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	ensureStyle(doc)
	ensureScript(doc)
	var buf bytes.Buffer
	if err := gohtml.Render(&buf, doc); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if n := strings.Count(out, "<style"); n != 1 {
		t.Errorf("%d stylesheets, want 1", n)
	}
	if n := strings.Count(out, "<script"); n != 1 {
		t.Errorf("%d scripts, want 1", n)
	}
}

// An image src is a URL: a ?query or #fragment is not part of the file name (finding O11).
func TestCollectBookImagesStripsQueryAndFragment(t *testing.T) {
	dir, paths := writeBook(t,
		[]string{"a.png", "b.png"},
		`<img src="a.png?v=2"><img src="b.png#page=1">`,
	)
	got := collectBookImages(paths)
	want := []string{filepath.Join(dir, "a.png"), filepath.Join(dir, "b.png")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("query/fragment not stripped:\n got  %v\n want %v", got, want)
	}
}
