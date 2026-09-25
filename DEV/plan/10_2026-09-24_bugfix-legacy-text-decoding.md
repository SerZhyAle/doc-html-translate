# Strategic spec: 10_2026-09-24_bugfix-legacy-text-decoding - Decode RTF, FB2 and TXT text correctly in both editions

**Ticket:** 10_2026-09-24_bugfix-legacy-text-decoding
**Status:** Implemented
**Priority:** 80
**Date:** 2026-09-24
**Tier:** Moderate
**Tactical plan:** `DEV/plan/10_2026-09-24_bugfix-legacy-text-decoding/` (created by /spec-tech)
**Findings:** X1 X2 X3 X4 X5 X18 X19 X21 X22 P23 B21 (see the [findings register](../research/audit_2026-09-24/README.md))

> **Scope:** STRATEGIC.

---

## 1. Problem
The text-format readers lose or garble text on inputs that are common in the product's core audience.
- **RTF:**
  - Unicode escapes produce NUL in the desktop edition and doubled letters in the extension.
  - Font tables, colour tables, document info and embedded pictures appear as body text: every WordPad file opens with a paragraph of font names, and a picture becomes megabytes of hex.
  - Code-page bytes are always decoded as Cyrillic, whatever the file declares.
- **FB2:** a Russian FB2 declared as windows-1251, a very common case, fails to open at all. Verse, subtitles and tables are dropped.
- **TXT:**
  - A UTF-8 file with a single damaged byte is decoded as a legacy code page, so the whole book is mojibake.
  - UTF-16 without a byte-order mark comes through with NUL characters.
  - Western legacy files are written raw into pages that declare UTF-8.

## 2. Goals
1. RTF body text comes out exactly as a word processor shows it: Unicode escapes with their fallback rule, destinations skipped, binary data skipped, control symbols mapped, and the declared code page honoured.
2. FB2 in any encoding its XML declaration names converts, and all prose-level elements (verse, subtitles, epigraph authors, table cells) are kept.
3. A mostly valid UTF-8 TXT stays UTF-8, and only the damaged bytes become replacement characters.
4. BOM-less UTF-16 TXT is detected and decoded.
5. No page is ever written with bytes that are invalid for its declared charset; damage is visible as a replacement character, never silently dropped.
6. FB2 image names never collide, and a pathological id cannot fail the book.
7. Both editions behave identically on a shared fixture set, pinned by the parity test.

**Non-goals:**
- Supporting RTF layout (tables and columns) beyond text flow.

## 3. Wishes and constraints
### 3.2 Hard constraints
- **Platform / versions:** the Go edition and the JS extension; the extension's decoding is limited to what the browser provides (TextDecoder labels).
- **Performance:** RTF must not create a decoder per byte.
- **Data compatibility:** n/a.
- **Localization:** n/a.

### 3.3 Owner inputs (Approval gate)
- **Related tickets:** none blocking.
- **Platform constraints:** cross-edition, one ticket. docs/PARITY.md gains the decoding rules, including removing the documented "UTF-8 only" gap for FB2 if the extension can now follow it.
- **Validation level:** a fixture corpus with WordPad RTF, LibreOffice RTF with `\uc`, cp1252 RTF, a cp1251 FB2 with a poem, truncated UTF-8 TXT, BOM-less UTF-16 TXT, and a Latin-1 TXT; the Go tests and JS tests share the expected outputs.
- **Owner sign-off:** decide the damaged-UTF-8 threshold (§6.1).

## 4. Current architecture context
The RTF reader is a flat scanner with no group state. It consumes a control word's parameter, then
tries to read it again. The FB2 reader uses the standard XML decoder with no charset hook, and
collects only paragraph elements. The TXT reader treats any UTF-8 invalidity as proof of a legacy
code page, and scores Cyrillic letter frequency. The extension edition hand-ports the same
heuristics with its own differences.

## 5. Proposed approach
### 5.1 Pillars / modules
- **RTF reader with group state:** a destination stack, the `\ucN` skip count, signed Unicode values, `\binN` skipping, control-symbol mapping, and code-page selection from `\ansicpg` and per-font charsets.
- **FB2 charset hook:** encodings declared in the XML are resolved through a label lookup, and the prose element set is widened.
- **TXT decision ladder:** BOM, then a UTF-16 NUL-pattern check, then "UTF-8 with damage below threshold", then legacy detection, then a Western fallback.
- **Output charset guarantee:** a final validity pass maps invalid sequences to U+FFFD, never to nothing.
- **FB2 asset naming:** a collision-safe unique name per binary.
- **Shared fixtures:** one set of inputs with expected text, consumed by both editions' tests.

