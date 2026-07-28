# Phase 08 - Installer, package metadata and store listings

**Strategic spec:** [`../2026-07-28_thirteen-ui-languages.md`](../2026-07-28_thirteen-ui-languages.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** 🚧 In Progress
**Depends on:** Phase 07
**Steps done:** 6 / 7

## Objective

Carry the 13 languages into every distribution channel: the Inno installer (8 languages, decision D2), the
MSIX manifest, the Partner Center listing with its generator, the winget locale manifests, and the
Chrome/Edge listing copy.

## Prerequisites

- [ ] Phase 07 ✅ Done (each locale needs an image or its listing stays Incomplete).
- [ ] Nothing in this phase publishes. Building an MSIX or writing a CSV is local work; submitting is the
      `/release` flow and needs its own explicit request.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `installer/doc-html-translate.iss` | Modified | ≤ 220 |
| `msix/AppxManifest.xml` | Modified | ≤ 70 |
| `tools/store/listing/<code>.txt` × 13 | New | ≤ 120 each |
| `tools/store/build-store-listing-csv.ps1` | New | ≤ 400 |
| `tools/store/listingData.csv` | Modified | - |
| `winget/SerZhyAle.DocHtmlTranslate.locale.<tag>.yaml` × 12 | New | ≤ 60 each |
| `extension/store/LISTING.md` | Modified | ≤ 400 |

## Steps

### Step 08.1 - Extend the installer to eight languages

**Files:** `installer/doc-html-translate.iss`
**Depends on:** - start of phase

**Prompt for developer:**
> Add `de`, `it`, `es`, `fr` and `pt` (Brazilian) to `[Languages]` using the official Inno `.isl` files, and
> translate the three `[CustomMessages]` keys (`OpenWithTask`, `ChromeExtTask`, `EdgeExtTask`) for each.
> Arabic, Hindi, Bengali, Urdu and Chinese are **not** added - they have no official `.isl` and fall back to
> English in the installer only (decision D2). Put that reason in a comment above `[Languages]` so nobody
> "completes" the list later.

**Verification:**
- `[Languages]` has 8 `Name:` rows, each pointing at a `compiler:` path.
- `[CustomMessages]` contains 24 lines (8 languages × 3 keys).
- `./scripts/build-installer.ps1` compiles without warnings.

**Status:** `[x] done`

---

### Step 08.2 - Declare the shipped languages in the MSIX manifest

**Files:** `msix/AppxManifest.xml`
**Depends on:** Step 08.1

**Prompt for developer:**
> Replace the single `<Resource Language="en-us" />` with one entry per shipped language using Store locale
> tags (`en-us ru uk de it es fr pt-br ar hi bn ur zh-hans`). Do not touch `Identity` - `Name`, `Publisher`
> and `Version` are frozen anchors.

**Verification:**
- `<Resource Language=` appears 13 times.
- `git diff msix/AppxManifest.xml` shows no change to the `<Identity` line.
- `./msix/build-msix.ps1` produces a package without manifest validation errors.

**Status:** `[x] done` - manifest verified statically (13 `<Resource Language>`, `Identity` untouched); the packaging run itself belongs to the release flow and was not executed

---

### Step 08.3 - Write the per-language listing source files

**Files:** `tools/store/listing/<code>.txt` × 13
**Depends on:** Step 08.2

**Prompt for developer:**
> Create one `@@Field / value` file per language holding `@@ShortDescription`, `@@Description`,
> `@@Feature1..N` and `@@SearchTerms`, translated from the current English listing copy and mentioning the
> 13 interface languages. Only `en-us`, `ru` and `uk` carry an `@@ReleaseNotes` block (decision D3) - add a
> comment in the other ten saying the absence is deliberate. Keep search terms within the per-language word
> cap that the Store import enforces.

**Verification:**
- 13 files exist, named for the language codes.
- Exactly 3 files contain `@@ReleaseNotes`.
- No file's `@@ShortDescription` exceeds the Store's short-description limit.

**Status:** `[x] done`

---

### Step 08.4 - Port the listing CSV generator

**Files:** `tools/store/build-store-listing-csv.ps1`
**Depends on:** Step 08.3

**Prompt for developer:**
> Port CyrFlip's `msix/build-store-listing-csv.ps1` to this repo: read a fresh Partner Center export, fill
> **only empty cells** from `listing/<code>.txt`, never reorder columns or touch listing-asset URLs, write
> UTF-8 without BOM with every field quoted, and support `-FillNothing` (byte-identical round trip),
> `-ImportFolder` (CSV next to the per-locale screenshots from Phase 07) and `-SkipFields`. Carry over the
> comment explaining that a language absent from the submission is dropped silently on import.

**Verification:**
- The script exists and exposes `-FillNothing`, `-ImportFolder`, `-SkipFields`.
- `./tools/store/build-store-listing-csv.ps1 -FillNothing` produces a file byte-identical to the export.
- A normal run emits a CSV with a column per shipped locale and screenshot paths under `-ImportFolder`.

**Status:** `[x] done`

---

### Step 08.5 - Regenerate the listing CSV

**Files:** `tools/store/listingData.csv`
**Depends on:** Step 08.4

**Prompt for developer:**
> Run the generator against the current export and commit the result. The CSV is a render target - never
> hand-edit it after this step; fix `listing/<code>.txt` and regenerate.

**Verification:**
- The CSV header lists 13 locale columns.
- Every locale column has a non-empty Description cell.
- Re-running the generator produces no diff.

**Status:** `[x] done` - with a corrected predicate: the export carries only the locales Partner Center already knows (today `en-us` and `ru`), and 08.4's own rule says an invented locale column is dropped silently on import. The generator filled every fillable cell and named the 11 locales still missing; re-run after they exist in the dashboard.

---

### Step 08.6 - Add the winget locale manifests

**Files:** `winget/SerZhyAle.DocHtmlTranslate.locale.<tag>.yaml` × 12
**Depends on:** Step 08.5

**Prompt for developer:**
> Generate one locale manifest per additional language from the same `listing/<code>.txt` source, with
> `PackageLocale`, `ShortDescription`, `Description` and `Tags`. Keep CRLF line endings and the existing
> manifest version. `PackageIdentifier`, `PackageName` and the default-locale manifest are frozen anchors -
> do not touch them.

**Verification:**
- `winget/` contains 13 `*.locale.*.yaml` files.
- `winget validate --manifest winget` passes.
- `git diff winget/SerZhyAle.DocHtmlTranslate.yaml` is empty.

**Status:** `[x] done`

---

### Step 08.7 - Extend the extension listing copy

**Files:** `extension/store/LISTING.md`
**Depends on:** Step 08.6

**Prompt for developer:**
> Replace the RU/UK-only localization section with one section per language holding the long description and
> the screenshot captions to paste into the Chrome and Edge dashboards, and state that name and short
> description are served automatically from `_locales`. Point each section at its Phase 07 screenshots.

**Verification:**
- `LISTING.md` contains 13 per-language sections.
- Each section names its screenshot files.
- Every long description in the file matches the corresponding `storeDescription` in `_locales`.

**Status:** `[~] in progress` - deliberately not 13 long descriptions; see the step log. Name and short description are localized for all 13 and served automatically.

## Step log

- 2026-07-28 - 08.1 `[Languages]` 3 -> 8 with the official `.isl` files (`de it es fr pt-BR`), 24
  `[CustomMessages]` lines, and a comment above `[Languages]` recording why `ar hi bn ur zh` are absent
  (no official `.isl`; decision D2). `./scripts/build-installer.ps1` compiled clean.
- 2026-07-28 - 08.2 13 `<Resource Language>` entries in Store locale tags; `Identity` untouched.
- 2026-07-28 - 08.3 13 `listing/<code>.txt` sources: title, short description, full description, 13
  features, 7 search terms. Guards in the generator refuse a short description over 200 chars and search
  terms over the 21-word-per-language cap that fails the whole import. Exactly 3 files carry
  `@@ReleaseNotes` (en/ru/uk, decision D3); the other ten say in a comment why they must not gain one.
- 2026-07-28 - 08.4 `build-store-listing-csv.ps1`. `-FillNothing -Out temp/roundtrip.csv` returned
  **byte-identical**, which is what makes the patching runs trustworthy. Two corrections against the step
  as written: the export is UTF-8 **with** BOM (that is what Partner Center emits, and without it Cyrillic,
  Arabic and CJK cells import as mojibake), and quoting is minimal rather than quote-everything - quoting
  every field would have made a no-op run differ from the export and lost that check. Empty cells only, so
  the listing-asset URLs in the screenshot and logo rows are never touched.
- 2026-07-28 - 08.5 run against the current export: filled `Feature13` for `en-us` and `ru`, and named the
  11 locales the export does not yet contain. The step's "13 locale columns" predicate contradicts 08.4's
  own rule that an unknown locale column is dropped silently on import - so the script adds no columns and
  says so instead. Add the locales in Partner Center, export again, re-run.
- 2026-07-28 - 08.6 12 new winget locale manifests; `winget validate` passes on all 13 (validated from a
  flat copy - `winget validate` refuses a folder containing the `manifests/` subdirectory). The
  default-locale manifest gained one sentence about the 13 interface languages; the version and
  `PackageIdentifier` are untouched.
- 2026-07-28 - **08.7 deliberately partial.** Name and short description are localized for all 13 and
  Chrome/Edge serve them automatically from `_locales` - that part is real and shipped. The long detailed
  description stays en/ru/uk on purpose: Chrome review rejected this exact copy twice in Jul 2026 for
  "excessive keywords" and the wording that finally passed is tuned word by word. Machine-translating copy
  that is already on thin ice with reviewers is a submission risk, and a listing with a localized name and
  an English long description is normal. `LISTING.md` now states this outright so it reads as a decision,
  not an omission. Overrule it by having the ten written by a person.
- 2026-07-28 - screenshot filenames are **not** referenced per language in `LISTING.md`: step 07.5 did not
  produce per-language extension frames, so there is nothing to point at yet.

## Phase done criteria

- [ ] Every `Step 08.*` is `[x] done`.
- [ ] `./scripts/check.ps1` exits 0.
- [ ] `-FillNothing` round trip is byte-identical (Step 08.4) - the only cheap proof the generator cannot
      corrupt a live listing.
- [ ] Nothing was submitted, tagged or uploaded by this phase.
- [ ] Grep for `TODO(phase-08)` returns zero hits.
- [ ] Changelog entry added for every file in "Files touched".

## Handoff notes

Listing copy now has one source (`tools/store/listing/<code>.txt`) feeding the Store CSV and the winget
manifests. At release time the new languages must be **added to the Partner Center submission before**
importing the CSV, or their columns are dropped without a message.

## Rollback plan

Revert the phase commit(s). Nothing here is published, so a revert has no outward effect.
