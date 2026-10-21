//go:build windows

// probes_windows.go holds the platform probes of strategic section 6: the window ping (P1), the
// clipboard and message-latency citizen probes and the focus observer (P6), and the P4 process
// sampler plus the Level-B registry snapshot helpers. Everything here runs outside the app
// processes; the app is never debugged or injected into (strategic section 2).

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/net/html"
	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	psapi    = windows.NewLazySystemDLL("psapi.dll")
	iphlpapi = windows.NewLazySystemDLL("iphlpapi.dll") //nolint:unused // kept for listeningPortsFor (ticket 113, P2 host ping)

	procSendMessageTimeoutW = user32.NewProc("SendMessageTimeoutW")
	procEnumWindows         = user32.NewProc("EnumWindows")
	procGetWindowTextW      = user32.NewProc("GetWindowTextW")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procGetGuiResources     = user32.NewProc("GetGuiResources")
	procOpenClipboard       = user32.NewProc("OpenClipboard")
	procCloseClipboard      = user32.NewProc("CloseClipboard")
	procIsClipboardFormat   = user32.NewProc("IsClipboardFormatAvailable")
	procRegisterClassW      = user32.NewProc("RegisterClassW")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procPeekMessageW        = user32.NewProc("PeekMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procPostMessageW        = user32.NewProc("PostMessageW")

	procGetProcessHandleCount = kernel32.NewProc("GetProcessHandleCount")

	procGetProcessMemoryInfo = psapi.NewProc("GetProcessMemoryInfo")
	procGetExtendedTcpTable  = iphlpapi.NewProc("GetExtendedTcpTable") //nolint:unused // kept for listeningPortsFor (ticket 113, P2 host ping)
)

const (
	wmNull              = 0x0000
	wmApp               = 0x8000
	cfUnicode           = 13
	smtoAbortIfHung     = 0x0002
	grpcGDIObjects      = 0
	grpcUserObjects     = 1
	processQueryLimited = 0x1000
	th32csSnapProcess   = 0x2
	th32csSnapThread    = 0x4
	tcpTableOwnerPid    = 3
)

// platformProbes is the Windows half of the probe registry (probes.go provides the rest).
func platformProbes() []latencyProbe {
	return []latencyProbe{
		{
			name:        "P1.window-ping",
			budgetNames: []string{"H2"},
			available:   windowPingAvailable,
			run:         runWindowPing,
		},
		{
			name:        "P6.clipboard-read",
			budgetNames: []string{"H9"},
			available:   func() (bool, string) { return true, "" },
			run:         runClipboardProbe,
		},
		{
			name:        "P6.clipboard-write",
			budgetNames: []string{"H9"},
			available: func() (bool, string) {
				return false, "writer coverage requires a disposable desktop; clipboard contents are never changed on the owner's desktop"
			},
		},
		{
			name:        "P6.message-latency",
			budgetNames: []string{"H10"},
			available:   func() (bool, string) { return true, "" },
			run:         runMessageLatencyProbe,
		},
		{
			name:        "P6.focus-observer",
			budgetNames: []string{"H9"},
			available:   func() (bool, string) { return true, "" },
			run:         runFocusObserverProbe,
		},
		{
			name:        "P6.shell-notify",
			budgetNames: []string{"H11"},
			available:   func() (bool, string) { return true, "" },
			run:         runShellNotifyProbe,
		},
	}
}

// --- P1: window ping ---------------------------------------------------------------

func findAuditWindows() ([]uintptr, error) {
	var found []uintptr
	cb := windows.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		if strings.Contains(strings.ToLower(windowText(hwnd)), strings.ToLower(auditWindowMarker)) {
			found = append(found, hwnd)
		}
		return 1 // continue enumeration
	})
	r, _, err := procEnumWindows.Call(cb, 0)
	if r == 0 {
		return nil, fmt.Errorf("EnumWindows: %v", err)
	}
	return found, nil
}

