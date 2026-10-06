package main

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"doc-html-translate/internal/i18n"
	"doc-html-translate/internal/iconart"
	"doc-html-translate/internal/pipeline"
)

// Static gates of the second WINDOWS-UI / APP-SETTINGS / APP-BEHAVIOUR conformance pass over the
// launcher page, one per rule, in the style of contract_test.go (which holds ticket 93's gates).
// They live in their own file to keep each under the size budget.

// cssRule returns the declarations of the stylesheet rule whose selector is exactly sel.
func cssRule(t *testing.T, sel string) string {
	t.Helper()
	style := uiStyle(t)
	start := strings.Index(style, "\n"+sel+" {")
	if start < 0 {
		t.Fatalf("the stylesheet has no %q rule", sel)
	}
	body := style[start+len(sel)+3:]
	return body[:strings.Index(body, "}")]
}

// divByID returns the markup of the <div> with the given id, through its matching </div>.
func divByID(t *testing.T, id string) string {
	t.Helper()
	start := strings.Index(uiHTML, `<div id="`+id+`"`)
	if start < 0 {
		t.Fatalf("ui.html has no <div id=%q>", id)
	}
	depth := 0
	for _, loc := range regexp.MustCompile(`<div\b|</div>`).FindAllStringIndex(uiHTML[start:], -1) {
		if uiHTML[start+loc[0]+1] == '/' {
			depth--
		} else {
			depth++
		}
		if depth == 0 {
			return uiHTML[start : start+loc[1]]
		}
	}
	t.Fatalf("<div id=%q> never closes", id)
	return ""
}

// roleOf reads the palette role a declaration paints with: "color: var(--accent)" -> "accent".
func roleOf(t *testing.T, decls, property string) string {
	t.Helper()
	m := regexp.MustCompile(`(?:^|[;\s])` + property + `:\s*var\(--([a-z-]+)\)`).FindStringSubmatch(decls)
	if m == nil {
		t.Fatalf("no %s: var(--role) in %q", property, decls)
	}
	return m[1]
}

// WINDOWS-UI section 2.3: the pane starts with its title and one short, muted purpose line.
func TestPaneStartsWithTitleAndPurpose(t *testing.T) {
	if !regexp.MustCompile(`<h1>DOC-HTML-UI</h1>[\s\S]*?<p class="hint purpose" data-i18n="appPurpose">[^<]+</p>\s*</div>`).MatchString(uiHTML) {
		t.Error("the header does not end with the muted, translated purpose line under the title")
	}
	if !strings.Contains(cssRule(t, ".head .purpose"), "flex-basis: 100%") {
		t.Error("the purpose line does not take its own row below the header items")
	}
}

// APP-SETTINGS rule 3 / WINDOWS-UI section 4: each setting row this pass covered has its muted
// hint right below it - what the setting does, what it costs, what zero or empty means.
func TestSettingsRowsCarryAMutedHint(t *testing.T) {
	for _, id := range []string{"outputFolder", "srcLang", "ollamaModel", "ollamaParallel", "ollamaCtx", "splitSize", "maxCost"} {
		pos := strings.Index(uiHTML, `id="`+id+`"`)
		if pos < 0 {
			t.Errorf("setting %s is gone from the markup", id)
			continue
		}
		rest := uiHTML[pos:]
		next := strings.TrimSpace(rest[strings.Index(rest, "</div>")+len("</div>"):])
		for strings.HasPrefix(next, "<!--") {
			next = strings.TrimSpace(next[strings.Index(next, "-->")+3:])
		}
		if !strings.HasPrefix(next, `<div class="hint" data-i18n`) {
			t.Errorf("setting %s has no muted hint below its row: %.60q", id, next)
		}
	}
}

