# ocrlab - the OCR visual-fidelity lab

The instrument required by [`DEV/plan/16_2026-08-11_ocr-visual-fidelity-lab.md`](../../DEV/plan/done/16_2026-08-11_ocr-visual-fidelity-lab.md)
before any OCR or redraw change is accepted. It measures the shipped program: plate geometry comes
from the DOM the app produced and from the app's own diagnostics sidecar, never from a
reimplementation.

**Not shipped.** No build script compiles it, it is in no package, and neither `doc-html-translate`
nor `doc-html-ui` links it.

## Commands

Run from the repository root.

| Command | What it does |
|---|---|
| `go run ./tools/ocrlab verify` | Validate the manifest, media hashes, annotations and the coverage table. Exits 1 on any problem. |
| `go run ./tools/ocrlab fetch [id..]` | Idempotently download licence-verified media. Refuses anything a human has not verified. |
| `go run ./tools/ocrlab synth` | Redraw the deterministic diagnostic scenes, their exact annotations and their exact lettering masks. |
| `go run ./tools/ocrlab seed <id..>` | Write an OCR-seeded annotation draft for a human to correct. Never counts as truth. |
| `go run ./tools/ocrlab run [-split dev\|holdout\|all] [-scene id] [-lang code] [-fresh] [-reuse-from dir]` | Convert, render at three viewports, apply the stress cases, record evidence, then score and report. Each scene is read with its declared language unless `-lang` forces one. A scene whose inputs are identical to an earlier complete run is copied instead of collected (see "Reusing an unchanged scene"); `-fresh` collects every scene. |
| `go run ./tools/ocrlab reuse -run <dir> -edition extension -scene <id> -tesseract .. -tessdata .. -browser-name .. -browser-version ..` | The extension producer's lookup-and-copy of one reusable scene into a declared run; prints the scene record as one JSON line, or `null` and the reason on stderr. Not for hand use. |
| `go run ./tools/ocrlab score <run-dir>` | Grade a saved run offline - no browser, no recognizer. |
| `go run ./tools/ocrlab report <run-dir>` | Render `report.md` and a self-contained `report.html`. |
| `go run ./tools/ocrlab gate [-against <run-dir>] <run-dir>` | Judge a scored run against `DEV/ocrlab/thresholds.json`. Exits 1 on FAIL. |
| `go run ./tools/ocrlab exchange <run-dir>` | Write the `OCR-OVERLAY` section 7 comparison record, one `exchange/<scene>.json` per scene, from the run's diagnostics. Offline; works on either edition's run. |
| `go run ./tools/ocrlab compare <before-dir> <after-dir>` | Write per-scene `comparison.json`, `comparison.md` and an offline `comparison.html` with embedded aligned renders. Changed inputs, removed scenes and missing observations cannot certify regression safety. |
| `go run ./tools/ocrlab review export [-out <html>]` | Export an offline annotation workspace with rights, annotation and independent-review queues, geometry preview, JSON editing, undo and local draft storage. |
| `go run ./tools/ocrlab review import <draft.json>` | Validate geometry and write a draft beside reviewed truth. Both signatures are cleared; disagreements survive. Never approves or replaces reviewed truth. |
| `go run ./tools/ocrlab cost [-condition cold\|warm\|unclassified] [-out <json>] <run-dir..>` | Summarize compatible repeated per-scene conversion and rendering wall times: sample count, median, nearest-rank p95, range and population standard deviation. Memory keeps its actual snapshot semantics. No budget is inferred. |

Flags shared by most commands: `-manifest` (default `DEV/ocrlab/corpus.json`), `-root` (default
`test_doc/ocrlab`), `-annotations` (default `DEV/ocrlab/annotations`).

## Declared scope, portable evidence and compatibility

`run -purpose exploratory` is the default diagnostic workflow. `-purpose selected-dev` judges an
explicit development selection; `-purpose full-benchmark` additionally requires the original corpus
coverage and independent holdout requirements. A small development selection does not complete the
original campaign. The extension accepts the same `--purpose` and `--annotations` options.

