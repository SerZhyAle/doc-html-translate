# Strategic spec: 12_2026-09-24_bugfix-resource-budgets - Memory, disk and size budgets for hostile or huge inputs

**Ticket:** 12_2026-09-24_bugfix-resource-budgets
**Status:** BlockNeedUserTest - peak memory of a 2 GB CBZ on the 386 Windows build (done criterion 3), and CBR/CB7 through a real 7-Zip on Windows (the list-then-extract flags are verified here only against a stub)
**Priority:** 75
**Date:** 2026-09-24
**Tier:** Strategic
**Tactical plan:** `DEV/plan/done/12_2026-09-24_bugfix-resource-budgets/` (created by /spec-tech)
**Findings:** X11 X12 X13 X14 X15 X16 X17 X20 X24 O6 E16 B23 (see the [findings register](../../research/audit_2026-09-24/README.md))

> **Scope:** STRATEGIC.

---

## 1. Problem
The app ships a 32-bit build with a 2 GB address space. Several paths still let one input exhaust
memory or disk:
- **Image decoding:** images are decoded with no pixel budget, several full-frame copies of an image exist at once, and up to 16 OCR workers run in parallel.
- **Comic archives:** every page is held in memory up to a 4 GB cap. RAR and 7z archives are fully unpacked to the temp drive before any limit applies.
- **EPUB:** there is no total-size cap, and an oversized entry is silently truncated.
- **Multi-page TIFF:** the whole file is copied once per frame.
- **32-bit TIFF parsing:** an offset overflow can panic the parser.
- **Memory multipliers:** FB2 keeps several copies of its embedded images, and the PDF TIFF flip is pixel-by-pixel slow.
- **Extension edition:** it has no caps at all, an undocumented drift from the desktop.

## 2. Goals
1. No single input can make either edition crash from memory exhaustion. Oversized inputs are refused or degraded with a message.
2. Archive extraction checks sizes before writing, and cannot fill the disk beyond a stated budget.
3. Large comics and scans convert with a working set that is independent of the page count.
4. Truncation is never silent: exceeding a per-entry cap is an error or a named warning.
5. Integer handling in binary parsers is safe on the 32-bit build.
6. Both editions enforce the same published limits, and docs/PARITY.md records them.
7. Container-type mismatches (a `.cbz` that is really RAR) are detected by content, not only by extension.

**Non-goals:**
- Streaming translation or OCR of arbitrarily large books.

## 3. Wishes and constraints
### 3.2 Hard constraints
- **Platform / versions:** the 386 build is the binding case; the browser tab memory limits apply to the extension.
- **Performance:** a 500-page comic must not get slower.
- **Data compatibility:** n/a.
- **Localization:** "file too large" messages in 13 languages.

### 3.3 Owner inputs (Approval gate)
- **Related tickets:** `bugfix-external-process-bounds` (process deadlines are the time half of the same budget).
- **Performance budget:** the limits themselves (§6.1).
- **Platform constraints:** cross-edition. The limits are pinned in docs/PARITY.md and the parity test.
- **Validation level:** bomb fixtures (zip, 7z, a TIFF with a huge header, a TIFF with a huge IFD offset) run under the 386 build, plus a memory-profile check on a large comic.
- **Owner sign-off:** required for the numeric limits.

## 4. Current architecture context
Each reader sets its own caps. Comics collect all page bytes, then sort, then write. The 7-Zip path
unpacks everything first. Image readers decode without a header check, although a header-only read
already exists elsewhere in the code. The extension inflates every EPUB entry eagerly into memory.