### 5.2 Data & event flows
Bytes -> encoding decision -> decoded text -> structure extraction -> paragraphs -> HTML.

## 6. Open questions / research items
1. **Damaged-UTF-8 threshold**
   - **Question:** what share of invalid bytes still counts as UTF-8?
   - **Options:** under 1% of multi-byte sequences; only a truncated tail; the valid-sequence ratio.
   - **To find out:** run it against the local test_doc corpus.
   - **Status:** Decided (owner, 2026-09-25): the input is UTF-8 when it holds at least one valid multi-byte sequence and its invalid bytes are under 1% of those sequences, OR when the only invalidity is a sequence truncated in the last 3 bytes. Invalid bytes become U+FFFD. Counting follows the WHATWG UTF-8 decoder so both editions agree. The local `test_doc/` corpus is not present in the implementation environment, so the rule is validated on the shared fixtures and unit cases only (truncated tail, one bad byte in a Russian book, 20% damage, Latin-1, cp1251).
2. **Extension encoding coverage**
   - **Question:** can the extension decode windows-1251 FB2 via TextDecoder?
   - **To find out:** check TextDecoder label support in Chrome/Edge.
   - **Status:** Decided (owner, 2026-09-25): yes. Chrome and Edge implement the WHATWG Encoding Standard labels (windows-1251, koi8-r, windows-1252, iso-8859-x, ..), so the extension follows the FB2 XML-declared encoding too. The "UTF-8 only" FB2 gap is removed from docs/PARITY.md and the decoding rules are shared invariants there.

## 7. Risks
- **Destination skipping hides real text in unusual RTF generators.** Likelihood: low. Impact: text missing. Mitigation: skip only known destinations and `\*`-prefixed ones.
- **The TXT heuristic change misclassifies true cp1251 files.** Likelihood: low. Impact: mojibake in legacy files. Mitigation: validate against the corpus.

## 8. User impact (docs)
No changes to user docs.

## 9. Architecture decisions (ADR)
No ADRs. The decision follows established project patterns (hand-ported logic pinned by the parity test).

## 10. Links to other specs
None.

## 11. Done criteria (strategic)
1. A WordPad RTF in Russian opens with no font-table paragraph and every letter correct, in both editions.
2. A windows-1251 FB2 with a poem converts, and the poem is present.
3. A Russian UTF-8 TXT with its last byte cut off reads correctly, apart from one replacement character.
4. A French Latin-1 TXT shows accented letters correctly.

## 12. Next step
`/spec-check 10_2026-09-24_bugfix-legacy-text-decoding` (implemented directly from the owner decisions; no tactical plan directory was created).

## Implementation

Implemented 2026-09-25 in both editions. Also takes finding X20 (FB2 peak memory, filed under ticket 12) because the FB2 reader was rewritten here anyway.

**Shared (Go):** `internal/textutil/utf8.go` - `MeasureUTF8` and `DecodeUTF8`, the WHATWG UTF-8 decoder (one U+FFFD per maximal invalid subpart, the same as `TextDecoder`). `internal/textutil/codec.go` - `LookupCodec`, legacy decoding by WHATWG label through `x/text/encoding/htmlindex`, with the browser's C1 mapping for undefined single-byte positions and a streaming reader. `internal/textutil/lines.go` - P23: invalid UTF-8 from `pdftotext` becomes U+FFFD instead of being dropped.

**TXT (X5, X21, X22):** the ladder BOM -> BOM-less UTF-16 by NUL pattern -> UTF-8 below the damage threshold -> Cyrillic detection -> windows-1252. Go: `internal/txt/extract.go` `decodeText`, `internal/txt/decode.go` (`sniffUTF16`, `acceptAsUTF8`), `internal/txt/legacy.go` (candidates as WHATWG labels, `westernLabel`); `LooksBinary` accepts BOM-less UTF-16. JS: `extension/src/txt.js` (`sniffUtf16`, `measureUtf8`, `acceptAsUtf8`, `decodeText`).

