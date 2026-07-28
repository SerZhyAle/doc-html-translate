# AGENTS

Purpose: help coding agents become productive in this repository quickly.

## SZA Unified Rules (canon)

This repo follows the portfolio-wide **SZA Unified Rules** by *reference* (not a mirror - there is
nothing to keep in sync here). Canon / source of truth:
the **`sza` Claude Code plugin**, from `github.com/SerZhyAle/sza-unified-rules` - read its `rules/README.md`
for the universal core
(repository layout, documentation, development, testing, release & distribution, localization,
security & privacy). This file keeps only the **deltas and repo specifics** below; it does not restate
the universal rules. The full evidence-backed delta record for this project is
`<canon>/contrib/epub_2_html.md`.

This repo's overlay shape (four overlay facts):

- **Overlay A distribution (GitHub + winget + Microsoft Store) on an Overlay C Go source body**
  (`cmd/` + `internal/`, module `doc-html-translate`), plus a second, code-independent JS **extension
  edition** (Chrome Web Store + Edge Add-ons) under `extension/`. The two editions share no code and are
  kept in sync by [docs/PARITY.md](docs/PARITY.md) + gates (see "Cross-Edition Parity").
- **No `publishing/` umbrella** - channels are top-level siblings: `winget/`, `msix/`, `installer/`,
  `tools/store/`, `extension/`. Internal docs live under the **`DEV/`** umbrella, not `docs/`; the
  changelog is the engineering ledger `DEV/CHANGELOG.md`, not a root Keep-a-Changelog.
- **Version shape `YY.MMDD.HHmm`** (MMDD zero-padded, a per-project frozen choice), stamped via
  `-ldflags "-X main.Version=.."`; the extension edition versions on its own clock.
- **Release is up to 5 independent one-way ops**, each its own trigger and cost tag - see
  [DEV/RELEASE.md](DEV/RELEASE.md) and `<canon>/CHANNEL_MATRIX.md`.
- **Frozen anchors** (reserve once, never change): winget `SerZhyAle.DocHtmlTranslate`, Go module
  `doc-html-translate`, MSIX Identity `SZA.Doc-HTML-Translate`, Inno AppId
  `{E8B4F1C7-2A9D-4E63-9F1B-7C3A5D8E2B04}`, and distinct Chrome / Edge store ids (never a shared or
  pinned manifest key).

## Project Snapshot

- Language: Go (module: doc-html-translate, go 1.25)
- Primary target: Windows desktop usage (CLI + GUI launcher)
- Main binaries:
  - cmd/doc-html-translate (CLI)
  - cmd/doc-html-ui (GUI wrapper for CLI)

## Start Here

1. Read README.md for product behavior and user-facing flags.
2. Prefer script-based workflows from scripts/ instead of ad-hoc commands.
3. For conversion logic changes, start in internal/pipeline/pipeline.go and trace into format-specific packages.

## Build vs Release (READ THIS)

Two distinct flows - see [DEV/RELEASE.md](DEV/RELEASE.md):

- **Build ("сборка")** = local and FREE. `./scripts/build-local.ps1 -Message "..."` (gate + build
  CLI + build UI + commit). Never touches GitHub/CI. This is the default for any "build"/"сборка" ask.
- **Release ("релиз")** = published and PAID. `./scripts/release.ps1` prints the full checklist and
  runs nothing; tags (`v*`, `ext-cws-v*`, `ext-edge-v*`) trigger paid CI. Only do this for an explicit "release"/"релиз".

Never push a tag, submit to winget, upload to the Store, or publish the extension unless the request
is explicitly a release.

## Build, Test, Lint

Run from repository root in PowerShell.

- Local build + commit (the "сборка" flow): ./scripts/build-local.ps1 -Message "..."
- Build CLI only: ./scripts/build.ps1
- Build UI only: ./scripts/build-ui.ps1
- Build universal installer (setup.exe, x86+x64, per-user, local/free): ./scripts/build-installer.ps1 (needs Inno Setup / ISCC)
- Test: ./scripts/test.ps1
- Lint: ./scripts/lint.ps1
- Full local checks: ./scripts/check.ps1
- Release checklist (prints only, runs nothing): ./scripts/release.ps1

Tool bootstrap (when missing):

- ./scripts/bootstrap-tools.ps1

Notes:

- scripts/lint.ps1 expects golangci-lint.
- scripts/typo.ps1 expects typos (typos-cli via cargo).

## Architecture Map

- Entry points:
  - cmd/doc-html-translate/main.go
  - cmd/doc-html-ui/main.go
- App wiring:
  - internal/app/app.go
  - internal/config/flags.go
- Pipeline orchestration:
  - internal/pipeline/pipeline.go
- Format extractors:
  - internal/epub, internal/pdf, internal/mobi, internal/fb2, internal/rtf, internal/txt, internal/md, internal/htmlconv
  - internal/img: standalone image input (PNG/JPG/WebP/..) - wraps the picture in a one-page HTML doc; the pipeline forces the OCR overlay so the image gets translatable text plates (see internal/ocr)
  - internal/comic: comic archives (CBZ/CBR/CB7/CBT) - one page image per spine entry in natural filename order, forced OCR (same rationale as internal/img). CBZ=zip, CBT=tar (stdlib); CBR/CB7 shell out to 7-Zip (LookPath + probe paths, the MOBI/Calibre precedent)
