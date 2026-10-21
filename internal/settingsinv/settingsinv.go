// Package settingsinv is the settings inventory of the GUI launcher and the gate that keeps it aligned with the
// documentation (APP-SETTINGS rule 11, SETTINGS-NAVIGATION section 6; ticket 122, item 4).
//
// The inventory is generated, never written: the persisted keys of collectSettings() in cmd/doc-html-ui/ui.html, each
// traced to its control, the page it sits on, its caption (the dictionary key and the English text), its default, the
// help region that explains it and, where the window passes one to the converter, the command-line flag. The render
// is docs/settings-inventory.json. The gate holds three things true:
//
//   - the render equals what the window says now (a setting added without regenerating it fails);
//   - every persisted key is either a setting of the window or a recorded piece of session context, and every setting
//     has a control, a page, a caption in the dictionary and a help region the help map lists;
//   - the manual says it: the caption of every setting is a name in the help copy deck of the surface that owns its
//     help region, and a setting that maps to a command-line flag is a row of the three README flag tables, defined in
//     internal/config/flags.go and really passed by the window's command builder.
//
// It is developer tooling: the GUI does not import it. tools/settingsinv is its command, scripts/settings-inventory.ps1
// its gate wrapper.
package settingsinv

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/net/html"

	"doc-html-translate/internal/helpcheck"
)

// Files the gate reads, relative to the repository root.
const (
	UIPath        = "cmd/doc-html-ui/ui.html"
	I18NPath      = "cmd/doc-html-ui/i18n.js"
	ServerPath    = "cmd/doc-html-ui/main.go"
	FlagsPath     = "internal/config/flags.go"
	DeckDir       = "tools/helpgen/copy"
	InventoryPath = "docs/settings-inventory.json"
)

// READMEs are the three flag tables a setting that maps to a flag must appear in.
var READMEs = []string{"README.md", "README_RU.md", "README_UK.md"}

// Row is one setting of the window.
type Row struct {
	Key       string   `json:"key"`
	Control   string   `json:"control"`
	Kind      string   `json:"kind"`
	Page      string   `json:"page"`
	Caption   string   `json:"caption"`
	CaptionEN string   `json:"captionEN"`
	Default   string   `json:"default,omitempty"`
	Help      string   `json:"help"`
	Flags     []string `json:"flags,omitempty"`
	Note      string   `json:"note,omitempty"`
}

// Context is a persisted key that is session state and not a setting: the user never sets it, so it has no caption.
type Context struct {
	Key    string `json:"key"`
	Reason string `json:"reason"`
}

// Inventory is the render.
type Inventory struct {
	SchemaVersion int       `json:"schemaVersion"`
	Source        string    `json:"source"`
	Settings      []Row     `json:"settings"`
	Context       []Context `json:"context"`
}

// Report is one run of the gate.
type Report struct {
	Failures []string
	Settings int
	Context  int
	Flags    int
}

func (r *Report) fail(format string, a ...any) {
	r.Failures = append(r.Failures, fmt.Sprintf(format, a...))
}

// Detail is the parenthesis of the verdict line.
func (r *Report) Detail() string {
	if len(r.Failures) > 0 {
		return fmt.Sprintf("%d finding(s)", len(r.Failures))
	}
	return fmt.Sprintf("%d settings, %d context keys, %d flags in the README tables", r.Settings, r.Context, r.Flags)
}

// Code is the CHECK-VERDICT exit code: 1 on a failure, 0 otherwise.
func (r *Report) Code() int {
	if len(r.Failures) > 0 {
		return 1
	}
	return 0
}

// ── what the window persists, and what each key is ───────────────────────────

// sessionContext are the persisted keys that are state of the session, with the reason each is not a setting.
var sessionContext = map[string]string{
	"assocPrompted": "whether the first-run Windows question was answered; the answer itself lives in Windows",
	"recent":        "the recent-documents list, a convenience the user clears with a button",
	"win":           "the last window geometry, restored once at the next start",
	"page":          "the page the user left, restored at the next start",
	"groups":        "the open or closed state of each group by its stable id",
	"pageScrolls":   "each page's viewport as a stable anchor and an offset",
}

