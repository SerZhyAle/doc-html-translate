// Package helpcheck is the gate of the in-app help (CONTEXT-HELP rule 19, ticket 118). It enumerates the GUI
// launcher's surfaces from their source of truth - the regions and the dialogs of cmd/doc-html-ui/ui.html and the
// routes of its server - and holds them against the help map; it holds the pages against the locale set, the
// links against the files, the pages against rule 12 (a page makes no request) and the house text style, the two
// copies of the set against each other, and the set against the inputs it was generated from.
//
// It is developer tooling: the GUI does not import it. tools/helpcheck is its command, scripts/help-check.ps1 its
// gate wrapper, tests/help_test.go its hold inside the Go suite.
package helpcheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/net/html"

	"doc-html-translate/internal/i18n"
)

// Report is one run of the gate: what it counted, what is wrong, and what only deserves a mention.
type Report struct {
	Failures   []string
	Advisories []string
	Surfaces   int // rows of the map
	Pages      int // help pages (without the index) per locale
	Locales    int
	Files      int // files of the embedded set
	Links      int // links resolved
	Debt       int // locales accepted into the baseline
}

func (r *Report) fail(format string, a ...any) {
	r.Failures = append(r.Failures, fmt.Sprintf(format, a...))
}
func (r *Report) advise(format string, a ...any) {
	r.Advisories = append(r.Advisories, fmt.Sprintf(format, a...))
}

// Detail is the parenthesis of the verdict line.
func (r *Report) Detail() string {
	if len(r.Failures) > 0 {
		return fmt.Sprintf("%d finding(s)", len(r.Failures))
	}
	d := fmt.Sprintf("%d surfaces, %d pages x %d locales, %d links, %d locales in the baseline", r.Surfaces, r.Pages, r.Locales, r.Links, r.Debt)
	if len(r.Advisories) > 0 {
		d += fmt.Sprintf("; %d advisory", len(r.Advisories))
	}
	return d
}

// Code is the CHECK-VERDICT exit code: 1 on a failure, 3 on an advisory, 0 otherwise.
func (r *Report) Code() int {
	switch {
	case len(r.Failures) > 0:
		return 1
	case len(r.Advisories) > 0:
		return 3
	}
	return 0
}

// File names the gate reads, relative to the repository root.
const (
	MapPath      = "cmd/doc-html-ui/help/help-map.json"
	HelpDir      = "cmd/doc-html-ui/help"
	MirrorDir    = "help"
	UIPath       = "cmd/doc-html-ui/ui.html"
	ServerPath   = "cmd/doc-html-ui/main.go"
	BaselinePath = "configs/help-baseline.txt"
	CopyDir      = "tools/helpgen/copy"
	GenDir       = "tools/helpgen"
)

// Map is the help map (the shape of the catalog's reference/help-map.example.json, plus the locales the set holds).
type Map struct {
	SchemaVersion int       `json:"schemaVersion"`
	Product       string    `json:"product"`
	HelpRoot      string    `json:"helpRoot"`
	PageTemplate  string    `json:"pageTemplate"`
	Index         string    `json:"index"`
	SourceLocale  string    `json:"sourceLocale"`
	Locales       []string  `json:"locales"`
	Surfaces      []Surface `json:"surfaces"`
}

// Surface is one row of the map.
type Surface struct {
	ID     string    `json:"id"`
	Name   string    `json:"name"`
	Form   string    `json:"form"`
	Help   string    `json:"help"`
	Parent string    `json:"parent"`
	Reason string    `json:"reason"`
	Source string    `json:"source"`
	Pages  []PageRow `json:"pages"`
}

// PageRow is one region of a surface that has an entry of its own on the surface's page.
type PageRow struct {
	ID     string `json:"id"`
	Anchor string `json:"anchor"`
}

var forms = map[string]bool{"page": true, "layer": true, "both": true, "inherit": true, "none": true}

