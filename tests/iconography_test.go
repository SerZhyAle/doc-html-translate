package tests

// Icon vocabulary guards (ICON-SET / ICON-RENDER, docs/contracts/ICON-SET.md). Every surface
// draws a meaning with the catalog's own picture: the vendored copies in assets/glyphs are the
// one local source, both editions' tables and every inline copy in the GUI and the site must
// carry their path data verbatim, and the two editions call a meaning by the same words. The map
// of every glyph is docs/GLYPH-MAP.md; the parity row is docs/PARITY.md "Vocabulary glyphs".

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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
	if catalog != "" {
		catalogVersion(t, catalog)
	}
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
	for _, f := range []string{"viewer.html", "popup.html", "options.html", "ocr.html"} {
		for _, m := range regexp.MustCompile(`data-glyph="([^"]+)"`).FindAllStringSubmatch(readRepoFile(t, "extension", "src", f), -1) {
			used[m[1]] = true
		}
	}
	// The pages built in script (ticket 119, slice X1) name a meaning as a string literal - a glyph() argument, a
	// commandButton() or confirmDialog() option, the viewer's answerButton() glyphId - so every quoted vocabulary id in these modules is a use.
	quoted := regexp.MustCompile(`"((?:action|app|nav|view|content|feature|system|media|status)\.[a-z-]+)"`)
	for _, f := range []string{"options.js", "options-about.js", "options-data.js", "ocr-lang-ui.js", "ui-help.js", "ui-dialog.js", "popup.js", "viewer.js", "missing-image.js"} {
		for _, m := range quoted.FindAllStringSubmatch(readRepoFile(t, "extension", "src", f), -1) {
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
// the vendored path, verbatim - also where it sits URL-encoded inside a CSS mask, and in the
// GUI's BTN_GLYPHS button-glyph table.
func TestInlineGlyphCopiesAreVerbatim(t *testing.T) {
	vendored := vendoredGlyphs(t)
	pages := map[string][]string{
		// The launcher's group markers are state triangles drawn by its own component (COLLAPSIBLE-GROUP 3.2), not
		// the nav.expand / nav.collapse action chevrons, so those two are no longer carried; the rail and the new
		// meanings are (ticket 119).
		"cmd/doc-html-ui/ui.html": {"action.swap", "content.document",
			"nav.close", "action.cancel", "action.confirm", "action.delete", "action.clear-all", "action.remove",
			"action.add", "action.choose-folder", "action.download", "action.email", "action.copy",
			"action.translate", "nav.scroll-bottom", "nav.open-external", "action.convert", "action.save", "app.help",
			"action.extract-text", "action.create-shortcut", "action.text-settings", "action.adjust", "app.info",
			"nav.contents", "view.text-layer", "action.retry", "app.update", "nav.go-to"},
		// the disclosure markers of the site are state triangles drawn by the page layer (COLLAPSIBLE-GROUP 3.2), no
		// longer the nav.expand / nav.collapse action chevrons; site.css carries the external-link marker (BUTTONS 2.3)
		"extension.html":  {"nav.go-to", "action.install"},
		"assets/site.css": {"nav.open-external"},
		"index.html":      {"nav.scroll-top"},
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
	// action.retry and app.update were the two provisional meanings the launcher drew under ICON-SET section 15; the
	// catalog holds their files since ICON-SET 0.36, so they are vendored and checked with the rest above.
}

// Pre-contract glyphs that stood for another meaning's picture must not come back on the
// surfaces this ticket moved to the vocabulary.
func TestRetiredGlyphsStayRetired(t *testing.T) {
	cases := map[string][]string{
		"internal/htmlgen/navbar.go":  {"&#9664;", "&#9654;", "&#9776;", "&#9728;", "&#9681;", "&#9790;", "&#9679;", "A&minus;", "&#9636;"},
		"internal/htmlgen/htmlgen.go": {"&#9656;", "#1a0dab"},
		// The contents panel's close was a typed cross standing for nav.close; it is the word now.
		"internal/htmlgen/singlecontents.go": {"&#215;", "×"},
		"extension/src/viewer.html":          {"&#9776;", "&#8595;", "Toggle contents", "A&minus;", "&#9636;"},
		"extension/src/viewer.js":            {`"▾"`, `"▸"`},
		// A typed arrow is never a glyph (ICON-RENDER rule 7), in a control or in the note's prose.
		"extension/src/popup.html": {"&#8599;", "↗", "&rarr;", "→"},
		"cmd/doc-html-ui/ui.html":  {`\25B8`, `\25BE`, "&#x2913;", "&#x21C4;"},
		"extension.html":           {"▸", "▾", "← ", "⤓"},
		"index.html":               {"'✓ '"},
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
		"Search":            "vSearch", // action.search's word, ticket 119 slice R1
		"Continue reading":  "vResumeContinue",
		// Ticket 59's reading-comfort controls, shared by both editions.
		"Night mode":      "ariaNightMode",
		"Reset text size": "ariaResetTextSize",
		"Line spacing":    "ariaLineSpacing",
		"Column width":    "ariaColumnWidth",
		"Default":         "optDefault",
		"Narrow":          "optWidthNarrow",
		"Normal":          "optWidthNormal",
		"Wide":            "optWidthWide",
		"Full width":      "optWidthFull",
		// app.theme's choices, named by the record's note (Ночная / Нічна, not Ночь / Ніч).
		"Light": "themeLight",
		"Sepia": "themeSepia",
		"Dark":  "themeDark",
		"Night": "themeNight",
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

// catalogRecord is the part of a vocabulary.jsonl line the name checks read.
type catalogRecord struct {
	ID   string            `json:"id"`
	Name map[string]string `json:"name"`
}

var catalogVersionLine = regexp.MustCompile(`^ICON-SET (\d+)\.(\d+)$`)

// catalogVersion reads the mounted catalog's vocabulary.version and logs it. A MAJOR above 0 is a
// breaking change this product has not been adapted to (VERSIONING section 4.3), so it fails here
// rather than compare against a shape the checks do not know.
func catalogVersion(t *testing.T, root string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "iconography", "vocabulary.version"))
	if err != nil {
		t.Fatalf("vocabulary.version: %v", err)
	}
	line := strings.TrimSpace(string(raw))
	m := catalogVersionLine.FindStringSubmatch(line)
	if m == nil {
		t.Fatalf("vocabulary.version is %q, want \"ICON-SET <major>.<minor>\"", line)
	}
	t.Logf("catalog vocabulary: %s", line)
	if major, _ := strconv.Atoi(m[1]); major > 0 {
		t.Fatalf("the catalog vocabulary is at MAJOR %d (%s): adapt the product before consuming it (VERSIONING section 4.3)", major, line)
	}
}

func catalogVocabulary(t *testing.T, root string) []catalogRecord {
	t.Helper()
	f, err := os.Open(filepath.Join(root, "iconography", "vocabulary.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var records []catalogRecord
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var r catalogRecord
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("vocabulary.jsonl: %v", err)
		}
		records = append(records, r)
	}
	if err := sc.Err(); err != nil || len(records) == 0 {
		t.Fatalf("vocabulary.jsonl: %d records, error %v", len(records), err)
	}
	return records
}

// guiLanguageBlocks splits the launcher dictionary into language -> the text of its object.
func guiLanguageBlocks(t *testing.T) map[string]string {
	t.Helper()
	js := readRepoFile(t, "cmd", "doc-html-ui", "i18n.js")
	start := strings.Index(js, "const I18N = {")
	if start < 0 {
		t.Fatal("i18n.js does not declare const I18N")
	}
	body := js[start:]
	locs := regexp.MustCompile(`(?m)^ {4}([a-z]{2}): \{`).FindAllStringSubmatchIndex(body, -1)
	out := map[string]string{}
	for i, loc := range locs {
		end := len(body)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		out[body[loc[2]:loc[3]]] = body[loc[1]:end]
	}
	return out
}

func guiString(t *testing.T, block, key string) string {
	t.Helper()
	m := regexp.MustCompile(`(?m)^ {8}` + regexp.QuoteMeta(key) + `: "((?:[^"\\]|\\.)*)",`).FindStringSubmatch(block)
	if m == nil {
		return ""
	}
	var s string
	if err := json.Unmarshal([]byte(`"`+m[1]+`"`), &s); err != nil {
		t.Fatalf("i18n.js %s: %v", key, err)
	}
	return s
}

func extensionMessage(t *testing.T, lang, key string) string {
	t.Helper()
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
	return msgs[key].Message
}

// ICON-SET rule 3: a control is named with its meaning's word, never with another meaning's. With
// the catalog mounted, the words of the controls that were once wrong ("Отменить" is Undo, "Clear"
// is the text field's clear, "Повторить" is Redo, Stop is the media stop) are read from the product
// files in all thirteen languages and none may be the name of a different record. A string equal to
// the drawn meaning's own name passes even when the catalog gives that word to a second record too
// (uk "Скасувати" names both Cancel and Undo): that collision is the catalog's to resolve.
// "OK" (dlgOK) is deliberately not listed: it is a permanent variant, ticket 119 decision D6.
func TestIconographyNamesAreNotAnotherMeaningsName(t *testing.T) {
	root := os.Getenv("SZA_CONTRACTS_ROOT")
	if root == "" {
		t.Skip("SZA_CONTRACTS_ROOT does not name the catalog")
	}
	catalogVersion(t, root)
	records := catalogVocabulary(t, root)
	byID := map[string]catalogRecord{}
	for _, r := range records {
		byID[r.ID] = r
	}
	gui := guiLanguageBlocks(t)
	controls := []struct{ surface, key, drawn string }{
		{"GUI", "btnCancel", "action.cancel"},
		{"GUI", "queueClear", "action.clear-all"},
		{"GUI", "dlgClose", "nav.close"},
		{"extension", "dlgCancel", "action.cancel"},
		{"extension", "dlgClose", "nav.close"},
		{"extension", "ocrRetry", "action.retry"},
		{"extension", "pageOcrStop", "action.cancel"},
	}
	checked := 0
	for _, c := range controls {
		own, ok := byID[c.drawn]
		if !ok {
			t.Errorf("%s %s draws %s, which the catalog does not hold", c.surface, c.key, c.drawn)
			continue
		}
		for _, lang := range i18n.Codes {
			var got string
			if c.surface == "GUI" {
				got = guiString(t, gui[lang], c.key)
			} else {
				got = extensionMessage(t, lang, c.key)
			}
			if got == "" {
				t.Errorf("%s %s: no %s string", c.surface, lang, c.key)
				continue
			}
			checked++
			if strings.EqualFold(own.Name[lang], got) {
				continue
			}
			for _, r := range records {
				if r.ID != c.drawn && strings.EqualFold(r.Name[lang], got) {
					t.Errorf("%s %s %q (%s, draws %s) is the name of %s: another meaning's word (ICON-SET rule 3)", c.surface, lang, got, c.key, c.drawn, r.ID)
				}
			}
		}
	}
	t.Logf("checked %d (surface, language) strings of %d controls against %d catalog records", checked, len(controls), len(records))
}
