package iconart

import (
	"encoding/binary"
	"testing"
	"time"
)

// Numbers after a closepath have no command to belong to; the parser must refuse
// them instead of looping on the same byte forever.
func TestParsePathRefusesNumbersAfterClosepath(t *testing.T) {
	done := make(chan error, 1)
	go func() {
		_, err := parsePath("M0 0L1 1Z 5 5")
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("numbers after Z were accepted")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("parsePath did not return on numbers after Z")
	}
}

// A truncated or lying ICO header is an error, not a slice panic.
func TestDecodeICORejectsTruncatedDirectory(t *testing.T) {
	raw := make([]byte, 6)
	binary.LittleEndian.PutUint16(raw[2:], 1)
	binary.LittleEndian.PutUint16(raw[4:], 3) // claims three entries, holds none
	if _, err := DecodeICO(raw); err == nil {
		t.Fatal("a directory past the end of the file was accepted")
	}
	one := make([]byte, 6+16)
	binary.LittleEndian.PutUint16(one[2:], 1)
	binary.LittleEndian.PutUint16(one[4:], 1)
	binary.LittleEndian.PutUint32(one[6+8:], 0xFFFFFFF0) // size
	binary.LittleEndian.PutUint32(one[6+12:], 0x20)      // offset; off+size wraps in 32 bits
	if _, err := DecodeICO(one); err == nil {
		t.Fatal("a frame whose extent wraps around was accepted")
	}
}