// Run judges the tree under root. The error is for a tree it cannot read at all (COULD NOT VERIFY).
func Run(root string) (*Report, error) {
	rd := func(rel string) ([]byte, error) { return os.ReadFile(filepath.Join(root, filepath.FromSlash(rel))) }
	mapRaw, err := rd(MapPath)
	if err != nil {
		return nil, err
	}
	ui, err := rd(UIPath)
	if err != nil {
		return nil, err
	}
	server, err := rd(ServerPath)
	if err != nil {
		return nil, err
	}
	var m Map
	if err := json.NewDecoder(bytes.NewReader(mapRaw)).Decode(&m); err != nil {
		return nil, fmt.Errorf("%s: %w", MapPath, err)
	}

	r := &Report{Surfaces: len(m.Surfaces), Locales: len(m.Locales)}
	checkMap(r, &m)
	checkSurfaces(r, &m, string(ui), string(server))
	checkPages(r, root, &m)
	checkBaseline(r, root, &m)
	checkMirror(r, root)
	checkFreshness(r, root)
	return r, nil
}

// ── the map itself ───────────────────────────────────────────

func checkMap(r *Report, m *Map) {
	if m.SchemaVersion != 1 {
		r.fail("%s: schemaVersion %d, this gate reads 1", MapPath, m.SchemaVersion)
	}
	if m.HelpRoot == "" || m.PageTemplate == "" || m.Index == "" || m.Product == "" {
		r.fail("%s: product, helpRoot, pageTemplate and index are required", MapPath)
	}
	if m.HelpRoot != "help" {
		r.fail("%s: helpRoot %q - the server serves the set under /help/ and the site publishes it as help/", MapPath, m.HelpRoot)
	}
	if !contains(m.Locales, m.SourceLocale) {
		r.fail("%s: the source locale %q is not among the locales %v", MapPath, m.SourceLocale, m.Locales)
	}
	for _, l := range m.Locales {
		if !contains(i18n.Codes, l) {
			r.fail("%s: locale %q is not an interface language of the product", MapPath, l)
		}
	}
	seen := map[string]bool{}
	anchors := map[string]bool{}
	for _, s := range m.Surfaces {
		if s.ID == "" || s.Name == "" {
			r.fail("%s: a surface needs an id and a name (row %+v)", MapPath, s)
			continue
		}
		if seen[s.ID] {
			r.fail("%s: surface %s appears twice", MapPath, s.ID)
		}
		seen[s.ID] = true
		if !forms[s.Form] {
			r.fail("%s: surface %s has form %q", MapPath, s.ID, s.Form)
		}
		switch s.Form {
		case "none", "inherit":
			if strings.TrimSpace(s.Reason) == "" {
				r.fail("%s: surface %s is %s without a reason (CONTEXT-HELP rule 4)", MapPath, s.ID, s.Form)
			}
		case "page", "both":
			if s.Help == "" {
				r.fail("%s: surface %s has form %s and no help id", MapPath, s.ID, s.Form)
			}
		case "layer":
			r.fail("%s: surface %s is a layer; this product ships pages only, so the form has no code behind it", MapPath, s.ID)
		}
		if s.Form == "inherit" {
			found := false
			for _, p := range m.Surfaces {
				found = found || p.ID == s.Parent
			}
			if !found {
				r.fail("%s: surface %s inherits from %q, which is no row", MapPath, s.ID, s.Parent)
			}
		}
		for _, p := range s.Pages {
			if !strings.HasPrefix(p.ID, s.ID+".") {
				r.fail("%s: page row %s does not belong to surface %s", MapPath, p.ID, s.ID)
			}
			if p.Anchor == "" || anchors[s.ID+"#"+p.Anchor] {
				r.fail("%s: page row %s has an empty or repeated anchor", MapPath, p.ID)
			}
			anchors[s.ID+"#"+p.Anchor] = true
		}
	}
}

// ── the surfaces of the source against the map ───────────────

var (
	dataHelpRe = regexp.MustCompile(`data-help="([a-z0-9.-]+)"`)
	surfaceRe  = regexp.MustCompile(`\bsurface:\s*'([a-z0-9-]+)'`)
	routeRe    = regexp.MustCompile(`mux\.HandleFunc\("(/[^"]+)"`)
)

