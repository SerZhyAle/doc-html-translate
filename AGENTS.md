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

- **Fast command (between releases):** `./scripts/check.ps1 -Fast` - the gate children recorded
  `"tier": "fast"` in `configs/check-placement.jsonl`, all at once (about 5 s warm; `a fc`). For the touched
  area name its children: `./scripts/check.ps1 -Fast -Plan scripts/help-check.ps1,scripts/settings-inventory.ps1`
  (a release-tier child is refused, exit 2). Go code: `go test -tags nodynamic ./internal/<pkg>/ -run <Test>`
  of the touched package, or `go vet -tags nodynamic ./internal/<pkg>/`; extension code:
  `node --test extension/test/<file>.test.mjs` - each a direct call whose run takes 5 s or less.
- **Release command (the release pre-flight only):** `./scripts/build-local.ps1 -Release -Message ".."` - both
  builds, then the full gate (`./scripts/check.ps1` with no switch), whose evidence `release.ps1` requires.
- **Measured list:** each gate record's `tier`, `seconds` and `timed` (date and how it was measured) in
  `configs/check-placement.jsonl`, held by `tests/placement_test.go` (a fast record carries at most 5 s).
  Release tier on 2026-10-29: `scripts/test.ps1` (212 s) and `scripts/test-extension.ps1` (20 s); every other
  gate child is fast (0.6 - 4.9 s). `check.ps1` prints `<child> took N s` for every child, so each run is a new record.
- Local build + commit (the "сборка" flow): ./scripts/build-local.ps1 -Message "<commit message>" (fast tier; `-Release` for the full gate)
- Build CLI only: ./scripts/build.ps1
- Build UI only: ./scripts/build-ui.ps1
- Build universal installer (setup.exe, x86+x64, per-user, local/free): ./scripts/build-installer.ps1 (needs Inno Setup / ISCC)
- Test (release tier): ./scripts/test.ps1 (Go edition) and ./scripts/test-extension.ps1 (extension edition, node --test)
- Lint: ./scripts/lint.ps1
- Full local checks (release tier): ./scripts/check.ps1
- Store listing (WINDOWS-STORE rules 7-9, 12, 13; ticket E0135): `tools/store/listingData.csv` is the **authoritative listing file** - the product's own Partner Center export, patched in place by `tools/store/build-store-listing-csv.ps1` from the decks `tools/store/listing/*.txt`, fill-only unless `-Refresh <field>` (which lists every cell it replaces); `msix/out/store-import/` holds a gitignored copy for the dashboard's folder import, never a second source. The builder exits 1 and writes nothing for a shipped language with no column (or no deck), a cell over its cap (Feature 200, ShortDescription 1000, Description 10000, ReleaseNotes 1500 after CRLF, 21 words over SearchTerm1..7) and any break of the export's dialect; it writes the export's own dialect (BOM, CRLF between records, LF inside cells, no trailing newline). `tools/store/listingData.sample.csv` is the same export with the copy cells emptied; both CSVs are pinned `-text` in `.gitattributes`, and `-FillNothing -Out <scratch>` on each is byte-identical. The cases run as `pwsh tools/store/test-store-listing-csv.ps1` (3 s) and as `go test -tags nodynamic ./tests -run TestStoreListing`.
- Every go command that reaches internal/jxl (the app, its pipeline, tests/, tools/ocrlab) needs the
  `nodynamic` build tag, or `internal/jxl/nodynamic_guard.go` stops the compile (ticket 115, see Pitfalls).
  The scripts set it through `scripts/lib/gobuildtags.ps1` (GOFLAGS), CI through release.yml's `env`,
  golangci-lint through its config and VS Code through `go.buildTags`. By hand:
  `$env:GOFLAGS = '-tags=nodynamic'` once per shell, or `go test -tags nodynamic ./tests/`.
