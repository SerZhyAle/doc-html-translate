package translator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

const testKey = "AIzaTESTKEY-0123456789abcdefghijklmnopqrs"

// fakeClock records the waits a client asks for instead of sleeping through them.
type fakeClock struct {
	mu    sync.Mutex
	now   time.Time
	waits []time.Duration
}

func (f *fakeClock) policy() retryPolicy {
	return retryPolicy{
		budget: maxRetryWait,
		sleep: func(_ context.Context, d time.Duration) error {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.waits = append(f.waits, d)
			f.now = f.now.Add(d)
			return nil
		},
		now: func() time.Time {
			f.mu.Lock()
			defer f.mu.Unlock()
			return f.now
		},
		jitter: func(d time.Duration) time.Duration { return d },
	}
}

func newTestClient(serverURL string) (*GoogleClient, *fakeClock) {
	clock := &fakeClock{now: time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)}
	return &GoogleClient{
		apiKey:     testKey,
		baseURL:    serverURL,
		httpClient: &http.Client{},
		retry:      clock.policy(),
	}, clock
}

// googleStub answers like v2: one translation per q, "tr:" + the text as sent. hook may answer
// instead (return true when it did).
func googleStub(t *testing.T, hook func(w http.ResponseWriter, r *http.Request, req translateRequest) bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req translateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		if hook != nil && hook(w, r, req) {
			return
		}
		writeTranslations(w, req.Q, "tr:")
	}))
	t.Cleanup(srv.Close)
	return srv
}