// dialogLiterals returns the text of every object literal passed to openDialog( - the dialogs the window can show.
func dialogLiterals(ui string) []string {
	var out []string
	i := 0
	for {
		j := strings.Index(ui[i:], "openDialog({")
		if j < 0 {
			return out
		}
		start := i + j + len("openDialog(")
		depth, end := 0, -1
		for k := start; k < len(ui) && end < 0; k++ {
			switch ui[k] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					end = k
				}
			}
		}
		if end < 0 {
			return out
		}
		out = append(out, ui[start:end+1])
		i = end
	}
}

func checkSurfaces(r *Report, m *Map, ui, server string) {
	rows := map[string]*Surface{}
	for i := range m.Surfaces {
		rows[m.Surfaces[i].ID] = &m.Surfaces[i]
	}
	evidence := map[string]bool{"main": true} // the converter window is the product
	inSource := map[string]bool{}
	for _, mt := range dataHelpRe.FindAllStringSubmatch(ui, -1) {
		id := mt[1]
		inSource[id] = true
		dot := strings.Index(id, ".")
		if dot < 0 {
			r.fail("%s: data-help=%q is not <surface>.<region>", UIPath, id)
			continue
		}
		evidence[id[:dot]] = true
		s, ok := rows[id[:dot]]
		if !ok {
			r.fail("%s: a region says data-help=%q but the map has no surface %s (a surface with no row)", UIPath, id, id[:dot])
			continue
		}
		hit := false
		for _, p := range s.Pages {
			hit = hit || p.ID == id
		}
		if !hit {
			r.fail("%s: region %s is in the window but not among the map's %s rows (a region with no entry)", UIPath, id, s.ID)
		}
	}
	for _, s := range m.Surfaces {
		for _, p := range s.Pages {
			if !inSource[p.ID] {
				r.fail("%s: the map lists region %s, which no element of the window carries (a row with no surface)", MapPath, p.ID)
			}
		}
	}
	for _, mt := range surfaceRe.FindAllStringSubmatch(ui, -1) {
		evidence[mt[1]] = true
		if _, ok := rows[mt[1]]; !ok {
			r.fail("%s: a dialog says surface: %q but the map has no such surface (a surface with no row)", UIPath, mt[1])
		}
	}
	for _, lit := range dialogLiterals(ui) {
		if !surfaceRe.MatchString(lit) {
			short := strings.Join(strings.Fields(lit), " ")
			if len(short) > 80 {
				short = short[:80]
			}
			r.fail("%s: an openDialog call names no surface: %s", UIPath, short)
		}
	}
	routes := map[string]bool{}
	for _, mt := range routeRe.FindAllStringSubmatch(server, -1) {
		routes[mt[1]] = true
	}
	for _, s := range m.Surfaces {
		if strings.HasPrefix(s.Source, "route:") {
			if route := strings.TrimPrefix(s.Source, "route:"); !routes[route] {
				r.fail("%s: surface %s says source %s, which the server does not route (a row with no surface)", MapPath, s.ID, s.Source)
			}
			evidence[s.ID] = true
		}
	}
	for _, s := range m.Surfaces {
		if !evidence[s.ID] {
			r.fail("%s: surface %s has a row but nothing in the window, its dialogs or its routes is that surface (a row with no surface)", MapPath, s.ID)
		}
	}
}

// ── the pages ────────────────────────────────────────────────

type pageInfo struct {
	rel string
	raw string
	doc *html.Node
	ids map[string]bool
}

type pageSet struct {
	r     *Report
	root  string
	pages map[string]*pageInfo
}

func (ps *pageSet) load(rel string) *pageInfo {
	if p, ok := ps.pages[rel]; ok {
		return p
	}
	raw, err := os.ReadFile(filepath.Join(ps.root, filepath.FromSlash(rel)))
	if err != nil {
		ps.pages[rel] = nil
		return nil
	}
	doc, err := html.Parse(bytes.NewReader(raw))
	if err != nil {
		ps.r.fail("%s: does not parse: %v", rel, err)
		ps.pages[rel] = nil
		return nil
	}
	p := &pageInfo{rel: rel, raw: string(raw), doc: doc, ids: map[string]bool{}}
	walk(doc, func(n *html.Node) {
		if n.Type == html.ElementNode {
			if id := attr(n, "id"); id != "" {
				p.ids[id] = true
			}
		}
	})
	ps.pages[rel] = p
	return p
}

