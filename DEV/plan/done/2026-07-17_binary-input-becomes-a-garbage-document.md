# A binary input becomes a multi-megabyte garbage document, reported as success

**Status:** Implemented - 2026-07-17; all 12 corpus binaries refuse, all texts (incl. UTF-16) still convert
**Priority:** 4
**Date:** 2026-07-17

> **What landed.** `internal/txt/sniff.go` `LooksBinary(head)` guards the `default:` arm of the
> `switch ext` in `pipeline.go`: it checks a BOM first (so UTF-16, full of NULs, is not refused), then
> named signatures (ZIP/RAR/7z/tar-at-257/DjVu, plus PDF/MOBI/image defensively), then a NUL byte in the
> first 4 KB as the catch-all. A match returns a non-zero exit, names what the file looks like, and
> leaves no output directory. The misleading `treating as plain text...` line (long dash + ellipsis) is
> gone, replaced by a clean `Unknown extension %q - reading as plain text..`.
>
> **Verified by running the corpus, not by reasoning about the sniff** (done criterion): all 12 binary
> fixtures - the real `.docx`, `.odt`, `.pptx`, `.xlsx`, both `.djvu`, three `.cbz`, `.cbr`, `.cb7`,
> `.cbt` - exit 3, name the format, and leave no book directory; all five text fixtures (plain, real
> cp1251, UTF-16LE/BE, UTF-8-BOM) still produce `index.html` at exit 0. The first harness lied the usual
> way (`-folder` is a *parent*; the book lands in a `<folder>/<name>` subdir), reporting `dirLeft=True`
> for the empty parent and `index.html=False` at the wrong path; corrected to check the real output dir.
>
> **Extension confirmed correct by tracing, not assuming** (parity criterion): `detectFormat` routes any
> `PK..` to the EPUB reader (CBZ -> "not a valid EPUB") and everything unrecognized - DjVu, RAR, 7z, tar
> - to `loadPdfData`, whose `catch` calls `handleLoadError`; both are clear notices, not garbage. The
> CBZ-as-EPUB *message* is misleading, but that is P12's concern (making comics work), not P4's (not
> making garbage). `docs/PARITY.md` now records signature-first detection as a shared invariant.

> Cross-edition ticket. The browser extension **already does this correctly** - this is a divergence
> to close on the Go side, not a feature to design twice. Read [`docs/PARITY.md`](../../../docs/PARITY.md).
> Split out of [`2026-07-17_cli-diagnostics-honesty`](2026-07-17_cli-diagnostics-honesty.md), where it
> was item 4 and a prediction; it is now measured across eight extensions.

## What / why

[`internal/pipeline/pipeline.go`](../../../internal/pipeline/pipeline.go) dispatches on the **file
extension alone**. Nine extensions are handled; everything else hits `default:` and is handed to
`internal/txt`:

```go
default:
    // Unknown extension — treat as plain text.
    logging.Printf("[1/4] Unknown format %q — treating as plain text...\n", ext)
    book, err = txt.Extract(inputPath, outputDir)
```

For a stray `.log` that is the friendly choice. For a **binary** it produces a document made of the
file's raw bytes rendered as prose, and reports success. Measured 2026-07-17, every row `exit=0`:

| Input | index.html | "Pages" | Prose in index.html | Control chars | First thing the reader sees |
|---|---|---|---|---|---|
| `.docx` (a real tax-opinion draft) | 576 KB | 15 | 472,879 | 18.8% | `PK..` |
| `.odt` | 17 KB | 2 | 1,099 | 17.7% | `PK..mimetypeapplication/vnd.oasis.opendocument.text` |
| `.pptx` | 39 KB | 1 | 23,168 | 23.9% | `PK..[Content_Types].xml` |
| `.xlsx` | 66 KB | 2 | 39,392 | 20.9% | `PK..[Content_Types].xml` |
| `.djvu` (tiny) | 123 KB | 3 | 90,881 | 19.4% | `AT&TFORM DJVMDIRM` |
| `.djvu` (mid) | 4.0 MB | 94 | - | - | `AT&TFORM DJVMDIRM` |
| `.cbz` | 5.1 MB | 100 | 4.9M | - | ZIP header + first JPEG's name |
| `.cbr` | **17 MB** | **363** | **14,301,891** | 18.4% | `Rar!..Plastic Man 17-01.jpg` |
| `.cb7` | 5.2 MB | 95 | - | - | `7z..'` |
| `.cbt` | 5.2 MB | 94 | - | 72.1% | the first entry's filename (tar) |

Real prose carries ~0% control characters. These carry 18-30%, and 72% for the uncompressed tar.

`.docx` is the one that matters most: it is the most common document format in the world, it is not in
the switch, and dragging one onto the app yields 576 KB of ZIP noise under a `Done.` The log *does*
announce the fallback (`Unknown format ".docx" — treating as plain text...`), so this is not silent -
but the warning describes the **mechanism**, and a reader who accepts "treating as plain text" is
told nothing about the **outcome** being nonsense. It then says `Paragraphs: 431 → Pages: 15`, which
is a confident lie about content that does not exist.

Supporting these formats is a separate question (`.docx` would be worth its own ticket; comic archives
already have [`2026-07-17_comic-archives`](../2026-07-17_comic-archives.md)). This ticket is narrower and
holds regardless of the roadmap: **feeding a binary to a text extractor must not produce a document
and a success exit.**

**Free fix while in there.** Both quotes above are verbatim, long dash and all: `pipeline.go:181-182`
uses a long dash and a three-dot ellipsis, and line 182 is *generated output* - which `CLAUDE.md`'s
typography rule explicitly covers ("short hyphens, .. not ..."). Whatever replaces this dispatch should
land on project typography rather than carry the violation forward.