// keyControls are the persisted keys whose value does not read as el('<id>').value or .checked in collectSettings().
var keyControls = map[string]string{
	"lang":          "langSwitch",
	"theme":         "themeSwitch",
	"engine":        "engine",
	"uiSize":        "uiSize",
	"textSize":      "textSize",
	"inputProfile":  "inputProfile",
	"ocrLangChoice": "ocrLang",
}

// keyFlags are the command-line flags the window passes for a setting (cmd/doc-html-ui/main.go builds the arguments).
var keyFlags = map[string][]string{
	"lang":           {"-ui-lang"},
	"engine":         {"-notranslate", "-google", "-ollama"},
	"output":         {"-folder"},
	"srcLang":        {"-src"},
	"dstLang":        {"-dst"},
	"splitSize":      {"-split"},
	"tocDepth":       {"-toc-depth"},
	"maxCost":        {"-max-cost"},
	"ollamaModel":    {"-ollama-model"},
	"ollamaParallel": {"-ollama-parallel"},
	"ollamaCtx":      {"-ollama-ctx"},
	"singlePage":     {"-multipage"},
	"noOpen":         {"-noopen"},
	"force":          {"-force"},
	"verbose":        {"-v"},
	"ocr":            {"-ocr"},
	"ocrLangChoice":  {"-ocr-lang"},
}

// keyNotes explain a row that does not read one to one against its flags.
var keyNotes = map[string]string{
	"singlePage": "ticked is the default; unticked passes -multipage",
	"engine":     "Chrome passes no flag; Google Cloud, Ollama and Don't translate pass -google, -ollama and -notranslate",
}

var (
	collectBodyRe = regexp.MustCompile(`(?s)function collectSettings\(\)\s*\{\s*return\s*\{(.*?)\n    \};\s*\n\}`)
	keyLineRe     = regexp.MustCompile(`^\s*(\w+):\s*(.*?),?\s*(?://.*)?$`)
	elValueRe     = regexp.MustCompile(`^el\('(\w+)'\)\.(?:value|checked)$`)
	spreadRe      = regexp.MustCompile(`^\.\.\.(\w+)\(\),?$`)
)

type persisted struct {
	key, control string
	context      bool
}

// persistedKeys reads the keys collectSettings() returns, in its order.
func persistedKeys(r *Report, ui string) []persisted {
	m := collectBodyRe.FindStringSubmatch(ui)
	if m == nil {
		r.fail("%s: collectSettings() was not found in its usual shape (return { key: value, .. };) - the gate cannot list what the window persists", UIPath)
		return nil
	}
	var out []persisted
	for _, line := range strings.Split(m[1], "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "//") {
			continue
		}
		if sp := spreadRe.FindStringSubmatch(t); sp != nil {
			if sp[1] == "ocrLangSetting" {
				out = append(out, persisted{key: "ocrLangChoice", control: keyControls["ocrLangChoice"]})
				continue
			}
			r.fail("%s: collectSettings() spreads %s(), which this gate does not know: teach internal/settingsinv which keys it adds", UIPath, sp[1])
			continue
		}
		k := keyLineRe.FindStringSubmatch(line)
		if k == nil {
			r.fail("%s: a line of collectSettings() is not `key: value`: %s", UIPath, t)
			continue
		}
		key, val := k[1], strings.TrimSpace(k[2])
		switch {
		case sessionContext[key] != "":
			out = append(out, persisted{key: key, context: true})
		case elValueRe.MatchString(val):
			out = append(out, persisted{key: key, control: elValueRe.FindStringSubmatch(val)[1]})
		case keyControls[key] != "":
			out = append(out, persisted{key: key, control: keyControls[key]})
		default:
			r.fail("%s: collectSettings() persists %q, which is neither a setting this gate can trace to a control nor a recorded piece of session context - add it to keyControls or sessionContext in internal/settingsinv", UIPath, key)
		}
	}
	return out
}

// ── the window ───────────────────────────────────────────────

type window struct {
	byID     map[string]*html.Node
	labelFor map[string][]*html.Node
}