func writeTranslations(w http.ResponseWriter, qs []string, prefix string) {
	resp := translateResponse{}
	for _, q := range qs {
		resp.Data.Translations = append(resp.Data.Translations, struct {
			TranslatedText string `json:"translatedText"`
		}{TranslatedText: prefix + q})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func TestTranslateSuccess(t *testing.T) {
	srv := googleStub(t, func(_ http.ResponseWriter, r *http.Request, req translateRequest) bool {
		if req.Source != "en" || req.Target != "ru" || req.Format != "html" {
			t.Errorf("unexpected request: %+v", req)
		}
		return false
	})
	client, _ := newTestClient(srv.URL)
	results, err := client.Translate(context.Background(), []string{"hello", "world"}, "en", "ru")
	if err != nil {
		t.Fatalf("Translate failed: %v", err)
	}
	if len(results) != 2 || results[0] != "tr:hello" || results[1] != "tr:world" {
		t.Fatalf("results = %v", results)
	}
}

// Done criterion 1 and ADR-1: the key goes in a header, never the URL, and a transport error
// cannot carry it into the console or the run log.
func TestAPIKeyTravelsInHeaderOnly(t *testing.T) {
	srv := googleStub(t, func(_ http.ResponseWriter, r *http.Request, _ translateRequest) bool {
		if got := r.Header.Get("X-Goog-Api-Key"); got != testKey {
			t.Errorf("header key = %q", got)
		}
		if strings.Contains(r.URL.String(), testKey) || r.URL.Query().Get("key") != "" {
			t.Errorf("key in URL: %s", r.URL)
		}
		return false
	})
	client, _ := newTestClient(srv.URL)
	if _, err := client.Translate(context.Background(), []string{"x"}, "en", "de"); err != nil {
		t.Fatal(err)
	}
}

func TestNetworkErrorNeverShowsKey(t *testing.T) {
	// A listener that is closed at once: every request fails in the transport.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	client, _ := newTestClient("http://" + addr + "/language/translate/v2")
	_, err = client.Translate(context.Background(), []string{"x"}, "en", "de")
	if err == nil {
		t.Fatal("expected a network error")
	}
	if strings.Contains(err.Error(), testKey) || strings.Contains(err.Error(), "AIza") {
		t.Fatalf("key leaked: %v", err)
	}
	if strings.Contains(err.Error(), "/language/translate/v2") {
		t.Fatalf("request URL leaked into the error: %v", err)
	}
}

func TestErrorBodyEchoingKeyIsScrubbed(t *testing.T) {
	srv := googleStub(t, func(w http.ResponseWriter, _ *http.Request, _ translateRequest) bool {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintf(w, `{"error":{"code":400,"message":"bad key %s"}}`, testKey)
		return true
	})
	client, _ := newTestClient(srv.URL)
	_, err := client.Translate(context.Background(), []string{"x"}, "en", "de")
	if err == nil || strings.Contains(err.Error(), testKey) {
		t.Fatalf("err = %v", err)
	}
}

// Done criterion 2: "It's" must not come back as "It&#39;s", and markup characters in prose
// survive the round trip.
func TestEntityRoundTrip(t *testing.T) {
	srv := googleStub(t, func(w http.ResponseWriter, _ *http.Request, req translateRequest) bool {
		for _, q := range req.Q {
			if strings.ContainsAny(q, `<>'"`) {
				t.Errorf("text sent unescaped in html mode: %q", q)
			}
		}
		// The provider answers in HTML, entities and all - exactly what showed up as a literal
		// "&#39;" on the page before.
		writeTranslations(w, req.Q, "")
		return true
	})
	client, _ := newTestClient(srv.URL)
	in := []string{"It's", "a<b & c>d", `say "hi"`}
	got, err := client.Translate(context.Background(), in, "en", "de")
	if err != nil {
		t.Fatal(err)
	}
	for i := range in {
		if got[i] != in[i] {
			t.Errorf("round trip %q -> %q", in[i], got[i])
		}
	}
}

// Done criterion 3: a page of 300 short strings is split under v2's 128-per-request limit and
// translated completely, in order.
func TestThreeHundredSegments(t *testing.T) {
	var mu sync.Mutex
	var sizes []int
	srv := googleStub(t, func(w http.ResponseWriter, _ *http.Request, req translateRequest) bool {
		mu.Lock()
		sizes = append(sizes, len(req.Q))
		mu.Unlock()
		if len(req.Q) > maxSegmentsPerRequest {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"code":400,"message":"Too many text segments"}}`))
			return true
		}
		return false
	})
	client, _ := newTestClient(srv.URL)
	in := make([]string, 300)
	for i := range in {
		in[i] = fmt.Sprintf("s%d", i)
	}
	got, err := client.Translate(context.Background(), in, "en", "de")
	if err != nil {
		t.Fatal(err)
	}
	for i := range in {
		if got[i] != "tr:"+in[i] {
			t.Fatalf("got[%d] = %q", i, got[i])
		}
	}
	if len(sizes) != 3 {
		t.Fatalf("request sizes = %v", sizes)
	}
}

func TestBatchTextsBounds(t *testing.T) {
	batches := batchTexts([]string{"hello", "world", "foo", "bar"}, 10, 128)
	if len(batches) != 2 {
		t.Errorf("by size: %v", batches)
	}
	batches = batchTexts([]string{"short", "this is a very long text that exceeds limit", "tiny"}, 10, 128)
	if len(batches) != 3 {
		t.Errorf("oversized text alone: %v", batches)
	}
	// Characters, not bytes: ten Cyrillic letters are 20 bytes and still fit a 10-character batch.
	batches = batchTexts([]string{"абвгд", "еёжзи"}, 10, 128)
	if len(batches) != 1 {
		t.Errorf("by characters: %v", batches)
	}
	batches = batchTexts([]string{"a", "b", "c"}, 100, 2)
	if len(batches) != 2 {
		t.Errorf("by count: %v", batches)
	}
}

// A short reply is retried once, then fails with context - never a panic in the cache or a
// translation shifted onto the wrong text.
func TestShortReplyIsAnError(t *testing.T) {
	calls := 0
	srv := googleStub(t, func(w http.ResponseWriter, _ *http.Request, req translateRequest) bool {
		calls++
		writeTranslations(w, req.Q[:len(req.Q)-1], "tr:")
		return true
	})
	client, _ := newTestClient(srv.URL)
	cached := NewCachingClient(client)
	_, err := cached.Translate(context.Background(), []string{"a", "b", "c"}, "en", "de")
	if err == nil || !strings.Contains(err.Error(), "2 translations for 3 texts") {
		t.Fatalf("err = %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want one retry", calls)
	}
}

func TestShortReplyRecoveredOnRetry(t *testing.T) {
	calls := 0
	srv := googleStub(t, func(w http.ResponseWriter, _ *http.Request, req translateRequest) bool {
		calls++
		if calls == 1 {
			writeTranslations(w, req.Q[:1], "tr:")
			return true
		}
		return false
	})
	client, _ := newTestClient(srv.URL)
	got, err := client.Translate(context.Background(), []string{"a", "b"}, "en", "de")
	if err != nil || got[1] != "tr:b" {
		t.Fatalf("got %v, err %v", got, err)
	}
}

// The cache used to index an engine's reply by position without checking its length.
func TestCacheRejectsWrongLength(t *testing.T) {
	c := NewCachingClient(shortEngine{})
	if _, err := c.Translate(context.Background(), []string{"a", "b"}, "en", "de"); err == nil {
		t.Fatal("a short reply was accepted")
	}
}

type shortEngine struct{}

func (shortEngine) Translate(context.Context, []string, string, string) ([]string, error) {
	return []string{"only one"}, nil
}

func TestRetryHonoursRetryAfterSeconds(t *testing.T) {
	calls := 0
	srv := googleStub(t, func(w http.ResponseWriter, _ *http.Request, _ translateRequest) bool {
		calls++
		if calls <= 2 {
			w.Header().Set("Retry-After", "7")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"code":429,"message":"rate limited"}}`))
			return true
		}
		return false
	})
	client, clock := newTestClient(srv.URL)
	got, err := client.Translate(context.Background(), []string{"x"}, "en", "de")
	if err != nil || got[0] != "tr:x" {
		t.Fatalf("got %v, err %v", got, err)
	}
	if len(clock.waits) != 2 || clock.waits[0] != 7*time.Second || clock.waits[1] != 7*time.Second {
		t.Fatalf("waits = %v, want the server's 7s twice", clock.waits)
	}
}