Each new run freezes `declaration.json`, selected `corpus.json`, the full source-family inventory
`family-corpus.json` and the selected `truth/` files before recognition. It records hashes of source
bytes, dependency manifests, scenes, media, annotations, scorer and the configured thresholds.
`execution.json` checks the source digest again after execution; editing source during a run makes
its acceptance unverifiable. The extension uses the Go `declare` and `finish` helpers, so it needs
Go as well as Node. Use a new output directory for every run.

Both editions record a successful observation independently of plate counts for every combination
of the three pinned viewports and six stress cases. Every observation carries normal and replacement
text-hidden captures in natural image coordinates. The desktop probe hides `.ocr-box`; the extension
hides `.ocr-plate`. They check that diagnostic capture preserves layout. Source copies, truth,
declarations, observations, scores, reports and a separately generated `gate.json` travel together
when the entire run folder is copied. Keep an accepted baseline folder intact; acceptance and its
dated limitations remain a human decision.

Both editions capture tall images in bounded bands without changing the viewport, then map the
assembled render to natural image coordinates. The desktop bounds replies by pixel count; the
extension uses its existing band helper. A truncated legacy crop is refused rather than stretched
to manufacture a full-image observation.

Scoring and judging both revalidate frozen truth, review gates, family boundaries and source identity.
Judging also detects evidence, score or declaration edits since scoring. Missing inputs yield
`COULD NOT VERIFY` (exit 2); actual observed hard defects retain `FAIL` (exit 1), even if a reference
run is missing. Historical schema-1 evidence is still readable, but lacks the declaration required
for acceptance. Rescoring archives prior scores, summary, self-diagnosis and verdict under
`score-history/`; it does not rewrite historical evidence.

`report.html` puts failing and unavailable scenes first, offers text/finding/viewport/stress filters
and synchronized zoom, and shows strict and normalized text errors and their denominators. A scene
with no comparable transcript explicitly has unavailable text accuracy. A scene without a hard failure
is labelled accordingly; only the scoped gate can grant acceptance.

## Reusing an unchanged scene

A full campaign costs about an hour per edition, so a scene is not collected again when nothing that
could change its result changed and the earlier attempt finished well. **Collection** (conversion,
rendering, observations, captures, sidecar lines) is what is reused. **Scoring, the gate and the
report always run on the evidence as it stands**, so a changed scorer, threshold or procedure takes
effect without a browser run.

A scene is reused when an earlier run directory (the newest wins) matches on all of:

| Component | What is compared |
|---|---|
| edition | `desktop` or `extension`; editions never reuse each other |
| producer digest | every non-test byte of `internal`, `cmd/doc-html-translate`, `extension/src`, `extension/scripts`, `tools/ocrlab/runner`, `tools/ocrlab/evidence`, `tools/ocrlab/synth`, `go.mod`, `go.sum`, `extension/package.json`, `extension/package-lock.json` (`evidence.ProducerEntries`). Test files, `testdata` and the scorer packages (`metrics`, `truth`, `report`) are not in it. One edited product byte, related or not, forces a fresh collection |
| scene inputs | media, annotation, lettering mask and corpus-record hashes |
| language | the language the scene is read with |
| matrix | viewports and stress cases |
| engine and browser | Tesseract / tesseract.js version, the traineddata hashes **of the packs this scene reads**, the browser name and version |
| environment | OS, architecture and Go toolchain from the declaration (CPU counts are not compared) |
| procedure | the `ocrlab-110-v1` string |

An identity that was not recorded (an old declaration with no producer digest, an unknown engine
version) is never equal to another one, so evidence written before this feature is simply collected
again once and reusable afterwards.

The earlier scene must also have **passed OK**: no error, not unmeasured, all 18 viewport-by-stress
observation pairs, every file it names present and decodable at the scene's size, the source copy
byte-identical to the declared media, the run's `execution.json` showing its source unchanged, and
the files still the sizes the run recorded in `files.json` (a run from before that manifest is
accepted on the dimension check alone and says `integrity: dimensions`). When the earlier run was
scored by the **same scorer digest** as the working tree, a scene it counted as a hard failure is
collected again. A failed, partial, unmeasured or errored scene is never reused.