- Every check speaks `CHECK-VERDICT` (pointer: docs/contracts/CHECK-VERDICT.md): exit `0` PASS, `1` FAIL,
  `2` COULD NOT VERIFY (a missing tool, input or git result - **never a pass**), `3` PASS WITH ADVISORIES,
  and the **last line** is the verdict (`check: PASS`, `test: COULD NOT VERIFY (1)`, ..). Quote that line
  as evidence, not the scrollback above it. A fresh clone without `test_doc/` ends in COULD NOT VERIFY
  by design. Which runner owns each check: `configs/check-placement.jsonl` (a new check gets a record).
- `check.ps1` writes `temp/logs/gate-evidence.json` (verdict, plan, every child's verdict, and the tree
  hash taken before and after the run - an edit during the run voids it); `release.ps1` blocks the tag
  step unless it is the full default plan, every child passing, on HEAD's tree (only `DEV/COMMIT_LOG.md`,
  appended by build-local after its commit, may differ). A `-Plan` subset never unlocks it, and a `-Fast`
  run writes no evidence at all.
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
  or telemetry-shaped dependency fails it until a row covers it. `scripts/site-addresses.ps1`: every
  address of the site that a program, extension message, README, listing source, manifest or test holds is
  in `configs/site-held-addresses.jsonl` and still answers (page, query, section anchor); a new holder or a
  new address fails it until the list has it (`-Suggest` prints the record), and a moved page leaves a
  forwarder - the policy is [docs/SITE_ADDRESSES.md](docs/SITE_ADDRESSES.md). `scripts/release-notes.ps1`: the site's
  release-notes page is the render of `docs/release-notes.json` (one record per `v*` app tag, the same bullets in
  en/ru/uk, newest first); the check fails on a tag with no record, a record out of order or a hand-edited block,
  and a release adds its record first (`-Render` rewrites the page) - never edit the page per ticket. The same
  render writes the release facts of the other pages: the landing's newest-record block, the version beside each
  download and store button, the asset names. The extension's `ext-cws-v` / `ext-edge-v` tags name the version
  `node extension/build.mjs record-version` recorded there, and the store build stops on any other.