func pageFile(m *Map, locale, help string) string {
	return path.Join(HelpDir, strings.NewReplacer("{locale}", locale, "{help}", help).Replace(m.PageTemplate))
}

// The anatomy of a page (CONTEXT-HELP rule 8), as markers in the order they must appear.
var anatomy = []string{`class="help-crumbs"`, `<h1>`, `class="lead"`, `class="shot"`, `<h2>`, `class="elements"`, `class="related"`, `class="help-foot"`}

func checkPages(r *Report, root string, m *Map) {
	ps := &pageSet{r: r, root: root, pages: map[string]*pageInfo{}}
	var helpIDs []string
	for _, s := range m.Surfaces {
		if s.Form == "page" || s.Form == "both" {
			helpIDs = append(helpIDs, s.Help)
		}
	}
	r.Pages = len(helpIDs)

	sourceFiles := dirFiles(root, path.Join(HelpDir, m.SourceLocale))
	for _, loc := range m.Locales {
		// the locale holds exactly the files the source locale holds: no page and no screenshot missing or stray
		if have := dirFiles(root, path.Join(HelpDir, loc)); !equalSets(have, sourceFiles) {
			r.fail("%s/%s: its files differ from the source locale's (missing %v, extra %v)", HelpDir, loc, subtract(sourceFiles, have), subtract(have, sourceFiles))
		}
		for _, id := range append(append([]string{}, helpIDs...), m.Index) {
			rel := pageFile(m, loc, id)
			p := ps.load(rel)
			if p == nil {
				r.fail("%s: the page %s is missing in locale %s (CONTEXT-HELP rule 13)", rel, id, loc)
				continue
			}
			ps.checkDoc(p, loc, id == m.Index)
		}
		for _, s := range m.Surfaces {
			if s.Form != "page" && s.Form != "both" {
				continue
			}
			p := ps.load(pageFile(m, loc, s.Help))
			if p == nil {
				continue
			}
			for _, row := range s.Pages {
				if !p.ids[row.Anchor] {
					r.fail("%s: the entry #%s (%s) is missing, so F1 in that region would open the top of the page", p.rel, row.Anchor, row.ID)
				}
			}
		}
	}
	for _, shared := range []string{"sza-kit.css", "help.css", "page.js", "captures.json", "freshness.json"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(HelpDir), shared)); err != nil {
			r.fail("%s/%s is missing", HelpDir, shared)
		}
	}
	checkCaptures(r, root, m, helpIDs)
	r.Files = len(dirFilesDeep(root, HelpDir))
}