The copy is explicit. The scene record carries `reusedFrom` (`bundle`, `declarationSha256`,
`producerDigest`, `collectedAt`, `reusedAt`, `integrity`); a copy of a copy keeps the first collection
time. The run logs `reused <id> from <bundle>` or `fresh <id>: <reason>`, ends with `collected N,
reused M of K scene(s)`, `summary.json` carries `collectedScenes` / `reusedScenes`, `report.md` and
`report.html` list the reused scenes with their origin, and `cost` ignores them (their times belong
to the run they came from). Pages, captures and the scene's sidecar lines are copied, so the new run
folder stays self-contained.

`-fresh` (`--fresh` for the extension) collects every scene. `-reuse-from <dir>` (`--reuse-from`)
limits the search to one run directory, or to a folder of run directories; the default is
`temp/ocrlab`. The extension producer asks the same Go rule through `ocrlab reuse`, so both editions
reuse by one implementation and write one field.

## Calibration and cost limitations

The procedure is `ocrlab-110-v1`. Thresholds that do not name it cannot certify it, and no
acceptance bound was chosen for the new meanings.

### What changed meaning in `ocrlab-110-v1`

Ticket 110 found three pixel metrics that measured something other than their name. Each now has
one meaning, and the older figures are not comparable with the new ones:

| Metric | `ocrlab-109-v1` | `ocrlab-110-v1` |
|---|---|---|
| Residual (`residual`, `concealment`) | Median-luma "ink" in source and in the **normal** render. On a gradient or a texture the background itself was ink on both sides (90.86% / 96.89% with the lettering exactly removed), and replacement glyphs counted as old ink. | Measured on the **text-hidden** capture. Against a declared lettering mask (`basis: known-mask`) it is the share of known lettering pixels the capture still shows. With no mask it is measured only where the region is a single background tone (`basis: flat-region`); otherwise `state: unmeasured`, a `reason` and **no number**. |
| Readability / contrast | `minLumaSeparation` of the whole plate rectangle was reported as "readability"; it ignored the glyphs and the font (a texture with no text read 140). | That figure is `BackgroundContrast` (`backgroundContrast`, `minBackgroundContrast`) and is never called readability. `replacementReadability` describes the glyphs actually drawn: pixels that differ between the normal and the text-hidden render inside a plate. It reports glyph pixels, the smallest text-row height, ink-vs-underlying-pixel luma separation and the share of text rows cut by the plate edge. No glyph pixels is `state: no-glyphs`. |
| Damage (`damage`, `protectedHit`) | Plate **rectangles** intersected with protected regions, ignoring the plate mode (600 px for a plate whose mask would paint none). | `damage` counts protected pixels that the text-hidden render changed relative to the source inside the plates (`paintedDamage`). The rectangle figure remains as `rectangleIntrusion`, a diagnostic that never fails a scene or the gate. Without a text-hidden capture painted damage is `unmeasured`, never zero. |

Unmeasured concealment and painted damage are counted per bucket (`unmeasuredConcealment`,
`unmeasuredDamage`) and make the gate `UNVERIFIED`, exactly like a missing capture. Pixels already
proven painted over protected content stay a hard failure when another observation of the scene
could not be measured. Scores written before `state` existed read as before (`measured` alone).

Two readings are not the metric's to decide and are not decided: a translucent plate that moves a
pixel by more than codec noise counts as concealing, and the single-tone support condition
(`maxBackgroundSpread`, `maxInkShare`, `unchangedChannelTolerance` in `metrics/lettering.go`) is a
statement about where the estimate is defined, measured on the generated scenes, not a quality bound.

### The OCR language is declared per scene

The whole 109 campaign ran with `-lang eng` while the truth declared Russian, French and English. A
Cyrillic poster read as Latin yields debris at confidence 20-30, which the confidence gate rightly
drops, and the report read that as a product loss. Under `ocrlab-110-v1` every scene is read with its
own declared language:

