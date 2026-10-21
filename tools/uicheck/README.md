# tools/uicheck - UI-contract checks of the GUI launcher

Ticket 119. Measures the real `doc-html-ui.exe` against the mandatory shared UI contracts - `BUTTONS` 2.0,
`BOOLEAN-OPTION` 2.0, `COLLAPSIBLE-GROUP` 1.1, `SETTINGS-NAVIGATION` 2.0 and `UI-SIZING` 1.1 - in a headless
Chromium. A screenshot alone proves none of these, so every scenario reads geometry, colours, focus and events
from the engine and writes what it measured. Not part of the shipped product and not run by `scripts/check.ps1`
(it needs a browser and takes minutes); the static pins of the same rules are Go tests in `cmd/doc-html-ui`.

```
node tools/uicheck/run.mjs                 the whole matrix, with timing samples
node tools/uicheck/run.mjs --quick         the baseline set, no timing samples
node tools/uicheck/run.mjs --only options  one scenario: buttons | dialogs | options | groups | nav | boot | timing | captures | site | reader
node tools/uicheck/run.mjs --only reader   the reader chrome of the converted book and of the extension viewer (slice R1);
                                           --go-only skips the extension, the CLI is built into temp/uicheck/bin
node tools/uicheck/run.mjs --only extension the extension's own pages (slice X1): options page, popup, image-OCR page, in-page OCR bar;
                                           no exe and no build, the real extension/ folder is served over loopback; --quick skips the timings
node tools/uicheck/run.mjs --no-build      reuse temp/uicheck/bin
```

Needs the Go toolchain (it builds both executables with the `nodynamic` tag into `temp/uicheck/bin`) and
`npm ci` once in `tools/sitecheck` (the Playwright Chromium, shared with `tools/helpgen`). Run it in PowerShell
from the repository root. The window runs under a scratch profile (`temp/uicheck/work`, `LOCALAPPDATA`
redirected, `DOCHT_GUI_NO_WINDOW`), so nothing of the developer's own settings is read or written.

Output goes to `temp/uicheck/<stamp>/`: `report.json` (every check with its measured detail, the git tree id,
the node version), `timings.raw.json` (every sample), `screenshots/`. The last line is the verdict in the
`CHECK-VERDICT` shape (exit 0 PASS, 1 FAIL, 2 COULD NOT VERIFY, 3 PASS WITH ADVISORIES).

## What it measures