func windowText(hwnd uintptr) string {
	buf := make([]uint16, 256)
	n, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return windows.UTF16ToString(buf[:n])
}

func windowPingAvailable() (bool, string) {
	wins, err := findAuditWindows()
	if err != nil {
		return false, fmt.Sprintf("window enumeration failed: %v", err)
	}
	if len(wins) == 0 {
		return false, fmt.Sprintf("no window titled like %q; the window ping measures the thread the user sees and needs it (start the GUI, or retarget with -window)", auditWindowMarker)
	}
	return true, ""
}

// runWindowPing pings every matching window with SendMessageTimeout(WM_NULL) and sinks each
// round-trip time. A stall here caused by an app action is the finding (strategic section 6 P1).
func runWindowPing(ctx context.Context, d time.Duration, sink func(ms float64)) error {
	const timeoutMs = 2000
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		wins, err := findAuditWindows()
		if err != nil {
			return fmt.Errorf("window ping: %w", err)
		}
		if len(wins) == 0 {
			return fmt.Errorf("window ping: the window vanished mid-probe")
		}
		start := time.Now()
		var res uintptr
		for _, w := range wins {
			r, _, callErr := procSendMessageTimeoutW.Call(
				w, wmNull, 0, 0, smtoAbortIfHung, timeoutMs, uintptr(unsafe.Pointer(&res)))
			if r == 0 {
				return fmt.Errorf("window ping: SendMessageTimeout failed: %v", callErr)
			}
		}
		sink(float64(time.Since(start).Nanoseconds()) / 1e6)
		time.Sleep(20 * time.Millisecond)
	}
}

// --- P6: clipboard -----------------------------------------------------------------

// runClipboardProbe measures foreign clipboard write+read latency (P6, H9). It saves the current
// text content first and restores it after - the probe leaves the clipboard as it found it.
// The control runs on the owner's desktop. Opening/querying the clipboard measures reader
// contention without changing any format or retaining content. Writer coverage requires a
// disposable desktop and remains an explicit unavailable probe.
func runClipboardProbe(ctx context.Context, d time.Duration, sink func(float64)) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	for {
		if ctx.Err() != nil {
			return nil
		}
		start := time.Now()
		if err := openClipboard(); err != nil {
			return err
		}
		_, _, _ = procIsClipboardFormat.Call(cfUnicode)
		if ok, _, err := procCloseClipboard.Call(); ok == 0 {
			return fmt.Errorf("CloseClipboard: %v", err)
		}
		sink(float64(time.Since(start).Nanoseconds()) / 1e6)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(20 * time.Millisecond):
		}
	}
}
func openClipboard() error {
	// Another program may hold the clipboard open for a moment; retry briefly, since a foreign
	// program starving here is exactly what this probe exists to notice.
	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		if r, _, _ := procOpenClipboard.Call(0); r != 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("OpenClipboard failed (clipboard held or wedged)")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// --- P6: message latency ------------------------------------------------------------

// runMessageLatencyProbe measures how long Windows takes to deliver a posted message to a normal
// listening window while the machine does everything else a scene does (P6, H10). One stand
// thread owns the window, pumps its queue with PeekMessage and times each post-to-dispatch gap.
func runMessageLatencyProbe(ctx context.Context, d time.Duration, sink func(ms float64)) error {
	const className = "auditstandListener"
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

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
	// The wndproc records nothing itself: the dispatch time is taken in the pump loop the moment
	// DispatchMessage returns from our WM_APP marker, which is the delivery moment.
	delivered := false
	wc := wndClass{wndProc: windows.NewCallback(func(hwnd, msg, wParam, lParam uintptr) uintptr {
		if msg == wmApp+1 {
			delivered = true
			return 0
		}
		r, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
		return r
	}), className: uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(className)))}
	if r, _, err := procRegisterClassW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return fmt.Errorf("message probe: RegisterClass: %v", err)
	}
	defer func() { _, _, _ = user32.NewProc("UnregisterClassW").Call(wc.className, 0) }()
	hwnd, _, err := procCreateWindowExW.Call(0,
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(className))),
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr("auditstand listener"))),
		0x0F0000 /*WS_OVERLAPPEDWINDOW (unshown)*/, 0, 0, 0, 0, 0, 0, 0, 0)
	if hwnd == 0 {
		return fmt.Errorf("message probe: CreateWindowEx: %v", err)
	}
	defer func() { _, _, _ = procDestroyWindow.Call(hwnd) }()

	type msg struct {
		hwnd, message, wParam, lParam uintptr
		time                          uint32
		pt                            struct{ x, y int32 }
	}
	var m msg
	const pmRemove = 1
	deadline := time.Now().Add(d)
	for d == 0 || time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		start := time.Now()
		delivered = false
		if r, _, err := procPostMessageW.Call(hwnd, wmApp+1, 0, 0); r == 0 {
			return fmt.Errorf("message probe: PostMessage: %v", err)
		}
		for !delivered {
			if r, _, _ := procPeekMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, pmRemove); r != 0 {
				_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
				_, _, _ = procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
				continue
			}
			if time.Since(start) > 2*time.Second {
				return fmt.Errorf("message probe: no dispatch within 2 s")
			}
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(500 * time.Microsecond):
			}
		}
		sink(float64(time.Since(start).Nanoseconds()) / 1e6)
		time.Sleep(20 * time.Millisecond)
	}
	return nil
}

