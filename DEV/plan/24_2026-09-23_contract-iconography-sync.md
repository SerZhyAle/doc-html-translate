# One glyph and one name per meaning - adopt the shared icon vocabulary

**Status:** Draft
**Priority:** 52
**Date:** 2026-09-23

> Contract sync ticket, both directions.
> Contracts: `ICON-SET` 0.10, `ICON-RENDER` 0.10, `ICON-EXTERNAL` 0.9 (domain `iconography/` of the shared
> catalog, owner FastMediaSorter Android, all drafts). No pointer in [`docs/contracts/`](../../docs/contracts/) yet.

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
3. Every glyph-only control carries its canonical name, localized, as the accessible name - both editions.
4. The OCR-layer toggle shows its state.
5. Extension viewer and popup: the same meanings, the same glyphs, the same names as the Go edition (a
   cross-edition parity row in [`docs/PARITY.md`](../../docs/PARITY.md), guarded by a test).
6. GUI: the four glyphs it uses take their vocabulary shapes; disclosure mirrors in RTL.
7. Site: to-top, copied, store/download, cross-link and disclosure glyphs from the vocabulary - after the
   `PAGE-STYLE` §9 conflict (B9) is settled.
8. Glyphs leave translatable strings (`↓ File` -> glyph in markup, word in the message).
9. The source and licence of every copied glyph is on record (`ICON-EXTERNAL` rule 5), in a
   third-party-notices file that ships where the glyphs ship.
10. A guard test pins the book chrome's glyph + name pairs, so the founding-rule breach cannot come back.

## Direction B - what the vocabulary needs from this product

Written as `PROPOSAL-2026-09-23-<topic>.md` in the catalog's `iconography/` folder, never as an edit.
Each item states the evidence above.

- **B1 new meanings (MINOR):** theme choice (with levels light / sepia / dark / night, or one record per
  theme - settles the clash with `app.night-mode`); text smaller / larger, distinct from image zoom;
  OCR text layer shown/hidden, distinct from `action.extract-text`; convert a document to HTML (also the
  Windows shell verb); jump to page, since "Go to" is already `nav.go-to`'s name; a `done` state for
  `action.copy`; install-from-store, unless `action.download` is meant to cover it.
- **B2 ten new languages** (de it es fr pt ar hi bn ur zh) for every record this product ships - rule 3
  makes the first shipper of a language responsible for it.
- **B3 RTL paging:** `media.previous` / `media.next` are `rtl: fixed`, but paging a document follows the
  reading direction in ar/ur. Possibly a split, and then MAJOR - propose, do not assume.
- **B4** `nav.scroll-top` / `nav.scroll-bottom` say "list"; widen to "page or list", ru name accordingly.
- **B5** `source.local` is drawn as a phone with an Android-only rationale - ask for a desktop variant.
- **B6** `app.logo` is one product's mark inside the portfolio vocabulary, while rule 7 puts product marks
  out of scope - namespace it or remove it.
- **B7** `palette.json` `state.warning` has day contrast 2.7, below the 3:1 floor the contract demands.
- **B8** 44 px touch target on Windows and the web contradicts dense mouse-first toolbars and the
  portfolio's own kit (`.theme-btn` 32 px): propose 44 px for coarse pointers and a smaller floor for
  fine pointers.
- **B9 cross-contract conflict:** `PAGE-STYLE` §9 recommends the kit glyphs `◐ ⤓ → ▸`; `ICON-SET` gives
  `⤓` to `nav.scroll-bottom` and `▸` to nothing. One proposal to both owners (`iconography/` and
  `product-web-pages/`). Linked from [`26_2026-09-23_contract-product-web-pages-sync`](26_2026-09-23_contract-product-web-pages-sync.md).
- **B10** Windows system surfaces missing from rule 9: file-type icon, shell-verb icon, MSIX unplated and
  light-theme assets, browser-extension action icon.
- **B11** platform-native disclosure markers (`<details>` marker): a declared platform shape or forbidden.
- **B12** `ICON-EXTERNAL` rules 3-4: are a converted book's embedded images "downloaded pictures", and what
  is the fallback for a missing one.
- **B13** ru name of the table of contents: the vocabulary says "Содержание", book readers and both
  editions here say "Оглавление" - propose or comply, owner's call (open question 2).

## Done criteria

- [ ] Pointer files `docs/contracts/ICON-SET.md`, `ICON-RENDER.md`, `ICON-EXTERNAL.md` exist and are listed
      in [`docs/contracts/README.md`](../../docs/contracts/README.md).
- [ ] Registry: this product's adoption rows for the three ids, each deviation still open recorded as a
      dated exception with an `until` date.
- [ ] The proposals B1-B12 are filed in the catalog folder (or each is withdrawn in writing here with the
      reason), and the book chrome ships no meaning the vocabulary lacks (rule 5: the vocabulary moves
      first).
- [ ] The glyph map (A1) exists and every row is `conforms`, `exception` or `proposal filed`.
- [ ] Multi-page output: previous/next are `media.previous` / `media.next` in glyph and name in all 13
      languages, pinned by a test.
- [ ] No glyph-only control on any surface takes its accessible name from the glyph character.
- [ ] Index TOC text passes 3:1 on all four reader themes.
- [ ] Both editions show the same glyph for the same meaning, guarded by a parity test.
- [ ] Glyph sources and licences are on record.
- [ ] Site and `README*` changes land in every authored locale in one edit (canon invariant 17).

## Open questions

1. Target conformance rung (`iconography/README.md` §6): rung 2 (mapped) now, rung 3 (label vs glyph
   check) with this ticket, rung 4 (theme contrast) later?
2. Toc name in ru: comply with "Содержание" or propose "Оглавление"?
3. Rendering in the self-contained offline book: inline SVG per page, one CSS `mask` data-URI per page in
   the reader stylesheet, or keep text characters and propose that the contract allow them. Size cost
   multiplies by hundreds of chapter pages.
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