1. an explicit `-lang` / `--lang` (a value for every scene, as before);
2. the `language` of the human-reviewed annotation's groups (an OCR-seeded draft declares nothing);
3. the manifest's `languages`;
4. `eng`.

A declared language is mapped to an OCR-catalogue pack (`ru` -> `rus`, `uk` -> `ukr`, `zh-TW` ->
`chi_tra`, ..); several declared languages become one `rus+eng` value in order of first appearance. A
language the catalogue cannot read (`ar`, `hi`, ..) has no pack and falls back to `eng`. The mapping
lives in `evidence/lang.go` and `extension/scripts/_ocrlab-lang.mjs`, and
`TestParityOCRLabLangTable` keeps both identical to each other and to `internal/ocr` `TessLang`.

Each scene records the language it was read with (`lang`) and where that came from (`langSource`:
`flag`, `truth`, `corpus` or `default`); the declaration freezes the same value per scene, and the run
header reads `per-scene`.

**Language data is never downloaded by the lab.** The desktop lab reads the packs already in a
tessdata folder the app reads; the extension lab reads the packs vendored in
`extension/vendor/tesseract/lang` (today only `eng`; the extension loads every other language from a
CDN on a user's explicit action, which the lab has no opt-in for). A scene whose pack is missing is
**unmeasured**, not failed and not "no plates": the scene carries `unmeasured: "language data
unavailable: <code>"`, appears in the skipped list with that reason, and keeps the run from
certifying acceptance. Install the pack (`doc-html-translate -ocr-download rus`) or force one language
with `-lang eng` to measure it.

### Where the text was lost

A scene with annotated text and no plates keeps its failure text (`no plates at all over N annotated
group(s) - ..`, compared verbatim by `compare`) and gains a `loss` record in `scores.json`, shown as a
`loss point:` line in the console, `report.md` and `report.html`. It is derived from the run's
diagnostics sidecar (`ocr-diag.jsonl`), which both editions write identically:

| `loss.stage` | Sidecar shows |
|---|---|
| `engine-returned-nothing` | no block and no dropped line: the recognizer returned nothing |
| `dropped-at-confidence-gate` | no block, every dropped line failed the confidence gate; the detail gives the count, the best confidence and its floor |
| `grouped-then-lost` | no block; some lines cleared the confidence floor and a later gate (`translatable`, `screen-merge`, `grey-merge`) discarded them; the detail gives the counts per gate |
| `rendered-nothing` | the pipeline kept blocks but the render recorded no plate |
| `unknown` | the scene has no record in the sidecar |

It is a diagnosis for a reader: it adds no failure, no threshold and no gate input.

### Comparing across procedures

`compare` refuses to compare a run declared under another procedure and says why in
`comparison.md`: the three metrics above changed meaning, so a drop in "residual" between a
`ocrlab-109-v1` and an `ocrlab-110-v1` run is the instrument and not the product. A run's own
declaration is frozen, so old evidence stays inspectable but cannot certify acceptance under the new
procedure. Rescoring (`ocrlab score`) writes new scores and archives the previous scores, summary,
self-diagnosis and verdict under `score-history/`; nothing is overwritten.

### Lettering masks

A scene may carry `<annotations>/<scene-id>.lettering.png`: a PNG of the image's size, white where
the original lettering's core pixels are. It must come from something independent of the pixels it
judges. `go run ./tools/ocrlab synth` writes the exact masks of the generated scenes as a by-product
of drawing their text. A real scene without a mask is unmeasured on any region that is not a single
background tone. `declare` freezes the sidecar into the run's `truth/` and records its digest
(`letteringSha256`), so the mask used for a score is part of the portable evidence.

### Remaining calibration work

`pixelDiagnostics` keeps all five measurements per observation. The self-diagnosis
(`self-diagnosis.json`) still reads the normal render, so replacement glyphs aligned on old
lettering can count as old ink there; it applies the same single-tone guard but not the hidden
capture. Metric false-positive fixtures for real (non-generated) scenes, exact actually-loaded
dependency tracing, holdout freezing/use history and a human usability review remain tracked in
ticket 109.


`ocrMs` now means the desktop conversion/recognition stage, without a second OCR call to recover
dimensions. The extension's recognition stage includes viewer startup; neither value is engine CPU
time. `renderMs` is a separate browser stage. `memoryKind` distinguishes Go runtime system allocation
from browser JS heap snapshots; `peakRssBytes` is unavailable on new runs. Cost summaries require
compatible complete evidence and the same edition, engine and environment. Cache conditions are
operator-labelled; the tool does not reset caches. Two smoke repeats do not establish latency budgets.

Rights metadata acquired by `harvest` remains pending human review. Legacy `commons-api` approval
stamps do not count as human decisions. Verification reports acquired, rights-reviewed,
synthetic/derived, gradable and independently reviewed holdout counts separately.

## The other edition

The browser extension is a separate codebase with its own OCR engine, so it produces its own
evidence and is graded by the same `score` / `report` / `gate` against the same annotations:

```text
cd extension && npm run ocrlab -- --help
```

It writes the identical record (`extension/scripts/_ocrlab-evidence.mjs` mirrors the Go `evidence`
package field for field, and `TestParityOCRLabEvidenceSchema` fails when one side moves alone).
Recognized *text* will differ between the Tesseract CLI and tesseract.js and is meant to - each
edition is compared with the annotations, never with the other's output.

## What it needs

- `tesseract` on `PATH`, next to the app, or at `DOCHT_TESSERACT` - the same resolution the app uses.
- Chrome or Edge. Override with `DOCHT_BROWSER=<path to the executable>`.

A missing dependency is a hard error, never a quietly smaller run.

## Layout

```text
DEV/ocrlab/corpus.json          versioned manifest - metadata only, no media
DEV/ocrlab/annotations/*.json   versioned ground truth
DEV/ocrlab/thresholds.json      acceptance bounds (written by the baseline phase only)
test_doc/ocrlab/                the media - gitignored, ships in nothing
temp/ocrlab/<run-id>/           evidence.json, scores.json, summary.json, shots/, report.html
```

## The three rules a contributor must not break

1. **OCR output never becomes truth.** A seeded draft carries `origin: ocr-seed` and `IsTruth()`
   is false for it, so the engine can never be graded against its own output. Correct a draft by
   hand, set `origin: human`, fill `review.annotatedBy`, and drop the `.draft` from the filename.
2. **No threshold outside a dated report.** The metrics package contains no bound at all, and a test
   parses it to keep that true. Acceptance numbers come from a recorded baseline and cite the value
   they came from, so a retrofit is visible in the diff.
3. **A licence is verified by a person, not by tooling.** `licenceVerifiedBy` is filled by hand after
   opening the asset's own licence page. A search-result snippet or a site's category label is not
   proof, and no code path may write that field.

## Windows traps, already handled

Recorded because each one presents as "the runner hangs" and costs an afternoon to find again:

- `msedge.exe --version` does not print a version and exit - it launches a browser and stays. The
  version is read from the installer's sibling version directory instead.
- `cmd.Stdout` as an `io.Writer` makes `cmd.Wait` block forever: browser helper processes inherit
  the pipe. Output goes to real files.
- A relative `--user-data-dir` hangs headless Edge rather than failing. Absolute paths only, and a
  test guards it.
- Headless helpers outlive the launch, so `Browser.Close` kills exactly the processes whose command
  line names this run's profile directory - never by image name, which would take the user's own
  browser with it.
- **`--dump-dom` and `--screenshot` stopped working** (measured 2026-08-12, Chrome 151 and Edge, in
  every headless mode): zero bytes, no file, no error. The runner speaks the DevTools protocol
  instead. Two traps of that transport: Chrome refuses a debugger WebSocket carrying an `Origin`
  header unless it is allow-listed, and a fragment-only navigation does not reload - so every page
  state is opened via `about:blank`, or the probe reports the previous state's numbers.
