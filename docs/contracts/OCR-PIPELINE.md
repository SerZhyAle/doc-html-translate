# Pointer: OCR-PIPELINE

- **Id:** `OCR-PIPELINE`
- **Version:** 1.0
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
