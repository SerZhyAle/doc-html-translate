# Documentation surfaces

The single list of user-facing text surfaces that move **together** when a feature ships.
Session logs show this set being re-discovered by hand every time ("обнови документацию") and
individual languages being asked for as separate follow-ups. Treat it as one manifest so no
surface is forgotten and **a language is never a separate request**.

Driven by the `/docs-sync` skill. Keep this file honest: when a surface is added or renamed,
edit the table.

## How many languages

Two tiers, and which tier a surface belongs to is a decision, not an accident:

- **13 languages** - `en ru uk de it es fr pt ar hi bn ur zh`. Everything the product *says while you
  use it*, plus the copy a store or a search engine serves per language. Adding a language means adding
  it to every row in this tier at once.
- **3 languages** - `en ru uk`. Long-form prose that is rewritten often or read by store reviewers:
  the documentation trio, the "what's new" notes, the extension's long store description. These stay in
  the languages their author can actually proofread; a stale or unreviewable machine translation of this
  material is worse than none. Strategic decisions D3 and D4 in
  [`DEV/plan/2026-07-28_thirteen-ui-languages.md`](plan/2026-07-28_thirteen-ui-languages.md).

## Surfaces

| Surface | Files | Languages | Notes |
|---|---|---|---|
| CLI + converted page strings | `internal/i18n/i18n_cli.go` · `i18n_reader.go` · `internal/app/splash/*.txt` | **13** | `i18n.Codes` is the code list. Every `Add()` takes 12 translations or panics; the splash has one `.txt` per language. |
| GUI dictionary | `cmd/doc-html-ui/i18n.js` | **13** | One object per language, all keys present. Guarded by `cmd/doc-html-ui/i18n_test.go`. |
| Extension UI | `extension/_locales/<code>/messages.json` × 13 | **13** | Chrome names the Chinese directory `zh_CN`. Keys must match `en` exactly; guarded by `extension/test/i18n.test.mjs`. |
| Repo README | `README.md` · `README_RU.md` · `README_UK.md` | en / ru / uk | Cross-linked at the top of each. Feature list and flag table mirrored. |
| Landing page | `index.html` | en/ru/uk in-page | GitHub Pages hero. A new feature **leads the hero**, not a collapsed section (see memory `feature-prominence-3-lang`). Language handled in-page - edit all three copies inside the one file. Also carries the `hreflang` block and the link row to the ten per-language pages. |
| Per-language landing pages | `<code>/index.html` × 10 (`de it es fr pt ar hi bn ur zh`) | **13** with the root page | One complete page per language, single-language (no `data-l`), self-canonical. `ar` and `ur` are `dir="rtl"` with LTR islands for the product name, format pills and commands. |
| Sitemap / robots | `sitemap.xml` · `robots.txt` | n/a | Lists every page with all 14 `hreflang` alternates. Regenerate when a page is added. |
| Extension page | `extension.html` | en/ru/uk in-page | Browser-extension landing section. Carries the same `hreflang` block. |
| App docs | `docs.html` · `docs.ru.html` · `docs.uk.html` | en / ru / uk (separate files) | **The 3-language trio.** Change all three in lockstep; content must be mirrored, only the prose is translated. Each carries the machine-translation disclosure. |
| Store listing sources | `tools/store/listing/<code>.txt` × 13 | **13** | Title, short description, description, features, search terms and `@@ReleaseNotes` - **all 13** (owner's decision 2026-08-13, superseding D3's 3-language rule: a listing that reads complete in a language and then shows no "what's new" reads as neglect). The ten machine-translated files end their notes with the same disclosure their description carries. Rendered into the CSV by `tools/store/build-store-listing-csv.ps1` - never hand-edit `listingData.csv`. |
| Partner Center CSV | `tools/store/listingData.csv` | render target | Patched, never regenerated: only empty cells are filled, so the listing-asset URLs survive. A locale absent from the export is dropped silently on import - add it in the dashboard, export again, re-run. |
| winget manifests | `winget/SerZhyAle.DocHtmlTranslate.locale.<tag>.yaml` × 13 | **13** | Short description + description per locale. `PackageIdentifier`, version and the version manifest are frozen anchors. |
| Installer strings | `installer/doc-html-translate.iss` | **8** | `en ru uk de it es fr pt`. `ar hi bn ur zh` have no official Inno `.isl` and fall back to English **in the installer only** (decision D2) - the comment above `[Languages]` says so; do not "complete" the list. |
| MSIX manifest | `msix/AppxManifest.xml` | **13** | `<Resource Language>` per Store locale tag. Without a row here the Store will not offer the listing in that locale. |
| Extension README | `extension/README.md` | en | Dev/user readme for the extension edition. |
| Extension store listing | `extension/store/LISTING.md` | name + short in **13**, long description en/ru/uk | Name and short description come from `_locales` and the stores serve them automatically. The long description stays in three languages on purpose - this copy was rejected twice by Chrome review for keyword spam and the wording that passed is tuned word by word. |
| Store privacy | `extension/store/PRIVACY.md` | en | Only when data handling changes. |
| App screenshots | `tools/store/*.png` (39) | **13** | `reading-view-<locale>`, `table-of-contents-<locale>`, `gui-<locale>`, Partner Center locale codes. Generated by `make-screenshot.ps1` + `make-gui-screenshot.ps1`. |
| Extension screenshots | `extension/store/screenshots/shot{1,2}-<code>.png` (26) | **13** | Generated by `npm run screenshots` in `extension/` (`scripts/make-screenshots.mjs`) - never hand-edited. 1280x800, page viewport only. The run refuses a frame whose chrome carries the wrong language or mirrors when it should not. Screenshot 3 (the native right-click menu) stays a manual capture - the OS draws it, so it cannot appear in a headless frame. |
| Dev changelog | `DEV/CHANGELOG.md` | en | Not "docs" per se, but ships with the change - use `/changelog`. |