- HTML processing/generation:
  - internal/htmlproc, internal/htmlsplit, internal/htmlgen
- Translation:
  - internal/translator
- Interface language (13 languages, `en ru uk de it es fr pt ar hi bn ur zh`):
  - internal/i18n: `Codes` is the list, `Add()` takes 12 translations or panics, `Resolve()` is the
    order (explicit `-ui-lang` -> saved -> OS -> English). `app.New` calls `SetLanguage` once, so
    `i18n.S()` works process-wide instead of threading a language through htmlgen's signatures.
  - internal/app/splash/*.txt: the console splash, one embedded file per language.
  - cmd/doc-html-ui/i18n.js (GUI dictionary), extension/_locales/<code>/messages.json (extension).

## Cross-Edition Parity (READ BEFORE ADDING FEATURES)

The app ships as two independent codebases that do **not** share code: the Go desktop app (CLI / GUI /
MSIX) and the JavaScript browser extension (`extension/src`). Logic is ported from Go to JS by hand, so
shared constants and heuristics drift unless they are pinned. **[docs/PARITY.md](docs/PARITY.md) is the
single source of truth** for the port map (which Go file maps to which JS file), the shared invariants
that must stay identical (theme palette, PDF reflow constants, EPUB TOC rules, OCR download host /
catalog / class names, settings defaults), and the intentional divergences (do not "fix" them).

Process: a user-facing feature is **one cross-edition ticket** (template
[DEV/plan/_TEMPLATE_cross-edition.md](DEV/plan/_TEMPLATE_cross-edition.md)) covering every edition -
each edition is either implemented or explicitly declined with a rationale. Do not open a separate
ticket per edition. When you touch a shared invariant, update docs/PARITY.md in the same change. Open
gaps are tracked in [DEV/plan/2026-07-01_cross-edition-parity.md](DEV/plan/2026-07-01_cross-edition-parity.md).

## Conventions And Behavior To Preserve

- Script-first dev flow: prefer existing scripts in scripts/ for routine tasks.
- Idempotent output reuse: if output index exists, pipeline reuses it unless -force is set.
- Default CLI with no args enters registration flow (not conversion).
- Translation is optional; default run is convert + open without translation engine unless -google or -ollama is passed.
- Paid engines respect -max-cost: the estimate (chars/1e6*$20) is enforced as a pre-flight guard in internal/pipeline/pipeline.go before any request is sent.
- The output HTML carries a client-side reader layer injected by internal/htmlgen (navbar.go readerScript/readerCSS on chapter pages, plus a matching toolbar/script on index.html). Keep two storage scopes distinct: zoom uses sessionStorage; reading themes and reading position use localStorage (must survive sessions). Reading position is namespaced by bookStorageKey, which must be computed identically on chapter pages and index.html.
- Windows and non-Windows behavior is split via *_windows.go and *_nonwindows.go files in several packages.
- The interface language dresses the **chrome only**. A converted page keeps the *document's* `<html lang>`; the navbar and reader controls carry their own `lang`/`dir` and mirror for `ar`/`ur`. Putting the UI language on `<html lang>` stops Chrome offering "Translate page" - the product's entire free workflow. Guarded by `TestConvertedChromeLanguage` in tests/smoke_test.go and by the RTL assertions in tools/store/make-screenshot.ps1.
- House typography (short hyphens, Russian ё, ".." not "...") applies to `en`, `ru` and `uk` only; the other ten follow their own script and are exempt - tests/typography_test.go scopes the check by language.
- Optional OCR overlay (-ocr): internal/ocr shells out to the external Tesseract binary (parses TSV for bboxes) and rewrites document images into positioned, translatable text plates. It runs in internal/pipeline/pipeline.go after nav injection and before translation (so overlay text is translated too), and is strictly best-effort - a missing tesseract or a failed image never aborts the conversion. English data ships in <exe>/tessdata; other languages download on demand (-ocr-download / GUI). Applies to EPUB and PDF (formats whose images exist at HTML stage).

## Pitfalls

- scripts/build.ps1 and scripts/build-ui.ps1 copy artifacts to C:/GD/tc/SZA/_APP. This is environment-specific and may fail on other machines.
- Build scripts rely on goversioninfo for Windows resource embedding.
- MOBI/AZW3 conversion depends on Calibre at runtime; CBR/CB7 comics depend on 7-Zip at runtime (CBZ/CBT need nothing).

## Editing Guidance

- Keep changes minimal and package-scoped.
- Add or update tests in the same internal package when behavior changes.
- Avoid changing public CLI flag semantics unless explicitly requested.

## Existing Docs (Link, Do Not Duplicate)

- README.md
- DEV/README.md
- docs/PARITY.md (cross-edition invariants + port map - read before adding features)
- docs/how-i-posted-this-project-to-winget.md
