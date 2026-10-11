//go:build windows

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHostPingReadsCompletePageAndSpecificToken(t *testing.T) {
	want := strings.Repeat("b", 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			_, _ = w.Write([]byte(strings.Repeat("a", 64) + strings.Repeat(" ", 4096)))
			w.(http.Flusher).Flush()
			_, _ = w.Write([]byte(`<meta name="dht-token" content="` + want + `">`))
			return
		}
		if r.URL.Path != "/api/version" || r.Header.Get("X-DHT-Token") != want {
			http.Error(w, "wrong token", http.StatusForbidden)
			return
		}
		_, _ = w.Write([]byte(`{"version":"synthetic"}`))
	}))
	defer server.Close()
	m, err := hostPing(server.URL, 3, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if m.N != 3 {
		t.Fatalf("samples=%d", m.N)
	}
}

func TestMessageProbeCanBeRepeatedWithoutLeakingWindowClass(t *testing.T) {
	for i := 0; i < 2; i++ {
		var n int
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err := runMessageLatencyProbe(ctx, 20*time.Millisecond, func(float64) { n++ })
		cancel()
		if err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			t.Fatal("message probe measured nothing")
		}
	}
}
