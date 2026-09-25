package main

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"doc-html-translate/internal/i18n"
)

// Static gates for the desktop-app-ux contracts (APP-STYLE, APP-BEHAVIOUR) - what can be pinned
// without a running window. See DEV/plan ticket 23 and docs/contracts/.

// uiStyle is the page's own stylesheet.
func uiStyle(t *testing.T) string {
	t.Helper()
	start, end := strings.Index(uiHTML, "<style>"), strings.Index(uiHTML, "</style>")
	if start < 0 || end < start {
		t.Fatal("ui.html has no <style> block")
	}
	return uiHTML[start:end]
}

// APP-STYLE rules 3-4: one palette table, every role a light/dark pair, named by the vocabulary.
// A role declared with one value would stop changing when the theme does.
func TestPaletteDeclaresEveryRoleForBothThemes(t *testing.T) {
	style := uiStyle(t)
	roles := []string{
		"surface-window", "surface-raised", "surface-sunken",
		"control", "control-hover", "control-pressed", "border",
		"text-primary", "text-muted", "text-disabled",
		"accent", "accent-ink", "link", "info",
	}
	for _, role := range roles {
		re := regexp.MustCompile(`--` + regexp.QuoteMeta(role) + `:\s*light-dark\(\s*#[0-9a-fA-F]{6}\s*,\s*#[0-9a-fA-F]{6}\s*\);`)
		if n := len(re.FindAllString(style, -1)); n != 1 {
			t.Errorf("role --%s: %d light-dark() pair declarations, want exactly 1", role, n)
		}
	}
	// Every themed custom property in :root is a pair; a bare value would be one theme only.
	root := style[strings.Index(style, ":root {"):]
	root = root[:strings.Index(root, "}")]
	for _, line := range strings.Split(root, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "--") && !strings.Contains(line, "light-dark(") {
			t.Errorf("palette entry is not a light/dark pair: %s", line)
		}
	}
	// A reference to a property nobody declares renders as nothing (the old var(--fg) bug).
	declared := map[string]bool{}
	for _, m := range regexp.MustCompile(`(--[a-z-]+)\s*:`).FindAllStringSubmatch(style, -1) {
		declared[m[1]] = true
	}
	for _, m := range regexp.MustCompile(`var\((--[a-z-]+)\)`).FindAllStringSubmatch(uiHTML, -1) {
		if !declared[m[1]] {
			t.Errorf("var(%s) is used but never declared", m[1])
		}
	}
}

// APP-STYLE rule 2: system, light and dark; system follows the OS, the other two pin it.
func TestThemeOffersSystemLightAndDark(t *testing.T) {
	style := uiStyle(t)
	for _, want := range []string{
		"color-scheme: light dark;",
		`:root[data-theme="light"] { color-scheme: light; }`,
		`:root[data-theme="dark"]  { color-scheme: dark; }`,
	} {
		if !strings.Contains(style, want) {
			t.Errorf("stylesheet lacks %q", want)
		}
	}
	for _, v := range []string{"system", "light", "dark"} {
		if !strings.Contains(uiHTML, `<option value="`+v+`"`) {
			t.Errorf("theme switch has no %q option", v)
		}
	}
}

// APP-STYLE rule 5: the console surfaces are out of theme and carry their own text colour.
func TestConsoleIsDeclaredOutOfTheme(t *testing.T) {
	style := uiStyle(t)
	block := style[strings.Index(style, ".console {"):]
	block = block[:strings.Index(block, "}")]
	for _, want := range []string{"background: var(--console-bg)", "color: var(--console-text)"} {
		if !strings.Contains(block, want) {
			t.Errorf(".console lacks %q", want)
		}
	}
	for _, id := range []string{`id="logArea"`, `id="cmdLine"`} {
		tag := regexp.MustCompile(`<div class="([^"]*)" ` + id)
		m := tag.FindStringSubmatch(uiHTML)
		if m == nil || !slices.Contains(strings.Fields(m[1]), "console") {
			t.Errorf("%s is not a .console surface", id)
		}
	}
}

