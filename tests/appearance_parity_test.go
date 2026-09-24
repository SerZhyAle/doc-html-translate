package tests

// The shared appearance - the OCR overlay unit and the reader theme palette - is written once, in
// internal/appearance/appearance.json, and both editions derive their CSS from it: the desktop app
// at run time, the extension into marked regions of its stylesheets. These tests are the rule that
// keeps that true. They compare every declaration of every role and theme, not a chosen few - the
// hairline box-shadow ring that shipped on the extension's plate and nowhere else sat for the whole
// life of the feature under a guard that pinned three plate values out of seventeen.
//
// The comparison keys on the role or theme, never on a selector or a custom-property name: naming
// is per-edition (docs/PARITY.md "Intentional divergences"). A difference is legal only when the
// source's divergences list names it. See docs/PARITY.md "Shared appearance".

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"doc-html-translate/internal/appearance"
	"doc-html-translate/internal/htmlgen"
	"doc-html-translate/internal/ocr"
)

const (
	genBegin = "/* >>> generated from internal/appearance - do not edit by hand */"
	genEnd   = "/* <<< generated */"
)

// cssRule is one rule of a stylesheet. Label is the text of a `/* role: X */` or `/* theme: X */`
// comment immediately in front of it - how the extension's generated region says which role a
// rule is, so that the gate never has to know the extension's selectors.
type cssRule struct {
	Selector string
	Label    string
	Decls    [][2]string // property, normalized value; in source order
}

var (
	spaceRun   = regexp.MustCompile(`\s+`)
	shortHex   = regexp.MustCompile(`#([0-9a-fA-F])([0-9a-fA-F])([0-9a-fA-F])\b`)
	longHex    = regexp.MustCompile(`#[0-9a-fA-F]{6}\b`)
	labelRe    = regexp.MustCompile(`^(role|theme): (\S+)$`)
	commentAny = regexp.MustCompile(`(?s)/\*.*?\*/`)
)

// normValue makes two spellings of the same value compare equal: whitespace runs collapse, the
// space after a comma outside quotes goes (`Georgia, "Times New Roman"` == `Georgia,"Times New
// Roman"`), and hex colours are lowercase 6-digit (`#222` == `#222222`).
func normValue(v string) string {
	v = spaceRun.ReplaceAllString(strings.TrimSpace(v), " ")
	var sb strings.Builder
	var quote byte
	for i := 0; i < len(v); i++ {
		c := v[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == ' ' && i > 0 && v[i-1] == ',':
			continue
		}
		sb.WriteByte(c)
	}
	v = shortHex.ReplaceAllString(sb.String(), "#$1$1$2$2$3$3")
	return longHex.ReplaceAllStringFunc(v, strings.ToLower)
}

// splitTop splits s on sep where sep is outside quotes and parentheses - a data: URL carries
// semicolons and a selector list can hold :not(a, b).
func splitTop(s string, sep byte) []string {
	var out []string
	var quote byte
	depth, last := 0, 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '(':
			depth++
		case c == ')':
			depth--
		case c == sep && depth == 0:
			out = append(out, s[last:i])
			last = i + 1
		}
	}
	return append(out, s[last:])
}

// parseDeclarations parses a stylesheet into its rules, descending into @media and similar
// blocks. It is a reader for the stylesheets in this repo, not a general CSS parser: it assumes no
// brace inside a string.
func parseDeclarations(css string) []cssRule {
	var rules []cssRule
	label := ""
	i := 0
	for i < len(css) {
		switch {
		case strings.HasPrefix(css[i:], "/*"):
			end := strings.Index(css[i+2:], "*/")
			if end < 0 {
				return rules
			}
			text := strings.TrimSpace(css[i+2 : i+2+end])
			if m := labelRe.FindStringSubmatch(text); m != nil {
				label = text
			} else {
				label = ""
			}
			i += 2 + end + 2
		case css[i] == '}' || css[i] == ' ' || css[i] == '\n' || css[i] == '\t' || css[i] == '\r':
			i++
		default:
			open := strings.IndexByte(css[i:], '{')
			if open < 0 {
				return rules
			}
			prelude := strings.TrimSpace(commentAny.ReplaceAllString(css[i:i+open], ""))
			start := i + open + 1
			depth, j := 1, start
			for ; j < len(css) && depth > 0; j++ {
				switch css[j] {
				case '{':
					depth++
				case '}':
					depth--
				}
			}
			body := css[start : j-1]
			if strings.HasPrefix(prelude, "@") {
				if strings.Contains(body, "{") {
					rules = append(rules, parseDeclarations(body)...)
				}
			} else {
				r := cssRule{Selector: spaceRun.ReplaceAllString(prelude, " "), Label: label}
				for _, d := range splitTop(commentAny.ReplaceAllString(body, ""), ';') {
					p, v, ok := strings.Cut(d, ":")
					if !ok {
						continue
					}
					r.Decls = append(r.Decls, [2]string{strings.TrimSpace(p), normValue(v)})
				}
				rules = append(rules, r)
			}
			label = ""
			i = j
		}
	}
	return rules
}

