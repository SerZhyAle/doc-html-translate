package ocr

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	gohtml "golang.org/x/net/html"
)

// The whole point of the classification is that "no text in this image" and "recognition
// broke" are different events that used to collapse into one silent number. A graphic novel
// whose art pages carry no dialogue must not look like a book whose OCR is failing.
func TestClassifyRecognition(t *testing.T) {
	boom := errors.New("exit status 1")
	page := Result{Width: 800, Height: 600, Blocks: []Block{{Text: "hi"}}}

	tests := []struct {
		name       string
		res        Result
		err        error
		wantOK     bool
		wantReason bool // a reason is expected, i.e. this is a failure worth naming
	}{
		{"text found: plates to draw", page, nil, true, false},
		{"recognition errored: a real failure", Result{}, boom, false, true},
		{"no text in the image: ordinary, not a failure",
			Result{Width: 800, Height: 600}, nil, false, false},
		{"no page geometry: a failure, not an empty page",
			Result{Blocks: []Block{{Text: "hi"}}}, nil, false, true},
		{"zero height is as broken as zero width",
			Result{Width: 800, Blocks: []Block{{Text: "hi"}}}, nil, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, reason := classifyRecognition(tt.res, tt.err)
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if got := reason != nil; got != tt.wantReason {
				t.Errorf("reason = %v, want a reason: %v", reason, tt.wantReason)
			}
		})
	}

	// The error from Recognize must survive to the report, not be replaced by a generic one:
	// the non-ASCII staging failure was diagnosed from exactly this text.
	if _, reason := classifyRecognition(Result{}, boom); !errors.Is(reason, boom) {
		t.Errorf("the underlying error must reach the caller, got %v", reason)
	}
}

// TestEnsureOverlayAssets: an overlaid page carries both the plate CSS and the runtime re-fit
// script, so the plate that clips at compile time is fixed in the browser. The CSS centres text
// (not top-aligned) and the script fits it - the P10 behaviour, guarded so a refactor can't drop it.
func TestEnsureOverlayAssets(t *testing.T) {
	doc, err := gohtml.Parse(strings.NewReader("<html><head></head><body><p>x</p></body></html>"))
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

	for _, want := range []string{
		"align-items:center",    // plates centre their text, not flex-start
		"<style",                // the plate stylesheet is injected
		"<script",               // the re-fit script is injected
		"querySelectorAll",      // ...and it is the fit routine
		".ocr-box",              // ...operating on plates
		`b.style.height="auto"`, // ...with the grow-so-nothing-clips fallback
	} {
		if !strings.Contains(out, want) {
			t.Errorf("overlaid page is missing %q", want)
		}
	}
	// The script must not be HTML-entity-escaped (script is a raw-text element), or the browser
	// sees &gt; instead of > and the fit never runs.
	if strings.Contains(out, "&gt;") || strings.Contains(out, "&amp;") {
		t.Errorf("script/style was entity-escaped, breaking the JS:\n%s", out)
	}
}

func TestOverlayResultAdd(t *testing.T) {
	var total OverlayResult
	total.Add(OverlayResult{Overlaid: 3, NoText: 1})
	total.Add(OverlayResult{Overlaid: 2, NoText: 4, Failed: []OverlayFailure{{File: "a.png"}}})

	if total.Overlaid != 5 || total.NoText != 5 {
		t.Errorf("Overlaid=%d NoText=%d, want 5 and 5", total.Overlaid, total.NoText)
	}
	if len(total.Failed) != 1 || total.Failed[0].File != "a.png" {
		t.Errorf("failures must accumulate with their file, got %v", total.Failed)
	}
}
