package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// The help set is embedded, served read-only under /help/, and carries the headers that keep a page what
// CONTEXT-HELP rule 12 says it is: nothing is requested from another origin.
func TestHelpRouteServesTheEmbeddedSet(t *testing.T) {
	srv, _ := guardedServer(t)

	get := func(p string) *http.Response {
		resp, err := http.Get(srv.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}
	for _, c := range []struct {
		path, ctype string
	}{
		{"/help/help-map.json", "application/json"},
		{"/help/en/converter.html", "text/html"},
		{"/help/ru/index.html", "text/html"},
		{"/help/help.css", "text/css"},
		{"/help/page.js", "text/javascript"},
		{"/help/en/converter-ready-dark.png", "image/png"},
		{"/help.js", "text/javascript"},
	} {
		resp := get(c.path)
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 || len(body) == 0 {
			t.Errorf("GET %s = %d (%d bytes)", c.path, resp.StatusCode, len(body))
			continue
		}
		if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, c.ctype) {
			t.Errorf("GET %s Content-Type = %q, want %s", c.path, ct, c.ctype)
		}
		if strings.HasPrefix(c.path, "/help/") {
			if resp.Header.Get("X-Content-Type-Options") != "nosniff" || resp.Header.Get("Referrer-Policy") != "no-referrer" {
				t.Errorf("GET %s lacks nosniff or no-referrer", c.path)
			}
			csp := resp.Header.Get("Content-Security-Policy")
			for _, want := range []string{"default-src 'none'", "script-src 'self'", "img-src 'self'", "frame-ancestors 'none'"} {
				if !strings.Contains(csp, want) {
					t.Errorf("GET %s: the content policy %q lacks %q", c.path, csp, want)
				}
			}
			if strings.Contains(csp, "http:") || strings.Contains(csp, "https:") || strings.Contains(csp, "*") {
				t.Errorf("GET %s: the content policy admits another origin: %q", c.path, csp)
			}
		}
	}
	// no listing, no way out of the set, no write
	for _, p := range []string{"/help/", "/help/en/", "/help/../ui.html", "/help/en/missing.html"} {
		resp := get(p)
		resp.Body.Close()
		if resp.StatusCode == 200 && p != "/help/en/" {
			t.Errorf("GET %s = 200, want a refusal", p)
		}
	}
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/help/en/converter.html", strings.NewReader("x"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode == 200 {
		t.Error("POST /help/ was accepted; the set is read-only")
	}
}

// A directory is answered only when it has an index.html of its own.
func TestHelpDirectoriesHaveNoListing(t *testing.T) {
	files := fstest.MapFS{
		"help/en/index.html": {Data: []byte("<p>en")},
		"help/en/page.html":  {Data: []byte("<p>page")},
		"help/secret/a.txt":  {Data: []byte("a")},
	}
	h := newHelpHandler(files)
	for p, want := range map[string]int{"/help/secret/": 404, "/help/secret": 404, "/help/en/page.html": 200, "/help/": 404} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
		if rec.Code != want {
			t.Errorf("GET %s = %d, want %d", p, rec.Code, want)
		}
	}
}

// The server outlives the window for a while after a help page was served, so a help window left open still
// follows its links; and only for that while.
func TestHelpKeepsTheServerAliveForAGrace(t *testing.T) {
	lastHelpServed.Store(0)
	lastPing.Store(time.Now().Add(-time.Hour).Unix())
	now := time.Now()
	if !shouldExit(now) {
		t.Fatal("with no window and no help the server must exit")
	}
	lastHelpServed.Store(now.Add(-time.Minute).Unix())
	if shouldExit(now) {
		t.Error("a help page served a minute ago must keep the server up")
	}
	lastHelpServed.Store(now.Add(-helpGrace - time.Minute).Unix())
	if !shouldExit(now) {
		t.Error("the help grace is over: the server must exit")
	}
	lastHelpServed.Store(0)
}

// CONTEXT-HELP rule 2: the help button sits at the trailing corner of the header and nothing else is added to
// that corner. Ticket 119 (SETTINGS-NAVIGATION 3.1) moved the theme and interface-language selects into the
// first page, each with its own caption, and added the Pages button that appears only when the window is too
// narrow for the rail; the header is now the Pages button, the three links and the help button.
func TestHelpButtonIsTheOnlyAdditionToTheHeader(t *testing.T) {
	start := strings.Index(uiHTML, `<div class="head"`)
	end := strings.Index(uiHTML, `<div class="hint fail" id="cliWarn"`)
	if start < 0 || end < start {
		t.Fatal("the header block of ui.html was not found")
	}
	header := uiHTML[start:end]
	controls := regexp.MustCompile(`<(button|select|a|input)\b[^>]*>`).FindAllString(header, -1)
	got := map[string]bool{}
	for _, c := range controls {
		key := c
		if m := regexp.MustCompile(`id="([^"]+)"`).FindStringSubmatch(c); m != nil {
			key = m[1]
		} else if m := regexp.MustCompile(`href="([^"]+)"`).FindStringSubmatch(c); m != nil {
			key = m[1]
		}
		got[key] = true
	}
	before := []string{"btnPages", "https://sza.od.ua", "https://serzhyale.github.io/doc-html-translate/", "mailto:sza@ukr.net"}
	want := map[string]bool{"btnHelp": true}
	for _, b := range before {
		want[b] = true
	}
	for k := range got {
		if !want[k] {
			t.Errorf("the header gained a control the contract did not ask for: %s", k)
		}
	}
	for k := range want {
		if !got[k] {
			t.Errorf("the header lost %s", k)
		}
	}
	// the button is the LAST control of the header row and is pushed to its trailing end
	if !strings.HasSuffix(strings.TrimSpace(header), "</svg></button>\n</div>") || !strings.Contains(header, `class="help-btn secondary head-help" id="btnHelp"`) {
		t.Error("the help button is not the trailing control of the header")
	}
	// the selects live on the first page now, once each, with a visible caption
	for _, id := range []string{"themeSwitch", "langSwitch"} {
		if n := strings.Count(uiHTML, `id="`+id+`"`); n != 1 {
			t.Errorf("%s appears %d times, want one editor", id, n)
		}
		if !strings.Contains(uiHTML, `<label for="`+id+`"`) {
			t.Errorf("%s has no visible caption", id)
		}
	}
}

// Every dialog the window can open names its surface, and F1 is handled by one resolver (help.js), registered on
// the page and never system-wide.
func TestHelpWiringOfThePage(t *testing.T) {
	for _, snippet := range []string{
		`<script src="/help.js"></script>`,
		`data-help-open="main"`,
		`data-help-open="dialog"`,
		`id="dlgHelp"`,
		`dlg.dataset.surface = opts.surface;`,
	} {
		if !strings.Contains(uiHTML, snippet) {
			t.Errorf("ui.html is missing %q", snippet)
		}
	}
	if strings.Contains(uiHTML, "F1") && strings.Contains(strings.ReplaceAll(uiHTML, "Help (F1)", ""), "'F1'") {
		t.Error("ui.html handles F1 itself; the key belongs to help.js, one resolver (CONTEXT-HELP rule 17)")
	}
}

// The resolver (help.js) is plain functions; node drives them in help_resolver_test.mjs.
func TestHelpResolver(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Fatal("node is needed for the help resolver test")
	}
	cmd := exec.Command("node", "--test", "help_resolver_test.mjs")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("help resolver test: %v\n%s", err, out)
	}
}
