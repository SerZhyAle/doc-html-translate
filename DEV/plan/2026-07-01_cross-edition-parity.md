# Cross-edition parity: close the drift between the Go app and the JS extension

**Status:** In Progress - all P0 + the safe P1 aligns done and guarded by tests; P2 (product decisions) documented
**Priority:** 40
**Date:** 2026-07-01

Reference: [`docs/PARITY.md`](../../docs/PARITY.md) (invariants + port map + intentional divergences).
This ticket is the **backlog** of concrete gaps found in the 2026-07-01 parity audit. Each item is
tagged with a target resolution:

- **Align** - make the two sides use the same value/behaviour (unintended drift).
- **Port** - implement a missing capability on the side that lacks it (product decision needed).
- **Document** - keep the difference, record it in `docs/PARITY.md` as intentional (no code change).

Pick an area, split it into its own tactical ticket if it needs one, and update `docs/PARITY.md` +
this file as items close. Do the **P0** items first - those are unintended and likely to bite.

## Execution log (2026-07-01)

**Aligned in code (done):**
- #1 OCR tessdata version - pinned Go `cdnBase` to `.../tessdata_fast/raw/4.0.0` to match the extension's
  `4.0.0_fast` (host/format differ, bytes identical). Guarded by `TestParityOCRVersion`.
- #2 Bundled-eng comment corrected (build-time provisioning, both sides 4.0.0).
- #3 Extension `DEFAULT_OPTIONS` unified into `extension/src/defaults.js`, imported by background.js /
  popup.js / options.js / viewer.js (background.js no longer misses the OCR fields).
- #7 PDF reflow constants named in a documented `const` block in `extract.go`. Guarded by
  `TestParityReflowConstants`.
- #9 OCR language catalog aligned at 13 (extension `LANGS` + `HTML_LANG` expanded to match Go
  `Available`). Guarded by `TestParityOCRCatalog`.
- #10 EPUB TOC: Go NCX titles now use `collapseWS`; extension `isExternalHref` now mirrors `toc.go`.
- #21 Guard tests added: `tests/parity_test.go` (palette, OCR version, OCR catalog, reflow constants).
- #23 Guard test added: `tests/ui_cli_parity_test.go` (GUI exposes every CLI flag).
- #5 / #6 (palette / FAMILIES) - not restructured into a shared source, but now pinned in `docs/PARITY.md`
  and enforced by `TestParityThemePalette`, which was the pragmatic resolution.

Verified: `go build ./...`, `go test ./internal/... ./cmd/... ./tests/...` green; extension `npm test`
28/28 green.

**Documented as intentional (no code change):**
- #4 split default - the GUI deliberately sends `-split 0` (asserted by `TestAssembleArgsPassesExplicitSplitZero`);
  kept, recorded under `docs/PARITY.md` "Settings defaults".
- #8 OCR CSS class names - cosmetic; deferred, noted in `docs/PARITY.md` OCR.
- #16 OCR default-language rule - the extension has no translation source, so a fixed `eng` default is
  intentional; recorded in `docs/PARITY.md`.
- #11-#15, #17-#20 (P2 ports) - each is an existing intentional divergence in `docs/PARITY.md`; left as-is
  pending a product decision to port.

**Deferred (still open):**
- #8 class-name unification, and the P2 ports (#11-#20) if/when the product wants full parity.
- #22 extension DOM-path unit tests (large; the concurrent format-parity work is adding coverage here).

## P0 - unintended drift / probable bugs (Align)

| # | Gap | Current state | Target |
|---|---|---|---|
| 1 | OCR download host + tessdata version | Go: GitHub `tessdata_fast/raw/main` (plain); JS: `tessdata.projectnaptha.com/4.0.0_fast` (gzip). Different data -> different recognition. `tessdata.go:42` vs `ocr-lang.js:27` | **Align** on one host + one pinned version; update `docs/PARITY.md` OCR table |
| 2 | Go "bundled English" claim | `Bundled=["eng"]` + "works offline out of the box" comment, but no traineddata in repo; relies on the build script provisioning it. `tessdata.go:38` | **Align/verify**: confirm `scripts/build.ps1` always provisions `<exe>/tessdata/eng.traineddata` from the same version as the extension; fix comment |
| 3 | Extension `DEFAULT_OPTIONS` triplicated and out of sync | `background.js:14-19` omits `ocrImages`/`ocrLang`; `popup.js:6` / `options.js:6` include them | **Align**: one shared defaults module imported by all three |
| 4 | Split-size default inverted GUI vs CLI | CLI `-split 5000`; GUI always sends `-split 0`. `flags.go:45` vs `ui.html:232` | **Align** or **Document** as a deliberate GUI default (decide + record) |

