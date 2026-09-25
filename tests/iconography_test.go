package tests

// Icon vocabulary guards (ICON-SET / ICON-RENDER, docs/contracts/ICON-SET.md). Every surface
// draws a meaning with the catalog's own picture: the vendored copies in assets/glyphs are the
// one local source, both editions' tables and every inline copy in the GUI and the site must
// carry their path data verbatim, and the two editions call a meaning by the same words. The map
// of every glyph is docs/GLYPH-MAP.md; the parity row is docs/PARITY.md "Vocabulary glyphs".

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"doc-html-translate/internal/htmlgen"
	"doc-html-translate/internal/i18n"
)

var pathD = regexp.MustCompile(`<path[^>]*\sd="([^"]+)"`)

// vendoredGlyphs returns id -> path data of every assets/glyphs/<id>.svg.
func vendoredGlyphs(t *testing.T) map[string]string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "assets", "glyphs", "*.svg"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no vendored glyphs under assets/glyphs (%v)", err)
	}
	out := map[string]string{}
	for _, f := range files {
		id := strings.TrimSuffix(filepath.Base(f), ".svg")
		m := pathD.FindAllStringSubmatch(readRepoFile(t, "assets", "glyphs", filepath.Base(f)), -1)
		if len(m) != 1 {
			t.Fatalf("%s: want exactly one <path d>, found %d", f, len(m))
		}
		out[id] = m[0][1]
	}
	return out
}

// extensionGlyphs parses the GLYPHS table of extension/src/glyphs.js.
func extensionGlyphs(t *testing.T) map[string]string {
	t.Helper()
	body := between(readRepoFile(t, "extension", "src", "glyphs.js"), "const GLYPHS = {", "};")
	out := map[string]string{}
	for _, m := range regexp.MustCompile(`"([a-z-]+\.[a-z-]+)":\s*"([^"]+)"`).FindAllStringSubmatch(body, -1) {
		out[m[1]] = m[2]
	}
	if len(out) == 0 {
		t.Fatal("could not parse GLYPHS in extension/src/glyphs.js")
	}
	return out
}

// The vendored files are byte copies of the catalog, pinned by SHA-256 in PROVENANCE.txt, and
// with the catalog mounted (SZA_CONTRACTS_ROOT) they are compared with the catalog itself.
func TestVendoredGlyphsMatchTheirProvenance(t *testing.T) {
	prov := readRepoFile(t, "assets", "glyphs", "PROVENANCE.txt")
	listed := map[string]string{}
	for _, m := range regexp.MustCompile(`(?m)^([0-9a-f]{64})\s+(\S+\.svg)\b`).FindAllStringSubmatch(prov, -1) {
		listed[m[2]] = m[1]
	}
	catalog := os.Getenv("SZA_CONTRACTS_ROOT")
	for id := range vendoredGlyphs(t) {
		name := id + ".svg"
		raw, err := os.ReadFile(filepath.Join("..", "assets", "glyphs", name))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		if got := hex.EncodeToString(sum[:]); listed[name] != got {
			t.Errorf("%s: sha256 %s, PROVENANCE.txt says %q - a vendored glyph is never edited, only re-copied", name, got, listed[name])
		}
		if catalog != "" {
			src, err := os.ReadFile(filepath.Join(catalog, "iconography", "glyphs", name))
			if err != nil {
				t.Errorf("%s: not in the catalog: %v", name, err)
			} else if string(src) != string(raw) {
				t.Errorf("%s drifted from the catalog - copy it again and update PROVENANCE.txt", name)
			}
		}
	}
	for name := range listed {
		if _, err := os.Stat(filepath.Join("..", "assets", "glyphs", name)); err != nil {
			t.Errorf("PROVENANCE.txt lists %s, which is not vendored", name)
		}
	}
}

// Both editions draw a meaning with the vendored picture - so with the same picture.
func TestEditionGlyphTablesAreTheVendoredDrawings(t *testing.T) {
	vendored := vendoredGlyphs(t)
	for edition, table := range map[string]map[string]string{
		"internal/htmlgen/glyphs.go": htmlgen.Glyphs,
		"extension/src/glyphs.js":    extensionGlyphs(t),
	} {
		for id, d := range table {
			want, ok := vendored[id]
			switch {
			case !ok:
				t.Errorf("%s draws %s, which is not vendored under assets/glyphs", edition, id)
			case d != want:
				t.Errorf("%s: %s path differs from assets/glyphs/%s.svg\n got  %s\n want %s", edition, id, id, d, want)
			}
		}
	}
}