// --- P6: focus observer -------------------------------------------------------------

// focusTransitions accumulates foreground-window transitions the focus observer sees. Scenes
// read them to score H9 ("only the app's own scenes change the foreground window") and H12.
var (
	focusMu          sync.Mutex
	focusTransitions []focusEvent
)

// takeFocusTransitions returns and clears the collected transitions.
func takeFocusTransitions() []focusEvent { //nolint:unused // kept for the focus-transition probe (ticket 113)
	focusMu.Lock()
	defer focusMu.Unlock()
	out := focusTransitions
	focusTransitions = nil
	return out
}

// runFocusObserverProbe watches the foreground window at 20 ms and sinks each poll's own call
// cost (the syscall latency a scene compares against control); transitions accumulate for the
// scene's H9/H12 verdicts.
func runFocusObserverProbe(ctx context.Context, d time.Duration, sink func(ms float64)) error {
	var last uintptr
	first := true
	deadline := time.Now().Add(d)
	for d == 0 || time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		start := time.Now()
		h, _, _ := procGetForegroundWindow.Call()
		sink(float64(time.Since(start).Nanoseconds()) / 1e6)
		if h != last || first {
			if !first && h != 0 {
				focusMu.Lock()
				focusTransitions = append(focusTransitions, focusEvent{At: time.Now(), Window: h, Title: windowText(h)})
				focusMu.Unlock()
			}
			last = h
			first = false
		}
		time.Sleep(20 * time.Millisecond)
	}
	return nil
}

// --- P4: process sampler -------------------------------------------------------------

