# One glyph and one name per meaning - adopt the shared icon vocabulary

**Status:** Draft
**Priority:** 52
**Date:** 2026-09-23

> Contract sync ticket, both directions.
> Contracts: `ICON-SET` 0.10, `ICON-RENDER` 0.10, `ICON-EXTERNAL` 0.9 (domain `iconography/` of the shared
> catalog, owner FastMediaSorter Android, all drafts). No pointer in [`docs/contracts/`](../../docs/contracts/) yet.

> **Remote execution (2026-09-25):** the contract text this ticket needs is quoted in "Contract snapshot" below, so every step not marked ⛔ runs in a cloud session from this repository alone. Steps marked **⛔ Local only** edit the shared contracts catalog (or another repository) and can run only on the owner's machine, where the catalog is mounted.

> **Re-verified 2026-09-25** against the catalog and this tree: the contracts now stand at `ICON-SET` 0.13,
> `ICON-RENDER` 0.11, `ICON-EXTERNAL` 0.9 (the header above names the 2026-09-23 versions; 0.11-0.13 were
> additive - `nav.scroll-top` turned active, `action.save` / `action.export` turned active, references
> moved). Still true: no `docs/contracts/ICON-*.md` pointer exists, the registry (`_meta/REGISTRY.md` §2 and
> §3) has no doc-html-translate row for any of the three ids, and nothing in the code names a vocabulary id.
> The founding defect is still in the code: `internal/htmlgen/navbar.go:579-586` draws `&#9664;` / `&#9654;`
> around the prev/next labels, and `internal/htmlgen/htmlgen.go:86,89` hard-codes `#1a0dab`. Several
> Direction B items were meanwhile raised by other products' proposals; each B item below says which.

## What / why

The catalog now carries one vocabulary for every icon in the portfolio: one glyph and one canonical name
per meaning, on every surface - windows, controls, tiles, shortcuts, documentation and web pages. The
founding example is exactly the defect this product ships: "Back" and "Previous" are different meanings
with different glyphs.

This product has **no icon set at all**. Every glyph is a Unicode character typed into markup (`◀ ▶ ☰ ▤ ☀
◑ ☾ ● ▸ ▾ ⤓ ⇄ ↓ ↗ ↑ ◐ ✓ ←`), none is drawn on the contract's grid, and several are, by shape, another
meaning's glyph. The product is not in the contract's consumer list and has no registry row.

A draft binds only the products that opted in, and any product may supplement a draft with what its own
function needs (`VERSIONING.md` §1). This ticket opts in, maps every glyph, and sends back the meanings the
vocabulary lacks.

Found by reading the code on 2026-09-23; nothing in this ticket was verified in a rendered browser.

## Findings that drive the work

**Founding-rule breach in the reading surface** (multi-page output, `-multipage`):
- the previous-page link is labelled with `nav.back`'s name ("Back" / "Назад") and drawn `◀`, a shape that
  belongs to no meaning - it should be `media.previous`, "Previous page";
