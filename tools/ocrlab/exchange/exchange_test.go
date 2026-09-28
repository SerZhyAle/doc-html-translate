package exchange

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The shape of OCR-OVERLAY section 7, field by field, from a desktop diagnostic line.
func TestFromLineMapsTheDiagnosticLine(t *testing.T) {
	line := `{"file":"temp\\ocrlab\\r1\\pages\\synth-rtl-layout\\synth-rtl-layout.png","width":800,"height":600,` +
		`"blocks":[{"text":"First","x0":10,"y0":20,"x1":110,"y1":60,"lineH":20,"conf":91.5,"style":"x"},` +
		`{"text":"Second","x0":10,"y0":80,"x1":90,"y1":100,"lineH":20,"conf":77,"style":"y"}],"dropped":[]}`
	id, rec, err := FromLine([]byte(line), func(string) int { return 6 })
	if err != nil {
		t.Fatal(err)
	}
	if id != "synth-rtl-layout" {
		t.Errorf("scene = %q", id)
	}
	if rec.Image != (Image{Width: 800, Height: 600, Orientation: 6}) {
		t.Errorf("image = %+v", rec.Image)
	}
	if len(rec.Blocks) != 2 {
		t.Fatalf("blocks = %d", len(rec.Blocks))
	}
	b := rec.Blocks[1]
	if b.ID != "b-002" || b.Text != "Second" || b.Confidence != 77 || b.ReadingOrder != 2 ||
		b.Box != (Box{X: 10, Y: 80, Width: 80, Height: 20}) {
		t.Errorf("block = %+v", b)
	}

	// translation is absent, not empty: OCR-OVERLAY 1.2 item B.
	out, _ := json.Marshal(rec)
	if strings.Contains(string(out), "translation") {
		t.Errorf("record carries a translation field: %s", out)
	}
	for _, k := range []string{`"image"`, `"orientation"`, `"blocks"`, `"id"`, `"text"`, `"confidence"`, `"box"`, `"readingOrder"`} {
		if !strings.Contains(string(out), k) {
			t.Errorf("record lacks %s: %s", k, out)
		}
	}
	// rotationDegrees is reserved and not part of the format.
	if strings.Contains(string(out), "rotation") {
		t.Errorf("record carries a rotation field: %s", out)
	}
}

// An extension line names the corpus media path; an unreadable orientation degrades to 1, and a
// read-fine-found-nothing image is an empty block list, never null.
func TestFromLineExtensionLineAndDefaults(t *testing.T) {
	line := `{"file":"P:\\x\\test_doc\\ocrlab\\synthetic\\synth-uniform-paper.png","width":10,"height":10,"blocks":[],"dropped":[]}`
	id, rec, err := FromLine([]byte(line), func(string) int { return 0 })
	if err != nil {
		t.Fatal(err)
	}
	if id != "synth-uniform-paper" || rec.Image.Orientation != 1 {
		t.Errorf("scene %q orientation %d", id, rec.Image.Orientation)
	}
	out, _ := json.Marshal(rec)
	if !strings.Contains(string(out), `"blocks":[]`) {
		t.Errorf("empty blocks must serialize as []: %s", out)
	}
}

func TestWriteDirOneDocumentPerSceneLastLineWins(t *testing.T) {
	dir := t.TempDir()
	lines := strings.Join([]string{
		`{"file":"a/s1.png","width":1,"height":1,"blocks":[{"text":"old","x0":0,"y0":0,"x1":1,"y1":1,"conf":1}]}`,
		`{"file":"a/s2.png","width":2,"height":2,"blocks":[]}`,
		`{"file":"a/s1.png","width":1,"height":1,"blocks":[{"text":"new","x0":0,"y0":0,"x1":1,"y1":1,"conf":2}]}`,
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(dir, "diag.jsonl"), []byte(lines), 0o644); err != nil {
		t.Fatal(err)
	}
	ids, err := WriteDir(dir, "diag.jsonl", nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(ids, ",") != "s1,s2" {
		t.Errorf("scenes = %v", ids)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "exchange", "s1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rec Record
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatal(err)
	}
	if len(rec.Blocks) != 1 || rec.Blocks[0].Text != "new" {
		t.Errorf("s1 = %+v, want the last line", rec)
	}
}

func TestWriteDirRefusesAnEmptySidecar(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "diag.jsonl"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteDir(dir, "diag.jsonl", nil); err == nil {
		t.Error("an empty sidecar wrote records")
	}
}