// APP-BEHAVIOUR rule 5: clearing the recent list cannot be undone, so it is confirmed before
// anything is forgotten, and every destructive confirmation leaves the safe answer focused.
func TestDestructiveActionsAreConfirmedSafely(t *testing.T) {
	handler := uiHTML[strings.Index(uiHTML, "el('btnRecentClear').addEventListener"):]
	handler = handler[:strings.Index(handler, "\n});")]
	confirm := strings.Index(handler, "await confirmDialog(t('confirmRecentClear'")
	if confirm < 0 || strings.Index(handler, "recentDocs = [];") < confirm {
		t.Error("Recent \"Clear all\" forgets the list without a confirmation first")
	}
	calls := regexp.MustCompile(`confirmDialog\([^\n]*'danger'[^\n]*\)`).FindAllString(uiHTML, -1)
	if len(calls) < 4 {
		t.Fatalf("found %d destructive confirmations; the scan is wrong", len(calls))
	}
	for _, call := range calls {
		if !strings.Contains(call, "'danger', true)") {
			t.Errorf("a destructive confirmation focuses the acting button: %s", call)
		}
	}
}

// APP-BEHAVIOUR rule 3, no second start: the run's in-flight flag and the disabled button are
// set before runConvert's first await, every entry point checks the flag, and the flag is
// cleared only in the run's own finally (ui_dialog_test.mjs drives the double press).
func TestRunCannotStartTwice(t *testing.T) {
	for _, want := range []string{
		"let runInFlight = false;",
		"if (queueBusy || runAbort || runInFlight) return;",
		"if (runInFlight || queueBusy) return;\n    runInFlight = true;\n    const btn = el('btnRun');\n    btn.disabled = true;\n    try {",
		"} finally {\n        runInFlight = false;",
		"el('btnRun').disabled = queueBusy || runInFlight;",
		"if (queueBusy || runAbort || runInFlight || queue.length < 2) return;",
	} {
		if !strings.Contains(uiHTML, want) {
			t.Errorf("ui.html lost %q - a second Convert can start while the first is preparing", want)
		}
	}
}

// WINDOWS-UI section 2.5: technical values keep their data's direction in a right-to-left
// window, and the log lets each line find its own base direction.
func TestTechnicalValuesKeepTheirDirection(t *testing.T) {
	for _, id := range []string{"cmdLine", "logArea", "prevResultDir", "outputFolder", "ollamaModel", "googleKey"} {
		tag := regexp.MustCompile(`<[a-z]+\b[^>]*\bid="` + id + `"[^>]*>`).FindString(uiHTML)
		if tag == "" {
			t.Errorf("#%s is gone from the markup", id)
			continue
		}
		if !strings.Contains(tag, `dir="ltr"`) {
			t.Errorf("#%s is a technical value without dir=\"ltr\": %s", id, tag)
		}
	}
	if !strings.Contains(cssRule(t, ".log"), "unicode-bidi: plaintext;") {
		t.Error(".log lacks unicode-bidi: plaintext - a mixed-direction line reads out of order")
	}
}

// WINDOWS-UI section 6.5: under forced colors a mask-drawn glyph keeps a visible system colour.
// Forced colors repaint its background with Canvas, so every rule that draws through a mask
// needs a forced-colors twin that opts out and paints CanvasText or ButtonText.
func TestMaskGlyphsSurviveForcedColors(t *testing.T) {
	style := regexp.MustCompile(`(?s)/\*.*?\*/`).ReplaceAllString(uiStyle(t), "")
	var forced strings.Builder
	const media = "@media (forced-colors: active) {"
	for rest := style; ; {
		i := strings.Index(rest, media)
		if i < 0 {
			break
		}
		block := rest[i+len(media):]
		end := strings.Index(block, "\n}")
		forced.WriteString(block[:end])
		rest = block[end:]
	}
	maskDecl := regexp.MustCompile(`(?:^|[;\s])(?:-webkit-)?mask(?:-image)?\s*:`)
	seen := map[string]bool{}
	for _, m := range regexp.MustCompile(`([^{}]+)\{([^{}]*)\}`).FindAllStringSubmatch(style, -1) {
		if !maskDecl.MatchString(m[2]) {
			continue
		}
		// An [open] variant swaps the mask image on the same element; its twin is the base rule's.
		sel := strings.TrimSpace(strings.ReplaceAll(m[1], "[open]", ""))
		if seen[sel] {
			continue
		}
		seen[sel] = true
		twin := regexp.MustCompile(regexp.QuoteMeta(sel) + `\s*\{([^}]*)\}`).FindStringSubmatch(forced.String())
		if twin == nil {
			t.Errorf("mask glyph %q has no forced-colors rule", sel)
			continue
		}
		if !strings.Contains(twin[1], "forced-color-adjust: none") ||
			!(strings.Contains(twin[1], "CanvasText") || strings.Contains(twin[1], "ButtonText")) {
			t.Errorf("mask glyph %q under forced colors: %q, want forced-color-adjust: none and a system text colour", sel, twin[1])
		}
	}
	if len(seen) == 0 {
		t.Fatal("no mask-drawn glyph found - the disclosure chevron should be one; the scan is wrong")
	}
}

