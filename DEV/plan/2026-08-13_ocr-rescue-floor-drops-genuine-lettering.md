# The rescue floor was derived for inventions and is now rejecting real words

**Status:** Partial
**Priority:** 49
**Date:** 2026-08-13
**Closed:** 2026-08-15 - evidence in
[`DEV/research/ocr_rescue_floor_2026-08-15.md`](../research/ocr_rescue_floor_2026-08-15.md)

> **What the measurement found, ahead of the original text below.** The floor could not be
> re-derived, because the two populations **overlap on confidence**: genuine rescued lettering runs
> 32.8-69.2 and invented lettering 8.4-73.9, with the highest invention *above* the highest genuine
> line. This ticket's third bullet anticipated exactly that and asked for the alternative to be
> named rather than for the number to be nudged.
>
> The axis that does separate them is **length**: of the 175 lines the floor rejected over 46 scenes,
> the eight highest-scoring are all debris of one to six characters, and requiring a run of four
> letters leaves nine lines bracketing an **empty band** (36.1 `allie` / 58.3 `KPECTbAHHH!`). That
> rule was implemented in both editions and **the corpus rejected it**: under the app's default
> `eng` a Cyrillic poster then gets a 782x310 px plate of transliterated debris where it previously
> got none. `ocrRescueLineConf` stays at **80** and the gap stays open.
>
> **Status is `Partial`, not `Implemented`.** What ships is the instrument (the floor's discard
> record) and the measurement; the recall this ticket asked for does not. The next attempt needs a
> third axis - most promisingly, not running the rescue ladder at all when the script check says the
> language is wrong, which is where this poster's debris comes from.

> Cross-edition feature ticket. One feature = one ticket covering every edition.
> Read [`docs/PARITY.md`](../../docs/PARITY.md) before starting; update it when a shared invariant moves.

## What / why

`ocrRescueLineConf` is **80**: a line recovered by the rescue ladder is kept only if its mean word
confidence clears that. The number was chosen for a different question - "the ordinary pass found
nothing, so how sure must a second guess be before it paints words over artwork?" - and the band it
was set between is recorded in `internal/ocr/rescue_test.go`: genuine rescued lettering measured
93.1-97.0, hallucinated lettering 50.8. With those two numbers, 80 sits in an empty gap and any value
in it is as good as any other.

The gap is no longer empty. On `poster-display-type-on-flat-colour`, with the ladder's sparse rung and
`rus` data, the app's own staging returns:

| line | mean confidence | kept at 80? |
|---|---:|:---:|
| `ЗАЧЕМ` | 69.2 | no |
| `ОБ ЗЛОМ` (a misread of `ОБ ЭТОМ`) | 73.9 | no |
| `ТРАХАТЬСЯ:` | 80.7 | yes, by 0.7 |

`ЗАЧЕМ` is the poster's first word, correctly read, and it is thrown away. Its group is two lines;
with only the second recognized, the plate reaches IoU 0.455 against the group's bounds and the lab's
0.5 detection floor scores the group a miss. **That single dropped line is the whole difference
between recall 0.50 and recall 1.00 on the scene P46 was opened for** - the rest of that ticket's work
is done and measured.

P46's tactical plan put this out of scope in advance and said why, and also said what would reopen it:
*"If the Phase 04 run shows the floor rejecting genuine rescued lettering, it becomes a follow-up
ticket."* It does, twice on one image. This is that ticket.

## The shape the fix has to have

- **The band has to be re-measured, not re-guessed.** 93.1 and 50.8 are two points from one cycle. A
  floor is only as good as the distribution behind it, and the corpus now holds scenes that cycle did
  not: display lettering on flat colour, a halftone caption, two Cyrillic posters.
- **A dropped line is invisible today, and that is half the defect.** Nothing in the log or the
  diagnostics says "six lines were recognized and two were discarded by the floor". Whatever the floor
  becomes, the discard should be recordable - the lab cannot score a decision it cannot see.
- **Confidence may not be the only axis.** `ОБ ЗЛОМ` at 73.9 is a *misread*, not an invention: the
  right decision for it is arguably still to drop it, while `ЗАЧЕМ` at 69.2 should be kept. If one
  number cannot separate those two, say so and name what else is available (agreement between rungs,
  the line's fit to the page's own type sizes) rather than moving the number until one scene passes.
- **Lowering a floor can only add plates.** Every scene that reads today must be re-scored, and the
  hallucination side is the risk: the same 21-document sweep that measured composition also found a
  Russian UI screenshot producing transliterated debris under `eng`. A lower floor makes that worse.

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[x]` | `internal/ocr/tesseract.go`: `keepLine`, `Result.Dropped`, the diagnostics record. `ocrRescueLineConf` unchanged at 80 |
| GUI (`doc-html-ui`) | `[x]` | inherits the pipeline; no surface |
| MSIX Store app | `[x]` | inherits the GUI |
| Browser extension | `[x]` | ported to `ocr-cluster.js`: `keepLine`, `droppedLines`. `OCR_RESCUE_LINE_CONF` moved here from `ocr-overlay.js`, beside the other floors and the predicate that applies it |
| Website / docs | `[x]` | `OCR-PIPELINE.md` §2.4 (shared contracts catalog) records the re-measurement and why the floor did not move |

## Shared invariants touched

- `ocrRescueLineConf` itself, and the two measured bands `TestRescueConfidenceFloorIsStricter` pins
  around it (93.1 / 50.8) - a re-derivation replaces those numbers, so the test is part of the change.

## Done criteria

- [ ] The floor is re-derived from a distribution measured over the current corpus, and both edges of
      the band are named with the scenes they came from, in a note under `DEV/research/`.
- [ ] `poster-display-type-on-flat-colour` reaches recall 1.00, or the reason it cannot is stated as a
      measurement rather than as an expectation.
- [ ] No scene gains a plate over artwork that holds no text: precision and the concealment/damage
      metrics do not regress on the dev split against a named baseline run.
- [ ] A line discarded by the floor is visible somewhere a person or the lab can read.
- [ ] Both editions, `docs/PARITY.md` updated, a parity test pinning the value and its band.
- [ ] `./scripts/test.ps1`, `./scripts/lint.ps1`, `npm test` green; `DEV/CHANGELOG.md` entry.

## Open questions

- **Is one floor right for every rung?** The sparse rung reads a poster; the Leptonica rung reads a
  gradient. Their confidence distributions have never been compared.
- **Should a line the floor drops still count towards `resultStrength`?** Today a rung's strength is
  the words that survived its floor, so lowering the floor also changes which rung wins.
- **Does the ordinary floor (50) need the same look?** It was derived against the same two bands and
  has the same provenance; nothing here has measured it.
