package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

// Connect only to an explicitly selected browser page over loopback. Neither the host API
// token nor arbitrary pages in the owner's browser are collected into artifacts.
type browserProbe struct {
	conn *websocket.Conn
	mu   sync.Mutex
	id   int
}

func loopbackURL(raw, scheme string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	ip := net.ParseIP(u.Hostname())
	if u.Scheme != scheme || ip == nil || !ip.IsLoopback() || u.User != nil {
		return fmt.Errorf("browser endpoint must use %s on a loopback IP", scheme)
	}
	return nil
}

func connectBrowserProbe(ctx context.Context, endpoint, targetID string) (*browserProbe, error) {
	if err := loopbackURL(endpoint, "http"); err != nil {
		return nil, err
	}
	if targetID == "" {
		return nil, fmt.Errorf("an explicit browser target id is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"/json/list", nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("target discovery HTTP %d", resp.StatusCode)
	}
	var targets []struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		WS   string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(nil, resp.Body, 1<<20)).Decode(&targets); err != nil {
		return nil, err
	}
	for _, target := range targets {
		if target.ID != targetID || target.Type != "page" {
			continue
		}
		if err := loopbackURL(target.WS, "ws"); err != nil {
			return nil, err
		}
		cfg, err := websocket.NewConfig(target.WS, endpoint)
		if err != nil {
			return nil, err
		}
		conn, err := cfg.DialContext(ctx)
		if err != nil {
			return nil, err
		}
		conn.MaxPayloadBytes = 4 << 20
		return &browserProbe{conn: conn}, nil
	}
	return nil, fmt.Errorf("selected browser page is absent")
}

func (p *browserProbe) call(ctx context.Context, method string, params any, result any) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.id++
	deadline := time.Now().Add(5 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if err := p.conn.SetDeadline(deadline); err != nil {
		return err
	}
	if err := websocket.JSON.Send(p.conn, map[string]any{"id": p.id, "method": method, "params": params}); err != nil {
		return err
	}
	for {
		var reply struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := websocket.JSON.Receive(p.conn, &reply); err != nil {
			return err
		}
		if reply.ID != p.id {
			continue
		}
		if reply.Error != nil {
			return fmt.Errorf("CDP %s: %d %s", method, reply.Error.Code, reply.Error.Message)
		}
		if result == nil {
			return nil
		}
		return json.Unmarshal(reply.Result, result)
	}
}

func (p *browserProbe) evaluate(ctx context.Context, expression string, result any) error {
	var reply struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		Exception json.RawMessage `json:"exceptionDetails"`
	}
	if err := p.call(ctx, "Runtime.evaluate", map[string]any{"expression": expression, "returnByValue": true, "awaitPromise": true}, &reply); err != nil {
		return err
	}
	if len(reply.Exception) > 0 {
		return fmt.Errorf("browser expression failed")
	}
	return json.Unmarshal(reply.Result.Value, result)
}

func (p *browserProbe) sampleDOM(ctx context.Context) (map[string]any, error) {
	var dom struct {
		Documents int `json:"documents"`
		Nodes     int `json:"nodes"`
		Listeners int `json:"jsEventListeners"`
	}
	if err := p.call(ctx, "Memory.getDOMCounters", map[string]any{}, &dom); err != nil {
		return nil, err
	}
	var heap struct {
		Used  float64 `json:"usedSize"`
		Total float64 `json:"totalSize"`
	}
	if err := p.call(ctx, "Runtime.getHeapUsage", map[string]any{}, &heap); err != nil {
		return nil, err
	}
	return map[string]any{"dom_nodes": dom.Nodes, "documents": dom.Documents, "listeners": dom.Listeners, "js_heap_bytes": heap.Used, "js_heap_total_bytes": heap.Total}, nil
}

// P3 follows an actual Tab focus change through the next painted frame. SendInput remains
// platform-owned and opt-in; this method cannot inject into an unfocused foreign window.
func (p *browserProbe) keyReaction(ctx context.Context, hwnd uintptr) (float64, error) {
	var armed bool
	const arm = `(()=>{if(window.__auditFocusCleanup)window.__auditFocusCleanup();window.__auditFocusPainted=false;const f=()=>requestAnimationFrame(()=>{window.__auditFocusPainted=true});document.addEventListener('focusin',f,{once:true});window.__auditFocusCleanup=()=>{document.removeEventListener('focusin',f);delete window.__auditFocusPainted;delete window.__auditFocusCleanup};return true})()`
	if err := p.evaluate(ctx, arm, &armed); err != nil {
		return 0, err
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		var ok bool
		_ = p.evaluate(cleanupCtx, `(()=>{if(window.__auditFocusCleanup)window.__auditFocusCleanup();return true})()`, &ok)
	}()
	start := time.Now()
	if err := injectAuditTab(hwnd); err != nil {
		return 0, err
	}
	for {
		var painted bool
		if err := p.evaluate(ctx, `window.__auditFocusPainted===true`, &painted); err != nil {
			return 0, err
		}
		if painted {
			return float64(time.Since(start).Microseconds()) / 1000, nil
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(time.Millisecond):
		}
	}
}
