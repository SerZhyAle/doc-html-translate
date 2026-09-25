# The OCR contracts describe the mechanism that ships, and the mechanism holds them

**Status:** In Progress - Direction A items 1, 2, 4, 6 (code), 8, 9, 10 and the browser half of 5 done on 2026-09-25 (see "Implementation record"); every Direction B step, items 3, 5 (first half), 7, 11 and the registry half of 6 wait on the owner's machine.
**Priority:** 49
**Date:** 2026-09-23

> Contract sync ticket, both directions.
> Contracts: `OCR-PIPELINE` 1.0 (owned by this product), `OCR-OVERLAY` 1.0 (shared owner, this product is
> the reference implementation), `OCR-INVOCATION` 1.0 (owned by this product). Domain `ocr-overlay/`.
> Pointers: [`OCR-PIPELINE`](../../docs/contracts/OCR-PIPELINE.md),
> [`OCR-OVERLAY`](../../docs/contracts/OCR-OVERLAY.md), [`OCR-INVOCATION`](../../docs/contracts/OCR-INVOCATION.md).
> Related tickets, not duplicated here:
> [`15_2026-09-22_ocr-discard-record-missing-for-blank-images`](done/15_2026-09-22_ocr-discard-record-missing-for-blank-images.md) (done),
> [`17_2026-09-22_tsv-columns-read-by-position`](done/17_2026-09-22_tsv-columns-read-by-position.md) (done).

> **Remote execution (2026-09-25):** the contract text this ticket needs is quoted in "Contract snapshot" below, so every step not marked ⛔ runs in a cloud session from this repository alone. Steps marked **⛔ Local only** edit the shared contracts catalog (or another repository) and can run only on the owner's machine, where the catalog is mounted.

## What / why

The 2026-09-22 alignment run moved the OCR documents into the catalog and verified their *place*, half of
`OCR-OVERLAY` and all of `OCR-INVOCATION`. It left two things open on purpose: the `OCR-PIPELINE` registry
row is `pending` because nobody read its constant table against the code, and nine `OCR-OVERLAY` rules
(2, 3, 4, 7, 9, 11, 13, 14, 15) were not re-read. (Re-verified 2026-09-25: the `OCR-PIPELINE` row is no
longer `pending` - another run dated it verified on 2026-09-24; see Direction B step 6 for what that row
still lacks.)

That read was done on 2026-09-23. The constants hold - all twenty in `OCR-PIPELINE` §5 match in Go and JS.
The **document** does not: it was registered already stale against the word-gap stage of 2026-09-12 and
the page-OCR surface of 2026-09-19, so the "catalog first, then code" order this repo's own pointer states
was broken - and FastMediaSorter_Lite ports from that document. The read also found deviations the
previous run did not.

The working-tree edits in `internal/ocr/*.go` and `internal/pipeline/pipeline.go` are comment-only; no
pinned value, flag, exit code or lookup order moved since 2026-09-22.

## Findings that drive the work

**`OCR-PIPELINE` - missing from the document (code is right, document behind):**
- the word-gap split (`ocrMaxWordGapRatio` 3.5), the column reordering and the outlier-word trim of
  2026-09-12, and the fact that type size is now the median of word heights - all pinned by parity tests,
  none in the document;
- the fit ladder's grow branch (cap 1.15, step 4%, 20 iterations) - the document says shrink only;
- the colour fallback (luma > 140 -> 17, else 240), the min-6 floor on the ink share, the 1.3-line ink
  strip, the column-overlap test (0.1) and the negative-gap tolerance, plate padding and radius, the CJK
  and ratio details of the translatability test;
  *Re-verified 2026-09-25:* since commit `6b952c2` (2026-09-24) the plate padding (`0.08em 0.28em`) and
  radius (`0.35em`) live in `internal/appearance/appearance.json` and every declaration is pinned on both
  editions by `tests/appearance_parity_test.go` - the code-side pin is done; the catalog gap stays;
- the page-OCR surface (extension page agent, 96 px minimum picture) in §4 and §6;
- the script-confidence floor 6.4 is in prose only, not in the §5 table.

**`OCR-PIPELINE` - wrong in the document:**
- §2.7 points at `hasWordRun`, which exists in neither edition (a rolled-back rule; re-checked 2026-09-25,
  still absent from `internal/` and `extension/src/`);
- the ring band is "one line-height wide" in §3.3 - the code samples a third of one; a port built from the
  prose samples a band three times wider (open question 3);
- §3.2 "fully transparent pixels skipped" - the code skips alpha < 128;
- §2.5 "3 median ink heights" - the code multiplies the median line-box height;
- the module map names `ocr-overlay.js` for plate rendering (moved to `ocr-plates.js`) and a per-image
  diagnostics line in `diagnostics.js` that does not exist;
- §7 "every threshold above was measured" overclaims (see rule 13 below).

**`OCR-OVERLAY` - verdicts of the nine unread rules:**
- held: 3 (one explicit transform), 7 (opaque plate, padding part of it), 11 (OCR text never becomes
  truth), 14 (not measured is not nothing visible);
- **rule 2 partial** - Go `scaleDown` does not scale `Dropped`, so the discard record's boxes on an
  upscaled image stay in 2x space while width and height are halved;
- **rule 4 partial** - held for generated HTML and the viewer; the page-OCR surface reads the border-box
  rectangle and handles neither CSS rotation, `object-fit`, padding nor clear-on-rotation;
- **rule 9 partial** - the bounded ladder and release exist; the written UI rule for a released plate that
  overlaps the next one or runs past the image bottom does not;
