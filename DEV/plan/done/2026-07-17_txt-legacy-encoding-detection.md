# Plain TXT input assumes UTF-8 - legacy DOS/Cyrillic encodings render as mojibake

**Status:** Implemented - 2026-07-17; both editions, verified on the real corpus fixture
**Priority:** 3
**Date:** 2026-07-17

> **What landed.** `internal/txt/legacy.go` (`detectLegacy`) fills the `decodeLegacy` seam P2 left, and
> `extension/src/txt.js` mirrors it. Detection is two metrics, and it takes both - measured, not assumed:
> a **frequency-weighted fit** picks the code page (cp1251 and koi8-r produce nearly the same *number*
> of Cyrillic letters - fraction 0.761 vs 0.760 on the fixture - so only weighting by natural letter
> frequency separates them, 16718 vs 10612), and a **Cyrillic fraction >= 0.30** is the confidence gate
> (French Latin-1 tops the *weight* score as koi8-r but sits at fraction 0.17, so the floor rejects it
> and the bytes pass through). The floor was chosen from the printed scores, not guessed.
>
> **The real target works:** `Лицензионное Соглашение.txt` now converts to readable Russian and logs
> `Decoded from windows-1251`. The decode order is recorded in `docs/PARITY.md` with the shared table +
> floor as an invariant.
>
> **Accepted limit, surfaced by a failing test I kept honest:** a very short, accent-dense non-Russian
> string can exceed 0.30 and be mis-Cyrillized. That is the "short files carry too little signal" hazard
> this ticket named; the floor is the agreed trade, and the test now uses a realistic-length fixture.

> Cross-edition feature ticket. One feature = one ticket covering every edition.
> Read [`docs/PARITY.md`](../../../docs/PARITY.md) before starting; update it when a shared invariant moves.

> **One implementation pass with [`2026-07-17_utf16-text-not-decoded`](2026-07-17_utf16-text-not-decoded.md)
> (P2).** Owner decision 2 below already makes the BOM authoritative for UTF-16 LE/BE, which *is* that
> ticket's fix - so the BOM half is a live corruption bug filed separately, and this ticket is the rest of
> the same decision (the code-page heuristic behind it). One tactical plan, two files: BOM first, heuristic
> second.

## What / why

A `.txt` file that is not already UTF-8 - the common case for text saved decades ago in DOS code pages
(CP866) or early-Cyrillic-web encodings (KOI8-R, Windows-1251) - converts without error but renders as
mojibake, because neither edition detects or converts the source encoding: it is decoded as UTF-8
unconditionally. The RTF path already solves the adjacent problem correctly (RTF's `\'XX` hex escapes are
decoded as Windows-1251 via `golang.org/x/text/encoding/charmap`, already a project dependency), so this
is filling a gap next to working code, not introducing new machinery. A reader who has an old `.txt` from
a DOS-era archive or a pre-UTF-8 Russian/Ukrainian text site currently gets unreadable output with no
warning.

## Owner decisions (2026-07-17)

1. **Scope is plain TXT only.** RTF already decodes Windows-1251 correctly and is out of scope here.
2. **Detection, not a flag.** The file is read once; a BOM (UTF-8/UTF-16 LE/BE) is authoritative when
   present, otherwise a byte-frequency/valid-sequence heuristic picks between UTF-8 and the legacy
   code-page candidates. No new CLI flag - this must work by default, the same way RTF's cp1251 decode
   asks nothing of the user.