// WINDOWS-UI section 6.4: a list row's text clears 4.5:1 on the fill it actually sits on in
// every state - resting, hovered, selected, and selected while hovered - in both themes. The
// roles are read from the rules themselves, so a restyle is measured, not assumed.
func TestListRowStatesMeetWCAGAA(t *testing.T) {
	themes := paletteThemes(t)
	rest := roleOf(t, cssRule(t, "#queueList"), "background")
	if got := roleOf(t, cssRule(t, "#recentList"), "background"); got != rest {
		t.Fatalf("queue and recent lists rest on different surfaces (%s, %s)", rest, got)
	}
	text := roleOf(t, cssRule(t, ".qsel"), "color")
	hover := roleOf(t, cssRule(t, ".qsel:hover"), "background")
	selected := roleOf(t, cssRule(t, ".qsel.selected"), "color")
	selHoverBG, selHoverFG := hover, selected
	if strings.Contains(uiStyle(t), "\n.qsel.selected:hover {") {
		sh := cssRule(t, ".qsel.selected:hover")
		selHoverBG, selHoverFG = roleOf(t, sh, "background"), roleOf(t, sh, "color")
	}
	pairs := [][3]string{
		{"resting", text, rest},
		{"hovered", text, hover},
		{"selected", selected, rest},
		{"selected and hovered", selHoverFG, selHoverBG},
	}
	for i, theme := range []string{"light", "dark"} {
		for _, p := range pairs {
			fg, okF := themes[i][p[1]]
			bg, okB := themes[i][p[2]]
			if !okF || !okB {
				t.Fatalf("%s: --%s or --%s is not a hex light-dark() pair", theme, p[1], p[2])
			}
			if r := iconart.Contrast(fg, bg); r < 4.5 {
				t.Errorf("%s, %s row: --%s on --%s is %.2f:1, want >= 4.5", theme, p[0], p[1], p[2], r)
			}
		}
	}
}

// WINDOWS-UI section 6.1: accent identifies interaction, so a static section heading is not
// painted like the clickable group headers, and its ink clears 4.5:1 on the window in both themes.
func TestStaticHeadingsDoNotWearTheAccent(t *testing.T) {
	role := roleOf(t, cssRule(t, ".stitle"), "color")
	if strings.HasPrefix(role, "accent") || role == "link" {
		t.Fatalf(".stitle is painted --%s - a static heading would read as a control", role)
	}
	themes := paletteThemes(t)
	for i, theme := range []string{"light", "dark"} {
		if r := iconart.Contrast(themes[i][role], themes[i]["surface-window"]); r < 4.5 {
			t.Errorf("%s: .stitle --%s on --surface-window is %.2f:1, want >= 4.5", theme, role, r)
		}
	}
}

// WINDOWS-UI section 7: an unknown log count is reported as unknown - never folded into zero by
// the page, and never answered with "nothing to clear".
func TestUnknownLogCountIsNotZero(t *testing.T) {
	for _, want := range []string{
		"aboutFacts.logs = d.logs || null;",
		"aboutFacts.logs = env.logs || null;",
		"logs.textContent = t('aboutLogsUnknown');",
		"if (!aboutFacts.logs) { setStatus('sendLogsMsg', 'logsUnknownClear', null, 'fail'); return; }",
	} {
		if !strings.Contains(uiHTML, want) {
			t.Errorf("ui.html lost %q - an unreadable log store would read as empty", want)
		}
	}
	if strings.Contains(uiHTML, "logs || {count: 0") {
		t.Error("ui.html folds a missing log count into zero")
	}
}

