//go:build windows

package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"doc-html-translate/internal/i18n"
)

const (
	defaultMutexName = `Local\DocHtmlTranslateSingleInstance`

	pipeRejectRemoteClients = 0x00000008
	pipeWait                = 0x00000000
	pipeReadModeByte        = 0x00000000
	pipeTypeByte            = 0x00000000
	pipeAccessDuplex        = 0x00000003
)

var (
	kernel32Pipe                 = windows.NewLazySystemDLL("kernel32.dll")
	procWaitNamedPipeW           = kernel32Pipe.NewProc("WaitNamedPipeW")
	procGetNamedPipeClientPID    = kernel32Pipe.NewProc("GetNamedPipeClientProcessId")
	procGetNamedPipeServerPID    = kernel32Pipe.NewProc("GetNamedPipeServerProcessId")
	user32Activation             = windows.NewLazySystemDLL("user32.dll")
	procAllowSetForegroundWindow = user32Activation.NewProc("AllowSetForegroundWindow")
)

// DefaultMutexName returns the per-session single-instance mutex name (APP-ACTIVATION rule 2).
func DefaultMutexName() string { return defaultMutexName }

// DefaultPipeName returns the per-user, per-session command pipe name (APP-ACTIVATION rule 3, third shape).
func DefaultPipeName() string {
	sid, session, _ := currentIdentity()
	return fmt.Sprintf(`\\.\pipe\DocHtmlTranslate_Command_%s_%d`, sid, session)
}

func currentIdentity() (userSID string, sessionID uint32, elevated bool) {
	tok := windows.GetCurrentProcessToken()
	if tu, err := tok.GetTokenUser(); err == nil && tu != nil && tu.User.Sid != nil {
		userSID = tu.User.Sid.String()
	}
	_ = windows.ProcessIdToSessionId(windows.GetCurrentProcessId(), &sessionID)
	elevated = tok.IsElevated()
	return userSID, sessionID, elevated
}

type winMutex struct {
	h windows.Handle
}

func (m *winMutex) Release() {
	if m == nil || m.h == 0 || m.h == windows.InvalidHandle {
		return
	}
	_ = windows.ReleaseMutex(m.h)
	_ = windows.CloseHandle(m.h)
	m.h = 0
}

// AcquireInstanceLock attempts to acquire the single-instance mutex. acquired is false when another instance
// in the same session already holds it.
func AcquireInstanceLock(mutexName string) (lock InstanceLock, acquired bool, err error) {
	if mutexName == "" {
		mutexName = DefaultMutexName()
	}
	p, err := windows.UTF16PtrFromString(mutexName)
	if err != nil {
		return nil, false, err
	}
	h, err := windows.CreateMutex(nil, true, p)
	if err != nil {
		if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
			if h != 0 && h != windows.InvalidHandle {
				_ = windows.CloseHandle(h)
			}
			return nil, false, nil
		}
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &winMutex{h: h}, true, nil
}

func mutexExists(mutexName string) (exists bool, accessDenied bool) {
	if mutexName == "" {
		return false, false
	}
	p, err := windows.UTF16PtrFromString(mutexName)
	if err != nil {
		return false, false
	}
	h, err := windows.OpenMutex(windows.SYNCHRONIZE, false, p)
	if err != nil {
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			return true, true
		}
		return false, false
	}
	_ = windows.CloseHandle(h)
	return true, false
}

// PipeServer listens on the local named pipe for one command at a time.
type PipeServer struct {
	pipeName string
	stopEvt  windows.Handle
	done     chan struct{}
}

// StartPipeServer creates and runs the named-pipe listener on pipeName.
func StartPipeServer(pipeName string, handler func(line []byte) (reply []byte, afterSend func()), lang string) (*PipeServer, error) {
	if pipeName == "" {
		pipeName = DefaultPipeName()
	}
	stopEvt, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return nil, err
	}
	userSID, sessionID, elevated := currentIdentity()
	h, err := createServerPipe(pipeName, userSID)
	if err != nil {
		_ = windows.CloseHandle(stopEvt)
		return nil, err
	}
	ps := &PipeServer{
		pipeName: pipeName,
		stopEvt:  stopEvt,
		done:     make(chan struct{}),
	}
	go ps.serve(h, userSID, sessionID, elevated, handler, lang)
	return ps, nil
}

