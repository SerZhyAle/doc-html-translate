package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
)

// Three uncompressed grayscale frames, each with a distinct solid tone. Keeping the
// chained IFDs explicit makes this fixture independent of installed imaging tools.
func genMultiFrameTIFF(dir string) error {
	const width = 64
	const height = 48
	const entries = 9
	const ifdSize = 2 + entries*12 + 4
	const frameSize = ifdSize + width*height
	data := make([]byte, 8+3*frameSize)
	copy(data, "II")
	binary.LittleEndian.PutUint16(data[2:], 42)
	binary.LittleEndian.PutUint32(data[4:], 8)
	for frame := 0; frame < 3; frame++ {
		off := 8 + frame*frameSize
		binary.LittleEndian.PutUint16(data[off:], entries)
		tags := [][3]uint32{{256, 4, width}, {257, 4, height}, {258, 3, 8}, {259, 3, 1}, {262, 3, 1}, {273, 4, uint32(off + ifdSize)}, {277, 3, 1}, {278, 4, height}, {279, 4, width * height}}
		for i, tag := range tags {
			entry := data[off+2+i*12:]
			binary.LittleEndian.PutUint16(entry, uint16(tag[0]))
			binary.LittleEndian.PutUint16(entry[2:], uint16(tag[1]))
			binary.LittleEndian.PutUint32(entry[4:], 1)
			binary.LittleEndian.PutUint32(entry[8:], tag[2])
		}
		if frame < 2 {
			binary.LittleEndian.PutUint32(data[off+2+entries*12:], uint32(off+frameSize))
		}
		for i := off + ifdSize; i < off+frameSize; i++ {
			data[i] = byte(64 + frame*64)
		}
	}
	return os.WriteFile(filepath.Join(dir, "tiff-multiframe.tif"), data, 0o600)
}

// Header-only hostile input: the published pixel budget must refuse before allocating.
func genOverCapTIFF(dir string) error {
	const entries = 2
	data := make([]byte, 8+2+entries*12+4)
	copy(data, "II")
	binary.LittleEndian.PutUint16(data[2:], 42)
	binary.LittleEndian.PutUint32(data[4:], 8)
	binary.LittleEndian.PutUint16(data[8:], entries)
	for i, tag := range []uint16{256, 257} {
		entry := data[10+i*12:]
		binary.LittleEndian.PutUint16(entry, tag)
		binary.LittleEndian.PutUint16(entry[2:], 4)
		binary.LittleEndian.PutUint32(entry[4:], 1)
		binary.LittleEndian.PutUint32(entry[8:], 60000)
	}
	return os.WriteFile(filepath.Join(dir, "tiff-over-cap.tif"), data, 0o600)
}