func (ps *pageSet) checkDoc(p *pageInfo, locale string, isIndex bool) {
	r := ps.r
	html0 := findFirst(p.doc, "html")
	if html0 == nil || attr(html0, "lang") != locale {
		r.fail("%s: <html lang> is not %q", p.rel, locale)
	}
	wantDir := "ltr"
	if i18n.IsRTL(locale) {
		wantDir = "rtl"
	}
	if html0 == nil || attr(html0, "dir") != wantDir {
		r.fail("%s: <html dir> is not %q", p.rel, wantDir)
	}
	if t := findFirst(p.doc, "title"); t == nil || strings.TrimSpace(textOf(t)) == "" {
		r.fail("%s: no <title>", p.rel)
	}
	referrer := false
	walk(p.doc, func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" && attr(n, "name") == "referrer" && attr(n, "content") == "no-referrer" {
			referrer = true
		}
	})
	if !referrer {
		r.fail(`%s: no <meta name="referrer" content="no-referrer"> (CONTEXT-HELP rule 12)`, p.rel)
	}

	// rule 8: the parts, in order (the index page is a list of pages, not the help of a surface)
	if !isIndex {
		last := -1
		for _, mk := range anatomy {
			at := strings.Index(p.raw, mk)
			switch {
			case at < 0:
				r.fail("%s: the page has no %s (CONTEXT-HELP rule 8)", p.rel, mk)
			case at < last:
				r.fail("%s: %s is out of order (CONTEXT-HELP rule 8)", p.rel, mk)
			default:
				last = at
			}
		}
	} else if !strings.Contains(p.raw, `<h1>`) || !strings.Contains(p.raw, `class="help-foot"`) {
		r.fail("%s: the index has no title or footer", p.rel)
	}

	// the entries and the pins
	entries := map[string]bool{}
	pins := map[string]bool{}
	walk(p.doc, func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		switch {
		case n.Data == "details" && hasClass(n, "el"):
			entries[attr(n, "id")] = true
			if findFirst(n, "summary") == nil {
				r.fail("%s: entry #%s has no summary (rule 9: the disclosure is a native <details>)", p.rel, attr(n, "id"))
			}
		case n.Data == "a" && hasClass(n, "pin"):
			pins[strings.TrimPrefix(attr(n, "href"), "#")] = true
		case n.Data == "img":
			if _, ok := attrOK(n, "alt"); !ok {
				r.fail("%s: an image has no alt attribute", p.rel)
			}
			if src := attr(n, "src"); strings.Contains(src, "/") {
				r.fail("%s: image %s is not in the page's own locale folder (a screenshot in another language than the page is a defect)", p.rel, src)
			}
		}
	})
	if !isIndex && len(entries) == 0 {
		r.fail("%s: no element entries", p.rel)
	}
	for id := range pins {
		if !entries[id] {
			r.fail("%s: a pin points at #%s, which is no entry", p.rel, id)
		}
	}
	for id := range entries {
		if !pins[id] {
			r.fail("%s: entry #%s has no pin on the screenshot", p.rel, id)
		}
	}

	// rule 12 and the links
	hasIndexLink := false
	dir := path.Dir(p.rel)
	walk(p.doc, func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		if n.Data == "script" && attr(n, "src") == "" {
			r.fail("%s: an inline <script> - the page's script is the one file page.js (rule 12, and the server's content policy)", p.rel)
		}
		for _, name := range []string{"href", "src"} {
			v, ok := attrOK(n, name)
			if !ok {
				continue
			}
			external := strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") || strings.HasPrefix(v, "//")
			switch {
			case strings.HasPrefix(v, "mailto:"):
				if n.Data != "a" {
					r.fail("%s: <%s %s=%q> is not a link", p.rel, n.Data, name, v)
				}
				r.Links++
			case external:
				if n.Data != "a" || name != "href" {
					r.fail("%s: <%s %s=%q> makes a request to another origin (CONTEXT-HELP rule 12)", p.rel, n.Data, name, v)
				}
				r.Links++
				if strings.HasPrefix(v, SiteBase) {
					ps.checkSiteLink(p, v)
				}
			default:
				r.Links++
				target, frag, _ := strings.Cut(v, "#")
				if i := strings.Index(target, "?"); i >= 0 {
					target = target[:i]
				}
				if target == "" { // the same page
					if frag != "" && !p.ids[frag] {
						r.fail("%s: link #%s names no element of the page", p.rel, frag)
					}
					continue
				}
				if n.Data == "a" && target == "index.html" {
					hasIndexLink = true
				}
				full := path.Clean(path.Join(dir, target))
				if _, err := os.Stat(filepath.Join(ps.root, filepath.FromSlash(full))); err != nil {
					r.fail("%s: %s=%q resolves to %s, which is not there", p.rel, name, v, full)
					continue
				}
				if frag != "" && strings.HasSuffix(full, ".html") {
					if tp := ps.load(full); tp != nil && !tp.ids[frag] {
						r.fail("%s: link %s names no element of %s", p.rel, v, full)
					}
				}
			}
		}
		if n.Data == "style" || attr(n, "style") != "" {
			text := attr(n, "style")
			if n.Data == "style" {
				text = textOf(n)
			}
			if strings.Contains(strings.ToLower(text), "url(http") || strings.Contains(strings.ToLower(text), "@import") {
				r.fail("%s: a style fetches from outside the set (rule 12)", p.rel)
			}
		}
	})
	if !isIndex && !hasIndexLink {
		r.fail("%s: the page does not reach the help index (a page never dead-ends, rule 11)", p.rel)
	}

	// the house text style for the languages the author writes (CLAUDE.md, tests/typography_test.go)
	if locale == "en" || locale == "ru" || locale == "uk" {
		text := visibleText(p.doc)
		for _, bad := range badStyle() {
			if strings.Contains(text, bad.s) {
				r.fail("%s: the text carries %s (house text style: a short hyphen, and two dots)", p.rel, bad.name)
			}
		}
	}
}

