# The anchored rescue admission, measured - 2026-09-28/29

**Ticket:** [`29_2026-09-25_ocr-rescue-third-axis`](../plan/done/51_2026-09-29_ocr-rescue-third-axis.md)
**Question:** the 2026-09-25 note left the type-size anchor one step from shippable: admit a
sub-floor line only where a floor-clearing line of the same pass stands at the same type size, but
first make the sparse rung's row order safe for `clusterLines`, and find a guard against an anchor
that is itself debris. Can the anchor be landed without the two `rus` regressions that sank it?
**Answer:** yes, with one added condition. The pass that read an English page with `rus` data held
**one** confident line, and so did the cartoon whose single-anchor admission plated debris. Requiring
**two** floor-clearing 4-letter-run anchors before a pass may vouch for anything refuses both, keeps
the poster at recall 1.00 under `rus`, and leaves every other scene untouched. Landed in both
editions as `OCR-PIPELINE` amendment 1.6.

## Instrument

The lab corpus over the dev split, twice over: the default `eng` and the annotation's declared
`rus` (`rus` data: tessdata_fast 4.0.0 via `-ocr-download rus`). Baselines are same-day runs on
clean `HEAD` (commit `68ba4c0`, ticket 50's tree); the with-rule runs are the first working tree
with this change. The desktop runner records what the shipped packages produce
(`DOCHT_OCR_DIAG` sidecar); `gate` judges the with-rule runs against the same-day baselines.

| run | tree | language | rule |
|---|---|---|---|
| `temp/ocrlab/t29c-base-eng` | HEAD | `eng` | none (baseline) |
| `temp/ocrlab/t29c2-base-rus` | HEAD (worktree) | `rus` | none (baseline) |
| `temp/ocrlab/t29c-anchor-eng` | this change | `eng` | anchored admission |
| `temp/ocrlab/t29c-anchor-rus` | this change | `rus` | anchored admission |

One recording note, kept because evidence must survive its own housekeeping: the `rus` with-rule run
was launched under the name `t29c-base-rus` in the same batch as the `eng` baseline. Its `go run`
compiled after the change was complete (the eng baseline, launched first from the clean tree,
finished first), which the poster scene's own record confirms - it carries the admitted headline
plate. The directory is renamed `t29c-anchor-rus`; the `runId` inside its `summary.json` still says
`t29c-base-rus`.

The 2026-09-25 pre-runs (`t29-base-*`, `t29-axis-*`) remain the record of the unanchored attempt;
their evidence is quoted where the comparison is against them, never as a baseline.

## What the probe showed that the 2026-09-25 note did not

A throwaway probe re-read the three scenes the unanchored rule moved, line by line, per rung:

- **The poster** (`poster-display-type-on-flat-colour`, `rus`, sparse rung): nine lines in TSV
  order `ЗАЧЕМ, ТРАХАТЬСЯ:, МЫ ЖЕ, ВЗРОСЛЫЕ, ЛЮДИ,, МОЖЕМ, ОБ ЗЛОМ, ПРОСТО, ПОГОВОРИТЬ` -
  `ОБ ЗЛОМ` arrives **before** `ПРОСТО`, the row standing above it. The 2.5 walk closes the open
  body plate the moment `ОБ ЗЛОМ` fails its join test, so admitting the row split the body into
  plates `[375-918]` and `[755-1005]`. That is the whole of the "out of order" blocker: one row,
  one position late.
- **The balloons** (`synth-adjacent-balloons`, `rus`): the winning pass's only floor-clearing line
  is `МОТ ЕУЕМ` (81.4) - itself debris from reading an English scene with Russian data. One anchor,
  and it vouched for `АВОЧТ ТН1$?` (62.0) across the balloon boundary onto the protected outline.
- **A third scene the 2026-09-25 note did not name.** The unanchored rule moved not two but three
  scenes (`louis-brandeis-nomination-...-1916` included): an English cartoon read with `rus`, whose
  pass held exactly one confident line (`А регзо`, 87.9) - and the single-anchor admission plated
  `арропитеве` (57.8), wrong-alphabet debris, at the bottom caption's position. Diffing
  `t29-base-rus` against `t29-axis-rus` per scene found it.

Both failures share one shape, and it is not "the anchor is debris" in general: it is **a pass with
a single confident line**. On the poster the pass holds six floor-clearing lines (80.7, 96.1, 92.6,
95.9, 87.2, 95.0) - a pass that read the page's own script. One confident line is no evidence of the
alphabet at all.

## The rule

`markRescueAdmission` (both editions) runs once per rescue-rung parse, over the pass's complete
lines. It marks a line `rescued` when:

1. its mean confidence clears **47** (the 2026-08-15 band's middle, inherited);
2. it carries a run of **4** letters (inherited);
3. some line of the same pass cleared the floor on its own with a 4-letter run and stands at the
   same type size (`sameTypeSize`, 1.6) - the anchor;
4. the pass holds **at least 2** such anchors.

`keepLine` admits a marked line; the discard record loses the entry and nothing it carries changes.
The grey rungs are the only passes that ask for the admission - the ordinary pass and the screen
passes keep the floors they were measured with.

Its companion is the unordered-row join: the sparse rung's lines carry an `unordered` mark, and a
marked line that fails the ordinary join test but fits inside the open cluster's band - same column
(the cluster's own test), same type size, y-range overlapping the cluster's span - joins the open
cluster instead of closing it. Everything else about the join is the ordinary join. Regrouping the
sparse rung into columns instead (the 1.1 B regrouping run unconditionally) was already measured on
2026-09-25 and rejected: the honest page-wide pitch reference it produces refuses the poster's own
joins (the headline stands 168 px from its second line against a ~95 px body pitch), and the
scene's recall fell back to 0.50 with the headline split in two.

## Result

**`rus`: the poster is fixed and nothing else moves.** Against the same-day baseline:

- `poster-display-type-on-flat-colour`: detection 0.50/0.50 → **1.00/1.00** (2 plates, 0 false
  positives), the headline plate reading `ЗАЧЕМ ТРАХАТЬСЯ:` - exactly the annotation - and the body
  one whole plate `МЫ ЖЕ ЛЮДИ, МОЖЕМ ОБ ЗЛОМ ПРОСТО ПОГОВОРИТЬ`. No merges, no splits, no
  cross-group plates over the stress cases.
- Every other scene: unchanged. The balloon scene keeps its single debris plate and `АВОЧТ ТН1$?`
  stays in the discard record; the Brandeis cartoon never gains `арропитеве`.
- Hard gates all 0 in every category (protected damage, clipped, cross-group, merges, splits).
- Aggregates over the 14 annotated scenes: mean recall 0.393 → 0.429, mean precision 0.429 → 0.464,
  mean CER 0.777 → 0.763, mean IoU 0.548 → 0.571 - all four move the right way, on one scene.

**`eng`: no change.** Under the wrong alphabet no real lettering clears the floor, so no pass holds
the anchors and the admission cannot fire - the property that separates this rule from the two
rejected relaxations, now measured on the same day against the same tree. The per-scene diff of
`DOCHT_OCR_DIAG` records: desktop 0 of 47 scenes changed; the extension edition (its own run,
`t29c-ext-anchor-eng` vs `t29c-ext-base-eng`) likewise 0 of 47.

**Extension edition, `rus`, the poster scene** (`t29c-ext-anchor-rus-poster`): tesseract.js reads
the whole annotation - `ЗАЧЕМ ТРАХАТЬСЯ:` and `МЫ ЖЕ ВЗРОСЛЫЕ ЛЮДИ, МОЖЕМ ОБ ЗЛОМ ПРОСТО
ПОГОВОРИТЬ` as two plates, the body in one piece including `ВЗРОСЛЫЕ`, which the desktop CLI leaves
in the discard record. The admission and the late-row join behave identically on tesseract.js
confidences.

**Gates.** `ocrlab gate` against the frozen thresholds (derived 2026-08-12 from `base06-desktop`)
fails 6 checks on today's clean `eng` baseline and the anchor `eng` run with **identical** numbers
(mean recall 0.7143 vs limit 0.72, texture IoU 0.7359 vs 0.74, worst residual 0.9992 vs 0.28,
review scenes 3 vs 2, cost 33.9s/40.3s vs 18.5s) - pre-existing drift of HEAD against
2026-08-12 thresholds, present without this change and unchanged by it (the `eng` diag records are
byte-identical). The `rus` runs are measurement instruments, not gate subjects (the thresholds are
`eng`-derived; reading English scenes with `rus` fails recall/position/concealment by
construction). The evidence this ticket stands on is the per-scene diff plus the hard gates:
protected damage, clipped plates, cross-group overlaps, merges and splits are all 0 in every
category under both languages, before and after.

## What this settles

- **The corroboration floor is the debris-anchor guard.** "Is the anchor itself debris" was the
  2026-09-25 open question; the answer is that the question is unanswerable per line (confidence
  cannot separate the populations - the parent ticket's own finding) and answerable per pass: a
  pass that read the page's script clears the floor several times, a pass that did not clears it
  once or not at all.
- **The sparse order blocker was one late row, not a general disorder.** The band-overlap join
  fixes exactly that case and cannot merge separated regions: outside the band, in another column,
  or at another type size the walk splits as before.
- **The screen passes stay closed.** Their candidates on the corpus are half-read masthead lines
  (2026-09-25 note); a relaxation on their blurred rendition remains unmeasured and unshipped.
