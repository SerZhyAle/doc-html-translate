# The rescue ladder needs a third axis to keep real lettering it now drops

**Status:** Partial
**Priority:** 49
**Date:** 2026-09-25

> **Measured 2026-09-25, nothing shipped.** Evidence:
> [`DEV/research/ocr_rescue_third_axis_2026-09-25.md`](../research/ocr_rescue_third_axis_2026-09-25.md).
>
> - The OSD axis and the rung-agreement axis were rejected on the probe.
> - The **type-size anchor** was implemented in both editions. It keeps a sub-floor word only when a
>   line of the same pass cleared the floor at the same size. Under `eng` it left all 47 scenes
>   byte-identical, and under `rus` it brought the poster to recall 1.00. It still failed the lab's
>   hard gates under `rus`, for two reasons:
>   - `ОБ ЗЛОМ` arrives out of order from PSM 11 and splits the body into two overlapping plates;
>   - on an English scene read with the wrong `rus`, the anchor is itself debris, and the rule
>     extended that debris onto a protected outline (984 px).
> - Both were reverted. Reordering the sparse rung by column was tried as a fix and made the poster
>   worse.
>
> **Next step:** keep the anchor, and first make the sparse rung's row order safe for
> `clusterLines`. Then look for a guard against an anchor that is itself debris.

> Cross-edition ticket. One feature = one ticket covering every edition.
> Read [`docs/PARITY.md`](../../docs/PARITY.md) before starting; update it when a shared invariant moves.
> Split out of [`done/2026-08-13_ocr-rescue-floor-drops-genuine-lettering`](done/2026-08-13_ocr-rescue-floor-drops-genuine-lettering.md),
> which closed on the measurement; evidence in
> [`DEV/research/ocr_rescue_floor_2026-08-15.md`](../research/ocr_rescue_floor_2026-08-15.md).

> **Owner machine only for the measuring steps.** The lab corpus (`test_doc/ocrlab/`) is gitignored,
> the Commons media is not reachable from a cloud session, and the scene this ticket is about is the
> owner's own material. Code can be drafted anywhere; no candidate is accepted without a lab run.

## What / why

A line the rescue ladder recovers is kept only above `ocrRescueLineConf` (80). On
`poster-display-type-on-flat-colour`, read with `rus`, the poster's first word `ЗАЧЕМ` comes back
correctly at 69.2 and is dropped, and that one line is the whole difference between recall 0.50 and
1.00 on the scene.

The previous ticket established two things, both measured:

- **Confidence cannot separate the populations.** Genuine rescued lettering runs 32.8-69.2, invented
  lettering 8.4-73.9; the highest invention sits above the highest genuine line.
- **Length alone lets the wrong language through.** A four-letter-run rule with a floor of 47 recovers
  the headline under `rus`, and under the app's default `eng` paints a 782x310 px plate of
  transliterated debris (`TPAXATBCR: 4 y`) over the same poster. The corpus refused it.

The asymmetry is the finding: the scene improves under the right language and regresses under the
default one. Whatever keeps `ЗАЧЕМ` has to know which of those two situations it is in.

## Candidate axes, none measured yet

- **Script agreement, per image.** The script check (`script.go`) runs OSD once, on the book's first
  image, and only when the language was not chosen. The debris comes from a rescue pass reading one
  script with another script's data. Gating a relaxed rule - not the ladder itself - on "OSD does not
  contradict the language for *this* image" would leave the `eng` case at today's behaviour. Known
  weakness, from the same research: OSD is unreliable below 6.4 and called one Cyrillic poster `Latin`
  at 3.97, so the gate must fail closed (no relaxation when OSD is silent or unsure).
- **Agreement between rungs.** Every rung now runs; a line two rungs return with the same text is less
  likely to be invented than one only the sparse rung sees. Needs the per-rung record, which the
  discard record does not carry today.
- **Fit to the page's own type sizes.** `sameTypeSize` already measures ink height against a cluster;
  a rescued line whose height matches a line that cleared the floor is a candidate to keep.

## Measurement hygiene carried over

- The 2026-08-15 note classified `KPECTbAHHH!` (58.3, under `eng`) as genuine. Under OCR-OVERLAY
  rule 10 a transliteration is not a correct plate, so "genuine" has to be split into *correct under
  the language used* and *real lettering, wrong alphabet* before any band is drawn again.
- Re-score against a named baseline on the dev split, both `eng` (the default a reader gets) and the
  annotation's declared language, and report both.
- Ticket 15 lands first: without the discard record for a no-plate image, a scene the relaxed rule
  newly reads cannot be compared with what it rejected before.

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[ ]` | `internal/ocr/tesseract.go` `keepLine` and whatever feeds it the new axis |
| GUI (`doc-html-ui`) | `[ ]` | inherits the pipeline |
| MSIX Store app | `[ ]` | inherits the GUI |
| Browser extension | `[ ]` | `ocr-cluster.js` `keepLine`; tesseract.js confidences differ, so its own lab run is required |
| Website / docs | `[ ]` | `docs/PARITY.md`, `OCR-PIPELINE.md` §2.4 |

## Done criteria

- [x] At least one candidate axis measured over the corpus under both `eng` and the declared language,
      in a dated note under `DEV/research/`, with the rejected candidates recorded as such
      ([`ocr_rescue_third_axis_2026-09-25.md`](../research/ocr_rescue_third_axis_2026-09-25.md)).
- [ ] `poster-display-type-on-flat-colour` reaches recall 1.00 under `rus`, and gains no plate under
      `eng` - or the reason it cannot is stated as a measurement.
- [ ] No scene gains a plate over artwork that holds no text: precision and the concealment/damage
      metrics do not regress on the dev split against a named baseline run.
- [ ] Both editions, `docs/PARITY.md` updated, parity test extended to the new rule.
- [ ] `./scripts/test.ps1`, `./scripts/lint.ps1`, `npm test` green; `DEV/CHANGELOG.md` entry.

## Open questions (carried from the parent ticket)

- **Is one floor right for every rung?** The sparse rung reads a poster; the Leptonica rung reads a
  gradient. Their confidence distributions have never been compared.
- **Should a line the floor drops still count towards `resultStrength`?** Today a rung's strength is
  the words that survived its floor, so any relaxation also changes which rung wins.
- **Does the ordinary floor (50) need the same look?** Same provenance, never re-measured.
