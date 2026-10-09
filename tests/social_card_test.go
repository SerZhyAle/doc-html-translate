package tests

// Social-card guards (ticket 111, phase 03). One generator, tools/store/make-social-card.ps1, renders the
// Open Graph card and the GitHub repository preview; a card of the wrong size is cropped or refused by the
// platforms, and GitHub rejects a preview of 1 MB or more. The dimensions are declared in the file names.

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

const socialCardMaxBytes = 1_000_000

var socialCards = []struct {
	file          string
	width, height int
}{
	{"social-card-1200x630.png", 1200, 630},
	{"social-card-1280x640.png", 1280, 640},
}

func TestSocialCardSizes(t *testing.T) {
	for _, c := range socialCards {
		path := filepath.Join("..", "assets", c.file)
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("%s: %v - run tools/store/make-social-card.ps1", c.file, err)
		}
		cfg, err := png.DecodeConfig(f)
		f.Close()
		if err != nil {
			t.Errorf("%s is not a decodable PNG: %v", c.file, err)
			continue
		}
		if cfg.Width != c.width || cfg.Height != c.height {
			t.Errorf("%s is %dx%d, want %dx%d", c.file, cfg.Width, cfg.Height, c.width, c.height)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() >= socialCardMaxBytes {
			t.Errorf("%s is %d bytes, want under %d", c.file, info.Size(), socialCardMaxBytes)
		}
	}
}
