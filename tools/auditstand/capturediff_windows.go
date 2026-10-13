//go:build windows

// capturediff_windows.go is P3's fallback timing (strategic section 6): when no debugging
// port can be opened, the visible reaction is timed by diffing the window's pixels. It
// carries no content anywhere - only a downscaled gray grid is ever held in memory, and the
// reported number is a latency.

package main

import (
	"fmt"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	gdi32 = windows.NewLazySystemDLL("gdi32.dll")

	procGetDC             = user32.NewProc("GetDC")
	procReleaseDC         = user32.NewProc("ReleaseDC")
	procGetClientRect     = user32.NewProc("GetClientRect")
	procSetForegroundWind = user32.NewProc("SetForegroundWindow")
	procIsWindowVisible   = user32.NewProc("IsWindowVisible")

	gdiCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	gdiCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	gdiSelectObject           = gdi32.NewProc("SelectObject")
	gdiDeleteObject           = gdi32.NewProc("DeleteObject")
	gdiDeleteDC               = gdi32.NewProc("DeleteDC")
	gdiSetStretchBltMode      = gdi32.NewProc("SetStretchBltMode")
	gdiStretchBlt             = gdi32.NewProc("StretchBlt")
	gdiGetDIBits              = gdi32.NewProc("GetDIBits")
)

const (
	srccopy             = 0x00CC0020
	dibRGBColors        = 0
	stretchColorOnColor = 3
)

// captureDiffGrid is the fallback's fixed grid: coarse enough that a cursor blink does not
// count as a reaction, fine enough that a real repaint shows.
const (
	captureDiffWidth       = 64
	captureDiffHeight      = 48
	captureDiffSensitivity = 12 // gray levels of max pixel change that count as "painted"
)

// captureWindowGrid copies the window's client area into a small gray grid.
func captureWindowGrid(hwnd uintptr) ([]byte, error) {
	var rect struct{ Left, Top, Right, Bottom int32 }
	if r, _, _ := procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rect))); r == 0 {
		return nil, fmt.Errorf("GetClientRect failed")
	}
	if rect.Right-rect.Left <= 0 || rect.Bottom-rect.Top <= 0 {
		return nil, fmt.Errorf("the window has no client area to capture")
	}
	srcDC, _, _ := procGetDC.Call(hwnd)
	if srcDC == 0 {
		return nil, fmt.Errorf("GetDC failed")
	}
	defer procReleaseDC.Call(hwnd, srcDC)

	memDC, _, _ := gdiCreateCompatibleDC.Call(srcDC)
	if memDC == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC failed")
	}
	defer gdiDeleteDC.Call(memDC)

	bmp, _, _ := gdiCreateCompatibleBitmap.Call(srcDC, captureDiffWidth, captureDiffHeight)
	if bmp == 0 {
		return nil, fmt.Errorf("CreateCompatibleBitmap failed")
	}
	defer gdiDeleteObject.Call(bmp)

	old, _, _ := gdiSelectObject.Call(memDC, bmp)
	defer gdiSelectObject.Call(memDC, old)

	gdiSetStretchBltMode.Call(memDC, stretchColorOnColor)
	if r, _, _ := gdiStretchBlt.Call(memDC, 0, 0, captureDiffWidth, captureDiffHeight,
		srcDC, uintptr(rect.Left), uintptr(rect.Top),
		uintptr(rect.Right-rect.Left), uintptr(rect.Bottom-rect.Top), srccopy); r == 0 {
		return nil, fmt.Errorf("StretchBlt failed")
	}

	type bitmapInfoHeader struct {
		Size          uint32
		Width         int32
		Height        int32
		Planes        uint16
		BitCount      uint16
		Compression   uint32
		SizeImage     uint32
		XPelsPerMeter int32
		YPelsPerMeter int32
		ClrUsed       uint32
		ClrImportant  uint32
	}
	info := bitmapInfoHeader{Size: 40, Width: captureDiffWidth, Height: -captureDiffHeight, Planes: 1, BitCount: 32}
	buf := make([]byte, captureDiffWidth*captureDiffHeight*4)
	if r, _, _ := gdiGetDIBits.Call(memDC, bmp, 0, captureDiffHeight,
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&info)), dibRGBColors); r == 0 {
		return nil, fmt.Errorf("GetDIBits failed")
	}
	grid := make([]byte, captureDiffWidth*captureDiffHeight)
	for i := range grid {
		b, g, r := int(buf[i*4]), int(buf[i*4+1]), int(buf[i*4+2])
		grid[i] = byte((r*299 + g*587 + b*114) / 1000)
	}
	return grid, nil
}

// gridMaxDiff returns the largest per-cell gray difference between two grids.
func gridMaxDiff(a, b []byte) int {
	if len(a) != len(b) || len(a) == 0 {
		return 1 << 30 // incomparable captures must never read as "unchanged"
	}
	max := 0
	for i := range a {
		d := int(a[i]) - int(b[i])
		if d < 0 {
			d = -d
		}
		if d > max {
			max = d
		}
	}
	return max
}

// findWindowsByTitle returns every visible top-level window whose title contains the
// substring, case-insensitively. The marker title is the only caller's business here.
func findWindowsByTitle(marker string) ([]uintptr, error) {
	var found []uintptr
	cb := windows.NewCallback(func(hwnd, lparam uintptr) uintptr {
		visible, _, _ := procIsWindowVisible.Call(hwnd)
		if visible != 0 && strings.Contains(strings.ToLower(windowText(hwnd)), strings.ToLower(marker)) {
			found = append(found, hwnd)
		}
		return 1
	})
	r, _, err := procEnumWindows.Call(cb, 0)
	if r == 0 {
		return nil, fmt.Errorf("EnumWindows: %v", err)
	}
	return found, nil
}

// bringWindowToFront makes the stand's own marker window foreground, the precondition of
// the key-injection probe. It is never called on a window the stand did not open.
func bringWindowToFront(hwnd uintptr) error {
	r, _, _ := procSetForegroundWind.Call(hwnd)
	if r == 0 {
		return fmt.Errorf("SetForegroundWindow refused (the foreground lock belongs to the user's session)")
	}
	return nil
}

// waitWindowPaint is the capture-diff fallback's timer: it takes a baseline of the window
// and reports how long until the pixels changed beyond the sensitivity threshold. It is
// used only when the debugging port could not be opened; the DOM-level probes of that run
// are n/a with the recorded reason.
func waitWindowPaint(hwnd uintptr, timeout time.Duration) (float64, bool, error) {
	baseline, err := captureWindowGrid(hwnd)
	if err != nil {
		return 0, false, err
	}
	start := time.Now()
	deadline := start.Add(timeout)
	for time.Now().Before(deadline) {
		grid, err := captureWindowGrid(hwnd)
		if err != nil {
			return 0, false, err
		}
		if gridMaxDiff(baseline, grid) >= captureDiffSensitivity {
			return float64(time.Since(start).Microseconds()) / 1000, true, nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return 0, false, nil
}