// loadAppearance returns the source both editions derive from.
func loadAppearance(t *testing.T) appearance.Source {
	t.Helper()
	s := appearance.Load()
	if len(s.Themes) == 0 || len(s.Roles) == 0 {
		t.Fatal("internal/appearance/appearance.json decoded empty")
	}
	return s
}

// region returns the text between the one generated-marker pair in css, and css with that region
// cut out.
func region(t *testing.T, file, css string) (inside, outside string) {
	t.Helper()
	b, e := strings.Index(css, genBegin), strings.Index(css, genEnd)
	if b < 0 || e < b || strings.Count(css, genBegin) != 1 || strings.Count(css, genEnd) != 1 {
		t.Fatalf("%s: expected exactly one generated-region marker pair", file)
	}
	return css[b+len(genBegin) : e], css[:b] + css[e+len(genEnd):]
}

// edition is one side's view of the shared appearance, keyed by role / theme name.
type edition struct {
	name   string
	roles  map[string]map[string]string // role -> property -> value
	themes map[string][]string          // theme -> colour values, in emitted order
	props  map[string][]string          // theme -> custom-property names, same order
	sels   map[string]string            // role or theme -> the selector this edition uses
}

func newEdition(name string) *edition {
	return &edition{name: name, roles: map[string]map[string]string{}, themes: map[string][]string{},
		props: map[string][]string{}, sels: map[string]string{}}
}

func (e *edition) addRole(role, sel string, decls [][2]string) {
	m := map[string]string{}
	for _, d := range decls {
		m[d[0]] = d[1]
	}
	e.roles[role] = m
	e.sels[role] = sel
}

func (e *edition) addTheme(theme, sel string, decls [][2]string) {
	for _, d := range decls {
		e.themes[theme] = append(e.themes[theme], d[1])
		e.props[theme] = append(e.props[theme], d[0])
	}
	e.sels["theme:"+theme] = sel
}

// sourceEdition is the canonical source in the same shape, so the comparison has three sides.
func sourceEdition(s appearance.Source) *edition {
	e := newEdition("source")
	for role, decls := range s.Roles {
		var ds [][2]string
		for _, d := range decls {
			ds = append(ds, [2]string{d.Property, normValue(d.Value)})
		}
		e.addRole(role, "", ds)
	}
	p, v, _ := strings.Cut(appearance.HiddenPlateDecl, ":")
	e.addRole("hidden-plate", "", [][2]string{{p, v}})
	for _, th := range s.Themes {
		var ds [][2]string
		for _, c := range th.Colors {
			ds = append(ds, [2]string{c.Token, normValue(c.Value)})
		}
		e.addTheme(th.Name, "", ds)
	}
	return e
}

