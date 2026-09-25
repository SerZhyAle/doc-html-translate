package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"mime"
	"net"
	"net/http"
	"strings"
)

// The GUI is a web server on a loopback port, and every page the user visits in any browser
// can send requests to it. Loopback alone therefore proves nothing about the caller. Three
// checks stand in front of the API, each covering a hole the others leave:
//
//   - Host must name this server. A DNS-rebinding page reaches the port under its own host
//     name, and without this check it could read the page, and the token inside it.
//   - Origin, when the browser sends one, must be this server. A cross-site fetch or form
//     post carries the attacker's origin.
//   - Every /api call carries a per-launch token that exists only inside the served page. A
//     cross-site page cannot read it, and a custom header also forces a CORS preflight that
//     this server never approves.
//
// The route wrappers then pin each endpoint to its method and, for a JSON body, to the JSON
// content type, so a state change never rides on a GET or a "simple" text/plain POST.

// tokenHeader carries the per-launch secret on every API call.
const tokenHeader = "X-DHT-Token"

// tokenPlaceholder is replaced in the served page with the live token.
const tokenPlaceholder = "__DHT_TOKEN__"

func newToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand never fails on a supported OS; an unguessable token is not optional.
		panic("crypto/rand: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// apiGuard wraps the whole mux. addr is the listener's "127.0.0.1:<port>".
type apiGuard struct {
	next    http.Handler
	token   string
	hosts   map[string]bool
	origins map[string]bool
}

func newAPIGuard(next http.Handler, addr, token string) *apiGuard {
	_, port, _ := net.SplitHostPort(addr)
	g := &apiGuard{next: next, token: token, hosts: map[string]bool{}, origins: map[string]bool{}}
	for _, h := range []string{"127.0.0.1:" + port, "localhost:" + port} {
		g.hosts[h] = true
		g.origins["http://"+h] = true
	}
	return g
}

func (g *apiGuard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !g.hosts[strings.ToLower(r.Host)] {
		http.Error(w, "forbidden host", http.StatusForbidden)
		return
	}
	if o := r.Header.Get("Origin"); o != "" && !g.origins[strings.ToLower(o)] {
		http.Error(w, "forbidden origin", http.StatusForbidden)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		// Sec-Fetch-Site is absent from non-browser callers; when a browser sends it, anything
		// but same-origin is another site talking to us.
		if s := r.Header.Get("Sec-Fetch-Site"); s != "" && s != "same-origin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if subtle.ConstantTimeCompare([]byte(r.Header.Get(tokenHeader)), []byte(g.token)) != 1 {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	} else if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	g.next.ServeHTTP(w, r)
}

// getOnly admits a read-only endpoint.
func getOnly(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h(w, r)
	}
}

// jsonPost admits a state change whose body is JSON.
func jsonPost(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !isJSON(r) {
			http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
			return
		}
		h(w, r)
	}
}

// postAction admits a state change that carries no body (or, for a drop, raw file bytes).
func postAction(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h(w, r)
	}
}

// getOrPostJSON admits an endpoint that reads on GET and writes a JSON body on POST.
func getOrPostJSON(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h(w, r)
		case http.MethodPost:
			jsonPost(h)(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func isJSON(r *http.Request) bool {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && mt == "application/json"
}
