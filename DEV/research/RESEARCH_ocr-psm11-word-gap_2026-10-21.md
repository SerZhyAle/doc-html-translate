# The word-gap rules at PSM 11: the 3.5 cut holds, the stroke test holds, the tall-word cut does not

2026-10-21. Phase A (OC-2) of [`DEV/plan/done/E0121_ocr-owed-evidence-and-rule-10-reverification.md`](../plan/done/E0121_ocr-owed-evidence-and-rule-10-reverification.md).
Follows [`ocr_word_gap_2026-09-12.md`](ocr_word_gap_2026-09-12.md) (PSM 3 only) and
[`ocr_balloon_boundary_2026-09-25.md`](ocr_balloon_boundary_2026-09-25.md). No constant was moved and no
code was changed by this phase.

## Question

`splitWideGaps` (`internal/ocr/tesseract.go`, `ocr-cluster.js`) runs on every recognizer line of every rung and
applies three cuts: a ratio cut (`ocrMaxWordGapRatio` 3.5 median word heights), a stroke test between two
words (`ocrBoundaryReach` 0.14) and a tall-word cut (a word box over `ocrTypeSizeRatio` 1.6 x the line's median
word height, OCR-PIPELINE amendment 1.7 A). All three were bracketed at PSM 3. The pipeline reaches PSM 11
only as the third grey rescue rung (floor 80 plus the anchored rescue admission), but the cuts act there too.
Does 3.5 still stand at PSM 11, and where do the other two cuts act?

## Answer

- **3.5 holds at PSM 11, in the weak sense.** The ratio cut never cut an honest line in any of the four
  populations (two engines x two renditions), at either floor. The widest honest gap is **2.14x** (50 floor) and
  **1.50-2.13x** at the shipped floor 80, below the PSM 3 figure of 2.57x. The stitch side is nearly empty:
  PSM 11 does not walk across a figure, so there is almost nothing for the cut to repair (`test_doc/1.png`:
  5-6 cut lines at PSM 3, **0** at PSM 11). Only three true stitches stand over 3.0x: 3.22x, 3.43x (both under the cut,
  caught by the tall-word cut) and 3.57x (over it). The PSM 3 bracket (2.57 / 4.80) is neither narrowed nor
  contradicted; the stitch side at PSM 11 is n=3.
- **The stroke test holds.** It fired only on outline tokens, artwork and noise at PSM 11 and never on an honest
  line (0 of 30 labelled firings).
- **The tall-word cut does not hold.** It fires on honest lines more often than on stitches, at PSM 11 and at the
  PSM 3 default alike, and no ratio separates the two populations (honest 1.62-3.54x, stitches 1.62-7.09x). It
  visibly damages real plates (section "The tall-word cut"). This is new evidence against an amendment of the
  catalog, so it goes to the catalog first (rule 13); nothing is changed here.

## Instrument

Both engines read the 47 lab scenes of `DEV/ocrlab/corpus.json` plus `test_doc/1.png` (48 images, `eng`, the
same recipe as the 2026-09-12 note). Three renditions per image:

| Variant | What it reads | Why |
|---|---|---|
| `p3` | the staged colour frame, PSM 3 | control: reproduces the earlier notes |
| `p11c` | the same staged colour frame, PSM 11 | PSM 11 on the ordinary rendition |
| `p11g` | the grey rendition, PSM 11, engine-default thresholder | what rescue rung 3 actually hands the engine |

- **Desktop edition:** tesseract 5.4.0.20240606, the app's own staging (`prepareForOCR`: upscale below the DPI
  floor, declared DPI), words from the level-4/5 TSV rows, the shipped `strokeBetween` on the frame's luminance.
  A throwaway Go test inside `internal/ocr` (deleted after the run, kept as text under `temp/ocr-psm11/`).
- **Extension edition:** tesseract.js 7.0.0 with tesseract.js-core 6.1.2 in Node and the vendored `eng` pack,
  the extension's own upscale rule and luma weights, `blocks -> paragraphs -> lines -> words` as `collectLines`
  reads them, the extension's own `strokeBetween` and constants imported from `ocr-cluster.js`. Not run in a
  browser: no EXIF orientation and no `createImageBitmap`; the grey rendition is the luma written into RGB.

Control: `p3` on `synth-two-columns` returns the 12.41x / 12.35x / 17.62x of the 2026-09-12 note on the desktop
engine (11.93 / 11.43 / 16.36 on tesseract.js), and `test_doc/1.png` 99.9x / 86.1x at the widest, so the harness
reads what the earlier notes read.

### What every multi-word line records

