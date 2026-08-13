package logging

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout swaps os.Stdout for a pipe and returns what was written while fn ran.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	saved := os.Stdout
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()
	fn()
	os.Stdout = saved
	_ = w.Close()
	out := <-done
	_ = r.Close()
	return out
}

func TestStartRunLogTeesEveryLevel(t *testing.T) {
	var sink bytes.Buffer
	StartRunLog(&sink)
	defer StopRunLog()

	_ = captureStdout(t, func() {
		Printf("printf-%s\n", "text")
		Println("println-text")
		Errorf("errorf-%s\n", "text")
		Progress("progress-%s", "text")
	})

	got := sink.String()
	for _, want := range []string{"printf-text", "println-text", "errorf-text", "progress-text"} {
		if !strings.Contains(got, want) {
			t.Errorf("run log missing %q; got %q", want, got)
		}
	}
	if strings.Contains(got, "\r") {
		t.Errorf("run log contains a carriage return: %q", got)
	}
}

func TestStopRunLogStopsWriting(t *testing.T) {
	var sink bytes.Buffer
	StartRunLog(&sink)
	_ = captureStdout(t, func() { Println("before") })
	StopRunLog()
	_ = captureStdout(t, func() { Println("after") })

	got := sink.String()
	if !strings.Contains(got, "before") {
		t.Fatalf("run log missing the line written while installed; got %q", got)
	}
	if strings.Contains(got, "after") {
		t.Errorf("run log kept writing after StopRunLog; got %q", got)
	}
}

// failingWriter stands in for a store that cannot be written - a full disk, a revoked
// permission, a read-only packaged install.
type failingWriter struct{ writes int }

func (f *failingWriter) Write(p []byte) (int, error) {
	f.writes++
	return 0, errors.New("sink is unavailable")
}

func TestRunLogWriteErrorIsIgnored(t *testing.T) {
	sink := &failingWriter{}
	StartRunLog(sink)
	defer StopRunLog()

	out := captureStdout(t, func() {
		Printf("printf-%s\n", "text")
		Println("println-text")
		Errorf("errorf-%s\n", "text")
		Progress("progress-%s", "text")
	})

	if sink.writes == 0 {
		t.Fatal("the failing sink was never written to, so the test proves nothing")
	}
	for _, want := range []string{"printf-text", "println-text", "progress-text"} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout missing %q after a failing run log; got %q", want, out)
		}
	}
}
