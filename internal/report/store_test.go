package report

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunLogPathIsTimestamped(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())

	at := time.Date(2026, 7, 29, 13, 5, 9, 0, time.UTC)
	got := RunLogPath(at, 42)
	if want := filepath.Join(LogsDir(), "run-20260729-130509-42.log"); got != want {
		t.Fatalf("RunLogPath = %q, want %q", got, want)
	}
}

// writeLogs creates n dated run logs of size bytes each, oldest first, and returns their names.
func writeLogs(t *testing.T, n int, size int) []string {
	t.Helper()
	if err := os.MkdirAll(LogsDir(), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	names := make([]string, 0, n)
	for i := 0; i < n; i++ {
		p := RunLogPath(base.Add(time.Duration(i)*time.Minute), 1)
		if err := os.WriteFile(p, []byte(strings.Repeat("x", size)), 0o600); err != nil {
			t.Fatalf("write log: %v", err)
		}
		names = append(names, filepath.Base(p))
	}
	return names
}

func TestTrimKeepsNewestWithinBounds(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	names := writeLogs(t, 25, 16)

	removed, err := Trim()
	if err != nil {
		t.Fatalf("Trim: %v", err)
	}
	if removed != 5 {
		t.Errorf("Trim removed %d, want 5", removed)
	}
	left, err := os.ReadDir(LogsDir())
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(left) != MaxLogFiles {
		t.Fatalf("%d logs left, want %d", len(left), MaxLogFiles)
	}
	for _, gone := range names[:5] {
		if _, err := os.Stat(filepath.Join(LogsDir(), gone)); err == nil {
			t.Errorf("oldest log %s survived the trim", gone)
		}
	}
	for _, kept := range names[5:] {
		if _, err := os.Stat(filepath.Join(LogsDir(), kept)); err != nil {
			t.Errorf("newer log %s was removed", kept)
		}
	}
}

func TestTrimHonoursTheByteBound(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	// Three logs, each over half the budget: only the newest can stay.
	writeLogs(t, 3, (MaxLogBytes/2)+1)

	if _, err := Trim(); err != nil {
		t.Fatalf("Trim: %v", err)
	}
	left, err := os.ReadDir(LogsDir())
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(left) != 1 {
		t.Fatalf("%d logs left, want 1", len(left))
	}
}

func TestTrimOnMissingStoreIsNotAnError(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())

	removed, err := Trim()
	if err != nil || removed != 0 {
		t.Fatalf("Trim on a missing store = (%d, %v), want (0, nil)", removed, err)
	}
	if err := ClearLogs(); err != nil {
		t.Fatalf("ClearLogs on a missing store: %v", err)
	}
}

func TestClearLogsEmptiesDirectory(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	writeLogs(t, 4, 32)

	if err := ClearLogs(); err != nil {
		t.Fatalf("ClearLogs: %v", err)
	}
	left, err := os.ReadDir(LogsDir())
	if err != nil {
		t.Fatalf("ReadDir after ClearLogs: %v", err)
	}
	if len(left) != 0 {
		t.Fatalf("%d files left after ClearLogs, want 0", len(left))
	}
	if _, err := os.Stat(LogsDir()); err != nil {
		t.Errorf("ClearLogs removed the directory itself: %v", err)
	}
}

func TestDirFallsBackWithoutLocalAppData(t *testing.T) {
	t.Setenv("LOCALAPPDATA", "")

	if got := Dir(); !strings.Contains(got, "doc-html-translate") {
		t.Fatalf("Dir() = %q, want a doc-html-translate folder under the temp directory", got)
	}
}

// Two runs started in the same second must not share a file: with O_APPEND their lines would
// interleave into one log that belongs to neither.
func TestRunLogPathSeparatesSameSecondRuns(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	at := time.Date(2026, 7, 29, 13, 5, 9, 0, time.UTC)
	if a, b := RunLogPath(at, 100), RunLogPath(at, 101); a == b {
		t.Fatalf("same-second runs share %q", a)
	}
}

// numberedLines writes n lines of lineLen bytes each, the first digits of a line being its
// number, and returns what was written.
func numberedLines(t *testing.T, w io.Writer, n, lineLen int) string {
	t.Helper()
	var all strings.Builder
	for i := 0; i < n; i++ {
		line := fmt.Sprintf("%06d ", i) + strings.Repeat("x", lineLen-8) + "\n"
		if got, err := io.WriteString(w, line); err != nil || got != len(line) {
			t.Fatalf("write %d: n=%d err=%v, want a silent success", i, got, err)
		}
		all.WriteString(line)
	}
	return all.String()
}

