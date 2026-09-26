//go:build !windows

package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"doc-html-translate/internal/procrun"
)

// P30: the run's context reaches the helper process. Ctrl+C during a Calibre conversion stops
// ebook-convert at once instead of at its deadline, and the run ends as interrupted, not as a
// file that failed to parse.
func TestCancelStopsHelperAndReportsInterrupted(t *testing.T) {
	s := newPipelineSandbox(t)
	saved := procrun.Calibre
	// Long enough that stopping at the deadline is told apart from stopping on cancel, short
	// enough that a regression costs seconds, not the real ten minutes.
	procrun.Calibre = procrun.Budget{Base: 5 * time.Second, Max: 5 * time.Second}
	t.Cleanup(func() { procrun.Calibre = saved })

	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "ebook-convert"), []byte("#!/bin/sh\nexec sleep 1000\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	in := s.input("book.mobi", "BOOKMOBI")
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(300*time.Millisecond, cancel)
	defer cancel()

	start := time.Now()
	code, err := s.runner(s.config(in)).RunContext(ctx)
	elapsed := time.Since(start)

	if code != ExitInterrupted {
		t.Errorf("code = %d (%v), want ExitInterrupted %d", code, err, ExitInterrupted)
	}
	if elapsed > 3*time.Second {
		t.Errorf("the run took %v after a cancel at 300ms: the helper waited for its deadline", elapsed)
	}
}
