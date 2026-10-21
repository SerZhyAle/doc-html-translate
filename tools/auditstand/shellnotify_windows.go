//go:build windows

// shellnotify_windows.go is P6's shell-notification listener (strategic section 6): a stand-owned
// hidden window registered with SHChangeNotifyRegister that
//
//   - counts every shell change event it receives - the app's registration code ends with
//     SHCNE_ASSOCCHANGED, and a registration that fires it more than once, or alongside a burst
//     of other events, is the "broadcast storm" H11 is about; and
//   - times its own marker notifications (SHCNE_UPDATEITEM on a stand-owned scratch file) from
//     the SHChangeNotify call to the moment the window's message arrives, which is how late the
//     shell delivers to an ordinary listener while the app changes associations (H11's "foreign
//     message latency within 2x control p95").
//
// The stand never sends SHCNE_ASSOCCHANGED itself: that event is a system-wide icon and handler
// refresh, an effect the audit measures in the app and must not cause on the owner's desktop.

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	shellDLL                     = windows.NewLazySystemDLL("shell32.dll")
	procSHChangeNotifyRegister   = shellDLL.NewProc("SHChangeNotifyRegister")
	procSHChangeNotifyDeregister = shellDLL.NewProc("SHChangeNotifyDeregister")
	procSHChangeNotifyEvent      = shellDLL.NewProc("SHChangeNotify")
	procSHGetPathFromIDListW     = shellDLL.NewProc("SHGetPathFromIDListW")
)

const (
	shcnrfShellLevel  = 0x0002     // events raised through SHChangeNotify, not file-system interrupts
	shcneAllEvents    = 0x7FFFFFFF // SHCNE_ALLEVENTS
	shcneUpdateItem   = 0x00002000
	shcneAssocChanged = 0x08000000
	shcnfPathW        = 0x0005
	shcnfFlushNoWait  = 0x3000
	wmShellNotify     = wmApp + 2
	shellMarkerPool   = 4 // rotating marker files: consecutive identical events may be coalesced
)

// Events the listener has seen since the last take. Scenes S11 and S14 read them next to the
// -register/-unregister window they bracket.
var (
	shellMu          sync.Mutex
	shellEventCounts = map[uint32]int{}
	shellMarkersSent int
)

// takeShellEvents returns and clears the counters. The stand's own marker notifications are
// included under shcneUpdateItem; MarkersSent lets a scene subtract what it sent itself.
func takeShellEvents() shellEvents {
	shellMu.Lock()
	defer shellMu.Unlock()
	out := shellEvents{Counts: shellEventCounts, MarkersSent: shellMarkersSent}
	shellEventCounts = map[uint32]int{}
	shellMarkersSent = 0
	return out
}

type shNotifyStruct struct{ item1, item2 uintptr }

type shChangeNotifyEntry struct {
	pidl      uintptr
	recursive int32
}

