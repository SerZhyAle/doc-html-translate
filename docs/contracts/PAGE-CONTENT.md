# Pointer: PAGE-CONTENT

- **Id:** `PAGE-CONTENT`
- **Version:** 1.2
- **Home:** the shared contracts catalog, `product-web-pages/PAGE-CONTENT.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - variant "Medium app"
- **Wire carrier:** none - rendered page markup

What this repo owes it (the landing `index.html` and the ten locale landings):
- Page order: sticky header with one neutral link to `#get` -> eyebrow, outcome H1 and tagline -> what it is and
  who it is for -> one proof strip -> the get-started block `#get` with every real channel (Microsoft Store,
  GitHub Releases, setup installer, winget copy box, Chrome Web Store, Edge Add-ons) and a three-step
  quickstart -> scenarios -> details in numbered disclosure groups -> footer.
- The product mark once, in the header, monochrome; the hero does not repeat the mark or the name.
- The free, key-less path first; paid and metered modes are named after it, never as the first action.
- Related tools only as contextual body copy (Fast Media Sorter for Windows as a companion); the full family is
  the footer.
- Never an invented channel, command or claim; the release link resolves the latest release at run time and
  falls back to `/releases/latest`.

Deviation (owner decision 2026-10-06, to be recorded as a dated registry exception and raised as a catalog
amendment): the hero of the landing and of the ten locale landings carries the full name "Doc-HTML-Translate"
in a prominent size beside the real application icon (`assets/doc-html-translate.ico`), repeating the header
mark, and a row of help and documentation links under the proof strip. The contract's "the hero does not repeat
the mark or the name" is overridden for this site until the catalog is amended.

Child pages follow style and footer rules without the landing order or distribution block (1.2).
`extension.html` distributes its edition and keeps its own get-started block; docs, privacy and trust
pages need none. The old child-page exception is closed.

**Conformance.** Read off the rendered page; the acceptance test is the contract's own ("This is ___; it is
for me when ___; I start by ___" before a deep scroll). The static parts are guarded by `tests/site_test.go`.
