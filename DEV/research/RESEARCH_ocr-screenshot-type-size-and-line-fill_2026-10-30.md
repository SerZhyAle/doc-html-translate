# OCR on the product's own window: the type-size ratio (OC-1) and the 0.72 line-fill bracket (OC-3)

2026-10-30. Phases C and D of [`DEV/plan/done/E0121_ocr-owed-evidence-and-rule-10-reverification.md`](../plan/done/E0121_ocr-owed-evidence-and-rule-10-reverification.md),
on the owner's decision of 2026-10-30: a lab category `screenshot` (not a tag) with a floor of 10, scenes from the
product's own interface, truth geometry from the rendered DOM, scenes in `dev` until a human signs them. Evidence owed to
`OCR-ACCURACY` / `OCR-EXCHANGE` since the catalog decision of 2026-10-17. No constant or threshold was changed.

## Question

- **OC-1.** `ocrTypeSizeRatio` 1.6 was bracketed at 1.42 (the widest spread inside one text) and 1.86 (the narrowest
  step between two texts) on posters and comics. Does a UI screenshot put anything inside that bracket?
- **OC-3.** `ocrMinPlateLineFill` 0.72 was bracketed with n=1 on each side (`ocr_plate_coverage_2026-08-13.md`): keep
  0.7921 (the hand-drawn `samson-and-delilah-03-scroll` caption), release 0.6608 (`accounts.jpg`, six list rows in one
  plate). Does the corpus with screenshot scenes recompute either side?

## Answer

- **OC-1: nothing on the window comes near the bracket.** On the 12 screenshot scenes the type-size step between two
  stacked reading groups is under 1.42 in all 212 pairs whose font sizes differ (the window uses 11 to 17 px), and the
  spread inside a group is 1.00. The ratio neither cuts nor is asked to cut anything on this interface; a UI region is
  separated from its neighbour, when it is, by the pitch and gap rules. The bracket is neither confirmed nor moved by
  this evidence: no data point falls inside 1.42-1.86.
- **OC-3: the bracket cannot be recomputed from live output - its release side is empty.** Over the 60 images (the 59
  lab scenes and `accounts.jpg`), the ordinary pass forms no cluster above the coverage bound 0.52, so the line-fill
  test is never reached: the largest cluster is the scroll caption at 0.2795 (fill 1.04). `accounts.jpg` no longer
  merges its rows: its largest cluster is 3 lines at coverage 0.1214 (fill 0.6567), against the 6-line plate at
  0.6829 / 0.6608 the bound was derived from. The screenshot scenes stay far under the bound: largest recognized
  cluster 0.0340, largest DOM reading group 0.0360. The keep side is still the one hand-drawn caption (0.7921); the
  release side survives only as the unit fixture `TestReleaseOversizedNeedsBothConditions`.
- **So: no recomputed bound, nothing to propose.** 0.72 and 0.52 keep their derivation of 2026-08-13 (`OCR-OVERLAY`
  rule 13 "derived"); the rule fires on no ordinary-pass cluster of the current corpus.
- **Observation, not measured further:** the recognizer joins neighbouring UI groups. Of the 42 multi-line clusters on
  the screenshot scenes, 27 take lines from two or more DOM groups (a checkbox label with its description, a heading with
  the line under it) and 11 stay inside one group; 4 could not be mapped. All are small (coverage at most 0.034), and
  their line fill (0.40-1.18) overlaps the one-group clusters' (0.55-1.58).

## Phase C: the class and its scenes

- `tools/ocrlab/corpus`: category `screenshot` (`CatScreenshot`), floor 10 (`CategoryMinimum`); every consumer reads the
  category list, so `verify`, the coverage table, the report and the gate take it without further change.
- `tools/ocrlab/screens/capture.mjs` runs the real `doc-html-ui.exe` under a scratch profile with the help generator's
  machinery (`tools/helpgen/app.mjs`, the window states of `tools/helpgen/shots.mjs`), captures the window and reads the
  box of every word the page paints: a word counts only when it is visible, inside every clipping ancestor and the
  topmost element at its centre; a word the layout broke over two lines is cut into its per-line pieces; form fields
  contribute the text they show, measured with their own font. Words are grouped into lines per block box.
