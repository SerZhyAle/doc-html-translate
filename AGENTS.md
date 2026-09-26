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
  <!-- canon-ok: names this repo's ledger shape (stamp ledgerShape 2) as a delta from the canon default,
       rather than re-authoring the changelog rule - the rule itself stays in DOCUMENTATION_CONCEPT.md. -->
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

- **Build ("сборка")** = local and FREE. `./scripts/build-local.ps1 -Message "<commit message>"` (build
  CLI + build UI, then the gate over that tree, then commit). Never touches GitHub/CI. This is the default for any "build"/"сборка" ask.
- **Release ("релиз")** = published and PAID. `./scripts/release.ps1` prints the full checklist and
  runs nothing; tags (`v*`, `ext-cws-v*`, `ext-edge-v*`) trigger paid CI. Only do this for an explicit "release"/"релиз".

Never push a tag, submit to winget, upload to the Store, or publish the extension unless the request
is explicitly a release.

## Build, Test, Lint

Run from repository root in PowerShell.

- Local build + commit (the "сборка" flow): ./scripts/build-local.ps1 -Message "<commit message>"
- Build CLI only: ./scripts/build.ps1
- Build UI only: ./scripts/build-ui.ps1
- Build universal installer (setup.exe, x86+x64, per-user, local/free): ./scripts/build-installer.ps1 (needs Inno Setup / ISCC)
- Test: ./scripts/test.ps1 (Go edition) and ./scripts/test-extension.ps1 (extension edition, node --test)
- Lint: ./scripts/lint.ps1
- Full local checks: ./scripts/check.ps1
- Every check speaks `CHECK-VERDICT` (pointer: docs/contracts/CHECK-VERDICT.md): exit `0` PASS, `1` FAIL,
  `2` COULD NOT VERIFY (a missing tool, input or git result - **never a pass**), `3` PASS WITH ADVISORIES,
  and the **last line** is the verdict (`check: PASS`, `test: COULD NOT VERIFY (1)`, ..). Quote that line
  as evidence, not the scrollback above it. A fresh clone without `test_doc/` ends in COULD NOT VERIFY
  by design. Which runner owns each check: `configs/check-placement.jsonl` (a new check gets a record).
- `check.ps1` writes `temp/logs/gate-evidence.json` (verdict, plan, every child's verdict, and the tree
  hash taken before and after the run - an edit during the run voids it); `release.ps1` blocks the tag
  step unless it is the full default plan, every child passing, on HEAD's tree (only `DEV/COMMIT_LOG.md`,
  appended by build-local after its commit, may differ). A `-Plan` subset never unlocks it.
- `scripts/commit-push.ps1` (`a c`) refuses to push `main` without `-PublishSite`: Pages serves the site
  from `main`, so that push publishes it, ungated.
- `check.ps1` also holds two whole-tree checks. `scripts/doc-registry.ps1`: every maintained document is a
  record of `docs/DOCUMENT_REGISTRY.jsonl` (record shape 1, declared in the stamp), every `.md` / `.html`
  file is claimed by a record or by `configs/doc-registry-exclusions.jsonl`, every announced page carries
  the SEO block, and `sitemap.xml` is generated - after adding or renaming a site page, run it with
  `-Generate`, never edit the sitemap. `scripts/security-posture.ps1`: the permission and network-surface
  inventories in `docs/security-posture.json` against `extension/manifest.json`, `msix/AppxManifest.xml`,
  the code and the dependency set; the privacy pages, `extension/store/PRIVACY.md`, the permission
  justifications in `extension/store/LISTING.md` and the runFullTrust text in `msix/README.md` are
  rendered from those rows - edit the rows, then run it with `-Render`. A new permission, network call
  or telemetry-shaped dependency fails it until a row covers it.
