# Pointer: PAGE-CONTENT

- **Id:** `PAGE-CONTENT`
- **Version:** 1.1
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

Child pages (extension, docs, privacy) follow the style and footer rules but carry no get-started block - a
sub-page role the contract does not describe yet (ticket 26, B6).

**Conformance.** Read off the rendered page; the acceptance test is the contract's own ("This is ___; it is
for me when ___; I start by ___" before a deep scroll). The static parts are guarded by `tests/site_test.go`.
