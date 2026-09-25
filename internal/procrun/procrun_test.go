package procrun

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestCappedBufferKeepsTheHeadAndReportsFullWrites(t *testing.T) {
	b := &cappedBuffer{limit: 5}
	for _, chunk := range []string{"abc", "defg", "hij"} {
		n, err := b.Write([]byte(chunk))
		if err != nil || n != len(chunk) {
			t.Fatalf("Write(%q) = %d, %v; a capped write must still report success", chunk, n, err)
		}
	}
	if string(b.buf) != "abcde" || !b.truncated {
		t.Errorf("buf = %q truncated = %v, want \"abcde\" true", b.buf, b.truncated)
	}
}

func TestBudgetScalesWithSizeAndClamps(t *testing.T) {
	b := Budget{Base: time.Minute, PerMB: time.Minute, Max: 5 * time.Minute}
	if got := b.For(0); got != time.Minute {
		t.Errorf("For(0) = %v, want 1m", got)
	}
	if got := b.For(2 << 20); got != 3*time.Minute {
		t.Errorf("For(2 MB) = %v, want 3m", got)
	}
	if got := b.For(100 << 20); got != 5*time.Minute {
		t.Errorf("For(100 MB) = %v, want the 5m ceiling", got)
	}
}

func TestParseScale(t *testing.T) {
	cases := map[string]float64{"": 1, "2": 2, "0.5": 0.5, "abc": 1, "0": 1, "-3": 1, "NaN": 1, "1e9": 1}
	for in, want := range cases {
		if got := parseScale(in); got != want {
			t.Errorf("parseScale(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestRunMissingBinaryKeepsThePathError(t *testing.T) {
	_, err := Run(context.Background(), Cmd{Tool: "nothing", Path: "/definitely/not/here/tool", Timeout: time.Second})
	var pe *os.PathError
	if !errors.As(err, &pe) {
		t.Fatalf("err = %v, want an *os.PathError in the chain", err)
	}
	var re *Error
	if !errors.As(err, &re) || re.Tool != "nothing" {
		t.Errorf("err = %v, want a *procrun.Error naming the tool", err)
	}
}

func TestRunNonZeroExitKeepsTheExitError(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("needs sh")
	}
	_, err := Run(context.Background(), Cmd{Tool: "sh", Path: "sh", Args: []string{"-c", "echo boom >&2; exit 3"}, Timeout: 10 * time.Second})
	var ee *exec.ExitError
	if !errors.As(err, &ee) || ee.ExitCode() != 3 {
		t.Fatalf("err = %v, want exit status 3", err)
	}
	if errors.Is(err, ErrTimeout) {
		t.Error("a failed run must not read as a timeout")
	}
	var re *Error
	if errors.As(err, &re) && re.Stderr != "boom" {
		t.Errorf("Stderr = %q, want \"boom\"", re.Stderr)
	}
}