## 5. Proposed approach
### 5.1 Pillars / modules
- **Pixel budget:** a header-only size check before any full decode; the image is refused or downscaled above the budget.
- **Decode once:** one decoded frame per image is shared by all OCR passes. Worker concurrency is sized by memory as well as by CPU.
- **Archive budgets:** list entries first, then check the per-entry, total-size and entry-count limits, then extract only accepted entries. Symlinks are never followed.
- **Streaming pages:** collect names and sizes, sort, then stream each page to its destination.
- **Honest caps:** reading one byte past a cap is detected and reported.
- **Safe arithmetic:** offsets are compared in a wide type before slicing.
- **Frame iteration without copying:** one reusable buffer or view per frame.
- **Content sniffing:** the container is identified from its signature, and the extension serves only as a hint.
- **Extension parity:** lazy inflation with byte counters, the same limits.

### 5.2 Data & event flows
Input -> header or listing probe -> budget decision -> streamed extraction or decode -> writer.

## 6. Open questions / research items
1. **The limits**
   - **Question:** what are the pixel budget, the per-entry and total archive sizes, the entry count, and the comic page cap?
   - **To find out:** check the largest legitimate files in the local corpus.
   - **Status:** Decided: full image decode at most 100 megapixels and 32768 px per side, probed from the
     header first. Archives (EPUB, CBZ/CBT, CBR/CB7): at most 20000 entries and 4 GB unpacked in total,
     checked from the listing before anything is unpacked. Per-entry caps: the owner's general 512 MB was
     replaced by the stricter caps already in the code - 100 MB for one EPUB file, 200 MB for one comic
     page. Comic page cap: the existing 20000, now the same number as the entry cap. Published in
     README.md (and RU/UK), extension/README.md and docs/PARITY.md "Input limits"; pinned by
     `TestParityInputLimits`.
2. **Downscale vs refuse**
   - **Question:** should oversized images be downscaled for display and OCR, or skipped?
   - **Status:** Decided: display always keeps the original file. Where a full decode is needed to convert
     (TIFF to PNG, the PDF TIFF flip) an image over the budget is refused with a localized message naming
     the limit. For OCR the owner asked for a downscale to fit the budget; as implemented, an image under
     the budget is never enlarged past it, and an image over it is not decoded in process at all, because
     every standard decoder must decode the full raster before it can be shrunk - that decode is what the
     budget forbids. Such an image degrades instead: Tesseract reads the original in its own process, the
     in-process passes (staging, grey ladder, screen pass, plate colours) are skipped, and a localized
     warning names the image and the limit. Owner to confirm this reading.

## 7. Risks
- **A limit refuses a legitimate large scan.** Likelihood: medium. Impact: the user cannot convert. Mitigation: generous defaults and a clear message naming the limit.
- **Listing a 7z archive first doubles the helper calls.** Likelihood: high. Impact: slightly slower. Mitigation: acceptable.

## 8. User impact (docs)
README and extension docs: the published input limits.

## 9. Architecture decisions (ADR)
**ADR-1: probe before allocate.** Why: header and listing reads are cheap and turn crashes into messages.

## 10. Links to other specs
`bugfix-external-process-bounds`.

## 11. Done criteria (strategic)
1. A 1 KB TIFF declaring 60000x60000 is refused with a message, and the 32-bit app survives.
2. A 7z bomb is refused before the temp drive grows beyond the budget.
3. Converting a 2 GB CBZ keeps peak memory well under 1 GB.
4. The same bomb EPUB is refused by both editions.
5. A RAR renamed to `.cbz` converts (or is refused with an accurate message).

## 12. Next step
`/spec-tech 12_2026-09-24_bugfix-resource-budgets`

## Implementation

X20 (FB2 image copies) is not in this change: it is covered by ticket 10 in the FB2 reader.

**Desktop (Go).**
- `internal/limits` (new): the published numbers, `CheckPixels`, `CheckArchive`, `EntryTooLarge`,
  `CopyCapped` (one byte past the cap is an error), `UncheckableListing`, and a TIFF IFD size reader
  (`TIFFFrameSize` / `CheckTIFF`). Messages in `internal/i18n/i18n_limits.go`, all 12 translations.
