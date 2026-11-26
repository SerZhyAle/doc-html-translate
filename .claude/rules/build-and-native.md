---
paths:
  - "scripts/build*.ps1"
  - "scripts/lib/gobuildtags.ps1"
  - "scripts/lib/goversioninfo.ps1"
  - "scripts/generate-icon.ps1"
  - "internal/jxl/**"
  - "internal/iconart/**"
  - "tools/icongen/**"
  - "go.mod"
  - "assets/*.ico"
  - "extension/icons/**"
  - "installer/**"
---

# Build scripts, native code posture, icons

Moved verbatim from `AGENTS.md` / `CLAUDE.md` (2026-11-25); the short core there still binds. A section
name quoted below is a section of `AGENTS.md` unless it says otherwise.

## The nodynamic tag

- Every go command that reaches internal/jxl (the app, its pipeline, tests/, tools/ocrlab) needs the
  `nodynamic` build tag, or `internal/jxl/nodynamic_guard.go` stops the compile (ticket 115, see Pitfalls).
  The scripts set it through `scripts/lib/gobuildtags.ps1` (GOFLAGS), CI through release.yml's `env`,
  golangci-lint through its config and VS Code through `go.buildTags`. By hand:
  `$env:GOFLAGS = '-tags=nodynamic'` once per shell, or `go test -tags nodynamic ./tests/`.

## Pitfalls

- scripts/build.ps1 and scripts/build-ui.ps1 copy artifacts to C:/GD/tc/SZA/_APP. This is environment-specific and may fail on other machines.
- Build scripts embed Windows resources with goversioninfo pinned in `scripts/lib/goversioninfo.ps1` (`go run module@v1.7.0`), never the copy on PATH: `IconPath` lists three ICOs (mark, verb, document type) whose resource indexes internal/windowsreg writes into the registry, anything older than v1.5.0 fails opening the comma-joined path, and the shared `%USERPROFILE%\go\bin` copy gets reinstalled at v1.4.1 by another repo on this machine.
- System-surface icons (the program ICO and its favicon copies, `assets/convert-verb.ico`, `assets/document-type.ico`, `extension/icons/*`, the MSIX assets) are render targets of internal/iconart: redraw them with `scripts/generate-icon.ps1` (`go run ./tools/icongen`), never by hand - tests/icons_test.go compares every pixel.
- MOBI/AZW3 conversion depends on Calibre at runtime; CBR/CB7 comics depend on 7-Zip at runtime (CBZ/CBT need nothing).
- JPEG XL goes through `github.com/gen2brain/jpegxl`, whose package init, unless built with `nodynamic`, loads
  `libjxl.dll` by bare name - so a planted DLL in the current directory or on PATH would run inside every start
  of the app (ticket 115). Hence the tag on every go command (Build, Test, Lint above) and the guard file that
  refuses to compile without it; `TestNoNativeDecoder` proves the tag still disables the native path after a
  module update. Never drop the tag to make a build pass, and never add a native library load:
  `scripts/security-posture.ps1` holds the app's own code to the Windows system DLLs it lists.

## This machine (from CLAUDE.md)

- **An ad-hoc go command needs `-tags nodynamic`** (or `$env:GOFLAGS = '-tags=nodynamic'` in the same call) once
  it reaches `internal/jxl`; without it the compile stops on the guard (ticket 115, AGENTS.md "Pitfalls").
- **Line endings are settled, don't re-litigate them:** `.gitattributes` pins `*.go text eol=lf`, so `gofmt`
  and the gofmt linter agree with the committed source. `gofmt -l .` comes back clean apart from throwaway
  files under the gitignored `temp/`. The old "normalize to LF first" dance is obsolete.
- `scripts/build.ps1` / `scripts/build-ui.ps1` copy artifacts to `C:/GD/tc/SZA/_APP` and need `goversioninfo`;
  this is environment-specific and may fail on other machines.