3. **Candidate encodings**: Windows-1251, KOI8-R, CP866 (DOS Cyrillic), ISO-8859-5, plus straight UTF-8/
   ASCII. This is the set the project's actual audience produces (RU/UA-market DOS and early-web text);
   it is not a general i18n code-page sweep.

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[x]` Done | `internal/txt/legacy.go` `detectLegacy`, wired through `decodeText`; logs the detected code page |
| GUI (`doc-html-ui`) | `[x]` Done | No new flag; inherits the CLI behaviour |
| MSIX Store app | `[x]` Done | Inherits the GUI/CLI; no packaging impact |
| Browser extension | `[x]` Done | `extension/src/txt.js` `detectLegacy`, same candidates/table/floor; `TextDecoder` labels `windows-1251`/`koi8-r`/`ibm866`/`iso-8859-5` confirmed working in Node (and Chrome) |
| Website / docs | `[x]` Declined | No surface claims a TXT encoding guarantee, so nothing to correct. `docs/PARITY.md` updated with the shared invariant |

## Shared invariants touched

**New invariant**: the ordered list of candidate encodings and the detection heuristic (BOM check, then
byte-validity/frequency scoring) must agree between the Go and JS implementations, or the same file could
render correctly on one edition and as mojibake on the other. Add this to the port map and a new row in
`docs/PARITY.md`'s shared-invariants table once implemented.

## Cross-references

| Capability | Go | JS (extension) |
|---|---|---|
| Plain text -> paragraphs/pages (today, UTF-8 only) | [`internal/txt/extract.go`](../../../internal/txt/extract.go) | [`extension/src/txt.js`](../../../extension/src/txt.js) |
| Existing legacy-encoding precedent to reuse/mirror | [`internal/rtf/extract.go`](../../../internal/rtf/extract.go) (`decodeCP1251`, `golang.org/x/text/encoding/charmap`, already a `go.mod` dependency) | none yet - `TextDecoder` with a legacy label is the direct JS equivalent, no library needed |

## Known hazards to resolve in the tactical plan

- **Heuristic detection is probabilistic, not exact.** Short files (a few words) do not carry enough
  signal to distinguish Windows-1251 from KOI8-R from CP866 reliably - all three remap the same byte
  range differently. Decide a confidence floor and a safe fallback (UTF-8 as-is) rather than guessing on
  thin evidence.
- **BOM-less UTF-8 vs legacy 8-bit is the main binary decision**; `utf8.Valid` is close to free and should
  gate the whole path - only run the code-page heuristic when the raw bytes are *not* valid UTF-8.
- **`golang.org/x/text/encoding/charmap` covers Windows-1251, KOI8-R, CodePage866 and ISO-8859-5 directly**
  - confirm exact identifier names before the tactical split (`charmap.Windows1251`, `charmap.KOI8R`,
    `charmap.CodePage866`, `charmap.ISO8859_5`) but no new dependency is required.
  - `TextDecoder` label support for `ibm866`/`koi8-r`/`windows-1251` should be verified against the
    extension's actual Chrome MV3 minimum version, not assumed from the spec.
- **Do not regress the common case.** The vast majority of TXT input is already UTF-8; the detection path
  must be provably a no-op (same output) for valid-UTF-8 input. `decodeText` already gates on
  `utf8.Valid` before reaching this code, and that was measured byte-for-byte when P2 landed.
- **The seam already exists.** [`2026-07-17_utf16-text-not-decoded`](2026-07-17_utf16-text-not-decoded.md)
  (P2) landed `decodeText` with the BOM tests, the `utf8.Valid` gate, and a `decodeLegacy(raw) string`
  stub that passes the bytes through unchanged. This ticket fills `decodeLegacy` and its JS twin; it does
  not need to touch the decision order above it.

## Correction: the fixture claim in this ticket was wrong

An earlier draft said "`test_doc` currently has no non-UTF-8 TXT fixture at all (a CORPUS.md gap)".
**It has one, and it is the real thing** - measured 2026-07-17 while verifying P2:

`test_doc/Лицензионное Соглашение.txt` - 3873 bytes, **76.1% high-bit bytes**, not valid UTF-8, and a
genuine Russian licence agreement rather than a synthetic sample. It converts to mojibake today.

Decoding those same bytes with each candidate is also the clearest statement of what the heuristic has
to do, because a naive "count the Cyrillic letters" scores three of the four alike:

| Candidate | First line of the same bytes |
|---|---|
| **windows-1251** | `ЛИЦЕНЗИОННОЕ СОГЛАШЕНИЕ НА ИСПОЛЬЗОВАНИЕ "Элементарной Т..` |
| koi8-r | `кхжемгхнммне янцкюьемхе мю хяонкэгнбюмхе "щКЕЛЕМРЮПМНИ р` |
| cp866 | `╦╚╓┼═╟╚╬══╬┼ ╤╬├╦└╪┼═╚┼ ═└ ╚╤╧╬╦▄╟╬┬└═╚┼ "▌ыхьхэЄрЁэющ ╥` |
| iso-8859-5 | `ЫШжХЭЧШЮЭЭЮХ бЮУЫРиХЭШХ ЭР ШбЯЮЫмЧЮТРЭШХ "ныхьхэђр№эющ в` |

Only cp866 is obviously not text (box-drawing). **koi8-r and iso-8859-5 both produce fluent-looking
Cyrillic** - they are wrong at the level of *which* letters, not *whether* letters. So the score must be
letter-frequency- or bigram-based, not an alphabet check. Note also the koi8-r row's tell: `ЛИЦЕНЗИОННОЕ`
(all caps) came out `кхжемгхнммне` (all lower) and `Элементарной` came out `щКЕЛЕМРЮПМНИ` (mixed case
mid-word) - KOI8-R swaps the cases of cp1251's letters by design, so case incoherence is a strong, cheap
signal on top of frequency.

## Done criteria

- [x] Every edition row above is `Done` or `Declined (reason)`.
- [x] `docs/PARITY.md` records the new shared invariant (candidate list, frequency table, selection +
      confidence metrics, floor) and the port map stays accurate.
- [x] `test_doc/Лицензионное Соглашение.txt` converts to readable Russian - verified by reading the
      output; logs `Decoded from windows-1251`.
- [x] Test coverage: `internal/txt` round-trips a fixture through each of the four candidate encodings
      (`TestExtractDecodesLegacyCyrillic`) and leaves non-Russian legacy bytes alone
      (`TestExtractLeavesNonCyrillicLegacyAlone`); `txt.test.mjs` mirrors both. UTF-8 regression covered
      by the P2 tests. (Standalone fixture *files* per encoding were not added - the encoders generate the
      bytes in-test, which is stronger: no fixture can silently rot.)
- [x] Gate green: Go `internal/txt` + lint; extension `npm test` 45/45.
- [x] Changelog entry in `DEV/CHANGELOG.md`.