func attr(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

func hasAttr(n *html.Node, name string) bool {
	for _, a := range n.Attr {
		if a.Key == name {
			return true
		}
	}
	return false
}

func hasClass(n *html.Node, c string) bool {
	for _, f := range strings.Fields(attr(n, "class")) {
		if f == c {
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

func parseWindow(ui string) (*window, error) {
	doc, err := html.Parse(strings.NewReader(ui))
	if err != nil {
		return nil, err
	}
	w := &window{byID: map[string]*html.Node{}, labelFor: map[string][]*html.Node{}}
	walk(doc, func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		if id := attr(n, "id"); id != "" {
			if _, dup := w.byID[id]; !dup {
				w.byID[id] = n
			}
		}
		if n.Data == "label" {
			if f := attr(n, "for"); f != "" {
				w.labelFor[f] = append(w.labelFor[f], n)
			}
		}
	})
	return w, nil
}

// ancestor returns the nearest ancestor (or the node itself) the predicate accepts.
func ancestor(n *html.Node, ok func(*html.Node) bool) *html.Node {
	for ; n != nil; n = n.Parent {
		if n.Type == html.ElementNode && ok(n) {
			return n
		}
	}
	return nil
}

// firstI18n is the dictionary key of the first element at or below n that carries one.
func firstI18n(n *html.Node) string {
	key := ""
	walk(n, func(c *html.Node) {
		if key == "" && c.Type == html.ElementNode {
			key = attr(c, "data-i18n")
		}
	})
	return key
}

// captionKey is the dictionary key of the caption a control has: a <label for> of it, the label of the field or option
// that holds it, or the group's own aria name.
func (w *window) captionKey(control string) string {
	n := w.byID[control]
	if n == nil {
		return ""
	}
	for _, l := range w.labelFor[control] {
		if k := firstI18n(l); k != "" {
			return k
		}
	}
	if box := ancestor(n.Parent, func(a *html.Node) bool { return hasClass(a, "field") || hasClass(a, "opt") }); box != nil {
		key := ""
		walk(box, func(c *html.Node) {
			if key == "" && c.Type == html.ElementNode && c.Data == "label" {
				key = firstI18n(c)
			}
		})
		if key != "" {
			return key
		}
	}
	return attr(n, "data-i18n-aria")
}

func kindOf(n *html.Node) string {
	switch {
	case n.Data == "select":
		return "select"
	case n.Data == "textarea":
		return "text"
	case n.Data == "input":
		if t := attr(n, "type"); t != "" {
			return t
		}
		return "text"
	case attr(n, "role") == "radiogroup":
		return "radio"
	}
	return n.Data
}

func defaultOf(n *html.Node) string {
	switch kindOf(n) {
	case "checkbox":
		return strconv.FormatBool(hasAttr(n, "checked"))
	case "radio":
		v := ""
		walk(n, func(c *html.Node) {
			if c.Type == html.ElementNode && c.Data == "input" && hasAttr(c, "checked") && v == "" {
				v = attr(c, "value")
			}
		})
		return v
	case "select":
		first, picked, seen := "", "", false
		walk(n, func(c *html.Node) {
			if c.Type == html.ElementNode && c.Data == "option" {
				if !seen {
					first, seen = attr(c, "value"), true
				}
				if hasAttr(c, "selected") && picked == "" {
					picked = attr(c, "value")
				}
			}
		})
		if picked != "" {
			return picked
		}
		return first
	}
	return attr(n, "value")
}

// englishText is a dictionary entry of the English block (the first block of i18n.js), without a trailing colon.
func englishText(i18n, key string) (string, bool) {
	m := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `:\s*("(?:[^"\\]|\\.)*")`).FindStringSubmatch(i18n)
	if m == nil {
		return "", false
	}
	text, err := strconv.Unquote(m[1])
	if err != nil {
		text = strings.Trim(m[1], `"`)
	}
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(text), ":")), true
}

// ── the inventory ────────────────────────────────────────────

