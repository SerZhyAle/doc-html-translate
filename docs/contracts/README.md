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
| [OCR-OVERLAY.md](OCR-OVERLAY.md) | `OCR-OVERLAY` | 1.3 | reference implementation (producer + consumer) |
| [OCR-PIPELINE.md](OCR-PIPELINE.md) | `OCR-PIPELINE` | 1.8 | producer & owner - the document describes this product's mechanism |
| [OCR-INVOCATION.md](OCR-INVOCATION.md) | `OCR-INVOCATION` | 1.1 | producer & owner - the CLI another product calls |
| [DIAGNOSTIC-REPORT.md](DIAGNOSTIC-REPORT.md) | `DIAGNOSTIC-REPORT` | 0.12 draft | producer - `internal/report` generates diagnostic zip archives and environment summaries |
| [INSTALL-TRUST.md](INSTALL-TRUST.md) | `INSTALL-TRUST` | 1.1 | producer - bound, adopted (2026-09-28, ticket 27) |
| [MEDIA-CLASSIFICATION.md](MEDIA-CLASSIFICATION.md) | `MEDIA-CLASSIFICATION` | 0.10 draft | consumer - input format dispatch across books, documents, comics, and images |
| [UPDATE-MANIFEST.md](UPDATE-MANIFEST.md) | `UPDATE-MANIFEST` | 0.10 draft | consumer - release discovery and winget package synchronization |
| [SITE-FAMILY-MAP.md](SITE-FAMILY-MAP.md) | `SITE-FAMILY-MAP` | 1.2 | consumer - the footer family grid and the one contact on every site page |
| [PAGE-STYLE.md](PAGE-STYLE.md) | `PAGE-STYLE` | 1.2 | consumer - the kit `assets/sza-kit.css` byte-identical, page rules in `assets/site.css` |
| [PAGE-CONTENT.md](PAGE-CONTENT.md) | `PAGE-CONTENT` | 1.2 | consumer - landing page order, "Medium app" variant |
| [SITE-STRUCTURE.md](SITE-STRUCTURE.md) | `SITE-STRUCTURE` | 0.1 draft | consumer - guide tier, `partial`; gaps in tickets 97, 103; the held-addresses list and gate (rule 8) are ticket 98 |
| [SITE-EXPERIENCE.md](SITE-EXPERIENCE.md) | `SITE-EXPERIENCE` | 0.1 draft | consumer - guide tier, `partial`; gaps in tickets 99, 100, 102 |
| [SITE-REPRESENTATION.md](SITE-REPRESENTATION.md) | `SITE-REPRESENTATION` | 0.1 draft | consumer - guide tier, `partial`; gaps in tickets 101, 102 |
| [APP-BEHAVIOUR.md](APP-BEHAVIOUR.md) | `APP-BEHAVIOUR` | 0.12 draft | consumer - GUI launcher behaviour (`cmd/doc-html-ui`) |
| [APP-SETTINGS.md](APP-SETTINGS.md) | `APP-SETTINGS` | 0.2 draft | consumer - the GUI launcher's settings surface, adopted 2026-10-05 (ticket 93) |
| [APP-STYLE.md](APP-STYLE.md) | `APP-STYLE` | 0.12 draft | consumer - desktop GUI and reader styling |
| [ICON-SET.md](ICON-SET.md) | `ICON-SET` | 0.17 draft | consumer - one glyph and one name per meaning, inventory in [`../GLYPH-MAP.md`](../GLYPH-MAP.md) |
| [ICON-RENDER.md](ICON-RENDER.md) | `ICON-RENDER` | 0.16 draft | consumer - grid, theme colour, RTL, accessible names |
| [WINDOWS-UI.md](WINDOWS-UI.md) | `WINDOWS-UI` | 0.1 draft | consumer - the shared Windows UI profile for the GUI launcher, adopted 2026-10-05 (ticket 93) |
| [ICON-EXTERNAL.md](ICON-EXTERNAL.md) | `ICON-EXTERNAL` | 0.11 draft | consumer - no third-party marks; glyph sources on record |
| [CHECK-VERDICT.md](CHECK-VERDICT.md) | `CHECK-VERDICT` | 0.11 draft | consumer - the gate scripts' exit codes and verdict line |
| [CHECK-BASELINE.md](CHECK-BASELINE.md) | `CHECK-BASELINE` | 0.10 draft | consumer, dormant - no baseline file in use |
| [CHECK-PLACEMENT.md](CHECK-PLACEMENT.md) | `CHECK-PLACEMENT` | 0.11 draft | consumer - `configs/check-placement.jsonl` |
| [BUILD-EVIDENCE.md](BUILD-EVIDENCE.md) | `BUILD-EVIDENCE` | 0.10 draft | consumer - subject banners, artifact version, tested tree = tagged tree |
| [REPO-STAMP.md](REPO-STAMP.md) | `REPO-STAMP` | 0.11 draft | producer - `.sza-canon.json` at the repository root |
| [REPO-LAYOUT.md](REPO-LAYOUT.md) | `REPO-LAYOUT` | 0.10 draft | consumer - repository structure and named entry points |
| [HARNESS-PROFILE.md](HARNESS-PROFILE.md) | `HARNESS-PROFILE` | 0.10 draft | not applicable - the shipped harness is never run here, no `.sza-profile.json` |
| [RULE-DELIVERY.md](RULE-DELIVERY.md) | `RULE-DELIVERY` | 0.11 draft | consumer - canon rule set via the `sza` plugin; stamp reconciliation: ticket 90 |
| [DOC-QUALITY.md](DOC-QUALITY.md) | `DOC-INTERNAL-QUALITY`, `DOC-EXTERNAL-QUALITY` | 0.9 / 0.10 draft | consumer - the documentation registry and its gates; gaps in tickets 48, 49 |
| [CAPTURE-OUTPUT.md](CAPTURE-OUTPUT.md) | `CAPTURE-OUTPUT` | 0.3 draft | consumer - a reader of the `documents` kinds (`text`, `ocr_text`, `translation`) through the TXT input; writes no file of rule 1 |
| [PACKAGE-VERSIONING.md](PACKAGE-VERSIONING.md) | `PACKAGE-VERSIONING` | 0.2 draft | consumer - Windows desktop dotted profile; independent edition clocks remain excepted; frozen `YY.MMDD.HHmm` |
| [INPUT-PARITY.md](INPUT-PARITY.md) | `INPUT-PARITY` | 0.3 draft | consumer, keyboard and mouse - `confirm` / `cancel` / `move` / `search` in the GUI, the extension viewer and the converted book's reader |
| [FDSEC.md](FDSEC.md) | `FDSEC-BEHAVIOUR`, `FDSEC-FORMAT` | 1.5 / 1.3 | consumer, through the installed FileDO CLI - opens `.fd-sec` secret files; the length screen only of the format, adopted 2026-10-06 (ticket 94) |