## P1 - drift-prone shared invariants (Align + pin)

| # | Gap | Current state | Target |
|---|---|---|---|
| 5 | Theme palettes hand-duplicated | 4 palettes copied hex-for-hex, kept in sync by a comment. `navbar.go:353-373` vs `viewer.css:5-51` | **Align** via a single generated source of truth if practical; else pin in `docs/PARITY.md` (done) + add a guard test |
| 6 | `FAMILIES` map duplicated | identical today, independently maintained. `navbar.go:404` vs `viewer.js:91` | **Align**/pin |
| 7 | PDF reflow constants as inline literals | 1.5 / 8 / 25pct / 12 / ligature<3.0,>=4 / short<=8,<=14 duplicated. `extract.go` vs `reflow.js` | **Align**: name them; pinned in `docs/PARITY.md` |
| 8 | OCR plate CSS class names + rounding policy | `.ocr-fig`/`.ocr-box` vs `.ocr-overlay`/`.ocr-plate`; Go rounds all coords to 2dp, JS rounds only font. `overlay.go:16` vs `ocr-overlay.css` | **Align** class vocabulary + rounding, or **Document** |
| 9 | OCR language catalog size | Go 13 vs JS 5; Go comment "mirrors the extension" is stale. `tessdata.go:22-36` vs `ocr-lang.js:15-21` | **Align** the catalogs (or document the superset rationale) |
| 10 | EPUB TOC whitespace + external-href drift | Go NCX path only trims (no internal collapse); "external" = `://` (Go) vs `^https?` (JS); TOC target scope = manifest (Go) vs spine (JS). `toc.go` vs `epub.js` | **Align** the three rules; pin in `docs/PARITY.md` |

## P2 - capability gaps (Port - needs a product decision)

| # | Gap | Missing where | Decision needed |
|---|---|---|---|
| 11 | Reading position + "Continue reading" | extension | Port to the extension? If yes, make `bookStorageKey` (FNV-32a of `title|total`) reproducible cross-front-end |
| 12 | Page zoom (Ctrl+wheel / `?z=`) | extension | Port, or keep font-size as the extension's only model (**Document**) |
| 13 | Heuristic source-language detection (`lang.js`) | Go | Port to Go, or keep Go's "copy source `<html lang>`" behaviour (**Document**) |
| 14 | PDF: double-spaced/ZWSP merge + `hrefForPDFPage` remap | extension | Port to JS, or **Document** (extension renders every page) |
| 15 | PDF: font-size heading dimension + Cyrillic ALL-CAPS | Go | Port to Go, or **Document** as a JS-only improvement |
| 16 | OCR default-language rule | mismatch | Go derives from `-src` via `TessLang` (else `eng`); JS uses a fixed persisted `eng`. Unify the rule |
| 17 | OCR: scanned-page rasterization fallback + `minSize=64` filter | Go OCR layer | Port the extension's `pdf-images.js` behaviour, or **Document** the Go limitation |
| 18 | Right-click "OCR this image" | Go | N/A for a converter - **Document** as extension-only |
| 19 | Extension UI localization (RU) | extension | Localize to match Go's `syslocale.IsRussian`, or **Document** |
| 20 | Reader theme attribute + CSS var namespace | mismatch | Go `data-dht-theme`/`--dht-*` vs JS `data-theme`/`--*`. **Document** as intentional (values identical) unless a shared stylesheet is pursued |

## P3 - test / guard debt

| # | Gap | Target |
|---|---|---|
| 21 | No automated parity guard for shared constants | Add a test that asserts the palette hexes / reflow constants / OCR host+catalog match across Go and JS (fixture compare) |
| 22 | Extension DOM path (sanitize/image/link/TOC) has zero automated tests | Add `node --test` coverage for `renderChapter`/`rewriteImg`/`convertSvgImage`/`rewriteAnchor`/`parseNavToc`/`parseNcxToc` |
| 23 | No "GUI exposes every CLI flag" test | Add a parity test enumerating flags vs GUI controls (enforce the `ui-cli-parity` invariant by code) |

## Done criteria

- [ ] Every P0 item is Aligned (or explicitly Documented as intentional with a rationale).
- [ ] P1 invariants are either unified in code or pinned in `docs/PARITY.md` + guarded by a test (item 21).
- [ ] Every P2 item has a recorded decision (Ported or Documented) - no silent "not done".
- [ ] `docs/PARITY.md` reflects the final state of every item above.
