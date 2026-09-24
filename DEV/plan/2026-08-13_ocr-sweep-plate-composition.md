# The overlay reads the words and then puts them in the wrong shape

**Status:** Partial
**Priority:** 47
**Date:** 2026-08-13

> **Implemented 2026-08-13**, seven of eight done-criteria met and measured; the one that is not is
> named under "Done criteria" with the reason (it is blocked on the owner by the lab's own rules). Evidence:
> [`DEV/research/ocr_plate_coverage_2026-08-13.md`](../research/ocr_plate_coverage_2026-08-13.md).
> What landed:
>
> - **The occupancy measure the ticket asked for was measured and does not work**, which is the most
>   useful result here. How much of a plate's box its lines fill *by area* puts the defect
>   (`accounts.jpg`, 0.5891) above four scenes that must never be broken (a balloon at 0.4582, a
>   cartoon caption at 0.3621). What does separate is a **conjunction on two other axes**: a cluster
>   is released into its own lines when its box covers more than `ocrMaxPlateCoverage` (0.52) of the
>   image *and* its own line boxes fill less than `ocrMinPlateLineFill` (0.72) of that box's height.
>   Both bracketed from opposite directions; both required, because either alone releases a scene the
>   corpus says is one plate. `accounts.jpg`: 1 plate over 80.6% -> **7 plates, largest 7.67%**.
> - **The paper carrier came back to the plate box**, in both editions, and `docs/PARITY.md` now
>   records it as a decision with the 17%-against-93% measurement behind it rather than as a default.
>   Verified by rendering: the broadside and the poster read as one layer.
> - **The OCR language now follows the document** when the reader did not choose one, via Tesseract's
>   script pass, at a floor bracketed by how badly that detector performs (wrong at 5.00, right at
>   8.15 - floor 6.4). The Russian screenshot now produces **no plates and a line naming the
>   download**, instead of transliterated debris.
> - The no-raster PDF fallback states the fact that decides the outcome instead of guessing at a flag
>   it was never given; `page_*.html` orphans were already removed by the single-page merge.

> Cross-edition feature ticket. One feature = one ticket covering every edition.
> Read [`docs/PARITY.md`](../../docs/PARITY.md) before starting; update it when a shared invariant moves.

## What / why

A 21-document sweep of the local corpus in the shipping OCR mode
(`-notranslate -noopen -force -ocr -ocr-lang eng`, build 26.0813.0245) was converted and every
`index.html` was rendered in headless Edge and read as an image. Every run exited 0 and every log
said `Done.`; `scripts/verify-html.ps1` reported `broken=0` on all of them. **Recognition is not the
problem - composition is.** The words are found, and then they are placed in a shape that damages
the page.

Evidence and method: [`DEV/research/ocr_sweep_2026-08-13.md`](../research/ocr_sweep_2026-08-13.md).
Run artefacts under `temp/ocrsweep/` (gitignored): per-case logs, outputs and `_shots/*.png`.

What a reader gets today, measured as plate area against image area:

| Case | Plates | Largest plate | Plate area summed | Longest plate |
|---|---:|---:|---:|---:|
| `accounts.jpg` | 1 | **80.6% of the image** | 80.6% | 268 ch |
| `pdf-1page-poster_Soldiers-Creed` | 3 | **30.9%** | 31.8% | 635 ch |
| `comic-scan-tiny_First-Earthman` | 47 | 27.7% | 194.6% | **1730 ch** |
| `cbz-tiny_Nyoka` (37 pages) | 480 | 18.3% | 624.9% | 1070 ch |
| `comic-scan-mid_Plastic-Man` (37 pages) | 416 | 10.4% | 518.7% | 661 ch |
| `img-jpeg_Nyoka-comic-page` | 17 | 11.5% | 34.5% | 744 ch |

`accounts.jpg` is the terminal form and the one to look at first: the whole picture becomes **one
plate** holding 268 characters, rendered at a font large enough to fill it, and the screenshot is a
wall of grey words with the source image no longer visible behind them. Nothing in the log
distinguishes this from a good result - it reports `1 image(s) overlaid`.

## The four defects, ranked by what a reader loses

### 1. Lines from unrelated regions merge into one plate

The sweep's dominant failure and the same mechanism P47 was named for, now measured outside the
poster it was found on. `clusterLines` groups on the **image's median line pitch**; on a page whose
text sits in separated regions - a poster, a screenshot, a comic cover, a form - that median is wide
enough to join regions that share nothing.

> **Half of this is fixed, 2026-08-13, and the half that is left is now stated precisely.** P46 added
> a second condition to `clusterLines`: a line joins a cluster only if its ink height is within
> `ocrTypeSizeRatio` (1.6) of that cluster's own median, either way round. Where two regions carry two
> **type sizes**, they now separate - the poster below goes from one plate to two, recall 0.00 to
> 0.50, cross-group overlaps 6 to 0, and `accounts.jpg` splits its header off and drops from one plate
> over 80.6% of the image to 68.3%. `ocrClusterPitchFactor` was not moved, and the ratio is bracketed
> by two corpus measurements (within-text spread 1.42x, across-text step 1.86x).
>
> **What remains is separated regions at the same size**, which is `accounts.jpg`'s list, a form, and
> a screenshot: nothing about their type distinguishes them, so the occupancy measure this ticket asks
> for is still the answer. The three cases below are re-stated with that in mind rather than deleted -
> `Soldiers-Creed` and the `First-Earthman` cover have not been re-measured since the change.

Three shapes of the same defect, all read off the rendered pages:

- `Soldiers-Creed`: twelve creed lines collapse into one plate covering 31% of the poster; the plate
  sits over the middle of the source lettering while the lines above and below it stay uncovered, so
  the reader sees the source text and the recognized text at once.
- `First-Earthman` cover: a plate whose text is the fine print at the foot of the page is positioned
  **over the `PLANET COMICS` logo** at the top.
- `accounts.jpg`: one plate, 80.6% of the image.

### 2. A plate does not conceal the text it replaces

`.ocr-box` is `background:transparent` and the opaque background lives on the inner `.ocr-ink` span,
which hugs the **text**, not the box. When the recognized string is shorter than the source region -
which is every merged plate, and every plate whose recognition dropped words - the source lettering
stays visible around it. On `Lincoln-Proclamation-broadside` (17 plates, none oversized) both
layers are legible simultaneously and the page is harder to read than the untouched scan.

This is the same axis the lab already measures as residual ink (baseline: mean 17%, max 56%) and it
is what makes defect 1 costly rather than merely untidy.

> **Measured against the gate, 2026-08-13, and it is worse than "untidy".** The corpus run P46's last
> phase made (`temp/ocrlab/p46b`, 46 scenes) reports residual ink at a **mean of 93% and a worst of
> 100%**, against a threshold bound of 0.28 set at the recorded baseline. Over the 13 annotated
> scenes the worst is 0.9996; the same scenes measured 0.2707 before the plate-shape change. In other
> words the overlay currently conceals almost nothing it covers.
>
> The attribution is not an inference. `synth-uniform-paper` draws three lines of one height, so the
> new type-size break cannot fire on it, and the app's own diagnostics record a **byte-identical
> plate** across the two runs - same box, same text, same `style` string - while its residual moves
> 0.085 -> 0.998. Same plate, different picture underneath it: what changed is how a plate is drawn.
> The changelog row for that change (2026-08-13 02:40) says its corpus re-measure was "running"; this
> is that measurement, and it is a regression against a recorded bound rather than a neutral trade.

### 3. The OCR language never follows the document

`image.png` is a screenshot of a Russian UI. Converted with the default `eng`, it produces
transliterated debris - `Katanoru-nonyyarenn`, `Npocmorp`, `Bugeo u Kavecteo`, `HaxkmuTe «Hayatb»` -
placed as plates over a perfectly readable interface, so the output is strictly worse than the
input. `2026-08-11_ocr-language-mismatch-is-silent` made the *reporting* honest; the language choice
itself is still a flag the user must know to pass.

### 4. Two honesty defects in the surrounding flow

- **`-ocr` is passed and the page says OCR is disabled.** `pdf-1page-tiny_NASA-Quaoar-sky-chart`
  extracts no raster, and the fallback page hard-codes *"No extractable text layer was found in this
  PDF. OCR is disabled, so the original PDF is shown as-is."*
  ([`internal/pdf/extract.go:1323`](../../internal/pdf/extract.go)) - the sentence is a literal, not a
  reading of the flag. The run had `-ocr`.
- **`page_*.html` carries no overlay at all.** In single-page mode the plates are written into
  `index.html` only; `page_001.html` is left as the bare `<img>`. Anyone who opens a page file
  directly, or links to one, gets an un-overlaid page with no indication that a translated layer
  exists elsewhere.

## Found on the way, not this ticket

- **`pdf-1page-blackletter_Plague-Proclamation-1625` extracts a corrupt raster.** The page comes out
  as a pink-and-brown smear with the text smeared into vertical streaks; OCR then correctly finds
  nothing and the log blames the language data (*"nothing matched the eng (English) data"*). The
  same bytes are produced **without** `-ocr` (sha256 identical), so this is `internal/pdf` raster
  extraction, not the overlay. Needs its own ticket.
- **`scripts/verify-html.ps1` reports a false FAIL on EPUB output.** For an EPUB the root
  `index.html` is a 142-byte redirect to `OEBPS/index.html`; the script checks the redirect, resolves
  its 55 image references against the wrong directory and prints `broken=55`. Pointed at `OEBPS/` the
  same output is `total=55 render=55 broken=0`. A defect in the check, not the product - but it means
  the tool cannot currently gate an EPUB conversion.

## The shape the fix has to have

- **Grouping needs a measure of its own occupancy, not a nudge to a constant.** How much of a plate's
  box is its own recognized text - the question P47 already names. `ocrClusterPitchFactor` is a
  shared invariant derived on balloon scenes and must not be retuned to fix posters. The type-size
  break added on 2026-08-13 is the precedent for how: a second condition with its own constant, that
  constant bracketed by two measurements from opposite directions, and the losing side of the trade
  stated. It is not the occupancy measure and does not replace it.
- **Concealment is a decision, not a side effect of markup.** Whether the box or the ink carries the
  background changes what a translated page looks like on every scene; it has to be chosen against
  the corpus, in both editions, and recorded in `docs/PARITY.md`.
- **A plate that covers most of its image is a failure signal the app can compute.** Whatever bound
  is chosen, it must be derived from the corpus, and the fallback when it trips must be stated -
  refuse the plate, split it, or report the image as unread.
- **No scene that reads today may lose a plate.** The gate runs against a named baseline run, as in
  every prior OCR cycle.

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[x]` | `releaseOversized` in `internal/ocr/tesseract.go`, the box carrier in `overlay.go`, the script rule in the new `internal/ocr/script.go` |
| GUI (`doc-html-ui`) | `[x]` | inherits the pipeline; **no new flag**, so nothing to expose - the script rule rides on `-ocr-lang` being empty, which the GUI already offers |
| MSIX Store app | `[x]` | inherits the GUI; no packaging impact |
| Browser extension | `[~]` | the grouping rule and the plate carrier are ported (`ocr-cluster.js` `releaseOversized`, `ocr-overlay.css` `.ocr-plate`), with tests mirroring the Go ones. **The script rule is not**: its language is an explicit popup choice rather than a value inferred from a translation flag, and the pass needs `osd.traineddata` (~10 MB, neither vendored nor downloaded). Recorded as a divergence in `docs/PARITY.md` with the consequence stated |
| Website / docs | `[x]` | `OCR-PIPELINE.md` (shared contracts catalog) gains §2.0 (language) and the release rule in §2.5; §3.1 now records the carrier decision; `docs/PARITY.md` updated on three rows |

## Shared invariants touched

- `ocrClusterPitchFactor` / `ocrMaxLeadingRatio` and the clustering decision order.
- The plate's background carrier (`.ocr-box` vs `.ocr-ink`) - currently identical in both editions
  and unrecorded in `docs/PARITY.md` as a decision.
- The OCR language default, if detection is added.

## Done criteria

- [ ] **Blocked on the owner, by design.** `accounts.jpg`, `Soldiers-Creed` and the `First-Earthman`
      cover enter the lab corpus with reviewed annotations, as the "separated regions on one image"
      class. Two of the lab's three contributor rules put this outside what an agent may do:
      annotations are truth only when a person has corrected them (`origin: human`,
      `review.annotatedBy`), and `licenceVerifiedBy` is filled by hand after opening the asset's own
      licence page - no code path may write it. Adding half-filled entries would also turn
      `ocrlab verify` red, which gates the lab. What is ready for whoever does it:
      - `accounts.jpg` - **should not enter the versioned corpus at all.** It is a private account
        list; its annotation transcript would publish family members' names in a public repository.
        Its geometry is preserved where the rule is actually measured, in the unit-test fixtures of
        both editions, with the row texts redacted and that stated in place.
      - `Soldiers-Creed` and the `First-Earthman` cover - the rasters are extracted by converting
        `test_doc/pdf-1page-poster_Soldiers-Creed.pdf` and
        `test_doc/comic-scan-tiny_First-Earthman-on-Mars-1944.pdf`; `test_doc/CORPUS.md` already
        records both as public domain (Wikimedia Commons; archive.org CC0/PDM), which is the
        provenance a person needs to check before signing the manifest row.
- [x] No plate on a corpus scene exceeds the chosen occupancy bound, and the bound is stated with the
      measurement that chose it in a note under `DEV/research/`
      ([`ocr_plate_coverage_2026-08-13.md`](../research/ocr_plate_coverage_2026-08-13.md)). The bound
      turned out to be a **pair** of bounds on two axes, and the note says why the single measure the
      ticket named does not work.
- [x] Residual ink under plates improves against the recorded baseline on the dev split; recall, IoU,
      merges/splits and cross-group overlap do not regress. **Lab run `p47b`, 46 dev scenes, 13
      annotated:** worst residual **0.9996 -> 0.2705** against the 0.28 bound, so the three
      concealment gate rows go FAIL -> **PASS**; self-diagnosis over all 46 scenes 93% -> 19% mean.
      Recall 0.6154, IoU 0.7756, worst IoU 0.3489, merges 1, splits 0, cross-group 6, clipped 0,
      drift 0 - every one identical to the run this work started from. The gate still exits 1 on rows
      that were already red before it: `merges`/`crossGroup` (red in the reference run too -
      `synth-two-columns`), `recall`/`review` (the corpus grew from 11 annotated scenes to 13 after
      the thresholds were derived, and both new scenes score 0), and `cost` (the rescue ladder now
      runs every rung). Numbers and the full comparison:
      [`ocr_plate_coverage_2026-08-13.md`](../research/ocr_plate_coverage_2026-08-13.md).
- [x] A Russian-UI screenshot converted with no `-ocr-lang` produces either correct Cyrillic plates or
      no plates - never transliterated debris. Measured end to end on `test_doc/image.png`: 0 plates
      and a line naming the script, the download and the override.
- [x] The no-raster PDF fallback page states what actually happened, with `-ocr` on and off - it now
      states the fact that decides the outcome (no text layer *and* no extractable page image), which
      holds either way, instead of asserting a flag it was never given.
- [x] `page_*.html` either carries the overlay or says where it is: in single-page mode the absorbed
      page files are removed with the merge, so there is no un-overlaid orphan left to open
      (`TestGenerateSinglePageRemovesAbsorbedPages`).
- [x] Both editions, invariants in `docs/PARITY.md`, a parity test pinning them - three rows updated;
      `TestParityOCRClustering` pins both new constants and both halves of the rule on both sides,
      `TestParityOCRFontFit` pins the carrier and fails if `ocr-ink` returns to either edition.
- [x] `./scripts/test.ps1`, `./scripts/lint.ps1`, `./scripts/typo.ps1`, `./scripts/check.ps1` and
      `npm test` (136/136) green; `DEV/CHANGELOG.md` entry.

## Open questions - answered 2026-08-13

- **Is the occupancy bound per plate, or per image?** **Per plate, against the image.** The measure
  is one plate's box as a share of the picture it sits on, so the 480-plates-summing-to-625%-across-37
  -pages case never arises: each page is its own image and each plate is judged on its own. And the
  bound is not on occupancy alone - see the note; the single measure the question assumes does not
  separate the corpus.
- **Should the background move to the box, or should the box shrink to its ink?** **The box.** No
  third option beat 17%, and the two rules that bound the box's over-cover from the other side - type
  size and now coverage - are what makes 17% acceptable where it was not before. The cost the ink
  carrier was adopted for (paper beside a short last line) is back, is stated in `docs/PARITY.md`, and
  is one line of a caption against the whole document showing through every plate.
- **Does language detection belong to OCR at all?** **Yes, but only where the reader did not choose.**
  Cost measured: 0.43 s, once per book rather than once per image. The thing that needed measuring
  turned out to be the detector's accuracy, not its cost - it is wrong more often than right, which is
  what sets the floor at 6.4 and what makes the correction additive (`rus+eng`) rather than a
  replacement.

## Still open after this ticket

- **The `First-Earthman` cover plate overlaps the `PLANET COMICS` logo.** Re-rendered, the ticket's own
  reading of this is wrong: the plate's text is the cover's own top banner line, not foot-of-page fine
  print, and the plate sits on that banner - but its box is taller than its line and rides up over the
  logo. Coverage 0.2774, inside the bound. A plate-box height question, not a merge; needs its own
  ticket.
- **`Soldiers-Creed` still misses lines.** The creed is one text and one plate is right for it; what
  is left after the carrier move is recall, not composition.
- **`pdf-1page-blackletter_Plague-Proclamation-1625` extracts a corrupt raster** - unchanged, still
  needs its own ticket (see "Found on the way").
- **`scripts/verify-html.ps1` false-FAILs EPUB output** - unchanged.
- **Released rows sample their own colours.** On `accounts.jpg` the six released rows each contain a
  coloured avatar, so `blockColors` now returns six different inks where the merged plate returned
  one. Every row still clears the contrast floor; the page is more colourful than its source. A
  consequence of judging colour on a smaller box, recorded rather than fixed.
- **`thresholds.json` is stale against its own corpus.** It was derived from 11 annotated scenes;
  there are 13, and the two added since are the hardest in the corpus, so `recall` and `review` fail
  on arithmetic rather than on a regression. `cost` is stale for a different reason - the rescue
  ladder now runs every rung. Re-deriving those three belongs to a dated baseline run, not to a
  ticket that would like them green; the lab's own rule 2 says as much.
- **The halo metric penalises a correct plate on a screened ground.** `synth-text-on-halftone` scores
  0.6532 against 0.0091 in the reference run, with an identical plate rect - because the sampled pair
  was inverted then and is right now (the scene's generator draws black on cream under a grey dot
  screen). Halo counts ink in a ring inside the group's bounds, so the inverted dark plate blended
  into the screen and scored better. No gate, but the metric is measuring the wrong thing here.
