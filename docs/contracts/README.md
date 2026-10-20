# docs/contracts/

**Pointers, never copies.** Every contract this product produces or consumes lives in the shared
contracts catalog, one folder per *function* (`ocr-overlay/`, `install-trust/`, ..), and each file here
names one of them: its id, its version, this repo's role, and what this repo owes it. The catalog's
location is written in exactly one tracked file, [`AGENTS.md`](../../AGENTS.md) ("Existing Docs"); nothing
else in this repository may carry that path.

The rule that makes this work: **cite by id, never link.** A source comment says `OCR-OVERLAY rule 8` or
`OCR-PIPELINE.md section 5`, because whoever clones this repository does not have the catalog mounted. A
pointer that grows a second page has become a copy, and two copies drift - which is why the text moved out.

Versions below are the ones read on 2026-10-20 (ticket 120, the contract synchronization of 2026-10-17); the
registry row of each contract says `partial` where something is still owed.

| Pointer | Id | Version | Role |
| --- | --- | --- | --- |
| [OCR-OVERLAY.md](OCR-OVERLAY.md) | `OCR-OVERLAY` | 1.3 | reference implementation (producer + consumer); rule 10 and rule 1 are dated exceptions |
| [OCR-PIPELINE.md](OCR-PIPELINE.md) | `OCR-PIPELINE` | 1.9 | producer & owner - the document describes this product's mechanism |
| [OCR-INVOCATION.md](OCR-INVOCATION.md) | `OCR-INVOCATION` | 1.1 | producer & owner - the CLI another product calls |
| [DIAGNOSTIC-REPORT.md](DIAGNOSTIC-REPORT.md) | `DIAGNOSTIC-REPORT` | 0.14 draft | producer, `partial` - `internal/report` generates diagnostic zip archives and environment summaries |
| [INSTALL-TRUST.md](INSTALL-TRUST.md) | `INSTALL-TRUST` | 1.2 | producer - bound, adopted (2026-09-28, ticket 27) |
| [MEDIA-CLASSIFICATION.md](MEDIA-CLASSIFICATION.md) | `MEDIA-CLASSIFICATION` | 0.12 draft | consumer, scoped - a reader of the categories, no classifier; input format dispatch across books, documents, comics and images |
| [UPDATE-MANIFEST.md](UPDATE-MANIFEST.md) | `UPDATE-MANIFEST` | 0.12 draft | declared consumer, not bound for the GitHub Releases link check - no manifest client is implemented |
| [SITE-FAMILY-MAP.md](SITE-FAMILY-MAP.md) | `SITE-FAMILY-MAP` | 1.5 | consumer - the footer family grid, its labels and the one contact on every site page |
| [PAGE-STYLE.md](PAGE-STYLE.md) | `PAGE-STYLE` | 3.1 | consumer, `partial` - the kit `assets/sza-kit.css` byte-identical (re-vendor owed, ticket 119), the adapter is the "UI CONTRACTS" block of `assets/site.css`, `?lang=` locale URLs |
| [PAGE-CONTENT.md](PAGE-CONTENT.md) | `PAGE-CONTENT` | 1.5 | consumer - landing page order, "Medium app" variant |
| [SITE-STRUCTURE.md](SITE-STRUCTURE.md) | `SITE-STRUCTURE` | 0.3 draft | consumer, `partial` - guide tier; the Support page, release notes and not-found page exist; the held-addresses list and gate (rule 8) are ticket 98 |
| [SITE-EXPERIENCE.md](SITE-EXPERIENCE.md) | `SITE-EXPERIENCE` | 0.4 draft | consumer, `partial` - guide tier; the kit hash and the screen reader pass are open |
| [SITE-REPRESENTATION.md](SITE-REPRESENTATION.md) | `SITE-REPRESENTATION` | 0.1 draft | consumer, `partial` - guide tier; gaps in tickets 101, 102; no capability matrix |
| [APP-BEHAVIOUR.md](APP-BEHAVIOUR.md) | `APP-BEHAVIOUR` | 0.12 draft | consumer - GUI launcher behaviour (`cmd/doc-html-ui`) |
| [APP-SETTINGS.md](APP-SETTINGS.md) | `APP-SETTINGS` | 0.7 draft | consumer, `partial` - the GUI launcher's settings surface, adopted 2026-10-05 (ticket 93), read at 0.7 on 2026-10-20 |
| [APP-STYLE.md](APP-STYLE.md) | `APP-STYLE` | 0.13 draft | consumer - desktop GUI and reader styling; the status-fill roles do not apply (no state-filled surface) |
| [ICON-SET.md](ICON-SET.md) | `ICON-SET` | 0.36 draft | consumer, `partial` - one glyph and one name per meaning, inventory in [`../GLYPH-MAP.md`](../GLYPH-MAP.md) |
| [ICON-RENDER.md](ICON-RENDER.md) | `ICON-RENDER` | 0.24 draft | consumer - grid, theme colour, RTL, accessible names; buttons and sizes are [BUTTONS.md](BUTTONS.md) and [UI-SIZING.md](UI-SIZING.md) |
| [WINDOWS-UI.md](WINDOWS-UI.md) | `WINDOWS-UI` | 0.10 draft | consumer, `partial` - the shared Windows UI profile for the GUI launcher, adopted 2026-10-05 (ticket 93) |
| [ICON-EXTERNAL.md](ICON-EXTERNAL.md) | `ICON-EXTERNAL` | 0.12 draft | consumer - no third-party marks; glyph sources on record; the missing-image replacement is owed (ticket 122) |
| [CHECK-VERDICT.md](CHECK-VERDICT.md) | `CHECK-VERDICT` | 0.13 draft | consumer, `partial` - the gate scripts' and tools' exit codes and verdict line |
| [CHECK-BASELINE.md](CHECK-BASELINE.md) | `CHECK-BASELINE` | 0.11 draft | consumer, `partial` - one set baseline in use, `configs/help-baseline.txt` |
| [CHECK-PLACEMENT.md](CHECK-PLACEMENT.md) | `CHECK-PLACEMENT` | 0.11 draft | consumer - `configs/check-placement.jsonl` |
| [BUILD-EVIDENCE.md](BUILD-EVIDENCE.md) | `BUILD-EVIDENCE` | 0.10 draft | consumer - subject banners, artifact version, tested tree = tagged tree |
| [REPO-STAMP.md](REPO-STAMP.md) | `REPO-STAMP` | 0.12 draft | producer - `.sza-canon.json` at the repository root |
| [REPO-LAYOUT.md](REPO-LAYOUT.md) | `REPO-LAYOUT` | 0.11 draft | consumer - repository structure and named entry points |
| [HARNESS-PROFILE.md](HARNESS-PROFILE.md) | `HARNESS-PROFILE` | 0.12 draft | not applicable - the shipped harness is never run here, no `.sza-profile.json` |
| [RULE-DELIVERY.md](RULE-DELIVERY.md) | `RULE-DELIVERY` | 0.12 draft | consumer - canon rule set via the `sza` plugin; stamp reconciliation: ticket 90 |
| [DOC-QUALITY.md](DOC-QUALITY.md) | `DOC-INTERNAL-QUALITY`, `DOC-EXTERNAL-QUALITY` | 0.10 / 0.12 draft | consumer - the documentation registry and its gates; the local-corpus registry is owed (internal, `partial`, ticket 122) |
| [CAPTURE-OUTPUT.md](CAPTURE-OUTPUT.md) | `CAPTURE-OUTPUT` | 0.6 draft | consumer - a reader of the `documents` kinds (`text`, `ocr_text`, `translation`) through the TXT input; writes no file of rule 1 |
| [PACKAGE-VERSIONING.md](PACKAGE-VERSIONING.md) | `PACKAGE-VERSIONING` | 0.5 draft | consumer - Windows dotted profile and declared independent extension clock; frozen `YY.MMDD.HHmm` |
| [INPUT-PARITY.md](INPUT-PARITY.md) | `INPUT-PARITY` | 0.4 draft | consumer, keyboard and mouse - `confirm` / `cancel` / `move` / `search` / `help` in the GUI, the extension viewer and the converted book's reader; the media-viewer rows are out of scope |
| [INPUT-PARITY.md](INPUT-PARITY.md) (decline section) | `INPUT-CHORD` | 0.4 draft | not applicable, scoped decline - no chord is stored, captured or bound; tooltips show canonical names; the extension ships two browser commands |
| [FDSEC.md](FDSEC.md) | `FDSEC-BEHAVIOUR`, `FDSEC-FORMAT` | 1.6 / 1.4 | consumer, through the installed FileDO CLI - opens `.fd-sec` secret files; the length screen only of the format, adopted 2026-10-06 (ticket 94) |
| [CONTEXT-HELP.md](CONTEXT-HELP.md) | `CONTEXT-HELP` | 0.1 draft | consumer, `partial` - F1 and the `?` button of the GUI launcher open one generated help page per surface, adopted 2026-10-20 (ticket 118); the walk and the other ten locales are open |
| [UI-SIZING.md](UI-SIZING.md) | `UI-SIZING` | 1.1 | consumer, `partial` - the desktop launcher implements and measures it (ticket 119, 2026-10-20); extension, reader chrome and site are planned slices ([../UI_PROFILE.md](../UI_PROFILE.md)) |
| [BUTTONS.md](BUTTONS.md) | `BUTTONS` | 2.0 | consumer, `partial` - the desktop launcher implements and measures it (ticket 119); other surfaces planned |
| [BOOLEAN-OPTION.md](BOOLEAN-OPTION.md) | `BOOLEAN-OPTION` | 2.0 | consumer, `partial` - the desktop launcher implements and measures it (ticket 119); extension options and popup planned; the reader toolbar and the site have no boolean options |
| [COLLAPSIBLE-GROUP.md](COLLAPSIBLE-GROUP.md) | `COLLAPSIBLE-GROUP` | 1.1 | consumer, `partial` - the desktop launcher implements and measures it (ticket 119); extension popup and site disclosures planned |
| [SETTINGS-NAVIGATION.md](SETTINGS-NAVIGATION.md) | `SETTINGS-NAVIGATION` | 2.0 | consumer, `partial` - the desktop launcher implements and measures it (ticket 119); the extension options page is a planned slice; the reader pages and the site are not settings surfaces |

