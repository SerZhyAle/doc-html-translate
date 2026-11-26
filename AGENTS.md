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

- Language: Go (module: doc-html-translate, go 1.26)
- Primary target: Windows desktop usage (CLI + GUI launcher)
- Main binaries:
  - cmd/doc-html-translate (CLI)
  - cmd/doc-html-ui (GUI wrapper for CLI)

## Start Here

1. Read README.md for product behavior and user-facing flags.
2. Prefer script-based workflows from scripts/ instead of ad-hoc commands.
3. For conversion logic changes, start in internal/pipeline/pipeline.go and trace into format-specific packages.
4. Scripts and agents drive the product by its command line: read [docs/CLI.md](docs/CLI.md), and `doc-html-translate.exe commands --json` lists every command as data (APP-COMMANDS rule 20). The manifest is `docs/cli-commands.json`; `docs/CLI.md` is its render (`go run ./tools/clidoc -render`, held by `scripts/cli-doc.ps1`), never edited by hand.

## Read Before Touching (rule files)

Detail lives in `.claude/rules/`. Claude Code loads a file by its `paths:`; any other agent (Codex) opens it by
hand **before** touching the area.

| Touching | Read first (`.claude/rules/`) |
| --- | --- |
| `scripts/`, `configs/`, `tests/`, gate evidence, audit slices | [checks-and-gates](.claude/rules/checks-and-gates.md) |
| site pages, `docs/`, sitemap, security posture, release notes | [site-and-docs-gates](.claude/rules/site-and-docs-gates.md) |
| `tools/store/`, `msix/`, `winget/`, release checklist, promotion | [store-and-release](.claude/rules/store-and-release.md) |
| `cmd/doc-html-ui/`, help, settings inventory, UI contracts | [gui-and-help](.claude/rules/gui-and-help.md) |
| `internal/`: run order, helpers, reuse, cost guard, reader | [conversion-pipeline](.claude/rules/conversion-pipeline.md) |
| OCR overlay, plates, `tools/ocrlab` | [ocr](.claude/rules/ocr.md) |
| interface languages, text-style scope | [i18n](.claude/rules/i18n.md) |
| build scripts, `nodynamic`, version resources, icons | [build-and-native](.claude/rules/build-and-native.md) |
| tickets, `DEV/plan/`, the queue | [tickets](.claude/rules/tickets.md) |
| `DEV/research/` | [research-notes](.claude/rules/research-notes.md) |
| `.claude/commands/`, `.claude/agents/` | [agent-commands](.claude/rules/agent-commands.md) |

## Build vs Release (READ THIS)

Two distinct flows - see [DEV/RELEASE.md](DEV/RELEASE.md):

- **Build ("сборка")** = local and FREE. `./scripts/build-local.ps1 -Message "<commit message>"` (build
  CLI + build UI, then the fast gate tier over that tree, then commit). Never touches GitHub/CI. This is the default for any "build"/"сборка" ask.
  The full gate runs only with `-Release` - the release pre-flight and the release's docs commit.
- **Release ("релиз")** = published and PAID. `./scripts/release.ps1` prints the full checklist and
  runs nothing; tags (`v*`, `ext-cws-v*`, `ext-edge-v*`) trigger paid CI. Only do this for an explicit "release"/"релиз".

Never push a tag, submit to winget, upload to the Store, or publish the extension unless the request
is explicitly a release.

## Build, Test, Lint

Run from repository root in PowerShell.

**Between releases nothing above 5 seconds runs** (canon `TESTING_AND_QA` section 2). No full build, no full
suite, no check whose recorded time is above 5 s - unless the owner names that run, or it is the release.
A technical piece, a point fix or a small improvement is checked by the fast tier, a direct call of what
changed and static inspection; a slower rung skipped for this reason goes into the ticket's audit as a
`RELEASE` item naming the command the release runs, and the report says "not built, not tested until the release".

- **Fast command (between releases):** `./scripts/check.ps1 -Fast`; for the touched area
  `./scripts/check.ps1 -Fast -Plan scripts/<child>.ps1,..`. Go: `go test -tags nodynamic ./internal/<pkg>/ -run <Test>`
  or `go vet -tags nodynamic ./internal/<pkg>/`; extension: `node --test extension/test/<file>.test.mjs`.
- **Release command (pre-flight only):** `./scripts/build-local.ps1 -Release -Message ".."`.
- Each check's measured time is in `configs/check-placement.jsonl`. Release tier, not run between releases:
  `scripts/test.ps1`, `scripts/test-extension.ps1`, `./scripts/check.ps1` with no switch.
- One build: `./scripts/build.ps1` (CLI), `./scripts/build-ui.ps1` (UI), `./scripts/build-installer.ps1`
  (setup.exe). Lint: `./scripts/lint.ps1`.
- **Every go command that reaches `internal/jxl` needs `-tags nodynamic`** (or `$env:GOFLAGS = '-tags=nodynamic'`
  once per shell): without it the JPEG XL dependency would load a planted `libjxl.dll`, and a guard file stops the
  compile. Never drop the tag, never add a native library load.
- Check verdicts (`CHECK-VERDICT`): exit `0` PASS, `1` FAIL, `2` COULD NOT VERIFY (**never a pass**), `3` PASS WITH
  ADVISORIES; quote the **last line** as evidence.
- `scripts/commit-push.ps1` (`a c`) refuses to push `main` without `-PublishSite`: Pages serves the site
  from `main`, so that push publishes it, ungated.