Read and **not applicable**: `WAVE-PARTICLES` 0.12 (`animated-backdrop/`, checked 2026-09-25, re-checked at
0.12 on 2026-09-29). No site page and no
GUI surface draws a canvas or runs `requestAnimationFrame`; the only background is the kit's CSS blobs. The
product is not in the contract's consumers and owes no row; adopting the backdrop would be a separate opt-in.

**Evaluated and declined (2026-09-29, ticket 69).** Three more draft contracts were read end to end and
deliberately not adopted. `CLI-EVENT-STREAM` 0.10 - this product's GUI wrapper is exactly the supervisor it
addresses, but joining means emitting a second channel from the CLI, which is feature work; ticket 51's
progress work closed as a GUI-presentation feature and introduced no machine channel, so the contract
stays unread-by-code with this dated reason. `INPUT-CHORD` 0.2 - nothing here
stores, captures or globally binds a chord, and the GUI's three in-window shortcuts are out of the
contract's own scope. `CLIPBOARD-GUARD` 0.10 - the product is not in its consumers and makes no direct
Win32 clipboard call; the one copy path is the web view's own.

Two further documents in the same catalog folder are **records** owned by other products and cite this one:
`OCR-ACCURACY` (FastMediaSorter Android's measurement record) and `OCR-EXCHANGE` (FastMediaSorter_Lite's
positioning exchange). This repo implements neither; it is the counterpart they were measured against, so a
change here that moves a shared constant is announced in their rows rather than applied to their documents.

[`../PARITY.md`](../PARITY.md) is **not** a catalog contract and deliberately stays here: it binds the two
editions of this one product inside this one repository (Go desktop vs the JS extension), which is exactly
what the catalog's rules call a product's own business.