// desktopEdition reads the desktop app's CSS through the builders and the names the desktop code
// passes them. That the desktop code emits nothing else for these roles is
// TestAppearanceNoRoleDeclaredOutsideSource's job.
func desktopEdition(t *testing.T, s appearance.Source) *edition {
	t.Helper()
	e := newEdition("desktop")
	n := ocr.OverlayStyleNames
	bySel := map[string]string{n.Container: appearance.RoleContainer, n.Image: appearance.RoleImage,
		n.Plate: appearance.RolePlate, n.HiddenPlate: "hidden-plate"}
	for _, r := range parseDeclarations(appearance.OverlayCSS(n)) {
		role, ok := bySel[r.Selector]
		if !ok {
			t.Fatalf("desktop overlay CSS: rule %q maps to no role", r.Selector)
		}
		e.addRole(role, r.Selector, r.Decls)
	}
	pn := htmlgen.PaletteStyleNames
	for i, r := range parseDeclarations(appearance.PaletteCSS(pn)) {
		if i >= len(s.Themes) {
			t.Fatalf("desktop palette CSS: more rules than themes")
		}
		e.addTheme(s.Themes[i].Name, r.Selector, r.Decls)
	}
	return e
}

// extensionEdition reads the extension's generated regions as they sit in the shipped files.
func extensionEdition(t *testing.T) *edition {
	t.Helper()
	e := newEdition("extension")
	for _, file := range []string{"ocr-overlay.css", "viewer.css"} {
		inside, _ := region(t, "extension/src/"+file, readRepoFile(t, "extension", "src", file))
		for _, r := range parseDeclarations(inside) {
			kind, name, _ := strings.Cut(r.Label, ": ")
			switch kind {
			case "role":
				e.addRole(name, r.Selector, r.Decls)
			case "theme":
				e.addTheme(name, r.Selector, r.Decls)
			default:
				t.Errorf("extension/src/%s: generated rule %q has no role or theme label", file, r.Selector)
			}
		}
	}
	return e
}

// drift is one difference the comparator found.
type drift struct {
	Area, Name, Property string // Area is "role" or "theme"
	Detail               string
}

func (d drift) String() string {
	return fmt.Sprintf("%s %s: %s: %s", d.Area, d.Name, d.Property, d.Detail)
}

// exempt returns the index of the divergence entry that lets this edition's declaration of
// role/prop differ, or -1.
func exempt(divs []appearance.Divergence, role, prop, side string) int {
	for i, d := range divs {
		if d.Role == role && d.Property == prop && d.Edition == side {
			return i
		}
	}
	return -1
}

