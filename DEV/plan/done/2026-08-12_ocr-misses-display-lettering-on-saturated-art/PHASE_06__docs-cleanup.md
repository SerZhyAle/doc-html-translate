# Phase 06 — Docs cleanup

**Strategic spec:** [`../2026-08-12_ocr-misses-display-lettering-on-saturated-art.md`](../2026-08-12_ocr-misses-display-lettering-on-saturated-art.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** all
**Steps done:** 3 / 3

## Objective

Leave the record accurate: what changed, what it cost, and what is still open.

## Prerequisites
- [x] Phases 01-05 are ✅ Done.

## Files touched
| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `DEV/CHANGELOG.md` | Modified | ≤ 60 |
| `DEV/plan/ROADMAP.md` | Modified | ≤ 20 |
| `DEV/research/RESEARCH_INDEX.md` | Modified | ≤ 10 |

## Steps

### Step 06.1 — Changelog entries
**Files:** `DEV/CHANGELOG.md`
**Depends on:** - start of phase

**Prompt for developer:**
> Add one row per meaningful change via `./scripts/add_to_dev_log.ps1`, in the house style: what a
> reader of a poster sees now that they did not see before, and the corpus result behind it.

**Verification:**
- `DEV/CHANGELOG.md` names the ticket and the corpus outcome.
- `./scripts/typo.ps1` exits 0.

**Status:** `[x]` done

---

### Step 06.2 — Index the research note
**Files:** `DEV/research/RESEARCH_INDEX.md`
**Depends on:** Step 06.1

**Prompt for developer:**
> Add `ocr_display_lettering_2026-08-12.md` to the index with a one-line summary of what it settles.

**Verification:**
- The filename appears exactly once in `DEV/research/RESEARCH_INDEX.md`.

**Status:** `[x]` done

---

### Step 06.3 — Update the queue row with the measured outcome
**Files:** `DEV/plan/ROADMAP.md`
**Depends on:** Step 06.2

**Prompt for developer:**
> Rewrite the P46 row to say what was measured rather than what was intended, in the style the
> neighbouring rows use, and note anything the cycle left open.

**Verification:**
- The P46 row names the before/after corpus numbers.
- No row claims a state the ticket's own status line contradicts.

**Status:** `[x]` done

## Phase done criteria
- [x] Every `Step 06.*` is `[x] done`.
- [x] `./scripts/check.ps1` green.
- [x] Strategic §8 mandates no user-facing docs change, and none was made.

## Handoff notes

See INDEX.md Completion gate.

## Rollback plan

Revert the phase commit - documentation only.

## Execution notes (2026-08-12)

- Steps 06.1 and 06.2 are done: a changelog row via `add_to_dev_log.ps1`, and the research note
  indexed in `DEV/research/RESEARCH_INDEX.md`.
- Step 06.3 rewrote the P46 row to what was measured - **Partial**, not Implemented - and added a P47
  row naming the grouping question the corpus run exposed.
- The ticket does not reach Verified. The reported scene still fails its own done-criterion, and the
  gate still exits 1 for two reasons recorded in the research note (two new Cyrillic scenes the
  runner can only read with `eng`, and a cost figure taken while the test suite ran on the same
  machine). Neither is a regression: every scene the baseline scored keeps its recall to the digit.

## Closing pass (2026-08-13)

The three steps were executed on 2026-08-12 but their checkboxes were never flipped, so the phase
read as not started while its work was on disk. Re-verified each predicate against the tree rather
than against the note, then flipped them:

- **06.1** - the row was there but named neither the ticket nor the corpus result, which is half of
  the predicate. It now carries both: the ticket path, and the dev-split outcome against
  `temp/ocrlab/base06-desktop` by name. `./scripts/typo.ps1` exits 0.
- **06.2** - `ocr_display_lettering_2026-08-12.md` appears exactly once in `RESEARCH_INDEX.md`
  (`grep -c` = 1). Unchanged.
- **06.3** - the P46 row names the before/after (no plates at all -> recall 1.00 on the sibling) and
  says **Partial**, which is what the ticket's own status line says. Unchanged.

**Two defects surfaced while verifying, both inside this ticket's own colour fix, both fixed here.**

- **The ring test had drifted between the editions.** `blockColors` in `ocr-overlay.js` used one
  variable for two heights - the line, and the 1.3-line strip the ink is sampled in - and passed the
  strip to `ringNearerInk`, so the extension weighed a band `0.43 x` the line height where the
  desktop app weighs `0.33 x`. Same block, different vote. Both sides now derive the ring from the
  raw line height and name the ink strip `firstBand`.
- **`docs/PARITY.md` described an algorithm neither edition runs any more.** It still called the ink
  the *mean* of the deviating pixels; both editions moved to the median on 2026-08-13. The row now
  says median, carries the measurement, and describes the ring as a third of a line - the code
  comments beside `lh / 3` said "a line tall", which is what the row was copied from.

`TestParityOCRPlateColourOrientation` only pinned the vote floor and the call shape, so it saw
neither. It now also pins the band divisor on both sides, the two heights being kept apart by name,
and the ink being a median - three checks the drift would have failed.

**A third correction, outside this ticket's change but found the same way and fixed rather than
filed.** `docs/PARITY.md`'s "plate shape is per line" bullet named `padStyles`, `padRects` and
`.ocr-pad` - symbols that exist in neither edition and in no commit (`git log -S padStyles --all` is
empty). What ships is what `TestParityOCRFontFit` pins: a transparent `.ocr-box` and the sampled
paper on an inline `.ocr-ink` span with `box-decoration-break:clone`. The bullet now describes that,
and records the consequence the 2026-08-13 sweep measured - paper that follows the string leaves the
source lettering showing around any plate whose text is shorter than its region. It was caught by
trying to restate the design in another project's specification, which is the only reader that had
to act on it.

**One helper script was broken and is fixed here rather than worked around.**
`scripts/add_to_dev_log.ps1` used `Add-Content`, so every row it wrote landed at the *bottom* of a
table that is read newest-first - this ticket's own row sat below entries from 2026-07-01. It now
splices the row under the `|---|---|---|---|` rule and refuses to guess if that rule is missing. The
two rows it had already misfiled (this ticket's, and the 2026-08-13 i18n row) were moved into place;
their text is untouched apart from 06.1's addition.

Gate for this pass, run fresh: `./scripts/test.ps1` green (all packages `ok`), `./scripts/lint.ps1`
`Lint passed`, `./scripts/typo.ps1` `Typo check passed`, `npm test` 131/131, `./scripts/check.ps1`
exit 0. After the two fixes above: `go test ./tests/ -run TestParityOCR` 10/10 PASS, `npm test`
131/131, `lint.ps1` and `typo.ps1` green again.
