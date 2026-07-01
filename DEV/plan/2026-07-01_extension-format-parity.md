# Strategic spec: 2026-07-01_extension-format-parity - Extension document-format parity with the desktop app

**Ticket:** 2026-07-01_extension-format-parity
**Status:** BlockNeedUserTest
**Priority:** 50
**Date:** 2026-07-01
**Tier:** Strategic
**Tactical plan:** `DEV/plan/2026-07-01_extension-format-parity/` (created by /spec-tech)

> **BlockNeedUserTest (2026-07-01):** all code + docs landed; `npm test` (31) and `npm run build`
> pass. Needs hands-on browser acceptance: load the unpacked extension and, via the file picker,
> open one sample each of TXT, Markdown, FB2, RTF, local HTML, MOBI and AZW3 - each must reflow and
> Chrome must offer "Translate page". Confirm PDF/EPUB still work; confirm a real .mobi and .azw3
> render text + TOC (images where present); confirm a DRM-protected ebook fails with a clear notice.
> Store name change is drafted and **awaits owner sign-off** before publishing.

> **Scope:** STRATEGIC. Goals, constraints, open questions. No class names, paths, line
> budgets, or framework module details - those belong to the tactical plan.

Reference: [`docs/PARITY.md`](../../docs/PARITY.md) (JS<->Go port map) and the feasibility
study [`DEV/research/extension_formats_feasibility_ru.md`](../research/extension_formats_feasibility_ru.md).

---

## 1. Problem

The browser extension opens only PDF and EPUB, while the desktop editions (CLI, GUI, MSIX)
convert nine formats. A user who installs the extension expecting the app's advertised range
hits a wall for TXT, Markdown, FB2, RTF, HTML, MOBI and AZW3 and must fall back to the
desktop app. The gap also pins the store listing to a narrow "PDF & EPUB" identity that
undersells the product. Affected area: the extension's reader intake (not the desktop app,
which already handles these formats).

## 2. Goals

1. The extension opens and reflows TXT, Markdown, FB2, RTF, local HTML, MOBI and AZW3, in
   addition to the existing PDF and EPUB.
2. Each new format flows through the same reader experience users already get: reflow,
   theming, a table of contents when the source carries one, language detection, and
   on-scroll image OCR where images exist.
3. Everything stays fully client-side and offline - no dependence on the desktop binary,
   Calibre, or any network service.
4. The store listing, privacy copy, READMEs, website and file associations describe the
   true wider format range, so the next publication is correct.

**Non-goals:**