## Invariants

- **Tier parity is mandatory.** Within a tier, never leave a language behind for "later": a 13-language
  surface takes all 13 in the same edit, and the `docs.*` trio takes all three. Moving a surface between
  tiers is a decision to record here, not a shortcut to take quietly.
- The **interface language is not the document language**. `<html lang>` on a converted page is the
  book's language; the chrome carries its own `lang`/`dir`. Breaking this stops Chrome offering
  "Translate page", which is the product's entire free workflow. See `docs/PARITY.md`.
- Translations outside en/ru/uk are machine-made and **disclosed as such** in the three READMEs, the three
  docs pages and the ten landing pages - 16 places. If you add a surface with user-facing prose in those
  ten languages, it gets the disclosure too.
- A **new user-facing feature leads** the landing hero and is mirrored across every surface above, in all
  its languages, without being asked (memory `feature-prominence-3-lang`).
- Typography everywhere: short hyphens, Russian ё, ".." not "..." - in `en`, `ru` and `uk`. The other ten
  follow their own script's conventions and are exempt from the house rule; `tests/typography_test.go`
  scopes the check accordingly.
- Screenshots are regenerated by their own scripts (`tools/store/make-screenshot.ps1`,
  `tools/store/make-gui-screenshot.ps1`), not hand-edited.

## Developer-only surfaces (no user-facing propagation)

Not every change moves the sixteen places above. A ticket recorded here is one that deliberately
touches none of them, so a later reader does not go looking for the missing README line.

- **`2026-08-11_ocr-visual-fidelity-lab`** - the OCR quality benchmark (`tools/ocrlab`). Developer
  surfaces only: `tools/ocrlab/README.md`, `AGENTS.md`, `test_doc/CORPUS.md`, `DEV/plan/ROADMAP.md`
  and the changelog. **No** README, site, landing-page, store-listing or `_locales` change, and no
  translation work. The strategic spec declines it outright: no public quality claim moves until the
  benchmark has a stable, reproducible result, and the corpus that would make it stable does not
  exist yet. The one hook into shipped code (`DOCHT_OCR_DIAG`) is off unless set and changes no user
  output, so there is nothing for a user to be told. Revisit if a tuning phase makes a fallback
  visible to a reader - that would be a user-facing feature and would lead the hero like any other.
