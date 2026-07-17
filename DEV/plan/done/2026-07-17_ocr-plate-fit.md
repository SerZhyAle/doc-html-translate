# OCR plates do not fit their text: a third clip it, an eighth half-empty

**Status:** Implemented 2026-07-17 - the durable fix, not a constant nudge: a runtime re-fit shrinks the
plate font to its source box after layout and again when the translator swaps the text, growing the box
only if it still overflows at the font floor so nothing clips; CSS centres text (fixes the half-empty
shape). Verified in real headless Chrome: `comic-scan-tiny` before `clipped=1` -> after `clipped=0`, and
holds `clipped=0` after a simulated longer-translation swap. Both editions + parity + docs.
**Priority:** 10
**Date:** 2026-07-17

> Cross-edition feature ticket. One feature = one ticket covering every edition.
> Read [`docs/PARITY.md`](../../docs/PARITY.md) before starting; update it when a shared invariant moves.

## What / why

Owner-reported from real reading (2026-07-17), three screenshots, three shapes:

1. a plate large enough for its text, but the text sits in the **top half** and the rest is empty;
2. a plate that **missed** - a large plate far from the bubble it should cover;
3. a plate whose **font is too big**, so the text is cut off mid-sentence.

These are not three bugs. They are one: the plate box and the font size are both computed from the
**source** geometry (the recognized line boxes), and then different text - reflowed to the box's full
width, and later swapped again by the browser's translator for a string of another length entirely -
is poured into that fixed box. Any formula that guesses a size before knowing the final text is
guessing. The CSS then makes each failure mode visible: `align-items:flex-start` pins short text to
the top (shape 1), `overflow:hidden` clips long text (shape 3), and a bad cluster produces a box that
covers the wrong region (shape 2).

Not anecdotal. Converting `test_doc/comic-scan-tiny_First-Earthman-on-Mars-1944.pdf` produces 60
plates; measuring each box against the text actually placed in it:

| | Plates | Share |
|---|---|---|
| Clipped (needs more height than the box has, `overflow:hidden` eats it) | 19 | 32% |
| Text needs under 55% of the box, floats at the top | 7 | 12% |
| Fits | 34 | 57% |

So roughly **one plate in three is clipped before translation has even run** - and translation
usually makes text longer, so the shipped number is worse than this.

Note the ordering dependency: much of the clipping is fed by garbage recognition from
[`2026-07-17_ocr-upscale-threshold-misses-page-scans`](2026-07-17_ocr-upscale-threshold-misses-page-scans.md)
(salad tokens wrap unpredictably, and the median-line-height the font-size derives from is measured
on noise). Fix the upscale gate **first**, then re-measure the table above - the honest fix here may
be smaller than it looks today, and tuning the constants against noisy input would be tuning against
the wrong signal.

The durable fix is to stop guessing: fit the text to the box **after** it exists, in the page, and
re-fit when the translator swaps the text. That is a behaviour change on both editions and wants its
own tactical pass - do not start by nudging the constants.

## Edition parity checklist

For each edition: **Done**, or **Declined** with a one-line rationale (record lasting declines in
`docs/PARITY.md` under "Intentional divergences"). "Not applicable" needs a reason too.

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[x]` | `ocrScript` + `ensureScript` inject the re-fit; CSS centres text (`overlay.go`) |
| GUI (`doc-html-ui`) | `[x]` | inherits the pipeline output; no GUI change needed |
| MSIX Store app | `[x]` | inherits the GUI; the script is inlined into the static HTML, no runtime dependency |
| Browser extension | `[x]` | `fitPlate` + `scheduleFit` ported; `.ocr-plate` gains `overflow:hidden` + `align-items:center` |
| Website / docs | Declined | plate fidelity is not claimed publicly; nothing to correct |

## Shared invariants touched

Every one of these is in `docs/PARITY.md` and guarded by `tests/parity_test.go` - a change to any is
a both-editions change:

- **Plate font-fit factor** `0.92` (`overlay.go` `fontFitFactor` == `ocr-overlay.js` `FONT_FIT`),
  guarded by `TestParityOCRFontFit`. Plate font-size = median line height x this factor.
- **Overlay grouping** `OCR_MIN_LINE_CONF = 50`, `OCR_CLUSTER_GAP_FACTOR = 1.2` - clustering owns
  shape 2 (the plate that missed).
- **Overlay CSS** `.ocr-box` / `.ocr-fig` - the `align-items` / `overflow` / `min-height` triple that
  turns a bad fit into a visible defect. If the fix is a runtime re-fit, this introduces a **new**
  shared behaviour and the invariant tables need a row for it.

## Cross-references

- Go: [`internal/ocr/overlay.go`](../../internal/ocr/overlay.go) - `ocrCSS`, `wrapImage`, plate sizing,
  `blockColors`.
- JS: `extension/src/ocr-overlay.js` + `extension/src/ocr-overlay.css`.
- Reproduction and measurement: `test_doc/CORPUS.md`; the plate-fit measurement used to build the
  table above is ad-hoc - fold it into a test rather than re-deriving it by hand.

## Done criteria

- [x] Every edition row above is `Done` or `Declined (reason)`.
- [x] Re-measured **after** the upscale gate (P9) landed, so the fit ran against real text: `comic-scan-tiny`
      now produces 33 plates (single-page); before the fit `clipped=1`, after it `clipped=0 shrunk=4 grew=1`.
- [x] Shapes checked in a **real browser** (headless Chrome, `--dump-dom`), not reasoned about: shape 1
      (half-empty) fixed by centring; shape 3 (clipped) fixed by the shrink-then-grow fit; shape 2 (the
      plate that missed its bubble) is a recognition problem, addressed by P9's DPI declaration, out of
      scope for a fit change (recorded in the ticket body).
- [x] Text stays inside its plate after the translator swaps it - simulated by doubling every plate's
      text; the `MutationObserver` re-fit holds `clipped=0`.
- [x] `docs/PARITY.md` updated: plate-geometry row (centred, `overflow:hidden`) + new "Plate runtime
      re-fit" shared-behaviour row.
- [x] `scripts/test.ps1` green (Go incl. 226s integration + new `TestEnsureOverlayAssets`), `npm test` 75/75.
- [x] Changelog entry in `DEV/CHANGELOG.md` (2026-07-17 23:20).

**Re-measurement note.** The ticket's 60-plate / 32%-clipped table was measured on the pre-P9 salad. After
P9 the recognition is real text and single-page mode merges the book into one file, so the plate count and
distribution differ (33 plates here). Rather than re-derive the old table by hand, the fit is proven by the
invariant that matters - **0 clipped after fit, and 0 clipped after a translation-length swap** - measured
in the browser and guarded by a test, which is the property the table was a proxy for.
