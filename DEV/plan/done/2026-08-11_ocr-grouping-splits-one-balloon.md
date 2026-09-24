# One balloon becomes three plates: the line-gap test measures ink, not line pitch

**Status:** Implemented 2026-08-11
**Priority:** 43
**Date:** 2026-08-11

> Cross-edition feature ticket. One feature = one ticket covering every edition.
> Read [`docs/PARITY.md`](../../../docs/PARITY.md) before starting; update it when a shared invariant moves.

## What / why

A speech balloon should become one plate: one box, one string, one thing for the browser to
translate. Measured 2026-08-11 with the fresh build on the lab scene `synth-balloon-on-panel`, it
becomes three:

```
WELL, THAT IS    |    ONE WAY TO    |    SOLVE IT!
```

The recognition is perfect. The grouping is wrong, and the consequence is not cosmetic: three plates
means three independent translations of three sentence fragments, three separately fitted fonts, and
three boxes where the artwork had one. A translator handed `ONE WAY TO` on its own has no sentence to
work with, so a split balloon degrades the very thing the overlay exists for.

The cause is arithmetic. Lines join into one plate while the vertical gap to the next line stays
within a factor (1.2) of the **median line height** - but the height used is the height of the
*recognized ink box*, and for all-caps lettering with no descenders that box is far shorter than the
line it came from. Measured on this scene: ink boxes 15-17 px on text drawn with 36 px leading, gaps
between them 19-21 px, and 1.2 x 15 = 18 px. The gate fails by two pixels on text a reader sees as
obviously one balloon.

So the gate compares against the wrong quantity. What it wants is the **line pitch** - the distance
between successive line tops - not how tall the ink happens to be. Pitch is available from the same
line boxes the clustering already holds.

**This is one half of a pair, and the halves pull in opposite directions.** That is why it must not
be "fixed" by raising the factor:

- **Splitting** (this ticket): one balloon becomes several plates.
- **Merging**: two adjacent balloons, or two text columns, become one plate that crosses an unrelated
  reading group. The lab scores this as `merges` and `crossGroup`, and it is Step 07.3 of
  [`ocr-visual-fidelity-lab`](../2026-08-11_ocr-visual-fidelity-lab/PHASE_07__concealment-and-grouping.md),
  whose written instruction is explicitly *"keep `ocrMinLineConf` and `ocrClusterGapFactor`
  unchanged; this is an added condition, not a retuned one"*.

