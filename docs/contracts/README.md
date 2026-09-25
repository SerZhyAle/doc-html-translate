# docs/contracts/

**Pointers, never copies.** Every contract this product produces or consumes lives in the shared
contracts catalog, one folder per *function* (`ocr-overlay/`, `install-trust/`, ..), and each file here
names one of them: its id, its version, this repo's role, and what this repo owes it. The catalog's
location is written in exactly one tracked file, [`AGENTS.md`](../../AGENTS.md) ("Existing Docs"); nothing
else in this repository may carry that path.

The rule that makes this work: **cite by id, never link.** A source comment says `OCR-OVERLAY rule 8` or
`OCR-PIPELINE.md section 5`, because whoever clones this repository does not have the catalog mounted. A
pointer that grows a second page has become a copy, and two copies drift - which is why the text moved out.

| Pointer | Id | Version | Role |
| --- | --- | --- | --- |
| [OCR-OVERLAY.md](OCR-OVERLAY.md) | `OCR-OVERLAY` | 1.0 | reference implementation (producer + consumer) |
| [OCR-PIPELINE.md](OCR-PIPELINE.md) | `OCR-PIPELINE` | 1.0 | producer & owner - the document describes this product's mechanism |
| [OCR-INVOCATION.md](OCR-INVOCATION.md) | `OCR-INVOCATION` | 1.0 | producer & owner - the CLI another product calls |
| [DIAGNOSTIC-REPORT.md](DIAGNOSTIC-REPORT.md) | `DIAGNOSTIC-REPORT` | 0.9 draft | producer - `internal/report` generates diagnostic zip archives and environment summaries |
| [INSTALL-TRUST.md](INSTALL-TRUST.md) | `INSTALL-TRUST` | 1.0 | producer - bound, not yet adopted |
| [MEDIA-CLASSIFICATION.md](MEDIA-CLASSIFICATION.md) | `MEDIA-CLASSIFICATION` | 0.9 draft | consumer - input format dispatch across books, documents, comics, and images |
| [UPDATE-MANIFEST.md](UPDATE-MANIFEST.md) | `UPDATE-MANIFEST` | 0.9 draft | consumer - release discovery and winget package synchronization |
| [SITE-FAMILY-MAP.md](SITE-FAMILY-MAP.md) | `SITE-FAMILY-MAP` | 1.1 | consumer - the footer family grid and the one contact on every site page |
| [PAGE-STYLE.md](PAGE-STYLE.md) | `PAGE-STYLE` | 1.1 | consumer - the kit `assets/sza-kit.css` byte-identical, page rules in `assets/site.css` |
| [PAGE-CONTENT.md](PAGE-CONTENT.md) | `PAGE-CONTENT` | 1.1 | consumer - landing page order, "Medium app" variant |
| [APP-BEHAVIOUR.md](APP-BEHAVIOUR.md) | `APP-BEHAVIOUR` | 0.10 draft | consumer - GUI launcher behaviour (`cmd/doc-html-ui`) |
| [APP-STYLE.md](APP-STYLE.md) | `APP-STYLE` | 0.10 draft | consumer - desktop GUI and reader styling |
| [ICON-SET.md](ICON-SET.md) | `ICON-SET` | 0.15 draft | consumer - one glyph and one name per meaning, inventory in [`../GLYPH-MAP.md`](../GLYPH-MAP.md) |
| [ICON-RENDER.md](ICON-RENDER.md) | `ICON-RENDER` | 0.13 draft | consumer - grid, theme colour, RTL, accessible names |
| [ICON-EXTERNAL.md](ICON-EXTERNAL.md) | `ICON-EXTERNAL` | 0.10 draft | consumer - no third-party marks; glyph sources on record |
| [CHECK-VERDICT.md](CHECK-VERDICT.md) | `CHECK-VERDICT` | 0.9 draft | consumer - the gate scripts' exit codes and verdict line |
| [CHECK-BASELINE.md](CHECK-BASELINE.md) | `CHECK-BASELINE` | 0.9 draft | consumer, dormant - no baseline file in use |
| [CHECK-PLACEMENT.md](CHECK-PLACEMENT.md) | `CHECK-PLACEMENT` | 0.10 draft | consumer - `configs/check-placement.jsonl` |
| [BUILD-EVIDENCE.md](BUILD-EVIDENCE.md) | `BUILD-EVIDENCE` | 0.9 draft | consumer - subject banners, artifact version, tested tree = tagged tree |
| [REPO-STAMP.md](REPO-STAMP.md) | `REPO-STAMP` | 0.9 draft | producer - `.sza-canon.json` at the repository root |
| [REPO-LAYOUT.md](REPO-LAYOUT.md) | `REPO-LAYOUT` | 0.9 draft | consumer - repository structure and named entry points |
| [HARNESS-PROFILE.md](HARNESS-PROFILE.md) | `HARNESS-PROFILE` | 0.9 draft | not applicable - the shipped harness is never run here, no `.sza-profile.json` |
| [RULE-DELIVERY.md](RULE-DELIVERY.md) | `RULE-DELIVERY` | 0.9 draft | consumer - canon rule set via the `sza` plugin; stamp current at `2026.09.24.1` |
| [DOC-QUALITY.md](DOC-QUALITY.md) | `DOC-INTERNAL-QUALITY`, `DOC-EXTERNAL-QUALITY` | 0.9 draft | consumer - the documentation registry and its gates; gaps in tickets 48, 49 |

Read and **not applicable**: `WAVE-PARTICLES` 0.10 (`animated-backdrop/`, checked 2026-09-25). No site page and no
GUI surface draws a canvas or runs `requestAnimationFrame`; the only background is the kit's CSS blobs. The
product is not in the contract's consumers and owes no row; adopting the backdrop would be a separate opt-in.

Two further documents in the same catalog folder are **records** owned by other products and cite this one:
`OCR-ACCURACY` (FastMediaSorter Android's measurement record) and `OCR-EXCHANGE` (FastMediaSorter_Lite's
positioning exchange). This repo implements neither; it is the counterpart they were measured against, so a
change here that moves a shared constant is announced in their rows rather than applied to their documents.

[`../PARITY.md`](../PARITY.md) is **not** a catalog contract and deliberately stays here: it binds the two
editions of this one product inside this one repository (Go desktop vs the JS extension), which is exactly
what the catalog's rules call a product's own business.
