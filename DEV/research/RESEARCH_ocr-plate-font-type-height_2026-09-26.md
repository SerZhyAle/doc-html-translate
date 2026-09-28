# OCR plate font from the type height, and the overflow lift - lab run

2026-09-26. Feeds [`DEV/plan/done/21_2026-09-23_contract-ocr-pipeline-sync.md`](../plan/done/21_2026-09-23_contract-ocr-pipeline-sync.md)
items A3 and A5, and the contract text `OCR-PIPELINE` amendment 1.4 A and B.

## 1. Question

`OCR-OVERLAY` rule 5, read as binding the plate font (`OCR-PIPELINE` 1.2 item J), asks the font to come
from the median of word heights rather than line boxes. Does moving both editions to that basis, together
with the written overflow rule (a released plate lifted at the picture's bottom edge), cost anything on
the annotated corpus?

## 2. Change measured

- Font basis: `Block.TypeH` / `typeHeight` - the median over the block's lines of each line's word-height
  median - replaces `LineH` / `lineHeight` in the font only. Colour, ring and concealment keep the line
  height.
- Overflow: a released plate whose bottom would pass the picture is lifted so its bottom sits on that
  edge, never above the top.

## 3. Runs

`go run ./tools/ocrlab run` and `npm run ocrlab` (extension) on the working tree of 2026-09-26, against
the concealment-mode runs of the same day as the named baseline. Run folders are temp and not committed.

| Edition | Baseline | This run |
| --- | --- | --- |
| desktop | `temp/ocrlab/p16m-mode` | `temp/ocrlab/t21-font` |
| extension | `temp/ocrlab/p16m-ext-mode` | `temp/ocrlab/t21-ext-font` |

The new runs covered 47 scenes (the `dev` split has grown); scoring covers the 14 annotated scenes, the
same 14 as the baselines.

## 4. Results

- **Every hard gate is 0 in both new runs**: protected damage, merges, clipped plates after stress. The
  extension's one cross-group plate of the baseline (`samson-and-delilah-03-scroll`) is gone: 1 -> 0.
- **Overall aggregates are unchanged** in both editions: recall, precision, IoU mean and worst, covered,
  worst residual, worst halo.
- **`ocrlab gate -against`** fails the same stale checks as the baselines (desktop 6, extension 5): recall
  0.7143 against a 0.72 bound, texture IoU, worst residual 0.9992 against 0.28, named failures, OCR time.
  None of them moved.
- **Font** changed on 7 of 14 desktop scenes, by -20% to +13%, for example `synth-two-columns` 30.3 ->
  23.2 px and `synth-display-lettering` 26.3 -> 29.8 px at the desktop viewport. The grow branch absorbs
  the smaller base where the text allows.
- **Per-scene residual and halo** (desktop; extension in brackets where it differs in kind):

| Scene | Residual | Halo |
| --- | --- | --- |
| `synth-caption-on-gradient` | 0.138 -> 0.037 | 0.081 -> 0.002 |
| `synth-two-columns` | 0.144 -> 0.096 (0.160 -> 0.075) | 0.140 -> 0.020 (0.150 -> 0.003) |
| `synth-uniform-paper` | 0.080 -> 0.089 (0.078 -> 0.092) | 0.059 -> 0.001 (0.055 -> 0.001) |
| `samson-and-delilah-15-court-caption` | 0.114 -> 0.133 (0.108 -> 0.138) | 0.007 -> 0.008 |
| `synth-display-lettering` | 0.216 -> 0.262 (0.214 -> 0.223) | unchanged |
| `samson-and-delilah-03-scroll` | 0.271 -> 0.274 (0.204 -> 0.170) | 0.355 -> 0.354 (0.253 -> 0.227) |

The halo - plate lettering outside the source lines - falls sharply wherever the line box was taller than
the type, which is the defect rule 5 describes. Residual ink rises by 0.01 to 0.05 on three scenes where
the smaller font leaves more of the source's own strokes visible inside the plate's padding; it falls on
two. No worst value moves.

## 5. Section 7 record

`go run ./tools/ocrlab exchange` wrote 47 scene records from each run, `translation` absent, confidences
present in both editions.

## 6. Verdict

Adopted. The change does what rule 5 asks with no hard-gate or aggregate regression; the residual rises
are small and are recorded here rather than argued away.
