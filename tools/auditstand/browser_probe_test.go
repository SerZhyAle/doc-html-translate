package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func TestBrowserProbeTargetSelectionAndCounters(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()
	mux.HandleFunc("/json/list", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":"owned","type":"page","webSocketDebuggerUrl":"` + strings.Replace(server.URL, "http://", "ws://", 1) + `/ws"}]`))
	})
	mux.Handle("/ws", websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		for {
			var request struct {
				ID     int    `json:"id"`
				Method string `json:"method"`
			}
			if websocket.JSON.Receive(ws, &request) != nil {
				return
			}
			// Async protocol events must not be mistaken for command responses.
			_ = websocket.JSON.Send(ws, map[string]any{"method": "Runtime.consoleAPICalled"})
			var result any
			switch request.Method {
			case "Memory.getDOMCounters":
				result = map[string]int{"documents": 1, "nodes": 42, "jsEventListeners": 3}
			case "Runtime.getHeapUsage":
				result = map[string]int{"usedSize": 1024, "totalSize": 2048}
			default:
				result = map[string]any{"result": map[string]any{"value": true}}
			}
			if websocket.JSON.Send(ws, map[string]any{"id": request.ID, "result": result}) != nil {
				return
			}
		}
	}))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := connectBrowserProbe(ctx, server.URL, "foreign"); err == nil {
		t.Fatal("unselected target accepted")
	}
	p, err := connectBrowserProbe(ctx, server.URL, "owned")
	if err != nil {
		t.Fatal(err)
	}
	defer p.conn.Close()
	fields, err := p.sampleDOM(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if fields["dom_nodes"] != 42 || fields["js_heap_bytes"] != float64(1024) {
		t.Fatalf("wrong counters: %v", fields)
	}
}

func TestBrowserProbeRejectsNonLoopbackEndpoints(t *testing.T) {
	for _, raw := range []string{"http://example.com", "http://127.0.0.1@example.com", "https://127.0.0.1", "http://192.168.0.1", "file:///tmp/browser"} {
		if loopbackURL(raw, "http") == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}