- `go run ./tools/ocrlab screens` turns the capture's `index.json` into manifest entries and annotations: licence
  `OWN` (the owner's own interface, recorded by the decision of 2026-10-30, as `ocrlab add` records an owner's image),
  split `dev`, categories `screenshot` plus `script` for the Cyrillic windows, truth `origin: human` with
  `annotatedBy: tools/ocrlab/screens (rendered DOM boxes)` and no `checkedBy` - a human signs by filling it. A signed
  record of unchanged bytes is kept on a re-run.
- 12 scenes, 1160 px wide: the converter ready and with a finished queue, the OCR settings page and the About page,
  each in `en`, `ru` and `uk`, light and dark alternating; 24-79 groups and 25-103 lines a scene. Built from
  `temp/helpgen/bin/doc-html-ui.exe` (v26.1028.0224). The overlay of the boxes on two scenes was checked by eye.
- `go run ./tools/ocrlab verify`: `screenshot 12 / need 10`, and no problem names a screenshot scene (the exit stays 1
  on the 37 pre-existing rights defects and 33 annotation problems of the other scenes, ticket 109).

## How D was measured

A throwaway test in `internal/ocr` (deleted after the run; copy kept as `temp/E0121/fill/fill_probe_test.go.txt`) ran
the ordinary pass exactly as `Recognize` does (`prepareForOCR`, the same Tesseract arguments, `tsvLines`) and clustered
the lines with `clusterLines` given no image size, which skips `releaseOversized` and so exposes every cluster before
the coverage release. Coverage is the cluster box over the page; line fill is the sum of its line heights over the box
height - the two quantities `releaseOversized` reads. Tesseract 5.4.0, `eng`, the shipped staging. Rescue rungs and
sweeps were not run.

Clusters on the screenshot scenes were labelled against the DOM truth: a cluster line belongs to the truth group whose
line box holds its centre.

| Set | Multi-line clusters | Coverage, max | Line fill |
|---|---:|---:|---|
| all 60 images | 91 | 0.2795 (scroll caption) | - |
| over the 0.52 bound | 0 | - | - |
| screenshots, inside one DOM group | 11 | 0.0253 | 0.55-1.58 |
| screenshots, two or more DOM groups | 27 | 0.0340 | 0.40-1.18 |
| screenshot DOM groups of two or more lines (truth) | 66 | 0.0360 | 0.93-1.03 |
| `accounts.jpg`, largest cluster | 1 | 0.1214 | 0.6567 |

OC-1 used the DOM truth only: for each group, the group directly below it (columns overlapping by a tenth of the
narrower, within 1.2 line heights), and the ratio of their median line heights.

## What this does not say

- One interface (the GUI launcher) at one width, 3 locales, 2 themes; no right-to-left or CJK window, no phone width.
- The truth is unsigned: it is the page's own geometry, independent of OCR, but no human has checked a scene yet, so
  the scenes stay `dev` and these figures are exploratory.
- Text a page paints without a DOM text node (an icon glyph, CSS generated content) is not in the truth.
- 15 of the 60 images form no cluster on the ordinary pass; in the shipped path they go to the rescue ladder, whose
  clusters (which `releaseOversized` also judges) were not measured here.
- The ordinary pass only; the `eng` model reads the Cyrillic windows as debris, which matters to rule 10, not to the
  geometry measured here.

## Reproducing

`node tools/ocrlab/screens/capture.mjs --exe temp/helpgen/bin/doc-html-ui.exe`, then `go run ./tools/ocrlab screens`
(PowerShell, `GOFLAGS=-tags=nodynamic`). Raw data in `temp/E0121/fill/` (gitignored): `pre-release.jsonl` (the probe's
clusters), `analyze.mjs`, `typesize.mjs`; box overlays in `temp/E0121/overlay/`.
