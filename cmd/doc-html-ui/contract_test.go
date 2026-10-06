package main

import (
	"image/color"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/iconart"
)

// Static gates for the desktop-app-ux contracts (APP-STYLE, APP-BEHAVIOUR) - what can be pinned
// without a running window. See DEV/plan ticket 23 and docs/contracts/.
// Ticket 93 adds the WINDOWS-UI 0.1 / APP-SETTINGS 0.2 gates below (search "WINDOWS-UI").

// uiStyle is the page's own stylesheet.
func uiStyle(t *testing.T) string {
	t.Helper()
	start, end := strings.Index(uiHTML, "<style>"), strings.Index(uiHTML, "</style>")
	if start < 0 || end < start {
		t.Fatal("ui.html has no <style> block")
	}
	return uiHTML[start:end]
}

// ICON-RENDER rule 5 (ticket 90): compact Windows desktop controls clear the 28px
// target-size floor, the chrome mirrors for RTL through logical properties, the log does not
// flood screen readers (the stage line announces instead), and stage text is a status region.
// docs/PARITY.md "Reader chrome accessibility floor" holds the extension to the same list.
func TestUIKeepsTheAccessibilityFloor(t *testing.T) {
	for _, want := range []string{
		".inline-btn { min-height: 28px; }",
		"max-width: 150px; min-height: 28px", // the theme/language selects
		"min-width: 0; min-height: 28px",     // the output-path button
		".switches { margin-inline-start: auto",
		`id="logArea" role="log" aria-live="off"`,
		`id="progressStage" role="status"`,
		`id="readiness" role="status" aria-live="polite"`,
	} {
		if !strings.Contains(uiHTML, want) {
			t.Errorf("ui.html lost %q - the GUI accessibility floor regressed", want)
		}
	}
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

// paletteThemes reads the palette table into one role -> colour map per theme, light then dark.
// Only hex light-dark() pairs are read; a translucent role (--accent-soft) has no fixed contrast.
func paletteThemes(t *testing.T) [2]map[string]color.RGBA {
	t.Helper()
	hex := func(s string) color.RGBA {
		v, err := strconv.ParseUint(s[1:], 16, 32)
		if err != nil {
			t.Fatalf("bad colour %q: %v", s, err)
		}
		return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xFF}
	}
	themes := [2]map[string]color.RGBA{{}, {}}
	for _, m := range regexp.MustCompile(`--([a-z-]+):\s*light-dark\((#[0-9a-fA-F]{6}),\s*(#[0-9a-fA-F]{6})\)`).FindAllStringSubmatch(uiStyle(t), -1) {
		themes[0][m[1]], themes[1][m[1]] = hex(m[2]), hex(m[3])
	}
	return themes
}