// Close stops the pipe listener and waits for its goroutine to exit.
func (ps *PipeServer) Close() {
	if ps == nil {
		return
	}
	if ps.stopEvt != 0 {
		_ = windows.SetEvent(ps.stopEvt)
		<-ps.done
		_ = windows.CloseHandle(ps.stopEvt)
		ps.stopEvt = 0
	}
}

func createServerPipe(pipeName, userSID string) (windows.Handle, error) {
	namePtr, err := windows.UTF16PtrFromString(pipeName)
	if err != nil {
		return 0, err
	}
	var sa *windows.SecurityAttributes
	if userSID != "" {
		// Admit current user only; allow Medium-IL callers to connect so an elevated instance can answer `elevation` (12).
		sddl := fmt.Sprintf("D:P(A;;GA;;;%s)S:(ML;;NW;;;ME)", userSID)
		sd, err := windows.SecurityDescriptorFromString(sddl)
		if err != nil {
			sddl = fmt.Sprintf("D:P(A;;GA;;;%s)", userSID)
			sd, err = windows.SecurityDescriptorFromString(sddl)
		}
		if err == nil {
			sa = &windows.SecurityAttributes{
				Length:             uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
				SecurityDescriptor: sd,
			}
		}
	}
	h, err := windows.CreateNamedPipe(
		namePtr,
		pipeAccessDuplex|windows.FILE_FLAG_OVERLAPPED,
		pipeTypeByte|pipeReadModeByte|pipeWait|pipeRejectRemoteClients,
		8,
		MaxPipeLineBytes+1024,
		MaxPipeLineBytes+1024,
		uint32(DeliveryBound/time.Millisecond),
		sa,
	)
	if err != nil && sa != nil && userSID != "" {
		// Fall back to DACL-only if a sandboxed integrity level refuses the SACL.
		if sd, sdErr := windows.SecurityDescriptorFromString(fmt.Sprintf("D:P(A;;GA;;;%s)", userSID)); sdErr == nil {
			sa.SecurityDescriptor = sd
			h, err = windows.CreateNamedPipe(
				namePtr,
				pipeAccessDuplex|windows.FILE_FLAG_OVERLAPPED,
				pipeTypeByte|pipeReadModeByte|pipeWait|pipeRejectRemoteClients,
				8,
				MaxPipeLineBytes+1024,
				MaxPipeLineBytes+1024,
				uint32(DeliveryBound/time.Millisecond),
				sa,
			)
		}
	}
	return h, err
}

func (ps *PipeServer) serve(h windows.Handle, serverSID string, serverSession uint32, serverElevated bool, handler func([]byte) ([]byte, func()), lang string) {
	defer close(ps.done)
	defer func() { _ = windows.CloseHandle(h) }()

	connEvt, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return
	}
	defer func() { _ = windows.CloseHandle(connEvt) }()

	for {
		_ = windows.ResetEvent(connEvt)
		ov := windows.Overlapped{HEvent: connEvt}
		err := windows.ConnectNamedPipe(h, &ov)
		connected := false
		switch {
		case err == nil:
			connected = true
		case errors.Is(err, windows.ERROR_PIPE_CONNECTED):
			connected = true
		case errors.Is(err, windows.ERROR_IO_PENDING):
			ev, waitErr := windows.WaitForMultipleObjects([]windows.Handle{ps.stopEvt, connEvt}, false, windows.INFINITE)
			if waitErr != nil || ev == windows.WAIT_OBJECT_0 {
				_ = windows.CancelIoEx(h, &ov)
				return
			}
			var n uint32
			if windows.GetOverlappedResult(h, &ov, &n, false) == nil {
				connected = true
			}
		default:
			if ev, _ := windows.WaitForSingleObject(ps.stopEvt, 20); ev == windows.WAIT_OBJECT_0 {
				return
			}
			continue
		}

		if connected {
			afterSend := ps.handleConnection(h, serverSID, serverSession, serverElevated, handler, lang)
			_ = windows.DisconnectNamedPipe(h)
			if afterSend != nil {
				afterSend()
				return
			}
		}
		if ev, _ := windows.WaitForSingleObject(ps.stopEvt, 0); ev == windows.WAIT_OBJECT_0 {
			return
		}
	}
}

