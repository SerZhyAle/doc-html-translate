# UI profile - how the product meets the shared interface contracts

The product's own record of the mandatory shared UI rules (`UI-SIZING`, `BUTTONS`, `BOOLEAN-OPTION`,
`COLLAPSIBLE-GROUP`, `SETTINGS-NAVIGATION`, with `WINDOWS-UI`, `APP-SETTINGS`, `ICON-RENDER` and `PAGE-STYLE`
where they apply). It holds the size-profile record the sizing contract asks every product to publish, the
permanent variants (design choices, not expiring exceptions), what is proven and how to re-prove it, and what
is still owed. The contracts themselves live in the shared catalog and are cited here by id and section; their
text is never copied. The work is ticket 119; its tactical plan carries the dated slices.

**State of 2026-10-20.** The desktop launcher (`cmd/doc-html-ui`) implements the five component contracts and
is measured by `tools/uicheck` on the real build; the official site's page layer (`assets/site.css`, `site.js`,
the pages) is corrected and measured (`node tools/uicheck/run.mjs --only site`, 655 checks). The generated reader chrome
(the chapter bar, the single-page bar, the index toolbar, the search and contents panels) and the extension viewer's toolbar
twin implement the two contracts that apply to a reading page and are measured by
`node tools/uicheck/run.mjs --only reader` (slice R1, 2026-10-20). The browser extension's own pages - the options page, the toolbar popup, the image-OCR page and the in-page OCR bar - are the plan's X1 and were
taken on 2026-10-20 (`node tools/uicheck/run.mjs --only extension`), and the viewer's notices, resume offer, export card, remote-images notice and
status Cancel followed the same day (the second part of X1, measured by the same scenario). A plan is not conformance: the registry rows move only
after measured proof.

## 1. Surfaces and what applies