- **rule 13 not held** - 0.52, 0.72, 1.6, 3.5, 6.4, 80 and the screen set are bracketed with dated
  reports; 50, 1.2, 3, 120/11/70, 0.92, 1.15, the colour numbers, the fit ladder and 40 are neither
  bracketed nor marked inherited;
- **rule 15 partial** - negative results (the word-gap overlap band 1.87-2.57, per-paragraph ordering
  rejected, trimmed box in decisions rejected) live in code and PARITY, not in the catalog.

**`OCR-OVERLAY` - findings outside the assigned rules:**
- **rule 5** - both editions size the plate font from the median of line-box heights; the rule says type
  size is the median of word heights, not the line box. The 2026-09-22 row recorded rule 5 as held, and
  FastMediaSorter_Lite reads the rule as binding the font;
- **rule 12** - silent gates beyond the blank-image ticket: whole clusters rejected by the translatability
  test, the screen-merge overlap rejection, and the sweep pass's own drops are recorded nowhere; Go appends
  primary + ladder drops while JS keeps only the ladder's, and `docs/PARITY.md` says "not merged";
- **rules 1 / 10** - the script-detection pass reads the raw first path, without the ASCII staging the
  document says recognition needs and without the EXIF turn, so on a Cyrillic file name it fails silently
  and no correction happens.

**`OCR-INVOCATION`:** unchanged in code. The document's "an omitted `-ocr-lang` derives from `-src`, else
`eng`" omits the script correction and the "no plates, with a note" stop that exist since 2026-08.

**§7 exchange JSON:** not emitted. Go blocks carry no confidence; `translation` does not exist at overlay
time (translation runs after); the lab's evidence schema is the nearest artefact.

## Contract snapshot (2026-09-25)

Quoted from the catalog on 2026-09-25 - `OCR-OVERLAY` 1.0, `OCR-PIPELINE` 1.0, `OCR-INVOCATION` 1.0, and the registry rows for doc-html-translate. A working copy for executing this ticket without the catalog, not a second source: the catalog stays authoritative, and this section is deleted when the ticket moves to done/.

Nothing in these documents changed between the ticket's date (2026-09-23) and this snapshot: the last log row of `ocr-pipeline.md` and of `integration-image-translate.md` is dated 2026-09-22, and `README.md`'s only later row (2026-09-24) edits the FastMediaSorter Android line of its section 8. `[..]` marks an omitted passage; everything else is verbatim.

### `OCR-OVERLAY` 1.0 - the shared contracts catalog, `ocr-overlay/README.md`

#### 2. The geometry rules

1. **Recognition happens in display space.** The pixels handed to the engine are the pixels the reader
   sees. EXIF orientation is applied **before** recognition, never after. A portrait phone shot stored
   sideways otherwise returns an upside-down transcript *and* lands it in a space the plates are not
   positioned in - both halves of the feature break at once.
   (`ocr-pipeline.md` §2.1, `ocr-overlay-accuracy.md` §3, exchange §5.1)

2. **Every preprocessing step keeps the way back.** Any upscale, downscale, crop or grey rendition carries
   the transform that returns a box to display space. Coordinates are never recovered by re-deriving
   factors in a later layer, and never by a chain of independent coefficients spread across layers.
   (exchange §5.2, §5.3)

3. **One explicit image-to-viewport transform.** The renderer holds exactly one, and it is the only thing
   that changes when the view changes.

4. **A plate never drifts.** Resize, zoom, page scale, fullscreen: the plate stays over its text, to the
   pixel. A product that cannot hold this for **rotation** clears the overlay instead of drawing it in the
   wrong place, and says so in its registry row. (exchange §5.4)

5. **Boxes are taken at the line level.** Block and paragraph boxes from the engine are discarded: trusting
   them makes an opaque plate span imagery the engine folded into a "paragraph". Type size is the **median
   of word heights**, not the height of the line box - a line box inflated by one artifact otherwise sets
   the type size for the whole plate. (`ocr-pipeline.md` §2.3, `ocr-overlay-accuracy.md` §5, S1711)

[..]

#### 3. The rules for what is drawn

8. **Paper and ink are sampled from the image, never assumed.** Both as **medians**, never means - a mean
   lands on a glyph's antialiased ramp, between the ink and the paper. Orientation is decided by a ring
   sampled **outside** the box, with a minimum vote count, because a box drawn tightly around capitals has
   their strokes on its own edges. A contrast floor replaces a sampled ink that fails it. A constant white
   plate with black text is a defect, not a simplification.
   (`ocr-pipeline.md` §3.2-3.3, `ocr-overlay-accuracy.md` §5, S1714)

9. **Nothing a user must read is clipped.** A translation longer than its source shrinks by a bounded
   ladder, and when the floor is reached the box is **released** and grows. Never an ellipsis, never a crop.
   The anchor point of the source block does not move while this happens, and whatever the product does
   when the text still does not fit is written down as a UI rule rather than decided per case.
   (`ocr-pipeline.md` §3.4, exchange §5.5)

10. **Correct plates, or none.** When the recognition language is wrong, unavailable or merely assumed, the
    product produces **no overlay** and says why, naming what would fix it. Transliterated debris over
    artwork that was readable before is output strictly worse than the input, and it is the failure mode a
    default language setting produces silently. (`ocr-pipeline.md` §2.0)

#### 4. The rules that keep it honest

12. **A silent decision leaves a record.** Every gate that drops a line records the text, the confidence,
    the box and **which** threshold it failed - through the *same predicate* the pipeline applies, not a
    second copy of the condition - and it records it **also for an image that produced nothing at all**,
    which is the case it exists for. Until that record exists, "the engine found nothing" and "we threw
    four lines away" are indistinguishable in a bug report. (`ocr-overlay-accuracy.md` §8)