// Build derives the inventory from the window. Findings land in the report.
func Build(root string, r *Report) *Inventory {
	rd := func(rel string) string {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			r.fail("%s cannot be read: %v", rel, err)
		}
		return string(b)
	}
	ui, i18n := rd(UIPath), rd(I18NPath)
	inv := &Inventory{SchemaVersion: 1, Source: UIPath + " collectSettings()", Settings: []Row{}, Context: []Context{}}
	if ui == "" {
		return inv
	}
	w, err := parseWindow(ui)
	if err != nil {
		r.fail("%s does not parse: %v", UIPath, err)
		return inv
	}
	for _, p := range persistedKeys(r, ui) {
		if p.context {
			inv.Context = append(inv.Context, Context{Key: p.key, Reason: sessionContext[p.key]})
			continue
		}
		n := w.byID[p.control]
		if n == nil {
			r.fail("%s: the setting %q has no element with id=%q in the window", UIPath, p.key, p.control)
			continue
		}
		row := Row{Key: p.key, Control: p.control, Kind: kindOf(n), Default: defaultOf(n), Flags: keyFlags[p.key], Note: keyNotes[p.key]}
		if page := ancestor(n, func(a *html.Node) bool { return a.Data == "section" && hasClass(a, "page") }); page != nil {
			row.Page = strings.TrimPrefix(attr(page, "id"), "page-")
		} else {
			r.fail("%s: the setting %q (#%s) is on no page of the window", UIPath, p.key, p.control)
		}
		if h := ancestor(n, func(a *html.Node) bool { return attr(a, "data-help") != "" }); h != nil {
			row.Help = attr(h, "data-help")
		} else {
			r.fail("%s: the setting %q (#%s) sits in no region with a data-help, so F1 on it explains nothing", UIPath, p.key, p.control)
		}
		if row.Caption = w.captionKey(p.control); row.Caption == "" {
			r.fail("%s: the setting %q (#%s) has no caption: a label for it, a label of its field, or an aria name from the dictionary", UIPath, p.key, p.control)
		} else if en, ok := englishText(i18n, row.Caption); ok {
			row.CaptionEN = en
		} else {
			r.fail("%s: the caption key %q of the setting %q is not in the English dictionary", I18NPath, row.Caption, p.key)
		}
		inv.Settings = append(inv.Settings, row)
	}
	return inv
}

// Marshal is the render: two-space JSON with a final newline.
func Marshal(inv *Inventory) []byte {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(inv)
	return b.Bytes()
}

// Render writes the inventory the window says now to InventoryPath.
func Render(root string) (*Report, error) {
	r := &Report{}
	inv := Build(root, r)
	if len(r.Failures) > 0 {
		return r, nil
	}
	return r, os.WriteFile(filepath.Join(root, filepath.FromSlash(InventoryPath)), Marshal(inv), 0o644)
}

// ── the gate ─────────────────────────────────────────────────

var (
	flagDefinition = regexp.MustCompile(`\bfs\.(?:Bool|String|Int|Int64|Uint|Uint64|Float64|Duration)(?:Var)?\(\s*(?:&[\w.]+,\s*)?"([^"]+)"`)
	readmeFlagRow  = regexp.MustCompile("(?m)^\\|\\s*`-([a-z0-9-]+)`\\s*\\|")
)