Per line: confidence, word count, median word height, the ratio of every consecutive word pair and whether
`strokeBetween` fires on it, how many words stand over 1.6 x the median, and whether the ratio cut would fire.
Raw tables are under `temp/ocr-psm11/` (`desktop/desktop-eng.jsonl`, `ext/ext-eng-all.jsonl`, summaries in
`RESULTS.txt`, line crops under `*/crops/`).

## The distribution

Multi-word lines, and the gaps (consecutive word pairs) of the lines at or over the floor:

| Engine | Variant | Lines | Over 50 | Over 80 | Gaps over 50 |
|---|---|---:|---:|---:|---:|
| tesseract 5.4 | `p3` | 329 | 240 | 156 | 1059 |
| tesseract 5.4 | `p11c` | 843 | 394 | 246 | 1193 |
| tesseract 5.4 | `p11g` | 854 | 384 | 233 | 1071 |
| tesseract.js 7 | `p3` | 337 | 239 | 148 | 1017 |
| tesseract.js 7 | `p11c` | 888 | 401 | 244 | 1081 |
| tesseract.js 7 | `p11g` | 912 | 417 | 240 | 1105 |

PSM 11 returns a line per text fragment, so it returns many more, shorter lines. Gaps of the lines over the
floor of 50, by ratio bucket (the upper edge of each bucket; the last bucket is everything over 10):

| Engine / variant | 0.5 | 1 | 1.5 | 2 | 2.5 | 3 | 3.5 | 5 | 10 | over 10 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| tesseract 5.4 `p3` | 513 | 389 | 50 | 17 | 8 | 11 | 9 | 10 | 19 | 33 |
| tesseract 5.4 `p11c` | 714 | 432 | 35 | 4 | 2 | 0 | 2 | 1 | 3 | 0 |
| tesseract 5.4 `p11g` | 599 | 430 | 29 | 7 | 3 | 0 | 1 | 0 | 1 | 1 |
| tesseract.js `p3` | 520 | 364 | 38 | 8 | 5 | 8 | 6 | 7 | 16 | 45 |
| tesseract.js `p11c` | 601 | 424 | 42 | 8 | 1 | 1 | 2 | 0 | 2 | 0 |
| tesseract.js `p11g` | 615 | 435 | 32 | 13 | 2 | 1 | 2 | 1 | 2 | 2 |

At PSM 3 the population has a long upper tail (62-68 gaps over 3.5). At PSM 11 it has none: 2-5 gaps over 3.5,
and every one of them is a line of two glyphs (`= =`, `- -`, `ae 4`) except one true stitch.

## Labelling rule and sample

Same idea as the 2026-09-12 and 2026-09-25 notes: the lines a verdict could depend on were labelled by eye from
a crop of the scene. The sample is every line over the floor of 50 that a cut fires on, plus every line whose
largest ratio is 1.4 or more. A line under 1.4 that no cut fires on is not labelled, so a stitch hiding there is
invisible to this method (as in 2026-09-25).

- **R** - the engine line is one line of one text region in one reading row (a misread, or an artefact token at
  its edge, does not change that). A cut splits one sentence of a balloon, caption or headline into plates.
- **S** - the line joins two regions (balloons, notices, a headline and its neighbour), or joins text with an
  outline, rule or artwork token as one of its words, or merges two rows. The cut is right.
- **X** - no text: glyph pairs, hatching, grain. A cut is harmless.
- **Q** - undecidable by eye; excluded from every count.

Sample: 34 + 31 lines (desktop `p11c`, `p11g`), 40 + 43 (tesseract.js), plus the 20 tall-word-only lines of the
desktop `p3` control: 168 labels, the same physical line often appearing in two to four variants. About 30 of
the 168 are `X` labels read from the line's text alone, without looking at the crop: every one is one to three
glyphs or punctuation (`= =`, `x e`, `{ 4}`). The labels are one reader's (`temp/ocr-psm11/labels.json`, ordered as
`manifest-*.json`); the contact sheets are in `temp/ocr-psm11/sheets/`.

## Where each cut acts

Lines over the floor of 50. Entries are `R / S / X` (honest lines cut / stitches cut / noise cut):

| Cut | `p11c` desktop | `p11g` desktop | `p11c` tesseract.js | `p11g` tesseract.js |
|---|---|---|---|---|
| ratio over 3.5 | 0 / 1 / 3 | 0 / 0 / 2 | 0 / 0 / 2 | 0 / 0 / 5 |
| stroke between two words (not already cut by ratio) | 0 / 3 / 4 | 0 / 3 / 3 | 0 / 4 / 5 | 0 / 4 / 4 |
| any word over 1.6 x the median | **11** / 3 / 1 | **11** / 2 / 0 | **11** / 5 / 1 | **12** / 3 / 5 |