- the next-page link is labelled "Forward" (`nav.forward`'s name) and drawn `▶`, the shape of `media.play`
  / `action.run` - it should be `media.next`;
- the 13 locales translate the one key inconsistently (hi/bn/zh already say "previous", ar/ru/uk/de say
  "back").
  Where: `internal/htmlgen/navbar.go` (`buildNavBarHTML`), `internal/i18n/i18n_reader.go`.

**Wrong-shape or clashing glyphs:**
- `☰` "Contents" is a bullet-less hamburger; `nav.contents` is lines with bullets, and its ru name is
  "Содержание" while both editions say "Оглавление" (Go reader bar, extension viewer);
- theme options: `☀` is `weather.clear`, `☾` is `app.night-mode` but sits on "Dark" while "Night" gets `●`,
  which is `camera.record-video` (`navbar.go`, `readerControlsHTML`);
- `▸` "Continue reading" should be `feature.continue-reading` (index toolbar, `htmlgen.go`);
- `⤓` means `nav.scroll-bottom` by shape, used for "drop a document" (GUI) and "install from a store"
  (`extension.html`);
- `↓ File` / `↓ HTML` in the extension viewer: `↓` is `action.move-down`; the meaning is `action.save` /
  `action.export`, and the glyph is baked into 13 `_locales/*/messages.json` strings;
- `↑` "Back to top" on the site: shape is `action.move-up`, the word "Back" is `nav.back`'s; the meaning
  is `nav.scroll-top` (proposed in the vocabulary);
- `← Desktop app` on `extension.html` is a cross-link (`nav.go-to`), not `nav.back`;
- `✓ Copied`, `↗` external link (arrow without the square of `nav.open-external`), `▸/▾` disclosure
  markers in the GUI, the extension TOC and `extension.html`.

**Render rules:**
- the index TOC `summary` colour is hard-coded `#1a0dab` - about 1.5:1 on the dark and night reader
  themes (hand-computed), against the 3:1 floor (`ICON-RENDER` rule 3, `htmlgen.go`);
- glyph-only controls take their accessible name from the glyph (`A−`, `A+`, `▤`, `⇄`) or carry an
  English-only `aria-label` (extension viewer) - `ICON-RENDER` rule 8;
- the OCR-layer toggle `▤` shows one form for both states (rule 4);
- disclosure `▸` does not mirror under `dir=rtl` in the GUI (rule 7);
- touch targets under 44 px: bar links about 26 px, site `.theme-btn` 32 px, `.to-top` 42 px (rule 5 - the
  value itself is disputed below).

**System surfaces:** the `.ico` and extension action icons are the text "DOC HTML" on navy (product
artwork, out of scope by rule 7); the MSIX 44x44 tile shows "DH", unreadable at 16 px, with no unplated or
light-theme variant (`msix/build-msix.ps1`). The file-type icon and the "Convert to HTML" verb icon reuse
the exe icon (`internal/windowsreg`).

**`ICON-EXTERNAL`** holds by absence: no third-party mark is drawn anywhere; store channels are text.
Adopting Material-derived SVGs would bring Apache-2.0 attribution into a repo that ships only an MIT
`LICENSE` (rule 5).

## Direction A - the product conforms

1. Map every glyph to a vocabulary id: one table, kept in the repo as this product's rung-2 record,
   covering the generated book, the GUI, the extension (viewer, popup, options) and the site.
2. Book reader chrome: previous/next become `media.previous` / `media.next` with the canonical names in
   all 13 languages; contents, continue-reading, expand/collapse and the theme options take their
   vocabulary glyphs; the index TOC colour comes from the theme variables.
   - Previous/next, contents, continue-reading, expand/collapse and the index TOC colour use existing
     meanings (snapshot below) and run remotely. The en/ru/uk names are in the vocabulary table; the other
     ten languages are this product's own translation of `name.en` until B2 lands (rule 3).
   - **⛔ Waits on B1 (local).** The theme options (theme choice: light / sepia / dark / night) - the
     vocabulary has no theme meaning; `☀` / `☾` / `●` are `weather.clear` / `app.night-mode` /
     `camera.record-video`, so no existing glyph may be borrowed for them.
3. Every glyph-only control carries its canonical name, localized, as the accessible name - both editions.
   Remote for every control whose meaning exists (prev/next, contents, swap `⇄` -> `action.swap`, ..).
   **⛔ Waits on B1 (local)** for text smaller / larger (`A−` / `A+`), the OCR text layer (`▤`), the
   theme options and the page-jump select (`dht-page-sel`, jump to page - "Go to" is `nav.go-to`'s name):
   rule 8 asks for the record's name, and those records do not exist yet.
4. **⛔ Waits on B1 (local).** The OCR-layer toggle shows its state - needs the text-layer meaning with its
   shown/hidden `states`.
5. Extension viewer and popup: the same meanings, the same glyphs, the same names as the Go edition (a
   cross-edition parity row in [`docs/PARITY.md`](../../docs/PARITY.md), guarded by a test). Remote for
   the existing meanings; the parts that wait on B1 in items 2-4 wait here too.
6. GUI: the four glyphs it uses take their vocabulary shapes; disclosure mirrors in RTL. Disclosure takes
   `nav.expand` / `nav.collapse` (both `rtl: fixed` - see B11 on the mirrored `▸`). **⛔ Waits on B1
   (local)** for the drop-a-document glyph (`⤓`, today `nav.scroll-bottom`'s shape) unless A1 maps it to
   an existing meaning; the convert meaning is one of B1's.
7. Site: to-top, copied, store/download, cross-link and disclosure glyphs from the vocabulary - after the
   `PAGE-STYLE` §9 conflict (B9) is settled.
   - Re-verified 2026-09-25: B9 need not block the parts that have a meaning. `PAGE-STYLE` §9 (quoted
     below) *prefers* inline SVG and only *permits* `◐ ⤓ → ▸`; FastMediaSorter_Lite drew `nav.scroll-top`
     and `action.download` on its kit page on that reading (catalog
     `iconography/PROPOSAL-2026-09-24-fms-page-style-kit-symbols.md`, "Update 2026-09-24"). So to-top
     (`nav.scroll-top`), the cross-link (`nav.go-to`) and disclosure (`nav.expand` / `nav.collapse`) run
     remotely - in this repo's own markup, not in the vendored kit stylesheet `assets/sza-kit.css`
     (`details.sec>summary::after{content:"▸"}`), which belongs to the kit.
   - **⛔ Waits on B1 (local).** Copied (a `done` state of `action.copy`; `✓` is `action.confirm`'s shape)
     and install-from-store (unless the owner rules `action.download` covers it).
8. Glyphs leave translatable strings (`↓ File` -> glyph in markup, word in the message).
9. The source and licence of every copied glyph is on record (`ICON-EXTERNAL` rule 5), in a
   third-party-notices file that ships where the glyphs ship. Re-verified 2026-09-25: every glyph this
   ticket maps is a `ref` record (exported from a FastMediaSorter Android drawable) and carries no
   `glyph.source`, so the catalog does not state its licence - see the licence note in
   [`glyphs-snapshot-2026-09-25/README.md`](24_2026-09-23_contract-iconography-sync/glyphs-snapshot-2026-09-25/README.md).
   Settling it may need a question to the owner; asking is not a catalog edit.
10. A guard test pins the book chrome's glyph + name pairs, so the founding-rule breach cannot come back.

## Direction B - what the vocabulary needs from this product

Written as `PROPOSAL-2026-09-23-<topic>.md` in the catalog's `iconography/` folder, never as an edit.
Each item states the evidence above. Every item here is **⛔ Local only - changes the contract catalog.**
"Already raised" (re-verified 2026-09-25) names a proposal another product filed in the catalog's
`iconography/` folder: this product then co-signs it with its own evidence instead of filing a duplicate.

- **⛔ Local only - changes the contract catalog.** **B1 new meanings (MINOR):** theme choice (with levels light / sepia / dark / night, or one record per
  theme - settles the clash with `app.night-mode`); text smaller / larger, distinct from image zoom;
  OCR text layer shown/hidden, distinct from `action.extract-text`; convert a document to HTML (also the
  Windows shell verb); jump to page, since "Go to" is already `nav.go-to`'s name; a `done` state for
  `action.copy`; install-from-store, unless `action.download` is meant to cover it.
  Already raised, in part: theme - `app.theme` ("Theme" / "Тема" / "Тема", a half-filled circle,
  `PROPOSAL-2026-09-24-fms-windows-meanings.md` item 7, co-signed by
  `PROPOSAL-2026-09-24-streamsplayer-meanings.md` item 7) and `app.theme-toggle` ("Switch theme" /
  "Сменить тему" / "Змінити тему", `PROPOSAL-2026-09-23-sza-hub-controls.md` item 2) - neither has theme
  levels; text size - `action.text-settings` or a qualified `action.adjust`
  (`PROPOSAL-2026-09-24-fms-accessible-names.md` item 1, "the size and typeface of shown text"). Not
  raised by anyone: the OCR text layer, convert to HTML, jump to page, the `done` state of `action.copy`,
  install-from-store. (`action.reveal` with an `off` state, fms-windows-meanings item 4, is a password eye,
  not the text layer.)
- **⛔ Local only - changes the contract catalog.** **B2 ten new languages** (de it es fr pt ar hi bn ur zh) for every record this product ships - rule 3
  makes the first shipper of a language responsible for it. Already raised for exactly these ten
  languages: `PROPOSAL-2026-09-24-fms-names-beyond-three-languages.md` (FastMediaSorter_Lite, including the
  German `media.previous` "Zurück" -> "Vorherige" worked example) and
  `PROPOSAL-2026-09-24-streamsplayer-meanings.md` item 8 - contribute this product's names for the records
  it ships to that proposal.
- **⛔ Local only - changes the contract catalog.** **B3 RTL paging:** `media.previous` / `media.next` are `rtl: fixed`, but paging a document follows the
  reading direction in ar/ur. Possibly a split, and then MAJOR - propose, do not assume. Not raised;
  related only: `PROPOSAL-2026-09-24-fms-windows-rendering.md` item (f) - a `fixed` record is never written
  with a bidi-mirrored character.
- **⛔ Local only - changes the contract catalog.** **B4** `nav.scroll-top` / `nav.scroll-bottom` say "list"; widen to "page or list", ru name accordingly.
  Not raised (the hub already ships `nav.scroll-top` on a page, `ICON-SET` 0.11 log).
- **⛔ Local only - changes the contract catalog.** **B5** `source.local` is drawn as a phone with an Android-only rationale - ask for a desktop variant.
  Not raised.
- **⛔ Local only - changes the contract catalog.** **B6** `app.logo` is one product's mark inside the portfolio vocabulary, while rule 7 puts product marks
  out of scope - namespace it or remove it. Already raised: `PROPOSAL-2026-09-23-filedo-meanings.md` §3
  (`means` to read "the mark of the product showing it").
- **⛔ Local only - changes the contract catalog.** **B7** `palette.json` `state.warning` has day contrast 2.7, below the 3:1 floor the contract demands.
  Not raised; related: `PROPOSAL-2026-09-23-windows-desktop.md` item 3 (do the `state.*` hues bind exact
  tones or the hue family).
- **⛔ Local only - changes the contract catalog.** **B8** 44 px touch target on Windows and the web contradicts dense mouse-first toolbars and the
  portfolio's own kit (`.theme-btn` 32 px): propose 44 px for coarse pointers and a smaller floor for
  fine pointers. Already raised: `PROPOSAL-2026-09-24-fms-windows-rendering.md` item (c) (44 px binds
  touch surfaces; a pointer-first surface takes the platform's published floor). Co-sign and add the web.
- **⛔ Local only - changes the contract catalog.** **B9 cross-contract conflict:** `PAGE-STYLE` §9 recommends the kit glyphs `◐ ⤓ → ▸`; `ICON-SET` gives
  `⤓` to `nav.scroll-bottom` and `▸` to nothing. One proposal to both owners (`iconography/` and
  `product-web-pages/`). Linked from [`26_2026-09-23_contract-product-web-pages-sync`](26_2026-09-23_contract-product-web-pages-sync.md).
  Already raised twice: `PROPOSAL-2026-09-23-filedo-scope.md` §2 (give the kit glyphs ids or declare them
  outside the vocabulary; `▸` rotating is `nav.go-to`'s shape used for `nav.expand`) and
  `PROPOSAL-2026-09-24-fms-page-style-kit-symbols.md`, whose 2026-09-24 update narrows the open part to the
  theme switch `◐` alone (see A7). Co-sign rather than file a third.
- **⛔ Local only - changes the contract catalog.** **B10** Windows system surfaces missing from rule 9: file-type icon, shell-verb icon, MSIX unplated and
  light-theme assets, browser-extension action icon. Already raised except the extension action icon:
  `PROPOSAL-2026-09-23-windows-desktop.md` item 2 (MSIX `Square44x44Logo` `targetsize-*` with
  `altform-unplated` / `altform-lightunplated`, ICO sizes, a 16 px context-menu verb icon, a
  document-type icon). Add the browser-extension action icon to it.
- **⛔ Local only - changes the contract catalog.** **B11** platform-native disclosure markers (`<details>` marker): a declared platform shape or forbidden.
  Related, not the same question: `PROPOSAL-2026-09-24-fms-windows-meanings.md` item 9 (a `nav.disclosure`
  record with two states, if `nav.expand` / `nav.collapse` are actions only) and
  `PROPOSAL-2026-09-24-streamsplayer-meanings.md` question 9 (platform control chrome out of scope?).
- **⛔ Local only - changes the contract catalog.** **B12** `ICON-EXTERNAL` rules 3-4: are a converted book's embedded images "downloaded pictures", and what
  is the fallback for a missing one. Not raised.
- **⛔ Local only - changes the contract catalog.** **B13** ru name of the table of contents: the vocabulary says "Содержание", book readers and both
  editions here say "Оглавление" - propose or comply, owner's call (open question 2). Only the "propose"
  branch is local; complying is an in-repo string change. Not raised.

## Done criteria

- [ ] Pointer files `docs/contracts/ICON-SET.md`, `ICON-RENDER.md`, `ICON-EXTERNAL.md` exist and are listed
      in [`docs/contracts/README.md`](../../docs/contracts/README.md).
- [ ] **⛔ Local only - changes the contract catalog.** Registry: this product's adoption rows for the three ids, each deviation still open recorded as a
      dated exception with an `until` date.
- [ ] **⛔ Local only - changes the contract catalog.** The proposals B1-B12 are filed in the catalog folder (or each is withdrawn in writing here with the
      reason), and the book chrome ships no meaning the vocabulary lacks (rule 5: the vocabulary moves
      first). The second half is checkable remotely: until B1 lands, the chrome keeps its current theme,
      text-size and text-layer controls under a registry exception rather than gaining new glyphs.
- [ ] The glyph map (A1) exists and every row is `conforms`, `exception` or `proposal filed`.
- [ ] Multi-page output: previous/next are `media.previous` / `media.next` in glyph and name in all 13
      languages, pinned by a test.
- [ ] No glyph-only control on any surface takes its accessible name from the glyph character.
      **⛔ Waits on B1 (local)** for the text-size, text-layer, theme and page-jump controls (see A3);
      the rest is remote.
- [ ] Index TOC text passes 3:1 on all four reader themes.
- [ ] Both editions show the same glyph for the same meaning, guarded by a parity test.
- [ ] Glyph sources and licences are on record.
- [ ] Site and `README*` changes land in every authored locale in one edit (canon invariant 17).

## Open questions

1. **⛔ Local only - changes the contract catalog.** Target conformance rung (`iconography/README.md` §6, quoted below): rung 2 (mapped) now, rung 3 (label vs glyph
   check) with this ticket, rung 4 (theme contrast) later? The decision is the owner's; the rungs held are
   stated in the registry row, which is the local part.
2. **⛔ Local only - changes the contract catalog.** Toc name in ru: comply with "Содержание" or propose "Оглавление"? (Only the "propose" branch is
   local; complying is an in-repo string change.)
3. Rendering in the self-contained offline book: inline SVG per page, one CSS `mask` data-URI per page in
   the reader stylesheet, or keep text characters and propose that the contract allow them. Size cost
   multiplies by hundreds of chapter pages. The first two options can be measured remotely from the
   byte-exact copies in
   [`glyphs-snapshot-2026-09-25/`](24_2026-09-23_contract-iconography-sync/glyphs-snapshot-2026-09-25/)
   (141-533 bytes each). **⛔ Local only - changes the contract catalog** for the third option (the
   proposal).
4. Keep text labels beside the glyphs in the book bar (rule 3 allows "Previous page")?
5. App mark: design a glyph-based mark readable at 16 px with a monochrome/unplated variant, or declare
   the current artwork and leave it?
6. Site feature cards (READ / TRANSLATE / OCR / LOCAL): take the decorated look of `ICON-RENDER` §10 C or
   stay text pills?

## Notes

Order inside the work: file the B1 proposals first, because the reader chrome uses meanings (theme, text
size, text layer) that do not exist yet - shipping them before the vocabulary is the exact violation rule
5 names. Direction A items 2-6 then land in one cross-edition change. `navbar.go` (818 lines) is over the
file budget; the glyph constants belong in a file of their own.

Split for remote execution (2026-09-25): filing B1-B13 and the registry rows is **⛔ Local only**. A remote
session can do, from this file and the snapshot alone: the pointer files, the glyph map (A1), the prev/next
fix with its guard test (A2 first bullet, A10), accessible names for existing meanings (A3), the index TOC
colour (A2), the extension parity for those meanings (A5), GUI disclosure (A6), the site parts listed in
A7, glyphs out of translatable strings (A8) and the notices file (A9). The one cross-edition change the
paragraph above asks for can then be two: the existing-meaning half now, the B1 half after the vocabulary
moves.

## Contract snapshot (2026-09-25)

Quoted from the catalog on 2026-09-25 - `ICON-SET` 0.13, `ICON-RENDER` 0.11, `ICON-EXTERNAL` 0.9, `PAGE-STYLE` 1.1 (§9 only). A working copy for executing this ticket without the catalog, not a second source: the catalog stays authoritative, and this section is deleted when the ticket moves to done/.

Source files: the shared contracts catalog, `iconography/README.md` (sections 2, 3, 4, 6 and 10),
`iconography/vocabulary.jsonl`, `iconography/palette.json`, `product-web-pages/PAGE-STYLE.md` §9. Quotes are
verbatim; markdown links inside them are reduced to their code text. The SVG sources of every glyph in the
vocabulary table are copied byte for byte into
[`glyphs-snapshot-2026-09-25/`](24_2026-09-23_contract-iconography-sync/glyphs-snapshot-2026-09-25/)
(29 files, with a README carrying source, versions and the licence note).

### `ICON-SET` 0.13 - section 2, the vocabulary

A record in `vocabulary.jsonl` carries: `id` (`group.name`), `group`, `status`, `name` (`en`, `ru`, `uk`),
`means` (one sentence), `distinct` (the meanings it must never be confused with), `shape` (the picture in
plain words, so an implementation without the reference files can still recognize or draw it), `rtl`,
`colour` (its role, section 3), and either `ref` (the reference drawable its glyph is exported from) or
`glyph` (the drawing itself with its `source`). Optional: `states`, `levels`, `variants`, `sharedWith` with
`sharedWhy`, `note`.

1. **One meaning, one glyph.** A product draws a meaning in the vocabulary with that meaning's glyph and
   nothing else, and never uses that glyph for any other meaning. "The same glyph" means the same picture
   in the words of `shape` - the product's own drawing may differ by a pixel, never by a feature a user
   would read. *Observe:* put the product's icon for a meaning beside `glyphs/<id>.svg`; a user names both
   the same. *From:* the owner's founding example, 2026-09-23, and `APP-STYLE` rule 5.
2. **The confusable meanings are the point of the list.** Each record's `distinct` names the meanings a
   designer is most tempted to reuse its glyph for. These sets are binding, and the most important of them
   are spelled out here so they are read even by someone who never opens the file:
   - *Leaving and stepping back:* `nav.back` (leave this screen), `media.previous` (the previous file,
     track, page, chapter or match), `media.rewind` (jump back in time inside the item), `media.to-start`
     (go to the first page), `action.undo` (take back a change), `nav.exit` (leave the app or a mode).
   - *Going on:* `nav.forward` (next step of a flow), `media.next` (next item), `nav.go-to` (this row opens
     a screen), `action.send-to` (send to a destination).
   - *Ending something:* `nav.close` (close a panel - nothing is lost), `action.cancel` (abandon the
     operation or the edit), `media.stop` (stop playback, recording or a broadcast), `media.exit-fullscreen`,
     `action.delete` (destroy the file), `action.remove` (take it off this list - the file stays),
     `action.clear-all` (empty a whole list).
   - *Starting over:* `action.refresh` (load again), `action.reset` (back to the shipped defaults),
     `action.sync` (bring two copies level), `action.undo`.
   - *Moving things out and in:* `action.share` (the system share sheet), `action.send-to` (the product's
     own destinations), `action.export` / `action.import` (to and from a file), `action.upload` /
     `action.download` (to and from a server), `nav.open-external` (open in another window or app).
   - *Three kinds of "broadcast":* `content.stream` (a live channel the app receives), `media.cast` (show
     this item on a TV), `feature.live-broadcast` (send this device's own camera, microphone or screen out).
   *Observe:* no two meanings in one `distinct` set share a glyph in any product. *From:* the conflicts
   found in the reference product on 2026-09-23, recorded in the registry.
3. **One name per meaning, in every surface.** The interface label, the tooltip, the accessible name, the
   manual and the web page call a meaning by its `name` in that language. A label may **qualify** the name
   with its object - "Previous page", "Предыдущая страница" - but never **substitute** the word of another
   meaning: "Назад" is `nav.back`, so it can never be the Russian for Previous. A language beyond the
   three authored ones translates `name.en` and is added to the record by the first product that ships
   it. *Observe:* search the product's strings and pages for each name - every hit shows that meaning's
   glyph, and no control showing the glyph carries another meaning's name. *From:* the reference product,
   whose Russian "Previous" read "Назад" and whose Russian "Cast to.." read the same word as "Streams".
4. **A knowingly shared shape is declared on both sides, with the reason.** Where two meanings rightly use
   one picture - the cross that closes a panel and clears a text field, the play triangle that also runs a
   task - the records name each other in `sharedWith` and one of them says why in `sharedWhy`. An undeclared
   share is a violation, and the exporter refuses a vocabulary that draws two meanings identically without
   the declaration. *Observe:* the exporter exits 0.
5. **New development starts here.** A new control picks an existing meaning. A meaning the vocabulary does
   not have is added to it - glyph, three names, `distinct` set - **before** the code that shows it ships,
   as a MINOR amendment (`../_meta/RULES.md` §6). A product never invents a private
   glyph for a meaning the vocabulary already has. *Observe:* every icon in a new release maps to an `id`.
6. **`proposed` is a decision, not a wish.** A record with `status: proposed` has a decided glyph that no
   product ships yet - drawn in the record, from Material Icons or for this contract. The first product to
   ship it turns it `active` by amendment and may replace the drawing with its own file if that file keeps
   the `shape`. *Observe:* `CATALOG.md` marks each proposed record.
7. **Product-private artwork is not a glyph.** Illustrations, onboarding pictures, game figures, splash
   art, the product's own logo animation and a set of decorative icons a user picks for their own items
   (FastMediaSorter's resource icon picker) are outside the vocabulary. They are never used to stand for a
   meaning, and a meaning is never drawn with them. *Observe:* no control's glyph is taken from such a set.
8. **Documentation and the site show the glyph, not a substitute.** Where a page refers to a control, it
   shows the meaning's glyph (the file in `glyphs/` or the product's own copy of it) beside the meaning's
   name - never an emoji, never a different picture chosen to decorate the page, never a shape word that
   the glyph does not have ("the gear icon" only for `app.settings`, which is a gear). *Observe:* read the
   page beside the product. *From:* `PAGE-STYLE` section 9, which asks for exactly these SVGs.

### `ICON-RENDER` 0.11 - section 3, how a glyph is drawn

1. **One grid, one style.** The design size is 24 x 24 units; a glyph is a single-colour filled silhouette
   in the family of Material Icons (filled), which is the base of the vocabulary. An outlined form appears
   only as the declared "off" state of a toggle (`action.favorite--off`, `action.pin--off`) or as a declared
   `variant`. Multi-coloured, shaded or three-dimensional drawings are not glyphs. *Observe:* every file in
   `glyphs/` except the `brand` and `fixed` records paints with `currentColor` only. Amended by section 10,
   items A and B: the style is measured, and a coloured or decorated look is derived from the glyph.
2. **Colour comes from the theme, never from the file.** A glyph takes its colour from the role its record
   names, resolved in the theme the user chose at the moment it is drawn:
   - `content` - the colour of text on that surface (on-surface);
   - `accent` - the theme's primary colour;
   - `category` - the content-category colour of the item's kind (music, video, image, document, other),
     which each theme defines in its own shade;
   - `state` - the success, warning, error or neutral role;
   - `brand` and `fixed` - the literal colours of the drawing, never tinted.
   A source file that bakes white or black into a `content` glyph is a defect even where it happens to look
   right, because it disappears on the opposite theme. *Observe:* switch the product between its themes;
   every glyph stays visible and keeps its role colour.
3. **Every theme a product ships, not two.** A product that offers its own themes beyond light and dark -
   FastMediaSorter ships six accent themes - owes each glyph legibility on each of them: a contrast of at
   least 3 : 1 between the glyph and the surface under it (WCAG 2.1 success criterion 1.4.11). `gallery.html`
   shows the reference product's themes read from its own resources at generation time. *Observe:* the
   gallery, theme by theme.
4. **A toggle shows its state; a live transport control shows its action.** A record's `states` are the
   forms a toggle takes (`on`/`off`, `recording`, `charging`); a live play control shows `media.pause` while
   playing and `media.play` while paused. `media.play-pause` names the toggle only where a setting assigns it
   to a gesture or a key. *Observe:* toggle each control and compare with `glyphs/<id>--<state>.svg`.
5. **Sizes change the scale, never the drawing.** The tiers are 16 (inline in text), 20 (dense rows), 24
   (buttons, toolbars, menus - the design size), 32 to 48 (tiles, launcher cells, shortcuts, watch
   controls). A glyph is the same drawing at every tier; a second file for a smaller size keeps the shape.
   The touch target stays at least 48 dp on Android and 44 px on the web and on Windows whatever the glyph
   size. *Observe:* the gallery shows each glyph at 16, 24 and 48 px. Amended by section 10, item E: the tiers
   are 16, 20, 24, 32, 40 and 48, and a glyph on a plate is sized by the plate.
6. **Large shortcuts carry the same glyph on a plate.** A launcher cell, a home-screen widget, a watch
   tile or a desktop shortcut may put the glyph on a coloured plate - the plate takes the `accent` or
   `category` colour and the glyph the contrasting content colour - but the glyph inside keeps its shape.
   A feature that has a recognizable glyph (the calculator, the stopwatch, the camera) shows that glyph on
   every entry point to it: its tile, its widget, its shortcut, its settings row, its page in the manual.
7. **Right-to-left mirrors direction, never time.** A record marked `rtl: mirror` is mirrored in a
   right-to-left layout; `fixed` never is. Media transport is fixed (`APP-BEHAVIOUR` rule 8: a mirrored
   Play reads as Rewind). *Observe:* the gallery's RTL form for every `mirror` record.
8. **The accessible name is the canonical name.** A control whose only label is a glyph carries the
   record's `name` in the interface language as its accessible name (`contentDescription`, `aria-label`,
   `AutomationProperties.Name`) - `APP-BEHAVIOUR` rule 9, applied to every platform.
9. **System surfaces follow the platform's shape rules, with the product's own mark.**
   - The application icon carries the product mark on every edition that ships. On Android it is an
     adaptive icon - foreground inside the 66 of 108 dp safe zone, a background layer, and a `monochrome`
     layer so the themed icons of Android 13 and later work - on **every** edition; an edition may differ
     by colour or a badge, never by dropping the mark.
   - A notification small icon, a status-bar or tray icon is a single-colour, alpha-only silhouette.
   - A watch shows the same glyphs as the phone, at 24 to 32 dp on black glass, coloured from the watch's
     own palette by the same roles.

### `ICON-SET` / `ICON-RENDER` - section 10, amendment 2026-09-23 A (items A-E and G)

**A. `[CONTRACT]` The style is measured.** A glyph conforms when a machine says so, not an eye. Rendered on
its 24 x 24 grid:
- the viewport is 24 x 24;
- the ink stays inside the box from 1 to 23 on both axes - one unit of margin on every side;
- the centre of the ink's box or the centre of its mass lies within one unit of the grid centre (12, 12),
  unless the record declares that the offset is the shape (a level of a series, a horizon). The mass centre
  is what spares Material's optical corrections: the play triangle's box sits 1.4 units right, its mass
  0.3 units left;
- the estimated line weight - twice the inked area over the ink's perimeter - is at least 1.3 units (Material's
  own lettering glyphs, drawn 1.5 units wide, measure 1.34), and a part drawn as a stroke has a stroke width
  of 2;
- one paint, no gradient.
`scripts/docs/export-icon-contract.ps1` in the reference product measures every exported glyph and lists
each departure in `CATALOG.md`, "Style report".

**B. `[NEW]` Three looks of one glyph.** A meaning may be shown in three looks, and all three are the same
drawing:
- **mono** - the glyph in one colour: the theme's content colour, or on a toggle the colour of its state.
  The default everywhere;
- **colour** - the same glyph filled with its hue (item D);
- **decorated** - the same glyph, painted in the on-plate colour, centred on a flat plate of its hue.
A look is derived, never redrawn: a product that ships a look as its own file keeps the glyph's geometry
unchanged. Only a brand mark keeps its owner's colours (`ICON-EXTERNAL` rule 1). This is what makes the
three recognisably one icon - the same silhouette, the same hue.

**C. `[NEW]` Which surface takes which look.**
- **decorated** - an app shortcut icon (static or pinned), an in-app launch grid whose tiles stand for
  programs, places or sources, an onboarding feature row, a watch tile, a feature card on the site;
- **colour** - a leading icon whose job is to say what kind of thing a row is (a media type, a source type,
  a program) in a list, a panel, a chip or a legend, and a state indicator;
- **mono** - everything else: toolbars, menus, buttons, settings rows, player controls, tabs, the status
  bar and notification icons, quick-settings tiles, watch complications, icons inline in text.
One surface, one look: the siblings on a surface share it. A picture from outside - an installed app's
icon, a favicon, a thumbnail - is never re-looked (`ICON-EXTERNAL` rules 2 and 3).

**D. `[CONTRACT]` The palette.** A hue is named by a key:
- `category.image`, `category.video`, `category.audio`, `category.document`, `category.other`,
  `source.local`, `source.smb`, `source.sftp`, `source.ftp`, `source.cloud`, `state.ok`, `state.warning`
  and `state.error` are shared: every product shows the same kind of thing in the same hue;
- `accent` and any `program.*` tone belong to each product - its face, which the contract does not take.
Every hue has a day tone and a night tone, each at least 3 : 1 against the surface it is drawn on. A plate
on a system surface uses the day tone, because the plate brings its own background. The on-plate colour is
white wherever white reaches 3 : 1 against the plate, and `#1F1F1F` otherwise - one glyph colour across a
family of plates, dark only on a plate too light for white (amber, yellow). A `content` meaning shown in a colour or
decorated look takes `accent`. `palette.json` lists every key with its tones and on-plate colour, read from
the reference product's resources.

**E. `[CONTRACT]` Proportions.** The size tiers of section 3 rule 5 become 16, 20, 24, 32, 40 and 48; a bare
glyph larger than 48 is only a declared empty-state or hero picture. A glyph on a plate is sized by the
plate: its 24 grid spans 0.6 of the visible plate side, within 0.05 (24 on a 40 plate). The in-app plate is
a circle. A system surface keeps the platform's mask (section 3 rule 9): an Android adaptive icon lays a
full-bleed background in the plate colour under a 44 dp glyph in the 72 dp safe zone; an Android 7.1
shortcut is a 44 dp circle with a 24 dp glyph in a 48 dp asset.

**G. `[CONTRACT]` Illustrations are not glyphs.** Section 2 rule 7 is extended: a multi-coloured picture at
glyph size - an onboarding guide, a game figure, a control face such as a camera shutter - is an
illustration. It is declared in the product's own exceptions list with its reason and never stands for a
meaning of the vocabulary.

### `ICON-EXTERNAL` 0.9 - section 4, pictures from outside

1. **A third-party mark is the owner's asset, shown as its owner allows.** Google Drive, Dropbox, OneDrive,
   YouTube, Chromecast, Android, Gemini, a messenger - each is shown only as its owner's own mark, or its
   owner's published monochrome variant, at the size the owner's guidelines allow. It is never redrawn to
   look like the mark, never recoloured into a theme colour unless its owner publishes that variant, and
   never used to mean anything but that product. *Observe:* every `brand` record's glyph is the owner's mark
   or its published variant.
2. **Another installed app is shown by its own icon.** A launcher, a share target or a "send to" row
   showing another app uses the icon the operating system reports for it, cached as delivered and never
   tinted. When the icon cannot be read, `content.apps` stands in - not a drawing that imitates the app.
3. **Downloaded pictures are content, not glyphs.** Stream favicons and logos (`STREAM-BANK`), album art,
   cloud thumbnails and contact photos sit in the box a glyph of that size would occupy, scaled to fit and
   clipped, never stretched. They are never used as an action glyph.
4. **Every downloaded picture has a vocabulary fallback.** A picture that is missing, not yet loaded,
   refused or broken is replaced by a picture made from the item's own data where the product has one - a
   contact's initials on a colour derived from the name - and otherwise by the glyph of what it stands
   for: `content.stream`, `content.audio`, `content.image`, `content.video`, `content.person`,
   `content.apps`. Never a blank box and never an error glyph, because nothing is wrong from the user's
   side. *Observe:* disconnect the network and open each surface that shows downloaded pictures.
5. **The source of every glyph is on record.** A glyph exported from a product names its drawable in `ref`;
   a drawn one names its origin in `glyph.source` (Material Icons, Apache-2.0, or "drawn for this
   contract"). A product that copies a glyph copies its licence obligation with it.

### Section 6 - conformance (the ladder for open question 1)

The ladder, cheapest first. A product states in the registry which rungs it holds.

1. **The vocabulary is consistent.** `scripts/docs/export-icon-contract.ps1` in FastMediaSorter Android
   exits 0: unique ids, every `distinct` and `sharedWith` reference resolves, every enum value is known,
   every reference drawable exists and converts, and no two meanings draw the same glyph undeclared. Rerun
   it after every amendment; its output is byte-identical when nothing changed.
2. **The product's icons are inventoried against the vocabulary.** Every glyph the product shows maps to
   an `id`, or is declared product-private artwork (section 2 rule 7).
3. **Labels agree with glyphs.** A mechanical check: a control whose label is the name of meaning X shows
   X's glyph, and a control showing X's glyph is labelled with X's name.
4. **Themes agree with roles.** Every glyph measured against every theme surface at 3 : 1.
5. **The manual and the site agree.** Every page that names a control shows its glyph and its name.

### `PAGE-STYLE` 1.1 - §9 Emoji & icon policy (for B9 and A7)

- **No emoji.** Do not use emoji in headings, prose, buttons, badges, quickstart steps, or decorative UI.
- For a section or action, prefer a **small, single-colour inline SVG** with a familiar meaning. It must support the
  label, not replace it. Use one consistent stroke/weight family per page.
- A small product icon is appropriate when it is already recognisable inside the product (for example, its app icon or
  a familiar product symbol) **and has a monochrome variant**. Otherwise use a neutral outline SVG. Do not create
  colourful illustrative icon sets just to decorate a page.
- For UI affordances (theme, copy, download, arrows) prefer inline SVG icons or the few neutral glyphs already in the
  kit (`◐ ⤓ → ▸`). Functional, not festive.

### Vocabulary records this ticket maps or contrasts

From `iconography/vocabulary.jsonl` (238 records on 2026-09-25). Every record below is `status: active` and
`colour: content`; each is a `ref` record (drawn from a FastMediaSorter Android drawable, named in the last
column), and its SVG is `glyphs-snapshot-2026-09-25/<id>.svg`. "Here" is where this product meets the
meaning (the Findings above); the other columns are the record's own fields.

| Id | Here | en | ru | uk | rtl | Shape | Distinct from | Shared with | Ref |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `nav.back` | wrongly named on the prev link | Back | Назад | Назад | mirror | arrow pointing left | `media.previous`, `media.rewind`, `action.undo`, `nav.exit` | - | `ic_arrow_back` |
| `nav.forward` | wrongly named on the next link | Forward | Вперёд | Вперед | mirror | arrow pointing right | `media.next`, `nav.go-to`, `action.send-to` | - | `ic_arrow_forward` |
| `media.previous` | previous page (target) | Previous | Предыдущий | Попередній | fixed | triangle pointing left against a bar | `nav.back`, `media.rewind`, `media.to-start` | - | `ic_skip_previous` |
| `media.next` | next page (target) | Next | Следующий | Наступний | fixed | triangle pointing right against a bar | `nav.forward`, `media.fast-forward` | - | `ic_skip_next` |
| `media.play` | shape of today's `▶` | Play | Воспроизвести | Відтворити | fixed | solid triangle pointing right | `media.play-pause`, `device.media-player` | `action.run` | `ic_play` |
| `action.run` | shape of today's `▶` | Run | Запустить | Запустити | fixed | solid triangle pointing right | `media.play` | `media.play` ("Start-now is drawn as play on every desktop and mobile platform.") | `ic_play` |
| `nav.contents` | Contents (`☰` today) | Table of contents | Содержание | Зміст | fixed | lines with bullets | `view.list` | - | `ic_toc` |
| `feature.continue-reading` | index "Continue reading" (`▸` today) | Continue reading | Продолжить чтение | Продовжити читання | fixed | open book | `content.book`, `content.map` | - | `ic_widget_continue_reading` |
| `nav.scroll-top` | site to-top (`↑` today) | Scroll to top | В начало списка | На початок списку | fixed | arrow up to a bar | `nav.page-up`, `action.move-up`, `media.to-start` | `action.move-first` | `ic_vertical_align_top` |
| `nav.scroll-bottom` | shape of today's `⤓` | Scroll to bottom | В конец списка | В кінець списку | fixed | arrow down to a bar | `nav.page-down`, `action.move-down` | `action.move-last` | `ic_vertical_align_bottom` |
| `action.save` | extension `↓ File` (target) | Save | Сохранить | Зберегти | fixed | floppy disk | `action.confirm`, `action.download` | - | `ic_save` |
| `action.export` | extension `↓ HTML` (target) | Export | Экспорт | Експорт | fixed | arrow rising from a tray | `action.share`, `action.upload`, `action.import` | - | `ic_export` |
| `action.copy` | Copy / Copied (`✓` today) | Copy | Копировать | Копіювати | fixed | two overlapping sheets | `action.move`, `action.share` | - | `ic_copy` |
| `action.confirm` | shape of today's `✓` | Apply | Применить | Застосувати | fixed | check mark | `action.save`, `status.ok` | - | `ic_check` |
| `nav.go-to` | `← Desktop app` cross-link (target) | Go to | Перейти | Перейти | mirror | chevron pointing right | `nav.forward`, `nav.expand` | - | `ic_chevron_right` |
| `nav.open-external` | external link (`↗` today) | Open in new window | Открыть в новом окне | Відкрити в новому вікні | mirror | arrow leaving a square at its top corner | `action.share`, `nav.go-to`, `action.import` | - | `ic_open_in_browse` |
| `nav.expand` | disclosure closed (`▸` today) | Expand | Развернуть | Розгорнути | fixed | chevron pointing down | `nav.dropdown`, `nav.page-down`, `nav.go-to` | - | `ic_expand_more` |
| `nav.collapse` | disclosure open (`▾` today) | Collapse | Свернуть | Згорнути | fixed | chevron pointing up | `nav.page-up` | - | `ic_expand_less` |
| `nav.dropdown` | shape nearest today's `▾` | Choose from list | Выбрать из списка | Вибрати зі списку | fixed | small solid triangle pointing down | `nav.expand` | - | `ic_arrow_drop_down` |
| `action.download` | store/download (candidate, B1) | Download | Скачать | Завантажити | fixed | cloud with a downward arrow | `action.import`, `action.upload` | - | `ic_cloud_download` |
| `action.move-up` | shape of today's `↑` | Move up | Переместить вверх | Перемістити вгору | fixed | arrow pointing up | `nav.scroll-top`, `nav.page-up`, `action.upload` | - | `ic_arrow_upward` |
| `action.move-down` | shape of today's `↓` | Move down | Переместить вниз | Перемістити вниз | fixed | arrow pointing down | `nav.scroll-bottom`, `nav.page-down`, `action.download` | - | `ic_arrow_downward` |
| `action.swap` | GUI swap `⇄` (target) | Swap | Поменять местами | Поміняти місцями | fixed | two arrows pointing opposite ways | `action.flip`, `action.sync` | - | `ic_swap_horizontal` |
| `action.extract-text` | contrast for the OCR text layer (B1) | Extract text | Извлечь текст | Витягти текст | fixed | the letters OCR | `action.translate`, `feature.camera-ocr` | - | `ic_ocr` |
| `media.zoom-in` | contrast for text larger (B1) | Zoom in | Увеличить | Збільшити | fixed | magnifier with a plus | `action.search` | - | `ic_zoom_in` |
| `media.zoom-out` | contrast for text smaller (B1) | Zoom out | Уменьшить | Зменшити | fixed | magnifier with a minus | `action.search` | - | `ic_zoom_out` |
| `weather.clear` | shape of today's `☀` (theme "Light") | Clear | Ясно | Ясно | fixed | sun | - | - | `ic_weather_clear` |
| `app.night-mode` | shape of today's `☾` (theme "Dark") | Night mode | Ночной режим | Нічний режим | fixed | crescent moon | `media.sleep-timer`, `camera.night` | - | `ic_night_mode` |
| `camera.record-video` | shape of today's `●` (theme "Night") | Record video | Записать видео | Записати відео | fixed | solid dot | `media.stop` | - | `ic_shutter_video_idle` (state `recording`: `ic_shutter_video_recording`) |

`means` of the meanings this ticket ships, verbatim: `media.previous` "Go to the previous item of a
sequence: file, track, page, chapter or search match."; `media.next` "Go to the next item of a sequence:
file, track, page, chapter or search match."; `nav.contents` "Open the list of chapters or sections of this
document."; `feature.continue-reading` "Open the book or document where reading stopped."; `nav.scroll-top`
"Jump the view to the first item of a long list." (B4 asks to widen "list"); `nav.expand` "Show the hidden
part of this section or panel in place."; `nav.collapse` "Hide the part of this section or panel that is
shown in place."; `nav.go-to` "Marks a row or tile that opens its own screen."; `nav.open-external` "Open
the item outside this screen - in a new window, another app or the browser."; `action.copy` "Make a copy of
the item, or put text on the clipboard."; `action.save` "Write the edit to storage."; `action.export`
"Write data out to a file that can be kept or moved elsewhere."; `action.download` "Fetch from the internet
or a cloud to this device."; `action.swap` "Exchange the two sides, such as source and target language.".
Record-level `looks` / `hue`: `media.play` and `action.run` `["decorated"]` / `accent`;
`feature.continue-reading` and `action.download` `["colour", "decorated"]` / `accent`; `action.swap`
`["decorated"]` / `accent`; the others none (mono only).

Not in the vocabulary (B1): a theme choice, text smaller / larger, the OCR text layer, convert to HTML,
jump to page, a `done` state of `action.copy`, install-from-store. `app.theme` and `app.theme-toggle`
exist only as proposals (see B1), not as records.

### `palette.json` - the entries `ICON-RENDER` rule 3 contrast needs

The file's own header still reads `"contract": "ICON-RENDER 0.10, section 10 item D"` (a generated label;
the contract is at 0.11). Verbatim excerpt:

```json
"onPlateRule": "white when white reaches 3:1 against the plate, otherwise #1F1F1F",
"accent": {
  "day": "#1976D2", "night": "#64B5F6", "plate": "#1976D2", "onPlate": "#FFFFFF",
  "dayContrast": 4.6, "nightContrast": 8.46, "onPlateContrast": 4.6,
  "source": "colorPrimary of the base themes"
},
"state.ok": {
  "day": "#2E7D32", "night": "#81C784", "plate": "#2E7D32", "onPlate": "#FFFFFF",
  "dayContrast": 5.13, "nightContrast": 9.31, "onPlateContrast": 5.13,
  "source": "success_color"
},
"state.warning": {
  "day": "#F57C00", "night": "#FFB74D", "plate": "#F57C00", "onPlate": "#1F1F1F",
  "dayContrast": 2.7, "nightContrast": 10.82, "onPlateContrast": 6.1,
  "source": "warning_color"
},
"state.error": {
  "day": "#D32F2F", "night": "#EF5350", "plate": "#D32F2F", "onPlate": "#FFFFFF",
  "dayContrast": 4.98, "nightContrast": 5.37, "onPlateContrast": 4.98,
  "source": "error_color"
}
```

(Whitespace compacted; keys and values exact.) `accent` is the reference product's own tone - section 10
item D gives `accent` to each product, so this product's accent is its own and only the 3 : 1 floor
binds. Every glyph this ticket maps is `colour: content`, so rule 3 measures it in this product's own text
colour against each of its four reader themes (light / sepia / dark / night) - which is what the index TOC
`#1a0dab` finding is about; the palette tones apply only where a colour or decorated look, or a `state`
hue, is used. `state.warning`'s `dayContrast` 2.7 is B7.