- X11, X12, X13 - `internal/img`: the IFD walk reads through an `io.ReaderAt` with every offset in int64;
  each frame decodes through a view that patches only the header's IFD pointer (no per-frame copy of the
  file); ImageWidth/ImageLength are read from the IFD before `tiff.Decode`. The IFD read is ours rather
  than `tiff.DecodeConfig` because on 386 that rejects a large frame with a bare "image too large" before
  the size is known.
- X11, X14 - `internal/pdf/images.go`: the TIFF flip probes the header, then swaps `Pix` rows in place on
  the decoder's concrete type (Gray, Gray16, RGBA, RGBA64, NRGBA, NRGBA64, CMYK, Paletted; anything else is
  converted once to NRGBA), and replaces the file atomically through `fsutil`.
- E16 - `internal/epub`: the central directory is checked before extraction; an entry over 100 MB is
  skipped by name, an entry inflating past its size fails and its partial file is removed; symlink entries
  are skipped; the leftover `fmt.Fprintf(os.Stderr)` warnings now go through `logging`.
- X15, X16, X17, X24 - `internal/comic`: every container is listed first (`archive` / `entry`), held to
  the budget in `selectPages`, sorted naturally, and each page is streamed into its file (`writePage`).
  TAR entries open as sections at recorded offsets. CBR/CB7: `7z l -slt` is parsed, and only the accepted
  pages are unpacked, named through a UTF-8 list file with `-spd`; a missing size or a truncated listing
  refuses the archive; symlink entries are dropped from the listing and unpacked files are `Lstat`-checked.
  `container.go` identifies ZIP/RAR/7z/TAR by signature, with the extension as fallback.
- O6 - `internal/ocr`: `decodeImage` refuses an image over the pixel budget from its header; the pool is
  `min(CPU count, memoryWorkers)` (`budget.go`, four 4-byte copies of the largest image against 1 GB on a
  32-bit build, 4 GB on 64-bit); one `ocrFrame` per recognition carries the staged picture to the grey
  ladder and the screen pass instead of a second decode, and drops the colour copy once the grey exists;
  `orientImage` reuses an RGBA source instead of copying it. Plate colours still decode once more in the
  overlay phase (one image at a time, after the pool), which keeps `overlay.go` untouched beyond the probe.

**Extension (JS).** B23 - `extension/src/limits.js` (new) holds the same numbers; `epub.js` `unzip` and
`comic.js` check the listing before inflating and inflate through `inflateRawCapped`, capped at the
listed size. Over the archive limits the viewer shows a localized notice (`vLimit*` keys, 13 locales);
an entry over its cap is skipped by name with `console.warn`, as the desktop skips it.

**Done criteria.**
1. `internal/img` `TestExtractTIFFPixelBombRefused` (a 378-byte TIFF declaring 60000 x 60000, refused with
   the limit named, under 16 MB allocated; passes under `GOARCH=386`), and
   `TestTIFFFrameOffsetsHugeIFDOffset` (no panic on 386). PDF: `TestFlipImageFileRefusesOverBudget`.
2. `internal/comic` `TestSevenZipBombRefusedFromListing` - refused from the listing and the stub proves
   7-Zip extraction never ran; `TestSevenZipUnknownSizeRefused`. Verified against a stub 7-Zip only.
3. `internal/comic` `TestExtractCBZStreamsPages`: a 64 MB CBZ converts allocating under a quarter of its
   size (the old reader allocated all of it). The 2 GB / 386 Windows measurement needs a human run.
4. Go `internal/epub` `TestExtractRefusesTotalBomb` and JS `test/limits.test.mjs` "unzip refuses an EPUB
   whose listing unpacks past the total limit" refuse the same bomb shape (50 x 90 MB of zeros).
5. `internal/comic` `TestRARNamedCBZConverts` (via the stub) and `TestRARNamedCBZWithout7Zip` (accurate
   notice naming the RAR container and the `.cbz` extension).