At the shipped floor of 80 the ratio cut fires nowhere but on two noise lines of the tesseract.js grey rendition
(`= =` at 16.4x, `A As` at 4.1x); the stroke test
fires on 2-4 stitches per population and never on an honest line; the tall-word cut fires on 5-6 honest lines
and 0-4 stitches. The desktop `p3` control, tall-word cut without a ratio cut: **12** honest lines, **7**
stitches, 1 undecided (n=20).

The honest side, widest first: 2.14 (`INT WORRY ABOUT`, a balloon line whose first word the outline cut short),
2.13 (`KIDS.. DID YOU`, an ellipsis), 2.09 (a handwritten notice with an edge artefact), 2.06 and 1.77 (the
letter-spaced `DIMANCHE 18 OCTOBRE 1908` masthead), 1.72 (`G MERICA CAN PREVENT`). The stitch side by ratio:
3.22 (`I CAN'T FIGURE OUT \/`, a balloon and the next balloon's tail, conf 95), 3.43 (the same balloon, colour
rendition), 3.57 (`ETRANGER` and a dotted rule, conf 52). The first two stand under 3.5 and are caught by the
tall-word cut only.

## The tall-word cut

The cut fires on a line where one word box stands over 1.6 x the line's median word height. In the labelled
sample that word is, on an honest line: a display word set larger than the rest of its line (`TARGET FOR THE 70's`),
a word whose box swallowed a spike of artwork (`ONLY A STRONG`), a word with accents or heavy ascenders (the Polish
poster text), italic or handwritten descenders. It was read off the crops, not tabulated word by word.
On a stitch it is the outline glyph between two balloons (`/`, `|`, `}`, `\`).

Tall-word ratio of the labelled lines the cut fires on: honest **1.62-3.54x (n=56)**, stitches **1.62-7.09x
(n=21)**. The two ranges are nested, so **no ratio separates them**; the Superman tokens at 2.1-2.3x that
amendment 1.7 A was built on sit inside the honest range. The amendment's statement that the cut "never fires on a
line of one size" does not hold on this corpus.

It reaches the page. The shipped `Recognize` on the default PSM 3 path, scenes from the lab corpus:

| Scene | The honest line | The plates it became |
|---|---|---|
| `atomicwar0201` | `ONLY A STRONG AMERICA CAN PREVENT` | two plates: `ONLY A` and `STRONG AMERICA CAN PREVENT` |
| `le-petit-journal-balkan-crisis-1908` | `LE REVEIL DE LA QUESTION D'ORIENT` | `LE REVEIL DE LA` merged into the row below it (`La Bulgarie proclame ..`), `QUESTION D'ORIENT` a plate of its own |

Not measured here: how many plates over the whole corpus change if the cut is removed, and which candidate rule
separates the two populations (for instance a tall token that carries no letter). Both belong to the amendment
that would change it, not to this evidence note.

## What this does not say

- Honest and stitch lines under 1.4x that no cut fires on were not labelled.
- tesseract.js ran in Node, not in the extension's viewer; the desktop and extension counts agree closely, which
  is what a shared engine would give, but the browser staging was not exercised.
- `eng` only, as in 2026-09-12. A Cyrillic or CJK corpus at PSM 11 is a different measurement.
- The labels are one reader's; the sheets are kept so another reader can re-label.
- Android round 8 (`OCR-ACCURACY` section 18, measured on its own pipeline) is context, not evidence here: its
  honest maximum of 1.71 and its 0 cross-figure stitches on `1.png` agree with the figures above.

## Consequence

- **No constant moves.** 3.5 stays; 0.14 stays; 1.6 stays as the clustering's type-size ratio.
- **For the catalog, first (rule 13):** `OCR-PIPELINE` amendment 1.7 A (the tall-word cut) is contradicted by this
  evidence on its own claim. A dated PROPOSAL addendum announces it; the amendment, if the owner takes it, is the
  catalog's step before any code change.
- **For `OCR-ACCURACY` / `OCR-EXCHANGE` (records of others, not edited):** a PROPOSAL addendum with this side's
  PSM 11 distribution, so Android's round 8 can be read against it.

## Reproducing

`temp/ocr-psm11/` (gitignored, local): `psm11_probe_test.go.txt` is the desktop harness - copy it to
`internal/ocr/zz_psm11_probe_test.go`, then `DOCHT_PSM11_PROBE=<out dir>` with `GOFLAGS=-tags=nodynamic go test
./internal/ocr -run TestPSM11Probe` (delete the file afterwards; it must not be committed); `ext-harness.mjs` is the
tesseract.js harness (`node ext-harness.mjs --out <dir> --slice i/n`, from PowerShell); `mksheets.mjs` and
`mkpick.mjs` build the labelling sheets; `aggregate.mjs` joins `labels.json` to the manifests.
