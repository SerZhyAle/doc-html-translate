# The OCR pipe repair: a serif capital I the recognizer reads as "|" is restored before translation

**Status:** Implemented - both editions, tests green, live repro clean on the desktop edition; left: the same page through the extension edition in a real browser
**Priority:** 80
**Date:** 2026-09-28

## 1. Problem

A photographed school-text page (`i.webp`, English exercise, serif face) converted with the OCR
overlay translated into Russian with a vertical bar between almost every sentence: "| am Andrew. |
am a pupil of the 5th form. | get up at seven o'clock." - 15 bars on one page. The bars are not a
translation artefact and not a join separator: the source image is clean, and every bar sits exactly
where the source has the pronoun "I" - a serif capital I is a bare vertical stroke, and the
recognizer read all 15 of the page's standalone "I" tokens as the pipe token `|` on an otherwise
well recognized page. The translation kept each bar and lost the subject with it, so "I get up at
seven o'clock" arrived as the imperative "Вставай в семь"; the browser's own page translation, the
extension's designed flow, preserved the bars in the rendered plate.

Nothing in the existing machinery catches this: `trimOutlierWords` cleans the box, not the token;
`isTranslatable` is a block gate; both editions carried the bars into the plate text.

## 2. Goals

1. A recognized plate reads as the source text did: "I am Andrew. I am a pupil of the 5th form."
2. No pipe the reader's page honestly needs - a table's column rules, a comic's outline read as a
   bar - is destroyed by the repair.

## 3. Constraints

- `OCR-PIPELINE` amendment 1.5 (catalog first): the repair runs in the cluster flush, after the
  translatability gate passes on the raw text, in both editions; no constant added to section 5.
- Cross-edition ticket: the same fix in `internal/ocr` and `extension/src`, pinned by one shared
  fixture, `docs/PARITY.md` row in the same change.

## 4. The rule (amendment 1.5)

In the flush, after `isTranslatable` passes on the raw text, every member line's bare `|` token
(exactly one pipe between the line's spaces) is rewritten to `I` - the ordinary plate and every line
the coverage release splits the cluster into alike. Two guards, both reading the token's own word
box, spare a bar:

- **Token height:** a pipe taller than `OCR_TYPE_SIZE_RATIO (1.6) x` the cluster's median ink height
  is the outline or rule the outlier trim handles (corpus balloon scene: 2.85x its neighbours).
- **Pipe grid:** the cluster's pipe centres are grouped within one median ink height; a group is a
  column when its pipes come from two or more lines; at least **two** columns, each repeating on at
  least **half** the pipe-carrying lines, form a grid, and only grid columns are spared. Every row
  of a real table carries the rule; a page of misreads sprinkles its bars, and even a margin of
  line-initial misreads is one column, not a grid.

A token with no word box behind it has nothing to distrust, and is repaired. The discard record
keeps the raw text.

Measured on the live repro (desktop edition, real pipeline): all 15 bars repaired, the plate reads
"I am Andrew. I am a pupil of the 5th form. I get up at seven o'clock. I wash my face and dress. I
make a bed." - every sentence carries its subject again. The extension edition runs the same rule
through the same fixture set.

Residual, accepted and recorded in the amendment: a lone genuine pipe in prose ("roses | violets")
becomes "roses I violets"; a table whose rules the recognizer read so poorly that fewer than two
columns repeat on half the rows loses its bars - both translate to garbage either way, and the
misread is the common case by the only measurement this rule has.

## 5. Implementation

- `internal/ocr/text.go`: `repairPipeMisreads(line string, protected map[int]bool)` - the token
  mechanic, spared indexes excepted.
- `internal/ocr/tesseract.go`: `repairLinePipes` (the guards, per line) and `pipeGrid` (the column
  grouping and the grid decision); the flush of `clusterLinesRecording` collects every member's
  `pipeCenters`, decides the grid once, repairs `ctexts` before `releaseOversized` splits it, and
  builds the block text from the repaired lines.
- `extension/src/ocr-text.js`: `repairPipeMisreads(line, protectedAt)` - the same mechanic.
- `extension/src/ocr-cluster.js`: `repairLinePipes` and `pipeGrid`; the flush wires them exactly as
  the desktop does.
- `extension/src/ocr-overlay.js`: `collectLines` keeps each bare pipe word's own scaled box
  (`pipes`) - the guards' evidence.
- Shared fixture `tests/testdata/ocr_pipe_repair_cases.json` runs in Go
  (`TestRepairPipeMisreadsSharedCases`) and the extension (`ocr-text.test.mjs`).
- Cluster-level tests in both editions: the school page (all bars repaired), a real table's columns
  (bars kept, a stray bar repaired), the raw-gate order (pipe garbage refused, record raw), the
  token-height guard (the balloon keeps its outline), the coverage release (released lines share the
  repair). The corpus balloon fixture gained its real word box.
- `docs/PARITY.md`: "Pipe misread repair" invariant row.

## 6. Acceptance and evidence

- `go test ./internal/ocr/ ./tests/ ./internal/img/ ./internal/pipeline/ ./tools/ocrlab/...` exit 0;
  extension `npm test` 330 pass / 0 fail (323 baseline + 7).
- Live repro: `doc-html-translate -ocr` on the source page, 0 pipes left in the plate.
- Catalog: `OCR-PIPELINE` 1.5 (amendment, document log, registry row, pointer file) - before the code.
- **Owner to verify in a real browser:** the same image through the extension edition, page
  translation on - the Russian should read "Я Эндрю. Я учусь в пятом классе. Я встаю в семь..." with
  no bars.
