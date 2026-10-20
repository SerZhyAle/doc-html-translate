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

// UI-SIZING 1.1 (ticket 119): every covered control takes its size from the one token resolver, so the
// ordinary pointer target is 32 and the ordinary touch target 48 logical pixels and compact scales both
// down to the declared 28 / 42 floor; the chrome mirrors for RTL through logical properties, the log
// does not flood screen readers (the stage line announces instead), and stage text is a status region.
// docs/PARITY.md "Reader chrome accessibility floor" holds the extension to its own list.
func TestUIKeepsTheAccessibilityFloor(t *testing.T) {
	for _, want := range []string{
		"--s: clamp(0.875, var(--ui), 2);", // the declared range: compact 0.875 .. huge 2, never zero
		"--tgt: calc(32px * var(--s));",
		"--tgt: calc(48px * var(--s));",
		"min-height: var(--tgt);", // buttons, rail rows, selects, fields
		".help-btn { padding: 0; width: var(--tgt); height: var(--tgt); }",
		".head-help { margin-inline-start: auto; }",
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

// BUTTONS 2.0 section 3 rules 5 and 6, measured for this product: a dialog answer's face is the plain
// control face - no green or red tint on a non-destructive answer - and the meaning rides the glyph
// only (the affirmative in state.ok, a caution in state.warning, Cancel in the ordinary ink). The
// glyph clears 3:1 and the caption 4.5:1 on the control face in its normal, hover and pressed states,
// and the destructive confirmation's own ink clears 4.5:1 on the danger face, in both themes.
func TestDialogAnswerFacesMeetContrast(t *testing.T) {
	style := uiStyle(t)
	if strings.Contains(style, "color-mix(") {
		t.Error("the stylesheet tints a face with color-mix(): a non-destructive answer must not acquire a decorative hue")
	}
	for _, bad := range []string{"ans-cancel", "--ans-hue"} {
		if strings.Contains(style, bad) {
			t.Errorf("the stylesheet still carries %q: the red dismissing glyph and the tinted answer faces were retired by BUTTONS 2.0", bad)
		}
	}
	themes := paletteThemes(t)
	for i, name := range []string{"light", "dark"} {
		for _, face := range []string{"control", "control-hover", "control-pressed"} {
			bg := themes[i][face]
			for _, hue := range []string{"success", "warning"} {
				if r := iconart.Contrast(themes[i][hue], bg); r < 3 {
					t.Errorf("%s: the %s glyph on --%s is %.2f:1, want >= 3", name, hue, face, r)
				}
			}
			if r := iconart.Contrast(themes[i]["text-primary"], bg); r < 4.5 {
				t.Errorf("%s: the caption on --%s is %.2f:1, want >= 4.5", name, face, r)
			}
		}
		if r := iconart.Contrast(themes[i]["danger-ink"], themes[i]["danger"]); r < 4.5 {
			t.Errorf("%s: --danger-ink on --danger is %.2f:1, want >= 4.5", name, r)
		}
	}
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
		// COLLAPSIBLE-GROUP 3.1 / BOOLEAN-OPTION 3.5 / SETTINGS-NAVIGATION 2.3 (ticket 119): a group
		// header's caption on its face in rest and hover, and the checkbox outline (a meaningful
		// graphic, 3:1) on every surface a checkbox is drawn on.
		{"text-primary", []string{"surface-header", "surface-header-hover"}, 4.5},
		{"border-strong", []string{"surface-window", "surface-raised", "surface-sunken", "control"}, 3},
		{"accent", []string{"surface-header"}, 4.5},
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
		// An option's help button is named by script - "Help: <option>" - once the language is known.
		if strings.Contains(m[1], `class="opt-help"`) {
			if !strings.Contains(uiHTML, "btn.setAttribute('aria-label', t('helpOptLabel', {name}));") {
				t.Error("the option help buttons are no longer given their accessible name")
			}
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
	body := uiHTML[strings.Index(uiHTML, "<body>"):strings.Index(uiHTML, `<script src="/i18n.js">`)]
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

// UI-SIZING 1.1 section 2 rule 1: the input profile is chosen once at opening and never follows a pointer
// event, so no media query may switch the geometry live; the profile is a data attribute the page sets
// from the primary pointer (or the reader's explicit choice). System motion preference still wins.
func TestDesktopTargetsAndMotionFollowSystem(t *testing.T) {
	style := uiStyle(t)
	for _, want := range []string{`html[data-input="touch"] {`, "@media (prefers-reduced-motion: reduce)", "animation: none !important; transition: none !important; scroll-behavior: auto !important;"} {
		if !strings.Contains(style, want) {
			t.Errorf("missing %q", want)
		}
	}
	for _, bad := range []string{"any-pointer: coarse", "(pointer: coarse)", "min-width: 44px !important", "min-height: 28px !important"} {
		if strings.Contains(style, bad) {
			t.Errorf("the stylesheet carries %q: the profile must be stable (data-input), and the 28/44 floors were superseded by the tokens", bad)
		}
	}
	for _, want := range []string{
		"const autoInput = (matchMedia('(pointer: coarse)').matches || matchMedia('(any-pointer: none)').matches) ? 'touch' : 'pointer';",
		"root.dataset.input = (input === 'touch' || input === 'pointer') ? input : autoInput;",
	} {
		if !strings.Contains(uiHTML, want) {
			t.Errorf("ui.html lost %q - the profile is no longer chosen once", want)
		}
	}
}

// ── WINDOWS-UI 0.1 / APP-SETTINGS 0.2 gates (ticket 93) ─────────────────────────────────────

// BOOLEAN-OPTION 2.0 (ticket 119): every boolean is an .opt row - the native checkbox first (the reading
// start), then one text cell whose heading row holds the caption label (for= the checkbox: the checkbox,
// the caption and the glyph are the activation regions) and, when present, the help wrapper as a SIBLING
// of that label, never inside it, so help can never toggle the option; the explanation sits below the
// heading and is the checkbox's description. No checkbox is styled or exposed as a command button.
func TestBooleanRowsFollowTheCheckboxConvention(t *testing.T) {
	for _, id := range []string{"chkSingle", "chkNoOpen", "chkForce", "chkVerbose", "chkOCR", "chkShellEntries", "chkDefaultHandler", "chkAutoUpdates", "chkRevealKey"} {
		input := regexp.MustCompile(`<input type="checkbox" class="opt-box" id="` + id + `"[^>]*>`).FindString(uiHTML)
		if input == "" {
			t.Errorf("boolean %s is gone from the markup or is not an .opt-box", id)
			continue
		}
		pos := strings.Index(uiHTML, input)
		// The checkbox is the first child of its .opt row.
		if before := strings.TrimRight(uiHTML[:pos], " \n"); !regexp.MustCompile(`<div class="opt"( style="[^"]*")?>$`).MatchString(before) {
			t.Errorf("boolean %s does not lead its .opt row", id)
		}
		window := uiHTML[pos : pos+min(2600, len(uiHTML)-pos)]
		label := regexp.MustCompile(`<label class="opt-name" for="` + id + `">(.*?)</label>`).FindStringSubmatch(window)
		if label == nil {
			t.Errorf("boolean %s has no heading label bound to it (for=)", id)
			continue
		}
		if strings.Contains(label[1], "opt-help") || strings.Contains(label[1], "<button") {
			t.Errorf("boolean %s: the help button sits inside the activation label", id)
		}
		// The explanation (when there is one) is the checkbox's description.
		if d := regexp.MustCompile(`aria-describedby="([A-Za-z]+)"`).FindStringSubmatch(input); d != nil {
			if !strings.Contains(uiHTML, `id="`+d[1]+`"`) {
				t.Errorf("boolean %s describes itself by #%s, which does not exist", id, d[1])
			}
		} else if id != "chkRevealKey" {
			t.Errorf("boolean %s has an explanation but no aria-describedby", id)
		}
	}
	if strings.Contains(uiHTML, `class="chk"`) || strings.Contains(uiHTML, "chk-hint") {
		t.Error("an old .chk row or .chk-hint survives - every boolean is an .opt row now")
	}
	// The checkbox is a vector box at the box size inside a hit area of the target size, drawn from roles.
	box := cssRule(t, ".opt-box")
	for _, want := range []string{"appearance: none;", "width: var(--box); height: var(--box);", "padding: calc((var(--tgt) - var(--box)) / 2);", "margin: 0 calc((var(--tgt) - var(--box)) / -2);"} {
		if !strings.Contains(box, want) {
			t.Errorf(".opt-box lost %q", want)
		}
	}
	// Help is outside the label, and it never toggles: its click stops here.
	for _, want := range []string{"btn.addEventListener('click', ev => { ev.preventDefault(); ev.stopPropagation();", "const optHelps = [];"} {
		if !strings.Contains(uiHTML, want) {
			t.Errorf("ui.html lost %q - option help may toggle its option", want)
		}
	}
}

// WINDOWS-UI section 4 (Numeric): readable digits and readable native spin controls - the whole
// editor scales (16px digits, 110px field, 32px minimum height), never just the box.
func TestNumericEditorsScaleDigitsAndSpinners(t *testing.T) {
	style := uiStyle(t)
	for _, want := range []string{
		`input[type="number"] { font-size: calc(16px * var(--s) * var(--t)); }`,
		`.field.num > input { width: calc(110px * var(--s));`,
		"min-height: var(--tgt);",
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

// WINDOWS-UI sections 3.2-3.4 / SETTINGS-NAVIGATION 4.3 / COLLAPSIBLE-GROUP 4.7: the selected page, each
// group's state and every page's viewport are remembered by stable id, restoring fires no user-edit save,
// and closing a group hands focus to its header.
func TestGroupsAndPaneContextAreRemembered(t *testing.T) {
	for _, want := range []string{
		"page:           currentPage,",                                         // the selected page, by its stable id
		"groups:         Object.fromEntries(",                                  // every group's state (open and closed) by stable id
		"pageScrolls:    pageScrolls,",                                         // each page's viewport rides the settings blob
		"d.tagName === 'DETAILS') d.open = !!s.groups[id]",                     // restore skips unknown ids, sets values only
		"head.focus();",                                                        // focus handoff to the header before hiding focused children
		"showPage(urlParam('page') || savedPage || 'convert', {force: true});", // the page, then its anchor
		// Section 3.4: the restore's own toggle and scroll events arrive after settingsLoaded is
		// set; the guard holds until they have run, and both listeners honour it.
		"let restoring = true;",
		"requestAnimationFrame(() => requestAnimationFrame(() => { restoring = false; }));",
		"if (!restoring) saveSettings();",
		"el('pane').addEventListener('scroll', () => {\n    if (restoring) return;",
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
	// The page and group ids the context is keyed by are stable and present.
	for _, id := range []string{`id="page-convert"`, `id="page-ollama"`, `id="page-ocr"`, `id="page-windows"`, `id="page-interface"`, `id="page-advanced"`, `id="page-about"`, `id="grpUpdates"`, `id="grpSupport"`, `id="grpResources"`, `id="grpRelated"`, `id="togglesRow"`, `id="outputRow"`, `id="langsRow"`} {
		if !strings.Contains(uiHTML, id) {
			t.Errorf("context anchor %s is gone - the saved key is not stable", id)
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

// Ticket 117: the FileDO password dialog is no <form>, and a typed type=password field that leaves
// the DOM before a request is a login signal to the browser's password manager. The field is drawn
// as a masked text input instead; losing either half would expose the password on screen or offer it
// to the browser profile.
func TestFileDOPasswordFieldIsMaskedButNotAPasswordField(t *testing.T) {
	if !regexp.MustCompile(`\.dlg-secret input \{[^}]*-webkit-text-security: disc;`).MatchString(uiHTML) {
		t.Error("the FileDO password field lost its CSS mask (-webkit-text-security)")
	}
	open := strings.Index(uiHTML, "secretInput = document.createElement('input');")
	if open < 0 {
		t.Fatal("the FileDO password field is no longer built in openDialog")
	}
	build := uiHTML[open:]
	build = build[:strings.Index(build, "const note =")]
	if strings.Contains(build, "'password'") {
		t.Error("the FileDO password field is a type=password input again - the browser may offer to save it")
	}
	for _, want := range []string{"secretInput.type = 'text';", "secretInput.autocomplete = 'off';", "secretInput.autocapitalize = 'off';", "secretInput.spellcheck = false;"} {
		if !strings.Contains(build, want) {
			t.Errorf("the FileDO password field lost %q", want)
		}
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

// SETTINGS-NAVIGATION 3.3 / APP-SETTINGS rule 12: About is the last page and carries the whole set - the
// description, the running version, the licence and privacy pages, help, the product site, support and
// contact, the user-initiated diagnostic export, the update information, the repository, the index of
// the program's resources and the related products.
func TestAboutCarriesLicenceAndPrivacy(t *testing.T) {
	about := uiHTML[strings.Index(uiHTML, `id="page-about"`):]
	if about == "" {
		t.Fatal("the About page is gone")
	}
	if strings.Index(uiHTML, `data-page="about"`) < strings.Index(uiHTML, `data-page="advanced"`) {
		t.Error("About is not the last rail destination")
	}
	for _, want := range []string{
		`data-i18n="appPurpose"`,
		`data-i18n="aboutLicence"`,
		`data-i18n="aboutLicenceLink"`,
		`data-i18n="aboutPrivacy"`,
		"privacy.html",
		`data-help-open="index"`,
		`data-i18n="aboutSite"`,
		`id="aboutMail"`,
		`id="btnSendLogs"`,
		`id="btnUpdateCheck"`,
		"releases/latest",
		`data-i18n="aboutRepo"`,
		`id="resTable"`,
		`id="relTable"`,
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
