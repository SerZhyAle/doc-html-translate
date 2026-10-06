//go:build windows

package fdsec

import (
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// isFileDO confirms by the version resource's product name. A file with no resource at all (the
// Store package's execution alias is a zero-byte link) is accepted: nothing contradicts it.
func isFileDO(path string) bool {
	size, err := windows.GetFileVersionInfoSize(path, nil)
	if err != nil || size == 0 {
		return true
	}
	buf := make([]byte, size)
	if err := windows.GetFileVersionInfo(path, 0, size, unsafe.Pointer(&buf[0])); err != nil {
		return true
	}
	langs := []string{"040904B0", "040904E4", "000004B0"}
	var tr *[2]uint16
	var n uint32
	if err := windows.VerQueryValue(unsafe.Pointer(&buf[0]), `\VarFileInfo\Translation`, unsafe.Pointer(&tr), &n); err == nil && n >= 4 && tr != nil {
		langs = append([]string{hex4(tr[0]) + hex4(tr[1])}, langs...)
	}
	for _, l := range langs {
		var p *uint16
		var plen uint32
		if err := windows.VerQueryValue(unsafe.Pointer(&buf[0]), `\StringFileInfo\`+l+`\ProductName`, unsafe.Pointer(&p), &plen); err != nil || p == nil {
			continue
		}
		return strings.EqualFold(strings.TrimSpace(windows.UTF16PtrToString(p)), "FileDO")
	}
	return true
}

func hex4(v uint16) string {
	const digits = "0123456789ABCDEF"
	return string([]byte{digits[v>>12&0xF], digits[v>>8&0xF], digits[v>>4&0xF], digits[v&0xF]})
}
