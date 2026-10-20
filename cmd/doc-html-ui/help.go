package main

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"sync/atomic"
	"time"
)

// The in-app help (CONTEXT-HELP, ticket 118) is a set of generated pages inside the executable:
// help/help-map.json is the one map the resolver (help.js) and the gate read, help/<locale>/<id>.html the
// pages, help/*.css and help/page.js their shared parts. The window opens them from this server, so every
// channel (zip, setup, MSIX, the standalone exe) carries the same pages and nothing sits beside the exe.
// tools/helpgen writes the directory; nobody edits it by hand.

//go:embed help.js
var uiHelpJS string

//go:embed all:help
var helpFiles embed.FS

// helpGrace keeps the server up after the last help page was served. The server exits once the converter
// window is gone; without this, a help window left open when the converter closes would lose its links to
// the other help pages. The help pages still need no server to be read once loaded.
const helpGrace = 10 * time.Minute

var lastHelpServed atomic.Int64

// helpRecentlyServed reports whether a help page was served within the grace.
func helpRecentlyServed(now time.Time) bool {
	last := lastHelpServed.Load()
	return last != 0 && now.Unix()-last <= int64(helpGrace/time.Second)
}

// helpCSP is the page's own promise (CONTEXT-HELP rule 12) enforced by the browser: the page loads its
// own styles, images and script and nothing else, and submits nothing. A link the user follows is a
// navigation, not a request the page makes, so it is not limited.
const helpCSP = "default-src 'none'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'; " +
	"base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

var helpTypes = map[string]string{
	".html": "text/html; charset=utf-8",
	".css":  "text/css; charset=utf-8",
	".js":   "text/javascript; charset=utf-8",
	".json": "application/json; charset=utf-8",
	".png":  "image/png",
	".svg":  "image/svg+xml",
	".webp": "image/webp",
}

// handleHelpJS serves the resolver script the window loads.
func handleHelpJS(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	_, _ = io.WriteString(w, uiHelpJS)
}

// noListing refuses a directory that has no index.html, so the set is addressed page by page and the
// server never answers with a file listing.
type noListing struct{ fs.FS }

func (n noListing) Open(name string) (fs.File, error) {
	f, err := n.FS.Open(name)
	if err != nil {
		return nil, err
	}
	if st, err := f.Stat(); err == nil && st.IsDir() {
		if _, err := fs.Stat(n.FS, path.Join(name, "index.html")); err != nil {
			_ = f.Close()
			return nil, fs.ErrNotExist
		}
	}
	return f, nil
}

// newHelpHandler serves the embedded help set under /help/. It is read-only, needs no token (the pages
// hold nothing of the session), and sends the headers that keep a help page what rule 12 says it is.
func newHelpHandler(files fs.FS) http.Handler {
	sub, err := fs.Sub(files, "help")
	if err != nil {
		panic("help: " + err.Error())
	}
	server := http.StripPrefix("/help/", http.FileServerFS(noListing{sub}))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		lastHelpServed.Store(time.Now().Unix())
		h := w.Header()
		if ct, ok := helpTypes[strings.ToLower(path.Ext(r.URL.Path))]; ok {
			h.Set("Content-Type", ct)
		}
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Content-Security-Policy", helpCSP)
		h.Set("Cache-Control", "no-cache")
		server.ServeHTTP(w, r)
	})
}
