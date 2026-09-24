# Tactical plan: 2026-08-15_plate-styling-single-source

**Strategic spec:** [`../2026-08-15_plate-styling-single-source.md`](../2026-08-15_plate-styling-single-source.md)
**Research inputs:** none
**Tier:** Moderate · **Priority:** 44
**Status:** In Progress
**Phases:** 4 / 6 done
**Last updated:** 2026-09-24

> **Scope:** tactical, English, developer handoff. Every step has a verification predicate.
> Rationale lives in the strategic spec.

## Phase overview

| # | Phase | Depends on | Status | Steps | File |
|---|-------|-----------|--------|------:|------|
| 01 | canonical-source | - | ✅ Done | 3/3 | [PHASE_01__canonical-source.md](PHASE_01__canonical-source.md) |
| 02 | desktop-derives | 01 | ✅ Done | 4/4 | [PHASE_02__desktop-derives.md](PHASE_02__desktop-derives.md) |
| 03 | extension-derives | 01 | ✅ Done | 5/5 | [PHASE_03__extension-derives.md](PHASE_03__extension-derives.md) |
| 04 | parity-gate | 02, 03 | ✅ Done | 4/4 | [PHASE_04__parity-gate.md](PHASE_04__parity-gate.md) |
| 05 | parity-record | 04 | ⛔ Blocked (05.3: catalog) | 2/3 | [PHASE_05__parity-record.md](PHASE_05__parity-record.md) |
| 06 | docs-cleanup | all | 🚧 In Progress | 1/2 | [PHASE_06__docs-cleanup.md](PHASE_06__docs-cleanup.md) |

Legend: ⬜ Not started · 🚧 In Progress · ✅ Done · ⛔ Blocked · ⏭️ Skipped

Phases 02 and 03 both depend only on 01 and touch disjoint trees, so they may be done in either
order or in parallel; 04 needs both.

## Pre-implementation blockers

None. Strategic §6 item 1 is Resolved (scope = overlay unit + reader theme palette); items 2 and 3
are constraints carried into the phases below, not research gates:

- Item 2 (offline self-contained output) is honoured by Phase 02 deriving at run time and still
  emitting the styles inline.
- Item 3 (unknown equivalence cases surfacing on the first comparison) is Step 04.3, which triages
  what the gate finds into "unify" or "name it".

## Completion gate

- [ ] All phases ✅ Done.
- [x] User-facing docs updated only if strategic §8 mandates it - it does not; the reader sees no
      change, so `README*` and the site stay untouched.
- [x] Changelog has an entry per modified file.
- [ ] `/spec-check 2026-08-15_plate-styling-single-source` returns Verified.

## How to track progress

1. Before a phase: flip its row to 🚧, update `Phases: X/N`.
2. During: flip a step to `[~]` when started, `[x]` when its Verification passes - never on intent.
3. On phase done: confirm every step `[x]`, confirm Done Criteria, flip row to ✅, bump.
4. If blocked: flip to ⛔, log it; if the whole spec is blocked, set a `Block*` status.

## Change log

- 2026-08-15 - initial tactical plan authored by /spec-tech.
- 2026-09-24 - Phases 01-04 done, 05.1-05.2 and 06.1 done, in a Linux cloud session (no PowerShell, no
  tesseract, no contracts catalog). Left for the owner's machine: Step 05.3 (catalog
  `ocr-pipeline.md`), and Step 06.2's PowerShell gates plus the corpus comic render.
  What ran instead, all on the finished tree:
  - `go test ./...` - every package ok except `internal/pdf` `TestPdfTitle`, which fails identically on
    the base commit (`pdfTitle("C:\Books\My Report.pdf")` on a non-Windows path separator) - not this
    ticket. `go test ./tests/ ./internal/appearance/` ok.
  - `golangci-lint` 2.x with a v2 transcription of `configs/.golangci.yml` (the installed linter
    rejects the v1 file): 7 findings, all in files this ticket does not touch
    (`internal/browser`, `comic`, `i18n`, `mobi`, `rtf`, `tools/ocrlab`); none in the changed files.
    `gofmt -l` clean.
  - `npm test` in `extension/`: 149 / 149 pass (with `vendor/` populated from `node_modules`; the
    tessdata download is blocked by the session's network policy and no test needs it).
  - `node scripts/gen-appearance.mjs --check`: fresh.
  - Criterion 5, by render instead of by corpus page: headless Chromium, old CSS (base commit) against
    new, 12 scenes - the viewer overlay in all four themes, the standalone OCR page with the layer on
    and off, the desktop overlay under a hostile page-level `img` reset with the layer on and off, and a
    converted page in all four desktop themes. 0 differing pixel values in every scene. The same
    harness reports a difference when the viewer's `:not(.ocr-overlay-img)` is removed, so it is not
    blind to the defect class.
  - Done criteria 1, 2, 6 demonstrated on the real tree - see Phase 04 handoff notes.
