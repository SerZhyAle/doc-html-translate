# The CI release build embeds no pdftotext.exe

**Status:** In Progress
**Priority:** 60
**Date:** 2026-09-25

> Found by [ticket 33](done/33_2026-09-25_bundled-binaries-notices.md) while recording the provenance of the
> bundled pdftotext set.

## What / why

The Windows build embeds `internal/bundledtools/pdftotext/` whole (`//go:embed pdftotext`) and unpacks it on first
use. Of the four files in that folder only the three MinGW runtime DLLs are tracked in git: `pdftotext.exe` matches the
`*.exe` rule in `.gitignore`, which re-includes `internal/bundledtools/pdftotext/*.dll` and `build/*.exe` but not it.
`git log --all -- internal/bundledtools/pdftotext/pdftotext.exe` is empty.

So a clean checkout builds an executable that carries the DLLs and no pdftotext; `PDFToTextPath` reports
`ErrNotBundled` and the converter falls back to a pdftotext on the system, or to the pure-Go path when there is none.
`.github/workflows/release.yml` builds from a clean checkout, which means the standalone `.exe` assets on the release
page and the zip winget installs do not carry the bundled pdftotext, while the locally built installer and MSIX do.
The `.gitignore` comment says the set "ships inside every package"; for two channels it does not.

Not yet measured: what a user of the portable build loses on a real PDF when no system pdftotext is present.

## Direction

1. Track `pdftotext.exe` (a `.gitignore` re-include, like the DLLs), so every checkout and every channel embeds the
   same set. Its hash is already on record in `internal/bundledtools/PROVENANCE.txt`.
2. A guard: the release workflow (or a test run in CI) fails when the embedded set has no `pdftotext.exe`, so a
   channel cannot silently ship without it again.

## Done criteria

- [ ] `git ls-files internal/bundledtools/pdftotext` lists all four files. The ignore rule no longer matches
      (`git check-ignore -q --no-index` exits 1, `git add --dry-run` adds the exe); ticks when the commit lands.
- [ ] A CI-built executable unpacks `pdftotext.exe` on first PDF conversion. Needs the next CI release build.
- [x] A check fails when the set lacks `pdftotext.exe` in a release build.

## Implementation (2026-09-26)

- `.gitignore` re-includes `internal/bundledtools/pdftotext/pdftotext.exe` next to the DLLs; `.gitattributes` marks the
  whole set `-text`, so no checkout rewrites the bytes the hash table pins.
- `.github/workflows/release.yml` step "Verify the bundled pdftotext set" runs before the first `go build`: every row
  of the hash table in `internal/bundledtools/PROVENANCE.txt` must be present with that SHA-256, and the table must
  have a `pdftotext.exe` row. Run locally on the real set: exit 0, "verified: 4 files"; on a copy with the DLLs only:
  exit 1, "Bundled set lacks pdftotext.exe".
- `tests/bundled_notices_test.go` `TestBundledSetIsTrackedAndGuarded`: the folder holds `pdftotext.exe`, no file of the
  set is git-ignored, and the workflow runs the guard before it builds.
- `PROVENANCE.txt` and the `pdftotext_windows.go` comments no longer describe the set as partly untracked.

## Open questions

1. ~~Is tracking a 1.5 MB third-party binary in git acceptable, or should CI fetch it from the upstream Git for Windows
   package and verify its SHA-256 instead?~~ Tracked (Direction 1): the 2.4 MB `libstdc++-6.dll` beside it already is,
   and a fetch would tie every release build to a third-party download and a URL that has to stay alive.