// APP-BEHAVIOUR rule 8: direction is declared once per language and gated. The GUI's list must
// be exactly the languages internal/i18n calls right-to-left.
func TestGUIRightToLeftListMatchesI18n(t *testing.T) {
	m := regexp.MustCompile(`const RTL_LANGS = \[([^\]]*)\];`).FindStringSubmatch(uiI18nJS)
	if m == nil {
		t.Fatal("i18n.js does not declare RTL_LANGS")
	}
	var gui []string
	for _, q := range regexp.MustCompile(`"([a-z]{2})"`).FindAllStringSubmatch(m[1], -1) {
		gui = append(gui, q[1])
	}
	var want []string
	for _, c := range i18n.Codes {
		if i18n.IsRTL(c) {
			want = append(want, c)
		}
	}
	slices.Sort(gui)
	slices.Sort(want)
	if !slices.Equal(gui, want) {
		t.Errorf("i18n.js RTL_LANGS = %v, internal/i18n.IsRTL = %v", gui, want)
	}
}

// APP-BEHAVIOUR rule 9: a control whose only label is a glyph carries a localized accessible
// name, and so does every select (their visible text is the chosen value, not what they are for).
func TestGlyphOnlyControlsHaveAccessibleNames(t *testing.T) {
	button := regexp.MustCompile(`(?s)<button([^>]*)>(.*?)</button>`)
	tags := regexp.MustCompile(`<[^>]*>`)
	letter := regexp.MustCompile(`\pL`)
	found := 0
	for _, m := range button.FindAllStringSubmatch(uiHTML, -1) {
		text := tags.ReplaceAllString(m[2], "")
		if letter.MatchString(text) || strings.Contains(m[1], "data-i18n=") {
			continue
		}
		found++
		if !strings.Contains(m[1], "data-i18n-aria=") {
			t.Errorf("glyph-only button %q has no data-i18n-aria", strings.TrimSpace(m[0]))
		}
	}
	if found == 0 {
		t.Fatal("no glyph-only button found - the swap button should be one; the scan is wrong")
	}
	for _, m := range regexp.MustCompile(`<select([^>]*)>`).FindAllStringSubmatch(uiHTML, -1) {
		attrs := m[1]
		id := regexp.MustCompile(`id="([^"]+)"`).FindStringSubmatch(attrs)
		if id == nil {
			t.Errorf("select without an id: %s", m[0])
			continue
		}
		if strings.Contains(attrs, "data-i18n-aria=") || strings.Contains(uiHTML, `for="`+id[1]+`"`) {
			continue
		}
		// A select inside a <label> or right after a labelled field row is named by that label.
		if regexp.MustCompile(`<label[^>]*>[^<]*</label>\s*<select[^>]*id="` + id[1] + `"`).MatchString(uiHTML) {
			continue
		}
		if regexp.MustCompile(`<label[^>]*data-i18n="[^"]+"[^>]*>[^<]*</label><select id="` + id[1] + `"`).MatchString(uiHTML) {
			continue
		}
		t.Errorf("select #%s has no accessible name", id[1])
	}
	// The main action must be reachable from the keyboard: the drop zone is a real button.
	if !strings.Contains(uiHTML, `<button type="button" class="drop" id="dropZone"`) {
		t.Error("the drop zone is not a <button>")
	}
}

// APP-BEHAVIOUR rule 1: every question goes through the page's one modal dialog. Native alert
// and confirm open with no owner and no safe default the page controls.
func TestNoNativeAlertOrConfirm(t *testing.T) {
	script := uiHTML[strings.Index(uiHTML, "<script>"):]
	if m := regexp.MustCompile(`(^|[^.\w])(alert|confirm|prompt)\(`).FindString(script); m != "" {
		t.Errorf("ui.html calls a native dialog: %q", m)
	}
}
