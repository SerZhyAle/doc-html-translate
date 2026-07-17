package ocr

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	gohtml "golang.org/x/net/html"
)

// writeFiles creates image stubs (content irrelevant - collectOverlayJobs only Stat's them)
// and HTML pages in a fresh temp dir, returning the dir and the page paths in order.
func writeBook(t *testing.T, images []string, pages ...string) (string, []string) {
	t.Helper()
	dir := t.TempDir()
	for _, n := range images {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var paths []string
	for i, body := range pages {
		p := filepath.Join(dir, fileNameForPage(i))
		if err := os.WriteFile(p, []byte("<html><body>"+body+"</body></html>"), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	return dir, paths
}

func fileNameForPage(i int) string {
	return "page_" + string(rune('0'+i)) + ".html"
}

// TestCollectBookImagesBatchesAndDedupes guards the batching boundary: images are collected
// across every content file (not one at a time), deduped by absolute path, kept in first-seen
// order, and external / missing srcs are filtered out. This is what lets the phase-2 pool run
// at full width over the whole book in -multipage mode instead of one page's one image.
func TestCollectBookImagesBatchesAndDedupes(t *testing.T) {
	dir, paths := writeBook(t,
		[]string{"p1.png", "p2.png", "shared.png"},
		`<img src="p1.png"><img src="shared.png">`,
		`<img src="p2.png"><img src="shared.png"><img src="https://x/ext.png"><img src="missing.png">`,
	)

	got := collectBookImages(paths)
	want := []string{
		filepath.Join(dir, "p1.png"),
		filepath.Join(dir, "shared.png"), // deduped: appears in both files, listed once
		filepath.Join(dir, "p2.png"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("batching/dedup wrong:\n got  %v\n want %v", got, want)
	}
}

// TestApplyOverlays checks phase 3: each image is wrapped from its precomputed recognition,
// with the three outcomes (overlaid / no-text / failed) counted apart and the "changed" flag
// set only when something was added.
func TestApplyOverlays(t *testing.T) {
	dir, paths := writeBook(t,
		[]string{"ok.png", "empty.png", "bad.png"},
		`<img src="ok.png"><img src="empty.png"><img src="bad.png">`,
	)
	doc, baseDir, err := parseHTMLFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	results := map[string]recognition{
		filepath.Join(dir, "ok.png"): {ok: true, res: Result{
			Width: 100, Height: 200, Blocks: []Block{{Text: "hi", X0: 1, Y0: 1, X1: 50, Y1: 20, LineH: 10}},
		}},
		filepath.Join(dir, "empty.png"): {ok: false},              // read fine, no text
		filepath.Join(dir, "bad.png"):   {err: errors.New("boom")}, // recognition failed
	}

	stats, changed := applyOverlays(doc, baseDir, results)
	if !changed {
		t.Error("changed should be true when an overlay was added")
	}
	if stats.Overlaid != 1 {
		t.Errorf("Overlaid = %d, want 1", stats.Overlaid)
	}
	if stats.NoText != 1 {
		t.Errorf("NoText = %d, want 1", stats.NoText)
	}
	if len(stats.Failed) != 1 || filepath.Base(stats.Failed[0].File) != "bad.png" {
		t.Errorf("Failed = %v, want one bad.png", stats.Failed)
	}

	var buf bytes.Buffer
	if err := gohtml.Render(&buf, doc); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "ocr-fig") {
		t.Error("the overlaid image should be wrapped in an .ocr-fig container")
	}
}

// TestApplyOverlaysNoChange: when no image yields an overlay, nothing is added and the file is
// reported as unchanged (so OverlayBook leaves it on disk untouched).
func TestApplyOverlaysNoChange(t *testing.T) {
	dir, paths := writeBook(t, []string{"empty.png"}, `<img src="empty.png">`)
	doc, baseDir, err := parseHTMLFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	results := map[string]recognition{filepath.Join(dir, "empty.png"): {ok: false}}
	stats, changed := applyOverlays(doc, baseDir, results)
	if changed {
		t.Error("changed should be false when nothing was overlaid")
	}
	if stats.Overlaid != 0 || stats.NoText != 1 {
		t.Errorf("stats = %+v, want NoText=1 only", stats)
	}
}

// TestOverlayBookNoImagesNeverTouchesEngine: a book whose pages hold no recognizable image
// returns an empty result without ever invoking the (here bogus) recognizer - the best-effort
// contract, and the short-circuit that keeps the pool off an empty queue.
func TestOverlayBookNoImagesNeverTouchesEngine(t *testing.T) {
	_, paths := writeBook(t, nil, `<p>no images here</p>`, `<img src="https://x/only-external.png">`)
	stats := OverlayBook("definitely-not-a-real-tesseract-binary", paths, "eng", "", nil)
	if stats.Overlaid != 0 || stats.NoText != 0 || len(stats.Failed) != 0 {
		t.Errorf("empty book should overlay nothing and touch no engine, got %+v", stats)
	}
}
