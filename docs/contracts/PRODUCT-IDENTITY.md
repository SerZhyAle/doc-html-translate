# Pointer: PRODUCT-IDENTITY

- **Id:** `PRODUCT-IDENTITY`
- **Version:** 0.1 draft
- **Home:** the shared contracts catalog, `product-identity/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, `partial` - adopted 2026-11-20 (ticket D2HTML-0162)
- **Wire carrier:** `assets/identity/identity-manifest.json`, `schemaVersion` 1

One picture of the product everywhere a user meets it: the program icon, the installer, the GUI window, the browser
extension, the Microsoft Store listing, the site, the social card and the help pages all show the same mark - a white
sheet with a folded corner and `</>` on the navy plate `#1E3A8A`.

What the repo holds:
- **The master** is the mark drawn in code, [`internal/iconart/mark.go`](../../internal/iconart/mark.go), in three
  drawings by size (from 96 px with the words DOC and HTML, 24 to 95 px, below 24 px). The generator writes them as
  `assets/identity/master/mark.svg` (the master), `mark-medium.svg` and `mark-small.svg`.
- **The generator** is `pwsh -NoProfile -File scripts/generate-icon.ps1`: `tools/icongen` draws every icon, the store
  display images (`tools/store/logos`), the social card's mark and the help-page head; `tools/store/make-social-card.ps1`
  composes the Open Graph card and the GitHub social preview from that mark; `tools/store/make-chrome-promo.ps1`
  composes the Chrome small promo tile from the mark and an extension screenshot; `tools/icongen -manifest` measures every
  picture into the manifest. A rerun on an unchanged mark changes no byte. Rerun it after the store screenshots or the
  demo are regenerated - the manifest records their hashes as `other.*` records with source and licence (rule 16).
- **The help pages** (`tools/helpgen`) inline the `help.head` picture beside every title, link the product's favicon,
  and show the ICON-SET glyph of every control an element entry names (`glyph` in the copy deck).
- **The MSIX logos** are drawn into the package at build time by `msix/build-msix.ps1` (`icongen -msix`); the generator
  keeps a byte-identical copy of the 23 that show the mark under `assets/identity/out/msix` so the manifest names them.
- **No diagram** exists in the README, help, site or docs; the first one is written as text and generated (section 3.2).

**Conformance.** Rungs 1, 2, 3, 4 and 7. Rung 2 is read with the catalog checker patched as
`PROPOSAL-2026-11-19-checker-byte-shift.md` proposes: the published checker misreads every size above 255 px.
`tests/icons_test.go` holds every PNG and ICO rendition equal to what `internal/iconart` draws;
`internal/iconart/identity_test.go` holds the SVG forms. At release 26.1126.2358 the help set was regenerated with the
head picture and the glyphs, and the 13 `reading-view-*` Store screenshots were captured from the real build, one per
language. Open: replacing the images in Partner Center, which is a hand upload. Proposal filed:
`PROPOSAL-2026-11-20-doc-html-translate-code-master-and-reduced-forms.md`.