## The extension already solves this - port its rule, do not invent one

`extension/src/viewer.js` routes on **byte signature first**, extension second:

```js
function detectFormat(data, name) {
  const b = new Uint8Array(data, 0, Math.min(12, data.byteLength));
  if (b[0] === 0x25 && b[1] === 0x50 && b[2] === 0x44 && b[3] === 0x46) return "pdf";   // %PDF
  if (b[0] === 0x50 && b[1] === 0x4b && b[2] === 0x03 && b[3] === 0x04) return "epub";  // PK..
  ...
  return FORMAT_EXT[fileExt(name)] || "unknown";
}
```

Unknown falls through to the PDF reader, "whose error handling reports an unreadable file clearly".
Every fixture in the table above would be caught there: the ZIP family as `epub` (failing with a real
error), DjVu/RAR/7z as `unknown` -> PDF -> a clear failure. So the correct behaviour already ships in
one edition and the Go app is the outlier. `docs/PARITY.md` should record byte-signature detection as
a shared invariant once this lands.

## A cheap sniff was measured, including its false positive

Rule under test: **any NUL byte in the first 4 KB => refuse**. Checked against every binary fixture and
against inputs that must keep working:

| | Result |
|---|---|
| 10 binary fixtures (odt, xlsx, pptx, djvu x2, cbr, cb7, cbt, cbz, docx) | **10/10 rejected** |
| 6 text fixtures (txt, Cyrillic txt, rtf, html, fb2 x2) | **6/6 accepted**, 0 NULs each |

Two findings that should shape the tactical pass rather than be re-derived:

- **A magic number at offset 0 is not sufficient on its own.** `.cbt` is a tar: offset 0 holds the
  first entry's *filename*, and the `ustar` magic sits at offset 257. The NUL sniff catches it (627
  NULs in the first 4 KB); a naive first-8-bytes table does not.
- **The NUL rule has a real false positive: UTF-16 text.** A Notepad "Unicode" save is UTF-16LE - 75
  NULs in a two-line ASCII file - and would be refused as binary. Any NUL rule must check for a
  UTF-16 BOM (`FF FE` / `FE FF`) first. **As of 2026-07-17 the decoder half is done**
  ([`2026-07-17_utf16-text-not-decoded`](2026-07-17_utf16-text-not-decoded.md), Implemented):
  `internal/txt.decodeText` already tests the BOM before anything else, so the "check the BOM first"
  precondition exists to build on. The sniff still has to make the same BOM check *before* it counts
  NULs, so it refuses a `.docx` while letting a UTF-16 `.txt` through to that decoder - reuse the same
  `bomUTF16LE`/`bomUTF16BE` check, do not invent a second one.

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[x]` Done | `LooksBinary` guards the `default:` arm; refuses with a named format + non-zero exit + no output dir |
| GUI (`doc-html-ui`) | `[x]` Done | inherits the pipeline; the CLI's non-zero exit + stderr message surfaces in the GUI's error path (it shells out and streams the log) |
| MSIX Store app | `[x]` Done | inherits the GUI; no packaging impact |
| Browser extension | `[x]` Done | confirmed by tracing: `detectFormat` -> EPUB reader (CBZ) or `loadPdfData`/`handleLoadError` (DjVu/RAR/7z/tar), both clear notices |
| Website / docs | `[x]` Declined | no surface enumerates a supported-format guarantee to correct; `docs/PARITY.md` gains the signature-first invariant |

## Cross-references

- Go: [`internal/pipeline/pipeline.go`](../../../internal/pipeline/pipeline.go) - `switch ext`, the
  `default:` arm; [`internal/txt`](../../../internal/txt) - the extractor being handed bytes it cannot read.
- JS: `extension/src/viewer.js` - `detectFormat`, `imageMime`, `isMobiBytes`, `loadFromData`.
- Fixtures and repro: `test_doc/CORPUS.md`. Every row above reproduces in under a second.

## Done criteria

- [x] A binary input is refused with a message naming what it looks like, and a non-zero exit.
- [x] No output directory is left behind for a refused input (the book dir is `RemoveAll`'d; verified the
      real `<folder>/<name>` subdir is gone, not just the empty parent).
- [x] `.docx`, `.djvu`, `.cbz`, `.cbr`, `.cb7`, `.cbt`, `.odt`, `.pptx`, `.xlsx` all refuse - verified by
      running all 12 corpus fixtures, not by reasoning about the sniff.
- [x] Every text fixture still converts, **including** UTF-16 (plain, cp1251, UTF-16LE/BE, UTF-8-BOM: all
      exit 0 with `index.html`).
- [x] `docs/PARITY.md` records byte-signature detection as a shared invariant.
- [x] Tests / gate green: `internal/txt` (unit) + `tests` (integration, 219s) + lint. Pre-existing
      `TestExtract_NoCalibre` still environment-fails as before, unrelated.
- [x] Changelog entry in `DEV/CHANGELOG.md`.

## Left open (a smaller sibling, deliberately out of scope)

- **A binary named `.txt` (or another handled extension) still reaches its reader unguarded.** The sniff
  sits only on the `default:` arm, which is where the eight measured fixtures land. A `.docx` renamed to
  `.txt` would still make garbage. Moving the guard into `txt.Extract` (and, by the same logic, a
  signature check in each reader) would close it, but that is a wider change than this ticket's measured
  problem and risks the "text fixture still converts" guarantee - worth its own ticket if it ever bites.