- "What must I read before touching this?": `./scripts/doc-query.ps1 -Area <area> -Trigger <trigger>`
  (`-Path <file>` for one file's record, `-List` for the vocabularies). Register a document before
  anything links to it.
- Release-only: `node tools/helpgen/generate.mjs` and `node tools/uicheck/run.mjs` (GUI build, minutes); a ticket
  that needs one records a `RELEASE` item.
- No agent runs `msstore` in a preflight or between releases; the one machine write into the Store is the release's
  `./tools/store/submit-submission.ps1`. Every outward promotion step waits for the owner's go.
- Store listing: `tools/store/listingData.csv` is the **authoritative listing file**; `msix/out/store-import/` is its gitignored copy, never a second source.
- **Render targets, never edited by hand** (source and render command: the rule files): `docs/CLI.md`,
  `sitemap.xml`, `release-notes.html`, the privacy pages and other security-posture texts,
  `docs/settings-inventory.json`, `help/` and `cmd/doc-html-ui/help/`, the system-surface icons.

## Architecture Map

The map only; the behaviour behind each entry is in the rule files above.

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
- internal/img: standalone image input, a one-page HTML doc with forced OCR (see internal/ocr)
- internal/comic: comic archives (CBZ/CBR/CB7/CBT), one page per spine entry, forced OCR
- HTML processing/generation: internal/htmlproc, internal/htmlsplit, internal/htmlgen, internal/assets; EPUB chapters
  are normalized on the parsed tree, never by text replacement
- External helpers (pdftotext, Tesseract, Calibre, 7-Zip, ffmpeg/ImageMagick, FileDO):
  - internal/procrun: the one way to run a helper - never call exec.Command for a helper directly
  - internal/readiness: what this machine can do, as data (the window pre-flight, `input check`, `doctor`)
  - internal/bundledtools: the Windows-only bundled pdftotext; nothing is ever installed on the user's machine
  - internal/fdsec: FileDO secret files (.fd-sec); the password only in the child's environment, FileDO's output
    never forwarded; a new GUI file named `*secret*` is gitignored - name it otherwise
- Translation:
  - internal/translator
- Interface language (13 languages, `en ru uk de it es fr pt ar hi bn ur zh`): internal/i18n
- GUI window structure: `cmd/doc-html-ui/ui.html` - token block, components, seven pages; no hard-coded px size
- In-app help: `cmd/doc-html-ui/help/help-map.json` is the help map; `tools/helpgen` renders the pages
- Developer tooling, not shipped: tools/ocrlab

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
- Output is reused only when its completion record matches the source identity and the result-affecting options;
  `-force` rebuilds. Pipeline page writes go through internal/fsutil, never an in-place truncating write.
- Default CLI with no args enters registration flow (not conversion).
- Translation is optional; default run is convert + open without translation engine unless -google or -ollama is passed.
- Paid engines respect `-max-cost`, enforced before any request; without a limit the dialog asks above 1000 characters,
  defaulting to "do not spend". The Google key travels in the X-Goog-Api-Key header, never the URL.
- Reader layer: fit-width uses sessionStorage; reading themes and reading position use localStorage (must survive
  sessions), keyed by `epub.Book.ReaderKey`, never the translated title. Values enter generated scripts only through
  `jsString`, paths enter hrefs only through `epub.URLPath`.
- Windows and non-Windows behavior is split via *_windows.go and *_nonwindows.go files in several packages.
- The interface language dresses the **chrome only**: a converted page keeps the document's `<html lang>`, or Chrome
  stops offering "Translate page" - the free workflow.
- The house text style binds `en`, `ru` and `uk` only; do not widen the check to the other ten languages.
- A user-visible change in the GUI launcher changes its help in the same edit (CONTEXT-HELP rule 18).
- Optional OCR overlay (`-ocr`) is strictly best-effort: a missing tesseract or a failed image never aborts the
  conversion; language downloads are catalogue codes only, verified against pinned SHA-256.

## Pitfalls

- scripts/build.ps1 and scripts/build-ui.ps1 copy artifacts to C:/GD/tc/SZA/_APP. This is environment-specific and may fail on other machines.
- Build scripts run the goversioninfo pinned in `scripts/lib/goversioninfo.ps1`, never the copy on PATH; the
  resource indexes of its three ICOs are what internal/windowsreg writes into the registry.
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
  `OCR-INVOCATION`, `DIAGNOSTIC-REPORT` and `REPO-STAMP`, implements `OCR-OVERLAY` as its reference
  implementation, is bound by `INSTALL-TRUST`, and consumes the rest - the full pointer roster (id,
  version, this repo's role per contract) is [docs/contracts/README.md](docs/contracts/README.md).
  `OCR-ACCURACY` and `OCR-EXCHANGE` are records owned by other products that cite this one: a change
  here that moves a shared constant is announced in their rows, never applied to their documents. Read `_meta/RULES.md`, `_meta/VERSIONING.md` and `_meta/REGISTRY.md` before
  touching any of them. Rules that bind work here: **edit the contract in the catalog, never a copy**;
  the contract changes **before** the code; a breaking change is a new dated section plus a version bump,
  never an in-place rewrite; a deviation is either an amendment or a dated exception in the registry,
  never silence. In this repo the contracts appear as pointer files under `docs/contracts/`, and code and
  docs **cite them by id** - `OCR-OVERLAY rule 8`, `OCR-PIPELINE.md section 5` - never by path.
- DEV/plan/ (public, collaborative specifications and their tactical plans; `RELEASE_QUEUE.md` says what is left before the next release, where the queue wins on order and the ticket wins on status)
- CLAUDE.md ("Spec / plan tickets") - the ticket store's file names, done-set and package numbering