func TestRetryAfterHTTPDate(t *testing.T) {
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	if d := parseRetryAfter(now.Add(12*time.Second).Format(http.TimeFormat), now); d != 12*time.Second {
		t.Fatalf("date form: %v", d)
	}
	for _, v := range []string{"", "soon", "-3", now.Add(-time.Minute).Format(http.TimeFormat)} {
		if d := parseRetryAfter(v, now); d != 0 {
			t.Errorf("%q -> %v, want no advice", v, d)
		}
	}
}

// A throttling 403 is waited out; a real refusal is not retried.
func TestForbiddenRateLimitIsRetried(t *testing.T) {
	calls := 0
	srv := googleStub(t, func(w http.ResponseWriter, _ *http.Request, _ translateRequest) bool {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":{"code":403,"message":"User Rate Limit Exceeded","errors":[{"reason":"userRateLimitExceeded"}]}}`))
			return true
		}
		return false
	})
	client, clock := newTestClient(srv.URL)
	if _, err := client.Translate(context.Background(), []string{"x"}, "en", "de"); err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(clock.waits) != 1 {
		t.Fatalf("calls = %d, waits = %v", calls, clock.waits)
	}
}

func TestForbiddenRefusalIsNotRetried(t *testing.T) {
	calls := 0
	srv := googleStub(t, func(w http.ResponseWriter, _ *http.Request, _ translateRequest) bool {
		calls++
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"code":403,"message":"API key not valid","errors":[{"reason":"forbidden"}]}}`))
		return true
	})
	client, _ := newTestClient(srv.URL)
	if _, err := client.Translate(context.Background(), []string{"x"}, "en", "de"); err == nil {
		t.Fatal("expected an error")
	}
	if calls != 1 {
		t.Fatalf("calls = %d", calls)
	}
}

// Throttling that outlasts the budget gives up with a reason, after waiting no more than the
// budget in total.
func TestRetryBudgetIsBounded(t *testing.T) {
	srv := googleStub(t, func(w http.ResponseWriter, _ *http.Request, _ translateRequest) bool {
		w.WriteHeader(http.StatusServiceUnavailable)
		return true
	})
	client, clock := newTestClient(srv.URL)
	_, err := client.Translate(context.Background(), []string{"x"}, "en", "de")
	if err == nil || !strings.Contains(err.Error(), "gave up") {
		t.Fatalf("err = %v", err)
	}
	var total time.Duration
	for i, w := range clock.waits {
		total += w
		if i > 0 && w < clock.waits[i-1] {
			t.Errorf("backoff shrank: %v", clock.waits)
		}
		if w > backoffCap {
			t.Errorf("wait %v above the cap", w)
		}
	}
	if total > maxRetryWait {
		t.Fatalf("waited %v in total", total)
	}
}

func TestRetryAfterBeyondBudgetGivesUpAtOnce(t *testing.T) {
	srv := googleStub(t, func(w http.ResponseWriter, _ *http.Request, _ translateRequest) bool {
		w.Header().Set("Retry-After", "3600")
		w.WriteHeader(http.StatusTooManyRequests)
		return true
	})
	client, clock := newTestClient(srv.URL)
	if _, err := client.Translate(context.Background(), []string{"x"}, "en", "de"); err == nil {
		t.Fatal("expected an error")
	}
	if len(clock.waits) != 0 {
		t.Fatalf("waited %v for advice past the budget", clock.waits)
	}
}

func TestTranslateCancelled(t *testing.T) {
	srv := googleStub(t, nil)
	client, _ := newTestClient(srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.Translate(ctx, []string{"x"}, "en", "de"); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestTranslateEmpty(t *testing.T) {
	client := NewGoogleClient("test-key")
	results, err := client.Translate(context.Background(), nil, "en", "ru")
	if err != nil || results != nil {
		t.Fatalf("results = %v, err = %v", results, err)
	}
}