// Every placeholder and every glyph() call in the extension names a meaning its table has.
func TestExtensionDrawsOnlyTabledGlyphs(t *testing.T) {
	table := extensionGlyphs(t)
	used := map[string]bool{}
	for _, f := range []string{"viewer.html", "popup.html", "options.html"} {
		for _, m := range regexp.MustCompile(`data-glyph="([^"]+)"`).FindAllStringSubmatch(readRepoFile(t, "extension", "src", f), -1) {
			used[m[1]] = true
		}
	}
	for _, m := range regexp.MustCompile(`glyph\(\s*(?:[^()]*\?\s*)?"([a-z-]+\.[a-z-]+)"(?:\s*:\s*"([a-z-]+\.[a-z-]+)")?`).FindAllStringSubmatch(readRepoFile(t, "extension", "src", "viewer.js"), -1) {
		used[m[1]] = true
		if m[2] != "" {
			used[m[2]] = true
		}
	}
	if len(used) == 0 {
		t.Fatal("found no glyph use in the extension - the parser is stale")
	}
	for id := range used {
		if _, ok := table[id]; !ok {
			t.Errorf("the extension draws %s, which extension/src/glyphs.js does not hold", id)
		}
	}
}

// The GUI and the site carry inline copies (static pages with no shared script); each copy is
// the vendored path, verbatim - also where it sits URL-encoded inside a CSS mask.
func TestInlineGlyphCopiesAreVerbatim(t *testing.T) {
	vendored := vendoredGlyphs(t)
	pages := map[string][]string{
		"cmd/doc-html-ui/ui.html": {"nav.expand", "nav.collapse", "action.swap", "content.document"},
		"extension.html":          {"nav.go-to", "nav.expand", "nav.collapse", "action.install"},
		"index.html":              {"nav.scroll-top"},
	}
	for _, code := range []string{"ar", "bn", "de", "es", "fr", "hi", "it", "pt", "ur", "zh"} {
		pages[code+"/index.html"] = []string{"nav.scroll-top"}
	}
	for page, ids := range pages {
		raw := readRepoFile(t, strings.Split(page, "/")...)
		decoded := raw
		for _, m := range regexp.MustCompile(`data:image/svg\+xml,([^"')]+)`).FindAllStringSubmatch(raw, -1) {
			if s, err := url.PathUnescape(m[1]); err == nil {
				decoded += "\n" + s
			}
		}
		for _, id := range ids {
			d := vendored[id]
			if !strings.Contains(decoded, `d="`+d+`"`) && !strings.Contains(decoded, `d='`+d+`'`) {
				t.Errorf("%s does not carry %s verbatim (assets/glyphs/%s.svg)", page, id, id)
			}
		}
	}
}

// Pre-contract glyphs that stood for another meaning's picture must not come back on the
// surfaces this ticket moved to the vocabulary.
func TestRetiredGlyphsStayRetired(t *testing.T) {
	cases := map[string][]string{
		"internal/htmlgen/navbar.go":  {"&#9664;", "&#9654;", "&#9776;", "&#9728;", "&#9681;", "&#9790;", "&#9679;", "A&minus;", "&#9636;"},
		"internal/htmlgen/htmlgen.go": {"&#9656;", "#1a0dab"},
		"extension/src/viewer.html":   {"&#9776;", "&#8595;", "Toggle contents", "A&minus;", "&#9636;"},
		"extension/src/viewer.js":     {`"▾"`, `"▸"`},
		"extension/src/popup.html":    {"&#8599;", "↗"},
		"cmd/doc-html-ui/ui.html":     {`\25B8`, `\25BE`, "&#x2913;", "&#x21C4;"},
		"extension.html":              {"▸", "▾", "← ", "⤓"},
		"index.html":                  {"'✓ '"},
	}
	codes := append([]string{}, i18n.Codes...)
	for _, c := range codes {
		dir := c
		if c == "zh" {
			dir = "zh_CN"
		}
		cases["extension/_locales/"+dir+"/messages.json"] = []string{"↓"}
	}
	// The copy button's done state is a word (action.copy's note, ICON-SET 0.15), never
	// action.confirm's check mark standing alone.
	for _, code := range []string{"ar", "bn", "de", "es", "fr", "hi", "it", "pt", "ur", "zh"} {
		cases[code+"/index.html"] = []string{"textContent='✓'"}
	}
	for page, banned := range cases {
		raw := readRepoFile(t, strings.Split(page, "/")...)
		for _, b := range banned {
			if strings.Contains(raw, b) {
				t.Errorf("%s still carries %q", page, b)
			}
		}
	}
	for _, page := range []string{"index.html", "ar/index.html", "zh/index.html"} {
		btn := regexp.MustCompile(`(?s)<button class="to-top".*?</button>`).FindString(readRepoFile(t, strings.Split(page, "/")...))
		if btn == "" || strings.Contains(btn, "↑") {
			t.Errorf("%s: the to-top button is missing or still draws ↑: %q", page, btn)
		}
	}
}