13. **No threshold outside a dated report.** A constant is either derived from two measurements taken from
    opposite sides and set between them - the widest legitimate spread and the narrowest legitimate step,
    with the margin stated - or it is marked **inherited**, naming where it came from and the ticket that
    owns deriving it here. A number tuned until one scene passed is neither.
    (`ocr-overlay-accuracy.md` §5 rule 2, §8)

[..]

15. **A negative result ships beside the positive one.** A rule that was implemented, measured and rolled
    back stays in the document with its measurement, so the next reader does not re-derive it. A ticket
    that produced a refusal is closed as `Partial` with the evidence, not quietly reopened later.
    (`ocr-overlay-accuracy.md` §8)

[..]

17. **A developer can see the source box and the final plate without changing the data.** Diagnostics are
    recomputed with the same pure functions the renderer used, so turning them on cannot change what is
    rendered. (exchange §5.6, `ocr-pipeline.md` §7)

#### 5. Constants are per-implementation, their status is not

Every number in `ocr-pipeline.md` §5 was measured on one corpus, with one engine distribution, on one class
of input. A second product may not copy them as facts. What binds is rule 13: each implementation keeps a
table of its own constants, and each row says **derived here** (with the report) or **inherited** (with the
source contract and the ticket that owns deriving it). `ocr-overlay-accuracy.md` §5 is the worked example
of that table - rule by rule, `as-is` / `form only` / `not applicable`, each with a reason.

[..]

#### 7. The comparison format

When two implementations compare results on the same scene, they exchange this, whatever they use
internally. It is a measuring instrument, not a requirement to converge on one internal model.

```json
{
  "image": { "width": 2400, "height": 1600, "orientation": 1 },
  "blocks": [
    {
      "id": "b-001",
      "text": "Source text",
      "translation": "Перевод",
      "confidence": 92.4,
      "box": { "x": 320, "y": 180, "width": 640, "height": 96 },
      "readingOrder": 1
    }
  ]
}
```

- `box` is in display-space pixels (rule 1), origin top-left, `x`/`y` at the top-left corner.
- `confidence` is the engine's 0..100 mean over the block's words.
- **`rotationDegrees` is reserved and is not part of this version.** Neither side fills it: OSD is off in
  both segmentation modes one uses, and the other has no source for it at all, so the only value it could
  carry is `0` meaning *unknown* - which a reader would take as *not rotated*. Decided here on 2026-09-22,
  closing the open question in exchange §6: a field that can only lie is left out until an implementation
  can fill it truthfully.

### `OCR-PIPELINE` 1.0 - the shared contracts catalog, `ocr-overlay/ocr-pipeline.md`

#### 2. Detection path

##### 2.0 The language the page is read with (desktop only)

`-ocr-lang` when the reader gives it; otherwise the **translation source language** (`-src`, itself
defaulting to English). That default is easy not to notice, and Tesseract does not fail on a mismatch:
pointed at Cyrillic with English data it transliterates, and a Russian UI screenshot comes back covered
in plates reading `Katanoru-nonyyarenn` over an interface that was readable before - output strictly
worse than input.

So when, and only when, the language was **not** chosen by the reader, the book''s first image goes
through Tesseract''s orientation-and-script pass (`--psm 0`) and the answer may correct the default.
Once per book, not per image: a book is one document in one language, and the pass costs an extra
process (0.43 s measured) that a 480-page comic would pay 480 times for one answer.

- data for the detected script is installed -> the language becomes `<script language>+<default>`
  (`rus+eng`), never a replacement, so a wrong verdict costs a slower pass and not an unreadable page;
- it is not -> the book gets **no plates**, and a line names the script, the `-ocr-download` that fixes
  it and the `-ocr-lang` that overrides it. Correct plates, or none - never transliterated debris.

The floor is **6.4** and it is high because the detector is poor: over the 46 lab scenes it calls an
English Archie cover Cyrillic at 3.81 and an 1887 English cartoon Arabic at 5.00, gives no verdict at
all on 19 files including two of the corpus''s three Cyrillic scenes, and gets a non-Latin script right
exactly twice - at 8.24 and 8.15. 6.4 is the geometric middle of the worst wrong answer and the weaker
right one, with no scene between them.

The browser extension has no port of this: its OCR language is an explicit, persisted choice in the
popup rather than a value inferred from a translation flag, and the pass needs `osd.traineddata`, ~10 MB
it neither vendors nor downloads. A reader who never opens the popup still gets the debris there.

> `script.go`: `DetectScript`, `resolveScript` - no extension counterpart (see docs/PARITY.md)

[..]

##### 2.1 Resolution estimate and staging

Go additionally copies the file to an ASCII temp path: Tesseract/Leptonica open paths through the
Windows ANSI codepage and mangle any byte outside it, so a book under a Cyrillic name would fail
recognition silently.

The staged copy is also where the picture is **turned**. A browser paints an `<img>` through its
EXIF orientation; Tesseract reads the file's stored pixels. For the ordinary portrait phone shot
(`Orientation=6`) those are two different pictures, and both halves of the feature break: the
lettering is stored on its side and no OSD runs in either PSM this app uses, so recognition returns
the upside-down transcript (measured: `"duin{ seiqaz yep 2INb A\bulxen MoH"` for a legible line);
and whatever does read comes back in a space the plates are not positioned in. Applying the
orientation before recognition answers both at once and leaves every coordinate in display space,
so nothing downstream has to know. The extension needs no equivalent - it recognizes what
`createImageBitmap` decoded and lays plates over that same `<img>`.

