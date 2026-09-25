# The bundled pdftotext ships with its licences

**Status:** Draft
**Priority:** 55
**Date:** 2026-09-25

> Found while ticket 24 wrote `THIRD-PARTY-NOTICES.txt` for the Material Icons glyphs
> ([ticket 24](done/24_2026-09-23_contract-iconography-sync.md), A9). Out of that ticket's scope: this is
> the product's own licence duty for a binary it embeds, not a vocabulary rule.

## What / why

The Windows build embeds a pdftotext (`internal/bundledtools`, `//go:embed pdftotext`) and unpacks it to a
cache on first use. What ships inside every Windows executable, read 2026-09-25:

| File | Size | What it is |
| --- | --- | --- |
| `pdftotext.exe` | 1 537 966 | `pdftotext -v`: "pdftotext version 4.00, Copyright 1996-2017 Glyph & Cog, LLC" - Xpdf 4.00, GPL (v2 or v3) |
| `libstdc++-6.dll` | 2 461 684 | GCC runtime, GPL v3 with the GCC Runtime Library Exception |
| `libgcc_s_seh-1.dll` | 150 196 | GCC runtime, same |
| `libwinpthread-1.dll` | 64 931 | MinGW-w64 winpthreads, MIT-style / ZPL |

`THIRD-PARTY-NOTICES.txt` names none of them, the binaries carry no version resource saying where they came
from, and nothing records the build they were taken from. Distributing Xpdf's pdftotext under the GPL
obliges the product to ship the licence text and to offer the corresponding source (or point at it). The
portable exe on the release page ships with no notices file at all, while the installer and the MSIX stage
one (`scripts/build-installer.ps1`, `msix/build-msix.ps1`).

## Direction

1. Record the provenance: which Xpdf build (the upstream archive and its SHA-256) and which MinGW-w64
   runtime the four files came from - `internal/bundledtools/pdftotext/PROVENANCE.txt`, like
   `assets/glyphs/PROVENANCE.txt`.
2. Add their section to `THIRD-PARTY-NOTICES.txt` (and its extension copy stays glyph-only, since the
   extension bundles no pdftotext): licence names, copyright lines, the GPL text, and the written offer or the
   upstream source link the GPL accepts.
3. Ship the notices with every channel that carries the binary: the portable exe on the release page gets
   the file beside it (release assets), and winget's portable manifest gets it too if it can.
4. A guard in `tests/`: every file under `internal/bundledtools/pdftotext/` is named in the notices file.

## Done criteria

- [ ] The four files' origin and hashes are on record.
- [ ] `THIRD-PARTY-NOTICES.txt` carries their licences and the source offer.
- [ ] Every release channel that ships the Windows executable ships the notices.
- [ ] A test fails when a bundled file is missing from the notices.

## Open questions

1. Upgrade to a current Xpdf (4.05) or Poppler build while recording the provenance, or record 4.00 as is?
   An upgrade touches ticket 11's cache check.
2. Source offer: a link to the upstream archive of the exact version, or a copy attached to each release?
