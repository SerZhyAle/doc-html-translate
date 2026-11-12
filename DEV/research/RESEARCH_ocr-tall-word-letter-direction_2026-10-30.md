# The tall-word cut: a token without a letter separates the labelled sample, but not the page the cut was built for

2026-10-30. Ticket [E0128](../plan/done/E0128_ocr-tall-word-cut-splits-honest-lines.md), measuring the first direction the owner
named in the quiz of 2026-10-30 on E0121. Follows
[`RESEARCH_ocr-psm11-word-gap_2026-10-21.md`](RESEARCH_ocr-psm11-word-gap_2026-10-21.md) (the finding) and
[`RESEARCH_ocr-comic-coverage_2026-09-29.md`](RESEARCH_ocr-comic-coverage_2026-09-29.md) (the page the cut was built on).
No constant moved and no code changed; the catalog is untouched.

## Question

`OCR-PIPELINE` amendment 1.7 A cuts a recognizer line before any word whose box stands over `ocrTypeSizeRatio` (1.6) times the
line's median word height. Measured on 2026-10-21 it fires on honest lines more often than on stitches and no ratio separates
them. The owner's first direction: cut only when the tall token carries no letter (the stitch tokens in the labelled sample are
outline glyphs: `/`, `|`, `}`, `\`). The fallback: a second piece of evidence, a stroke or a wide gap. Does either separate the
labelled sample, and does either keep what 1.7 A was built for?

## Answer

- **The first direction separates the labelled sample.** On the desktop engine, honest lines cut by the tall rule fall from 33 to
  2 and stitches keep 11 of 13 cuts; on the extension engine honest lines fall from 23 to 1 and all 8 stitches keep their cut.
- **It does not keep the page 1.7 A was built for.** Both boundary tokens of the Superman page carry a letter (`“J`, `A`), so
  the rule stops cutting both rows. No second piece of evidence of the kinds the owner named recovers them (section "The
  fallback"), and nothing in this record separates row two's `A` from honest narrow-tall `A` tokens.
- So the choice is between two measured losses, not a derived rule. It is the owner's (ticket E0128, question 1).

## Instrument

- **Desktop edition:** tesseract 5.4.0.20240606 with the app's own staging (`prepareForOCR`, declared DPI, the three
  renditions of 2026-10-21: `p3` the colour frame at PSM 3 - the shipped default path - `p11c` the same frame at PSM 11, `p11g`
  the grey rung at PSM 11). A throwaway test inside `internal/ocr` (deleted after the run, kept as text under
  `temp/E0128/`) records every word at index 1 or later that stands over 1.6 x its line's median word height, with: whether the
  text holds a Unicode letter or digit, the box, the engine's word confidence, the gap to both neighbours in median heights,
  the shipped `strokeBetween` between the previous word and the token, across the token (previous word to next word), and
  inside the token's own box (a path that runs `reach` beyond the box's top and bottom). 36 s a run over the 59 corpus scenes
  plus `test_doc/1.png` (and the Superman picture below).
- **Extension edition:** tesseract.js 7.0.0 in Node, the 2026-10-21 harness (the extension's own upscale, luma weights and
  constants imported from `ocr-cluster.js`), with the same per-token record minus the two span strokes.
- **Labels:** the 168 hand labels of 2026-10-21 (R honest line, S stitch, X no text, Q undecidable), joined by rendition, scene
  and line index. The re-run reproduces the earlier run: 0 text mismatches on the 48 desktop and 39 extension labelled lines that
  carry a tall word. Floor 50, as in that note.

## Results on the labelled sample

Lines of the labelled sample on which the tall rule acts (a tall word at index 1 or later): desktop 33 honest and 13 stitch
(10 of the stitches have no other cut), extension 23 honest and 8 stitch. Cells are lines still cut / lines.

| Rule: cut before a tall token when .. | Desktop honest | Desktop stitch | Extension honest | Extension stitch |
|---|---:|---:|---:|---:|
| (today) always | 33 / 33 | 13 / 13 | 23 / 23 | 8 / 8 |
| **A** it carries no letter | **2 / 33** | **11 / 13** | **1 / 23** | **8 / 8** |
| A, and no digit either | 2 / 33 | 11 / 13 | 1 / 23 | 8 / 8 |
| B no letter, or a stroke across the token's span | 4 / 33 | 11 / 13 | - | - |
| C no letter, or a stroke inside the token's own box | 3 / 33 | 12 / 13 | - | - |
| D no letter, or a stroke between the previous word and the token | 2 / 33 | 11 / 13 | - | - |
| E no letter, or any of those strokes | 4 / 33 | 12 / 13 | - | - |

- On the shipped default path alone (`p3`, the colour frame at PSM 3): the rule acts on 11 honest, 8 stitch and 1 undecided
  labelled lines; under A honest lines still cut 1 of 11 and stitches 8 of 8.
- The tokens that keep the cut under A are the outline glyphs the owner named (`/`, `\`, `|`, `}`, `\/`) in 8 of the 10
  desktop stitches that have no other cut, and in every extension stitch.
- The two stitches A loses on the desktop engine carry a letter: `Gale?` (7.09x, the Reggie cover read in colour; the grey rendition
  reads the same boundary as `\/` and keeps the cut) and `OT` (1.73x, handwritten notices).
- The two honest lines A still cuts are the `?` of a Timmy the Timid Ghost cover (23x44, standing 0.59 median heights inside its
  left neighbour) and a `-` of the Samson caption (18x54, -0.58). The extension engine has the `?` once.
- Honest tall-token records: 35 carry a letter, 2 do not. The honest tall tokens are display words set larger than their line,
  words whose box swallowed a spike of artwork, heavy ascenders and italic descenders.
- Stitch tokens that carry a letter are rare in this sample: 5 records, and 3 of those lines stay cut under A anyway (`aie` by
  another cut; `ARE` and `THE` by the `\` before them). Only `Gale?` and `OT` are lost.

## The fallback

Neither stroke test earns its place. B, C and E add 1-2 honest lines to the cut set (a stroke runs through a few honest display
words) for at most one stitch back; D equals A. A wide gap is no better: the honest letter-bearing tokens stand from -7.00 (overlapping the
left neighbour) to 1.69 median heights from it, the lost stitches 0.67 (`OT`) and 3.43 (`Gale?`, already caught by the grey
rendition's `\/`), the Superman tokens 1.27 and -0.81.

## The page 1.7 A was built for

The Superman page (`test_doc/New_test/027f032012d82d5e01cc1cc083914fd9.jpg`, 505x720), read on the real picture by the desktop
engine at PSM 3, gives the two stitched rows and their boundary tokens exactly as the shared fixture
(`tests/testdata/ocr_balloon_stitch_cases.json`) records them:

| Row | Token | Box | Over median | Engine confidence | Gap to the left neighbour | Stroke evidence on the real picture |
|---|---|---|---:|---:|---:|---|
| one | `“J` | 27x35 | 2.33x | 2 | 1.27 | across the token's span only |
| two | `A` | 11x32 | 2.00x | 90 | -0.81 (inside `(T'S`) | none of the three |

- Both tokens carry a letter. Under A neither row is cut, so by the cut points alone the page's middle panel returns to the one
  interleaved plate that 1.7 A removed. This was read off the cut points; the plate-level run is a release-tier item.
- Row two's `A` has the same shape as honest tokens: the `A` of the Samson caption (14x53, -0.69) and of the scroll (35x103,
  -0.61), `of` in a poster line (12x28). No geometric or stroke feature in this record tells them apart.