// runShellNotifyProbe is the P6.shell-notify probe. One locked thread owns the window, sends a
// marker and pumps until it arrives; each arrival is one latency observation.
func runShellNotifyProbe(ctx context.Context, d time.Duration, sink func(ms float64)) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	dir, err := os.MkdirTemp("", "auditstand-shell-*")
	if err != nil {
		return fmt.Errorf("shell probe: %w", err)
	}
	defer os.RemoveAll(dir)
	var markers [shellMarkerPool]string
	for i := range markers {
		markers[i] = filepath.Join(dir, fmt.Sprintf("marker-%d.tmp", i))
		if err := os.WriteFile(markers[i], []byte("x"), 0o600); err != nil {
			return fmt.Errorf("shell probe: %w", err)
		}
	}

	var (
		want      string // the marker path the pump is waiting for
		delivered bool
		pathBuf   = make([]uint16, windows.MAX_PATH)
	)
	proc := windows.NewCallback(func(hwnd, msg, wParam, lParam uintptr) uintptr {
		if msg != wmShellNotify {
			r, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
			return r
		}
		event := uint32(lParam)
		shellMu.Lock()
		shellEventCounts[event]++
		shellMu.Unlock()
		if event == shcneUpdateItem && want != "" && wParam != 0 {
			// wParam is a pointer into shell-owned memory, valid for this message only; copying
			// the pointer bits through its address keeps vet's uintptr rule out of the way.
			n := (*shNotifyStruct)(*(*unsafe.Pointer)(unsafe.Pointer(&wParam)))
			if n.item1 != 0 {
				if r, _, _ := procSHGetPathFromIDListW.Call(n.item1, uintptr(unsafe.Pointer(&pathBuf[0]))); r != 0 {
					if strings.EqualFold(windows.UTF16ToString(pathBuf), want) {
						delivered = true
					}
				}
			}
		}
		return 0
	})
	hwnd, cleanup, err := newHiddenWindow("auditstandShellListener", proc)
	if err != nil {
		return fmt.Errorf("shell probe: %w", err)
	}
	defer cleanup()

	entry := shChangeNotifyEntry{pidl: 0, recursive: 1}
	reg, _, regErr := procSHChangeNotifyRegister.Call(hwnd, shcnrfShellLevel, shcneAllEvents, wmShellNotify, 1, uintptr(unsafe.Pointer(&entry)))
	if reg == 0 {
		return fmt.Errorf("shell probe: SHChangeNotifyRegister: %v", regErr)
	}
	defer func() { _, _, _ = procSHChangeNotifyDeregister.Call(reg) }()

	type msg struct {
		hwnd, message, wParam, lParam uintptr
		time                          uint32
		pt                            struct{ x, y int32 }
	}
	var m msg
	pump := func() {
		for {
			if r, _, _ := procPeekMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, 1 /*PM_REMOVE*/); r == 0 {
				return
			}
			_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
			_, _, _ = procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
		}
	}

	deadline := time.Now().Add(d)
	for i := 0; d == 0 || time.Now().Before(deadline); i++ {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		want = markers[i%shellMarkerPool]
		delivered = false
		pathPtr, err := windows.UTF16PtrFromString(want)
		if err != nil {
			return err
		}
		start := time.Now()
		_, _, _ = procSHChangeNotifyEvent.Call(shcneUpdateItem, shcnfPathW|shcnfFlushNoWait, uintptr(unsafe.Pointer(pathPtr)), 0)
		shellMu.Lock()
		shellMarkersSent++
		shellMu.Unlock()
		for !delivered {
			pump()
			if delivered {
				break
			}
			if time.Since(start) > 2*time.Second {
				return fmt.Errorf("shell probe: marker notification not delivered within 2 s")
			}
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(500 * time.Microsecond):
			}
		}
		sink(float64(time.Since(start).Nanoseconds()) / 1e6)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(50 * time.Millisecond):
		}
	}
	return nil
}

// newHiddenWindow creates an unshown top-level window of a fresh class on the calling thread
// (which the caller has locked) and returns its handle with a cleanup function.
func newHiddenWindow(className string, proc uintptr) (uintptr, func(), error) {
	type wndClass struct {
		style      uint32
		wndProc    uintptr
		clsExtra   int32
		wndExtra   int32
		instance   uintptr
		icon       uintptr
		cursor     uintptr
		background uintptr
		menuName   uintptr
		className  uintptr
	}
	namePtr := windows.StringToUTF16Ptr(className)
	wc := wndClass{wndProc: proc, className: uintptr(unsafe.Pointer(namePtr))}
	if r, _, err := procRegisterClassW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return 0, nil, fmt.Errorf("RegisterClass: %v", err)
	}
	hwnd, _, err := procCreateWindowExW.Call(0,
		uintptr(unsafe.Pointer(namePtr)),
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr("auditstand hidden listener"))),
		0x0F0000 /*WS_OVERLAPPEDWINDOW, never shown*/, 0, 0, 0, 0, 0, 0, 0, 0)
	if hwnd == 0 {
		_, _, _ = user32.NewProc("UnregisterClassW").Call(uintptr(unsafe.Pointer(namePtr)), 0)
		return 0, nil, fmt.Errorf("CreateWindowEx: %v", err)
	}
	return hwnd, func() {
		_, _, _ = procDestroyWindow.Call(hwnd)
		_, _, _ = user32.NewProc("UnregisterClassW").Call(uintptr(unsafe.Pointer(namePtr)), 0)
	}, nil
}