> `tesseract.go`: `estimateDPI`, `prepareForOCR`, `stageForOCR`, `stageASCIIPath`;
> [`exif.go`](https://github.com/SerZhyAle/doc-html-translate/blob/main/internal/ocr/exif.go): `exifOrientation`, `orientImage` -
> `ocr-overlay.js`: `estimateDpi`, `upscaleForOcr`

[..]

##### 2.4 Confidence gate

A line whose mean word confidence is below **50** is dropped. Real text scores ~80-97; what the
engine hallucinates out of a drawing scores 0-50. Without the gate, noise becomes an opaque plate
over the artwork, and its oversized boxes also inflate the plate font. Rescue passes (2.6 onward)
run against a stricter floor of **80** - a rescue is a second guess, so the prior that there is text
at all is weaker, and invented words painted over artwork are worse than no overlay.

That floor alone throws away real words, and re-measuring it (2026-08-15, over 46 lab scenes and 13
annotated ones) showed it cannot be fixed by moving the number: genuine rescued lettering scores
32.8-69.2 and invented lettering 8.4-73.9, so the two **overlap**. A length rule was tried on top -
keep a line under the floor when it carries a run of four letters and clears 47, the middle of the
empty band those lines bracket - and the corpus rejected it: under the default `eng` a Cyrillic
poster then gets a 782x310 px plate of transliterated debris where it previously got none. So the
floor stays at **80** and the gap stays open. Evidence:
[`DEV/research/ocr_rescue_floor_2026-08-15.md`](https://github.com/SerZhyAle/doc-html-translate/blob/main/DEV/research/ocr_rescue_floor_2026-08-15.md).
A line the floor rejects is now recorded (`Result.Dropped`, written into the `DOCHT_OCR_DIAG`
sidecar) rather than silently lost, which is what made the re-measurement possible at all.

##### 2.5 Clustering lines into plates

Surviving lines are merged, in reading order, into one plate per run of vertically adjacent,
horizontally overlapping lines. A plate's box is the union of its line boxes; its font tracks the
median line height.

[..]

The reference pitch is the median **over the whole image**, not per cluster: a cluster's own pitch
is unavailable exactly when the decision is hardest - joining its second line. A pair contributes
only when it could plausibly be one text's leading: same column, moving forward, and no further than
**3** median ink heights apart (beyond that it is a section break, not leading). An image with no
measurable pitch at all falls back to the ink-box gap.

[..]

##### 2.6 Translatability filter

A cluster is kept only if there is something to translate: at least 5 letters, at least one vowel
(consonant soup is OCR noise), not wholly a URL / e-mail / bare domain / filesystem path, and -
among tokens carrying letters - at least half looking like real words. Short CJK runs bypass the
vowel and length rules.

> `text.go`: `isTranslatable` - `ocr-text.js`: `isTranslatable`

[..]

##### 2.9 Additive screen sweep for a page that *did* read

- the detector is restricted to area no plate covers (a tile more than **half** covered by one
  existing plate stops counting as evidence), so the second recognition is spent only where there is
  something to gain - and each rectangle is tested on its own, because two plates that between them
  cover a tile leave a gap down the middle, which is exactly where an unserved caption sits;
- a candidate is dropped when more than **0.2** of *its own* area is already covered, measured as
  the exact **union** of existing plates via coordinate compression - summing overlaps would
  double-count, and a per-rectangle test would let a candidate straddling two plates through;
- every failure path returns the input unchanged, because the page was already good enough to show.

[..]

#### 3. Replacement path: putting text back on the picture

##### 3.1 Geometry

Each plate is positioned in percent of natural image size - `left`, `top`, `width`, and
`min-height` (not `height`, so the plate may grow) - and its font is set in `cqw`, percent of
container width, derived from the block's median line height times a **0.92** fit factor. Percent
plus `cqw` is what makes the overlay survive responsive scaling with no JS at all: measured at ten
browser states - device scale and page zoom at 100/125/150/200 %, plus tablet and phone - the worst
plate-edge movement is 0 px of the source image.

[..]

##### 3.2 Colour: borrowing paper and ink from the source

A white patch over a coloured panel is worse than no overlay, so each plate samples the image under
it (sub-sampled to ~6000 pixels, fully transparent pixels skipped):

- **background** = the median colour over the whole block (text is the minority of its own box);
- **ink** = the median of pixels deviating from that background by more than 90 (sum of channel
  distances) within the **first line only** - real text lives there, not in imagery lower in a
  merged block. Median and not mean: the deviation test admits a glyph''s antialiased edge, and
  averaging that ramp lands between the ink and the paper (measured rgb(61,61,61) for source
  lettering of rgb(17,17,17), against rgb(7,7,7) for the median);
- if fewer than ~1.5% of samples qualify as ink, a near-black/near-white fallback is used;
- if final luma contrast is under 55, the fallback replaces the sampled ink.

##### 3.3 The ring test: which colour is the paper

The median assumes text is the minority of its own box. True of body copy in a balloon, false of
heavy display capitals, whose strokes cover more of a tight box than the paper between them - which
produced an exact colour inversion of a reported poster (cream lettering on a near-black ground).

So orientation is decided by what is **outside** the block: a band one line-height wide (min 2 px)
is sampled above, below, left and right; if more of those pixels sit nearer the ink colour than the
paper colour - with at least **40** samples voting - the pair is swapped. Outside rather than
inside, because a box drawn tightly around capitals has their strokes on its own edges and an inside
ring would answer with the ink it is supposed to be judging.

> `overlay.go`: `blockColors`, `ringNearerInk`, `samplePixels` - `ocr-overlay.js`: `blockColors`,
> `ringNearerInk`, `sampleColors`

##### 3.4 Fitting text the source never had

After layout, and again whenever the translator swaps a plate's text, each plate is re-fitted: the
box is pinned to the source region height, then the `cqw` font is stepped down (8% per iteration,
minimum 0.3, floor at 50% of base, max 40 iterations) until the content stops overflowing. If it
still overflows at the floor, the box is released to `height:auto` and grows downward - nothing is
ever clipped.

The desktop app inlines this as a small script in every overlaid page; the extension binds the same
logic to `MutationObserver`, `ResizeObserver`, `load` and `resize`. With JS disabled, the CSS
`overflow:hidden` is the degraded fallback.

> `overlay.go`: `ocrScript`, `ensureScript` - `ocr-overlay.js`: `fitPlate`, `scheduleFit`

[..]

#### 5. Shared constants

Cross-edition invariants; drift is caught by [`tests/parity_test.go`](https://github.com/SerZhyAle/doc-html-translate/blob/main/tests/parity_test.go) and
`extension/test/*.test.mjs`, which parse both codebases.

| Constant | Value | What it decides |
|----------|-------|-----------------|
| Page segmentation | PSM 3 / 11 | auto layout analysis; sparse text as a rescue rung |
| Line confidence floor | 50 (rescue: 80) | drops hallucinated text |
| Cluster pitch factor | 1.2 | how far the next line may sit and still join the plate |
| Max leading ratio | 3 | upper bound on what counts as leading when estimating pitch |
| Type size ratio | 1.6 | how far a line''s ink height may differ from its cluster''s and still join |
| Max plate coverage | 0.52 | share of the image above which a plate is a candidate for release |
| Min plate line fill | 0.72 | share of a plate''s height its own lines must fill to survive that |
| Upscale factor / DPI floor | 2 / 120 | when a low-res scan is enlarged before recognition |
| Assumed page inches | 11 | the DPI estimate's denominator |
| Min declared DPI | 70 | never declare below this; Tesseract ignores it anyway |
| Screen sigma divisor | 4.0 | pitch to Gaussian sigma for the halftone low-pass |
| Screen tile / max tiles | 64 / 96 | autocorrelation window; work cap independent of image size |
| Screen pitch bounds | 3-24 px | below is sensor noise, above is coarser than any lettering |
| Screen min energy | 3.0 | a tile flatter than this is paper or solid ink |
| Screen peak floor / vote share | 0.30 / 0.25 | autocorrelation peak vs lag 0; share of textured tiles that must agree |
| Screen tile cover max | 0.5 | how much of a tile a plate may cover before it stops being evidence |
| Screen merge max overlap | 0.2 | coverage above which a screen-pass plate is a duplicate |
| Ring min samples | 40 | votes needed before the paper/ink pair may be swapped |
| Font fit factor | 0.92 | plate font = median line height x this |

[..]

#### 7. Verification

Every threshold above was measured, and the harness is part of the source tree.

Setting `DOCHT_OCR_DIAG=<file>` makes the desktop app append one JSON line per overlaid image - the
image size and, for each block, its text, box, line height, the exact inline style and the sampled
colours. The style and colours are recomputed with the same pure functions the renderer used rather
than threaded out of it, so turning diagnostics on cannot change the rendered output (asserted by
`diag_test.go`).

### `OCR-INVOCATION` 1.0 - the shared contracts catalog, `ocr-overlay/integration-image-translate.md`, section 1

| Flag | Purpose | Recommendation |
|------|---------|----------------|
| `-ocr-lang <codes>` | Tesseract language(s) of the text in the image, e.g. `eng`, `rus`, `eng+rus`, `jpn`. Drives OCR accuracy. | Set it to the expected language. If omitted it derives from `-src`, else `eng`. |
| `-src <lang>` | Source language hint (also the fallback for `-ocr-lang`). | Optional; `-ocr-lang` is more direct. |

### Registry - the shared contracts catalog, `_meta/REGISTRY.md`, rows for doc-html-translate

Section 2, Adoption (the `OCR-OVERLAY` row of this product is not quoted: it is dated 2026-09-22 and rewriting it is step B5, local only):

| Contract | Product | Role | Implements | Reads | Verified | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| `OCR-PIPELINE` | doc-html-translate | P | 1.0 | - | 2026-09-24 | **Owner and reference implementation** (`internal/ocr`, `extension/src`). Verified this date: all 18 shared constants in section 5 match both codebases and are pinned by `tests/parity_test.go` and `tests/parity_ocr_test.go` (`TestParityOCR` 13 PASS / 0 FAIL). Detection path, rescue ladder rungs 1-4, halftone screen detection / sweep, plate geometry / colours, and re-fit loop hold across Go and extension editions. No deviation |

Section 3, Exceptions:

| Contract | Product | Deviation | Reason | Until |
| --- | --- | --- | --- | --- |
| `OCR-OVERLAY` | doc-html-translate | rule 12: the discard record is not written for an image that produced **no** plates - the case the rule exists for. The Go edition keeps the dropped lines as far as `applyOverlays` (`internal/ocr/overlay.go:419`) and then writes diagnostics only in the arm that drew plates (`overlay.go:242-251`); the extension computes `droppedLines` and no module consumes it | found by reading the code against the rule on 2026-09-22 and proven with a throwaway probe in package `ocr` (`applyOverlays: changed=false NoText=1`, no diagnostics file written). The repo's own ledger claimed this case was covered, so the gap was invisible from the inside; what landed earlier was the plumbing, not the last write. No user-visible behaviour is involved - the file is off unless `DOCHT_OCR_DIAG` is set - so it is scheduled rather than hot-fixed. Ticket `2026-09-22_ocr-discard-record-missing-for-blank-images` | 2026-12-31 |
| `OCR-OVERLAY` | doc-html-translate | rule 1: the orientation tag is read from JPEG only - a PNG `eXIf` chunk or a WebP orientation is ignored, so such an image is recognized and plated in stored space rather than display space (`internal/ocr/exif.go:60`) | a deliberate bound, not an oversight: a photo that reaches this app carrying a rotation is a JPEG in practice, and two more container parsers have no measured case behind them. It is listed here rather than only in the code because the consequence - plates in the wrong place, silently - is the one rule 1 exists to prevent. No ticket: it becomes one the day such an input is reported | 2026-12-31 |
| `OCR-OVERLAY` | doc-html-translate | rule 10, extension edition only: recognition runs with an assumed `eng` (`extension/src/defaults.js:16`) and there is no script-detection pass, so a Cyrillic or Greek page gets transliterated debris rather than no overlay and a reason. The Go edition does hold the rule (`internal/ocr/script.go`) | the browser edition's language is a popup choice and the orientation-and-script pass needs a 10 MB `osd.traineddata` the extension does not ship; the divergence is declared with its consequence in this repo's `docs/PARITY.md`. The no-text case *is* answered there ("No text found using English - if this page is in another language, pick it in the extension popup"); what is missing is the debris case, which is silent | 2026-12-31 |

## Direction B first - the catalog moves before the code

This product owns `OCR-PIPELINE` and `OCR-INVOCATION` and edits them directly (a dated amendment section,
never an in-place rewrite, plus a document-log row). `OCR-OVERLAY` has a shared owner, and the other
implementations are FastMediaSorter Android and FastMediaSorter_Lite, so a change to its rules is a
proposal beside the document.

Every step in this section edits the shared contracts catalog and runs on the owner's machine only.

1. **⛔ Local only - changes the contract catalog.** **`OCR-PIPELINE` 1.0 corrections** (log rows, no version): the `hasWordRun` pointer, the module map,
   the alpha wording, the line-box wording in §2.5, the 1.3-line ink strip, the §7 overclaim.
2. **⛔ Local only - changes the contract catalog.** **`OCR-PIPELINE` 1.1 (MINOR, additive):** the word-gap / column-order / trim stages and the word-median
   type size; the grow branch; the missing constants in §5 with a **status column** (derived with its
   report / inherited / policy); the page-OCR surface; the negative results of rule 15.
   *Part done 2026-09-25, by ticket 16 (Step 07.3):* `OCR-PIPELINE` is now **1.1** - a dated amendment
   section covering the word-gap split, the column regrouping, the new stroke test between two words and
   orphan parking, with the word-gap ratio (3.5) and the boundary reach (0.14) added to §5, and the
   registry rows bumped. Still owed from this step: the trim stage, the word-median type size, the grow
   branch, the status column, the page-OCR surface and rule 15's negative results - as **1.2**, since 1.1
   is taken.
3. **⛔ Local only - changes the contract catalog.** **Ring band width** - correction or MINOR with a notice to both consumers, per open question 3.
4. **⛔ Local only - changes the contract catalog.** **`OCR-INVOCATION` 1.1 (MINOR, clarification):** script correction and the no-plates stop for an
   omitted `-ocr-lang`; flags and exit codes unchanged.
5. **⛔ Local only - changes the contract catalog.** **`OCR-OVERLAY`:** this product's own adoption row rewritten with the per-rule result above (rule 5
   no longer "held"); new dated exceptions for rules 2, 4 (page surface), 5, 9, 12 (the extra gates), 13;
   a proposal if the owner group reads rule 5 as binding clustering only (open question 1).
   *Re-verified 2026-09-25:* not done. The adoption row is still dated 2026-09-22 and still lists rule 5
   among the rules read and held; the only exceptions for this product are rules 12 (blank image), 1
   (JPEG-only EXIF) and 10 (extension edition), quoted in the snapshot - none for 2, 4, 5, 9, 13 or the
   extra rule-12 gates. The domain `README.md` section 8 row for this product still says "rules 1-17".
6. **⛔ Local only - changes the contract catalog.** **Registry:** the `OCR-PIPELINE` row moves from `pending` to verified with this read as its evidence.
   *Done 2026-09-24 in form, not in substance:* the row is no longer `pending` - it reads verified
   2026-09-24 (quoted in the snapshot) - but its note counts 18 shared constants and says "No deviation",
   while this ticket's read found the document behind the code at every point listed in Findings. What is
   left: rewrite that note with this read as its evidence once steps 1-2 land.
7. **⛔ Local only - changes the contract catalog.** Notice to FastMediaSorter Android and FastMediaSorter_Lite in their rows' notes: what 1.1 adds and
   which prose they may have ported wrong (ring band).

## Direction A - the code holds the contracts

After the catalog change it depends on:

Items without a ⛔ mark bring the code to what the snapshot's rules already require, or add guards, and
change no contract text - they run from this repository alone. Items marked **⛔ Waits on** need a Direction
B decision or amendment first, because the pointers' "catalog first, then code" order applies to them.

1. `scaleDown` scales `Dropped` (rule 2); test asserts it. (`internal/ocr/tesseract.go:678`
   `scaleDown(res *Result, s int)` divides the kept blocks but not `res.Dropped`; the discard record's
   boxes on an upscaled image stay in 2x space. Re-checked 2026-09-25.)
2. The discard record covers every gate - translatability, screen merge, sweep - with the gate's name, in
   both editions; one merge semantic for primary + ladder drops in both editions, and PARITY says which.
   (Rule 12 and the rule-12 exception row are in the snapshot; the blank-image half stays with ticket 15.
   The merge choice is open question 4, decided in this repo and written into `docs/PARITY.md`.)
3. **⛔ Waits on B2 and open question 1 (local).** `OCR-PIPELINE` §2.5 and §3.1 describe the font from the
   median *line* height, so moving the code to the word-height median changes what the catalog document
   says and must land there first - or become a dated exception instead.
   Plate font from the word-height median in both editions (rule 5) - or a dated exception if the owner
   group decides otherwise.
4. Script detection stages the input to an ASCII path and applies the EXIF turn (rules 1, 10).
   (`internal/ocr/script.go:89` `DetectScript(bin, imgPath, dataDir)` passes `imgPath` straight to
   `tesseract --psm 0`; reuse the primary pass's staging - `stageForOCR` / `stageASCIIPath` and
   `orientImage` - as `OCR-PIPELINE` §2.1 in the snapshot describes it.)
5. **⛔ Waits on B5 (local)** for its first half: the overflow rule is written into the contract (or the
   proposal beside `OCR-OVERLAY`) before the code implements it. The browser check of the "JS disabled ->
   clipped" claim (`OCR-PIPELINE` §3.4) needs no catalog and can run now.
   Rule 9's overflow rule written into the contract and implemented; the "JS disabled -> clipped" claim
   checked in a browser.
6. Page-OCR surface: clear the layer on rotation / `object-fit` / padding it cannot place (rule 4), or an
   exception. (Surface: `extension/src/page-agent.js`, `page-ocr.js`, `page-overlay.css`. The clearing
   code runs from this repo; **⛔ Local only - changes the contract catalog.** Rule 4 also says the product
   "says so in its registry row", and the "or an exception" branch is a registry exception.)
7. **⛔ Waits on B2 and open question 5 (local).** The markers must use the vocabulary of the §5 status
   column, which does not exist yet, and whether "policy" is an admissible third status is undecided.
   Rule-13 markers on every unbracketed constant in both editions (derived with report / inherited /
   policy), matching the §5 status column.
8. Parity pins for the unpinned numbers: colour, fit ladder, column test, ring-pad rounding (Go floors, JS
   rounds - a small real drift no guard sees today). (Values to pin, as the code holds them today: colour
   fallback luma > 140 -> 17, else 240, with the min-6 ink floor; shrink ladder 8% per step, minimum 0.3,
   floor 50% of base, 40 iterations; grow branch cap 1.15 (`FONT_GROW_CAP` in
   `extension/src/ocr-plates.js`), step 4%, 20 iterations; column-overlap test 0.1. Pin beside
   `TestParityOCR` in `tests/parity_test.go` - there is no `tests/parity_ocr_test.go`, although the
   registry's `OCR-PIPELINE` note names one; pinning does not change any value.)
9. JS unit tests for `ocr-plates.js` (`fitPlate`, `plateSpecs`).
10. Stale comment in `overlay.go` ("ink is the mean") corrected. (`internal/ocr/overlay.go:539` still reads
    "ink is the mean of the pixels that stand out"; the code takes the median - `OCR-PIPELINE` §3.2.)
11. **⛔ Waits on open question 6 (local).** Whether `translation` is optional in the comparison format is
    a change to `OCR-OVERLAY` section 7, quoted in the snapshot.
    Optional, only if a consumer asks: the §7 exchange JSON, emitted beside the diagnostics line with the
    same pure-function principle.

## Done criteria

- [ ] **⛔ Local only - changes the contract catalog.** `OCR-PIPELINE` in the catalog describes every stage and constant both editions run; every §5 row
      carries a status; its log has the correction rows and the 1.1 row.
- [ ] **⛔ Local only - changes the contract catalog.** `OCR-INVOCATION` 1.1 in the catalog.
- [ ] **⛔ Local only - changes the contract catalog.** Registry: `OCR-PIPELINE` verified with a date; `OCR-OVERLAY` row carries the per-rule result for all
      17 rules; each open deviation is a dated exception.
      (Partly done 2026-09-24: the `OCR-PIPELINE` row carries a verified date, but its "No deviation" note
      predates this read - see B6. The `OCR-OVERLAY` row is unchanged since 2026-09-22.)
- [ ] **⛔ Local only - changes the contract catalog.** Both consumer rows carry the notice.
- [ ] Code items 1-10 landed, or each carried by a dated exception; `scripts/test.ps1` and the extension
      tests green, output cited. (Items 3, 5, 7 wait on local steps; a "dated exception" is itself a
      local registry edit.)
- [ ] **⛔ Waits on B2 / B4 (local).** Pointer files updated to 1.1 where the version moved. (The edit is
      in this repo - `docs/contracts/OCR-PIPELINE.md`, `OCR-INVOCATION.md`, `OCR-OVERLAY.md`, all at 1.0
      on 2026-09-25 - but only after the catalog version moved.)
- [ ] `docs/PARITY.md` handled for every behaviour change.
- [ ] **⛔ Local only - changes the FastMediaSorter_Lite repository.** The FastMediaSorter Lite spec sync
      obligation (standing practice: OCR changes are mirrored into its overlay-accuracy spec) handled for
      every behaviour change.

## Open questions

1. **⛔ Local only - changes the contract catalog.** Rule 5: does "type size is the median of word heights" bind the plate font, or only clustering? Change
   the code, or propose a wording to the `OCR-OVERLAY` owner group?
   (Evidence in the catalog as of 2026-09-25: FastMediaSorter_Lite read it as binding the font - its
   closed rule-5 exception moved its plate font to the word-height median and bumped its cache version.)
2. **⛔ Local only - changes the contract catalog.** If the font basis changes, MINOR ("tightening what rule 5 already required") or MAJOR? Apply
   `VERSIONING.md` §3's test to FastMediaSorter Android and Lite.
3. **⛔ Local only - changes the contract catalog.** Did either consumer port the "one line-height" ring band from prose? That decides correction vs MINOR
   with a notice.
   (Partly answered 2026-09-25 from the catalog: FastMediaSorter Android's transfer verdict,
   `ocr-overlay/ocr-overlay-accuracy.md` §5, records the ring as "`1/3` line height per side, floor 2 px,
   `>= 40` votes", adopted in S1714 with a 16 px ceiling of its own - the code's value, not the prose's.
   The exchange record (`ocr-overlay/doc2html-ocr-positioning-exchange.md`) repeats "one line height
   wide". FastMediaSorter_Lite's registry row names an outside ring with a minimum vote count but not its
   width; its source has to be read on the owner's machine.)
4. Rule 12: every gate recorded with a gate-name field (FastMediaSorter_Lite does this)? Merge primary +
   ladder drops (Go today) or keep the winner's only (JS today, PARITY's wording)?
   (Decided in this repo - rule 12 already requires "which threshold it failed", so a gate-name field
   changes no contract text; the merge semantic is recorded in `docs/PARITY.md`.)
5. **⛔ Local only - changes the contract catalog.** Rule 13 for policy constants (translatability, 0.92, padding): accept a third "policy" status, or force
   derived / inherited?
6. **⛔ Local only - changes the contract catalog.** §7 exchange JSON: is `translation` optional; emit from the Go diagnostics, the lab runner, or both?

## Notes

`tesseract.go` (1407 lines), `overlay.go` (828) and `ocr-overlay.js` (611) are over the file budget and
most code items touch them. The 386 toolchain's memory ceiling can kill `go test ./tests/` on a first run;
a rerun that passes is not a regression.

## Implementation record (2026-09-25, cloud session)

| Item | Result |
|---|---|
| A1 | `scaleDown` scales `Result.Dropped` with the plates; `TestScaleDownScalesTheDiscardRecord`. |
| A2 | Every gate is recorded with a `gate` field in both editions: `confidence` (`keepLine`), `translatable` (each line of a cluster `isTranslatable` refused, recorded inside `clusterLinesRecording` / `clusterLines(.., dropped)` where the decision is taken), `screen-merge` (a sweep plate `mergeScreenBlocks` refused, with its lines' mean confidence - new `Block.Conf` / block `conf`). The sweep's own floor drops are recorded too. Merge semantic (open question 4), decided and written into `docs/PARITY.md`: the ordinary pass's drops, then the ladder's or the sweep's - the extension used to replace the ordinary pass's drops with the ladder's. The diag line gains `"gate"`; both pinned literals moved. Guarded by `TestParityOCRDiscardGates`. |
| A4 | Already done before this session (commit `c2cfa21`, `stageForDetection`, `detect_stage_test.go`). |
| A5 (browser half) | The "JS disabled -> clipped" claim of `OCR-PIPELINE` §3.4 is **false in the reassuring direction**: nothing clips, the plate carries only `min-height` so `overflow:hidden` never engages, and the box grows at the unfitted size - 244 px over a 39 px source region. Evidence: [`DEV/research/page_ocr_placement_2026-09-25`](../research/page_ocr_placement_2026-09-25/README.md) §2. The catalog correction joins step B1 (local); the code comment in `overlay.go` and `docs/PARITY.md` are corrected. |
| A6 (code) | The page agent places the layer over the picture as drawn (`ocr-plates.js` `pictureBox`: border, padding, `object-fit`, `object-position`, a scaled ancestor, `clip-path` to what the element shows) and clears it for a picture rotated, skewed or mirrored by itself or an ancestor (`transformRotates`). Measured in Chromium against the real agent: every placeable case clean, where HEAD left half a `cover` picture's text uncovered and put a mirrored picture's plate on the other half (evidence §1). The registry half - "says so in its registry row" - stays local. |
| A8 | Colour numbers are named constants on both sides and pinned by `TestParityOCRPlateColourNumbers`; the fit ladder by `TestParityOCRFitLadder`; the column test by `TestParityOCRColumnTest`. Two real drifts fixed in the extension to the desktop's integer arithmetic: the ring band and the ink strip were rounded (now floored), and `luma` was fractional against `140` / `55` (now truncated). The extension also clamped the line height to the box, which the desktop does not. |
| A9 | `extension/test/ocr-plates.test.mjs`: `plateSpecs`, `renderPlates`, `fitPlate` (grow to the cap, stop a step before overflow, shrink above the floor, release instead of clip, re-fit from the base), `pictureBox`, `transformRotates`. |
| A10 | The `overlay.go` comment says median; `samplePixels`' "fully transparent" comment now says alpha < 128. |

Checks (Linux): `go vet ./...` (also `GOOS=windows`), `go test ./...`, extension `npm test` 262 pass / 0
fail. Mutation check of the new parity pins: a changed `INK_MIN_SAMPLES`, fit iteration bound and
column multiplier each fail their test.

For step B1, found by this session: the §3.4 JS-off sentence above.