Raising the factor fixes this ticket and breaks that step. Changing what the factor is *measured
against* may fix both, which is the hypothesis worth testing - and the lab already scores both
directions on the same run, so the trade is measurable instead of arguable. Sequencing note: this
ticket and Step 07.3 move the same gate and should be decided together, in whichever order, but not
in two unmeasured steps.

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[x]` | `clusterLines` in the shared recognition path; measured on the lab's dev split |
| GUI (`doc-html-ui`) | `[x]` | inherits the pipeline output - confirmed by reading, not assumed: nothing in `cmd/doc-html-ui` touches clustering |
| MSIX Store app | `[x]` | inherits the GUI |
| Browser extension | `[x]` | ported into the new `extension/src/ocr-cluster.js`; same constants, same expressions, guarded |
| Website / docs | Declined (expected) | plate grouping is not claimed publicly; the change is quality, not a new user-visible capability |

## Shared invariants touched

- **`ocrClusterGapFactor` == `OCR_CLUSTER_GAP_FACTOR` = 1.2**, a `docs/PARITY.md` invariant with a
  drift guard. If the quantity it multiplies changes, the invariant's *meaning* changes on both
  sides: the parity row has to be rewritten, not just the number, and the guard has to pin the new
  meaning. *Landed as `ocrClusterPitchFactor` == `OCR_CLUSTER_PITCH_FACTOR` = 1.2, joined by
  `ocrMaxLeadingRatio` == `OCR_MAX_LEADING_RATIO` = 3; the parity row was rewritten and the guard
  now pins both expressions, not only the numbers.*
- **`ocrMinLineConf` / `OCR_MIN_LINE_CONF` = 50** and the rescue floor of 80 feed the same clustering
  path - a line dropped by confidence is a line that cannot contribute to pitch.

## Cross-references

- Go: [`internal/ocr/tesseract.go`](../../../internal/ocr/tesseract.go) - `clusterLines`, where
  `heights` is built from `y1-y0` and `gapMax` from its median. *Now also `medianLinePitch`; the
  fixtures are in [`internal/ocr/cluster_test.go`](../../../internal/ocr/cluster_test.go).*
- JS: `extension/src/ocr-overlay.js` - `clusterLines`, the hand-ported twin. *Moved to
  `extension/src/ocr-cluster.js` so it can be unit-tested without the vendored Tesseract bundle;
  tests in `extension/test/ocr-cluster.test.mjs`.*
- The scene and its arithmetic: `tools/ocrlab/synth/synth.go` - `sceneBalloonOnPanel` (36 px leading)
  and the comment on `sceneAdjacentBalloons`, which works the same sum in the opposite direction.
- Evidence: `temp/testkit/` group A2 (three plates, correct text) and `temp/ocrlab/loop3/report.html`.

## Done criteria

- [x] `synth-balloon-on-panel` yields **one** plate carrying the whole balloon.
- [x] `synth-adjacent-balloons` still yields **two** - the opposite failure is not traded for this one.
- [x] `merges`, `splits` and `crossGroup` do not regress on the lab's dev split, measured on both
      editions rather than on the Go side alone. *The lab has no extension runner yet (that is the
      lab's own Phase 05), so the extension is measured on the recognizer's real line boxes through
      a unit test instead - see "What landed".*
- [x] The parity row is rewritten if the measured quantity changes, and the drift guard still fails
      on a one-sided change (prove it by making one).
- [x] `./scripts/test.ps1` green, `npm test` green; changelog entry in `DEV/CHANGELOG.md`.

## What landed

**The gate compares pitches, not ink.** `clusterLines` still grows a plate while the next line is
vertically adjacent, but adjacency is now `pitch <= ocrClusterPitchFactor x referencePitch`, where
the pitch is the distance from one line's top to the next line's top. The factor keeps its value
(1.2) and loses its old name: `ocrClusterGapFactor` -> `ocrClusterPitchFactor`, because a constant
whose name says "gap" and whose meaning is "pitch" is the next reader's trap.

**The reference is the image's median pitch,** taken over successive kept lines that share a column
and sit no further apart than a new `ocrMaxLeadingRatio` (3) median ink heights. Both filters exist
to keep a section break out of the estimate; without the second one, a page holding a heading and
one distant paragraph makes that single jump its "typical" pitch and merges the two - which the
existing `TestParseTSVSplitsOnBigGap` catches, and did.

**Measured, on the recognizer's own output rather than on the drawn scene.** The two scenes'
tesseract line boxes were dumped and are now the fixtures of `internal/ocr/cluster_test.go` and
`extension/test/ocr-cluster.test.mjs`, so both editions are handed identical input:

| scene | ink heights | pitch | old bound | new bound | plates before | after |
|---|---|---|---|---|---|---|
| `synth-balloon-on-panel` | 29-34 px | 72 px | 34.8 px gap | 86.4 px pitch | 3 | **1** |
| `synth-adjacent-balloons` | 28 px | 52 inside / 84 across | 33.6 px gap | 62.4 px pitch | 2 | **2** |

The balloon fixture was proved to be a real guard by disabling the pitch branch: 3 plates, test red.

**Lab, dev split, desktop edition** (`temp/ocrlab/pitch1` against `temp/ocrlab/loop3`, 8 scored
scenes): recall and precision 0.625 -> 0.750, mean IoU 0.695 -> 0.820, worst IoU 0.139 -> 0.349,
CER 0.163 -> 0.149, concealment 0.736 -> 0.777. The three grouping counters are **unchanged** -
`merges` 1, `splits` 0, `crossGroup` 6 - as are `clipped` 0, `protectedHitPx` 0, `worstDrift` 0 and
`failingScenes` 2. Two scenes changed plate count and no others did: the balloon 3 -> 1, and
`synth-two-columns` 2 -> 1. That second one is not a new merge: tesseract already recognizes that
scene as three lines each spanning both columns, so the columns were crossed before and after; the
plate that crosses them is now one instead of two, which is why `merges` and `crossGroup` hold.
`minContrast` moved 92 -> 86 (still far above the 55 floor) because the balloon's colours are now
sampled over one block instead of three.

**The extension is a port, not a copy-paste.** The clustering moved into a new pure module,
`extension/src/ocr-cluster.js`, for the same reason `ocr-text.js` exists: `ocr-overlay.js` imports
the vendored Tesseract bundle at load time, which is gitignored, so nothing inside it can carry a
unit test. `TestParityOCRClustering` now reads the constants from that file, gains the new one, and
- this is the part the ticket asked for - pins the *meaning* as well as the values: it fails if
either side stops computing its bound from a reference pitch or stops measuring a pitch top-to-top.
Proved by drifting the JS side alone (factor 1.3 and a pitch bound recomputed from the ink height):
both checks fired.

**Not done here, on purpose.** Step 07.3 of the fidelity lab still stands: this ticket did not add
the boundary condition (an edge or a background change between two lines), and `synth-two-columns`
still fails on a merge that starts in recognition, before clustering ever runs.

## Open questions

- ~~What is the pitch of a single-line block?~~ **Answered: the question was the wrong unit.** The
  reference is per image, so a single-line *block* is never the problem - a page that yields no
  pitch at all is, and it falls back to the ink-box gap this test used before. One line has nothing
  to join in any case, so the fallback is unreachable through the join path; it is there so the
  function has a defined answer, not because a scene needs it.
- ~~Is the unit per block or per image?~~ **Per image, and deliberately.** A cluster's own pitch is
  unavailable exactly when the decision is hardest - joining its second line, which is the join that
  split the balloon - so a per-block unit would leave the failing case unfixed. The median plus the
  `ocrMaxLeadingRatio` filter is what makes a page with two type sizes safe: the steps between the
  sizes are outliers around the body pitch rather than the middle of it, which `display-lettering`
  shows (70 px accepted between the two display lines, the 173 px step down to the caption rejected,
  two plates before and after).
- Does pitch survive rotated or curved lettering, which comic covers are full of? **Still open, and
  now bounded rather than unknown.** Every scene in the lab's dev split is axis-aligned, so nothing
  measured here says anything about it. What can be said from the arithmetic: a rotated line's box
  grows in both axes, so its ink height rises and `ocrMaxLeadingRatio` widens with it, while curved
  lettering recognized as several short lines at stepping tops would read as a pitch that varies -
  the median absorbs a minority of those and fails on a majority. That is a corpus question and it
  belongs to the lab's annotation backlog, not to a second guess here.
