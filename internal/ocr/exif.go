package ocr

// EXIF orientation, and why the overlay has to know about it.
//
// Tesseract reads the file's stored pixels; a browser paints an <img> through its EXIF
// orientation tag (CSS image-orientation defaults to from-image). For a photo tagged
// Orientation=6 - the ordinary portrait shot from a phone - those are two different pictures.
// Two things break at once, and the second hides the first:
//
//   - the lettering the reader sees upright is stored on its side, and PSM 3 runs no orientation
//     detection, so recognition simply returns nothing and the picture looks like one that holds
//     no text;
//   - anything that does read comes back in the stored space, while the plates are positioned in
//     percent of the displayed one. Measured before this existed: a 640x320 scene tagged
//     Orientation=6 rendered a 1216x2432 container whose one plate sat 480 px down a picture
//     whose text runs across the top.
//
// Both are answered in one place - the staged copy handed to tesseract is turned first (see
// stageForOCR), so recognition sees what the reader sees and every coordinate it reports is
// already in the space the plates use. The only thing left for the renderer is the decoded image
// the plate colours are sampled from, which is turned with orientImage so a plate keeps sampling
// the pixels underneath it.
//
// The extension needs no equivalent: it recognizes what createImageBitmap decoded and lays plates
// over the same <img>, so both sides of its comparison are already in display space.

import (
	"encoding/binary"
	"image"
	"image/draw"
	"os"
)

// Orientation values, as stored in EXIF tag 0x0112.
const (
	orientNormal     = 1 // as stored
	orientFlipH      = 2 // mirrored left-right
	orientRotate180  = 3
	orientFlipV      = 4 // mirrored top-bottom
	orientTranspose  = 5 // mirrored, then rotated 90 clockwise
	orientRotate90   = 6 // rotated 90 clockwise
	orientTransverse = 7 // mirrored, then rotated 270 clockwise
	orientRotate270  = 8 // rotated 270 clockwise
)

// exifHeaderBytes is how much of the file is searched for the APP1 segment. EXIF is required to
// sit in the first segments of a JPEG; reading a bounded prefix keeps a 20-megapixel page from
// being pulled into memory to answer a question about its header.
const exifHeaderBytes = 128 << 10

// swapsAxes reports whether an orientation exchanges width and height.
func swapsAxes(orientation int) bool {
	return orientation >= orientTranspose && orientation <= orientRotate270
}

// exifOrientation reads an image file's EXIF orientation, returning orientNormal when there is
// none, when it cannot be parsed, or when it is out of range. Best-effort by design: an
// unreadable header must leave the overlay exactly as it was, never abort a conversion.
//
// JPEG only. PNG (eXIf) and WebP can also carry the tag, but a photo that reaches this app with a
// rotation on it is a JPEG in practice, and a parser for a format nobody hits is a parser nobody
// tests.
func exifOrientation(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return orientNormal
	}
	defer f.Close()
	buf := make([]byte, exifHeaderBytes)
	n, _ := f.Read(buf)
	return exifOrientationFromHeader(buf[:n])
}

// exifOrientationFromHeader parses the tag out of a JPEG's leading bytes.
func exifOrientationFromHeader(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return orientNormal // not a JPEG
	}
	// Walk the marker segments looking for APP1/Exif. Entropy-coded data starts at SOS, and
	// nothing past it is a segment header, so the walk stops there rather than scanning the
	// whole prefix for a byte pair that happens to look like a marker.
	for i := 2; i+4 <= len(data); {
		if data[i] != 0xFF {
			return orientNormal
		}
		marker := data[i+1]
		if marker == 0xD8 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
			i += 2 // standalone marker, no payload
			continue
		}
		if marker == 0xDA || marker == 0xD9 { // SOS / EOI: no more headers
			return orientNormal
		}
		size := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		if size < 2 || i+2+size > len(data) {
			return orientNormal
		}
		payload := data[i+4 : i+2+size]
		if marker == 0xE1 && len(payload) > 6 && string(payload[:6]) == "Exif\x00\x00" {
			return tiffOrientation(payload[6:])
		}
		i += 2 + size
	}
	return orientNormal
}

// tiffOrientation reads IFD0's orientation entry out of the TIFF block inside an APP1 segment.
func tiffOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return orientNormal
	}
	var bo binary.ByteOrder
	switch {
	case tiff[0] == 'I' && tiff[1] == 'I':
		bo = binary.LittleEndian
	case tiff[0] == 'M' && tiff[1] == 'M':
		bo = binary.BigEndian
	default:
		return orientNormal
	}
	if bo.Uint16(tiff[2:4]) != 42 {
		return orientNormal
	}
	off := int(bo.Uint32(tiff[4:8]))
	if off < 8 || off+2 > len(tiff) {
		return orientNormal
	}
	count := int(bo.Uint16(tiff[off : off+2]))
	for i := range count {
		e := off + 2 + i*12
		if e+12 > len(tiff) {
			return orientNormal
		}
		if bo.Uint16(tiff[e:e+2]) != 0x0112 {
			continue
		}
		// SHORT, stored in the entry's own value field.
		v := int(bo.Uint16(tiff[e+8 : e+10]))
		if v < orientNormal || v > orientRotate270 {
			return orientNormal
		}
		return v
	}
	return orientNormal
}

// orientImage returns the displayed view of a decoded image. Best-effort: nil in, nil out.
func orientImage(src image.Image, orientation int) image.Image {
	if src == nil || orientation <= orientNormal || orientation > orientRotate270 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return src
	}
	dw, dh := w, h
	if swapsAxes(orientation) {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	src2 := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(src2, src2.Bounds(), src, b.Min, draw.Src)
	for y := range h {
		for x := range w {
			nx, ny := orientPixel(x, y, w, h, orientation)
			dst.Set(nx, ny, src2.At(x, y))
		}
	}
	return dst
}

// orientPixel maps one pixel centre from stored space (w x h) into display space.
//
// Not orientPoint with the result clamped: clamping is not a bijection, and two source pixels
// landing in the same destination cell means the second one wins and the first is lost. On a
// 4x2 image rotated a quarter turn that turned the top-left pixel black, because the pixel below
// it clamped onto the same cell - which is exactly the pixel a plate samples its paper colour
// from.
func orientPixel(x, y, w, h, orientation int) (int, int) {
	switch orientation {
	case orientFlipH:
		return w - 1 - x, y
	case orientRotate180:
		return w - 1 - x, h - 1 - y
	case orientFlipV:
		return x, h - 1 - y
	case orientTranspose:
		return y, x
	case orientRotate90:
		return h - 1 - y, x
	case orientTransverse:
		return h - 1 - y, w - 1 - x
	case orientRotate270:
		return y, w - 1 - x
	}
	return x, y
}
