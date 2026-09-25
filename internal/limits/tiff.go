package limits

import (
	"encoding/binary"
	"io"
)

// TIFFFrameSize reads ImageWidth and ImageLength straight from the IFD at ifdOffset.
// tiff.DecodeConfig is not enough on its own: on the 386 build it rejects a large frame with
// a bare "image too large" before its size is known, and the user is then told the file is
// broken rather than which limit it crossed. Every offset is computed in int64, so a uint32
// offset past 2 GB cannot wrap negative on that build.
func TIFFFrameSize(r io.ReaderAt, size int64, order binary.ByteOrder, ifdOffset uint32) (w, h int64, ok bool) {
	const (
		tagWidth, tagLength = 256, 257
		typeShort, typeLong = 3, 4
	)
	var n [2]byte
	if int64(ifdOffset)+2 > size {
		return 0, 0, false
	}
	if _, err := r.ReadAt(n[:], int64(ifdOffset)); err != nil {
		return 0, 0, false
	}
	var e [12]byte
	for i := int64(0); i < int64(order.Uint16(n[:])); i++ {
		if _, err := r.ReadAt(e[:], int64(ifdOffset)+2+i*12); err != nil {
			return 0, 0, false
		}
		var v int64
		switch order.Uint16(e[2:4]) {
		case typeShort:
			v = int64(order.Uint16(e[8:10]))
		case typeLong:
			v = int64(order.Uint32(e[8:12]))
		default:
			continue
		}
		switch order.Uint16(e[0:2]) {
		case tagWidth:
			w = v
		case tagLength:
			h = v
		}
	}
	return w, h, w > 0 && h > 0
}

// CheckTIFF probes the first frame of a TIFF and refuses it when its declared size is over
// the pixel budget. A header it cannot read is left to the decoder to report.
func CheckTIFF(r io.ReaderAt, size int64) error {
	var hdr [8]byte
	if n, _ := r.ReadAt(hdr[:], 0); n < len(hdr) {
		return nil
	}
	var order binary.ByteOrder
	switch string(hdr[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return nil
	}
	if w, h, ok := TIFFFrameSize(r, size, order, order.Uint32(hdr[4:8])); ok {
		return CheckPixels(w, h)
	}
	return nil
}
