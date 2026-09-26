//go:build !windows

package pdf

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"doc-html-translate/internal/procrun"
)

// A pdftotext that never exits used to hang the conversion forever. Now it is stopped at its
// deadline together with anything it started, and the book converts through the pure-Go reader
// (done criterion 1 of ticket bugfix-external-process-bounds).
func TestExtractFallsBackWhenPDFToTextHangs(t *testing.T) {
	saved := procrun.PDFToText
	procrun.PDFToText = procrun.Budget{Base: 500 * time.Millisecond, Max: 500 * time.Millisecond}
	t.Cleanup(func() { procrun.PDFToText = saved })

	tmp := t.TempDir()
	binDir := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pidFile := filepath.Join(tmp, "child.pid")
	stub := "#!/bin/sh\nsleep 1000 &\necho $! > " + pidFile + "\nwait\n"
	if err := os.WriteFile(filepath.Join(binDir, "pdftotext"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if got := findPDFToText(); got != filepath.Join(binDir, "pdftotext") {
		t.Fatalf("findPDFToText = %q, want the stub", got)
	}

	pdfPath := filepath.Join(tmp, "book.pdf")
	createTestPDF(t, pdfPath, []string{"Chapter one content\nWith two lines"})
	out := filepath.Join(tmp, "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	book, err := Extract(context.Background(), pdfPath, out)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("Extract took %v with a 500ms pdftotext deadline", elapsed)
	}
	if len(book.Spine) == 0 {
		t.Error("the fallback produced no pages")
	}

	b, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("the stub never started its child: %v", err)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	deadline := time.Now().Add(2 * time.Second)
	for !childGone(pid) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if !childGone(pid) {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Errorf("pdftotext's child %d outlived the timeout", pid)
	}
}

// childGone treats a zombie as gone: a killed, re-parented process can linger unreaped.
func childGone(pid int) bool {
	if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
		return true
	}
	stat, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return os.IsNotExist(err)
	}
	s := string(stat)
	i := strings.LastIndexByte(s, ')')
	return i >= 0 && i+2 < len(s) && (s[i+2] == 'Z' || s[i+2] == 'X')
}
