# Phase 06 - Baseline and thresholds

**Strategic spec:** [`../2026-08-11_ocr-visual-fidelity-lab.md`](../2026-08-11_ocr-visual-fidelity-lab.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done (5 of 5)
**Depends on:** Phase 04, Phase 05
**Steps done:** 5 / 5

## Objective

The first measured state of both editions, and - derived from it, never before it - the acceptance
bounds and the answers to strategic §9.

## Prerequisites

- [ ] Phase 04 and Phase 05 are ✅ Done.
- [ ] `ocrlab verify` passes, or its remaining problems are listed in the baseline report as the
      report's own coverage caveat.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `DEV/research/ocrlab/2026-08-11__baseline.md` | New | - |
| `DEV/ocrlab/thresholds.json` | New | - |
| `tools/ocrlab/report/gate.go` | New | ≤ 240 |
| `tools/ocrlab/main.go` | Modified | ≤ 460 |
| `DEV/research/RESEARCH_INDEX.md` | Modified | ≤ 120 |

## Steps

### Step 06.1 - Run and record the baseline

**Files:** `DEV/research/ocrlab/2026-08-11__baseline.md`
**Depends on:** - start of phase

**Prompt for developer:**
> Run `ocrlab run` + `score` + `report` for both editions over the whole available corpus, then write
> the dated report: corpus revision and scene count actually scored, tesseract and tessdata versions,
> browser version, viewport set, per-category and per-split numbers for all eight §3.2 dimensions,
> per-edition, and a screenshot reference for every failing scene. State the coverage honestly - which
> §4.1 category minimums are unmet and how many scenes were skipped for a non-truth annotation. A
> report that hides thin coverage is worse than no report.

**Verification:**
- The file exists and its first table names every one of the eight §3.2 dimensions.
- It states the scored scene count and the skipped count with reasons.
- It names the tesseract, tessdata and browser versions used.

**Status:** `[x]` done - [`DEV/research/ocrlab/2026-08-11__baseline.md`](../../research/ocrlab/2026-08-11__baseline.md).
§1 is the eight-dimension table, both editions side by side. §2 states 45 scenes run desktop / 28
extension, 11 and 10 scored, and every skip reason including the 17 extension scenes that crashed
the renderer. §3 names tesseract v5.4.0.20240606, tesseract.js 7.0.0, `eng.traineddata`
`sha256:7d4322bd2a774972` (**identical in both editions**), Edge 151.0.4129.78 and
Chrome 151.0.7922.109 - and records that the desktop evidence leaves `engine.tessdataVersion`
empty under `go run`, so that hash was taken by hand.

**The run could not be produced until the extension's transport was repaired** (§7): `CDP.send`
had no timeout, so a wedged renderer hung the whole run for ever inside a `waitFor` that could
never fire, while the Go side has always bounded each call at 90 s. Three fixes in
`extension/scripts/_ocrlab-cdp.mjs` and `extension/scripts/ocrlab.mjs`, a fourth for a
launch-time port race, and `extension/test/ocrlab-cdp.test.mjs` to hold them. No shipped code
changed, so no threshold, constant or rendering decision moved.

---

### Step 06.2 - Resolve strategic §9.1-§9.4 from the data

**Files:** `DEV/research/ocrlab/2026-08-11__baseline.md`
**Depends on:** Step 06.1

**Prompt for developer:**
> Add a "Research items resolved" section answering, each with the measurement that decided it: (1)
> whether OCR word/line boxes seed a sufficiently tight mask or a local binarization pass is needed -
> decide from the textured-background scenes; (2) which background reconstruction passes the damage
> measure most cheaply - compare local sampling against directional interpolation on the same protected
> annotations; (3) whether font classification changes the visual outcome versus a robust fallback; (4)
> the annotation cost per scene measured on the pilot, and whether a region-level proxy is sufficient
> in place of full glyph masks. An item the data cannot settle stays open and says so.

**Verification:**
- Each of the four items has a stated answer or an explicit "still open - needs N more scenes of class X".
- Each answer cites a table or scene ID from this report, not an opinion.

**Status:** `[x]` done - §10 of the baseline report. Three of the four stay open and say what would
close them: §9.1 needs 8-10 more annotated `texture` scenes (12 exist, 1 annotated, and the class
carries the highest cut-glyph ink at 14%); §9.2 is blocked on protected polygons rather than on the
method, because `protectedHitPx` reads 0 only for want of any protected annotation to hit; §9.3 is
answered "do not add the dependency" - none of §4.1's five findings is a typeface problem. §9.4 is
answered from a count: 11 annotations, 8 of them synthetic, so **3 real scenes are annotated against
37 harvested**, which is the measured argument for the region-level proxy the report leans on.

---

### Step 06.3 - Write the thresholds

**Files:** `DEV/ocrlab/thresholds.json`
**Depends on:** Step 06.2

**Prompt for developer:**
> Derive `thresholds.json` from the baseline: for each dimension, a per-category bound and a
> non-regression tolerance, plus the hard gates the strategic spec fixes at zero regardless of the
> baseline - a newly visible original word, damaged protected content, a clipped translation, a plate
> crossing an unrelated reading group. Record for every bound the baseline value it came from, so a
> later retrofit is visible in the diff.

**Verification:**
- `DEV/ocrlab/thresholds.json` parses and covers all eight dimensions.
- Each non-hard bound carries a `baseline` field.
- The four hard gates are present with value 0.

**Status:** `[x]` done - the file loads (`LoadThresholds` runs `Validate`, and `ocrlab gate` reaches
its verdict rather than stopping on the file), all eight dimensions are present, every bound carries
a `baseline`, and the four hard gates are 0.

**What it deliberately does not contain, and why** - all three recorded in the file's own `caveat`:

- **No `regressionTolerance` on any dimension.** It compares holdout against holdout and the corpus
  has no holdout, so the field could never fire. Add it the day the split exists.
- **No per-category bound where the category has one or two scored scenes** - `poster`, `cartoon`,
  `script`, `document`. Only `comic` (4) and `texture` (5) carry one.
- **Overall bounds sit AT the measured value, not above it.** The one exception is `cost`, set at
  about 1.5x, because it measures this machine rather than the product and a gate that fails on a
  busy CPU teaches a reader to ignore the gate.

Two bounds are worth reading as statements rather than limits: `concealment` 0.28 is not a quality
claim - 27% of the original lettering still showing is a failure, and the bound exists to stop it
worsening while Phase 07 works. `damage` 0 duplicates its hard gate on purpose, because the zero
means no scene declares a protected polygon, not that nothing is damaged.

---

### Step 06.4 - Turn the thresholds into a gate

**Files:** `tools/ocrlab/report/gate.go`, `tools/ocrlab/main.go`
**Depends on:** Step 06.3

**Prompt for developer:**
> Add `func Gate(sum *metrics.Summary, th *Thresholds, prev *metrics.Summary) GateResult` and the
> `ocrlab gate <dir>` subcommand: fail on any hard gate, on a category exceeding its bound, or on a
> holdout category regressing beyond tolerance against the previous accepted summary. Print a
> PASS/FAIL line per category and exit non-zero on FAIL. Thresholds live only in this file's input -
> the metrics package stays threshold-free.

**Verification:**
- `func Gate(...) GateResult` matches exactly once.
- `go run ./tools/ocrlab gate <dir>` prints a final `PASS` or `FAIL` line and sets the exit code.
- `tools/ocrlab/metrics/` still contains no threshold comparison.

**Status:** `[x]` done - **landed ahead of 06.1-06.3, deliberately.** `Gate` matches once,
`ocrlab gate` is wired and documented in the usage text, `metrics/` still compares nothing against
a bound, and `go build ./tools/...`, `go vet` and `gofmt` are clean. The command cannot pass or
fail anything yet: `DEV/ocrlab/thresholds.json` is 06.3's output and does not exist, so `gate`
stops with that file named. Written first because it is the one part of this phase that does not
depend on a single measured number - `LoadThresholds` refuses a file whose derived bound carries
no `baseline` field, which is the rule the *later* steps have to obey.

---

### Step 06.5 - Index the research artifact

**Files:** `DEV/research/RESEARCH_INDEX.md`
**Depends on:** Step 06.4

**Prompt for developer:**
> Add the baseline report to the research index with its date, the question it answers and the phase it
> unblocks.

**Verification:**
- `DEV/research/RESEARCH_INDEX.md` links `ocrlab/2026-08-11__baseline.md`.

**Status:** `[x]` done - a "Research artifacts in this repo" section now carries the baseline with
its date, the question it answers and the two phases it unblocks. The two existing OCR research
artifacts were listed alongside it rather than leaving a one-row index, and the grey-rescue row
repeats the "re-derive, do not inherit" warning that this phase's own notes carry.

## Phase done criteria

- [x] Every `Step 06.*` is `[x] done`.
- [x] `go run ./tools/ocrlab gate <baseline dir>` reproduces the report's own verdict - FAIL on
      `temp/ocrlab/base06-desktop` for exactly the two hard gates §6.1 of the report names (1 merge,
      6 cross-group), and FAIL on `temp/ocrlab/base06-ext` for the three it names there.
- [x] Grep for `TODO(phase-06)` returns zero hits (only this checklist line and other tickets'
      identical lines match).
- [x] Changelog entry added for every file in "Files touched", plus the transport repair 06.1
      depended on.

## Where this phase ended (2026-08-12)

Done, with the measurement made and the bounds it can carry written - and with the thing the phase
was *not* able to do stated in the file itself rather than in a commit message.

- **The corpus, not the tooling, is the limit.** `ocrlab verify` on 2026-08-12: 45 scenes, **11
  gradable annotations** (8 of them synthetic, so **3 real scenes are annotated against 37
  harvested**), **0 holdout scenes**, every one of the seven §4.1 category minimums unmet
  (`degraded` has no scene at all). §4.1's coverage targets are explicitly not a gate, but §4.3's
  "once a scene enters the holdout it does not move back" only means something if a holdout exists,
  and until one does `thresholds.json` can carry no `regressionTolerance`. **Annotating and
  splitting is still the next real work.**
- **The measurement went ahead anyway, and was worth it**, because the annotation-free
  self-diagnosis covers all 45 scenes and produced every item on the report's own "what to do next"
  list. Writing the baseline with the thin coverage as its headline was the right call over waiting.
- **The runs are `temp/ocrlab/base06-desktop/` and `temp/ocrlab/base06-ext/`** - evidence, scores,
  screenshots, `report.html`, `gate.json`. `temp/ocrlab/base-desktop/` is the older, superseded run.

### The two results Phase 07 should start from

| Finding | Desktop, of 45 | Extension, of 45 |
|---|---:|---:|
| **the browser tab crashes - the scene never renders** | 0 | **17** |
| **no plates at all - the reader gets the original text** | **11** | 7 (of 28 that ran) |
| the patch stops short of the glyphs | 14 | 5 |
| original lettering still legible under the plate | 6 | 2 |
| plates overlap each other | 4 | 1 |
| replacement text barely readable | 3 | 0 |

The extension's crash boundary is image size and it is sharp: **15 of 15 scenes above 2.8 Mpx died,
28 of 30 below it survived**, while the desktop app completed all 45 including a 17 Mpx newspaper
page. That is a user-visible product failure - an ordinary comic or newspaper scan kills the tab -
and nothing else about the extension edition can be measured over the real corpus until it is fixed.

Desktop mean residual ink under the plates is 17%, worst 56%
(`cover-of-archie-s-pals-n-gals-no-25-jpg`); ink continuing past the plate edges averages 9% of the
ink the plates cover. Against ground truth 11 scenes scored and 2 carry a hard failure -
`le-petit-journal-1908-map-labels` (no plates over 4 annotated groups) and `synth-two-columns`
(merges two columns, then crosses the neighbouring group at every stress case). **`synth-two-columns`
is the one defect both editions reproduce identically**, which makes it Phase 07's cleanest target.

### Two of these numbers were the instrument's fault (2026-08-12)

The table above is the **corrected** one. The first reading of this run named two defects that do
not exist, and both were found by trying to act on them rather than by re-reading the code, which
is the argument for the phase order: *the instrument has to be trusted before a number derived
from it can be*.

- **"the patch stops short of the glyphs", 26 scenes, 73%.** The measure asked whether source ink
  in the rim outside a plate survived the render. It always does - an overlay is CSS over an
  untouched `<img>`, so no pixel outside a plate can change - which made the number a tautology
  reading ~100% wherever a scene has artwork near its lettering, including on synthetic scenes
  built to be correct. Replaced with `CutGlyphInk`: rim ink *joined, through ink, to the ink under
  the plate* - the shape a patch leaves when it cuts a letter in half. Artwork merely standing
  beside the text is not connected to it and no longer counts. 7 of the 8 synthetic scenes now
  read exactly 0 (`synth-text-on-halftone` reads 21%, because a halftone screen connects across
  the plate edge - a known confound, not a defect).
- **"a plate clips under a longer translation", 17 scenes.** Every clipped record in the whole
  corpus overshot by exactly 2 or 3 px, none by more. The shipped re-fit's `height:auto` escape
  had fired on all of them and nothing was hidden: `scrollHeight` and `clientHeight` are each
  rounded to an integer from a fractional layout. The lab's 1 px tolerance was reading that
  rounding as hidden text - and clipping is one of the strategic spec's **hard gates**, so the
  gate would have failed on rounding forever. `evidence.ClipSlackPx` is now 4 px, mirrored in the
  extension and guarded by `TestParityOCRLabEvidenceSchema`.

What survives is the more interesting list, because it is now measured rather than assumed: the
biggest single defect is **11 scenes with no plates at all**, where the reader is simply shown the
original text.

## Out-of-order work already landed (2026-08-11)

One number was set before this phase ran: `ocrRescueLineConf = 80` in
[`internal/ocr/tesseract.go`](../../../internal/ocr/tesseract.go), part of the grey rescue ladder
([`../../research/ocr_grey_rescue_2026-08-11.md`](../../research/ocr_grey_rescue_2026-08-11.md)),
made on the owner's instruction to act on the loop1 report rather than wait for the phase order.

It is derived rather than chosen - genuine rescued lines measured 93.1-97.0 line confidence against
a hallucinated 50.8 - but the evidence is **8 annotated scenes on the dev split**, not the corpus.
**Re-derive it here, do not inherit it.** If the full corpus puts the two bands closer together, the
number moves and the parity guard forces both editions to move with it.

## Handoff notes

This is the only phase permitted to introduce a number. Phase 07 must cite a baseline table for every
change it makes, and must re-run `gate` against this summary as its non-regression reference.

## Rollback plan

Revert the phase commit(s). Deleting `thresholds.json` disables the gate but leaves every measurement
intact - the baseline report is the durable artifact.
