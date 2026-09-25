package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"doc-html-translate/internal/config"
)

// fakeCLIEnv switches the test binary into a stand-in converter, so the run tests drive the
// real handler, pipes and process control against a child whose behaviour they choose.
const fakeCLIEnv = "DHT_FAKE_CLI"

const longLineLen = 1 << 20

func TestMain(m *testing.M) {
	switch os.Getenv(fakeCLIEnv) {
	case "":
		os.Exit(m.Run())
	case "interleave":
		// Both streams at once, unsynchronized, plus one line far past any scanner buffer.
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			for i := 0; i < 2000; i++ {
				fmt.Fprintf(os.Stdout, "out %d\n", i)
			}
			fmt.Fprintf(os.Stdout, "%s\n", strings.Repeat("L", longLineLen))
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < 2000; i++ {
				fmt.Fprintf(os.Stderr, "err %d\n", i)
			}
		}()
		wg.Wait()
		os.Exit(0)
	case "sleep":
		gc := exec.Command(os.Args[0])
		gc.Env = append(os.Environ(), fakeCLIEnv+"=grandchild")
		if err := gc.Start(); err != nil {
			fmt.Println("grandchild:", err)
			os.Exit(1)
		}
		fmt.Printf("pid %d\ngpid %d\n", os.Getpid(), gc.Process.Pid)
		time.Sleep(60 * time.Second)
		os.Exit(0)
	case "linger":
		// Exits at once, leaving behind a process that still holds its stdout.
		gc := exec.Command(os.Args[0])
		gc.Env = append(os.Environ(), fakeCLIEnv+"=grandchild")
		gc.Stdout = os.Stdout
		if err := gc.Start(); err != nil {
			os.Exit(1)
		}
		fmt.Printf("pid %d\ngpid %d\n", os.Getpid(), gc.Process.Pid)
		os.Exit(0)
	case "grandchild":
		time.Sleep(60 * time.Second)
		os.Exit(0)
	}
	os.Exit(2)
}

// guardedServer runs the real mux behind the real guard, the way main wires it.
func guardedServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	srv := httptest.NewUnstartedServer(nil)
	token := newToken()
	srv.Config.Handler = newAPIGuard(newMux(), srv.Listener.Addr().String(), token)
	srv.Start()
	t.Cleanup(srv.Close)
	return srv, token
}

type reqOpt func(*http.Request)

func withToken(tok string) reqOpt { return func(r *http.Request) { r.Header.Set(tokenHeader, tok) } }
func withJSON() reqOpt {
	return func(r *http.Request) { r.Header.Set("Content-Type", "application/json") }
}
func withHeader(k, v string) reqOpt { return func(r *http.Request) { r.Header.Set(k, v) } }

func call(t *testing.T, srv *httptest.Server, method, path, body string, opts ...reqOpt) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range opts {
		o(req)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestGuardRefusesForeignCallers(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	srv, tok := guardedServer(t)
	body := `{"engine":"google"}`

	cases := []struct {
		name   string
		method string
		path   string
		opts   []reqOpt
		want   int
	}{
		{"no token", http.MethodPost, "/api/settings", []reqOpt{withJSON()}, http.StatusForbidden},
		{"wrong token", http.MethodPost, "/api/settings", []reqOpt{withJSON(), withToken("nope")}, http.StatusForbidden},
		{"foreign origin", http.MethodPost, "/api/settings", []reqOpt{withJSON(), withToken(tok), withHeader("Origin", "https://evil.example")}, http.StatusForbidden},
		{"cross-site fetch", http.MethodPost, "/api/settings", []reqOpt{withJSON(), withToken(tok), withHeader("Sec-Fetch-Site", "cross-site")}, http.StatusForbidden},
		{"text/plain CSRF body", http.MethodPost, "/api/settings", []reqOpt{withToken(tok), withHeader("Content-Type", "text/plain")}, http.StatusUnsupportedMediaType},
		{"GET on a state change", http.MethodGet, "/api/register", []reqOpt{withToken(tok)}, http.StatusMethodNotAllowed},
		{"GET on run", http.MethodGet, "/api/run", []reqOpt{withToken(tok)}, http.StatusMethodNotAllowed},
		{"POST to the page", http.MethodPost, "/", nil, http.StatusMethodNotAllowed},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp := call(t, srv, c.method, c.path, body, c.opts...)
			if resp.StatusCode != c.want {
				t.Fatalf("status = %d, want %d", resp.StatusCode, c.want)
			}
			if _, err := os.Stat(settingsPath()); err == nil {
				t.Fatal("a refused request still wrote the settings file")
			}
		})
	}

	resp := call(t, srv, http.MethodPost, "/api/settings", body, withJSON(), withToken(tok),
		withHeader("Origin", srv.URL), withHeader("Sec-Fetch-Site", "same-origin"))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("the page's own request = %d, want 200", resp.StatusCode)
	}
	if _, err := os.Stat(settingsPath()); err != nil {
		t.Fatalf("the page's own request did not save: %v", err)
	}
}

