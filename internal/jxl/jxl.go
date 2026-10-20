// Package jxl converts JPEG XL pictures to browser- and OCR-readable PNGs.
package jxl

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"

	"github.com/gen2brain/jpegxl"

	"doc-html-translate/internal/fsutil"
	"doc-html-translate/internal/limits"
)

// Is reports both the raw codestream and the ISO container signature.
func Is(b []byte) bool {
	return bytes.HasPrefix(b, []byte{0xff, 0x0a}) || bytes.HasPrefix(b, []byte{0, 0, 0, 12, 'J', 'X', 'L', ' ', 13, 10, 135, 10})
}

// Transcode checks the header before allocating pixels. Like GIF OCR, animated
// pictures use their first displayed frame; alpha is retained in the PNG.
func Transcode(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	const maxBytes = 200 << 20
	if st.Size() > maxBytes {
		return limits.EntryTooLarge(src, maxBytes)
	}
	im, err := decode(f)
	if err != nil {
		return err
	}
	return fsutil.Write(dst, 0o644, func(w io.Writer) error { return png.Encode(w, im) })
}

// PNG also handles embedded data URLs without writing temporary source files.
func PNG(data []byte) ([]byte, error) {
	if len(data) > 200<<20 {
		return nil, limits.EntryTooLarge("JPEG XL", 200<<20)
	}
	im, err := decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	err = png.Encode(&out, im)
	return out.Bytes(), err
}

func decode(f io.ReadSeeker) (image.Image, error) {
	const maxBytes = 200 << 20
	cfg, err := jpegxl.DecodeConfig(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("JPEG XL header: %w", err)
	}
	if err := limits.CheckPixels(int64(cfg.Width), int64(cfg.Height)); err != nil {
		return nil, err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	im, err := jpegxl.Decode(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("decode JPEG XL: %w", err)
	}
	return im, nil
}