**RTF (X1, X2, X3, X22, B21):** a group-state reader with a destination stack. Go: `internal/rtf/parse.go` (reader), `internal/rtf/codepage.go` (code-page, charset, destination, symbol and break tables); `internal/rtf/extract.go` keeps paging. JS: `extension/src/rtf.js` is the same reader with the same tables. `\ucN` fallback skipping ends the B21 doubled letters; `\'XX` and raw high bytes are batched and decoded by one decoder per code page per document.

**FB2 (X4, X18, X19, X20):** Go `internal/fb2/content.go` - `decodingReader` (BOM, XML-declared encoding via `LookupCodec`, else validating UTF-8), a single streaming pass `parseFB2` over the open file (title, prose, binaries; the second `xml.Unmarshal` pass and the whole-file read are gone), and `readBinary`, which decodes the base64 in 16 KB chunks straight into the output instead of copying the text three times. Prose elements: `p`, `subtitle`, `text-author`, `td`/`th`, and each `stanza` as one `p.stanza` with `<br>` between verse lines. `internal/fb2/extract.go` - collision-safe image names (`nameSet.imageFileName`: a sanitized id keeps its name when it was already safe, otherwise gains an FNV-32a hash of the full id; leading dots are stripped; uniqueness is case-insensitive with a numbered fallback), page CSS for the three classes. JS: `extension/src/fb2.js` - `decodeFb2` (the same encoding rule through `TextDecoder`) and `renderBlock` (the same element set and classes), a body's leading title/epigraph kept on its own page; `extension/src/viewer.css` styles the classes.

**Docs:** `docs/PARITY.md` - the TXT ladder table, the damage threshold and output rule, the WHATWG-label rule for code pages, a new "RTF and FB2 text decoding" section, the port-map RTF row renamed; the FB2 "UTF-8 only" scope note removed.

**Fixtures:** `tests/testdata/legacy-text/` - WordPad Russian RTF (`\ansicpg1252` plus a `\fcharset204` font), LibreOffice RTF (`\uc1`/`\uc2`, `\'XX` fallbacks, a surrogate pair, a `\bin` picture with braces), cp1252 RTF, windows-1251 FB2 with a poem, truncated-tail UTF-8 TXT, BOM-less UTF-16LE TXT, Latin-1 French TXT, each with an expected text; generated deterministically by `gen.go` (`go run tests/testdata/legacy-text/gen.go`), listed in `cases.json`, `-text` in `.gitattributes`. Against the pre-change extension readers the JS fixture test fails 7 of 7.

**Done criteria -> proof:**
1. WordPad Russian RTF, no font-table paragraph, every letter correct, both editions: fixture `wordpad-ru` in `tests/legacy_text_test.go` `TestLegacyTextFixtures` and `extension/test/legacy-text.test.mjs`; unit cases in `internal/rtf/parse_test.go` `TestStripRTF` and `extension/test/rtf.test.mjs` (same case list).
2. windows-1251 FB2 with a poem converts, poem present: fixture `cp1251-poem`; `internal/fb2/decode_test.go` `TestParseFB2DeclaredWindows1251`, `TestParseFB2ProseElements`; `extension/test/fb2.test.mjs`.
3. Russian UTF-8 TXT with the last byte cut off, one U+FFFD: fixture `utf8-truncated`; `internal/txt/decode_test.go` `TestDecodeTextToleratesDamagedUTF8`; `extension/test/txt.test.mjs`.
4. French Latin-1 TXT with correct accents: fixture `latin1-fr`; `internal/txt/extract_test.go` `TestExtractDecodesWesternLegacy`.

Goals 4-7: BOM-less UTF-16 (`utf16le-nobom`, `TestDecodeTextBOMlessUTF16`); valid output (`TestDecodeTextOutputAlwaysValid`, `internal/textutil` tests); image names (`TestImageFileNamesAreUnique`, `TestExtractSurvivesPathologicalImageID`); parity (`TestParityLegacyTextTables` compares the tables and constants value by value). Performance: `TestStripRTFLargeInput` (about 5 MB of `\'XX`-dense RTF) and `BenchmarkStripRTF` (about 100 MB/s), with a matching large-input test in `extension/test/rtf.test.mjs`; X20: `TestReadBinaryAcrossChunks`.

Not validated: the local `test_doc/` corpus (absent in the implementation environment).