// Run judges the tree under root. The error is for a tree it cannot read at all (COULD NOT VERIFY).
func Run(root string) (*Report, error) {
	rd := func(rel string) ([]byte, error) { return os.ReadFile(filepath.Join(root, filepath.FromSlash(rel))) }
	mapRaw, err := rd(helpcheck.MapPath)
	if err != nil {
		return nil, err
	}
	var m helpcheck.Map
	if err := json.Unmarshal(mapRaw, &m); err != nil {
		return nil, fmt.Errorf("%s: %w", helpcheck.MapPath, err)
	}
	flagsSrc, err := rd(FlagsPath)
	if err != nil {
		return nil, err
	}
	server, err := rd(ServerPath)
	if err != nil {
		return nil, err
	}

	r := &Report{}
	inv := Build(root, r)
	r.Settings, r.Context = len(inv.Settings), len(inv.Context)

	// the help regions the map lists, and the surface that owns each
	regions := map[string]helpcheck.Surface{}
	for _, s := range m.Surfaces {
		for _, p := range s.Pages {
			regions[p.ID] = s
		}
	}
	defined := map[string]bool{}
	for _, f := range flagDefinition.FindAllStringSubmatch(string(flagsSrc), -1) {
		defined[f[1]] = true
	}
	readme := map[string]map[string]bool{}
	for _, name := range READMEs {
		raw, err := rd(name)
		if err != nil {
			return nil, err
		}
		rows := map[string]bool{}
		for _, f := range readmeFlagRow.FindAllStringSubmatch(string(raw), -1) {
			rows[f[1]] = true
		}
		readme[name] = rows
	}

	decks := map[string]map[string]bool{}
	deckNames := func(help string) map[string]bool {
		if d, ok := decks[help]; ok {
			return d
		}
		d := map[string]bool{}
		decks[help] = d
		raw, err := rd(path.Join(DeckDir, help+".json"))
		if err != nil {
			r.fail("%s: the help copy deck of %q cannot be read: %v", DeckDir, help, err)
			return d
		}
		var doc any
		if err := json.Unmarshal(raw, &doc); err != nil {
			r.fail("%s/%s.json: %v", DeckDir, help, err)
			return d
		}
		collectNames(doc, d)
		return d
	}

	flagsSeen := map[string]bool{}
	for _, row := range inv.Settings {
		at := fmt.Sprintf("setting %q (#%s, page %s)", row.Key, row.Control, row.Page)
		surface, ok := regions[row.Help]
		switch {
		case row.Help == "":
		case !ok:
			r.fail("%s: its help region %s is not a row of %s (the help map)", at, row.Help, helpcheck.MapPath)
		case surface.Help == "":
			r.fail("%s: its help region %s belongs to the surface %s, which has no help page", at, row.Help, surface.ID)
		default:
			if row.Caption != "" && !deckNames(surface.Help)[row.Caption] {
				r.fail("%s: the manual does not name it - %s/%s.json has neither a nameKey nor a {k:..} reference to %q (%q); add it to the element of %s, then regenerate the help set (APP-SETTINGS rule 11)",
					at, DeckDir, surface.Help, row.Caption, row.CaptionEN, row.Help)
			}
		}
		for _, f := range row.Flags {
			name := strings.TrimPrefix(f, "-")
			flagsSeen[name] = true
			if !defined[name] {
				r.fail("%s: it maps to %s, which %s does not define", at, f, FlagsPath)
			}
			if !strings.Contains(string(server), `"`+f+`"`) {
				r.fail("%s: it maps to %s, which %s never passes to the converter", at, f, ServerPath)
			}
			for _, name2 := range READMEs {
				if !readme[name2][name] {
					r.fail("%s: it maps to %s, which has no row in the flag table of %s", at, f, name2)
				}
			}
		}
	}
	r.Flags = len(flagsSeen)

	if len(r.Failures) == 0 {
		cur, err := rd(InventoryPath)
		want := Marshal(inv)
		switch {
		case err != nil:
			r.fail("%s is missing: generate it with `go run ./tools/settingsinv -render`", InventoryPath)
		case !bytes.Equal(bytes.ReplaceAll(cur, []byte("\r\n"), []byte("\n")), want):
			r.fail("%s differs from what the window says now (a setting, caption, page, default or help region changed): regenerate it with `go run ./tools/settingsinv -render`, never by hand", InventoryPath)
		}
	}
	sort.Strings(r.Failures)
	return r, nil
}

// keyRefRe is a reference to a dictionary entry inside the running text of a deck: {k:queueParallel}.
var keyRefRe = regexp.MustCompile(`\{k:([A-Za-z0-9]+)\}`)

// collectNames gathers every dictionary key a deck names: the nameKey of an element or an item, each of a nameKeys
// list, and every {k:key} reference in its running text (the help page prints the entry's own words there).
func collectNames(v any, out map[string]bool) {
	switch t := v.(type) {
	case string:
		for _, m := range keyRefRe.FindAllStringSubmatch(t, -1) {
			out[m[1]] = true
		}
	case map[string]any:
		for k, val := range t {
			switch k {
			case "nameKey":
				if s, ok := val.(string); ok {
					out[s] = true
				}
			case "nameKeys":
				if list, ok := val.([]any); ok {
					for _, e := range list {
						if s, ok := e.(string); ok {
							out[s] = true
						}
					}
				}
			default:
				collectNames(val, out)
			}
		}
	case []any:
		for _, e := range t {
			collectNames(e, out)
		}
	}
}
