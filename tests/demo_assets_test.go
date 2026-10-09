package tests

// Ticket 111, phase 06 (canon PROMOTION section 4 item 4): one silent demo capture under a minute, an
// MP4 the site plays from its own origin and a GIF in the README, no third-party player. The master is
// recorded and encoded by tools/store/make-demo.ps1; these checks hold what the encoded files and the
// pages that reference them must keep true.

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	demoMP4MaxBytes = 4_000_000
	demoGIFMaxBytes = 5_000_000
	demoMaxSeconds  = 60
)

func readAsset(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "assets", name))
	if err != nil {
		t.Fatalf("assets/%s: %v - run tools/store/make-demo.ps1", name, err)
	}
	return b
}

func TestDemoMP4IsSilentAndShort(t *testing.T) {
	b := readAsset(t, "demo.mp4")
	if len(b) > demoMP4MaxBytes {
		t.Errorf("demo.mp4 is %d bytes, the limit is %d", len(b), demoMP4MaxBytes)
	}
	if len(b) < 12 || string(b[4:8]) != "ftyp" {
		t.Fatal("demo.mp4 does not start with an ftyp box")
	}
	if !bytes.Contains(b, []byte("vide")) {
		t.Error("demo.mp4 has no video track")
	}
	if bytes.Contains(b, []byte("soun")) {
		t.Error("demo.mp4 carries an audio track; the demo is silent")
	}
	i := bytes.Index(b, []byte("mvhd"))
	if i < 0 || i+24 > len(b) {
		t.Fatal("demo.mp4 has no movie header")
	}
	// mvhd, version 0: version/flags(4) creation(4) modification(4) timescale(4) duration(4).
	if b[i+4] != 0 {
		t.Skip("movie header version 1 is not read here")
	}
	timescale := binary.BigEndian.Uint32(b[i+16 : i+20])
	duration := binary.BigEndian.Uint32(b[i+20 : i+24])
	if timescale == 0 {
		t.Fatal("demo.mp4 movie header has a zero timescale")
	}
	if secs := float64(duration) / float64(timescale); secs >= demoMaxSeconds {
		t.Errorf("demo.mp4 runs %.1f s, the demo is under %d s", secs, demoMaxSeconds)
	}
}

func TestDemoGIFIsReadmeSized(t *testing.T) {
	b := readAsset(t, "demo.gif")
	if len(b) > demoGIFMaxBytes {
		t.Errorf("demo.gif is %d bytes, the limit is %d", len(b), demoGIFMaxBytes)
	}
	if !bytes.HasPrefix(b, []byte("GIF89a")) && !bytes.HasPrefix(b, []byte("GIF87a")) {
		t.Error("demo.gif is not a GIF")
	}
}

func TestDemoIsReferencedWithoutAThirdPartyPlayer(t *testing.T) {
	landing := readRepoFile(t, "index.html")
	for _, want := range []string{"assets/demo.mp4", "assets/demo-poster.jpg", "<video "} {
		if !strings.Contains(landing, want) {
			t.Errorf("index.html does not reference %q", want)
		}
	}
	if _, err := os.Stat(filepath.Join("..", "assets", "demo-poster.jpg")); err != nil {
		t.Errorf("the landing's poster is missing: %v", err)
	}
	for _, p := range []string{"README.md", "README_RU.md", "README_UK.md"} {
		if !strings.Contains(readRepoFile(t, p), "assets/demo.gif") {
			t.Errorf("%s does not show assets/demo.gif", p)
		}
	}
	for _, p := range []string{"index.html", "README.md", "README_RU.md", "README_UK.md"} {
		low := strings.ToLower(readRepoFile(t, p))
		for _, player := range []string{"youtube.com", "youtu.be", "vimeo.com", "<iframe"} {
			if strings.Contains(low, player) {
				t.Errorf("%s embeds or links a third-party player: %s", p, player)
			}
		}
	}
}