- On the real picture no stroke test finds anything between the words of either row (`lineStroke` and `lineWide` are false for
  both). The replay test `TestParseTSVSplitsBalloonsWhoseOutlinesTouch` draws its own stroke at x 250-253; the real picture
  does not give the strips that stroke.
- The extension engine reads this page differently: at PSM 3 its only tall word (`puRC`) stands on a line another cut already
  serves and the boundary tokens `“J` and `A` do not appear, so it is no measure of 1.7 A's page; the fixture is the desktop
  engine's.

## Unlabelled populations

For scale only, not labelled: on the 12 own-UI screenshot scenes of E0121 at PSM 3 the rule acts 66 times in 37 lines at the
floor; 22 tokens carry no letter (interface separators: `|`, `/`, `-`) and 44 carry a letter, mostly on lines that read as
ordinary interface text. On the 47 earlier scenes plus `test_doc/1.png` at PSM 3 it acts 40 times in 28 lines, 19 without a letter and 21 with one.

## What this does not say

- The plate-level effect (how many plates over the corpus change) is not measured; the lab gate and the Superman replay test are
  release-tier runs.
- The labels are one reader's, lines under the floor of 50 were not labelled, and the stitch side is small (13 desktop lines,
  8 extension lines; 5 letter-bearing stitch tokens).
- Only the desktop engine reads the Superman page's boundary tokens as the fixture records them.
- The catalog text of 1.7 A names the type-size ratio as 2.5; the code, both guards and the fixture comment use 1.6. A
  future amendment must state which.
- Nothing here is a constant. A rule on letters has no threshold to bracket.

## Consequence

- **Nothing is applied.** The code and the catalog stay as they are until the owner answers ticket E0128 question 1.
- **Option 1, adopt A:** honest lines stop being cut (33 -> 2 desktop, 23 -> 1 extension); the cost is the Superman page, whose
  two letter-bearing boundary tokens stop being cut. The amendment then names that page as a known limit and rewrites the
  shared fixture's two stitch cases and the replay test accordingly.
- **Option 2, keep 1.7 A as built:** the Superman page stays separated; the cut keeps splitting honest lines (13 of the 46
  desktop labelled lines it acts on are stitches), and the finding stays recorded.
- Neither the first nor the second direction yields a rule that serves both populations; a third direction would need a feature
  this record does not contain.

## Reproducing

`temp/E0128/` (gitignored, local): `tall_probe_test.go.txt` is the desktop probe (copy it and `psm11_probe_test.go.txt` to
`internal/ocr/zz_*_test.go`, then `DOCHT_TALL_PROBE=<dir>` and optionally `DOCHT_TALL_EXTRA=<picture>` with
`GOFLAGS=-tags=nodynamic go test ./internal/ocr -run TestTallProbe`; delete both files afterwards); `ext-tall.mjs` is the
tesseract.js probe (`node ext-tall.mjs --out <dir> [--slice i/n]`, from PowerShell; `EXTRA=<picture> --only extra-superman` for
one picture); `analyze.mjs desktop|ext` joins the labels of `temp/ocr-psm11/` and scores the rules; `letters.mjs`, `extra.mjs`,
`dp.mjs` and `cmp.mjs` print the letter-bearing tokens, the unlabelled populations, the default-path figures and the
old-versus-new line counts.