func (ps *PipeServer) handleConnection(h windows.Handle, serverSID string, serverSession uint32, serverElevated bool, handler func([]byte) ([]byte, func()), lang string) (afterSend func()) {
	defer func() {
		_ = recover()
	}()

	allowed, clientElevated := checkPipeClient(h, serverSID, serverSession)
	if !allowed {
		return nil
	}

	line, ok := readOverlappedLine(h, DeliveryBound, ps.stopEvt)
	if !ok {
		return nil
	}

	if serverElevated && !clientElevated {
		reqID := extractRequestID(line)
		reply := encodePipeResponse(lang, Response{
			SchemaVersion: SchemaVersion,
			RequestID:     reqID,
			Status:        StatusRefused,
			ExitCode:      ExitPermission,
			Reason:        ReasonElevation,
			Message:       i18n.T(lang, msgElevationRefused),
		})
		_ = writeOverlapped(h, reply, DeliveryBound, ps.stopEvt)
		_ = windows.FlushFileBuffers(h)
		return nil
	}

	reply, hook := handler(line)
	_ = writeOverlapped(h, reply, DeliveryBound, ps.stopEvt)
	_ = windows.FlushFileBuffers(h)
	return hook
}

func extractRequestID(line []byte) string {
	var partial struct {
		RequestID string `json:"requestId"`
	}
	if json.Unmarshal(bytes.TrimRight(line, "\r\n"), &partial) == nil {
		return partial.RequestID
	}
	return ""
}

func checkPipeClient(h windows.Handle, serverSID string, serverSession uint32) (allowed bool, elevated bool) {
	var pid uint32
	r1, _, _ := procGetNamedPipeClientPID.Call(uintptr(h), uintptr(unsafe.Pointer(&pid)))
	if r1 == 0 || pid == 0 {
		return false, false
	}
	var sess uint32
	if err := windows.ProcessIdToSessionId(pid, &sess); err != nil || sess != serverSession {
		return false, false
	}
	proc, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return false, false
	}
	defer func() { _ = windows.CloseHandle(proc) }()
	var tok windows.Token
	if err := windows.OpenProcessToken(proc, windows.TOKEN_QUERY, &tok); err != nil {
		return false, false
	}
	defer tok.Close()
	if serverSID != "" {
		tu, err := tok.GetTokenUser()
		if err != nil || tu == nil || tu.User.Sid == nil || tu.User.Sid.String() != serverSID {
			return false, false
		}
	}
	return true, tok.IsElevated()
}