- "What must I read before touching this?": `./scripts/doc-query.ps1 -Area <area> -Trigger <trigger>`
  (`-Path <file>` for one file's record, `-List` for the vocabularies). Register a document before
  anything links to it.
- Release checklist (prints only, runs nothing but read-only checks): ./scripts/release.ps1. Its header
  carries two verdicts the tag step takes as input: the gate evidence, and the contract gate
  (`scripts/contract-gate.ps1`: the pointers in `docs/contracts/` against the catalog registry -
  PASS / WARN pass, FAIL / UNVERIFIED block, and an unreachable catalog is UNVERIFIED, never PASS).
- Sliced code audit: ./scripts/audit-slices.ps1 cuts the audit scope (shipped code + release path) into
  slices a session can read line by line (`-Write` writes the manifest into the audit ticket's folder,
  `-Tail` slices files added since, `-Summary` measures the campaign and exits 0 only when it is closed).
  Hand-run; ticket 34 is its first campaign.
- OCR visual-fidelity lab: go run ./tools/ocrlab verify | run | score | report (see tools/ocrlab/README.md)

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
  - internal/assets: local asset copying shared by internal/htmlconv and internal/md (img src/srcset, picture sources, CSS url() and local stylesheets); names unique case-insensitively, generated names reserved, nothing resolved outside the source folder is copied
  - EPUB chapters are normalized on the parsed tree (internal/epub normalize.go, links.go `rewriteLinks`), never by text replacement; the splitter retargets TOC entries and links through an id-to-part map
- External helpers (pdftotext, Tesseract, Calibre, 7-Zip, ffmpeg/ImageMagick):
  - internal/procrun: the one way to run a helper - per-tool size-scaled deadline (`DOCHT_TOOL_TIMEOUT_SCALE` multiplies it), process-tree kill (job object on Windows, process group elsewhere), capped output, one error type naming the tool. Every call takes the run's context, so Ctrl+C stops a helper at once and the run ends as interrupted. Do not call exec.Command for a helper directly.
  - internal/bundledtools: the Windows-only bundled pdftotext, unpacked into a content-hash-named cache folder. Nothing is ever installed on the user's machine (tests/no_auto_install_test.go).
- Translation:
  - internal/translator
- Interface language (13 languages, `en ru uk de it es fr pt ar hi bn ur zh`):
  - internal/i18n: `Codes` is the list, `Add()` takes 12 translations or panics, `Resolve()` is the
    order (explicit `-ui-lang` -> saved -> OS -> English). `app.New` calls `SetLanguage` once, so
    `i18n.S()` works process-wide instead of threading a language through htmlgen's signatures.
  - internal/app/splash/*.txt: the console splash, one embedded file per language.
  - cmd/doc-html-ui/i18n.js (GUI dictionary), extension/_locales/<code>/messages.json (extension).
- Developer tooling, not shipped:
  - tools/ocrlab: the OCR visual-fidelity lab - a licence-gated corpus manifest, engine-independent
    ground truth, the eight quality measures from the spec's table, and a runner that converts each
    scene through the real pipeline and records what headless Chrome actually laid out. Not built by
    scripts/build.ps1 and in no package. Its one hook into shipped code is the opt-in `DOCHT_OCR_DIAG`
    sidecar in internal/ocr (off unless set; an output-identity test proves it changes nothing).
    See tools/ocrlab/README.md and DEV/plan/16_2026-08-11_ocr-visual-fidelity-lab/.

## Cross-Edition Parity (READ BEFORE ADDING FEATURES)

The app ships as two independent codebases that do **not** share code: the Go desktop app (CLI / GUI /
MSIX) and the JavaScript browser extension (`extension/src`). Logic is ported from Go to JS by hand, so
shared constants and heuristics drift unless they are pinned. **[docs/PARITY.md](docs/PARITY.md) is the
single source of truth** for the port map (which Go file maps to which JS file), the shared invariants
that must stay identical (theme palette, PDF reflow constants, EPUB TOC rules, OCR download host /
catalog / class names, settings defaults), and the intentional divergences (do not "fix" them).

Process: a user-facing feature is **one cross-edition ticket** covering every edition - each edition is
either implemented or explicitly declined with a rationale in that same ticket. Do not open a separate
ticket per edition. When you touch a shared invariant, update docs/PARITY.md in the same change. Open
gaps ride in [DEV/plan/RELEASE_QUEUE.md](DEV/plan/RELEASE_QUEUE.md) like any other work; there is no
separate parity backlog and no ticket template file - copy the shape from the newest ticket in
[DEV/plan/](DEV/plan/).

## Conventions And Behavior To Preserve

- Script-first dev flow: prefer existing scripts in scripts/ for routine tasks.
- Idempotent output reuse: an output is reopened only when its completion record (internal/outputpath completion.go, written as the run's last step) matches the source identity and the result-affecting options; anything else is rebuilt with a logged reason, and -force always rebuilds. Pipeline page writes go through internal/fsutil (temp file + rename), never an in-place truncating write.
- Default CLI with no args enters registration flow (not conversion).
- Translation is optional; default run is convert + open without translation engine unless -google or -ollama is passed.
- Paid engines respect -max-cost: the estimate (characters, not bytes, over pages + title + TOC labels, /1e6*$20) is enforced as a pre-flight guard in internal/pipeline/cost.go before any request is sent. A set limit pre-approves spending up to it (no dialog); without one the dialog asks above 1000 characters, defaulting to "do not spend" (OK/Cancel, Cancel default; under the GUI the question travels over the child's stdio - internal/dialog HostStdio - and is asked in the GUI's window). The Google key travels in the X-Goog-Api-Key header, never the URL.
- The output HTML carries a client-side reader layer injected by internal/htmlgen (navbar.go readerScript/readerCSS on chapter pages, plus a matching toolbar/script on index.html). Keep two storage scopes distinct: zoom uses sessionStorage; reading themes and reading position use localStorage (must survive sessions). Reading position is namespaced by `epub.Book.ReaderKey` (htmlgen.ReaderKey: source file name + size, original title, page count), set once in the pipeline before translation and read by every page and index.html - never recompute it from the (translated) title. A saved position is restored only when the URL has no fragment. Values enter generated scripts only through `jsString`, paths enter hrefs only through `epub.URLPath`.
- Windows and non-Windows behavior is split via *_windows.go and *_nonwindows.go files in several packages.
- The interface language dresses the **chrome only**. A converted page keeps the *document's* `<html lang>`; the navbar and reader controls carry their own `lang`/`dir` and mirror for `ar`/`ur`. Putting the UI language on `<html lang>` stops Chrome offering "Translate page" - the product's entire free workflow. Guarded by `TestConvertedChromeLanguage` in tests/smoke_test.go and by the RTL assertions in tools/store/make-screenshot.ps1.
- The canon's house text style (DOCUMENTATION_CONCEPT.md §5) is **scoped by language here**: it binds `en`, `ru` and `uk` only. The other ten interface languages follow their own script and are exempt - tests/typography_test.go enforces that scoping, so do not widen the check to all thirteen.
- Optional OCR overlay (-ocr): internal/ocr shells out to the external Tesseract binary (parses TSV for bboxes) and rewrites document images into positioned, translatable text plates. It runs in internal/pipeline/pipeline.go after nav injection and before translation (so overlay text is translated too), and is strictly best-effort - a missing tesseract or a failed image never aborts the conversion. English data ships in <exe>/tessdata; other languages download on demand (-ocr-download / GUI) into the per-user folder (os.UserCacheDir()/doc-html-translate/tessdata), catalogue codes only, verified against pinned SHA-256 digests (internal/ocr/download.go). Applies to EPUB and PDF (formats whose images exist at HTML stage).

## Pitfalls

- scripts/build.ps1 and scripts/build-ui.ps1 copy artifacts to C:/GD/tc/SZA/_APP. This is environment-specific and may fail on other machines.
- Build scripts embed Windows resources with goversioninfo pinned in `scripts/lib/goversioninfo.ps1` (`go run module@v1.7.0`), never the copy on PATH: `IconPath` lists three ICOs (mark, verb, document type) whose resource indexes internal/windowsreg writes into the registry, anything older than v1.5.0 fails opening the comma-joined path, and the shared `%USERPROFILE%\go\bin` copy gets reinstalled at v1.4.1 by another repo on this machine.
- System-surface icons (the program ICO and its favicon copies, `assets/convert-verb.ico`, `assets/document-type.ico`, `extension/icons/*`, the MSIX assets) are render targets of internal/iconart: redraw them with `scripts/generate-icon.ps1` (`go run ./tools/icongen`), never by hand - tests/icons_test.go compares every pixel.
- MOBI/AZW3 conversion depends on Calibre at runtime; CBR/CB7 comics depend on 7-Zip at runtime (CBZ/CBT need nothing).

## Editing Guidance

- Keep changes minimal and package-scoped.
- Add or update tests in the same internal package when behavior changes.
- Avoid changing public CLI flag semantics unless explicitly requested.

## Existing Docs (Link, Do Not Duplicate)

- README.md
- DEV/README.md
- docs/README.md (index of the docs/ tree)
- docs/PARITY.md (cross-edition invariants + port map - read before adding features)
- docs/DOCUMENT_REGISTRY.jsonl (every maintained document, its product areas and change triggers; queried by scripts/doc-query.ps1)
- docs/security-posture.json (what each edition can touch and send; docs/SECURITY_POSTURE.md is its readable render)
- docs/how-i-posted-this-project-to-winget.md
- **The shared contracts catalog is at `P:/Contracts` on this machine** - this is the only tracked file
  in the repository allowed to name that path. Anything a second product builds against lives there,
  organized by function (`ocr-overlay/`, `install-trust/`, ..), never in this repo: this product owns
  `OCR-PIPELINE` and `OCR-INVOCATION`, implements `OCR-OVERLAY` as its reference implementation, and is
  bound by `INSTALL-TRUST`. Read `_meta/RULES.md`, `_meta/VERSIONING.md` and `_meta/REGISTRY.md` before
  touching any of them. Rules that bind work here: **edit the contract in the catalog, never a copy**;
  the contract changes **before** the code; a breaking change is a new dated section plus a version bump,
  never an in-place rewrite; a deviation is either an amendment or a dated exception in the registry,
  never silence. In this repo the contracts appear as pointer files under `docs/contracts/`, and code and
  docs **cite them by id** - `OCR-OVERLAY rule 8`, `OCR-PIPELINE.md section 5` - never by path.
- DEV/plan/ (public, collaborative specifications and their tactical plans; `RELEASE_QUEUE.md` says what is left before the next release, where the queue wins on order and the ticket wins on status)
- CLAUDE.md ("Spec / plan tickets") - the ticket store's file names, done-set and package numbering
