package ocr

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

// exifJPEG encodes img as a JPEG carrying one IFD0 entry: Orientation = orientation.
func exifJPEG(t *testing.T, img image.Image, orientation int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	data := buf.Bytes()
	app1 := []byte{
		0xFF, 0xE1, 0x00, 0x22,
		'E', 'x', 'i', 'f', 0x00, 0x00,
		'I', 'I', 0x2A, 0x00, 0x08, 0x00, 0x00, 0x00,
		0x01, 0x00,
		0x12, 0x01, 0x03, 0x00, 0x01, 0x00, 0x00, 0x00, byte(orientation), 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
	return append([]byte{0xFF, 0xD8}, append(app1, data[2:]...)...)
}

func TestExifOrientationReadsTheTag(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 4))
	for _, want := range []int{1, 2, 3, 4, 5, 6, 7, 8} {
		path := filepath.Join(t.TempDir(), "photo.jpg")
		if err := os.WriteFile(path, exifJPEG(t, img, want), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := exifOrientation(path); got != want {
			t.Errorf("orientation %d read back as %d", want, got)
		}
	}
}

// A file with no EXIF at all, a non-JPEG and a missing file must all read as "as stored" - the
// overlay's behaviour on the overwhelming majority of images must not depend on this parser
// succeeding.
func TestExifOrientationDefaultsToNormal(t *testing.T) {
	dir := t.TempDir()
	var plain bytes.Buffer
	if err := jpeg.Encode(&plain, image.NewRGBA(image.Rect(0, 0, 4, 4)), nil); err != nil {
		t.Fatal(err)
	}
	cases := map[string][]byte{
		"plain.jpg": plain.Bytes(),
		"notjpeg":   []byte("\x89PNG\r\n\x1a\n and then some"),
		"empty.jpg": {},
		"truncated": {0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x40, 'E', 'x', 'i', 'f'},
	}
	for name, data := range cases {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		if got := exifOrientation(path); got != orientNormal {
			t.Errorf("%s: orientation %d, want %d", name, got, orientNormal)
		}
	}
	if got := exifOrientation(filepath.Join(dir, "no-such-file.jpg")); got != orientNormal {
		t.Errorf("missing file: orientation %d, want %d", got, orientNormal)
	}
}

// The case that mattered: a photo tagged Orientation=6 (the ordinary portrait phone shot). The
// copy handed to tesseract must be the turned one - a 640x320 file becomes a 320x640 picture -
// because that is both the only orientation PSM 3 can read and the space the plates use.
func TestPrepareForOCRAppliesEXIFRotation(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 40, 20))
	path := filepath.Join(t.TempDir(), "portrait.jpg")
	if err := os.WriteFile(path, exifJPEG(t, src, orientRotate90), 0o644); err != nil {
		t.Fatal(err)
	}
	staged, scale, _, cleanup := prepareForOCR(path)
	defer cleanup()
	if staged == path {
		t.Fatal("a rotated photo was handed to tesseract unchanged")
	}
	im := decodeImage(staged)
	if im == nil {
		t.Fatal("staged copy is not decodable")
	}
	// 40x20 is low-res enough to be enlarged as well, so the check is on the ratio.
	if w, h := im.Bounds().Dx(), im.Bounds().Dy(); w != 20*scale || h != 40*scale {
		t.Errorf("staged copy is %dx%d, want %dx%d (turned, scale %d)", w, h, 20*scale, 40*scale, scale)
	}
}

// An unrotated file must reach tesseract exactly as before: same path, no temp copy, no decode.
// This is every image in an ordinary book, and the cost of the rotation path must not land on it.
func TestPrepareForOCRLeavesUnrotatedFilesAlone(t *testing.T) {
	path := tempImageFile(t, 2200, 1600) // above the upscale floor
	staged, scale, _, cleanup := prepareForOCR(path)
	defer cleanup()
	if staged != path || scale != 1 {
		t.Errorf("unrotated image staged to %q at scale %d, want the original at scale 1", staged, scale)
	}
}

// The plate borrows its paper and ink from the pixels under it, so the decoded image the renderer
// samples has to make the same turn recognition did - otherwise a rotated photo takes each
// plate's colours from somewhere else in the picture.
func TestOrientImageMovesPixelsIntoDisplaySpace(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 2))
	mark := color.RGBA{200, 30, 30, 255}
	src.Set(0, 0, mark) // top-left of the stored picture

	img := orientImage(src, orientRotate90)
	if img.Bounds().Dx() != 2 || img.Bounds().Dy() != 4 {
		t.Fatalf("image is %v, want 2x4", img.Bounds())
	}
	// A quarter turn clockwise sends the stored top-left pixel to the displayed top-right.
	r, g, b, _ := img.At(1, 0).RGBA()
	if uint8(r>>8) != mark.R || uint8(g>>8) != mark.G || uint8(b>>8) != mark.B {
		t.Errorf("top-right pixel is rgb(%d,%d,%d), want rgb(%d,%d,%d)",
			r>>8, g>>8, b>>8, mark.R, mark.G, mark.B)
	}
}

// Every orientation must be a bijection over the pixel grid: clamping two source pixels into one
// destination cell silently loses the first, and it lost exactly the pixel a plate samples.
func TestOrientImageKeepsEveryPixel(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 5, 3))
	for y := range 3 {
		for x := range 5 {
			src.Set(x, y, color.RGBA{uint8(10 + x), uint8(20 + y), 200, 255})
		}
	}
	for o := orientNormal; o <= orientRotate270; o++ {
		img := orientImage(src, o)
		wantW, wantH := 5, 3
		if swapsAxes(o) {
			wantW, wantH = 3, 5
		}
		if img.Bounds().Dx() != wantW || img.Bounds().Dy() != wantH {
			t.Errorf("orientation %d: %v, want %dx%d", o, img.Bounds(), wantW, wantH)
			continue
		}
		seen := map[uint32]bool{}
		for y := range wantH {
			for x := range wantW {
				r, g, b, _ := img.At(x, y).RGBA()
				seen[r>>8<<16|g>>8<<8|b>>8] = true
			}
		}
		if len(seen) != 15 {
			t.Errorf("orientation %d: %d distinct pixels survived, want 15", o, len(seen))
		}
	}
}