// One meaning, one name in both editions (ICON-SET rule 3): the words the desktop chrome uses
// for a control are the extension's words for the same control, language by language.
func TestEditionsNameSharedControlsAlike(t *testing.T) {
	pairs := map[string]string{
		"Table of contents": "ttToc",
		"Smaller text":      "ariaSmallerText",
		"Larger text":       "ariaLargerText",
		"Font":              "ariaFont",
		"Theme":             "ariaTheme",
		"Text layer":        "ttOcrLayer",
		"Go to page":        "ttGoToPage",
	}
	for _, lang := range i18n.Codes {
		dir := lang
		if lang == "zh" {
			dir = "zh_CN"
		}
		var msgs map[string]struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal([]byte(readRepoFile(t, "extension", "_locales", dir, "messages.json")), &msgs); err != nil {
			t.Fatalf("%s messages.json: %v", dir, err)
		}
		for goKey, extKey := range pairs {
			if got, want := msgs[extKey].Message, i18n.T(lang, goKey); got != want {
				t.Errorf("%s: extension %s = %q, desktop %q = %q", lang, extKey, got, goKey, want)
			}
		}
	}
}

// docs/GLYPH-MAP.md is this product's inventory (ICON-SET section 6, rung 2): every glyph a
// surface draws appears in it by id.
func TestGlyphMapListsEveryDrawnGlyph(t *testing.T) {
	doc := readRepoFile(t, "docs", "GLYPH-MAP.md")
	var ids []string
	for id := range vendoredGlyphs(t) {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if !strings.Contains(doc, "`"+id+"`") {
			t.Errorf("docs/GLYPH-MAP.md does not list %s", id)
		}
	}
}

// The glyphs' licence ships where the glyphs ship (ICON-EXTERNAL rule 5): beside the desktop
// binaries (installer and MSIX stage the root file) and inside the extension package, whose zip
// takes extension/src whole. The extension copy is the root file up to the end of the glyph
// section: the sections after it cover the pdftotext set, which only the Windows executables carry.
func TestGlyphNoticesShipWithTheGlyphs(t *testing.T) {
	root := readRepoFile(t, "THIRD-PARTY-NOTICES.txt")
	ext := readRepoFile(t, "extension", "src", "THIRD-PARTY-NOTICES.txt")
	if !strings.HasPrefix(root, ext) || !strings.HasSuffix(ext, "END OF TERMS AND CONDITIONS\n") {
		t.Error("extension/src/THIRD-PARTY-NOTICES.txt is not the glyph part of the root THIRD-PARTY-NOTICES.txt - copy it again")
	}
	if strings.Contains(ext, "pdftotext") {
		t.Error("extension/src/THIRD-PARTY-NOTICES.txt names pdftotext, which the extension does not bundle")
	}
	for _, want := range []string{"Apache License", "Version 2.0, January 2004", "END OF TERMS AND CONDITIONS"} {
		if !strings.Contains(root, want) {
			t.Errorf("THIRD-PARTY-NOTICES.txt lacks %q", want)
		}
	}
	for id := range vendoredGlyphs(t) {
		if !strings.Contains(root, id) {
			t.Errorf("THIRD-PARTY-NOTICES.txt does not name the vendored glyph %s", id)
		}
	}
	for _, f := range [][]string{{"scripts", "build-installer.ps1"}, {"installer", "doc-html-translate.iss"}, {"msix", "build-msix.ps1"}} {
		if !strings.Contains(readRepoFile(t, f...), "THIRD-PARTY-NOTICES.txt") {
			t.Errorf("%s does not stage THIRD-PARTY-NOTICES.txt", filepath.Join(f...))
		}
	}
}