// APP-BEHAVIOUR rule 6: a failed run names its cause. Every exit code the CLI publishes maps to a
// localized cause, an unknown code keeps its number and points at the logs, and the failure
// dialog keeps its three actions.
func TestFailuresNameTheirCause(t *testing.T) {
	m := regexp.MustCompile(`const RUN_FAILURE_CAUSES = \{([^}]*)\};`).FindStringSubmatch(uiHTML)
	if m == nil {
		t.Fatal("ui.html has no RUN_FAILURE_CAUSES table")
	}
	causes := map[int]string{}
	for _, e := range regexp.MustCompile(`(\d+): '([A-Za-z]+)'`).FindAllStringSubmatch(m[1], -1) {
		code, _ := strconv.Atoi(e[1])
		causes[code] = e[2]
	}
	en := guiLangObjects(t)["en"]
	for _, code := range []int{pipeline.ExitArgsError, pipeline.ExitIOError, pipeline.ExitParse, pipeline.ExitAPI, pipeline.ExitInterrupted} {
		key, ok := causes[code]
		if !ok {
			t.Errorf("exit code %d has no named cause", code)
			continue
		}
		if !en[key] {
			t.Errorf("exit code %d maps to %q, which the dictionary does not define", code, key)
		}
	}
	for _, want := range []string{
		"t(RUN_FAILURE_CAUSES[end.code] || 'runCauseUnknown', {code: end.code})",
		"buttons: [{id: 'send', key: 'btnSendLogs'}, {id: 'retry', key: 'btnRetry', kind: 'primary'}, {id: 'close', key: 'dlgClose', focus: true}]",
	} {
		if !strings.Contains(uiHTML, want) {
			t.Errorf("ui.html lost %q", want)
		}
	}
	unknown := regexp.MustCompile(`(?m)^ {8}runCauseUnknown: "(.*)",\r?$`).FindAllStringSubmatch(uiI18nJS, -1)
	if len(unknown) != len(i18n.Codes) {
		t.Fatalf("runCauseUnknown declared %d times, want %d", len(unknown), len(i18n.Codes))
	}
	for _, u := range unknown {
		if !strings.Contains(u[1], "{code}") {
			t.Errorf("runCauseUnknown drops the exit code: %q", u[1])
		}
	}
}

// APP-BEHAVIOUR rule 11: what the packaged build hides is hidden whole - the update hint goes
// with the controls it describes.
func TestHiddenControlsTakeTheirHintAlong(t *testing.T) {
	block := divByID(t, "updateControls")
	for _, want := range []string{`id="btnUpdateCheck"`, `id="chkAutoUpdates"`, `data-i18n="hintAutoUpdate"`} {
		if !strings.Contains(block, want) {
			t.Errorf("#updateControls lacks %s - the packaged build would leave it half-hidden", want)
		}
	}
	if !strings.Contains(uiHTML, "document.getElementById('updateControls').style.display = aboutFacts.packaged ? 'none' : '';") {
		t.Error("the packaged build no longer hides #updateControls")
	}
}

// APP-BEHAVIOUR rule 3: cancelled is an outcome of its own; only a failure takes the failure
// style in the queue's log.
func TestCancelledIsNotStyledAsAFailure(t *testing.T) {
	if !strings.Contains(uiHTML, "log(itemStateText(it) + '\\n', it.state === 'failed' ? 'end failed' : 'end');") {
		t.Error("the queue no longer styles its end line by failure alone")
	}
	if strings.Contains(uiHTML, "it.state === 'done' ? 'end' : 'end failed'") {
		t.Error("a cancelled queue item is logged in the failure style")
	}
}

