# The bundled pdftotext ships with its licences

**Status:** Implemented (2026-09-25; the release-page and winget half is proven by the next release run)
**Priority:** 55
**Date:** 2026-09-25

> Found while ticket 24 wrote `THIRD-PARTY-NOTICES.txt` for the Material Icons glyphs
> ([ticket 24](24_2026-09-23_contract-iconography-sync.md), A9). Out of that ticket's scope: this is
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

- [x] The four files' origin and hashes are on record: [`internal/bundledtools/PROVENANCE.txt`](../../../internal/bundledtools/PROVENANCE.txt).
  The four files are byte-identical to Git for Windows' `mingw64/bin` (2.53.0.windows.2 on the build machine):
  MSYS2 packages `mingw-w64-x86_64-xpdf-tools 4.00-1` (built with GCC 7.3.0), `mingw-w64-x86_64-gcc-libs 15.2.0-11`,
  `mingw-w64-x86_64-libwinpthread 13.0.0.r488.g3fedac280-2`. Upstream `xpdf-4.00.tar.gz` downloaded and hashed
  (sha256 `ff3d92c4..16d6`).
- [x] `THIRD-PARTY-NOTICES.txt` carries their licences and the source offer: a new section after the glyphs with
  the copyright lines, upstream source links with the archive hash, a fallback request path via the repo's issues,
  then the Xpdf README and pdftotext manual page (Xpdf's README asks for both beside its binaries), GPL v2, GPL v3,
  the GCC Runtime Library Exception 3.1 and the winpthreads licence. The extension copy stays the glyph part only.
- [x] Every release channel that ships the Windows executable ships the notices: the installer and the MSIX already
  staged the root file; `.github/workflows/release.yml` now puts it in the zip (which winget's portable manifest
  unpacks whole) and uploads it as a release asset beside the standalone `.exe` files. Proven in CI only when the
  next release runs.
- [x] A test fails when a bundled file is missing from the notices: `tests/bundled_notices_test.go`
  (`TestBundledFilesAreInTheNotices` also checks each file's hash line in PROVENANCE; `TestReleaseWorkflowShipsTheNotices`
  pins the workflow). Mutation-checked with a throwaway `libdummy.dll`: both messages fired.

## Decisions (the former open questions)

1. **Record 4.00 as is.** An upgrade is a behaviour change with its own test surface (ticket 11's cache check,
   antivirus reputation of a new binary); it is not needed to meet the licence duty.
2. **Link the upstream archive of the exact version**, with its SHA-256, plus the build recipes, plus a request path
   through the repository's issues if a link dies. No per-release copy of the source.

## Deviation from the direction

The provenance file is `internal/bundledtools/PROVENANCE.txt`, beside the `pdftotext/` folder rather than in it:
`//go:embed pdftotext` embeds the folder whole, so a text file there would ship inside the executable and change the
content hash that names the unpacked cache folder.

## Found on the way

`pdftotext.exe` has never been tracked in git - the `*.exe` ignore rule catches it and only the DLLs are
re-included. So a clean checkout, and therefore the CI release build (the portable `.exe` assets and the winget zip),
embeds the three runtime DLLs without the executable and falls back to a system pdftotext; only the locally built
installer and MSIX carry the full set. The notices still apply to every channel (the DLLs ship everywhere). Filed as
[ticket 35](../35_2026-09-25_pdftotext-missing-from-ci-builds.md). The `.gitignore` comment also called the set
"Poppler"; corrected to Xpdf.