// sampleProcess reads one process's resource counters. A dead process returns ok=false, which
// the sampler records (a vanishing child mid-scene is data, not an error).
func sampleProcess(pid int) (procSample, bool) {
	h, _, _ := kernel32.NewProc("OpenProcess").Call(0x0400|0x0010 /*QUERY_INFORMATION | VM_READ*/, 0, uintptr(pid))
	if h == 0 {
		return procSample{PID: pid}, false
	}
	defer func() { _ = windows.CloseHandle(windows.Handle(h)) }()
	var s procSample
	s.PID = pid
	var count int32
	if r, _, _ := procGetProcessHandleCount.Call(h, uintptr(unsafe.Pointer(&count))); r != 0 {
		s.Handles = int64(count)
	}
	gdi, _, _ := procGetGuiResources.Call(h, grpcGDIObjects)
	s.GDI = int64(gdi)
	usr, _, _ := procGetGuiResources.Call(h, grpcUserObjects)
	s.USER = int64(usr)
	s.Threads = int64(countThreads(pid))
	type pmcEx struct {
		Cb                 uint32
		PageFaultCount     uint32
		PeakWorkingSetSize uintptr
		WorkingSetSize     uintptr
		QuotaPeakPagedPool uintptr
		QuotaPagedPool     uintptr
		QuotaPeakNonPaged  uintptr
		QuotaNonPaged      uintptr
		PagefileUsage      uintptr
		PeakPagefileUsage  uintptr
		PrivateUsage       uintptr
	}
	var mem pmcEx
	mem.Cb = uint32(unsafe.Sizeof(mem))
	if r, _, _ := procGetProcessMemoryInfo.Call(h, uintptr(unsafe.Pointer(&mem)), uintptr(unsafe.Sizeof(mem))); r != 0 {
		s.WorkingSetByte = float64(mem.WorkingSetSize)
		s.PrivateBytes = float64(mem.PrivateUsage)
	} else {
		return procSample{PID: pid}, false
	}
	return s, true
}

// countThreads counts the system threads owned by pid via a toolhelp snapshot.
func countThreads(pid int) int {
	snap, err := windows.CreateToolhelp32Snapshot(th32csSnapThread, 0)
	if err != nil {
		return 0
	}
	defer func() { _ = windows.CloseHandle(snap) }()
	var te windows.ThreadEntry32
	te.Size = uint32(unsafe.Sizeof(te))
	n := 0
	if err := windows.Thread32First(snap, &te); err != nil {
		return 0
	}
	for {
		if te.OwnerProcessID == uint32(pid) {
			n++
		}
		if err := windows.Thread32Next(snap, &te); err != nil {
			break
		}
	}
	return n
}

// findProcessesByName returns the pids whose executable image name matches (case-insensitive).
func findProcessesByName(name string) []int {
	snap, err := windows.CreateToolhelp32Snapshot(th32csSnapProcess, 0)
	if err != nil {
		return nil
	}
	defer func() { _ = windows.CloseHandle(snap) }()
	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	var out []int
	if err := windows.Process32First(snap, &pe); err != nil {
		return nil
	}
	for {
		if strings.EqualFold(windows.UTF16ToString(pe.ExeFile[:]), name) {
			out = append(out, int(pe.ProcessID))
		}
		if err := windows.Process32Next(snap, &pe); err != nil {
			break
		}
	}
	return out
}

// listeningPortsFor returns the ports a pid listens on (the P2 host ping's discovery half, since
// the GUI host binds a random loopback port each launch).
func listeningPortsFor(wantPID int) ([]int, error) { //nolint:unused // kept for the P2 host ping discovery half (ticket 113)
	var size uint32
	ret, _, _ := procGetExtendedTcpTable.Call(0, uintptr(unsafe.Pointer(&size)), 0, 2 /*AF_INET*/, tcpTableOwnerPid, 0)
	if ret != 122 /*ERROR_INSUFFICIENT_BUFFER*/ || size == 0 {
		return nil, fmt.Errorf("tcp table sizing failed: ret=%d size=%d", ret, size)
	}
	buf := make([]byte, size)
	ret, _, _ = procGetExtendedTcpTable.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)), 0, 2, tcpTableOwnerPid, 0)
	if ret != 0 {
		return nil, fmt.Errorf("tcp table read failed: ret=%d", ret)
	}
	n := *(*uint32)(unsafe.Pointer(&buf[0]))
	const rowSz = 24 // MIB_TCPROW_OWNER_PID
	var ports []int
	for i := uint32(0); i < n && 4+i*rowSz+rowSz <= uint32(len(buf)); i++ {
		base := uintptr(4 + i*rowSz)
		state := *(*uint32)(unsafe.Pointer(&buf[base]))
		const mibTcpStateListen = 2
		if state != mibTcpStateListen {
			continue
		}
		rowPID := *(*uint32)(unsafe.Pointer(&buf[base+20]))
		if int(rowPID) != wantPID {
			continue
		}
		raw := *(*uint32)(unsafe.Pointer(&buf[base+8]))
		ports = append(ports, int((raw&0xff)<<8|(raw>>8)&0xff))
	}
	return ports, nil
}

