//go:build !windows

package procrun

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// processGone reports whether pid no longer runs. A killed grandchild is re-parented and may
// sit as a zombie until something reaps it, which is dead for every purpose that matters here.
func processGone(pid int) bool {
	if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
		return true
	}
	stat, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return os.IsNotExist(err)
	}
	// The state is the field after the parenthesised command name.
	if i := bytes.LastIndexByte(stat, ')'); i >= 0 && i+2 < len(stat) {
		return stat[i+2] == 'Z' || stat[i+2] == 'X'
	}
	return false
}

func waitGone(pid int, within time.Duration) bool {
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if processGone(pid) {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return processGone(pid)
}

func readPID(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(path); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(b))); err == nil {
				return pid
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("the stub never wrote %s", path)
	return 0
}

// A helper that never exits is stopped at its deadline, and so is the process it started:
// the hung pdftotext of done criterion 1, with a grandchild to prove the whole tree goes.
func TestRunKillsTheTreeAtTheDeadline(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "grandchild.pid")
	start := time.Now()
	_, err := Run(context.Background(), Cmd{
		Tool:    "stub",
		Path:    "sh",
		Args:    []string{"-c", "sleep 1000 & echo $! > " + pidFile + "; wait"},
		Timeout: 500 * time.Millisecond,
	})
	elapsed := time.Since(start)

	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("err = %v, want a timeout", err)
	}
	if !strings.Contains(err.Error(), "stub") {
		t.Errorf("message %q does not name the tool", err)
	}
	if elapsed > 3*time.Second {
		t.Errorf("Run returned after %v; the deadline was 500ms", elapsed)
	}
	if pid := readPID(t, pidFile); !waitGone(pid, 2*time.Second) {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Errorf("grandchild %d is still running after the timeout", pid)
	}
}

// Cancelling the context is the other way into the same kill path.
func TestRunKillsTheTreeOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(300 * time.Millisecond); cancel() }()
	_, err := Run(ctx, Cmd{Tool: "stub", Path: "sleep", Args: []string{"1000"}, Timeout: time.Minute})
	var re *Error
	if !errors.As(err, &re) || !re.Canceled {
		t.Fatalf("err = %v, want a cancelled *procrun.Error", err)
	}
}

// A helper that exits but leaves a child holding its stdout must not keep Run waiting for
// that child, and the child must not outlive the call.
func TestRunDoesNotHangOnAnInheritedPipe(t *testing.T) {
	saved := waitDelay
	waitDelay = 500 * time.Millisecond
	t.Cleanup(func() { waitDelay = saved })
	pidFile := filepath.Join(t.TempDir(), "leftover.pid")
	start := time.Now()
	_, err := Run(context.Background(), Cmd{
		Tool:    "stub",
		Path:    "sh",
		Args:    []string{"-c", "sleep 1000 & echo $! > " + pidFile},
		Timeout: time.Minute,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if elapsed := time.Since(start); elapsed > waitDelay+3*time.Second {
		t.Errorf("Run took %v; an inherited pipe must not hold it", elapsed)
	}
	if pid := readPID(t, pidFile); !waitGone(pid, 2*time.Second) {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Errorf("leftover child %d survived the call", pid)
	}
}

// A helper that floods stdout is capped, not buffered whole.
func TestRunBoundsCapturedOutput(t *testing.T) {
	res, err := Run(context.Background(), Cmd{
		Tool:      "stub",
		Path:      "sh",
		Args:      []string{"-c", "head -c 8388608 /dev/zero; head -c 1048576 /dev/zero >&2"},
		Timeout:   30 * time.Second,
		MaxStdout: 1 << 20,
		MaxStderr: 1 << 10,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Stdout) != 1<<20 || !res.StdoutTruncated {
		t.Errorf("stdout: %d bytes truncated=%v, want %d true", len(res.Stdout), res.StdoutTruncated, 1<<20)
	}
	if len(res.Stderr) != 1<<10 || !res.StderrTruncated {
		t.Errorf("stderr: %d bytes truncated=%v, want %d true", len(res.Stderr), res.StderrTruncated, 1<<10)
	}
}

func TestRunPassesExtraEnv(t *testing.T) {
	res, err := Run(context.Background(), Cmd{
		Tool: "stub", Path: "sh", Args: []string{"-c", "printf %s \"$OMP_THREAD_LIMIT\""},
		Env: []string{"OMP_THREAD_LIMIT=1"}, Timeout: 10 * time.Second,
	})
	if err != nil || string(res.Stdout) != "1" {
		t.Fatalf("stdout = %q, err = %v; want the extra variable passed through", res.Stdout, err)
	}
}
