# Answering the FastMediaSorter Lite exchange - the questions our first reply left open, 2026-08-12

**Counterpart document:** `FastMediaSorter_Lite/docs/specifications/SPECIFICATION_DOC2HTML_OCR_POSITIONING_EXCHANGE.md`
(their sections 10-14 record our first reply, section 13 lists what it did not answer).
**Question:** answer their remaining questions from the shipped code rather than from memory, and
hand over the material their section 4 asked for.
**Scope:** this repository's OCR overlay, both editions. Two defects were found while answering and
are fixed here; everything else is a statement about what the code does today, with the file and
the measurement behind it.

The reader who wants the mechanism rather than the answers should start at
[`docs/ocr-pipeline.md`](../../docs/ocr-pipeline.md).

---

## 1. Reading order between blocks, RTL, CJK, vertical lettering (their 3.2)

**We do not decide reading order at all - the engine does, and we never re-sort.** `parseTSV`
appends lines in the order Tesseract emitted them, and `clusterLines` walks that order once: each
line either continues the open cluster or closes it and opens a new one
([`tesseract.go:648`](../../internal/ocr/tesseract.go#L648),
[`:712`](../../internal/ocr/tesseract.go#L712)). There is no sort anywhere between the TSV and the
DOM, so the order of plates in the document is the order PSM 3's layout analysis produced. The
same is true of the extension (`ocr-cluster.js: clusterLines`).

That is a real answer, not a modest one: it means our reading order is exactly as good as
Tesseract's layout analysis, and we have measured where that fails. The lab scene
`synth-two-columns` is two text columns, and the desktop edition merges them - one plate spans
both. It is recorded as the corpus's only grouping defect and it is *in the acceptance file* as an
interim ceiling (`grouping.overall.max = 1`, `replacement.overall.max = 6`, six cross-group
overlaps, all six from that one scene, [`DEV/ocrlab/thresholds.json`](../ocrlab/thresholds.json)).
So: columns are the known weak spot, it is the engine's segmentation we are inheriting, and the
gate is set so a second scene joining it fails the build.

**RTL: no direction logic exists.** A plate is an ordinary DOM node inheriting the document's
direction; nothing in either edition sets `dir`. What we do test is that an RTL replacement does
not break the geometry: `rtl-arabic` is one of the six fixed translation-stress cases every lab run
applies to every scene, at 1.8x the source length
([`tools/ocrlab/runner/stress.go`](../../tools/ocrlab/runner/stress.go)), alongside `long-latin`,
`long-cyrillic` and `cjk`. Those cases are literals, not translations, so a clipping result stays
reproducible. Being explicit about the limit: we verify that an RTL string lays out inside its
plate without clipping or drift; we do **not** verify that the recognized word order is right for
an RTL source, and we have no RTL scene with human ground truth (`synth-rtl-layout` is a direction
and alignment proxy, and its own note says so).

**CJK: recognition and filtering only.** Short CJK runs bypass the vowel and minimum-length rules
in the translatability filter ([`text.go: isTranslatable`](../../internal/ocr/text.go)), because
those rules are alphabetic assumptions. Nothing else in the path is CJK-aware.

**Vertical lettering: catalogued, not supported.** `jpn_vert` is in the language catalog
([`tessdata.go:27`](../../internal/ocr/tessdata.go#L27), mirrored in `ocr-lang.js`), so a user can
select the vertical-text model and recognition works. The *plate* does not: it is a horizontal text
box, so vertical Japanese comes back as a tall narrow plate whose text is laid out horizontally and
wrapped. Worse, the clustering assumption is wrong by construction for it - adjacency is judged on
vertical pitch between horizontally overlapping lines, and in vertical writing successive lines sit
side by side. We have no vertical scene in the corpus and no measurement. Treat this as unsupported
rather than as partially working.

## 2. Hyphenation, line joins, and rotation inside a block (their 3.2 / section 6)

**Lines are joined with a single space and nothing else.** `clusterLines` writes `' '` between
lines, so a word broken across a line boundary stays broken: `hyphen-` + `ation` becomes
`hyphen- ation`, and that is what reaches the translator. No dehyphenation, no soft-hyphen
handling, no language-specific rejoining. The same in `ocr-cluster.js`.

**`rotationDegrees` is not something we produce.** Our block carries `Text`, `X0/Y0/X1/Y1` and
`LineH` and nothing else ([`tesseract.go:35`](../../internal/ocr/tesseract.go#L35)). Tesseract runs
with **no orientation detection** in either mode we use - PSM 3 is "fully automatic, no OSD" and
the sparse rescue rung is PSM 11, also without OSD - so text rotated inside the picture is not
detected, not corrected and not reported. If their model keeps the field, we can only ever write 0
into it, and it should be read as "unknown", not as "measured 0".

The one rotation we now do handle is the picture's own, and it is new since our first reply:
section 5 below.

## 3. Print, and reopening the result (their 3.3)

**Reopening is a non-event.** The desktop output is a static file: percent geometry in the inline
style, no state, no storage, no layout that depends on when it was opened. Reopening reproduces the
same numbers, which is the same property the drift measurement checks across viewports.

**Print is not implemented and has a specific known defect.** Grepping both editions for
`@media print` / `print-color-adjust` returns nothing. A plate is opaque only because of its
`background` colour, and browsers do not print backgrounds by default - so on paper the plates
would come back transparent with their translated text printed *over* the still-visible source
lettering. The fix is one declaration (`print-color-adjust: exact` on `.ocr-box` / `.ocr-plate`),
but we have not measured a printed page, so this is recorded as a gap with a proposed fix rather
than claimed as handled. Ticket:
[`2026-08-12_ocr-exchange-followups`](../plan/2026-08-12_ocr-exchange-followups.md).

## 4. How positioning error is measured, and the tolerance (their 3.4)

The numbers exist, and the honest summary is: **our accepted tolerance for movement is zero pixels,
and we have no bound at all on absolute error.**

What the lab measures ([`tools/ocrlab/metrics`](../../tools/ocrlab/metrics)):

| Measure | Definition | Where |
|---|---|---|
| IoU | intersection over union of plate and annotated group, rasterized at image resolution | `geometry.go: IoU` |
| Edge error | per-edge distance plate-to-group, normalized by the group's own size, plus the worst of the four | `geometry.go: Edges` |
| Drift | largest movement of one group's plate across the pinned viewports, normalized by group size | `placement.go: Drift` |
| Concealment | residual source ink still visible under a plate | `concealment.go` |

What the gate accepts ([`DEV/ocrlab/thresholds.json`](../ocrlab/thresholds.json), schema version 1,
derived from run `base06-desktop` on 2026-08-12):

- **position: worst drift, max 0 px** - measured 0.000 in both editions, and the note in the file
  says why the bound is the measurement: the overlay is expressed in natural image coordinates and
  holds them under responsive scaling, so any non-zero value is a regression;
- four **hard gates fixed at zero** and not tunable: protected-area damage (px), merged reading
  groups, clipped plates after stress, plates crossing another group;
- recognition mean recall min 0.72; concealment worst residual ink max 0.28 (baseline 0.2707 - the
  file states outright that 27 % of the source lettering still showing is a failure and the bound
  exists to stop it worsening);
- grouping merges+splits max 1; replacement clipped+cross-group max 6; review scenes-with-a-named-
  failure max 2; cost 18.5 s total OCR wall-clock (the one bound deliberately above its baseline).

Every bound carries the baseline it came from, and `Thresholds.Validate` refuses a derived bound
with no baseline and no note ([`report/gate.go`](../../tools/ocrlab/report/gate.go)). The file's own
caveat is that it comes from 11 annotated dev scenes with **no holdout**, so no dimension carries a
regression tolerance yet.

**The gap their question exposed.** The position dimension is graded on *drift* - stability across
viewports - and nothing grades absolute IoU or edge error against ground truth. That is not a
theoretical hole: the defect in section 5.1 below put every plate on a wrong-sized picture, and it
drifted by exactly 0 px at every viewport, because it was equally wrong at all of them. A stability
metric cannot see a systematic offset. Adding an absolute-position bound is now a ticket.

## 5. Two defects found while answering, both fixed here

### 5.1 The navbar's aspect guard moved every plate off its text

The converted page ships a script that keeps image proportions when something changes an image's
height; it writes an **inline** `width` on every `<img>` it watches
([`internal/htmlgen/navbar.go`](../../internal/htmlgen/navbar.go)). An inline width beats the
overlay's own `.ocr-fig>img{width:100%}`, so on any page where the picture is narrower than the
text column, the image fell back to its natural width while the plate container kept the column's -
and the plates, positioned in percent of the container, spread across a picture that was no longer
under them.

Measured in headless Chrome on `synth-uniform-paper` (640x320) in a 1216 px column, before the fix:

```
fig  1216.0x608.0     img 640.0x320.0 (computed width: 640px)
box  left 7.5% -> 124.8 css px, width 63.28% -> 769.5 css px
```

The plate belonged at x=48 and 405 px wide in image pixels; it rendered at x=91, 770 px wide - off
its text and nearly twice its size. After the fix the image fills the container (`img 1216.0x608.0`)
and the plate lands where the diagnostics say it should.

Fix: the guard skips any image inside `.ocr-fig` at both entry points, with a regression test that
fails if the exemption is dropped (`TestImageAspectGuardSkipsOCROverlay`).

This is also the answer to why their §11 comparison found our concealment number poor: a plate that
is 1.9x too wide covers the wrong pixels.

### 5.2 EXIF orientation was ignored, so rotated photos read as noise

Tesseract reads the file's stored pixels; a browser paints an `<img>` through its EXIF orientation
tag. For the ordinary portrait phone shot (`Orientation=6`) those are two different pictures, and
two things broke at once:

- the lettering the reader sees upright is stored on its side, and with no OSD the recognizer
  returns garbage. Measured on a 320x640 JPEG whose EXIF restores it to 640x320, tesseract 5.4.0
  reading the file as stored:
  `"duin{ seiqaz yep 2INb A\bulxen MoH"` - the upside-down transcript of a legible line;
- anything that did read came back in stored space while plates are positioned in displayed space.
  Measured before the fix: the same file rendered a 1216x2432 container with its plate 480 px down
  a picture whose text runs across the top.

Fix: the copy handed to tesseract is turned first
([`internal/ocr/exif.go`](../../internal/ocr/exif.go), used by `stageForOCR` in `tesseract.go`), so
recognition sees what the reader sees and every coordinate arrives in the space the plates use; the
decoded image the plate colours are sampled from is turned with it. After the fix the same photo
recognizes the full line and produces byte-identical geometry to the unrotated PNG of the same
scene:

```
unrotated PNG : x0=48 y0=63 x1=453 y1=167 lineH=20  left:7.50%;top:19.69%;width:63.28%;min-height:32.50%;font-size:2.88cqw
EXIF portrait : x0=48 y0=63 x1=453 y1=167 lineH=20  left:7.50%;top:19.69%;width:63.28%;min-height:32.50%;font-size:2.88cqw
```

Tests: `exif_test.go` covers the tag parser (all eight values, plus a non-JPEG, an EXIF-less JPEG, a
truncated header and a missing file), the staging rotation, the untouched common path, and that the
pixel map is a bijection - the first draft clamped mirrored coordinates and silently lost the
top-left pixel, which is a pixel a plate samples its paper colour from.

Note for their side: this is what their §11 row "Вход OCR" already had right. Their pipeline feeds
OCR a `pb.Image` snapshot that is EXIF-corrected before recognition; ours fed the file. Their
invariant 1 was satisfied on their side and not on ours.

## 6. The unanswered tail of their 3.4 - known failures by input type

- **EXIF rotation:** was broken, fixed above. Rotation is read from JPEG only; PNG `eXIf` and WebP
  can carry the tag and we do not read it (no such file has been observed here, and an untested
  parser is worse than a documented gap).
- **Very large images:** the halftone detector's work is capped independent of size (up to 96 tiles
  of 64x64), and its intermediate buffer is float32 rather than float64 specifically because a
  600-DPI page is ~20 megapixels and the app builds for 386, i.e. a 2 GB process ceiling
  ([`screen.go`](../../internal/ocr/screen.go)). That ceiling is real and reachable: our own
  full-corpus conversion test dies with `out of memory` on the big CBZ samples on this machine -
  reproduced on a clean checkout of HEAD, so it is a standing limit of the 386 build rather than a
  regression.
- **PNG with transparency:** fully transparent pixels are skipped when sampling plate colours
  (`samplePixels`), and when fewer than ~1.5 % of samples qualify as ink the near-black/near-white
  fallback is used, so a mostly transparent block still gets a legible plate rather than a wrong
  one.
- **Animated GIF:** decoded as its first frame, both by Tesseract and by our colour sampling. The
  plate is placed against frame 1 and stays put while the picture animates underneath. Not handled,
  and it should be treated as unsupported input.
- **Cyrillic / CJK file names:** handled. Tesseract and Leptonica open paths through the Windows
  ANSI codepage and mangle anything outside it, so the file is copied to an ASCII temp path before
  recognition (`stageASCIIPath`) - without it, a book under a Cyrillic name fails recognition
  silently.

## 7. Material handed over (their section 4)

Assembled under `temp/ocr-exchange/` (gitignored; see its `README.md` for the map):

1. **Two minimal examples**, each with the source image, the exact tesseract command line, the raw
   TSV of the primary pass *and* of all three rescue rungs, the neutral JSON of their section 6, the
   produced `index.html`, and the app's own `DOCHT_OCR_DIAG` line:
   - `minimal-plain/` - a document page, primary pass reads it;
   - `minimal/` - a balloon on a coloured panel, where the primary pass returns **zero lines** and
     the grey rescue recovers the text. This is the case their §11 "Растр (halftone)" row and our
     §10.5 were about, in raw data.
2. **Ground truth and corpus** (`corpus/`): all 13 human-checked annotations, the 8 synthetic scenes'
   pixels (ours, public domain, regenerated deterministically by `go run ./tools/ocrlab synth`), and
   a manifest of the 5 third-party scenes by source URL, licence and sha256 rather than by copy.
   Covers page scan, comic balloons, adjacent balloons, two columns, caption on gradient, halftone,
   display lettering and an RTL layout proxy. **Not covered: CJK, and a UI screenshot with ground
   truth.**
3. **Scaling evidence** (`scaling-*.json`): each example measured in headless Chrome at ten states -
   device scale 100/125/150/200 %, browser page zoom 100/125/150/200 % at a fixed window, plus
   tablet and phone. Worst plate-edge drift **0 px of the source image** in all three runs, after
   the 5.1 fix.
4. **The transform, in both directions** (`transform.md`): stored pixels -> staged copy -> TSV ->
   plate percentages -> laid-out CSS pixels, with the inverse the probe uses to compare geometry
   between viewports.

## 8. What is still open on our side

Ticket: [`2026-08-12_ocr-exchange-followups`](../plan/2026-08-12_ocr-exchange-followups.md).

- No absolute-position bound in the gate (section 4).
- Print produces transparent plates (section 3).
- Vertical writing is unsupported and unmeasured; no CJK scene, no RTL scene with ground truth
  (section 1).
- Column segmentation merges `synth-two-columns` (section 1).
- `createImageBitmap` in the extension does not name `imageOrientation` explicitly, so the two
  editions agree on rotated input only as long as the browser default holds.
