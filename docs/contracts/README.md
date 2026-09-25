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
| [SITE-FAMILY-MAP.md](SITE-FAMILY-MAP.md) | `SITE-FAMILY-MAP` | 1.1 | consumer - footer tools grid and unified contact information |
| [PAGE-STYLE.md](PAGE-STYLE.md) | `PAGE-STYLE` | 1.0 | consumer - product landing page visual and technical system |
| [PAGE-CONTENT.md](PAGE-CONTENT.md) | `PAGE-CONTENT` | 1.1 | consumer - product landing page content structure |
| [APP-BEHAVIOUR.md](APP-BEHAVIOUR.md) | `APP-BEHAVIOUR` | 0.9 draft | consumer - GUI launcher behaviour (`cmd/doc-html-ui`) |
| [APP-STYLE.md](APP-STYLE.md) | `APP-STYLE` | 0.9 draft | consumer - desktop GUI and reader styling |
| [CHECK-VERDICT.md](CHECK-VERDICT.md) | `CHECK-VERDICT` | 0.9 draft | consumer - the gate scripts' exit codes and verdict line |
| [CHECK-BASELINE.md](CHECK-BASELINE.md) | `CHECK-BASELINE` | 0.9 draft | consumer, dormant - no baseline file in use |
| [CHECK-PLACEMENT.md](CHECK-PLACEMENT.md) | `CHECK-PLACEMENT` | 0.10 draft | consumer - `configs/check-placement.jsonl` |
| [BUILD-EVIDENCE.md](BUILD-EVIDENCE.md) | `BUILD-EVIDENCE` | 0.9 draft | consumer - subject banners, artifact version, tested tree = tagged tree |
| [REPO-STAMP.md](REPO-STAMP.md) | `REPO-STAMP` | 0.9 draft | producer - `.sza-canon.json` at the repository root |
| [REPO-LAYOUT.md](REPO-LAYOUT.md) | `REPO-LAYOUT` | 0.9 draft | consumer - repository structure and named entry points |
| [HARNESS-PROFILE.md](HARNESS-PROFILE.md) | `HARNESS-PROFILE` | 0.9 draft | not applicable - the shipped harness is never run here, no `.sza-profile.json` |
| [RULE-DELIVERY.md](RULE-DELIVERY.md) | `RULE-DELIVERY` | 0.9 draft | consumer - canon rule set via the `sza` plugin; stamp currently stale |

Two further documents in the same catalog folder are **records** owned by other products and cite this one:
`OCR-ACCURACY` (FastMediaSorter Android's measurement record) and `OCR-EXCHANGE` (FastMediaSorter_Lite's
positioning exchange). This repo implements neither; it is the counterpart they were measured against, so a
change here that moves a shared constant is announced in their rows rather than applied to their documents.

[`../PARITY.md`](../PARITY.md) is **not** a catalog contract and deliberately stays here: it binds the two
editions of this one product inside this one repository (Go desktop vs the JS extension), which is exactly
what the catalog's rules call a product's own business.
