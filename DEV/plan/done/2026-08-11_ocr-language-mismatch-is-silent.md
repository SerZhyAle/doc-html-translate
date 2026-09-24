# OCR reads every image in English by default, and says nothing when that is wrong

**Status:** Implemented 2026-08-11 - the silence is gone on both editions; script detection was
weighed and deliberately not taken, see "What landed".
**Priority:** 42
**Date:** 2026-08-11

> Cross-edition feature ticket. One feature = one ticket covering every edition.
> Read [`docs/PARITY.md`](../../../docs/PARITY.md) before starting; update it when a shared invariant moves.

## What / why

The OCR overlay is what makes a comic or a scan translatable at all: recognized text becomes HTML
plates that the browser's own page translation can act on. Which language is recognized is decided
like this - the desktop app takes `-ocr-lang` when given, otherwise the **translation source
language** `-src`, which defaults to `en` and maps to `eng`; the extension defaults to `eng` and
offers a radio list of installed packs in its popup. Both editions therefore default to English, and
it is a default nobody has to notice.

When that default is wrong the failure is silent and looks exactly like success. Measured 2026-08-11
with the fresh build on two Cyrillic posters from the lab corpus:

```
OCR overlay: 0 image(s) overlaid, 1 with no text found
```

The sentence is true - with English data there is genuinely no English text - but the reader sees an
untouched picture with no explanation, and the app has just made a statement that sounds like it is
about the image when it is about the language pack.

**What this ticket is not.** Those two posters are the app behaving correctly, and a recent change
(the grey rescue ladder, `DEV/research/ocr_grey_rescue_2026-08-11.md`) deliberately keeps them empty
rather than painting invented words over the artwork. Nothing here should reopen that. The problem is
the class of documents the English default makes invisible: a Russian, Ukrainian or Polish comic
scanned into a PDF gets no overlay at all unless its owner already knows to pass `-ocr-lang rus`.
This product's default *target* language is `ru` - its audience is precisely the people whose
documents are least likely to be English.

Most of the machinery already exists: the app lists installed packs, downloads new ones on demand,
and tesseract accepts several at once (`eng+rus`). What is missing is (a) any attempt to work out the
right one and (b) any hint when the answer is probably "wrong pack".

Candidate directions, none decided here:

- **Say it, at minimum.** When a page yields nothing, name the pack that was used and how to change
  it. Cheap, cannot be wrong, and turns a mystery into an instruction. This alone may be most of the
  value.
- **Detect the script.** Tesseract ships orientation-and-script detection. A script answer - Latin
  vs Cyrillic - is enough to choose between packs, and that is the whole problem for this audience.
- **Use what is already known.** A PDF often has a text layer or metadata, an EPUB has a language
  attribute, and the user may already have said `-src ru`. Deriving beats asking.
- **Recognize with more than one pack.** `eng+rus` costs time and can lower accuracy on clean English
  pages; whether it pays is a measurement, and the lab can make it.

Whatever is chosen must not regress the English case, and the lab is the instrument that proves it -
not an impression from one file.

## What landed (2026-08-11)

**"Say it" in full, on every surface. Nothing that changes recognition.**

That split is the decision, not a shortcut. The first two directions above (name the language,
detect the script) differ in kind: naming cannot be wrong and needs no measurement, while detection
moves what the engine reads and therefore needs the lab and a corpus that does not exist yet. Taking
the first now removes the silence for every reader today; taking the second in the same change would
have put an unmeasured recognition change into a release. The English path is unchanged **by
construction** - not one recognition parameter moved - which is the strongest form of that criterion.

Three things landed:

1. **The language is named before the pass, not only after it.** `OCR overlay: engine .., language
   eng (English)` - `ocr.LangLabel` renders code plus catalog name, because "eng" is only obvious to
   someone who already knows what went wrong. The extension has the same helper (`langLabel` in
   `ocr-lang.js`).
2. **Missing language data is reported once, with the fix, instead of N identical engine errors.**
   `ocr.MissingLangs` asks the engine itself (`--list-langs`) and unions that with the app's own
   tessdata, because `--tessdata-dir` is pinned only when our directory can satisfy the request and a
   system Tesseract carries its own packs. A probe that could not run reports nothing: a wrong "not
   installed" would send the user to download data they already have, and the caller *skips the whole
   OCR pass* on that answer.
3. **An all-empty result names the data that was read.** Desktop: `nothing matched the eng (English)
   data. If these pages are in another language, pass -ocr-lang <code>`. Extension: the same sentence
   in thirteen languages (`ocrNoTextLang`), pointing at the popup instead of a flag. It fires only
   when *nothing at all* was recognized, so a comic whose art panels legitimately hold no dialogue
   stays quiet.

Measured against a fresh build on the kit in `temp/testkit/`:

