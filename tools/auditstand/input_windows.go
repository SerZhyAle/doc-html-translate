//go:build windows

package main

import (
	"fmt"
	"unsafe"
)

func injectAuditTab(hwnd uintptr) error {
	if !allowKeyInject {
		return fmt.Errorf("P3 requires -allow-key-inject on the approved audit desktop")
	}
	foreground, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 || foreground != hwnd {
		return fmt.Errorf("P3 refuses injection: the audit window is not foreground")
	}
	// INPUT is 40 bytes on amd64 and 28 on 386; the union follows pointer alignment.
	type keyboardInput struct {
		VK, Scan    uint16
		Flags, Time uint32
		Extra       uintptr
	}
	size := uintptr(28)
	offset := uintptr(4)
	if unsafe.Sizeof(uintptr(0)) == 8 {
		size = 40
		offset = 8
	}
	data := make([]byte, size*2)
	for i := uintptr(0); i < 2; i++ {
		*(*uint32)(unsafe.Pointer(&data[i*size])) = 1
		key := (*keyboardInput)(unsafe.Pointer(&data[i*size+offset]))
		key.VK = 9
		if i == 1 {
			key.Flags = 2
		}
	}
	count, _, err := user32.NewProc("SendInput").Call(2, uintptr(unsafe.Pointer(&data[0])), size)
	if count != 2 {
		return fmt.Errorf("SendInput delivered %d/2 events: %v", count, err)
	}
	return nil
}