// --- P2: host ping --------------------------------------------------------------------

// tokenRe finds the per-launch API token the served page carries (guard.go bakes it into the
// page and nowhere else; it is a 64-char hex string).
var tokenRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

func hostPageToken(body []byte) (string, error) {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	var token string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			name, content := "", ""
			for _, a := range n.Attr {
				if a.Key == "name" {
					name = a.Val
				}
				if a.Key == "content" {
					content = a.Val
				}
			}
			if name == "dht-token" && tokenRe.MatchString(content) {
				token = content
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if token == "" {
		return "", fmt.Errorf("host page has no valid dht-token meta element")
	}
	return token, nil
}

// hostPing discovers the GUI host's token from the served page, then times n trivial
// /api/version round trips with the token header - the window-less equivalent of the window
// ping (strategic section 6 P2).
func hostPing(baseURL string, n int, timeout time.Duration) (measurement, error) {
	if err := loopbackURL(baseURL, "http"); err != nil {
		return measurement{}, err
	}
	if n < 1 || timeout <= 0 {
		return measurement{}, fmt.Errorf("host ping needs positive sample count and timeout")
	}
	client := &http.Client{Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(baseURL + "/")
	if err != nil {
		return measurement{}, fmt.Errorf("host ping: page fetch: %w", err)
	}
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	resp.Body.Close()
	if readErr != nil {
		return measurement{}, readErr
	}
	if len(body) > 1<<20 {
		return measurement{}, fmt.Errorf("host page exceeded 1 MiB cap")
	}
	if resp.StatusCode != http.StatusOK {
		return measurement{}, fmt.Errorf("host ping: page fetch: status %d", resp.StatusCode)
	}
	token, err := hostPageToken(body)
	if err != nil {
		return measurement{}, err
	}
	var ms []float64
	for i := 0; i < n; i++ {
		req, err := http.NewRequest(http.MethodGet, baseURL+"/api/version", nil)
		if err != nil {
			return measurement{}, err
		}
		req.Header.Set("X-DHT-Token", token)
		start := time.Now()
		r, err := client.Do(req)
		if err != nil {
			return measurement{}, fmt.Errorf("host ping: %w", err)
		}
		_, readErr := io.Copy(io.Discard, io.LimitReader(r.Body, 1<<20))
		_ = r.Body.Close()
		if readErr != nil {
			return measurement{}, readErr
		}
		if r.StatusCode != http.StatusOK {
			return measurement{}, fmt.Errorf("host ping: /api/version status %d", r.StatusCode)
		}
		ms = append(ms, float64(time.Since(start).Nanoseconds())/1e6)
		time.Sleep(10 * time.Millisecond)
	}
	return summarize("P2.host", ms), nil
}

// setKeyInject turns the opt-in flag on from main; on by default off, so a stand run uninvoked
// never types at the owner.
// allowKeyInject gates future input-injection probes (the hook negative control lands with
// S14, phase 02); off by default so a stand run uninvoked never types at the owner.
var allowKeyInject = false

func setKeyInject(v bool) { allowKeyInject = v }

// freeBytes reports the free disk space under dir (the -Big generation gate).
func freeBytes(dir string) (uint64, error) {
	p, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return 0, err
	}
	var free, total, totalFree uint64
	if err := windows.GetDiskFreeSpaceEx(p, &free, &total, &totalFree); err != nil {
		return 0, err
	}
	return free, nil
}
