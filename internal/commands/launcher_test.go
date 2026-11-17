package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestLauncherManifestValidAndHasLauncherVerbs(t *testing.T) {
	if problems := Check(Embedded()); len(problems) > 0 {
		t.Fatalf("Check(Embedded()) problems: %v", problems)
	}
	m, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct {
		name       string
		exe        string
		defaultFor string
		coldStart  string
	}{
		{"status", "", "", "refuse"},
		{"show", "doc-html-ui.exe", "", "start"},
		{"quit", "", "", "oneshot"},
		{"open", "doc-html-ui.exe", "doc-html-ui.exe", "start"},
	} {
		c := m.Find(want.name)
		if c == nil {
			t.Fatalf("manifest missing %q", want.name)
		}
		if c.Exe != want.exe || c.DefaultFor != want.defaultFor || c.ColdStart != want.coldStart || c.Forward != "always" {
			t.Errorf("%q: exe=%q defaultFor=%q coldStart=%q forward=%q", want.name, c.Exe, c.DefaultFor, c.ColdStart, c.Forward)
		}
	}
}

func TestLauncherOnlyVerbsRefusedByConsoleAndConsoleVerbsRefusedByLauncher(t *testing.T) {
	for _, verb := range [][]string{{"show"}, {"open", "--in", "x"}} {
		var c capture
		handled, code := Front(verb, c.env())
		if !handled || code != ExitUsage || !strings.Contains(c.err.String(), "doc-html-ui.exe") {
			t.Errorf("%v on console: handled=%v code=%d stderr=%q", verb, handled, code, c.err.String())
		}
	}
	for _, verb := range [][]string{{"convert", "a.epub"}, {"ocr", "download", "--lang", "rus"}} {
		var c capture
		env := c.env()
		env.Launcher = &LauncherHooks{BypassSingleInstance: true}
		handled, code := Front(verb, env)
		if !handled || code != ExitUsage || !strings.Contains(c.err.String(), "doc-html-translate.exe") {
			t.Errorf("%v on launcher: handled=%v code=%d stderr=%q", verb, handled, code, c.err.String())
		}
	}
}

type stubInstanceHandler struct {
	windowOpen bool
	activeRuns int
	focused    bool
	reopened   bool
	cancelled  int
	opened     [][]string
	onQuit     func()
}

func (s *stubInstanceHandler) Status(_ context.Context) (StatusData, int, string, string) {
	return StatusData{
		Version:                   testVersion,
		PID:                       os.Getpid(),
		WindowOpen:                s.windowOpen,
		ActiveRuns:                s.activeRuns,
		AcceptsRestrictedCommands: false,
	}, ExitDone, "", MsgStatusReady
}

func (s *stubInstanceHandler) Show(_ context.Context) (ShowData, int, string, string) {
	return ShowData{Focused: s.focused, Reopened: s.reopened}, ExitDone, "", MsgShowDone
}

func (s *stubInstanceHandler) Quit(_ context.Context, cancelRuns bool) (QuitData, int, string, string, func()) {
	if s.activeRuns > 0 && !cancelRuns {
		return QuitData{}, ExitNotFound, ReasonBusy, MsgQuitBusy, nil
	}
	s.cancelled = s.activeRuns
	s.activeRuns = 0
	return QuitData{CancelledRuns: s.cancelled}, ExitDone, "", MsgQuitDone, s.onQuit
}

func (s *stubInstanceHandler) Open(_ context.Context, paths []string) (OpenData, int, string, string) {
	s.opened = append(s.opened, paths)
	return OpenData{Queued: len(paths), Reopened: s.reopened}, ExitDone, "", MsgOpenDone
}

var testPipeSeq atomic.Int64

func uniqueTestPipeAndMutex() (string, string) {
	n := testPipeSeq.Add(1)
	pid := os.Getpid()
	return fmt.Sprintf(`\\.\pipe\DocHtmlTranslate_Test_%d_%d`, pid, n),
		fmt.Sprintf(`Local\DocHtmlTranslate_Test_%d_%d`, pid, n)
}

func TestHandlePipeRequestStrictParsing(t *testing.T) {
	m, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	h := &stubInstanceHandler{windowOpen: true}

	badCases := []struct {
		name     string
		line     []byte
		wantExit int
		wantReas string
	}{
		{"oversized", bytes.Repeat([]byte("a"), MaxPipeLineBytes+1), ExitUsage, ReasonTooLarge},
		{"empty", []byte("\n"), ExitUsage, ReasonBadArgument},
		{"not-json", []byte("not json\n"), ExitUsage, ReasonBadArgument},
		{"missing-request-id", []byte(`{"schemaVersion":1,"command":"status"}`), ExitUsage, ReasonBadArgument},
		{"newer-schema", []byte(`{"schemaVersion":99,"requestId":"r1","command":"status"}`), ExitNotProven, ReasonSchema},
		{"unknown-verb", []byte(`{"schemaVersion":1,"requestId":"r2","command":"frobnicate"}`), ExitUsage, ReasonUnknownVerb},
		{"non-forwarded-verb", []byte(`{"schemaVersion":1,"requestId":"r3","command":"version"}`), ExitUsage, ReasonUnknownVerb},
		{"unknown-option", []byte(`{"schemaVersion":1,"requestId":"r4","command":"status","args":["--nope"]}`), ExitUsage, ReasonBadArgument},
		{"open-no-in", []byte(`{"schemaVersion":1,"requestId":"r5","command":"open"}`), ExitUsage, ReasonBadArgument},
	}
	for _, tc := range badCases {
		reply, _ := HandlePipeRequest(context.Background(), m, h, "en", tc.line)
		var resp Response
		if err := json.Unmarshal(bytes.TrimSpace(reply), &resp); err != nil {
			t.Fatalf("%s: invalid reply JSON %q: %v", tc.name, reply, err)
		}
		if resp.ExitCode != tc.wantExit || resp.Reason != tc.wantReas {
			t.Errorf("%s: got exit=%d reason=%q, want exit=%d reason=%q (reply=%s)", tc.name, resp.ExitCode, resp.Reason, tc.wantExit, tc.wantReas, reply)
		}
	}

	// Following valid status still succeeds (C4).
	reply, _ := HandlePipeRequest(context.Background(), m, h, "en", []byte(`{"schemaVersion":1,"requestId":"ok1","command":"status"}`))
	var okResp Response
	if err := json.Unmarshal(bytes.TrimSpace(reply), &okResp); err != nil || okResp.ExitCode != ExitDone || okResp.RequestID != "ok1" {
		t.Fatalf("status after refusals failed: %s (%v)", reply, err)
	}
}

