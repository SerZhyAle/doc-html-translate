//go:build windows

package main

import (
	"bytes"
	"io"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"doc-html-translate/internal/commands"
)

// C5: a first instance that holds the pipe (or never pumps/answers) must cause the second process
// to return inside the bound plus a margin with exit 11 (busy) or exit 2 (timeout) and never hang.
func TestInstanceHungPipeNeverHangsCaller(t *testing.T) {
	t.Run("held pipe returns exit 11 within delivery bound", func(t *testing.T) {
		pipeName, mutexName := newTestPipeMutex()
		lock, acquired, err := commands.AcquireInstanceLock(mutexName)
		if err != nil || !acquired {
			t.Fatalf("AcquireInstanceLock: acquired=%v err=%v", acquired, err)
		}
		defer lock.Release()

		namePtr, err := windows.UTF16PtrFromString(pipeName)
		if err != nil {
			t.Fatal(err)
		}
		h, err := windows.CreateNamedPipe(
			namePtr,
			0x00000003|windows.FILE_FLAG_OVERLAPPED,
			0,
			1,
			commands.MaxPipeLineBytes,
			commands.MaxPipeLineBytes,
			uint32(commands.DeliveryBound/time.Millisecond),
			(*windows.SecurityAttributes)(unsafe.Pointer(nil)),
		)
		if err != nil {
			t.Fatalf("CreateNamedPipe: %v", err)
		}
		defer func() { _ = windows.CloseHandle(h) }()

		// Occupy the single pipe instance so the next caller sees ERROR_PIPE_BUSY / held pipe.
		holder, err := windows.CreateFile(namePtr, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING, 0, 0)
		if err != nil {
			t.Fatalf("holder CreateFile: %v", err)
		}
		defer func() { _ = windows.CloseHandle(holder) }()

		var outBuf, errBuf bytes.Buffer
		env := commands.Env{
			Version:   "26.1116.0001",
			Open:      func() (io.Writer, io.Writer) { return &outBuf, &errBuf },
			Lang:      "en",
			PipeName:  pipeName,
			MutexName: mutexName,
		}

		start := time.Now()
		handled, code := commands.Front([]string{"status"}, env)
		elapsed := time.Since(start)
		if !handled || code != commands.ExitNotFound {
			t.Fatalf("held pipe status: handled=%v code=%d elapsed=%v err=%q", handled, code, elapsed, errBuf.String())
		}
		if elapsed > commands.DeliveryBound+500*time.Millisecond {
			t.Fatalf("held pipe took %v, exceeded delivery bound + margin", elapsed)
		}
	})

	t.Run("unanswered pipe returns exit 2 at completion bound", func(t *testing.T) {
		pipeName, mutexName := newTestPipeMutex()
		lock, acquired, err := commands.AcquireInstanceLock(mutexName)
		if err != nil || !acquired {
			t.Fatalf("AcquireInstanceLock: acquired=%v err=%v", acquired, err)
		}
		defer lock.Release()

		namePtr, err := windows.UTF16PtrFromString(pipeName)
		if err != nil {
			t.Fatal(err)
		}
		h, err := windows.CreateNamedPipe(
			namePtr,
			0x00000003|windows.FILE_FLAG_OVERLAPPED,
			0,
			1,
			commands.MaxPipeLineBytes,
			commands.MaxPipeLineBytes,
			uint32(commands.DeliveryBound/time.Millisecond),
			(*windows.SecurityAttributes)(unsafe.Pointer(nil)),
		)
		if err != nil {
			t.Fatalf("CreateNamedPipe: %v", err)
		}
		defer func() { _ = windows.CloseHandle(h) }()

		var outBuf, errBuf bytes.Buffer
		env := commands.Env{
			Version:   "26.1116.0001",
			Open:      func() (io.Writer, io.Writer) { return &outBuf, &errBuf },
			Lang:      "en",
			PipeName:  pipeName,
			MutexName: mutexName,
		}

		start := time.Now()
		handled, code := commands.Front([]string{"status"}, env)
		elapsed := time.Since(start)
		if !handled || code != commands.ExitNotProven {
			t.Fatalf("unanswered pipe status: handled=%v code=%d elapsed=%v err=%q", handled, code, elapsed, errBuf.String())
		}
		if elapsed > 2500*time.Millisecond {
			t.Fatalf("unanswered pipe took %v, exceeded status timeout + margin", elapsed)
		}
	})
}
