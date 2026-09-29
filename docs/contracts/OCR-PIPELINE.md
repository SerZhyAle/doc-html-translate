# Pointer: OCR-PIPELINE

- **Id:** `OCR-PIPELINE`
- **Version:** 1.8 (2026-09-25: 1.1 added the line split - word-gap ratio, column regrouping, the stroke test
  between two words, orphans; 1.2 brought the document up to the code - eight corrections, the trim stage,
  the word-height type size and plate font, the grow branch, page OCR, a status column, negative results;
  2026-09-26: 1.3 added the concealment mode - fill / reconstruct / mask chosen from the ring outside the block;
  1.4 wrote the overflow rule for a released plate, implemented the plate font from the type height, put
  the constant statuses at their declarations and named the lab as the section 7 emitter;
  2026-09-28: 1.5 added the pipe repair in the cluster flush - a gated line's bare `|` tokens, the
  misreads of a serif capital I, become `I`; a token taller than the text's own type and a token
  inside the cluster's pipe grid are spared;
  2026-09-28: 1.6 added the anchored rescue admission - a rescue rung keeps a sub-floor line that
  clears 47 with a 4-letter run when its pass holds at least two floor-clearing 4-letter-run anchors
  and one stands at the line's own type size, and the clustering joins a late row of the unordered
  sparse rung into the open cluster it fits inside;
  2026-09-29: 1.7 added the tall-token cut - the split also cuts before a word whose box stands more
  than the type-size ratio above the line's median word height, the touching outlines of two adjacent
  balloons being read as tokens of the stitched line - and the grey sweep, the grey rung spent
  additively on a page that already read, fired by the layout analysis' wordless text regions
  standing outside the plates;
  2026-09-29: 1.8 joined CJK words without the space the recognizer's cut implies - on Han, kana and
  CJK punctuation always, on Hangul only when the boxes stand closer than 0.33 of the median word
  height; a plate's CJK lines join the same way, and the rescue comparator counts the recognizer's
  words instead of the plate text's fields)
- **Home:** the shared contracts catalog, `ocr-overlay/ocr-pipeline.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** producer and owner - the document describes this product's own mechanism, end to end
- **Read by:** FastMediaSorter Android and FastMediaSorter_Lite, which port parts of it

The reference mechanism behind `OCR-OVERLAY`: the detection path, the rescue ladder, clustering, plate
geometry and colour, and every constant with the measurement that set it (section 5). It used to live at
`docs/ocr-pipeline.md` in this repo; it moved out because two other products build against it.

**What this repo owes it**

- Edit the catalog document, never a copy here. A change to a constant or a stage lands there first, with
  its measurement, and only then in [`../../internal/ocr/`](../../internal/ocr/) and
  [`../../extension/src/`](../../extension/src/).
- A breaking change gets a new dated section and a version bump, never an in-place rewrite of what a
  consumer already ported.
- Source comments cite `OCR-PIPELINE.md section N`.
- The cross-edition invariant tables stay in [`../PARITY.md`](../PARITY.md), which is this repo's own
  document: the catalog holds what other products port, PARITY holds what the two editions here must match.

**Conformance.** No vectors in the catalog; the evidence is this repo's own measurement record under
[`../../DEV/research/`](../../DEV/research/) and the parity and diagnostics tests in
[`../../internal/ocr/`](../../internal/ocr/) and [`../../tests/`](../../tests/).