// dialAndExchange connects to the running first instance over pipeName, sends reqLine, and waits for one response line.
func dialAndExchange(pipeName, mutexName string, req Request, deliveryBound, completionBound time.Duration, allowFg bool, lang string) Response {
	if pipeName == "" {
		pipeName = DefaultPipeName()
	}
	if deliveryBound <= 0 {
		deliveryBound = DeliveryBound
	}
	if completionBound <= 0 {
		completionBound = 10 * time.Second
	}
	if mutexName != "" {
		exists, _ := mutexExists(mutexName)
		if !exists {
			return Response{
				SchemaVersion: SchemaVersion,
				RequestID:     req.RequestID,
				Status:        StatusRefused,
				ExitCode:      ExitNotFound,
				Reason:        ReasonNotRunning,
				Message:       i18n.T(lang, msgNotRunning),
			}
		}
	}

	h, connectErr := connectPipeBounded(pipeName, deliveryBound)
	if connectErr != nil {
		if errors.Is(connectErr, windows.ERROR_ACCESS_DENIED) {
			return Response{
				SchemaVersion: SchemaVersion,
				RequestID:     req.RequestID,
				Status:        StatusRefused,
				ExitCode:      ExitPermission,
				Reason:        ReasonElevation,
				Message:       i18n.T(lang, msgElevationRefused),
			}
		}
		if mutexName != "" {
			if exists, _ := mutexExists(mutexName); !exists {
				return Response{
					SchemaVersion: SchemaVersion,
					RequestID:     req.RequestID,
					Status:        StatusRefused,
					ExitCode:      ExitNotFound,
					Reason:        ReasonNotRunning,
					Message:       i18n.T(lang, msgNotRunning),
				}
			}
		}
		return Response{
			SchemaVersion: SchemaVersion,
			RequestID:     req.RequestID,
			Status:        StatusBusy,
			ExitCode:      ExitNotFound,
			Reason:        ReasonBusy,
			Message:       i18n.T(lang, msgLauncherBusy),
		}
	}
	defer func() { _ = windows.CloseHandle(h) }()

	if allowFg {
		var serverPID uint32
		if r1, _, _ := procGetNamedPipeServerPID.Call(uintptr(h), uintptr(unsafe.Pointer(&serverPID))); r1 != 0 && serverPID != 0 {
			_, _, _ = procAllowSetForegroundWindow.Call(uintptr(serverPID))
		}
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(req)
	reqBytes := buf.Bytes()
	if len(bytes.TrimRight(reqBytes, "\r\n")) > MaxPipeLineBytes {
		return Response{
			SchemaVersion: SchemaVersion,
			RequestID:     req.RequestID,
			Status:        StatusRefused,
			ExitCode:      ExitUsage,
			Reason:        ReasonTooLarge,
			Message:       i18n.T(lang, msgTooLarge),
		}
	}

	if err := writeOverlapped(h, reqBytes, deliveryBound, 0); err != nil {
		if errors.Is(err, syscall.ETIME) || errors.Is(err, windows.ERROR_SEM_TIMEOUT) {
			return Response{
				SchemaVersion: SchemaVersion,
				RequestID:     req.RequestID,
				Status:        StatusBusy,
				ExitCode:      ExitNotFound,
				Reason:        ReasonBusy,
				Message:       i18n.T(lang, msgLauncherBusy),
			}
		}
		return Response{
			SchemaVersion: SchemaVersion,
			RequestID:     req.RequestID,
			Status:        StatusFailed,
			ExitCode:      ExitNotProven,
			Reason:        ReasonSchema,
			Message:       i18n.T(lang, msgOlderInstance),
		}
	}

	rawReply, timedOut, ok := readOverlappedLineStatus(h, completionBound, 0)
	if timedOut {
		return Response{
			SchemaVersion: SchemaVersion,
			RequestID:     req.RequestID,
			Status:        StatusFailed,
			ExitCode:      ExitNotProven,
			Reason:        ReasonTimeout,
			Message:       i18n.T(lang, msgTimeout),
		}
	}
	if !ok {
		return Response{
			SchemaVersion: SchemaVersion,
			RequestID:     req.RequestID,
			Status:        StatusFailed,
			ExitCode:      ExitNotProven,
			Reason:        ReasonSchema,
			Message:       i18n.T(lang, msgOlderInstance),
		}
	}
	var resp Response
	if err := json.Unmarshal(bytes.TrimRight(rawReply, "\r\n"), &resp); err != nil || resp.SchemaVersion < 1 || resp.RequestID != req.RequestID {
		return Response{
			SchemaVersion: SchemaVersion,
			RequestID:     req.RequestID,
			Status:        StatusFailed,
			ExitCode:      ExitNotProven,
			Reason:        ReasonSchema,
			Message:       i18n.T(lang, msgOlderInstance),
		}
	}
	return resp
}

func connectPipeBounded(pipeName string, bound time.Duration) (windows.Handle, error) {
	namePtr, err := windows.UTF16PtrFromString(pipeName)
	if err != nil {
		return 0, err
	}
	deadline := time.Now().Add(bound)
	for {
		h, err := windows.CreateFile(
			namePtr,
			windows.GENERIC_READ|windows.GENERIC_WRITE,
			0,
			nil,
			windows.OPEN_EXISTING,
			windows.FILE_FLAG_OVERLAPPED,
			0,
		)
		if err == nil {
			return h, nil
		}
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			return 0, err
		}
		rem := time.Until(deadline)
		if rem <= 0 {
			return 0, err
		}
		if errors.Is(err, windows.ERROR_PIPE_BUSY) {
			ms := uint32(rem / time.Millisecond)
			if ms < 1 {
				ms = 1
			}
			_, _, _ = procWaitNamedPipeW.Call(uintptr(unsafe.Pointer(namePtr)), uintptr(ms))
			continue
		}
		if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
			sleep := 15 * time.Millisecond
			if rem < sleep {
				sleep = rem
			}
			time.Sleep(sleep)
			continue
		}
		return 0, err
	}
}

