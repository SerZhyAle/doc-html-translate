# OCR overlay pipeline: how image text is found and put back

Audience: developers who need the *mechanism*, not the CLI contract - porting it, reviewing it, or
building the same feature elsewhere (e.g. FastMediaSorter Lite). For how to **call** the app on an
image, see [integration-image-translate.md](integration-image-translate.md). For the cross-edition
invariant tables, see [PARITY.md](PARITY.md).

There are two independent implementations, no shared code: the Go desktop app (`internal/ocr`) and
the browser extension (`extension/src/ocr-*.js`). They are held to the same constants and the same
decision order by guard tests that parse both codebases.

---

## 1. The idea

The output is never a re-drawn image. Recognition produces text plus a bounding box per text region;
the renderer then wraps the original image in a positioned container and lays one opaque HTML
element - a **plate** - over each region, carrying the recognized string as real text.

Because a plate is an ordinary DOM text node, the translation layer (the app's own translator, or
the browser's built-in "Translate page") rewrites it exactly as it rewrites body copy, and the
translated string lands back on the picture at the coordinates the source text occupied. No pixel of
the image is modified.

One consequence drives every geometry decision downstream: the plate is sized from the **source**
geometry, but the text poured into it is a **different length** after translation. Percent
positioning, container-relative font units and the runtime re-fit loop all exist to absorb that
mismatch without clipping and without covering artwork the source text did not cover.

---

## 2. Detection path

### 2.0 The language the page is read with (desktop only)

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

### 2.1 Resolution estimate and staging

Raw pixel count is a bad gate - a page scan clears 1000 px tall even at a poor ~100 DPI, so a pixel
threshold either upscales everything (4x the OCR cost on a clean render that gains nothing) or
nothing (the low-res scan that needs it most). Instead the long side is divided by an assumed
11-inch page to estimate DPI:

- below **120 DPI** the image is resampled **2x** before recognition (Catmull-Rom in Go, canvas in
  JS) and every coordinate is divided back afterwards;
- in every case the resolution is **declared** to the engine (floor 70). This is what makes layout
  analysis separate adjacent regions - two speech balloons - instead of merging them; Tesseract's
  own estimate on a bare page scan runs far below reality (~70 DPI for a ~180 DPI page).

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
> [`exif.go`](../internal/ocr/exif.go): `exifOrientation`, `orientImage` -
> `ocr-overlay.js`: `estimateDpi`, `upscaleForOcr`

### 2.2 Primary recognition pass

Go shells out to the `tesseract` binary with `--psm 3`, `-c tessedit_create_tsv=1` and
`user_defined_dpi`, reading TSV from stdout. JS drives a lazily created tesseract.js worker, one per
language, pinned to the same PSM 3.

PSM matters: tesseract.js defaults to PSM 6 (single block), which reads an illustrated or scanned
page as one text block - folding scene edges into the recognized string and mis-merging regions.
PSM 3 runs real layout analysis.

> `tesseract.go`: `recognizePass`, `tesseractArgs` - `ocr-overlay.js`: `getWorker`, `recognize`

TSV is requested via `-c tessedit_create_tsv=1`, not the `tsv` config file: `--tessdata-dir`
redirects config lookup into the app's bundled data dir, which ships traineddata only, so the config
is not found and tesseract silently falls back to plain text (zero blocks, no overlay).

### 2.3 TSV to lines

The engine emits five levels: page, block, paragraph, line, word. Only three are read - page (image
size), line (the box) and word (text and confidence). Block and paragraph boxes are **deliberately
discarded**: trusting them makes an opaque plate span imagery the engine folded into a "paragraph".
Each line accumulates its words' text and the running mean of their confidences.

> `tesseract.go`: `parseTSV`, `ocrLine` - `ocr-overlay.js`: `collectLines`

### 2.4 Confidence gate

A line whose mean word confidence is below **50** is dropped. Real text scores ~80-97; what the
engine hallucinates out of a drawing scores 0-50. Without the gate, noise becomes an opaque plate
over the artwork, and its oversized boxes also inflate the plate font. Rescue passes (2.6 onward)
run against a stricter floor of **80** - a rescue is a second guess, so the prior that there is text
at all is weaker, and invented words painted over artwork are worse than no overlay.

### 2.5 Clustering lines into plates

Surviving lines are merged, in reading order, into one plate per run of vertically adjacent,
horizontally overlapping lines. A plate's box is the union of its line boxes; its font tracks the
median line height.

Adjacency is judged on the **line pitch** - top-to-top distance - not on the gap between ink boxes,
and the distinction is load-bearing. All-caps lettering with no descenders boxes far shorter than
the line it came from: measured on the lab scene `synth-balloon-on-panel`, a balloon drawn with
36 px leading recognizes as 14-17 px ink boxes with 19-22 px between them, so a gap tested against
1.2 x 14 px broke one balloon into three plates - three sentence fragments handed to a translator
with no sentence to work with. The same balloon's pitch is a flat 36 px, well inside 1.2 x 36. Pitch
also avoids the opposite failure: on `synth-adjacent-balloons` the pitch inside a balloon is 26 px
and the step across to the next is 42 px, so the two stay two plates.

The reference pitch is the median **over the whole image**, not per cluster: a cluster's own pitch
is unavailable exactly when the decision is hardest - joining its second line. A pair contributes
only when it could plausibly be one text's leading: same column, moving forward, and no further than
**3** median ink heights apart (beyond that it is a section break, not leading). An image with no
measurable pitch at all falls back to the ink-box gap.

Proximity is not the whole test, because on a page with separated regions there is no single pitch to
be near. A line also has to be the **same type size** as the cluster it would join: its ink height
within **1.6** of that cluster's own median height, either way round. Measured on
`poster-display-type-on-flat-colour`, the headline stands 168 px from the next recognized line while
two body lines a reader takes as one sentence stand 190 px apart - so no pitch bound cuts in the
right place, and the whole poster came out as one plate spanning 62% of its height. Their type sizes
do cut there: 281 px of display ink against a 155 px body median. The ratio is bracketed by the
corpus's hand-drawn line boxes - the widest spread inside one text is 1.42x
(`samson-and-delilah-03-scroll`, 19 lines of one caption) and the narrowest step between two texts is
1.86x - and 1.6 is the geometric middle of the two.

The last rule looks at the page instead of at a line''s neighbours, because a page can defeat all
three above at once. A form, a list, an application window carries **one type at one pitch in one
column**, so nothing in its typography separates its regions and the whole page arrives as a single
plate - measured on `accounts.jpg`, one plate over **80.6%** of the picture, its 268 characters set
large enough to fill it and the screenshot no longer visible behind it. A finished cluster is
therefore **released into one plate per line** when it is both too big and too loose: its box covers
more than **0.52** of the image *and* its own line boxes account for less than **0.72** of the box''s
height. Both conditions are required - size alone would release `samson-and-delilah-03-scroll`, a crop
whose one annotated caption covers 0.6087 of it; looseness alone would release `synth-uniform-paper`,
three body lines whose 0.6667 fill is the leading a paragraph has. Released and not refused: each line
keeps its own text and box, so every recognized word still reaches a plate, and the released lines
skip the translatability filter below because the assembled text already passed it. `accounts.jpg`
comes out as 7 plates whose largest covers 7.67%.

Both bounds are bracketed from opposite directions over the 46 lab scenes and the 13 hand-drawn
annotations ([`DEV/research/ocr_plate_coverage_2026-08-13.md`](../DEV/research/ocr_plate_coverage_2026-08-13.md)).
The *area* version of the fill - how much of a plate''s box its lines cover - was measured on the same
run and separates nothing (0.5891 for the defect against 0.4582 for a legitimate balloon), because
area mixes leading with ragged right edges; the rule is stated on the vertical axis, which is the axis
separated regions are separated on.

> `tesseract.go`: `clusterLines`, `medianLinePitch`, `sameTypeSize`, `releaseOversized` -
> `ocr-cluster.js`: `clusterLines`, `medianLinePitch`, `sameTypeSize`, `releaseOversized`

### 2.6 Translatability filter

A cluster is kept only if there is something to translate: at least 5 letters, at least one vowel
(consonant soup is OCR noise), not wholly a URL / e-mail / bare domain / filesystem path, and -
among tokens carrying letters - at least half looking like real words. Short CJK runs bypass the
vowel and length rules.

> `text.go`: `isTranslatable` - `ocr-text.js`: `isTranslatable`

### 2.7 Rescue ladder - only when nothing read at all

Colour is where recognition silently dies. Tesseract's default thresholder runs Otsu on each RGB
channel separately and counts a pixel as ink only where all three agree. On flat paper the channels
agree and this is harmless; on saturated artwork - a brick-red comic panel behind a white balloon, a
coloured poster - each channel splits the picture somewhere else, the channels disagree over the
lettering, and the mask that reaches recognition contains no text.

The ladder runs over a **luminance** copy, and each rung changes exactly one decision:

| Rung | What changes | Why |
|------|--------------|-----|
| 1 | grey + engine-default Otsu | one thresholding decision instead of three per-channel ones |
| 2 | grey + Leptonica tiled Otsu (`thresholding_method=1`) | thresholds locally, so a caption over a sky gradient stops vanishing into it |
| 3 | grey + PSM 11 (sparse text) | no layout analysis - for input that is not a page, e.g. display lettering on a poster |
| 4 | halftone screen pass (§2.8) | the only rung that alters the pixels, so it goes last |

All rungs run and the **strongest** result wins, where strength is the number of words placed on the
image; ties keep the incumbent (a retry is a second look at the same pixels, not a second piece of
evidence). First-non-empty-wins was the earlier rule and it measurably lost: on
`poster-display-type-on-flat-colour` rung 1 returned `| МОЖЕМ` and ended the search before the
sparse rung could return six lines of poster lettering.

The ladder is a ladder rather than a replacement because no rung is best everywhere - the colour
pass wins where lettering is separated by hue rather than brightness. Retrying only after the
ordinary pass returned nothing keeps every scene that works today byte-identical and spends the
extra shell-outs only where there is nothing to lose.

> `tesseract.go`: `greyRescue`, `greyRescuePasses`; `strength.go`: `resultStrength`,
> `strictlyBetter` - `ocr-overlay.js`: `greyRescue`, `GREY_RESCUE_PASSES`; `ocr-cluster.js`:
> `resultStrength`, `strictlyBetter`

### 2.8 Halftone screen detection and low-pass

A press screen - the dot lattice used to print a tone - is what no thresholder can see through: its
own tone sits between ink and paper, so the mask either swallows the lettering with the dots or
turns the picture into texture.

The detector needs no frequency transform, which is what makes it cheap enough to run inside a
rescue:

1. up to **96** tiles of **64x64** are spread over the picture (the step scales with image size, so
   a 20-megapixel page costs the same as a panel);
2. each tile is high-passed - pixel minus its 3x3 mean - and rejected if the residual energy is
   below **3** (paper or solid ink, nothing to find);
3. the residual is autocorrelated along both axes summed together (a press screen is rotated, so
   neither axis alone carries the whole signal);
4. the winning lag must be a **local maximum** (a smooth gradient decorrelates monotonically and
   would otherwise report the smallest lag), with a peak of at least **0.30** of lag 0, landing in
   **3-24 px**;
5. at least **a quarter** of the textured tiles must vote for the same lag. That cross-tile
   agreement is what separates a press screen from JPEG blocking, film grain or engraved hatching.

The measured pitch then sets the kernel: a separable Gaussian with **sigma = pitch / 4**, truncated
at 3 sigma. In the extension this is CSS `blur(Npx)`, which is the same Gaussian. Gaussian rather
than a box mean because the screen must be suppressed at its harmonics too, and a box kernel's stop
band has holes. The intermediate buffer is float32, not float64: a 600-DPI book page is 20
megapixels and the app builds for 386, so the process has a 2 GB ceiling.

> `screen.go`: `screenPitch`, `tilePeriod`, `tileResidual`, `gaussBlurGray` - `ocr-screen.js`:
> `screenPitch`, `tilePeriod`, `tileResidual`; `ocr-overlay.js`: `greyRendition`

### 2.9 Additive screen sweep for a page that *did* read

The ladder fires only when an image produced no plates at all - which never happens on a real comic
page, where balloon dialogue on clean white reads fine while a caption printed as a tint does not.
The page is never "unread", the rung never runs, and the caption stays untranslated with no
explanation.

So the screen pass has a second entry point on the opposite branch, and it is **additive**: every
plate from the primary pass survives untouched, and screen-pass plates are appended only where they
are mostly uncovered. Three parts make that safe:

- the detector is restricted to area no plate covers (a tile more than **half** covered by one
  existing plate stops counting as evidence), so the second recognition is spent only where there is
  something to gain - and each rectangle is tested on its own, because two plates that between them
  cover a tile leave a gap down the middle, which is exactly where an unserved caption sits;
- a candidate is dropped when more than **0.2** of *its own* area is already covered, measured as
  the exact **union** of existing plates via coordinate compression - summing overlaps would
  double-count, and a per-rectangle test would let a candidate straddling two plates through;
- every failure path returns the input unchanged, because the page was already good enough to show.

The trade was measured over the corpus: the screen pass wins on screened material (+47% and +18%
confident words on two real scenes, and a caption sentence that finally completes) and loses badly
where there is no screen (16 confident words down to 0 on one cover, 71 down to 49 on a poster).
Additive merging takes the gain without the loss.

> `tesseract.go`: `screenSweep`; `screen.go`: `screenPitchOutside`, `mergeScreenBlocks`,
> `coveredFraction` - `ocr-overlay.js`: `screenSweep`; `ocr-screen.js`: `mergeScreenBlocks`,
> `coveredFraction`

---

## 3. Replacement path: putting text back on the picture

### 3.1 Geometry

The image is wrapped in a **block** container carrying `aspect-ratio: W / H` and
`container-type: inline-size`. Block with an explicit width, not inline-block: a shrink-to-fit box
with size containment collapses to zero inline size and hides the image and every plate. The image
is `width:100%` with `margin:0` and `max-height:none` so a host page's `img` reset cannot offset or
shrink it and drift the plates.

Each plate is positioned in percent of natural image size - `left`, `top`, `width`, and
`min-height` (not `height`, so the plate may grow) - and its font is set in `cqw`, percent of
container width, derived from the block's median line height times a **0.92** fit factor. Percent
plus `cqw` is what makes the overlay survive responsive scaling with no JS at all: measured at ten
browser states - device scale and page zoom at 100/125/150/200 %, plus tablet and phone - the worst
plate-edge movement is 0 px of the source image.

That whole property rests on the image filling the container, because the percentages are of the
container and the measurement divides by the image. Anything that sizes the picture independently
breaks it silently - the page's own navbar script used to write an inline `width` on every image,
which beats `.ocr-fig>img{width:100%}`, and a 640 px scene in a 1216 px column rendered every plate
1.9x too large and off its text while still measuring 0 drift, because it was equally wrong
everywhere. The guard now skips images inside `.ocr-fig`.

**The opaque paper is on the plate box**, and the text sits directly in it. This is a decision taken
against the corpus, not a default: the paper spent one day (2026-08-13) on an inline span hugging the
string, so that it took the shape of the rendered words rather than of the block rectangle. That shape
is genuinely better where the plate is much wider than its last line - the rectangle put 91 px of paper
over a photograph on either side of a 984 px caption''s 759 px last line - but a plate exists to
*conceal* what it replaces, and over the 46 lab scenes the string carrier left a mean **93%** of the
source lettering still showing against **17%** for the box, against a recorded bound of 0.28. Both
layers then read at once and the page is harder to read than the untouched scan. The box''s over-cover
is what the coverage rule in 2.5 and the type-size rule bound from the other side.

> `overlay.go`: `wrapImage`, `percentStyle`, `ocrCSS` - `ocr-overlay.js`: `buildOverlay` +
> `ocr-overlay.css`

### 3.2 Colour: borrowing paper and ink from the source

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

### 3.3 The ring test: which colour is the paper

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

### 3.4 Fitting text the source never had

After layout, and again whenever the translator swaps a plate's text, each plate is re-fitted: the
box is pinned to the source region height, then the `cqw` font is stepped down (8% per iteration,
minimum 0.3, floor at 50% of base, max 40 iterations) until the content stops overflowing. If it
still overflows at the floor, the box is released to `height:auto` and grows downward - nothing is
ever clipped.

The desktop app inlines this as a small script in every overlaid page; the extension binds the same
logic to `MutationObserver`, `ResizeObserver`, `load` and `resize`. With JS disabled, the CSS
`overflow:hidden` is the degraded fallback.

> `overlay.go`: `ocrScript`, `ensureScript` - `ocr-overlay.js`: `fitPlate`, `scheduleFit`

### 3.5 Where translation happens

In the desktop pipeline the overlay is applied **before** the translation step, so plate text is
translated by the same engine as the rest of the book
([`pipeline.go`](../internal/pipeline/pipeline.go)). In the extension the plate is left in the
source language and the document is tagged via `<html lang>` (Tesseract code mapped to BCP-47),
which is what makes the browser offer to translate the page.

---

## 4. Execution model: where the two editions deliberately differ

| Concern | Go desktop app | Browser extension |
|---------|----------------|-------------------|
| When OCR runs | eagerly at conversion time - nothing is readable until the file is written | lazily on scroll, via `IntersectionObserver` |
| Concurrency | process pool, `NumCPU-2` capped at 16; each Tesseract process pins ~one core | one shared worker, single-flight FIFO queue |
| Batching boundary | the whole book - images deduped by absolute path across all content files, so multipage mode does not degrade the pool to serial | the viewport |
| Grey rendition | 8-bit `image.Gray` written to a temp PNG | canvas `filter: grayscale(1)`, stays RGBA (equivalent: per-channel Otsu over three equal channels is one decision) |
| Coordinate downscale | after the sweep, so sweep rectangles need no rescaling | inside `collectLines`, so the sweep multiplies covered rects back up |
| Language data | `<exe>/tessdata`, downloaded from tessdata_fast 4.0.0 (GitHub raw); `eng` bundled | `tessdata.projectnaptha.com/4.0.0_fast` (gzip); `eng` vendored |
| Failure policy | identical both sides: best-effort everywhere. A missing engine, an unreadable page or a failed image never aborts the run, and "read fine, no text" is tracked apart from "recognition failed" ||

---

## 5. Shared constants

Cross-edition invariants; drift is caught by [`tests/parity_test.go`](../tests/parity_test.go) and
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

---

## 6. Module map

| Responsibility | Go | JavaScript |
|----------------|----|------------|
| Engine invocation, staging, ladder, clustering | [`internal/ocr/tesseract.go`](../internal/ocr/tesseract.go) | [`extension/src/ocr-overlay.js`](../extension/src/ocr-overlay.js) |
| EXIF orientation into display space | [`internal/ocr/exif.go`](../internal/ocr/exif.go) | inherited from `createImageBitmap` |
| Line-to-plate clustering, rung comparator | `tesseract.go`, [`strength.go`](../internal/ocr/strength.go) | [`ocr-cluster.js`](../extension/src/ocr-cluster.js) |
| Halftone detection, blur, additive merge | [`internal/ocr/screen.go`](../internal/ocr/screen.go) | [`extension/src/ocr-screen.js`](../extension/src/ocr-screen.js) |
| Plate rendering, colours, re-fit | [`internal/ocr/overlay.go`](../internal/ocr/overlay.go) | `ocr-overlay.js` + `ocr-overlay.css` |
| Noise / translatability filter | [`internal/ocr/text.go`](../internal/ocr/text.go) | [`extension/src/ocr-text.js`](../extension/src/ocr-text.js) |
| Language catalog and data download | [`internal/ocr/tessdata.go`](../internal/ocr/tessdata.go) | [`extension/src/ocr-lang.js`](../extension/src/ocr-lang.js) |
| Per-image diagnostics (opt-in) | [`internal/ocr/diag.go`](../internal/ocr/diag.go) | [`extension/src/diagnostics.js`](../extension/src/diagnostics.js) |
| Call site / scheduling | [`internal/pipeline/pipeline.go`](../internal/pipeline/pipeline.go) | [`extension/src/viewer.js`](../extension/src/viewer.js) |
| Cross-edition contract | [PARITY.md](PARITY.md), [`tests/parity_test.go`](../tests/parity_test.go) ||

---

## 7. Verification

Every threshold above was measured, and the harness is part of the source tree.

Setting `DOCHT_OCR_DIAG=<file>` makes the desktop app append one JSON line per overlaid image - the
image size and, for each block, its text, box, line height, the exact inline style and the sampled
colours. The style and colours are recomputed with the same pure functions the renderer used rather
than threaded out of it, so turning diagnostics on cannot change the rendered output (asserted by
`diag_test.go`).

```
DOCHT_OCR_DIAG=temp/ocr.jsonl doc-html-translate -ocr -ocr-lang rus book.cbz

{"file":"page012.jpg","width":1600,"height":2400,"blocks":[
  {"text":"WE CAN JUST TALK","x0":412,"y0":233,"x1":905,"y1":361,"lineH":42,
   "style":"left:25.75%;top:9.71%;width:30.81%;min-height:5.33%;font-size:2.42cqw",
   "background":"rgb(247,244,236)","ink":"rgb(24,22,20)"}]}
```

The visual-fidelity lab ([`tools/ocrlab`](../tools/ocrlab/README.md)) grades both editions against
one corpus with one metrics package and one shared evidence schema.

Measurements behind the two most recent passes:
[`DEV/research/ocr_halftone_2026-08-12.md`](../DEV/research/ocr_halftone_2026-08-12.md) (screen
detection and the additive sweep),
[`DEV/research/ocr_display_lettering_2026-08-12.md`](../DEV/research/ocr_display_lettering_2026-08-12.md)
(sparse rung, strength comparator, colour-orientation ring test) and
[`DEV/research/ocr_grey_rescue_2026-08-11.md`](../DEV/research/ocr_grey_rescue_2026-08-11.md) (the
grey ladder).