// ICON-SET rule 3 / WCAG 2.5.3: a button with visible text is named by that text. An
// aria-label override replaced the visible word with another sentence - in Russian with the
// word of a different action; the description stays in the title.
func TestTextButtonsAreNamedByTheirText(t *testing.T) {
	tags := regexp.MustCompile(`<[^>]*>`)
	letter := regexp.MustCompile(`\pL`)
	text := 0
	for _, m := range regexp.MustCompile(`(?s)<button([^>]*)>(.*?)</button>`).FindAllStringSubmatch(uiHTML, -1) {
		if !letter.MatchString(tags.ReplaceAllString(m[2], "")) && !strings.Contains(m[1], "data-i18n=") {
			continue
		}
		text++
		if strings.Contains(m[1], "aria-label=") || strings.Contains(m[1], "data-i18n-aria=") {
			t.Errorf("text button carries an accessible-name override: %.120s", strings.TrimSpace(m[0]))
		}
	}
	if text < 15 {
		t.Fatalf("found %d text buttons; the scan is wrong", text)
	}
	// The queue's per-row Remove adds the file to its name, after the visible word.
	if !strings.Contains(uiHTML, "del.setAttribute('aria-label', t('queueRemoveItem') + ': ' + it.name);") {
		t.Error("the queue row's Remove button is not named by its visible word")
	}
}

// APP-BEHAVIOUR rule 7: one formatting pass, and a missing argument renders a blank
// (ui_dialog_test.mjs drives the cases).
func TestFormattingRendersAMissingArgumentBlank(t *testing.T) {
	fn := uiHTML[strings.Index(uiHTML, "function t(key, params) {"):]
	fn = fn[:strings.Index(fn, "\n}")]
	if !strings.Contains(fn, `.replace(/\{(\w+)\}/g, (m, k) => params && params[k] != null ? String(params[k]) : '')`) {
		t.Errorf("t() is not the single regex pass: %s", fn)
	}
	if strings.Contains(fn, ".split(") {
		t.Error("t() still splits per parameter - an absent key leaves {name} on screen")
	}
}

// APP-SETTINGS rule 10 / INPUT-PARITY rule 2: the focus ring is never switched off, and the
// pane gives it room so the pane's own clipping cannot hide it on a full-width field.
func TestFocusRingIsNeverSuppressed(t *testing.T) {
	style := uiStyle(t)
	if m := regexp.MustCompile(`outline:\s*(none|0)\b`).FindString(style); m != "" {
		t.Errorf("the stylesheet suppresses the focus ring: %q", m)
	}
	if !strings.Contains(style, ":focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }") {
		t.Error("the shared :focus-visible ring is gone")
	}
	pane := cssRule(t, ".col-main")
	if !strings.Contains(pane, "padding: 4px;") || !strings.Contains(pane, "margin: -4px;") {
		t.Errorf(".col-main clips the ring of its edge controls: %q", pane)
	}
}

// APP-STYLE section 4: warning ("this may not do what you expect") and danger (a failure or a
// refusal) are different roles. Only the caveats below are painted warning; everything else
// that went wrong is a failure.
func TestWarningAndDangerAreDistinct(t *testing.T) {
	if got := roleOf(t, cssRule(t, ".hint.warn"), "color"); got != "warning" {
		t.Errorf(".hint.warn is --%s, want --warning", got)
	}
	if got := roleOf(t, cssRule(t, ".hint.fail"), "color"); got != "danger" {
		t.Errorf(".hint.fail is --%s, want --danger", got)
	}
	caveats := []string{"googleKeyNone", "googleKeySavedUnverified", "registerBlocked", "registerUnknown", "registerPartial", "unregisterStill"}
	script := uiHTML[strings.Index(uiHTML, "<script>"):]
	warned := 0
	for _, re := range []string{`setStatus\('[^']+', '([A-Za-z]+)'[^;\n]*'warn'\)`, `\['([A-Za-z]+)'[^\]\n]*'warn'\]`} {
		for _, m := range regexp.MustCompile(re).FindAllStringSubmatch(script, -1) {
			warned++
			if !slices.Contains(caveats, m[1]) {
				t.Errorf("%s is painted warning; a failure or refusal takes 'fail' (danger)", m[1])
			}
		}
	}
	if warned == 0 || !strings.Contains(script, "'fail')") {
		t.Fatal("found no warning or no failure status; the scan is wrong")
	}
	if !strings.Contains(uiHTML, `<div class="hint fail" id="cliWarn"`) {
		t.Error("the missing-converter notice says Convert cannot work; it is a failure, not a caveat")
	}
}
