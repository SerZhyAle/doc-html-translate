package ocr

import (
	"context"
	"strings"
	"testing"
)

// An image that panics the decoder is one recorded failure; the other images of the book are
// still recognized, and the process is still alive to report it (done criterion 4 of ticket
// bugfix-external-process-bounds).
func TestRecognizePathsContainsAPanickingImage(t *testing.T) {
	saved := recognizeImage
	t.Cleanup(func() { recognizeImage = saved })
	recognizeImage = func(_, imgPath, _, _ string) (Result, error) {
		if strings.HasSuffix(imgPath, "bad.png") {
			panic("image: corrupt stream")
		}
		return Result{Width: 10, Height: 10, Blocks: []Block{{Text: "hello"}}}, nil
	}

	paths := []string{"p1.png", "bad.png", "p3.png", "p4.png"}
	got := recognizePaths(context.Background(), "tesseract", "eng", "", paths, nil)

	if len(got) != len(paths) {
		t.Fatalf("got %d outcomes, want %d", len(got), len(paths))
	}
	bad := got["bad.png"]
	if bad.ok || bad.err == nil || !strings.Contains(bad.err.Error(), "corrupt stream") {
		t.Errorf("bad.png = %+v, want a failure that names the panic", bad)
	}
	for _, p := range []string{"p1.png", "p3.png", "p4.png"} {
		if !got[p].ok {
			t.Errorf("%s was not recognized after another image panicked: %+v", p, got[p])
		}
	}
}

func TestTesseractRunsSingleThreaded(t *testing.T) {
	found := false
	for _, kv := range tesseractEnv {
		if kv == "OMP_THREAD_LIMIT=1" {
			found = true
		}
	}
	if !found {
		t.Errorf("tesseractEnv = %v, want OMP_THREAD_LIMIT=1", tesseractEnv)
	}
}