// SiteBase is the product site; a link into it must name a page and a section the tree serves (the site's own
// check runs online at the release, this one runs offline on every gate).
const SiteBase = "https://serzhyale.github.io/doc-html-translate/"

func (ps *pageSet) checkSiteLink(p *pageInfo, href string) {
	rest := strings.TrimPrefix(href, SiteBase)
	target, frag, _ := strings.Cut(rest, "#")
	if target == "" {
		target = "index.html"
	}
	raw, err := os.ReadFile(filepath.Join(ps.root, filepath.FromSlash(target)))
	if err != nil {
		ps.r.fail("%s: the site address %s resolves to %s, which the tree does not serve", p.rel, href, target)
		return
	}
	if frag != "" && !strings.Contains(string(raw), `id="`+frag+`"`) {
		ps.r.fail("%s: the site address %s names the section #%s, which %s does not have", p.rel, href, frag, target)
	}
}

type badToken struct{ s, name string }

// The marks the house style forbids, spelled by rune so this file's own literals stay clean.
func badStyle() []badToken {
	return []badToken{
		{string(rune(0x2014)), "a long dash"},
		{string(rune(0x2013)), "an en dash"},
		{string(rune(0x2026)), "an ellipsis character"},
		{strings.Repeat(".", 3), "a three-dot ellipsis"},
	}
}

// captures.json: the pin of every element is a percentage of its image, and every page has its shots.
func checkCaptures(r *Report, root string, m *Map, helpIDs []string) {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(HelpDir), "captures.json"))
	if err != nil {
		return
	}
	var c struct {
		Captures map[string]map[string]struct {
			W    int                               `json:"w"`
			H    int                               `json:"h"`
			Pins map[string]struct{ X, Y float64 } `json:"pins"`
		} `json:"captures"`
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		r.fail("%s/captures.json: %v", HelpDir, err)
		return
	}
	for _, loc := range m.Locales {
		for _, id := range helpIDs {
			shots := c.Captures[loc+"/"+id]
			if len(shots) == 0 {
				r.fail("%s/captures.json: no capture of %s in %s (CONTEXT-HELP rule 14)", HelpDir, id, loc)
			}
			for name, s := range shots {
				if s.W <= 0 || s.H <= 0 {
					r.fail("%s/captures.json: %s/%s/%s has no size", HelpDir, loc, id, name)
				}
				for el, p := range s.Pins {
					if p.X < 0 || p.X > 100 || p.Y < 0 || p.Y > 100 {
						r.fail("%s/captures.json: the pin of %s/%s/%s is outside its image (%.1f%%, %.1f%%)", HelpDir, loc, id, el, p.X, p.Y)
					}
				}
			}
		}
	}
}

// ── the baseline: locales the GUI ships and the help does not yet have ──