// A run that fits the head is on disk as it is written: a process killed mid-run leaves it all.
func TestCapRunLogWritesTheHeadThrough(t *testing.T) {
	var buf strings.Builder
	w := CapRunLog(&buf)
	want := numberedLines(t, w, 100, 1024)
	if buf.String() != want {
		t.Fatalf("head not written through: %d bytes, want %d", buf.Len(), len(want))
	}
	w.Finish()
	if buf.String() != want {
		t.Errorf("Finish changed a log that fits: %d bytes, want %d", buf.Len(), len(want))
	}
}

// Past the head the lines are held, and a run that still fits head + tail loses nothing.
func TestCapRunLogKeepsWhatFitsHeadAndTail(t *testing.T) {
	var buf strings.Builder
	w := CapRunLog(&buf)
	want := numberedLines(t, w, 4*1024, 1024) // 4 MiB
	if buf.Len() >= len(want) {
		t.Errorf("%d bytes on disk before Finish, want only the head of %d", buf.Len(), len(want))
	}
	w.Finish()
	if buf.String() != want {
		t.Errorf("a log within head + tail changed: %d bytes, want %d", buf.Len(), len(want))
	}
}

func TestCapRunLogCompactsTheMiddleWithMarker(t *testing.T) {
	var buf strings.Builder
	w := CapRunLog(&buf)
	const lineLen = 1024
	total := 6 * 1024 // 6 MiB, over the cap
	written := numberedLines(t, w, total, lineLen)
	w.Finish()
	out := buf.String()

	if len(out) > MaxRunLogBytes {
		t.Fatalf("log is %d bytes, cap is %d", len(out), MaxRunLogBytes)
	}
	if !strings.HasPrefix(out, written[:RunLogHeadBytes]) {
		t.Error("the head is not the start of the run")
	}
	var dropped, head, tail int
	i := strings.Index(out, "[Diag] LOG COMPACTED")
	if i < 0 {
		t.Fatal("no compaction marker")
	}
	marker := out[i : i+strings.Index(out[i:], "\n")+1]
	if _, err := fmt.Sscanf(marker, "[Diag] LOG COMPACTED | dropped_middle_bytes=%d | kept_head_bytes=%d | kept_tail_bytes=%d\n", &dropped, &head, &tail); err != nil {
		t.Fatalf("marker %q: %v", marker, err)
	}
	if strings.Count(out, "LOG COMPACTED") != 1 {
		t.Errorf("want exactly one marker, got %d", strings.Count(out, "LOG COMPACTED"))
	}
	if head != i || head+dropped+tail != len(written) {
		t.Errorf("marker numbers do not add up: head=%d (marker at %d) dropped=%d tail=%d, run was %d bytes", head, i, dropped, tail, len(written))
	}
	if !strings.HasSuffix(out, written[len(written)-tail:]) || tail > runLogTailBytes {
		t.Errorf("the tail is not the end of the run (kept_tail_bytes=%d)", tail)
	}
	if !strings.HasSuffix(out, fmt.Sprintf("%06d ", total-1)+strings.Repeat("x", lineLen-8)+"\n") {
		t.Error("the last line of the run is missing")
	}
	for _, part := range []string{out[:i], out[i+len(marker):]} {
		if len(part)%lineLen != 0 {
			t.Errorf("a line was cut: a part is %d bytes, lines are %d", len(part), lineLen)
		}
	}
}

// A line longer than the whole tail cannot be kept, but it is counted, so the marker stays true.
func TestCapRunLogCountsALineTooLongForTheTail(t *testing.T) {
	var buf strings.Builder
	w := CapRunLog(&buf)
	numberedLines(t, w, 2, 8)
	huge := strings.Repeat("y", runLogTailBytes+1)
	_, _ = io.WriteString(w, huge)
	_, _ = io.WriteString(w, "end\n")
	w.Finish()
	want := fmt.Sprintf("[Diag] LOG COMPACTED | dropped_middle_bytes=%d | kept_head_bytes=%d | kept_tail_bytes=4\n", len(huge), 2*8)
	if !strings.Contains(buf.String(), want) || !strings.HasSuffix(buf.String(), want+"end\n") {
		t.Errorf("got %q", buf.String()[:min(buf.Len(), 200)])
	}
}

func TestCapRunLogFinishIsFinal(t *testing.T) {
	var buf strings.Builder
	w := CapRunLog(&buf)
	numberedLines(t, w, 2*1024, 1024)
	w.Finish()
	before := buf.Len()
	w.Finish()
	if buf.Len() != before {
		t.Errorf("a second Finish wrote %d bytes", buf.Len()-before)
	}
	_, _ = io.WriteString(w, "after\n")
	if !strings.HasSuffix(buf.String(), "after\n") {
		t.Error("a line written after Finish was lost")
	}
}