func writeOverlapped(h windows.Handle, data []byte, bound time.Duration, stopEvt windows.Handle) error {
	evt, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(evt) }()

	deadline := time.Now().Add(bound)
	written := 0
	for written < len(data) {
		_ = windows.ResetEvent(evt)
		ov := windows.Overlapped{HEvent: evt}
		var n uint32
		err := windows.WriteFile(h, data[written:], &n, &ov)
		if err == nil {
			written += int(n)
			continue
		}
		if !errors.Is(err, windows.ERROR_IO_PENDING) {
			return err
		}
		rem := time.Until(deadline)
		if rem <= 0 {
			_ = windows.CancelIoEx(h, &ov)
			return windows.ERROR_SEM_TIMEOUT
		}
		ms := uint32(rem / time.Millisecond)
		if ms < 1 {
			ms = 1
		}
		handles := []windows.Handle{evt}
		if stopEvt != 0 {
			handles = append(handles, stopEvt)
		}
		ev, werr := windows.WaitForMultipleObjects(handles, false, ms)
		if werr != nil || ev == uint32(windows.WAIT_TIMEOUT) || (stopEvt != 0 && ev == windows.WAIT_OBJECT_0+1) {
			_ = windows.CancelIoEx(h, &ov)
			return windows.ERROR_SEM_TIMEOUT
		}
		if err := windows.GetOverlappedResult(h, &ov, &n, false); err != nil {
			return err
		}
		written += int(n)
	}
	return nil
}

func readOverlappedLine(h windows.Handle, bound time.Duration, stopEvt windows.Handle) ([]byte, bool) {
	line, _, ok := readOverlappedLineStatus(h, bound, stopEvt)
	return line, ok
}

func readOverlappedLineStatus(h windows.Handle, bound time.Duration, stopEvt windows.Handle) (line []byte, timedOut bool, ok bool) {
	evt, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return nil, false, false
	}
	defer func() { _ = windows.CloseHandle(evt) }()

	deadline := time.Now().Add(bound)
	var out bytes.Buffer
	buf := make([]byte, 4096)
	for {
		_ = windows.ResetEvent(evt)
		ov := windows.Overlapped{HEvent: evt}
		var n uint32
		err := windows.ReadFile(h, buf, &n, &ov)
		if errors.Is(err, windows.ERROR_IO_PENDING) {
			rem := time.Until(deadline)
			if rem <= 0 {
				_ = windows.CancelIoEx(h, &ov)
				return nil, true, false
			}
			ms := uint32(rem / time.Millisecond)
			if ms < 1 {
				ms = 1
			}
			handles := []windows.Handle{evt}
			if stopEvt != 0 {
				handles = append(handles, stopEvt)
			}
			ev, werr := windows.WaitForMultipleObjects(handles, false, ms)
			if werr != nil || ev == uint32(windows.WAIT_TIMEOUT) {
				_ = windows.CancelIoEx(h, &ov)
				return nil, true, false
			}
			if stopEvt != 0 && ev == windows.WAIT_OBJECT_0+1 {
				_ = windows.CancelIoEx(h, &ov)
				return nil, false, false
			}
			err = windows.GetOverlappedResult(h, &ov, &n, false)
		}
		if n > 0 {
			chunk := buf[:n]
			if idx := bytes.IndexByte(chunk, '\n'); idx >= 0 {
				out.Write(chunk[:idx+1])
				return out.Bytes(), false, true
			}
			out.Write(chunk)
			if out.Len() > MaxPipeLineBytes+2 {
				return out.Bytes(), false, true
			}
		}
		if err != nil {
			if out.Len() > 0 && strings.TrimSpace(out.String()) != "" {
				return out.Bytes(), false, true
			}
			return nil, false, false
		}
	}
}