// A DNS-rebinding page reaches the port under its own host name. It must not get the page,
// because the page holds the token.
func TestGuardRefusesForeignHost(t *testing.T) {
	srv, tok := guardedServer(t)
	for _, path := range []string{"/", "/api/version"} {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+path, nil)
		req.Host = "rebind.example:" + srv.URL[strings.LastIndex(srv.URL, ":")+1:]
		req.Header.Set(tokenHeader, tok)
		resp, err := srv.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s with a foreign Host = %d, want 403", path, resp.StatusCode)
		}
	}
}

func TestPageCarriesTheLaunchToken(t *testing.T) {
	srv := httptest.NewUnstartedServer(nil)
	srv.Config.Handler = newAPIGuard(newMux(), srv.Listener.Addr().String(), uiToken)
	srv.Start()
	defer srv.Close()

	resp := call(t, srv, http.MethodGet, "/", "")
	page, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(page), uiToken) || strings.Contains(string(page), tokenPlaceholder) {
		t.Fatal("the served page does not carry this launch's token")
	}
	if resp.Header.Get("Cache-Control") != "no-store" {
		t.Error("the page may be cached with a token that dies with this launch")
	}
	api := call(t, srv, http.MethodGet, "/api/version", "", withToken(uiToken))
	if api.StatusCode != http.StatusOK {
		t.Fatalf("the page's token was refused: %d", api.StatusCode)
	}
}

// ── run: relay, cancel, one run per output ─────────────────