- Translation backends, glossary, or any feature beyond format intake (tracked elsewhere).
- DRM-protected Kindle files (the desktop app cannot do these either).
- Byte-identical output with the Go extractors; the bar is behavioural parity ("opens and
  reads correctly"), not identical HTML.
- Comic formats (CBZ/CBR), DjVu and DOCX - separate roadmap items.

## 3. Wishes and constraints

### 3.1 Owner wishes

- Phased delivery inside one release: simple formats first (TXT, MD, FB2, RTF, HTML), then
  the binary ebook formats (MOBI, AZW3), so each phase is independently testable and the
  simple formats ship even if the ebook phase slips.

### 3.2 Hard constraints

- **Platform / versions:** Manifest V3, Chrome and Edge (min Chrome 105, already the
  baseline). Extension CSP forbids remote code - every parser must be vendored, not loaded
  from a CDN.
- **Performance:** no hard budget, but a large file must not freeze the reader tab; follow
  the existing lazy/queue patterns.
- **Data compatibility:** n/a - no persisted schema change; reuses existing viewer prefs.
- **Localization:** user-facing strings in the three supported locales (EN/RU/UK).
- **Accessibility:** reader output stays semantic HTML as today.

### 3.3 Owner inputs (Approval gate)

- **Related tickets:** [`2026-07-01_cross-edition-parity`](2026-07-01_cross-edition-parity.md)
  (shared-invariant alignment across editions - this ticket is the extension-side format
  catch-up). Feasibility: [`extension_formats_feasibility_ru.md`](../research/extension_formats_feasibility_ru.md).
- **Copy/tone policy:** the store name/short/long description move from "PDF & EPUB" to the
  wider range; the owner curates the final marketing wording at release.
- **Localization:** EN/RU/UK for any new UI strings and the store listing.
- **Validation level:** JS unit tests for pure parse/reflow logic, plus manual acceptance on
  a real sample file per format (the extension's established gate).
- **Owner sign-off:** required before publishing - the store rename is public-facing.

## 4. Current architecture context

The extension is a standalone, fully client-side product with no bridge to the desktop
binary. Its reader already separates format intake from rendering: a format-specific parser
produces an intermediate section/block structure, and a shared renderer, language detector
and TOC builder consume it. PDF and EPUB each have their own client-side parser today; the
consuming pipeline is format-agnostic. The problem cannot be solved as-is only because there
is no intake path for the other formats - the rendering half already exists and is reused.

The desktop app cannot be leaned on: its MOBI/AZW3 support shells out to Calibre and its
preferred PDF path uses a native binary, neither of which exists in a browser. So the
extension must gain independent client-side intake for each new format.

## 5. Proposed approach

Add one intake path per new format that terminates in the existing shared renderer. Each new
parser's only job is to turn source bytes (from the file picker or an intercepted URL) into
the intermediate structure the renderer already accepts; the reader, theming, TOC, language
detection and OCR layers are reused unchanged.

Three intake families, by source difficulty:

- **Trivial text sources** (TXT, HTML, Markdown): parsed in-browser with built-in platform
  capabilities, plus a small vendored converter for Markdown.
- **Self-contained structured sources** (FB2 single-file XML; RTF rich text): handled by a
  vendored ebook reader module (FB2) and a ported text extractor (RTF).
- **Binary ebooks** (MOBI, AZW3): the hard case, handled by a vendored pure-JS ebook library
  that reads these formats directly, replacing the desktop's Calibre dependency.

### 5.1 Pillars / modules

- **Intake dispatch:** recognise the format from the chosen file or URL and route to the
  right parser. File-picker intake accepts the full set; URL interception stays limited to
  true document formats so ordinary browsing (plain .txt / .html pages) is not hijacked.
- **Format parsers:** one per new format, each emitting the shared intermediate structure
  (sections/blocks, title, images, and a TOC when the source carries one).
- **Vendoring & packaging:** the new third-party reader/converter code is bundled into the
  extension package so it ships offline and passes store review (no remote code).
- **Cross-surface truth:** the supported-format list appears in many places (store listing,
  privacy copy, READMEs, website, file associations, changelog); all must move together.

### 5.2 Data & event flows

File picker or intercepted URL -> intake dispatch -> format parser -> intermediate
structure -> shared renderer (reflow, theme, TOC, language, lazy image OCR) -> reader tab.
No network, no persisted state beyond the existing viewer preferences.

### 5.3 Extension points

- The intermediate structure contract between parser and renderer is the single seam; new
  formats plug in only there.
- The vendoring step stays declarative so future formats (CBZ, DOCX) can be added the same
  way.
- The supported-format list should have as few sources of truth as practical, to keep future
  additions from drifting across docs.

## 6. Open questions / research items

1. **Minimal vendored module set for the ebook library**
   - **Question:** which exact modules and transitive helpers must be vendored for MOBI/AZW3
     (and optionally FB2) without pulling the whole library?
   - **To find out:** read the library's module imports; vendor the minimal set; test on real
     files.
   - **Status:** Open (resolve during /spec-tech - not blocking approval).
2. **FB2 via the ebook library vs a small port**
   - **Question:** does the vendored ebook reader or a direct port of the small Go FB2 parser
     give better fidelity for embedded images and nested sections at lower vendored weight?
   - **Status:** Resolved - DOMParser port (mirrors the pure-Go `internal/fb2`), FB2 in the
     simple phase; foliate-js reserved for MOBI/AZW3 only. Lower vendored weight and FB2 ships
     in the low-risk phase. Embedded `<binary>` images inline as `data:` URLs.
3. **Large-file pagination**
   - **Question:** do big TXT/MOBI outputs need the desktop's page-splitting behaviour to
     stay responsive and translation-friendly, or is the current single-document render
     enough?
   - **Status:** Resolved - reuse the single continuous-scroll render (as EPUB does); parsers
     emit multiple sections where natural (TXT/RTF chunked at 30 paragraphs, ebook at its own
     section breaks). No separate page-splitting pass.

## 7. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|:----------:|--------|-----------|
| Vendored ebook library is self-described "not stable" | Med | MOBI/AZW3 parsing fails on some files | Pin a specific version/commit; vendor minimal modules; gate on manual acceptance over a real sample set; ebook formats are the last phase, so simple formats ship regardless |
| URL interception over-reaches (plain .txt / .html) | Med | Breaks normal browsing | Restrict interception to true document formats; rely on the file picker for ambiguous text types |
| Store rename affects listing identity / SEO | Low | Temporary discoverability dip | Owner curates the new name/copy at release; retain prior keywords |
| Bundle size growth from vendored parsers | Low | Larger package | Precedent (PDF.js, Tesseract) shows size is acceptable for a local extension; vendor minimal modules |
| Format fidelity below desktop (RTF stripper, MD edge cases) | Med | Some documents render imperfectly | Behavioural-parity bar, not byte-parity; cover with sample-file tests; document known limits |

## 8. User impact (docs)

Yes - new capability. The extension gains TXT, Markdown, FB2, RTF, local HTML, MOBI and
AZW3. The store listing, privacy copy, READMEs and website format lists must be updated and
file associations extended before the next publication; the changelog records the addition.

## 9. Architecture decisions (ADR)

**ADR-1: Client-side JS parsers, not Go->WASM or native messaging.**
- **Decision:** reimplement/vendor format intake in JS feeding the existing renderer.
- **Alternatives:** (a) compile the pure-Go extractors to WASM; (b) native-messaging bridge
  to the installed desktop binary.
- **Why:** matches the extension's established pattern (PDF/EPUB are already JS ports tracked
  in `docs/PARITY.md`); Go->WASM fights the desktop pipeline's filesystem coupling and still
  cannot cover PDF's native path or MOBI's Calibre dependency; native messaging breaks the
  "install and it works" value and cross-browser reach.

**ADR-2: Vendor a pure-JS ebook library for MOBI/AZW3.**
- **Decision:** use a maintained MIT pure-JS ebook reader (foliate-js) for the binary ebook
  formats.
- **Alternatives:** bundle a WASM Calibre-equivalent; exclude MOBI/AZW3.
- **Why:** it reads MOBI and KF8/AZW3 directly in the browser, fits the no-bundler ES-module
  setup, and is the only practical way to drop the Calibre dependency client-side.
  Instability risk is mitigated by pinning and by phasing it last.

## 10. Links to other specs

- [`2026-07-01_cross-edition-parity`](2026-07-01_cross-edition-parity.md) - shared-invariant
  alignment; this ticket updates the parity mapping for the new formats.
- [`DEV/research/extension_formats_feasibility_ru.md`](../research/extension_formats_feasibility_ru.md)
  - feasibility study feeding this spec.

## 11. Done criteria (strategic)

1. From the extension, a user can open a TXT, Markdown, FB2, RTF, local HTML, MOBI and AZW3
   file and read it reflowed, the same way PDF/EPUB work today.
2. A source-provided table of contents appears for formats that carry one (FB2, MOBI, AZW3);
   language detection and image OCR behave as they do for EPUB.
3. Everything works offline with no desktop app, Calibre, or network dependency.
4. The store listing, privacy copy, READMEs, website and file associations all state the
   wider format range, and the changelog records it, so the next publication is correct.
5. The JS<->Go parity mapping in `docs/PARITY.md` records the new formats.

## 12. Next step

`/spec-tech 2026-07-01_extension-format-parity` - creates the phased tactical plan.
