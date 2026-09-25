# A third axis for the rescue floor, measured and rejected - 2026-09-25

**Ticket:** [`29_2026-09-25_ocr-rescue-third-axis`](../plan/29_2026-09-25_ocr-rescue-third-axis.md)
**Question:** `ЗАЧЕМ` on `poster-display-type-on-flat-colour` comes back correctly at 69.2 under `rus`
and is dropped by `ocrRescueLineConf` (80). Is there an axis, other than confidence and length, that
keeps it without letting the `eng` transliteration (`TPAXATBCR: 4 y`) through?
**Answer:** one axis separates the two languages cleanly, and the lab still rejects it. Nothing ships
except this note. `ocrRescueLineConf` stays at 80.

## Instrument

A throwaway probe (not committed) ran every rescue rung - the three grey rungs and the screen rung -
over all 47 corpus scenes, once with `eng` and once with `rus` (`rus` data: tessdata_fast 4.0.0 via
`-ocr-download rus`). For every line it recorded the mean confidence, the ink height (`inkHeight`),
the longest run of letters, and whether the line cleared the floor. The OSD script verdict was
recorded per image. Lab runs, all on the dev split:

| run | language | rule |
|---|---|---|
| `temp/ocrlab/t29-base-eng` | `eng` | none (baseline) |
| `temp/ocrlab/t29-base-rus` | `rus` | none (baseline) |
| `temp/ocrlab/t29-axis-eng`, `t29-axis2-eng` | `eng` | size rule (the second also had the sparse reorder) |
| `temp/ocrlab/t29-axis-rus` | `rus` | size rule |
| `temp/ocrlab/t29-axis2-rus` | `rus`, three scenes | size rule + sparse reorder |

## The three candidate axes

**Script agreement per image (OSD) - rejected without a lab run.** OSD returns no answer at all on
`poster-display-type-on-flat-colour`. A gate that does nothing when OSD is silent therefore never
admits `ЗАЧЕМ`, which is the one line the ticket is about.

**Agreement between rungs - rejected on the probe.** `ЗАЧЕМ` is read by rung 1 (71.8), rung 3 (69.2)
and the screen rung (81.4). But under `eng` the debris `MODKEM` is also read the same way by three
passes (41.2 / 34.1 / 37.0). Agreement shows that the recognizer is consistent, not that it is right.

**Fit to the page's own type sizes - separates the languages, then fails the lab.** The rule:
- A line under the floor is kept when it carries a run of 4 letters and clears 47. Both numbers are
  inherited from the 2026-08-15 band.
- Its ink height must also be the same type size (`sameTypeSize`, 1.6) as a line of **the same pass**
  that cleared 80 on its own and carries a 4-letter run.

On the probe this does what neither earlier rule did:
- Under `eng`, no line of any rung on the poster clears 80 with a word in it. Only a `\` (81.7) and
  a `4` clear it. So there is no anchor and nothing is admitted.
- Under `rus`, the sparse rung trusts `ТРАХАТЬСЯ:` (80.7, 281 px), and `ЗАЧЕМ` (69.2, 287 px) is the
  same size, so it is admitted.

By construction the rule cannot create a pass's first plate. It can only extend what the floor
already accepted.

## Lab result

**`eng`, all 47 scenes: no change.** `ocr-diag.jsonl` is byte-identical to the baseline apart from
the run-directory name: every plate and every dropped line on every scene is the same. This held in
both `eng` runs.

**`rus`: the poster is fixed, but two hard gates move.**

| scene | metric | baseline | size rule |
|---|---|---:|---:|
| `poster-display-type-on-flat-colour` | recall / precision | 0.50 / 0.50 | **1.00** / 0.67 |
| | CER | 0.367 | 0.284 |
| | plates crossing another group (summed over stress cases) | 0 | **6** |
| `synth-adjacent-balloons` (English scene, read with the wrong `rus`) | protected-area damage | 0 px | **984 px** |
| | plates crossing another group | 0 | **6** |

Both moves have a named cause:

- **The poster.** The headline plate reads `ЗАЧЕМ ТРАХАТЬСЯ:`, exactly the annotation. The rule
  also admits `ОБ ЗЛОМ` (73.9, a misread of `ОБ ЭТОМ`, but in the right alphabet and at body size).
  PSM 11 returns that row after `ПОГОВОРИТЬ`, which ends the running cluster. The body therefore
  comes back as two overlapping plates, `[375-918]` and `[755-1005]`. The second plate matches no
  group, so the metric counts it as crossing.
- **The balloons.** Read with `rus`, the floor already trusts debris: `МОТ ЕУЕМ` at 81.4. The rule
  extends that debris to `АВОЧТ ТН1$?` (62.0) in the neighbouring balloon. The joined plate
  `[88-144]` crosses both balloons and lands on the protected left outline. That scene was already
  wrong before the rule, but the rule makes it worse.

**Tried and reverted: hand a sparse pass to the clustering column by column.** Tesseract documents
PSM 11 as returning text "in no particular order", so `orderColumns` looked like the fix for the body
split. Measured on the poster it made things worse: the headline broke into two plates, `МЫ ЖЕ` was
lost, and recall dropped back to 0.50 with precision at 0.33.

## What this settles, and what is left

- **The size axis is the language test the earlier rules lacked.** Under the wrong alphabet the
  recognizer trusts none of the real lettering, so an anchored rule does nothing. Any future attempt
  should keep the anchor.
- **What blocks it is not the admission itself** but what happens after it:
  - the sparse rung's row order, which `clusterLines` walks as it comes;
  - the debris extension, where an anchor that is itself wrong vouches for more debris.
- **`eng` is the language a default reader gets, and there the rule was a no-op.** The regression is
  confined to `rus`: on the one scene declared `ru` that changed, and on an English scene read with
  the wrong language.
- **The extension edition was not lab-run.** The rule was ported with mirrored tests and reverted
  with the Go side.
