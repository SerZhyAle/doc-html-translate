package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunLogPathIsTimestamped(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())

	at := time.Date(2026, 7, 29, 13, 5, 9, 0, time.UTC)
	got := RunLogPath(at)
	if want := filepath.Join(LogsDir(), "run-20260729-130509.log"); got != want {
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
		p := RunLogPath(base.Add(time.Duration(i) * time.Minute))
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