Read and **not applicable**: `WAVE-PARTICLES` 0.12 (`animated-backdrop/`, checked 2026-09-25, re-checked at
0.12 on 2026-09-29). No site page and no
GUI surface draws a canvas or runs `requestAnimationFrame`; the only background is the kit's CSS blobs. The
product is not in the contract's consumers and owes no row; adopting the backdrop would be a separate opt-in.

**Evaluated and declined (2026-09-29, ticket 69; `INPUT-CHORD` re-checked 2026-10-20, ticket 120).** Three more
draft contracts were read end to end and deliberately not adopted. `CLI-EVENT-STREAM` 0.10 - this product's GUI
wrapper is exactly the supervisor it addresses, but joining means emitting a second channel from the CLI, which is
feature work; ticket 51's progress work closed as a GUI-presentation feature and introduced no machine channel, so
the contract stays unread-by-code with this dated reason. `INPUT-CHORD` 0.4 - nothing here stores, captures or
globally binds a chord, parses or writes a chord token, so no rule has anything to act on and no vector is
consumed; the product *shows* chords in tooltips in the canonical names (one French string misses), ships two
browser-scope default commands in the extension manifest, and its in-window shortcuts are outside the contract's
own scope. The catalog registry carries a not-applicable row. `CLIPBOARD-GUARD` 0.10 - the product is not in its
consumers and makes no direct Win32 clipboard call; the one copy path is the web view's own.

Two further documents in the same catalog folder are **records** owned by other products and cite this one:
`OCR-ACCURACY` (FastMediaSorter Android's measurement record) and `OCR-EXCHANGE` (FastMediaSorter_Lite's
positioning exchange). This repo implements neither; it is the counterpart they were measured against, so a
change here that moves a shared constant is announced in their rows rather than applied to their documents.
The evidence those records are owed (the type-size ratio on screenshots, the word-gap ratio at PSM 11, the
bracket behind the 0.72 line fill) is carried by ticket 121.

[`../PARITY.md`](../PARITY.md) is **not** a catalog contract and deliberately stays here: it binds the two
editions of this one product inside this one repository (Go desktop vs the JS extension), which is exactly
what the catalog's rules call a product's own business. Its "Contract variants (ticket 120)" section is the
design record of the permanent variants that are not UI profile choices; those that are live in
[`../UI_PROFILE.md`](../UI_PROFILE.md) section 4.
