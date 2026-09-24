# Screened captions stay unread on a page whose balloons read fine

**Status:** BlockNeedUserTest
**Priority:** 45
**Date:** 2026-08-12
**Tactical plan:** [`2026-08-12_ocr-screen-pass-for-pages-that-already-read/INDEX.md`](2026-08-12_ocr-screen-pass-for-pages-that-already-read/INDEX.md)

> Cross-edition feature ticket. One feature = one ticket covering every edition.
> Read [`docs/PARITY.md`](../../../docs/PARITY.md) before starting; update it when a shared invariant moves.

## What / why

The halftone rung shipped by
[`2026-08-11_ocr-halftone-defeats-recognition`](2026-08-11_ocr-halftone-defeats-recognition.md)
sits where the grey rescue ladder sits: it fires only for an image that produced **no plates at
all**. That position was chosen because it cannot regress anything, and it does close the
diagnostic scene. What the same cycle measured is that it cannot reach the case a reader actually
meets.

On a real comic page the two kinds of text coexist: dialogue on clean white balloons, and captions
printed as a tint with the lettering laid straight over the dots. The ordinary pass reads the
balloons, so the page is never "unread", so the rung never runs - and the caption stays untranslated
with no explanation, which is the complaint the parent ticket opened with.

The gain is already measured, on the shipped code path, over real material
([`DEV/research/ocr_halftone_2026-08-12.md`](../../research/ocr_halftone_2026-08-12.md) section 5):

| Scene | ordinary pass | screen pass | |
|---|---:|---:|---|
| `le-petit-journal-balkan-crisis-1908` | 55 | 81 | +47% confident words |
| `samson-and-delilah-03` | 76 | 90 | +18% |
| `samson-and-delilah-15-court-caption` | `IN THE COURT OF KING` | `IN THE COURT OF KING TARENT.` | the sentence completes |

## The shape the fix has to have, and why it is not a one-line move

The screen pass is **not** better as a replacement - the same sweep shows it losing badly where
there is no screen (`cover-of-archie-s-pals-n-gals` 16 -> 0 confident words,
`join-the-ranks-of-the-red-army` 71 -> 49). So the pass has to be **additive**: keep every plate the
ordinary pass produced, run the screen pass as well, and merge only the plates that do not overlap
one already there.

That is a real design, not a repositioning:

- **Merge rule.** Overlap against existing plates, at what threshold, and what happens to a new
  plate that overlaps two. A duplicate plate over the same lettering is visible damage.
- **Trigger.** Running a second full recognition on every image that carries a screen roughly
  doubles OCR time on exactly the material this app is used for. Some cheaper trigger - screened
  area with no plate over it, say - probably has to come first.
- **Confidence.** The rescue floor (80) was derived for a *second guess after nothing was found*.
  Adding plates to a page that already read is a different prior and the floor should be re-derived,
  not inherited.
- **Measurement.** Word counts are not enough here: adding plates changes concealment, damage,
  merges and cross-group overlap, which is what the lab scores. This needs annotated **whole
  pages**, and today the corpus has none - the three screened scenes are caption crops.

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[x]` | `screenSweep` in `internal/ocr/tesseract.go`, the merge and the trigger in `screen.go` |
| GUI (`doc-html-ui`) | `[x]` | inherits the pipeline output |
| MSIX Store app | `[x]` | inherits the GUI; adds no writable-directory requirement |
| Browser extension | `[x]` | `screenSweep` in `ocr-overlay.js`, `mergeScreenBlocks` in `ocr-screen.js`. The second `worker.recognize` is gated on the same trigger the desktop uses, so a page with no screen outside its plates pays only the detector - `getImageData` plus a sweep capped at 96 tiles |
| Website / docs | Declined (expected) | recognition quality on screened art is not claimed publicly |

## Shared invariants touched

- The rescue ladder's rung order and the screen rung's parameters are already recorded in
  `docs/PARITY.md` and guarded by `TestParityOCRScreenRung`. A merge rule and its overlap threshold
  become shared invariants of the same kind.

## Done criteria

- [ ] At least three real screened **whole pages** carry reviewed annotations - the crops are not
      enough to score plate composition. **Open, and human-owned:** acquiring the pages and reviewing
      their annotations is the gate, not the tooling - `ocrlab` grades them the day they land.
- [ ] The screened captions on those pages produce plates, and every plate the ordinary pass
      produced is still there, unchanged. **Open** on the first half, which needs the pages above.
      The second half is enforced in code rather than only measured: `mergeScreenBlocks` returns the
      ordinary plates first and unmodified, asserted by `TestMergeScreenBlocksDropsWhatIsAlreadyPlated`
      and its extension mirror, and pinned across editions by `TestParityOCRScreenRung`.
- [ ] Concealment, damage, merges/splits and cross-group overlap do not regress on the dev split.
      **Open** - same gate. These are the dimensions an additive pass moves, and scoring them needs
      whole pages.
- [x] The added cost is stated: unconditional is one decode plus the detector sweep, bounded at
      `ocrScreenMaxTiles` = 96 tiles whatever the page size, so a 600-DPI magazine page costs the same
      measurement as a panel. The second recognition is paid only where the detector finds a lattice
      in the area no plate covers. How often that fires on material which gains nothing is a corpus
      number and belongs to the gate above.
- [x] Both editions, and the shared invariants are in `docs/PARITY.md` under "Additive screen sweep"
      with `TestParityOCRScreenRung` guarding the two constants and three structural facts.
- [x] `./scripts/test.ps1` green, `./scripts/lint.ps1` green, `npm test` 117/117; changelog entry in
      `DEV/CHANGELOG.md`.

## Open questions

- ~~Is there a trigger cheaper than a second recognition?~~ **Answered:** the screen detector itself,
  restricted to the tiles no plate covers (`screenPitchOutside`). It is the same sweep the rung
  already ran, it is capped at 96 tiles, and it asks precisely the question the ticket posed -
  screened area with no plate over it. Whether it is reliable *enough* - how often it fires on
  material that gains nothing - is measurable only against the annotated whole pages.
- Should the merged plates be marked, so a reader (or a bug report) can tell which text came from
  the second pass? **Still open**, and deliberately not decided here: it is a user-visible surface,
  and nothing measured so far says which way it should go. Note that it would be the first place the
  overlay distinguishes one plate from another, so it is a design question rather than a flag.