| Surface | UI-SIZING | BUTTONS | BOOLEAN-OPTION | COLLAPSIBLE-GROUP | SETTINGS-NAVIGATION | State |
| --- | --- | --- | --- | --- | --- | --- |
| Desktop launcher window | yes | yes | yes (11 options) | yes (4 groups, About) | yes (rail, 7 pages) | implemented, measured |
| Browser extension: options page | yes (32 / 48; compact 28 / 42; five scales, text, controls) | yes | yes (3 options) | yes (3 groups on About) | yes (rail, 6 pages, drawer under 60 text-em) | implemented, measured (X1, 2026-10-20) |
| Browser extension: toolbar popup | yes (same tokens and profile) | yes | yes (3 options) | yes (1 group) | no - a task panel that links to Settings and edits the same stored keys | implemented, measured (X1) |
| Browser extension: image-OCR page, in-page OCR bar | yes | yes | no | no | no | implemented, measured (X1); the bar is isolated from the page it is injected into |
| Browser extension: reader toolbar | yes (32 / 48; the toolbar, its search panel and the contents tree's chevron buttons) | yes | no - mode toggles are not options | no | no | implemented, measured on an FB2 document (R1, 2026-10-20) |
| Browser extension: viewer notices, offers, export card, status Cancel | yes (the same 32 / 48 tokens; type from the tokens, never the reader's size) | yes (answers in reading order, one primary at most, neutral faces) | no | no | no | implemented, measured on a CBZ, a Markdown book and a broken PDF (X1, 2026-10-20) |
| Generated reader chrome (converted book) | yes (32 / 48) | yes | no - mode toggles are not options | no - the table of contents is navigation | no | implemented, measured (R1, 2026-10-20); no size selector is exposed (owner decision, section 4); custom Ctrl+wheel zoom removed in slice Z (owner decision Q8, browser zoom takes chord) |
| Official site | yes (44 fine / 48 coarse) | yes (with `PAGE-STYLE`) | no checkbox exists | yes (`details.sec`) | no - ordinary site navigation is out of scope | implemented in the page layer and measured (W1, 2026-10-20); the theme control's visible label and the kit's own gaps are open (section 6) |
| CLI | no graphical controls; `--help` is its own help | | | | | not applicable |

## 2. Size and input profile (UI-SIZING sections 2 and 4)

One token resolver serves every covered component of the launcher (`cmd/doc-html-ui/ui.html`, the `html { .. }`
token block): `--ui` is the interface scale, `--tx` the text-only enlargement, and the input profile
(`data-input`) is chosen **once when the window opens** - from the primary pointer when set to Automatic - and
never follows a pointer or touch event. The Interface page exposes **Interface size**, **Text size** and
**Controls** (Automatic, Mouse and keyboard, Touch); `?size=`, `?text=`, `?input=` and `?page=` are read-only
overrides for the check tooling and never touch the saved choice.

| Token | Pointer profile | Touch profile |
| --- | --- | --- |
| Ordinary single-line height and hit-target baseline | 32 | 48 |
| Horizontal padding per side | 12 | 16 |
| Caption glyph tier | 16 | 20 |
| Glyph-to-caption gap, button gap | 8 | 8 |
| Corner radius, border | 4, 1 | 4, 1 |
| Checkbox square (visible) | 20 | 24 |
| Group header hit height | 32 | 48 |
| Rail row hit height | 32 | 48 |

- **Supported interface scales:** compact 0.875, ordinary 1, large 1.25, extra large 1.5, huge 2. The resolver
  clamps `--ui` to 0.875 .. 2, so an unknown value never produces a zero-size or unboundedly shrinking target.
- **Declared smallest targets (measured, UI-SIZING 2.4):** 28 x 28 CSS px for the pointer profile and 42 x 42
  CSS px for the touch profile, both at compact scale (0.875 of 32 and of 48). Ordinary targets are 32 and 48.
- **Text enlargement:** 1, 1.25, 1.5, 2; it grows text and layout first. **Glyph bounds:** a companion glyph
  grows at half the text multiplier (`--gm` = 1 + (text - 1) / 2), so at the largest text a 16 px glyph is 24 px
  (ordinary scale); the interface scale multiplies glyphs fully.
- **Compact caption policy:** compact keeps every caption. No button is turned icon-only to fit a width or a
  translation; a whole button wraps to the next row first, and only a button that cannot fit alone wraps its
  caption and grows.
- **Composition:** every length is a CSS pixel, so the browser zoom and the operating-system scale compose
  underneath once; nothing is scaled twice. A change of size, text or profile remeasures the stateful buttons at
  their final width (`reserveStacks`) before the next frame.
- **Hybrid devices:** a touch laptop opens in the pointer profile (the primary pointer is a mouse); the reader
  switches deliberately on the Interface page, and nothing moves under an active interaction.

## 3. Component policies

**Buttons.** Every command button carries its ICON-SET glyph at the reading start, on a neutral face
(recorded policy: a glyph on every command, dialog answers included - more than the minimum the contract
requires). Ranks are separate from behaviour: ordinary, secondary (transparent face), primary (the accent; one
per task surface - Convert, and the first-run entry) and destructive (the danger face of an irreversible
confirmation's acting button). A command panel is a wrapping flex row: content widths, whole-button wrapping,
every button of one visual line takes the height of that line's tallest member, the gap is the token, and spare
space stays empty. A stateful button (Convert / Cancel, Download / Cancel) reserves its widest state. Dialog
answers read Cancel first, then any helping action, then the final affirmative; the safe answer holds the
focus; the glyph carries the meaning (`state.ok` affirmative, `state.warning` caution) on the neutral face.
**The drop zone is a drop target, not a command button** - a content card whose click is a convenience
(recorded 2026-10-11 and kept).

**Boolean options.** Eleven options use the `.opt` component: a leading vector checkbox (20 / 24 visible, hit
area 32 / 48 by padding that a negative margin takes back), a heading label that is the activation region, the
explanation below as the checkbox's description, optional local help. Local help exists only where the visible
explanation is not enough - the OCR overlay and the default-handler option; every other option omits it
(BOOLEAN-OPTION 4.1). The short hover/focus summary and the full explanation are one dictionary string (summary,
a blank line, the rest).

**Groups.** Groups are used only where a page has several topics: the About page (Updates, Support and
diagnostics, Resources, Related products). Support starts open, the others closed; the state is remembered by
the stable group id. Single-topic pages show their controls directly. Collapsed headers are not packed: the
page has four groups, and the design record chooses the plain stacked layout.

**Settings navigation.** The window is a start-side rail and a content pane. Seven pages, one level, in this
order and by these stable ids: `convert` (the work and the run parameters), `ollama`, `ocr`, `windows` (absent in
the packaged build), `interface` (the theme, the interface language, the interface and text size, the controls),
`advanced`, `about` (last). The rail glyph treatment is **monochrome**, one glyph and one short name per destination. A
window narrower than 60 text-em (about 780 CSS px at the ordinary scale, wider as the interface grows) keeps the
same menu as a start-side drawer opened from the header's Pages button; choosing a page, the scrim or Escape
closes it and focus returns to the button. Selecting a page never resizes the window, moves the rail, discards
an edit, commits a value or runs an operation; the page and each page's viewport are remembered by stable ids.

**About (SETTINGS-NAVIGATION 3.3).** Description, running version, licence and licence text, privacy, help,
product site, support and contact, the user-initiated diagnostic export (the stored-logs archive), the update
check and the download page, the repository, the resource index (converter engine, Tesseract, Calibre, 7-Zip,
FileDO, Ollama: what each is for, whether it was found - a local lookup - and where to get it) and the related
products (FileDO, the SZA portal). Opening About contacts nothing remote.

**Reader chrome (slice R1).** The generated reader and the extension viewer's toolbar take `BUTTONS` 2.0 and `UI-SIZING`
1.1 with one set of numbers, written once as CSS custom properties on the chrome root (`--dht-*` in
`internal/htmlgen/chromecss.go`, `--*` in `extension/src/viewer.css`): ordinary targets **32** CSS px for a fine pointer
and **48** under `@media (pointer: coarse)` (the primary pointer, chosen by the stylesheet, never per event), padding
12 / 16 per side, glyph tier 16 / 20, radius 4, border 1, button and wrapped-row gap 8, glyph-to-caption gap 8, type 14
with 12 for note text. The values multiply by `clamp(0.875, ui, 2)` with `ui` = 1: **no size selector is exposed**, so the
only shipped scale is ordinary, and the clamp (smallest target 28 / 42, largest 64 / 96) is what a later selector would
inherit; `tools/uicheck` measures 0.875, 1 and 2 by setting the property, and an unknown value (0, 100) lands on the
clamp. The 28 / 44 floors of tickets 57, 90 and 108 are superseded for the covered buttons. The other numbers of this
file (the launcher's scales, text enlargement, the Controls setting) do not apply to a reading page: its text size is
the reader's own A- / A+, which scales the book and never the chrome.

Command rows are task groups that flow from the reading start and wrap whole buttons before a caption wraps; nothing is
pinned to the far end; every button of a visual line takes the line's height; widths are content widths. The glyph
precedes the caption on every button that has one. Glyph-only commands (text size, reset, night, text layer, the
single-page contents button) are the familiar repeated ones, each with a tooltip and an accessible name; Search is
`action.search` and its word; **Fit width, Open file, Original PDF and the panels' Close are words** (the vocabulary has
no fit-width meaning; `nav.close` and `action.choose-file` are not vendored). Search and Contents are disclosures that
keep their action glyphs and say `aria-expanded`; the mode toggles say `aria-pressed`; a selected control is a distinct
face plus an inset bar along the block end in the ink colour, which forced colours swap for the system's selected pair;
a paging end with no page says `aria-disabled`. The version link and the extension's two outside links are external
destinations and carry the `nav.open-external` marker (mirrored in a right-to-left bar). Panels hang under a wrapped bar
at its measured height, never at a fixed offset. The bar stays an ordinary Tab row (no `role="toolbar"`); the reader
pages have **no help button** (ticket 118 decision 5). The contents tree and the search results are navigation rows, not
buttons; the extension's tree chevron, a real `<button>`, takes the target token.

**Browser extension (slice X1).** The extension's own pages carry the launcher's profile with its own carriers:
`ui-sizing.css` is the one token resolver (the launcher's names and numbers: 32 / 48 targets, 12 / 16 padding, glyph tier 16 / 20,
radius 4, gap 8, box 20 / 24; scales 0.875 .. 2, text 1 .. 2, glyphs at half the text multiplier, declared floors 28 / 42), set by
`data-size`, `data-text`, `data-input` and `data-chrome-theme` on the root. The saved **Interface size**, **Text size** and
**Controls** (Automatic, Mouse and keyboard, Touch) and the **Interface theme** (System, Light, Dark) live on the options page's
Interface and General destinations and in the stored `options`; the automatic profile is read once from the primary pointer in
`<head>` and never from an event. The options page is a rail of six destinations in the fixed order `general ocr sites data
interface about` (About last; monochrome glyphs; one level), a pane that scrolls on its own, and the same menu as a start-side
drawer when the window is narrower than 60 text-em (840 CSS px ordinary); the page, the groups and each page's viewport are
remembered by stable ids in the separate `uiState` key, so navigating writes no option and rebuilds no interception rule.
Groups appear only where a page has several topics (About: Support and diagnostics, open by default; Resources; Related products)
and on the popup's recognition-language list. The three options of the options page and the popup's three switches are `.opt`
rows (the sliding switches are gone); local help exists for automatic reflow and for remote images. The popup is a task panel:
one primary command (Open a document, content width), Settings is a button, outside links keep link semantics and the external
marker, and a disabled site switch says why and how to enable it. Reset and an overwriting import ask in the product's dialog.
About holds the whole set - description, running version from the manifest, licence and its text, privacy, help, product site,
support, the clipboard summary and a saved diagnostics file, update sources, the repository, a resource index with each
component's licence, the third-party notices and the related products - from one module (`product-inventory.js`); opening it
requests nothing. The in-page OCR bar injected into other sites takes the same tokens under `--dht-*` on the bar itself, the
saved size and controls (read once, hidden until then), logical properties for a right-to-left interface language, and its own
dark surface; nothing a page's rules say changes it and nothing it says changes the page.
## 4. Permanent variants (design choices, not exceptions)

- The drop zone is a drop target (section 3).
- Theme and interface language are two labelled editors of the Interface page, once each (owner decision of
  2026-10-28; `APP-SETTINGS` rule 2 and `SETTINGS-NAVIGATION` 3.1 name the first page, so this is a deviation to
  record in the catalog registry).
- Compact mode keeps captions (section 2).
- The rail is monochrome; no identity-colour table exists.
- The packaged (Microsoft Store) build has no Windows page: it registers nothing in Windows itself.
- The log and the command line stay out of the theme and out of text enlargement (they are a console);
  they follow the interface scale.
- Viewer notices (X1): an inline offer (the resume offer, the remote-images notice) is a status in the flow above the page, not a
  dialog - it takes no focus and has no Escape; the export card is the one dialog-like surface, so it holds the default focus on
  Cancel and Escape is its Cancel. The remote-images notice has no primary (the blocked state is the safe default; neither
  answer is recommended). The notices' own type is the chrome's 14 px at the ordinary scale, so the reader's A+ / A- no longer
  enlarges them (it used to: they inherited the body's `--reader-size`, 28 px by default).
- Browser extension: the popup is a task panel without a rail (it edits the same keys the options page does); the options page has six
  destinations, not the launcher's seven (no conversion run, Ollama or Windows page); the control border is the muted ink, not the
  palette's quiet border; the in-page bar keeps a dark surface of its own over any page.
- Reader chrome: Fit width and the file buttons are words; the version link is a borderless text link with the target
  height; the buttons are all secondary (transparent face, the theme's border) and only the index toolbar's Continue
  reading is primary; the bar is an ordinary Tab row.

Owner decisions of 2026-10-30 (ticket 119, questions Q4 to Q6):

- **No size selector in the reader** (`BUTTONS` 8 rule 1: exposing a selector is a product decision). The reader chrome
  ships the ordinary scale only; the clamp of section 3 is what a later selector would inherit. Review: if a reader
  settings surface appears.
- **The quiet button border of the shared reading palette stays** (1.2 to 1.3:1 against the bar). The reader's buttons
  are secondary with a transparent face; `BUTTONS` 3 rule 6 names the text (4.5:1) and meaningful-glyph (3:1) pairs,
  and every measured pair passes (section 5). Review: if a reader button takes a filled face or the contract names a
  border pair.
- **The extension's help route is its existing "Guide & help" link** - in the popup and as Help on the options page's
  About - to the product site's extension guide (`extension.html`). The extension has no help pages of its own, no
  surface `?` button and no F1 (`CONTEXT-HELP` names the desktop launcher; ticket 118 decision 5). Review: if the owner
  orders help pages for the extension.

Owner decisions of 2026-11-10 (ticket 119, questions Q8 to Q10, slice Z):

- **Custom Ctrl+wheel zoom removed in the Go reader** (Q8; `BUTTONS` 4 rule 7: key and gesture bindings must not override
  the browser's; Go and extension editions converge, chrome targets keep their CSS size, `dht_zoom` sessionStorage and `?z=`
  parameters are retired; `Fit width` stays per-session via sessionStorage `dht_fitw` and `?w=`).
- **Touch naming route for glyph-only buttons** (Q9; `BUTTONS` 8 rule 3): recorded as a dated exception to 2026-12-31 with
  remediation ticket 119 and catalog proposal `buttons/PROPOSAL-2026-11-10-web-reading-page-touch-naming.md` asking whether
  an accessible name (`aria-label`) and tooltip (`title`) suffice on a dedicated web reading page; if declined, a long-press
  discovery route will be built in both editions.
- **Reader help entry** (Q10; ticket 118 decision 5): confirmed none; the reader keeps no help entry, and the version
  link opens documentation on the product site.

Recorded by ticket 120 (2026-10-20, contract sync of 2026-10-17), re-read by ticket E0126 against the catalog decisions of 2026-10-29:

- **Escape never closes the launcher** (`APP-SETTINGS` rule 9). The settings surface is the product's main window, so Escape steps
  back one layer (the help bubble or the page drawer, then cancels a run, then does nothing); the close box and Alt+F4 close it. Review:
  if a separate settings window or a tray surface appears. Since `APP-SETTINGS` 0.8 this is rule 9 as written, not a variant.
- **No settings search** (`WINDOWS-UI` 3.5, `APP-SETTINGS` 8). The launcher has seven pages and about thirty editors, each page named by its
  purpose and one rail click away, so every setting is reachable in one page change from a named destination. Review: if the owner reads
  that as "large", or a page grows past one screen of editors at the ordinary scale. Since `WINDOWS-UI` 0.11 section 16 B this is the
  definition of "readily discoverable", not a variant.
- **The window rectangle is written after the window settles, and flushed once on `pagehide`** (`APP-BEHAVIOUR` rule 10, 0.14). A
  browser-hosted window has no reliable close event; the page writes the rectangle 800 ms after the last move or resize (one write per burst, never per move event), a
  maximized or full-screen box never overwrites the remembered normal rectangle (`windowFillsScreen`), and a rectangle that no longer lands
  on a display is clamped to the nearest one; on `pagehide` it sends the settings once more with `keepalive`. Since 0.14 this is the rule
  for a browser-hosted window, not a variant.
- **The notice dialog's single answer is the text-only word "OK"** (`ICON-SET` 0.37 section 22 E; `BUTTONS` section 21 permits a
  text-only answer). "OK" is the universal dialog answer and the contract's `status.ok` name, and beside the check of `action.confirm`
  (named "Apply") it is a finding, so the answer draws no glyph (ticket E0122, Q4, 2026-10-30; this replaced the permanent variant of
  decision D6 of ticket 119). The secret-file dialog's "Open" keeps its `action.confirm` glyph.
- **Settings destinations show the glyph of the nearest meaning under the destination's own topic name** (`ICON-SET` 2.1 and 2.3;
  `SETTINGS-NAVIGATION` 2.2; decision D4 of ticket 119): the rail and the extension options rail. The catalog has no per-destination
  meanings and no menu or drawer meaning, so the "Pages" drawer button borrows `nav.contents`. Monochrome throughout. Review: when the
  catalog adds a drawer meaning or per-destination rules.

## 5. Proof

`node tools/uicheck/run.mjs` (see [../tools/uicheck/README.md](../tools/uicheck/README.md)) builds the two
executables and measures the real window in a headless Chromium under a scratch profile: geometry of every
command button at the declared scales and profiles (target, radius, content width, row height, gap, clipping,
overlap, state width), dialogs (order, neutral face, safe default, Escape, focus return), options (activation
regions, help versus toggle, keyboard, contrast), groups (activation, marker, anchor, focus), navigation
(order, keyboard, no resize, edit preserved, no operation, stale result ignored, nothing remote on opening
About), a boot check for a flash of the wrong language, and page-switch timings with every raw sample. The
static pins live in `cmd/doc-html-ui/*_test.go`. The reader is measured by `--only reader`: it converts a Markdown
book and a CBZ comic with the real CLI (single page, multipage chapter page and index, the comic with and without an OCR
layer, fixtures in five interface languages), opens them from disk, and serves the real `extension/` folder over loopback
with a stub of the few `chrome.*` calls the viewer makes at load, then measures every control of both editions in a fine
and a coarse pointer context and a narrow window, in the four reading themes, forced colours, `prefers-contrast: more`,
Arabic (RTL), German, Russian and Hindi, and at interface scale 0.875 / 1 / 2 (and 0 and 100, which clamp): target,
radius, content width, clipping, row height, the gap tokens, flow from the start, overlap, glyph tier, type scale, the
accessible names, contrast of caption and glyph idle and hovered, the selected state and its non-colour cue, keyboard
ring and order, panels not covered by a wrapped bar (also at the reader's own page zoom of 50 and 200 percent), the external
marker, the Search chord in the tooltip, and - on the generated reader - eleven negative controls that break the page on
purpose and must be caught. The extension's own pages are measured by `--only extension` the same way: the real `extension/` folder over loopback with a stub of the
`chrome.*` calls the pages make, the options page (every destination) and the popup at the pointer and touch profiles, scales 0.875 /
1 / 1.25 / 1.5 / 2, text enlargement, English, Russian, German, Hindi and Arabic, light, dark, forced colours and a narrow window:
target, radius, content width, clipping, row height, gaps, overlap, horizontal scroll, contrast of the actual text, glyph, border and
checkbox pairs, the activation regions and local help of the options, the groups, the rail (order, keyboard, nothing written but the
remembered page, edits preserved, nothing remote on About, the page, groups and viewport remembered, the drawer), the reset and import
dialogs, activation-to-frame timings with every raw sample, the OCR bar injected into a plain page and into a hostile one, and 27 negative
controls that break a page on purpose and must be caught. What is **not** proven by an agent: a physical
100 / 150 / 200 percent display, operating-system high contrast, a real touch screen and a screen reader - they ride the
owner's pending pass (ticket 93).

## 6. Still owed

From the browser extension (slice X1): the viewer's password prompt
(the Unlock rank follows the notice rule but no encrypted PDF was measured) and the little-text banners of the EPUB and PDF paths (same
rule, not opened in the harness), the OCR language and download rows measured with a real
download (the harness stubs the engine), the extensionless licence files linked from About (a real Chrome may offer them as a download
instead of showing them), a real installed extension in Chrome or Edge, and the ten machine-translated locales' review of the new strings;
from the reader chrome (R1) the owner's decision on the Ctrl+wheel page zoom (Q8, resolved: custom zoom removed in slice Z), the touch naming route for glyph-only buttons (Q9, dated exception to 2026-12-31 and proposal filed), the reader's help route (Q10, confirmed none; version link opens documentation on site), and the viewer's toolbar measured
on PDF, EPUB and comic loads (only an FB2 document and a stubbed `chrome.*` were measured; the notices were opened on a CBZ,
a Markdown book and a broken PDF); the site's
remaining items (the re-vendor of the catalog's current kit `4aba1dee`, which answered the kit-owned gaps of
`product-web-pages/PROPOSAL-2026-10-20-doc-html-translate-kit-ui-contract-gaps.md` on 2026-10-29 as `PAGE-STYLE` 3.3, with
the overrides it makes unnecessary dropped from `assets/site.css`, and the theme control's visible label and its
vocabulary glyph; the header button below 480 px follows the kit's recorded header policy - by the owner's decision of
2026-10-30 the re-vendor rides the next release's pre-flight together with `node tools/helpgen/generate.mjs` and
`node tools/uicheck/run.mjs --only site`), the owner's physical pass, the
ten machine-translated locales' review of the new strings, and the closing proof of the catalog's adoption rows, which
already describe all four surfaces as `partial`. The final date of the plan is 2026-12-31.