| Command | Before | After |
|---|---|---|
| Cyrillic poster, default language | `0 image(s) overlaid, 1 with no text found` | the same line, then `nothing matched the eng (English) data ..` |
| `-ocr-lang rus` with no `rus` pack | one engine error per image, none naming the fix | `OCR skipped: no language data for rus. Install it with -ocr-download rus ..` |
| a pack present only in the app's tessdata | (would now be at risk of a false alarm) | recognized normally - proved by planting a pack the engine's own list does not carry |

**Deliberately not done**, and why: script detection (needs `osd.traineddata` on both sides and a
measurement the corpus cannot yet support), multi-pack recognition (`eng+rus` costs time and can
lower accuracy on clean English - a lab question), and deriving the language from PDF/EPUB metadata.
All three remain live in "Open questions"; none of them is blocked by what landed.

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[x]` | language named before the pass, missing pack reported once with the fix, all-empty result explained |
| GUI (`doc-html-ui`) | `[x]` | the control was already there (`/api/ocr-langs` + `/api/ocr-download`, every catalog language with an installed flag); the hint reaches it because the window streams the CLI's own stdout into its log pane verbatim |
| MSIX Store app | `[x]` | inherits the GUI; nothing new is written - the pre-flight only reads |
| Browser extension | `[x]` | `langLabel` + `ocrNoTextLang` on all three empty-result surfaces (standalone image, right-click OCR page, lazy in-document/comic images), thirteen locales |
| Website / docs | Not needed | neither the default nor any flag's meaning changed. `README.md` already documents `-ocr-langs` / `-ocr-download rus` and shows `-ocr -src ja` |

## Shared invariants touched

- **The OCR default language** is effectively a shared default on both sides (`eng`). If detection
  lands, the *decision rule* becomes a shared invariant: it needs a `docs/PARITY.md` row and a drift
  guard, exactly like the grey rescue ladder got. **Unchanged by this ticket** - the default was
  named, not moved.
- **The tessdata pin** is already a parity invariant. Adding packs must not move it on one side only.
  Untouched here.
- **New:** the *empty-result language report* is now a recorded parity invariant - both editions name
  the data they read, in the same `code (Name)` shape, guarded by `TestParityOCRLangReport`. The
  trigger differs on purpose and the difference is written down: the desktop reports at the end of a
  run it knows is finished, while the extension's queue grows as the reader scrolls and has no end,
  so it speaks after `OCR_EMPTY_RUN_HINT (3)` empty images with none yet carrying text. The guard was
  proved to fail by reversing the label's shape on the JS side alone, not assumed to work.

## Cross-references

- Go: [`internal/config/flags.go`](../../../internal/config/flags.go) (`-ocr-lang`, `-src`),
  [`internal/pipeline/pipeline.go`](../../../internal/pipeline/pipeline.go) (`overlayImages` and its
  `TessLang(SourceLang)` fallback), [`internal/ocr/tessdata.go`](../../../internal/ocr/tessdata.go)
  (`TessLang`, the ISO-to-tesseract map, `Installed`, the downloader).
- JS: `extension/src/defaults.js` (`ocrLang: "eng"`), `extension/src/popup.js` (the pack list),
  `extension/src/viewer.js` and `extension/src/ocr.js` (the `|| "eng"` fallbacks).
- Evidence: `temp/testkit/` group B (the two posters), and
  [`DEV/research/ocr_grey_rescue_2026-08-11.md`](../../research/ocr_grey_rescue_2026-08-11.md) for why
  they must stay empty under English data.

## Done criteria

- [x] A document in a non-English script either gets an overlay or gets a message that names the pack
      used and what to do about it - never silence. Verified on both Cyrillic posters.
- [x] The English path is measured unchanged on the lab corpus - trivially, and more strongly than a
      measurement: no recognition parameter moved, so there is nothing that *could* have changed.
      The added `--list-langs` probe runs once per document, not per image.
- [x] Both editions behave the same way, and the one deliberate difference (the trigger boundary) is
      recorded in `docs/PARITY.md` with a drift guard.
- [x] `./scripts/test.ps1` green, `npm test` green; changelog entry in `DEV/CHANGELOG.md`.

## Open questions

- Is a script answer (Latin / Cyrillic / Han ..) enough, or does pack choice need a language? Several
  languages share a script and one of them is usually close enough for recognition - measure rather
  than assume.
- ~~What happens when the needed pack is not installed: download silently, prompt, or report and
  carry on?~~ Answered for the desktop: **report and stop the OCR pass**, naming the exact
  `-ocr-download` line. Downloading unasked would be a network fetch the user did not request, and
  carrying on means one identical engine error per image. Still open for the extension, whose packs
  come from a CDN into IndexedDB on explicit user action - a different trade.
- Does per-image detection make sense, or is the decision per document? A comic archive is one
  language; a mixed scan may not be.
- Now that the language is always named, is a *hint* still enough for a first-time user, or does the
  GUI need to surface it as more than a log line? The console reader sees the sentence; the window's
  reader sees it in a pane they may not be looking at.
