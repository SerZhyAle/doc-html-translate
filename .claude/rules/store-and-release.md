---
paths:
  - "tools/store/**"
  - "msix/**"
  - "winget/**"
  - "scripts/release.ps1"
  - "scripts/promo-metrics.ps1"
  - "scripts/contract-gate.ps1"
  - "docs/discoverability.json"
  - "DEV/RELEASE.md"
  - "assets/social-card-*.png"
---

# Store listing, release checklist, promotion

Moved verbatim from `AGENTS.md` / `CLAUDE.md` (2026-11-25); the short core there still binds. A section
name quoted below is a section of `AGENTS.md` unless it says otherwise.

## Store listing

- Store listing (WINDOWS-STORE rules 7-9, 12, 13; ticket E0135): `tools/store/listingData.csv` is the **authoritative listing file** - the product's own Partner Center export, patched in place by `tools/store/build-store-listing-csv.ps1` from the decks `tools/store/listing/*.txt`, fill-only unless `-Refresh <field>` (which lists every cell it replaces); `msix/out/store-import/` holds a gitignored copy for the dashboard's folder import, never a second source. The builder exits 1 and writes nothing for a shipped language with no column (or no deck), a cell over its cap (Feature 200, ShortDescription 1000, Description 10000, ReleaseNotes 1500 after CRLF, 21 words over SearchTerm1..7) and any break of the export's dialect; it writes the export's own dialect (BOM, CRLF between records, LF inside cells, no trailing newline). `tools/store/listingData.sample.csv` is the same export with the copy cells emptied; both CSVs are pinned `-text` in `.gitattributes`, and `-FillNothing -Out <scratch>` on each is byte-identical. The cases run as `pwsh tools/store/test-store-listing-csv.ps1` (3 s) and as `go test -tags nodynamic ./tests -run TestStoreListing`.

## Release checklist and its verdicts

- Release checklist (prints only, runs nothing but read-only checks): ./scripts/release.ps1. Its header
  carries three verdicts the tag step takes as input: the gate evidence, the contract gate
  (`scripts/contract-gate.ps1`: the pointers in `docs/contracts/` against the catalog registry -
  PASS / WARN pass, FAIL / UNVERIFIED block, and an unreachable catalog is UNVERIFIED, never PASS) and the
  Store preflight (the sza plugin's shared read-only `tools/Store-Preflight.ps1`, `msstore submission get` only:
  BLOCKED on a pending submission, a version not above the live one or a price that is not Free; MANUAL, never
  a block, when the channel is unavailable). No agent runs `msstore` in a preflight or between releases;
  the release's Store submission step (ticket E0156, `./tools/store/submit-submission.ps1`) is the one
  machine write into the Store, under the owner's pre-authorization of 2026-11-17

## Free-promotion campaign

- Free-promotion campaign (ticket 111; canon PROMOTION.md): `./scripts/promo-metrics.ps1` writes a dated, read-only counters checkpoint (GitHub, winget, Chrome, Edge, Store - `unknown` where unreadable, no telemetry); `tools/store/make-social-card.ps1` renders `assets/social-card-*.png`; `tools/store/make-demo.ps1` records and encodes `assets/demo.*`; the discoverability fields live in `docs/discoverability.json`. Every outward step of the campaign waits for the owner's go.