| Scenario | Contract clauses |
| --- | --- |
| `buttons` | BUTTONS 3.1-3.4, 4.2-4.3 and UI-SIZING 2: per profile and scale, every command button's target, corner radius, content width (never stretched), clipping, the equal height of each visual row, the gap token, overlap of interactive targets, no horizontal page scroll, the reserved width of the stateful Convert / Cancel button |
| `dialogs` | BUTTONS 4.4-4.5, 3.5: answer order, neutral Cancel face, the destructive rank, caption contrast, the safe default, Enter from the background, Escape, focus returning to the invoker |
| `options` | BOOLEAN-OPTION 2-4: hit and box size, the heading line, explanation below, help after the name, contrast (outline 3:1, text 4.5:1), the activation regions (checkbox, caption, glyph) toggling once, explanation and help never toggling, short on hover and full on activation from one string, Escape and focus return, keyboard, disabled |
| `groups` | COLLAPSIBLE-GROUP 3-4, in LTR and RTL: no layout space when closed, distinct header, activation regions, marker state, full-width body, Enter / Space, focus handoff, unreachable children, the viewport anchor |
| `nav` | SETTINGS-NAVIGATION 2, 3.4, 4, 5: order, glyphs, names whole, selection cue, one tab stop, arrow keys, no outer-window resize, edits preserved, no operation on navigation, nothing remote when About opens, the page and groups remembered, a stale asynchronous result ignored |
| `boot` | SETTINGS-NAVIGATION 5.1: no visible frame in the wrong language |
| `timing` | SETTINGS-NAVIGATION 5.4: activation to the second animation frame after it - 20 warm transitions per page and 20 independent first visits per baseline page, nearest-rank p95 and the maximum, every raw sample kept |
| `reader` | BUTTONS 2.0 and UI-SIZING 1.1 on a reading page, for the generated reader (chapter bar, single-page bar, index toolbar, search and contents panels) and the extension viewer's toolbar: target 32 / 48 and the scale range 0.875 .. 2 (0 and 100 clamp), radius, content width, clipping, equal row height, the gap tokens, flow from the reading start, overlap, glyph tier, type scale, accessible names and tooltips, contrast of caption and glyph idle and hovered, the selected state and its non-colour cue, forced colours, keyboard ring and order, disclosure state and Escape, panels under a wrapped bar, the external marker, the Search chord, RTL and the longest translations, panels also at the reader's own page zoom of 50 and 200 percent, and - on the generated reader - eleven negative controls that must be caught; also `ICON-EXTERNAL` section 13 F (ticket 122), on both editions: a document picture that cannot be shown draws `content.image` and its alt text in place of the browser's broken-image icon - the stand-ins and their text, the valid picture left as it was, no further request from the handler, a picture inserted later, the OCR-plate guard, no horizontal scroll of its own, contrast, and a negative control that cuts the handler out |
| `extension` | BUTTONS 2.0, BOOLEAN-OPTION 2.0, COLLAPSIBLE-GROUP 1.1, SETTINGS-NAVIGATION 2.0 and UI-SIZING 1.1 on the extension's own pages (`extension.mjs` the matrix and the pages, `extension-rules.mjs` the geometry / contrast / option / group rules, `extension-nav.mjs` the rail, the dialogs and the timings, `extension-bar.mjs` the bar injected into a plain and a hostile page, `extension-notices.mjs` the viewer's export card, resume offer, remote-images notice, generic notices and status Cancel - opened on a real CBZ, Markdown book, broken PDF and the empty state: answer order, one primary, neutral faces, targets, row, glyphs, type from the chrome tokens and never the reader's size, default focus, Tab order, Escape and focus return, with their own negative controls; `extension-negative.mjs` the negative controls of the pages): every command button's target, radius, content width, clipping, row height, gap, overlap and the horizontal scroll at pointer and touch, scales 0.875 .. 2, text enlargement, five languages incl. Arabic, light, dark, forced colours and a narrow window; the actual text, glyph, border and checkbox contrast pairs; the options' activation regions and local help; the groups; the rail (order, keyboard, no write but the remembered page, edits preserved, nothing remote on About, memory of page, groups and viewport, the drawer); the reset and import dialogs; the popup's states; timings with every raw sample; the OCR bar's isolation; and the negative controls |
| `captures` | the screenshot set (light, dark, compact touch, extra-large, huge text in German, RTL Arabic, narrow drawer, forced colours, the open option help) |

The matrix is the baseline plus risk-selected combinations (UI-SIZING 4): compact, ordinary, extra-large and huge
scales, both profiles, the longest translations (German, Russian, Hindi), Arabic (RTL), narrow windows and
text enlargement. It is not the full Cartesian product.

The `extension` scenario stubs the few `chrome.*` calls the pages make (storage in memory, the manifest, tabs, runtime, `i18n` served from the real `_locales` file), so the pages' own scripts run unmodified; every page write is logged on `window.__log`, which is how it states that navigating wrote nothing but the remembered page. The 27 negative controls in `extension-negative.mjs` break a page on purpose (a stretched button, a pill corner, a washed-out explanation, a checkbox with no hit area, a sliding switch that came back, ..) and run the same rules; one that is not caught is a failed check.

`reader.mjs` is the generated reader (fixtures converted by the real CLI, the case matrix, the interaction checks), `reader-ext.mjs` the extension viewer (the real `extension/` folder served over loopback, a stub of the few `chrome.*` calls the viewer makes at load, an FB2 document with a nested contents tree), `reader-rules.mjs` the geometry rules both share and `reader-measure.mjs` the readers that run inside the page. `reader-missing.mjs` is the missing-picture scenario of both: a real EPUB (a cover wrapper, a valid picture and two missing ones) converted by the CLI to a single page and to chapter pages, and an HTML document opened in the extension viewer.

## Limits - what a run does not establish

Headless Chromium on the machine that ran it: not a physical 100 / 150 / 200 percent display, not operating-system
high contrast (forced colours are emulated), not a touch screen, not a screen reader, not the Edge or Chrome `--app`
window itself. For the reader: the extension viewer runs against a stub of `chrome.*` and one FB2 document (no PDF,
EPUB or comic load, no real extension origin), the generated pages come from a Markdown book and a synthetic CBZ (not a
real EPUB or PDF corpus), and the popup of a native `<select>` is not observable in a headless capture. The timing is activation to the second animation frame in a headless engine (it is frame-bound
near 36 ms), so it demonstrates "far under the budgets" on this machine, not a minimum-supported device. For the extension: the pages run against a stub of `chrome.*` (no real extension origin, no installed extension in Chrome or Edge), the OCR engine and the language downloads are not run (a download is only checked for its state width and its unit behaviour in `extension/test`), and the bar is injected by hand into a plain page.
