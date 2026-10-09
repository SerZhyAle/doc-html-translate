package evidence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Both editions write the same line; the desktop one names the page image, the extension one the
// corpus media path, and both key to the scene by the file's stem.
func TestLoadSidecarKeysBothEditionsByScene(t *testing.T) {
	lines := strings.Join([]string{
		`{"file":"temp\\r\\pages\\poster\\poster.png","width":9,"height":9,"blocks":[],"dropped":[{"text":"x","conf":21,"floor":60,"gate":"confidence","x0":0,"y0":0,"x1":1,"y1":1}]}`,
		`{"file":"P:\\x\\test_doc\\ocrlab\\map.jpg","width":9,"height":9,"blocks":[{"text":"kept","x0":0,"y0":0,"x1":1,"y1":1,"conf":90}],"dropped":[]}`,
		`{"file":"a/poster.png","width":9,"height":9,"blocks":[],"dropped":[]}`,
	}, "\n") + "\n"
	path := filepath.Join(t.TempDir(), "ocr-diag.jsonl")
	if err := os.WriteFile(path, []byte(lines), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSidecar(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("scenes = %d, want 2", len(got))
	}
	if r := got["poster"]; r == nil || len(r.Dropped) != 0 {
		t.Errorf("poster = %+v, want the last line (nothing dropped)", r)
	}
	if r := got["map"]; r == nil || len(r.Blocks) != 1 || r.Blocks[0].Conf != 90 {
		t.Errorf("map = %+v", r)
	}
}

func TestLoadSidecarMissingFileIsEmptyAndGarbageIsAnError(t *testing.T) {
	dir := t.TempDir()
	got, err := LoadSidecar(filepath.Join(dir, "absent.jsonl"))
	if err != nil || len(got) != 0 {
		t.Errorf("missing sidecar = %v, %v; want empty and no error", got, err)
	}
	bad := filepath.Join(dir, "bad.jsonl")
	if err := os.WriteFile(bad, []byte("{not json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSidecar(bad); err == nil {
		t.Error("a corrupt sidecar line was accepted")
	}
}