// APP-STYLE section 8 (contrast), measured for this product: every text role clears WCAG 2.1 AA
// on each surface the stylesheet draws it on, in both themes. --text-muted carries the 11-12px
// hints and field labels, so it gets a stricter floor - at 3.4:1 in dark it read as "hard to read".
func TestPaletteTextMeetsWCAGAA(t *testing.T) {
	themes := paletteThemes(t)
	surfaces := []string{"surface-window", "surface-raised", "surface-sunken", "control"}
	checks := []struct {
		fg    string
		on    []string
		floor float64
	}{
		{"text-primary", surfaces, 4.5},
		{"text-muted", surfaces, 6},
		{"accent", []string{"surface-window", "surface-sunken"}, 4.5},
		{"link", []string{"surface-window", "surface-raised", "surface-sunken"}, 4.5},
		{"success", []string{"surface-window", "surface-sunken", "control"}, 4.5},
		{"danger", []string{"surface-window", "surface-sunken"}, 4.5},
		{"warning", []string{"surface-window", "surface-sunken"}, 4.5},
		{"accent-ink", []string{"accent", "accent-hover"}, 4.5},
		{"danger-ink", []string{"danger"}, 4.5},
	}
	for i, name := range []string{"light", "dark"} {
		for _, c := range checks {
			for _, bg := range c.on {
				fg, okF := themes[i][c.fg]
				back, okB := themes[i][bg]
				if !okF || !okB {
					t.Fatalf("%s: --%s or --%s is not a hex light-dark() pair", name, c.fg, bg)
				}
				if r := iconart.Contrast(fg, back); r < c.floor {
					t.Errorf("%s: --%s on --%s is %.2f:1, want >= %.1f", name, c.fg, bg, r, c.floor)
				}
			}
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
// name. Fields and selects are held by TestEveryFieldIsNamedAndEveryLabelNamesOne below.
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
	// The main action must be reachable from the keyboard: the drop zone is a real button.
	if !strings.Contains(uiHTML, `<button type="button" class="drop" id="dropZone"`) {
		t.Error("the drop zone is not a <button>")
	}
}

// APP-BEHAVIOUR rule 9 / APP-SETTINGS rule 10: every visible field and select is named - by its
// own localized aria-label, by a <label for> pointing at its id, or by a <label> wrapping it -
// and every <label> names something. A caption that merely sits beside a field names nothing:
// clicking it focuses nothing and a screen reader announces the field unnamed.
func TestEveryFieldIsNamedAndEveryLabelNamesOne(t *testing.T) {
	body := uiHTML[strings.Index(uiHTML, "<body>"):strings.Index(uiHTML, "<script>")]
	insideLabel := func(pos int) bool {
		return strings.LastIndex(body[:pos], "<label") > strings.LastIndex(body[:pos], "</label>")
	}
	idOf := regexp.MustCompile(`\bid="([^"]+)"`)
	controls := 0
	for _, loc := range regexp.MustCompile(`<(select|input)\b([^>]*)>`).FindAllStringSubmatchIndex(body, -1) {
		tag, attrs := body[loc[0]:loc[1]], body[loc[4]:loc[5]]
		if strings.Contains(attrs, `type="hidden"`) {
			continue
		}
		controls++
		if strings.Contains(attrs, "data-i18n-aria=") || insideLabel(loc[0]) {
			continue
		}
		id := idOf.FindStringSubmatch(attrs)
		if id != nil && regexp.MustCompile(`<label\b[^>]*\bfor="`+regexp.QuoteMeta(id[1])+`"`).MatchString(body) {
			continue
		}
		t.Errorf("%s has no accessible name: no aria-label, no <label for>, no wrapping <label>", tag)
	}
	if controls < 15 {
		t.Fatalf("found %d fields and selects; the scan is wrong", controls)
	}
	for _, m := range regexp.MustCompile(`(?s)<label\b([^>]*)>(.*?)</label>`).FindAllStringSubmatch(body, -1) {
		if strings.Contains(m[2], "<input") || strings.Contains(m[2], "<select") {
			continue
		}
		f := regexp.MustCompile(`\bfor="([^"]+)"`).FindStringSubmatch(m[1])
		if f == nil {
			t.Errorf("label names nothing (no for=, wraps no control): %s", strings.TrimSpace(m[0]))
			continue
		}
		if !strings.Contains(body, `id="`+f[1]+`"`) {
			t.Errorf("label for=%q points at no element", f[1])
		}
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

// Rebuilding replaces an existing result and must never run through the default focus.
func TestRebuildConfirmationsChooseCancel(t *testing.T) {
	calls := regexp.MustCompile(`confirmDialog\([^\n]*'btnRebuild'[^\n]*\)`).FindAllString(uiHTML, -1)
	if len(calls) != 2 {
		t.Fatalf("got %d rebuild prompts, want single and queue", len(calls))
	}
	for _, call := range calls {
		if !strings.Contains(call, "'danger', true)") {
			t.Errorf("unsafe default: %s", call)
		}
	}
}

func TestDesktopTargetsAndMotionFollowSystem(t *testing.T) {
	style := uiStyle(t)
	for _, want := range []string{"@media (any-pointer: coarse)", "min-width: 44px !important; min-height: 44px !important", "@media (prefers-reduced-motion: reduce)", "animation: none !important; transition: none !important; scroll-behavior: auto !important;"} {
		if !strings.Contains(style, want) {
			t.Errorf("missing %q", want)
		}
	}
}

// ── WINDOWS-UI 0.1 / APP-SETTINGS 0.2 gates (ticket 93) ─────────────────────────────────────

// WINDOWS-UI section 4 (Boolean) + APP-SETTINGS rules 3 and section 8: a boolean setting draws
// its checkbox at the reading start inside the label (caption immediately beside it), and every
// audited setting carries a muted description below the caption at the caption's inset (.chk-hint).
func TestBooleanRowsFollowTheCheckboxConvention(t *testing.T) {
	// Every boolean label starts with the input - the control at the reading start. A .chk label
	// without a checkbox (the queue's "Parallel:" caption, naming a select) is not a boolean row.
	labels := regexp.MustCompile(`(?s)<label class="chk[^>]*>(.*?)</label>`)
	n := 0
	for _, m := range labels.FindAllStringSubmatch(uiHTML, -1) {
		if !strings.Contains(m[1], `<input type="checkbox"`) {
			continue
		}
		n++
		if !strings.HasPrefix(strings.TrimSpace(m[1]), `<input type="checkbox"`) {
			t.Errorf("boolean label does not start with its checkbox: %.80s", m[0])
		}
	}
	if n < 8 {
		t.Fatalf("found %d boolean labels; the scan is wrong", n)
	}
	// Every settings boolean has its description below the caption at the caption's inset.
	for _, id := range []string{"chkOCR", "chkSingle", "chkNoOpen", "chkForce", "chkVerbose", "chkAutoUpdates", "chkShellEntries", "chkDefaultHandler"} {
		pos := strings.Index(uiHTML, `id="`+id+`"`)
		if pos < 0 {
			t.Errorf("boolean %s is gone from the markup", id)
			continue
		}
		window := uiHTML[pos:min(pos+900, len(uiHTML))]
		if !strings.Contains(window, "chk-hint") {
			t.Errorf("boolean %s has no description below its caption at the caption inset", id)
		}
	}
}

// WINDOWS-UI section 4 (Numeric): readable digits and readable native spin controls - the whole
// editor scales (16px digits, 110px field, 32px minimum height), never just the box.
func TestNumericEditorsScaleDigitsAndSpinners(t *testing.T) {
	style := uiStyle(t)
	for _, want := range []string{
		`input[type="number"] { font-size: 16px; min-height: 32px; }`,
		`.field.num > input { width: 110px;`,
	} {
		if !strings.Contains(style, want) {
			t.Errorf("stylesheet lacks %q - the numeric editor lost its readable scale", want)
		}
	}
}

// WINDOWS-UI section 4, last paragraph: the wheel never edits a value the reader did not focus;
// the pane keeps scrolling after the guard eats the value change.
func TestWheelNeverEditsAnUnfocusedNumber(t *testing.T) {
	script := uiHTML[strings.Index(uiHTML, "WINDOWS-UI section 4: the wheel"):]
	if script == "" {
		t.Fatal("the wheel guard comment is gone - the gate lost its anchor")
	}
	block := script[:strings.Index(script, "});")+3]
	for _, want := range []string{
		`ev.target.closest('input[type="number"]')`,
		"ev.preventDefault();",
		"pane.scrollTop += ev.deltaY;",
		"{ passive: false }",
	} {
		if !strings.Contains(block, want) {
			t.Errorf("wheel guard lost %q", want)
		}
	}
}

// WINDOWS-UI sections 3.2-3.4: group expansion and the pane viewport are remembered by stable id,
// restoring fires no user-edit save, and closing a group hands focus to its header.
func TestGroupsAndPaneContextAreRemembered(t *testing.T) {
	for _, want := range []string{
		"openGroups:",                            // collected by stable id
		"scroll:         paneScroll,",            // the pane viewport rides the settings blob
		"d.tagName === 'DETAILS') d.open = true", // restore skips unknown ids, sets values only
		"head.focus();",                          // focus handoff to the header before hiding focused children
		"restorePaneScroll();",                   // the remembered anchor is applied after the window geometry
		// Section 3.4: the restore's own toggle and scroll events arrive after settingsLoaded is
		// set; the guard holds until they have run, and both listeners honour it.
		"let restoring = true;",
		"requestAnimationFrame(() => requestAnimationFrame(() => { restoring = false; }));",
		"if (!restoring) saveSettings();",
		"addEventListener('scroll', () => {\n    if (restoring) return;",
		// A save before the OCR catalog answers carries the remembered pick, not the empty select.
		"...ocrLangSetting(),",
		"if (ocrCatalog.length) return {ocrLangChoice: sel.value};",
		"return {ocrLangChoice: sel.dataset.want || ''};",
	} {
		if !strings.Contains(uiHTML, want) {
			t.Errorf("ui.html lost %q - remembered UI context regressed", want)
		}
	}
	// Restoring calls syncEngineUI; only the engine radios' change listener may open the Ollama
	// group, or every launch would undo the collapse the reader left.
	sync := uiHTML[strings.Index(uiHTML, "function syncEngineUI() {"):]
	sync = sync[:strings.Index(sync, "\n}")]
	if strings.Contains(sync, ".open = true") {
		t.Error("syncEngineUI opens a group - restoring would override a remembered collapse")
	}
	if strings.Contains(uiHTML, "ocrLangChoice:  el('ocrLang').value") {
		t.Error("collectSettings reads the OCR select directly - a save before the catalog loads erases the pick")
	}
	// The group ids the context is keyed by are stable and present.
	for _, id := range []string{`id="advancedDetails"`, `id="ollamaDetails"`, `id="ocrDetails"`, `id="integrationSection"`, `id="aboutSection"`, `id="togglesRow"`, `id="outputRow"`, `id="langsRow"`} {
		if !strings.Contains(uiHTML, id) {
			t.Errorf("pane anchor %s is gone - the scroll key is not stable", id)
		}
	}
}

// WINDOWS-UI section 4 (Text/password): the secret stays masked until the ordinary checkbox
// beside it says otherwise.
func TestSecretRevealsBehindAnExplicitCheckbox(t *testing.T) {
	if !strings.Contains(uiHTML, `type="password" id="googleKey"`) {
		t.Error("the Google key field is no longer masked by default")
	}
	if !strings.Contains(uiHTML, `id="chkRevealKey"`) || !strings.Contains(uiHTML, "function revealKey(on)") {
		t.Error("the Google key field has no reveal checkbox")
	}
}

// WINDOWS-UI section 4 (Units): units sit immediately next to the value; the range and sentinel
// meaning stay in the hints.
func TestUnitsSitNextToTheValues(t *testing.T) {
	for _, want := range []string{
		`<span class="unit" data-i18n="unitChars">`,
		`<span class="unit" data-i18n="unitUsd">`,
		`<span class="unit" data-i18n="unitTokens">`,
	} {
		if !strings.Contains(uiHTML, want) {
			t.Errorf("missing %q - a unit drifted away from its value", want)
		}
	}
	// The $ sits beside the Max cost field, so the caption carries no second one.
	captions := regexp.MustCompile(`(?m)^ {8}lblMaxCost: "(.*)",\r?$`).FindAllStringSubmatch(uiI18nJS, -1)
	if len(captions) != len(i18n.Codes) {
		t.Fatalf("lblMaxCost declared %d times, want %d", len(captions), len(i18n.Codes))
	}
	for _, m := range captions {
		if strings.Contains(m[1], "$") {
			t.Errorf("lblMaxCost %q repeats the unit the field already shows", m[1])
		}
	}
}

// APP-SETTINGS rule 12: About carries the version, the licence, the support bundle and the
// privacy policy link.
func TestAboutCarriesLicenceAndPrivacy(t *testing.T) {
	about := uiHTML[strings.Index(uiHTML, `id="aboutSection"`):]
	if about == "" {
		t.Fatal("the About section is gone")
	}
	for _, want := range []string{
		`data-i18n="aboutLicence"`,
		`data-i18n="aboutPrivacy"`,
		"privacy.html",
		`id="btnSendLogs"`,
		`id="aboutVer"`,
	} {
		if !strings.Contains(about, want) {
			t.Errorf("About lacks %q", want)
		}
	}
}

// APP-BEHAVIOUR rule 8 / WINDOWS-UI section 2.5: structural alignment mirrors in RTL, so the
// stylesheet carries no physical left/right rules.
func TestStyleCarriesNoPhysicalDirectionRules(t *testing.T) {
	style := uiStyle(t)
	for _, bad := range []string{"text-align: left", "text-align: right", "margin-left:", "margin-right:", "padding-left:", "padding-right:"} {
		if strings.Contains(style, bad) {
			t.Errorf("physical direction rule %q survives in the stylesheet; RTL mirrors nothing with it", bad)
		}
	}
}