func TestPipeRoundTripStatusShowOpenQuit(t *testing.T) {
	m, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	pipeName, mutexName := uniqueTestPipeAndMutex()

	// Not running: status -> 11, quit -> 0.
	{
		var c capture
		env := c.env()
		env.PipeName = pipeName
		env.MutexName = mutexName
		if handled, code := Front([]string{"status"}, env); !handled || code != ExitNotFound {
			t.Fatalf("status not running: handled=%v code=%d stderr=%q", handled, code, c.err.String())
		}
		var q capture
		qenv := q.env()
		qenv.PipeName = pipeName
		qenv.MutexName = mutexName
		if handled, code := Front([]string{"quit"}, qenv); !handled || code != ExitDone {
			t.Fatalf("quit not running: handled=%v code=%d stderr=%q", handled, code, q.err.String())
		}
	}

	lock, acquired, err := AcquireInstanceLock(mutexName)
	if err != nil || !acquired {
		t.Fatalf("AcquireInstanceLock failed: acquired=%v err=%v", acquired, err)
	}
	defer lock.Release()

	quitCalled := make(chan struct{})
	h := &stubInstanceHandler{
		windowOpen: true,
		activeRuns: 2,
		focused:    true,
		onQuit:     func() { close(quitCalled) },
	}
	ps, err := StartPipeServer(pipeName, func(line []byte) ([]byte, func()) {
		return HandlePipeRequest(context.Background(), m, h, "en", line)
	}, "en")
	if err != nil {
		t.Fatalf("StartPipeServer: %v", err)
	}
	defer ps.Close()

	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.epub")
	f2 := filepath.Join(dir, "b.pdf")
	if err := os.WriteFile(f1, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}

	// status --json --result
	resPath := filepath.Join(dir, "status.json")
	var sc capture
	senv := sc.env()
	senv.PipeName = pipeName
	senv.MutexName = mutexName
	if handled, code := Front([]string{"status", "--json", "--result", resPath}, senv); !handled || code != ExitDone {
		t.Fatalf("status running: handled=%v code=%d err=%q", handled, code, sc.err.String())
	}
	var st StatusData
	rb, err := os.ReadFile(resPath)
	if err != nil || json.Unmarshal(rb, &st) != nil || !st.WindowOpen || st.ActiveRuns != 2 || st.AcceptsRestrictedCommands {
		t.Fatalf("status result=%q err=%v parsed=%+v", rb, err, st)
	}

	// show via LauncherHooks
	var sh capture
	shenv := sh.env()
	shenv.PipeName = pipeName
	shenv.MutexName = mutexName
	shenv.Launcher = &LauncherHooks{PipeName: pipeName, MutexName: mutexName}
	if handled, code := Front([]string{"show", "--json"}, shenv); !handled || code != ExitDone {
		t.Fatalf("show running: handled=%v code=%d err=%q", handled, code, sh.err.String())
	}

	// open --in f1 --in f2 (and missing file -> 11)
	var om capture
	omenv := om.env()
	omenv.PipeName = pipeName
	omenv.MutexName = mutexName
	omenv.Launcher = &LauncherHooks{PipeName: pipeName, MutexName: mutexName}
	if handled, code := Front([]string{"open", "--in", filepath.Join(dir, "missing.epub")}, omenv); !handled || code != ExitNotFound {
		t.Fatalf("open missing: handled=%v code=%d", handled, code)
	}
	var op capture
	openv := op.env()
	openv.PipeName = pipeName
	openv.MutexName = mutexName
	openv.Launcher = &LauncherHooks{PipeName: pipeName, MutexName: mutexName}
	if handled, code := Front([]string{"open", "--in", f1, "--in", f2, "--json"}, openv); !handled || code != ExitDone {
		t.Fatalf("open valid: handled=%v code=%d err=%q", handled, code, op.err.String())
	}
	if len(h.opened) != 1 || len(h.opened[0]) != 2 {
		t.Fatalf("opened paths = %v, want 2 paths", h.opened)
	}

	// quit while busy -> 11, quit --cancel-runs -> 0
	var qb capture
	qbenv := qb.env()
	qbenv.PipeName = pipeName
	qbenv.MutexName = mutexName
	if handled, code := Front([]string{"quit"}, qbenv); !handled || code != ExitNotFound {
		t.Fatalf("quit busy: handled=%v code=%d err=%q", handled, code, qb.err.String())
	}
	var qc capture
	qcenv := qc.env()
	qcenv.PipeName = pipeName
	qcenv.MutexName = mutexName
	if handled, code := Front([]string{"quit", "--cancel-runs", "--json"}, qcenv); !handled || code != ExitDone {
		t.Fatalf("quit cancel-runs: handled=%v code=%d err=%q", handled, code, qc.err.String())
	}
	select {
	case <-quitCalled:
	case <-time.After(time.Second):
		t.Fatal("quit did not invoke afterSend hook")
	}
}
