# Strategic spec: 12_2026-09-24_bugfix-resource-budgets - Memory, disk and size budgets for hostile or huge inputs

**Ticket:** 12_2026-09-24_bugfix-resource-budgets
**Status:** Draft
**Priority:** 75
**Date:** 2026-09-24
**Tier:** Strategic
**Tactical plan:** `DEV/plan/12_2026-09-24_bugfix-resource-budgets/` (created by /spec-tech)
**Findings:** X11 X12 X13 X14 X15 X16 X17 X20 X24 O6 E16 B23 (see the [findings register](../research/audit_2026-09-24/README.md))

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
   - **Status:** Open.
2. **Downscale vs refuse**
   - **Question:** should oversized images be downscaled for display and OCR, or skipped?
   - **Status:** Open.

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
