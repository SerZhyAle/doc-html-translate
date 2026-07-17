# Docs + site reality check - close the roadmap by auditing what we claim against what ships

**Status:** Verified
**Priority:** 14
**Date:** 2026-07-17

## What / why

Every ticket in this queue (P1-P13) carries its own narrow "Website / docs" row, scoped to the one claim
that ticket's fix touches - e.g. P5 corrects the TIFF promise, P4 corrects the supported-format list only
"if claimed anywhere". That per-ticket scoping is deliberate and correct while the queue is open: it keeps
each ticket small and stops docs edits from blocking a code fix. But it also means no single ticket owns
the **cumulative** picture - twelve independent, narrowly-scoped edits to the same handful of files
(`README.md`, `index.html`, `docs.html` / `docs.ru.html` / `docs.uk.html`, `extension.html`) land in
sequence, and nothing re-reads them together afterward to check the combined result still tells one
coherent, accurate story.

Concretely, by the time P1-P13 close: eight extensions that used to "succeed" into garbage now fail
honestly or convert correctly (P4), TIFF's status changes from "silently broken" to either "supported" or
"explicitly declined" (P5), FB2 and saved-HTML keep their images (P6-P7), OCR quality and plate fit
improve measurably (P9-P10), and CBZ/CBR/CB7/CBT go from "no path" to a real input format with a stated
7-Zip dependency and a browser-side partial (P12). Any of `README.md`, `index.html`, `docs.html` (+ ru/uk),
`extension.html`, and `docs/PARITY.md` that still describes the *old* state - or omits the new one - is a
false claim sitting in production, which is the same class of problem P1 (cli-diagnostics-honesty) fixes
for the CLI's own output, just on the marketing/docs surface instead.

## Scope

Read every one of these against the shipped behaviour, not against what a previous version of this file
said:

- `README.md`, `AGENTS.md` (supported-format lists, known limitations, CLI examples)
- `index.html`, `docs.html`, `docs.ru.html`, `docs.uk.html`, `extension.html` (site copy, three languages
  kept in sync per the feature-prominence rule already in use for `CLAUDE.md`)
- `docs/PARITY.md` (port map + intentional-divergences tables - each landed ticket was supposed to update
  this already; verify it, don't re-derive it)
- `DEV/CHANGELOG.md` (entries exist and match what actually shipped)

Out of scope: rewriting site design/visual style (tracked separately if ever needed), and any claim not
touched by this roadmap sweep (don't go relitigate unrelated copy).

## Done criteria

- [x] Every format this roadmap changed (`.docx`/`.odt`/`.pptx`/`.xlsx`/`.djvu`/comics/TIFF/FB2/HTML/TXT
      encodings) is described identically - and correctly - across `README.md`, the three-language site,
      and `docs/PARITY.md`.
- [x] No page claims a capability that was declined (e.g. TIFF, or CBR/CB7 in the extension) and no page
      omits a capability that shipped (e.g. CBZ/CBT in the extension, legacy TXT encodings).
- [x] Three-language parity verified by diffing structure across `docs.html` / `docs.ru.html` /
      `docs.uk.html`, not just spot-checked in English.
- [x] `docs/PARITY.md`'s port map and intentional-divergences tables match the code, cross-checked against
      each closed ticket's own parity checklist rather than re-derived from scratch.
- [x] `DEV/CHANGELOG.md` has one entry per landed ticket.

## Audit outcome (2026-07-18)

Ran after P1-P13 all landed (P13 typography closed 2026-07-18 00:18; the raster fix and the rest earlier
on 2026-07-17). Read every surface against the shipped behaviour recorded in `docs/PARITY.md`, the closed
tickets, and `DEV/CHANGELOG.md` - not against a prior version of this file.

**Already correct and in sync (no change needed):** `README.md`, `index.html` (en/ru/uk), `docs.html` /
`docs.ru.html` / `docs.uk.html`, `extension.html` (en/ru/uk), `AGENTS.md`, and `docs/PARITY.md` all
describe comics (CBZ/CBR/CB7/CBT, CBR/CB7 via 7-Zip; extension CBZ/CBT-only), TIFF (app transcodes,
extension declines), FB2/HTML images, and the standalone-image OCR path correctly and consistently across
all three languages. No page claims a declined capability (no TIFF or CBR/CB7 in any extension surface),
and the extension pages state CBZ/CBT + "CBR/CB7 desktop-app only" in en/ru/uk. Grep confirmed **no** site
page claims `.docx`/`.odt`/`.pptx`/`.xlsx`/`.djvu` support. The per-ticket "Website / docs" rows did their
job; the cumulative read-together found the story already coherent.

**Two gaps found and fixed:**

1. **`DEV/CHANGELOG.md`** - the P8 (pdf-raster-extraction) entry had lost its own row: its description was
   mashed onto the P9 row as a stray 5th table column, with no timestamp/path/target. Split into its own
   proper row, so the log now has one clean entry per landed ticket (P1-P13).
2. **Legacy TXT encodings + binary refusal** (P2/P3/P4) shipped but were undocumented on the user-facing
   surfaces. Added a short, honest note to `README.md` Behavior Notes and the "Supported Formats" section
   of `docs.html` / `docs.ru.html` / `docs.uk.html` (three languages kept in sync): `.txt` encoding
   detection (BOM -> UTF-8 -> legacy Cyrillic cp1251/koi8-r/cp866), and that an unreadable binary is
   refused with a named format instead of becoming a garbage document. `docs/PARITY.md` already carried
   the deep version of both invariants.

Out of scope, left alone: site visual design, and typography of the HTML docs prose (P13 deliberately
scoped its sweep to Go string literals + `_locales`; a docs-prose sweep is a separate concern, not this
format-reality audit).
