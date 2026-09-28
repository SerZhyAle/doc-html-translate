# Research: OCR plate concealment modes (fill / reconstruct / mask)

**Date:** 2026-09-26
**Ticket:** [`16_2026-08-11_ocr-visual-fidelity-lab`](../plan/done/16_2026-08-11_ocr-visual-fidelity-lab.md), Phase 07 Steps 07.1-07.2
**Question:** From the pixels just outside a text block, can the overlay tell flat paper, a smooth
gradient and a busy surface (halftone, hatching, an outline or drawing) apart well enough to choose how a
plate hides the source - and where do the bounds go?

## 0. Standing of this note

Strategic spec §9.1 (mask fidelity) and §9.2 (background reconstruction) say to decide from the first
annotated textured-background batch. That batch does not exist: the §4.3 annotation gate is human-owned
and 14 of 47 scenes are annotated. On 2026-09-26 the owner chose to implement the modes now rather than
wait (recorded in the ticket), so this note answers the question from what *can* be measured without
annotation - every block the shipped desktop engine plates on the whole 47-scene corpus - and from the
exact synthetic scenes. It is a bound derived from unannotated pixels, not the annotated method §9 asked
for. §9.1 / §9.2 stay open for the annotated re-derivation.

## 1. Method

A throwaway harness (not committed) ran `ocr.Recognize` - the app's own staging, tesseract 5.4.0, `eng`
- over all 47 images under `test_doc/ocrlab/`, and for every block it produced read the ring
`ringNearerInk` already reads (a band `lh/3`, at least 2 px, just outside the block, four sides with the
corners on top and bottom). Per side: the median colour, and the number of pixels further than
`inkDeviationMin` (90, L1 over RGB) from that median. 162 blocks came back, on the 38 scenes that get a
plate at all; 47 of them have two or more lines. Crops of every contested block were inspected by eye.

## 2. Busy share - when to mask

The busy share (busy pixels / ring pixels) runs **continuously** from 0.000 to 0.799 over all 162 blocks:
there is no natural gap across the whole set. It does not need one, because of a property of the mask:

- A mask paints each line box grown by a pad, clipped to the plate. On a **one-line** block the plate is
  that line box, so the mask paints exactly what the fill paints. The bound decides nothing there.
- On the **47 multi-line blocks** the share splits with a gap: 0.000-0.059 on paper, balloon interiors and
  the gradient (the highest a seven-line caption on the Archie-and-Me cover), and **0.073 and up** where
  the block rectangle reaches past its short lines into something that is not paper.

`MODE_BUSY_MAX = 0.06` sits in that gap. What the high side looks like, from the crops:
`samson-and-delilah-15` block 7 (0.120) - the rectangle's lower right corner covers the balloon's outline
and the blue hatching beyond its short last line; `atomicwar0301` block 0 (0.149) - the lower left corner
covers red artwork outside the balloon under the indented lines; `synth-text-on-halftone` (0.253) - the
dot screen. In all three a block-wide patch erases artwork that the text never touched.

## 3. Spread - when to rebuild a gradient

Twelve blocks have two ring sides whose medians differ by more than 40 (L1). By eye, **most are edges, not
gradients**: `samson-and-delilah-15` block 24 (spread 515) is a white caption whose right ring side lands on
the black panel rule; `join-the-ranks` blocks 2-3 (76, 80) are paper beside a frame line; `kashmir` (96)
is a flat poster whose ring meets the picture edge. A gradient painted between the two sides there would
run from white to black across a caption - the worst plate the program could draw.

What separates them: a ramp passes through its middle colour halfway along, so the two sides **across**
the axis (which span the block's full length) must show the midpoint. The first attempt allowed them to be
within 40 of it and still passed `join-the-ranks` block 3 - a paper side is exactly half the spread from
the midpoint of paper and a rule, and at a spread of 80 that is 40. The rule that holds is relative: both
cross sides within **a sixth of the spread** of the midpoint (the middle third of the ramp).

With it, two blocks rebuild a gradient: the synthetic sky caption (top `176,176,166` to bottom
`205,189,157`, both cross sides `190,182,162`), and `join-the-ranks` block 4 - a genuine one: the poster's
paper is lit unevenly, `197,183,160` on the left to `227,214,195` on the right, with top and bottom at the
middle. Every edge case keeps the fill, which stays inside the block and never reaches the edge beyond it.
`MODE_FLAT_SPREAD = 40`: paper tone between two sides of a flat block reaches 37 (a Hohlwein title), the
synthetic gradient measures 51.

## 4. Mode per synthetic scene

| Scene | Mode | Why |
|---|---|---|
| `synth-uniform-paper`, `synth-two-columns`, `synth-rtl-layout` | fill | flat paper |
| `synth-balloon-on-panel` | **fill** (Step 07.1 expected mask) | the balloon interior is flat white all round the text and the outline lies outside the block, so the fill cannot reach it |
| `synth-side-by-side-balloons`, `synth-adjacent-balloons` | fill | same |
| `synth-caption-on-gradient` | reconstruct | a vertical ramp, its middle on both sides |
| `synth-text-on-halftone` | mask | dot screen round the text |

## 5. Before / after, both editions

`ocrlab run` over the 14 annotated scenes, the stroke-test code of 2026-09-25 as the base
(`temp/ocrlab/p16m-base`, `p16m-ext-base`) against the modes (`p16m-mode`, `p16m-ext-mode`); run
folders are temp and not committed.

- **Every hard gate is 0 in both runs of the desktop edition**; the extension's one cross-group plate
  (`samson-and-delilah-03-scroll`, long Cyrillic) is in its base run too. The gate fails the same stale
  checks before and after (desktop 6 = 6; extension 5 after against 6 before).
- **Residual ink:** identical on every scene except `synth-caption-on-gradient` on the desktop,
  0.1431 -> 0.1385 (halo 0.087 -> 0.081), and `samson-and-delilah-03-scroll` on the extension,
  0.2042 -> 0.2044 (noise). Identity is expected: the residual is read inside the annotated line regions,
  and there the mask paints what the fill paints.
- **Modes agree across editions** on every scene both plate: mask on the scroll, the court caption and the
  halftone, fill elsewhere. The extension reads nothing on the gradient caption, in both runs.

## 6. What the lab cannot see yet

The damage and overlay-area metrics are computed from the **plate rectangles** in the evidence, not from
painted pixels, so the mask's whole point - leaving the space beside and between short lines unpainted -
does not move any number: `overlayPx` is identical before and after. The screenshot shows it (the scroll's
left outline and the halftone between its lines reappear under the mask), but no metric does. Measuring it
needs the painted area read back from the render, and protected polygons on real texture scenes - the
human-owned annotation of §4.3.

## 7. Resulting constants

`modeBusyMax 0.06`, `modeFlatSpread 40`, `modeMaskPadDivisor 6` in `internal/ocr/conceal.go`, mirrored in
`extension/src/ocr-conceal.js`, held equal by `TestParityOCRConcealment`. The ring reuses
`inkDeviationMin`, `ringPadDivisor`, `ringMinPad` and `ringMinSamples` of the colour sampling.
