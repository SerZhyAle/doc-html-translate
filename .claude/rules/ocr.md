---
paths:
  - "internal/ocr/**"
  - "internal/img/**"
  - "internal/comic/**"
  - "extension/src/ocr*"
  - "tools/ocrlab/**"
  - "tests/ocr_*"
---

# OCR overlay and the fidelity lab

Moved verbatim from `AGENTS.md` / `CLAUDE.md` (2026-11-25); the short core there still binds. A section
name quoted below is a section of `AGENTS.md` unless it says otherwise.

## Overlay and plate concealment

- Optional OCR overlay (-ocr): internal/ocr shells out to the external Tesseract binary (parses TSV for bboxes) and rewrites document images into positioned, translatable text plates. It runs in internal/pipeline/pipeline.go after nav injection and before translation (so overlay text is translated too), and is strictly best-effort - a missing tesseract or a failed image never aborts the conversion. English data ships in <exe>/tessdata; other languages download on demand (-ocr-download / GUI) into the per-user folder (os.UserCacheDir()/doc-html-translate/tessdata), catalogue codes only, verified against pinned SHA-256 digests (internal/ocr/download.go). Applies to EPUB and PDF (formats whose images exist at HTML stage).
- Plate concealment modes: every OCR plate records how it hides the source lettering as
  `data-ocr-mode` = `fill` / `reconstruct` / `mask` (decided from the colour ring just outside the
  block), always as CSS over the untouched `<img>` - internal/ocr/conceal.go and
  extension/src/ocr-conceal.js are a pinned PARITY.md invariant. Every constant of the OCR-PIPELINE
  section 5 status table also carries a comment `OCR-OVERLAY rule 13: <status> - <source>.`
  (`derived` / `inherited` / `policy`) above its declaration in both editions;
  tests/ocr_constant_status_test.go fails on a missing marker or a status that disagrees.

## OCR visual-fidelity lab

- OCR visual-fidelity lab: go run -tags nodynamic ./tools/ocrlab verify | fetch | synth | seed | run | score | report | gate | exchange (see tools/ocrlab/README.md)
- Developer tooling, not shipped:
  - tools/ocrlab: the OCR visual-fidelity lab - a licence-gated corpus manifest, engine-independent
    ground truth, the eight quality measures from the spec's table, and a runner that converts each
    scene through the real pipeline and records what headless Chrome actually laid out. Not built by
    scripts/build.ps1 and in no package. Its one hook into shipped code is the opt-in `DOCHT_OCR_DIAG`
    sidecar in internal/ocr (off unless set; an output-identity test proves it changes nothing).
    See tools/ocrlab/README.md and DEV/plan/16_2026-08-11_ocr-visual-fidelity-lab/.