func fakeRun(t *testing.T, mode string) (*httptest.Server, string, string) {
	t.Helper()
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv(fakeCLIEnv, mode)
	prev := resolveCLI
	resolveCLI = func() string { return os.Args[0] }
	t.Cleanup(func() { resolveCLI = prev })

	dir := t.TempDir()
	input := filepath.Join(dir, "book.epub")
	if err := os.WriteFile(input, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	srv, tok := guardedServer(t)
	return srv, tok, fmt.Sprintf(`{"input":%q,"output":"","srcLang":"en","dstLang":"ru","noOpen":true}`, input)
}

func startRun(ctx context.Context, t *testing.T, srv *httptest.Server, tok, body string) *http.Response {
	t.Helper()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+"/api/run", strings.NewReader(body))
	req.Header.Set(tokenHeader, tok)
	req.Header.Set("Content-Type", "application/json")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestRunRelaysInterleavedStreamsAndALongLine(t *testing.T) {
	srv, tok, body := fakeRun(t, "interleave")
	resp := startRun(context.Background(), t, srv, tok, body)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	lines := map[string]bool{}
	for _, l := range strings.Split(string(data), "\n") {
		lines[l] = true
	}
	for i := 0; i < 2000; i++ {
		if !lines[fmt.Sprintf("out %d", i)] {
			t.Fatalf("stdout line %d missing or torn", i)
		}
		if !lines[fmt.Sprintf("[err] err %d", i)] {
			t.Fatalf("stderr line %d missing or torn", i)
		}
	}
	if !lines[strings.Repeat("L", longLineLen)] {
		t.Fatal("the 1 MiB line did not arrive whole")
	}
	if !strings.HasSuffix(string(data), "\nDone.\n") {
		t.Fatalf("the run did not finish: tail %q", tail(string(data)))
	}
}

func tail(s string) string {
	if len(s) > 200 {
		return s[len(s)-200:]
	}
	return s
}

// readPIDs reads the fake converter's "pid N" / "gpid N" lines from the run stream.
func readPIDs(t *testing.T, br *bufio.Reader) (pid, gpid int) {
	t.Helper()
	for pid == 0 || gpid == 0 {
		line, err := br.ReadString('\n')
		if err != nil {
			t.Fatalf("stream ended before the child reported its pids: %v", err)
		}
		f := strings.Fields(line)
		if len(f) == 2 {
			n, _ := strconv.Atoi(f[1])
			switch f[0] {
			case "pid":
				pid = n
			case "gpid":
				gpid = n
			}
		}
	}
	return pid, gpid
}

func waitGone(t *testing.T, pids ...int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for _, pid := range pids {
		for !processGone(pid) {
			if time.Now().After(deadline) {
				t.Fatalf("process %d is still running", pid)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func TestCancelStopsTheWholeTree(t *testing.T) {
	srv, tok, body := fakeRun(t, "sleep")
	resp := startRun(context.Background(), t, srv, tok, body)
	defer resp.Body.Close()
	br := bufio.NewReader(resp.Body)
	pid, gpid := readPIDs(t, br)

	c := call(t, srv, http.MethodPost, "/api/cancel", body, withToken(tok), withJSON())
	if c.StatusCode != http.StatusOK {
		t.Fatalf("cancel = %d", c.StatusCode)
	}

	done := make(chan string, 1)
	go func() { rest, _ := io.ReadAll(br); done <- string(rest) }()
	select {
	case rest := <-done:
		if !strings.Contains(rest, "Cancelled.") {
			t.Errorf("the log does not say the run was cancelled: %q", tail(rest))
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the run did not end after Cancel")
	}
	waitGone(t, pid, gpid)
}

func TestDroppedRequestStopsTheChild(t *testing.T) {
	srv, tok, body := fakeRun(t, "sleep")
	ctx, cancel := context.WithCancel(context.Background())
	resp := startRun(ctx, t, srv, tok, body)
	defer resp.Body.Close()
	pid, gpid := readPIDs(t, bufio.NewReader(resp.Body))
	cancel() // the page went away
	waitGone(t, pid, gpid)
}

// A process the converter left behind with its stdout must not hold the run open.
func TestRunEndsWhenALeftoverProcessHoldsThePipe(t *testing.T) {
	srv, tok, body := fakeRun(t, "linger")
	resp := startRun(context.Background(), t, srv, tok, body)
	defer resp.Body.Close()
	br := bufio.NewReader(resp.Body)
	_, gpid := readPIDs(t, br)
	t.Cleanup(func() {
		if p, err := os.FindProcess(gpid); err == nil {
			_ = p.Kill()
		}
	})

	done := make(chan string, 1)
	go func() { rest, _ := io.ReadAll(br); done <- string(rest) }()
	select {
	case rest := <-done:
		if !strings.Contains(rest, "Done.") {
			t.Errorf("the run did not finish cleanly: %q", tail(rest))
		}
	case <-time.After(pipeGrace + 10*time.Second):
		t.Fatal("a leftover process holding the pipe kept the run open")
	}
}

func TestSecondRunOnTheSameOutputIsRefused(t *testing.T) {
	srv, tok, body := fakeRun(t, "sleep")
	resp := startRun(context.Background(), t, srv, tok, body)
	defer resp.Body.Close()
	br := bufio.NewReader(resp.Body)
	pid, gpid := readPIDs(t, br)

	second := startRun(context.Background(), t, srv, tok, body)
	second.Body.Close()
	if second.StatusCode != http.StatusConflict {
		t.Errorf("second run on one output = %d, want 409", second.StatusCode)
	}

	call(t, srv, http.MethodPost, "/api/cancel", body, withToken(tok), withJSON())
	_, _ = io.ReadAll(br)
	waitGone(t, pid, gpid)
}

// ── argument hygiene: garbage in a form box never breaks the CLI parse ──

func TestAssembledArgsSurviveGarbageFields(t *testing.T) {
	junk := []string{"", " ", "abc", "-3", "1e3", "12x", "--force", "Inf"}
	for _, engine := range []string{"chrome", "google", "ollama", "none"} {
		for _, v := range junk {
			req := runRequest{
				Input:          `C:\books\story.epub`,
				NoTranslate:    engine == "none",
				Google:         engine == "google",
				Ollama:         engine == "ollama",
				OllamaModel:    v,
				OllamaParallel: v,
				OllamaCtx:      v,
				SplitSize:      v,
				TOCDepth:       v,
				MaxCost:        v,
				SrcLang:        v,
				DstLang:        v,
				OCR:            true,
				OCRLang:        v,
				UILang:         v,
			}
			args := assembleArgs(req)
			cfg, err := config.ParseArgs(args)
			if err != nil {
				t.Fatalf("engine %s, field value %q: the CLI rejects the GUI's command line: %v (args %v)", engine, v, err, args)
			}
			if cfg.InputFile != req.Input {
				t.Fatalf("engine %s, field value %q: input became %q", engine, v, cfg.InputFile)
			}
			if cfg.SplitSize != 0 {
				t.Errorf("engine %s, split %q: got %d, want the GUI default 0", engine, v, cfg.SplitSize)
			}
			if engine != "ollama" && (cfg.OllamaParallel != 1 || cfg.OllamaNumCtx != 8192 || cfg.OllamaModel != "gemma3:12b") {
				t.Errorf("engine %s: Ollama fields leaked into the command line: %v", engine, args)
			}
		}
	}
}

func TestAssembleArgsSendsEngineFieldsOnlyForThatEngine(t *testing.T) {
	base := runRequest{Input: `C:\b.epub`, OllamaModel: "llama3", OllamaParallel: "2", OllamaCtx: "4096", MaxCost: "3"}

	ollama := base
	ollama.Ollama = true
	cfg, err := config.ParseArgs(assembleArgs(ollama))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OllamaModel != "llama3" || cfg.OllamaParallel != 2 || cfg.OllamaNumCtx != 4096 {
		t.Errorf("Ollama run lost its settings: %+v", cfg)
	}
	if cfg.MaxCost != 0 {
		t.Errorf("max cost reached a free engine: %v", cfg.MaxCost)
	}

	google := base
	google.Google = true
	cfg, err = config.ParseArgs(assembleArgs(google))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxCost != 3 {
		t.Errorf("Google run lost its cost limit: %v", cfg.MaxCost)
	}
}

// ── durable settings ────────────────────────────────────────

func TestCorruptSettingsAreSetAsideAndReported(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	p := settingsPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(`{"engine":"goo`), 0o600); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	handleSettings(rec, httptest.NewRequest(http.MethodGet, "/api/settings", nil))
	moved := rec.Header().Get("X-Settings-Corrupt")
	if moved == "" {
		t.Fatal("a corrupt settings file was not reported")
	}
	if got, err := os.ReadFile(moved); err != nil || string(got) != `{"engine":"goo` {
		t.Fatalf("the corrupt file was not kept for recovery: %q, %v", got, err)
	}
	if strings.TrimSpace(rec.Body.String()) != "{}" {
		t.Fatalf("GET = %q, want {}", rec.Body.String())
	}
}

func TestSettingsSaveLeavesNoTemporaryFiles(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	for i := 0; i < 5; i++ {
		if err := writeSettings([]byte(fmt.Sprintf(`{"n":%d}`, i))); err != nil {
			t.Fatal(err)
		}
	}
	entries, _ := os.ReadDir(filepath.Dir(settingsPath()))
	for _, e := range entries {
		if e.Name() != filepath.Base(settingsPath()) {
			t.Errorf("left behind: %s", e.Name())
		}
	}
}

func TestParamsHistoryKeepsConcurrentUpdates(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := updateParamsHistory(func(m map[string]string) { m[strconv.Itoa(i)] = "x" }); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if got := len(loadParamsHistory()); got != 20 {
		t.Fatalf("history holds %d entries, want 20", got)
	}
}

// ── liveness ────────────────────────────────────────────────

func TestWatchdogOutlivesAQuietButOpenPage(t *testing.T) {
	now := time.Now()
	lastPing.Store(now.Add(-10 * time.Minute).Unix())
	aliveStreams.Add(1)
	if shouldExit(now) {
		t.Error("a page with an open alive stream was treated as gone")
	}
	aliveStreams.Add(-1)

	lastPing.Store(now.Add(-2 * time.Minute).Unix())
	if !shouldExit(now) {
		t.Error("no stream, no work and no ping for longer than the grace should end the server")
	}
	lastPing.Store(now.Add(-70 * time.Second).Unix())
	if shouldExit(now) {
		t.Error("a ping a minute old is a throttled page, not a closed one")
	}
}

func TestAliveStreamIsCountedWhileOpen(t *testing.T) {
	srv, tok := guardedServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/alive", nil)
	req.Header.Set(tokenHeader, tok)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if aliveStreams.Load() != 1 {
		t.Errorf("open stream count = %d, want 1", aliveStreams.Load())
	}
	cancel()
	resp.Body.Close()
	deadline := time.Now().Add(5 * time.Second)
	for aliveStreams.Load() != 0 {
		if time.Now().After(deadline) {
			t.Fatal("a closed alive stream is still counted")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