// compareEditions diffs every role and every theme across the given sides. A role declaration is
// compared by property; a theme by position, so a custom property renamed on one side is not a
// difference and a colour changed on one side is.
func compareEditions(divs []appearance.Divergence, used map[int]bool, sides ...*edition) []drift {
	var out []drift
	roleSet, themeSet := map[string]bool{}, map[string]bool{}
	for _, s := range sides {
		for r := range s.roles {
			roleSet[r] = true
		}
		for th := range s.themes {
			themeSet[th] = true
		}
	}
	for _, role := range sortedKeys(roleSet) {
		props := map[string]bool{}
		for _, s := range sides {
			for p := range s.roles[role] {
				props[p] = true
			}
		}
		for _, p := range sortedKeys(props) {
			// An edition a divergence exempts sits out the comparison; the entry counts as used
			// when that edition does differ from the ones still compared.
			var cmp []*edition
			exempted := map[int]*edition{}
			for _, s := range sides {
				if i := exempt(divs, role, p, s.name); i >= 0 {
					exempted[i] = s
				} else {
					cmp = append(cmp, s)
				}
			}
			var lacking, values []string
			distinct := map[string]bool{}
			for _, s := range cmp {
				v, ok := s.roles[role][p]
				if !ok {
					lacking = append(lacking, s.name)
					continue
				}
				distinct[v] = true
				values = append(values, s.name+"="+v)
			}
			for i, s := range exempted {
				v, ok := s.roles[role][p]
				if (ok && !distinct[v]) || (!ok && len(values) > 0) || (ok && len(values) == 0) {
					used[i] = true
				}
			}
			if len(values) > 0 && len(lacking) > 0 {
				out = append(out, drift{"role", role, p, "missing on " + strings.Join(lacking, ", ") + " (present: " + strings.Join(values, "; ") + ")"})
			} else if len(distinct) > 1 {
				out = append(out, drift{"role", role, p, "values differ: " + strings.Join(values, "; ")})
			}
		}
	}
	for _, th := range sortedKeys(themeSet) {
		n := 0
		for _, s := range sides {
			if len(s.themes[th]) > n {
				n = len(s.themes[th])
			}
		}
		for i := 0; i < n; i++ {
			var lacking, values []string
			distinct := map[string]bool{}
			name := fmt.Sprintf("colour #%d", i+1)
			for _, s := range sides {
				if i >= len(s.themes[th]) {
					lacking = append(lacking, s.name)
					continue
				}
				if s.name == "source" {
					name = s.props[th][i]
				}
				distinct[s.themes[th][i]] = true
				values = append(values, s.name+"="+s.themes[th][i])
			}
			if len(lacking) > 0 {
				out = append(out, drift{"theme", th, name, "missing on " + strings.Join(lacking, ", ")})
			} else if len(distinct) > 1 {
				out = append(out, drift{"theme", th, name, "values differ: " + strings.Join(values, "; ")})
			}
		}
	}
	return out
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestAppearanceRolesMatchSource: every role and every theme on both editions equals the source,
// declaration by declaration. A property on one side and not the other fails, naming the property,
// the role and the side that lacks it - including a property nobody thought to pin.
func TestAppearanceRolesMatchSource(t *testing.T) {
	s := loadAppearance(t)
	src, desk, ext := sourceEdition(s), desktopEdition(t, s), extensionEdition(t)
	for _, e := range []*edition{desk, ext} {
		for role := range src.roles {
			if _, ok := e.roles[role]; !ok {
				t.Errorf("%s: role %s is not emitted at all", e.name, role)
			}
		}
		for th := range src.themes {
			if _, ok := e.themes[th]; !ok {
				t.Errorf("%s: theme %s is not emitted at all", e.name, th)
			}
		}
	}
	used := map[int]bool{}
	for _, d := range compareEditions(s.Divergences, used, src, desk, ext) {
		t.Errorf("appearance drift - %s (edit internal/appearance/appearance.json and run `npm run appearance`, or name the difference in its divergences)", d)
	}
	// A divergence entry that lets a declaration differ and matches nothing is a stale exception:
	// it would silently license the next difference someone introduces there.
	for i, d := range s.Divergences {
		if d.Role != "" && !used[i] {
			t.Errorf("divergence %q (role %s, property %s, edition %s) matches no difference - remove it", d.What, d.Role, d.Property, d.Edition)
		}
		if (d.Role == "") != (d.Property == "") || (d.Role == "") != (d.Edition == "") {
			t.Errorf("divergence %q: role, property and edition go together", d.What)
		}
	}
}

// TestAppearanceNoRoleDeclaredOutsideSource: the derived path is the only one. A rule outside the
// generated regions - or anywhere in the desktop Go sources - that targets a role selector or
// declares a palette colour token would restyle the unit behind the gate's back.
func TestAppearanceNoRoleDeclaredOutsideSource(t *testing.T) {
	s := loadAppearance(t)
	desk, ext := desktopEdition(t, s), extensionEdition(t)

	check := func(file, css string, e *edition) {
		sels := map[string]string{}
		for k, sel := range e.sels {
			sels[sel] = k
		}
		palette := map[string]bool{}
		for _, props := range e.props {
			for _, p := range props {
				palette[p] = true
			}
		}
		for _, r := range parseDeclarations(css) {
			for _, sel := range splitTop(r.Selector, ',') {
				sel = strings.TrimSpace(sel)
				if k, ok := sels[sel]; ok && !strings.HasPrefix(k, "theme:") {
					t.Errorf("%s: rule %q restyles the %s role outside internal/appearance", file, sel, k)
				}
				if k, ok := sels[sel]; ok && strings.HasPrefix(k, "theme:") && sel != ":root" {
					t.Errorf("%s: rule %q restyles the %s palette outside internal/appearance", file, sel, strings.TrimPrefix(k, "theme:"))
				}
			}
			for _, d := range r.Decls {
				if palette[d[0]] {
					t.Errorf("%s: %q declares palette colour %s outside internal/appearance", file, r.Selector, d[0])
				}
			}
		}
	}

	cssFiles, err := filepath.Glob(filepath.Join("..", "extension", "src", "*.css"))
	if err != nil || len(cssFiles) == 0 {
		t.Fatalf("no extension stylesheets found: %v", err)
	}
	for _, f := range cssFiles {
		rel := "extension/src/" + filepath.Base(f)
		css := readRepoFile(t, "extension", "src", filepath.Base(f))
		if strings.Contains(css, genBegin) {
			_, css = region(t, rel, css)
		}
		check(rel, css, ext)
	}

	// Go: the CSS lives in string literals, so read declarations straight out of the source text.
	// A role selector followed by `{`, or a palette property followed by `:`, outside
	// internal/appearance is a hand-written copy.
	var patterns []*regexp.Regexp
	for k, sel := range desk.sels {
		if strings.HasPrefix(k, "theme:") && sel == ":root" {
			continue
		}
		patterns = append(patterns, regexp.MustCompile(regexp.QuoteMeta(sel)+`\s*\{`))
	}
	for _, props := range desk.props {
		for _, p := range props {
			patterns = append(patterns, regexp.MustCompile(`(^|[\s;{])`+regexp.QuoteMeta(p)+`\s*:`))
		}
	}
	for _, dir := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join("..", dir), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "appearance" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, re := range patterns {
				if loc := re.FindIndex(b); loc != nil {
					t.Errorf("%s: %q is declared by hand - the shared appearance comes from internal/appearance", filepath.ToSlash(path), strings.TrimSpace(string(b[loc[0]:loc[1]])))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// TestAppearanceComparatorDetectsDrift: the gate bites on the defect that motivated it. The
// hairline ring - `box-shadow: 0 0 0 1px rgba(0,0,0,0.06)` on the extension's plate only - must
// come back as exactly one difference, naming the property and the side that lacks it; a renamed
// selector or custom property on one side must come back as none; a colour changed on one side
// must come back as one.
func TestAppearanceComparatorDetectsDrift(t *testing.T) {
	plate := func(name, css string) *edition {
		e := newEdition(name)
		for _, r := range parseDeclarations(css) {
			kind, n, _ := strings.Cut(r.Label, ": ")
			if kind == "role" {
				e.addRole(n, r.Selector, r.Decls)
			} else {
				e.addTheme(n, r.Selector, r.Decls)
			}
		}
		return e
	}
	desk := plate("desktop", "/* role: plate */\n.ocr-box{background:#fff;padding:0.08em 0.28em}")
	ext := plate("extension", "/* role: plate */\n.ocr-plate {\n  background: #FFF;\n  padding: 0.08em  0.28em;\n  box-shadow: 0 0 0 1px rgba(0,0,0,0.06);\n}")
	got := compareEditions(nil, map[int]bool{}, desk, ext)
	if len(got) != 1 || got[0].Property != "box-shadow" || !strings.Contains(got[0].Detail, "missing on desktop") {
		t.Fatalf("ring on one side: got %v, want exactly box-shadow missing on desktop", got)
	}

	// Named, the same difference is legal - and the entry counts as used.
	divs := []appearance.Divergence{{What: "ring", Reason: "test", Role: "plate", Property: "box-shadow", Edition: "extension"}}
	used := map[int]bool{}
	if got := compareEditions(divs, used, desk, ext); len(got) != 0 || !used[0] {
		t.Fatalf("named divergence: got %v, used %v", got, used)
	}

	// Naming is outside the comparison: other selectors and other custom-property names, same values.
	a := plate("desktop", "/* role: plate */\n.ocr-box{color:#111}\n/* theme: light */\n:root{--dht-bg:#faf9f7;--dht-bar-fg:#222222}")
	b := plate("extension", "/* role: plate */\n.renamed-plate { color: #111; }\n/* theme: light */\nhtml.x { --paper: #faf9f7; --toolbar-ink: #222; }")
	if got := compareEditions(nil, map[int]bool{}, a, b); len(got) != 0 {
		t.Fatalf("renamed selector / token: got %v, want no difference", got)
	}

	// A theme colour changed on one side is a difference.
	c := plate("extension", "/* role: plate */\n.ocr-plate{color:#111}\n/* theme: light */\n:root{--bg:#faf9f7;--bar-fg:#333333}")
	if got := compareEditions(nil, map[int]bool{}, a, c); len(got) != 1 || got[0].Area != "theme" {
		t.Fatalf("palette colour on one side: got %v, want one theme difference", got)
	}
}