- "What must I read before touching this?": `./scripts/doc-query.ps1 -Area <area> -Trigger <trigger>`
  (`-Path <file>` for one file's record, `-List` for the vocabularies). Register a document before
  anything links to it.
- Local engineering corpus (`DEV/plan` and `DEV/private`, git-ignored): `./scripts/doc-registry-local.ps1` holds its own registry, `DEV/DOCUMENT_REGISTRY.jsonl` (every record resolves, every `.md` of the trees is claimed, every link resolves; hand-run, COULD NOT VERIFY on a clone). Run it after a ticket, research note or phase file is added, moved or renamed; the tracked registry names nothing in those trees (DOC-INTERNAL-QUALITY 0.10).
- Release checklist (prints only, runs nothing but read-only checks): ./scripts/release.ps1. Its header
  carries three verdicts the tag step takes as input: the gate evidence, the contract gate
  (`scripts/contract-gate.ps1`: the pointers in `docs/contracts/` against the catalog registry -
  PASS / WARN pass, FAIL / UNVERIFIED block, and an unreachable catalog is UNVERIFIED, never PASS) and the
  Store preflight (the sza plugin's shared read-only `tools/Store-Preflight.ps1`, `msstore submission get` only:
  BLOCKED on a pending submission, a version not above the live one or a price that is not Free; MANUAL, never
  a block, when the channel is unavailable). No agent runs `msstore` itself.
- Sliced code audit: ./scripts/audit-slices.ps1 cuts the audit scope (shipped code + release path) into
  slices a session can read line by line (`-Write` writes the manifest into the audit ticket's folder,
  `-Tail` slices files added since, `-Summary` measures the campaign and exits 0 only when it is closed).
  Hand-run; ticket 34 is its first campaign.
- Free-promotion campaign (ticket 111; canon PROMOTION.md): `./scripts/promo-metrics.ps1` writes a dated, read-only counters checkpoint (GitHub, winget, Chrome, Edge, Store - `unknown` where unreadable, no telemetry); `tools/store/make-social-card.ps1` renders `assets/social-card-*.png`; `tools/store/make-demo.ps1` records and encodes `assets/demo.*`; the discoverability fields live in `docs/discoverability.json`. Every outward step of the campaign waits for the owner's go.
- In-app help: ./scripts/help-check.ps1 (the gate child, no browser; the full gate and the release pre-flight run it with `-Release`, where a set older than its inputs FAILS, and the fast tier only advises); node tools/helpgen/repage.mjs [--check] is how a wording change of the copy deck, help.css or the renderer reaches both copies of the set in the same edit: it re-renders the pages from the committed captures with no build and no browser (0.07 s on 2026-10-29), touches no screenshot and exits 2, writing nothing, when a window input (ui.html, i18n.js, the capture code) changed after the last capture; node tools/helpgen/generate.mjs regenerates the screenshots, the pin places and the pages after a change of what the window shows (needs the Go toolchain and tools/sitecheck's Playwright Chromium; it builds the GUI and its capture alone took 106 s on 2026-10-28) and **it belongs to the release, not to the window edit**: CONTEXT-HELP rule 18 carries a dated exception for the screenshot half (ticket E0125, owner decision 2026-10-30, expires 2026-12-31), so between a window change and the release the committed screenshots lag the window and `help-check` names the moved input as an advisory; a ticket that changes the window records `node tools/helpgen/generate.mjs` as a `RELEASE` item, and the release pre-flight runs it, then `--check`; node tools/helpgen/walk.mjs drives F1 and the button on every surface in five locales and both themes
- Settings inventory (ticket 122; `APP-SETTINGS` rule 11): ./scripts/settings-inventory.ps1 (a gate child) holds `docs/settings-inventory.json` - every key `collectSettings()` persists, with its control, page, caption, default, help region and command-line flags - equal to `ui.html`, the help map, the help copy deck and the README flag tables; after adding, renaming or moving a setting run `go run ./tools/settingsinv -render` (never edit the JSON), and name the new caption in `tools/helpgen/copy/` before regenerating the help set
- UI contracts (ticket 119; `UI-SIZING`, `BUTTONS`, `BOOLEAN-OPTION`, `COLLAPSIBLE-GROUP`, `SETTINGS-NAVIGATION`): `node tools/uicheck/run.mjs` measures the real GUI window in headless Chromium (geometry, contrast, keyboard, negative cases, page-switch timings with every raw sample; `--quick`, `--only <scenario>`; last line is the CHECK-VERDICT line; see [tools/uicheck/README.md](tools/uicheck/README.md)); the product's size-profile record, permanent variants and what is still owed are in [docs/UI_PROFILE.md](docs/UI_PROFILE.md). It builds the GUI and runs for minutes, so it belongs to the release pre-flight: run it there when `ui.html` changed since the last release, and `--only reader` when the generated reader chrome (`internal/htmlgen` `chromeUICSS`, `navbar.go`, `search.go`) or the extension viewer's toolbar (`extension/src/viewer.*`) changed. A ticket that changes them records the run as a `RELEASE` item; the help screenshots wait for the release too (CONTEXT-HELP rule 18 exception, ticket E0125 - `node tools/helpgen/generate.mjs` is another `RELEASE` item, and `node tools/helpgen/repage.mjs` carries any wording change in the same edit) - the window's regions are the help's inputs
- OCR visual-fidelity lab: go run -tags nodynamic ./tools/ocrlab verify | fetch | synth | seed | run | score | report | gate | exchange (see tools/ocrlab/README.md)

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
- External helpers (pdftotext, Tesseract, Calibre, 7-Zip, ffmpeg/ImageMagick, FileDO):
  - internal/procrun: the one way to run a helper - per-tool size-scaled deadline (`DOCHT_TOOL_TIMEOUT_SCALE` multiplies it), process-tree kill (job object on Windows, process group elsewhere), capped output, one error type naming the tool. Every call takes the run's context, so Ctrl+C stops a helper at once and the run ends as interrupted. Do not call exec.Command for a helper directly.
  - internal/bundledtools: the Windows-only bundled pdftotext, unpacked into a content-hash-named cache folder. Nothing is ever installed on the user's machine (tests/no_auto_install_test.go).
  - internal/fdsec: FileDO secret files (.fd-sec) - screened by length, decrypted by the installed FileDO (`DOCHT_FILEDO` -> PATH -> setup folder -> winget Links -> Store alias) into one private copy under the user's temp folder, password only in the child's environment, FileDO's output never forwarded (read: the banner build and, beside exit 5, the exact stderr line `reason=destination-name`, which gets its own message); the container stays the run's identity and the plain copy is gone as soon as the extractor has read it (pointer [docs/contracts/FDSEC.md](docs/contracts/FDSEC.md); `internal/pipeline/container.go` is the pre-stage, `cmd/doc-html-ui/fdsecanswer.go` the window's masked-password answer). A new GUI file named `*secret*` is gitignored - name it otherwise.
- Translation:
  - internal/translator
- Interface language (13 languages, `en ru uk de it es fr pt ar hi bn ur zh`):
  - internal/i18n: `Codes` is the list, `Add()` takes 12 translations or panics, `Resolve()` is the
    order (explicit `-ui-lang` -> saved -> OS -> English). `app.New` calls `SetLanguage` once, so
    `i18n.S()` works process-wide instead of threading a language through htmlgen's signatures.
  - internal/app/splash/*.txt: the console splash, one embedded file per language.
  - cmd/doc-html-ui/i18n.js (GUI dictionary), extension/_locales/<code>/messages.json (extension).
- GUI window structure (ticket 119): `cmd/doc-html-ui/ui.html` is one page of three layers. The **token block** (`html { --ui .. --tx .. --tgt .. }`, `html[data-input=touch]`, `data-size`, `data-text`) is the one resolver for every covered component (never hard-code a px size in a component; use the tokens and `var(--s)`); the **components** are `button` (ranks `.primary` / `.secondary` / `.danger`, rows `.btn-row`, stateful `.state-stack`), `.opt` (a boolean option; `.opt-help-wrap` for local help), `.group` (a `<details>` group) and `.rail-item` (the page rail); the **pages** are seven `<section class="page" id="page-..">` shown one at a time by `showPage()` (`convert`, `ollama`, `ocr`, `windows`, `interface`, `advanced`, `about`; ids are stable and saved). The dialog answers read Cancel first, the affirmative last (BUTTONS 4.4). `/api/resources` is the About page's local, no-network resource lookup.
- In-app help (ticket 118, `CONTEXT-HELP`; pointer [docs/contracts/CONTEXT-HELP.md](docs/contracts/CONTEXT-HELP.md)):
  - `cmd/doc-html-ui/help/help-map.json` is the **help map** (CONTEXT-HELP rule 17): one row per surface of the GUI launcher, its form, its help id and the `none` / `inherit` reasons. It ships inside the exe with the generated pages (`go:embed all:help`, served under `/help/`) and the gate reads the same file.
  - `cmd/doc-html-ui/help.js` is the one resolver: F1 (bare, on the page only) and every `?` button call it. A region of `ui.html` is marked `data-help="<surface>.<region>"`, a dialog names its `surface:`; the gate counts both against the map.
  - `tools/helpgen` generates the pages, the screenshots with their pins and `help/` (the site copy, the same bytes) from the copy deck `tools/helpgen/copy/*.json`, the GUI dictionary `i18n.js` and the real build. **Never edit `cmd/doc-html-ui/help/` or `help/` by hand**; change the window, the dictionary or the copy deck, run `node tools/helpgen/repage.mjs` from PowerShell for the wording (the same edit), and leave `node tools/helpgen/generate.mjs` - the screenshots and pin places, about two minutes with the GUI build - to the release (dated exception on CONTEXT-HELP rule 18, ticket E0125; see [tools/helpgen/README.md](tools/helpgen/README.md)). `node tools/helpgen/walk.mjs` is the automated F1 / button walk.
  - `internal/helpcheck` is the gate (`scripts/help-check.ps1`, also `tests/help_test.go`): surfaces against the map, pages against the locale set, links, rule 12, text style, the two copies, and the freshness of the set against the digests of its inputs (an advisory in the fast tier, a failure under `-Release` in the full gate and the release pre-flight: `go run ./tools/helpcheck -strict-freshness`). The locales the GUI ships and the help has no pages in are the set baseline `configs/help-baseline.txt` - it may fall, never rise.
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
- The output HTML carries a client-side reader layer injected by internal/htmlgen (navbar.go readerScript/readerCSS on chapter pages, plus a matching toolbar/script on index.html). Keep storage scopes distinct: fit-width uses sessionStorage; reading themes and reading position use localStorage (must survive sessions). Reading position is namespaced by `epub.Book.ReaderKey` (htmlgen.ReaderKey: source file name + size, original title, page count), set once in the pipeline before translation and read by every page and index.html - never recompute it from the (translated) title. A saved position is restored only when the URL has no fragment. Values enter generated scripts only through `jsString`, paths enter hrefs only through `epub.URLPath`.
- Windows and non-Windows behavior is split via *_windows.go and *_nonwindows.go files in several packages.
- The interface language dresses the **chrome only**. A converted page keeps the *document's* `<html lang>`; the navbar and reader controls carry their own `lang`/`dir` and mirror for `ar`/`ur`. Putting the UI language on `<html lang>` stops Chrome offering "Translate page" - the product's entire free workflow. Guarded by `TestConvertedChromeLanguage` in tests/smoke_test.go and by the RTL assertions in tools/store/make-screenshot.ps1.
- The canon's house text style (DOCUMENTATION_CONCEPT.md §5) is **scoped by language here**: it binds `en`, `ru` and `uk` only. The other ten interface languages follow their own script and are exempt - tests/typography_test.go enforces that scoping, so do not widen the check to all thirteen.
- A change that alters what the user sees in the GUI launcher (a control, a label, a region, a dialog) changes its help in the same edit (CONTEXT-HELP rule 18): the window's own strings are the help's names and one-line summaries, the explanation is in `tools/helpgen/copy/`, and the screenshots are regenerated, not edited. A new region gets `data-help`, a new dialog a `surface:` and a row in the help map, or the gate fails.
- Optional OCR overlay (-ocr): internal/ocr shells out to the external Tesseract binary (parses TSV for bboxes) and rewrites document images into positioned, translatable text plates. It runs in internal/pipeline/pipeline.go after nav injection and before translation (so overlay text is translated too), and is strictly best-effort - a missing tesseract or a failed image never aborts the conversion. English data ships in <exe>/tessdata; other languages download on demand (-ocr-download / GUI) into the per-user folder (os.UserCacheDir()/doc-html-translate/tessdata), catalogue codes only, verified against pinned SHA-256 digests (internal/ocr/download.go). Applies to EPUB and PDF (formats whose images exist at HTML stage).
- Plate concealment modes: every OCR plate records how it hides the source lettering as
  `data-ocr-mode` = `fill` / `reconstruct` / `mask` (decided from the colour ring just outside the
  block), always as CSS over the untouched `<img>` - internal/ocr/conceal.go and
  extension/src/ocr-conceal.js are a pinned PARITY.md invariant. Every constant of the OCR-PIPELINE
  section 5 status table also carries a comment `OCR-OVERLAY rule 13: <status> - <source>.`
  (`derived` / `inherited` / `policy`) above its declaration in both editions;
  tests/ocr_constant_status_test.go fails on a missing marker or a status that disagrees.

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
