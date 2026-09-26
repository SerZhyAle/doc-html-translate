# Phase 07 - Type-size break

**Strategic spec:** [`../2026-08-12_ocr-misses-display-lettering-on-saturated-art.md`](../2026-08-12_ocr-misses-display-lettering-on-saturated-art.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** 04, 05, 06
**Steps done:** 5 / 5

## Objective

Close the ticket's second done-criterion: the reported scene's recovered lines must land in plates the
lab can score against the annotation. Phase 06 closed the ticket at `Partial` on the belief that this
was a pitch question and therefore P47's; the phase exists because that belief did not survive being
measured.

## Prerequisites
- [x] Phases 01-06 are ✅ Done.
- [x] A baseline run of the same edition exists to compare against (`temp/ocrlab/p46`, and
      `temp/ocrlab/p46-rus` for the two Cyrillic scenes).

## Files touched
| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `internal/ocr/tesseract.go` | Modified | ≤ 60 |
| `internal/ocr/cluster_test.go` | Modified | ≤ 120 |
| `extension/src/ocr-cluster.js` | Modified | ≤ 50 |
| `extension/test/ocr-cluster.test.mjs` | Modified | ≤ 90 |
| `tests/parity_test.go` | Modified | ≤ 20 |
| `docs/PARITY.md`, `docs/ocr-pipeline.md` | Modified | ≤ 40 |

## Steps

### Step 07.1 - Measure the scene's line geometry
**Files:** none (throwaway probe)
**Depends on:** - start of phase

**Prompt for developer:**
> Dump the line boxes the winning rung actually returns for the reported scene, in the space
> `clusterLines` works in, and decide from those boxes - not from the annotation - what separates the
> headline from the body.

**Verification:**
- The step's own record names the boxes and the ink heights, and states which quantity separates the
  two groups and which does not.

**Status:** `[x]` done - the cut a reader wants (335 px) is **smaller** than a step inside the body
(381 px), so no pitch bound reaches it; the type sizes differ by 1.81x and nothing inside the body
exceeds 1.12x.

---

### Step 07.2 - Derive the ratio from the corpus, not from the scene
**Files:** none
**Depends on:** Step 07.1

**Prompt for developer:**
> Measure both bands the constant has to sit between - the widest line-height spread inside one
> annotated group, and the narrowest step between two groups a reader separates - over every
> annotation in the corpus, using the hand-drawn line boxes.

**Verification:**
- Both bands are named with the scene they were measured on.
- The chosen value lies strictly between them, and the margin on each side is stated.

**Status:** `[x]` done - within-text worst 1.42x (`samson-and-delilah-03-scroll`), across-text
narrowest 1.86x (this poster); 1.6 is the geometric middle of 1.42 and the recognized 1.81, ~13% each
side.

---

### Step 07.3 - Implement the break in `internal/ocr` with tests
**Files:** `internal/ocr/tesseract.go`, `internal/ocr/cluster_test.go`
**Depends on:** Step 07.2

**Prompt for developer:**
> Add the condition to `clusterLines` beside pitch and shared column. The comparison is against the
> cluster's own median height so one odd box cannot end a plate, and an unmeasured height must never
> make the clustering stricter. Cover both directions: the scene that must split, and the corpus's
> widest within-text spread, which must not.

**Verification:**
- `go test ./internal/ocr/ -run "TestCluster|TestSameTypeSize"` passes.
- The new fixtures carry measured boxes, not invented ones.

**Status:** `[x]` done

---

### Step 07.4 - Port it, and pin the meaning in the parity test
**Files:** `extension/src/ocr-cluster.js`, `extension/test/ocr-cluster.test.mjs`, `tests/parity_test.go`
**Depends on:** Step 07.3

**Prompt for developer:**
> Hand-port to the extension and give the pair the treatment the colour fix should have had: pin the
> constant *and* what it is applied to, so a side that weighs the page's median instead of the
> cluster's fails the test.

**Verification:**
- `npm test` green; `go test ./tests/ -run TestParityOCR` green.
- The parity test fails if either side swaps `median(cheights)` for the page median.

**Status:** `[x]` done - 134/134 and 10/10; the two new `meaning` pins are the argument expression and
the symmetry of the ratio.

---

### Step 07.5 - Score it against the corpus and state the cost
**Files:** none (run artefacts under `temp/ocrlab/`)
**Depends on:** Step 07.4

**Prompt for developer:**
> Re-run the dev split and compare per scene against the ticket's own prior run. Isolate this change's
> contribution on the reported scene by re-running it with the rule neutralised. State what got worse
> as plainly as what got better.

**Verification:**
- Every scene that scored before scores the same or better on detection and grouping.
- The reported scene's result is quoted from the run, not from a prediction.

**Status:** `[x]` done - twelve of thirteen annotated scenes identical to the digit; the poster
0.00 -> 0.50 recall, 6 cross-group overlaps -> 0. Isolation run `temp/ocrlab/p46b-nosize` confirms the
change is what moved them. **The cost:** the headline plate is small enough that three of the six
stress translations clip in it where the merged plate had room.

## Phase done criteria
- [x] Every `Step 07.*` is `[x] done`.
- [x] `./scripts/check.ps1` runs; its result and any pre-existing failure are recorded below.
- [x] `docs/PARITY.md` and `docs/ocr-pipeline.md` carry the new invariant and its two measurements.
- [x] Nothing in the corpus regressed on detection or grouping.

## Handoff notes

The gate does **not** exit 0, and neither reason is this phase's:

- **Mean recall 0.6154 against a 0.72 bound.** Two Cyrillic scenes joined the scored set and the
  runner recognizes the whole corpus with one language, so they can only score zero under `eng`.
  Unchanged by this phase and already recorded in Phase 04's notes.
- **Concealment 0.9996 against a 0.28 bound**, and the same on both category bounds. This is the
  2026-08-13 plate-shape change, not grouping: `synth-uniform-paper` cannot trigger the size rule at
  all (three lines of one height) and its diagnostics record a byte-identical plate across the two
  runs while its residual moves 0.085 -> 0.998. It is **P47**'s second defect, now measured against
  the gate for the first time - the changelog row for that change says its corpus re-measure was
  "running", and this is it.
- **Cost 3.54e4 ms against an 1.85e4 bound** - the same machine-speed caveat the thresholds file
  already writes into that bound's own note.

## Rollback plan

One condition in `clusterLines` and its mirror in `ocr-cluster.js`; removing both restores the prior
grouping exactly, as the isolation run demonstrates.