// A set baseline (CHECK-BASELINE rule 3): one identifier per line, sorted, "locale:<code>". The set may fall and
// never rise: a GUI locale with no pages that the file does not list is new debt and fails; a listed locale that now
// has pages is named so the line can be pruned.
func checkBaseline(r *Report, root string, m *Map) {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(BaselinePath)))
	if err != nil {
		r.fail("%s is missing: the locales without pages are accepted there, one per line (CHECK-BASELINE)", BaselinePath)
		return
	}
	var lines []string
	for _, l := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
		if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
			lines = append(lines, l)
		}
	}
	if !sort.StringsAreSorted(lines) {
		r.fail("%s is not sorted; keep it sorted so a change reads in a diff (CHECK-BASELINE rule 3)", BaselinePath)
	}
	accepted := map[string]bool{}
	for _, l := range lines {
		accepted[l] = true
	}
	missing := map[string]bool{}
	for _, code := range i18n.Codes {
		if !contains(m.Locales, code) {
			id := "locale:" + code
			missing[id] = true
			if !accepted[id] {
				r.fail("the GUI ships %s but the help has no pages in it and %s does not accept that: new debt may not be added (CHECK-BASELINE rule 2)", code, BaselinePath)
			}
		}
	}
	for _, l := range lines {
		if !missing[l] {
			r.advise("%s lists %s, which is no longer a finding: prune the line", BaselinePath, l)
		} else {
			r.Debt++
		}
	}
}

// ── the two copies of the set ────────────────────────────────

// help/ at the repository root is the site's copy, published from the same bytes as the set embedded in the exe.
func checkMirror(r *Report, root string) {
	a := dirFilesDeep(root, HelpDir)
	b := dirFilesDeep(root, MirrorDir)
	if len(b) == 0 {
		r.fail("%s/ is missing: the site's copy of the help set is the generator's second output", MirrorDir)
		return
	}
	for rel := range a {
		other, ok := b[rel]
		if !ok {
			r.fail("%s/%s is in the embedded set but not in the site's copy", MirrorDir, rel)
			continue
		}
		if !bytes.Equal(a[rel], other) {
			r.fail("%s/%s differs from %s/%s: the two copies must be the same bytes", MirrorDir, rel, HelpDir, rel)
		}
	}
	for rel := range b {
		if _, ok := a[rel]; !ok {
			r.fail("%s/%s is in the site's copy but not in the embedded set", MirrorDir, rel)
		}
	}
}

// ── helpers ──────────────────────────────────────────────────

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func walk(n *html.Node, f func(*html.Node)) {
	f(n)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, f)
	}
}

func findFirst(n *html.Node, tag string) *html.Node {
	var out *html.Node
	walk(n, func(x *html.Node) {
		if out == nil && x.Type == html.ElementNode && x.Data == tag {
			out = x
		}
	})
	return out
}

func attrOK(n *html.Node, name string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val, true
		}
	}
	return "", false
}

func attr(n *html.Node, name string) string { v, _ := attrOK(n, name); return v }

func hasClass(n *html.Node, class string) bool {
	return contains(strings.Fields(attr(n, "class")), class)
}

func textOf(n *html.Node) string {
	var b strings.Builder
	walk(n, func(x *html.Node) {
		if x.Type == html.TextNode {
			b.WriteString(x.Data)
		}
	})
	return b.String()
}

// visibleText is the text a reader sees, including the alt text of the screenshots and the title.
func visibleText(doc *html.Node) string {
	var b strings.Builder
	var rec func(n *html.Node)
	rec = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return
		}
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			b.WriteByte(' ')
		}
		if n.Type == html.ElementNode && n.Data == "img" {
			b.WriteString(attr(n, "alt"))
			b.WriteByte(' ')
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			rec(c)
		}
	}
	rec(doc)
	return b.String()
}

// dirFiles lists the file names directly in dir (relative to root).
func dirFiles(root, dir string) map[string]bool {
	out := map[string]bool{}
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
	if err != nil {
		return out
	}
	for _, e := range entries {
		if !e.IsDir() {
			out[e.Name()] = true
		}
	}
	return out
}

// dirFilesDeep reads every file under dir: its path relative to dir, slash-separated, and its bytes.
func dirFilesDeep(root, dir string) map[string][]byte {
	out := map[string][]byte{}
	base := filepath.Join(root, filepath.FromSlash(dir))
	_ = filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		raw, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		rel, _ := filepath.Rel(base, p)
		out[filepath.ToSlash(rel)] = raw
		return nil
	})
	return out
}

func equalSets(a, b map[string]bool) bool {
	return len(subtract(a, b)) == 0 && len(subtract(b, a)) == 0
}

func subtract(a, b map[string]bool) []string {
	var out []string
	for k := range a {
		if !b[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
