# Strategic spec: 16_2026-08-11_ocr-visual-fidelity-lab - OCR that replaces image text convincingly

**Ticket:** 16_2026-08-11_ocr-visual-fidelity-lab
**Status:** In Progress (6 / 8 phases; phase 07 1 / 7 - Step 07.3 done 2026-09-25, 07.1 / 07.2 ⛔ blocked on human-owned annotation - see the tactical INDEX, which is the authority on phase state)
**Priority:** 40
**Date:** 2026-08-11
**Tier:** Complex
**Tactical plan:** [`16_2026-08-11_ocr-visual-fidelity-lab/INDEX.md`](16_2026-08-11_ocr-visual-fidelity-lab/INDEX.md)

> **Scope:** STRATEGIC. This is the quality contract and the experimental method. The tactical
> plan will name code, scripts, schemas and thresholds only after the baseline is measured.

---

## 1. Problem

The product can recognize text embedded in an image and draw selectable, translatable HTML over it.
That is necessary, but it is not yet the visual promise a reader needs. A good result must make the
old lettering disappear, put the replacement at the same reading location, retain the drawing,
paper, speech-bubble border and layout around it, and remain readable when the translation is longer,
shorter or uses another script. An opaque rectangle that erases part of a comic panel, a translated
line that floats away from its balloon, a halo of the original letters, or a clipped replacement are
all failures even when OCR returned the correct words.

The current checks prove several isolated properties - confidence filtering, clustering, responsive
coordinates, colour sampling and post-translation re-fit - but they do not yet form a permanent,
representative visual-regression loop. In particular, there is no deliberately curated mix of
posters, scanned documents, cartoon lettering, comics, speech balloons, captions, textured
backgrounds and difficult layouts; no independent holdout set against which tuning is forbidden; and
no scored evidence that a change improved concealment and placement rather than merely moving a
failure to another image class.

This ticket establishes that loop and uses its measurements to drive OCR and redraw improvements.
The target is not to claim impossible pixel-perfect restoration of every background. It is to make
the best truthful replacement available for each image, preserve the unmodified source underneath,
and refuse a visually destructive "fix" in the name of hiding text.

## 2. Goals

1. Build a varied, legally traceable visual corpus with text baked into the pixels, growing towards
   **200+ distinct scenes**, at least 80 of them comics, caricatures or illustrated dialogue, and no
   category above 35% of the scored holdout.

   > **Owner decision, 2026-08-11: these are targets, not a gate.** The numbers were being enforced
   > as a precondition, which meant the measure-look-fix loop could not start until the corpus was
   > finished - backwards, since it is running images through the program that tells you which
   > images you still need. `ocrlab verify` now reports coverage as a shortfall and blocks nothing;
   > `-strict` restores the hard gate for the day the corpus is meant to be complete. What still
   > blocks is a defect: a bad or unverified licence, a hash that does not match, a missing file.
2. For every scored scene, retain a transcript, text-region geometry, reading groups and an explicit
   annotation of artwork or a border that must not be painted over. Use source text, not OCR output,
   as the truth whenever it can be established.
3. Make every test run produce comparable evidence: recognized text, plate geometry, replacement
   mode, rendered screenshots before and after replacement, and machine-readable measurements. A
   reviewer must be able to open a failed scene and see *where* and *why* it failed.
4. Improve recognition and grouping until text is found in the right dialogue bubble, caption or
   document region without merging unrelated regions or placing plates over art.
5. Replace source lettering using the least destructive method that can conceal it: reconstruct the
   immediate background when it is safe, cover only the text-bearing region, and draw the replacement
   in the original reading position. Preserve bubble outlines, panel borders, faces and other
   marked non-text content.
6. Ensure that a replacement never clips, overlaps a separate reading group, or drifts from the
   source position when the page is responsive or the browser translates its text. Test ordinary
   translations plus deliberately long Latin, Cyrillic, Arabic and CJK replacements.
7. Make improvements in both independent editions. The CLI/GUI/MSIX path and browser extension may
   use different engine plumbing, but their visible quality contract, corpus, annotations,
   measurements and accepted fallbacks must agree.
8. Turn every confirmed production failure into a minimized, licensed regression scene before its
   fix is accepted.

### Non-goals

- Altering, flattening or permanently modifying a user's original image. The source image remains
  the base layer; all reconstruction and replacement is an overlay that can be inspected or removed.
- Promising typeface-identical translation, semantic image editing, or perfect inpainting where the
  original letters cross detailed artwork. Quality is judged by concealment, placement, legibility
  and damage avoided, not by pretending the original never contained text.
- Downloading arbitrary web images, material with unclear rights, personal documents or current
  commercial comics into the corpus.
- Optimizing exclusively for English. English has the deepest initial corpus, but script direction,
  glyph density and translation expansion are first-class test variables.
- Treating a higher OCR word count alone as a quality win.

## 3. Product contract

### 3.1 What the reader sees

For a successful replacement, the reader sees one coherent image: the drawing or document remains
intact, the original text is no longer legible, and the translated text is located where the source
was read. A speech balloon still looks like the same balloon; a caption remains a caption; a document
line stays on its original baseline region. The replacement may reflow inside the original region,
but it may not silently cover a neighbouring balloon, illustration or page rule.

The renderer chooses a concealment mode from local evidence:

1. **Uniform paper or fill:** use a sampled local background and matching ink colour.
2. **Gradient or lightly textured background:** reconstruct from a narrow surrounding context and
   apply the replacement only to the annotated text region.
3. **Text on artwork, a bubble border or an uncertain surface:** use a text-shaped or tightly bounded
   mask and a conservative reconstruction. The renderer must favour preserving non-text content over
   a large opaque patch.

Every mode records its confidence. If none can conceal text without harming protected content, the
result is marked as a visual-quality failure in the benchmark rather than being silently called a
success. The tactical pass decides the reader-facing fallback only after this failure rate is known;
it must be honest and must not destroy the source image.

### 3.2 Quality gates

The tactical plan will set final thresholds from the recorded baseline, but the benchmark must expose
all of these measures from its first run:

| Dimension | Measurement | Required direction |
|---|---|---|
| Recognition | Character/word error where a verified transcript exists; detection precision and recall for text regions | Higher recall without a precision collapse |
| Reading groups | One-to-one matching of plates to annotated balloons, captions, columns or document groups | No merge of independent groups; no split that changes reading order |
| Position | Intersection-over-union and normalized edge/baseline error against source regions | Plate remains over its source region at every viewport |
| Concealment | Portion of source glyph mask no longer visible in the rendered result; residual-ink / halo score | Original lettering is not readable |
| Damage | Overlay pixels outside the permitted text-replacement area and within protected art/border areas | Near zero; any visible border/art damage is a blocker |
| Replacement | Clipping, collision and overflow after real and synthetic translation swaps | Zero clipping and zero cross-group overlap |
| Visual review | Side-by-side source/result decision with a named failure reason | Every holdout failure is actionable, not subjective "looks bad" |
| Cost | OCR/render wall-clock time, memory and failure rate by image class | No unbounded quality-for-time trade |

An accepted change must improve the primary failing metric on its development scenes, pass all
existing hard safety gates, and not regress any holdout category beyond the agreed tolerance. A
single newly visible original word, damaged bubble border, clipped translation or plate crossing an
unrelated reading group is a hard visual regression even if aggregate OCR accuracy rises.

## 4. Corpus and provenance

### 4.1 Corpus shape

The corpus is a matrix, not a folder of attractive examples. Each scene receives one or more labels:

| Required coverage | Minimum scenes | Cases deliberately represented |
|---|---:|---|
| Clean printed/scanned documents | 35 | serif/sans, narrow columns, forms, stamps, low-contrast ink |
| Posters, labels and signs | 25 | display type, coloured fills, rotated and large text |
| Comics and illustrated dialogue | 60 | separate balloons, tails, captions, narration boxes, all-caps lettering |
| Caricatures, cartoons and memes | 20 | text integrated with drawing, irregular outlines, small labels |
| Text over imagery or texture | 20 | gradients, halftone, photographs, patterned paper, artwork behind glyphs |
| Layout and image degradation | 20 | skew, blur, scan noise, low DPI, compression, partial crop |
| Script and direction variants | 20 | Cyrillic, Arabic/Urdu RTL, Hindi/Bengali, Chinese, mixed-script scenes |

Counts overlap by label, but the total must be at least 200. The fixed holdout is at least 30% of
the corpus, stratified by the table. Parameters and algorithms may be tuned on development scenes
only. Once a scene enters the holdout, it does not move back to development after a failure.

The existing public-domain and openly licensed Golden Age comic scans remain valuable because they
contain both prose pages and genuine balloon pages, but they are not enough on their own. Candidate
discovery begins with Wikimedia Commons files whose individual file pages state Public Domain, PDM,
CC0 or an attribution-compatible Creative Commons licence, public-domain/clearly licensed Internet
Archive items, and government or institutional public-domain collections. Initial source examples
already verified during this specification are Wikimedia Commons' public-domain
[speech-bubble image](https://commons.wikimedia.org/wiki/File:Blue-Speech-Bubble.png), the public-domain
[US government "Right to Read" poster](https://commons.wikimedia.org/wiki/File:The_Right_to_Read_poster,_1970.jpg),
and the existing Internet Archive comic sources recorded in `test_doc/CORPUS.md`. They seed discovery;
they do not by themselves satisfy any scene category.

### 4.2 Legal and reproducibility rules

- The downloaded media stays in the ignored local corpus and never ships with the application.
- A small, versioned manifest records for every scene: stable source URL, retrieval date, licence and
  licence URL, author/attribution text where required, source-item identifier, byte hash, dimensions,
  language/script labels, and whether it is development or holdout.
- The manifest entry is invalid until its licence has been read **from the asset's own page**.
  Search-result snippets and a site's category label are not licence proof.

  > **Owner decision, 2026-08-11:** a Wikimedia Commons file page exposes its own licence through
  > the API, and reading that is not a search snippet - it is the same statement a person would
  > read, fetched mechanically. `ocrlab harvest` uses it and stamps `licenceVerifiedBy:
  > commons-api`, never a person's name, so machine-read and human-read provenance stay
  > distinguishable and any entry can be re-checked against its `licenceUrl`. Anything the file
  > page does not declare as PD, PDM, CC0 or CC BY is refused with its licence named. Material from
  > anywhere else still needs a person.
- Prefer PD, PDM and CC0. CC BY is allowed only when the required attribution is captured in the
  manifest. Do not use non-commercial, no-derivatives, unclear, revoked or user-private material.
- A generated crop, blur, downscale or synthetic translation stays tied to the source entry and is
  marked as a derivative. It never overwrites the downloaded original.
- A corpus bootstrap must be idempotent: it verifies hashes and licences before fetching, reports
  missing assets clearly, and never needs credentials or a browser session.

### 4.3 Ground truth

Each scored scene has an annotation record independent of the OCR engine:

- original transcript, language and direction where knowable;
- text-line and reading-group polygons or boxes, in natural image coordinates;
- group type: document line/paragraph, balloon, caption, label, title or incidental text;
- expected reading order and permitted replacement area;
- protected polygons for bubble outlines, panel borders, faces, illustrations and other content that
  must remain visible;
- a quality label for source ambiguity, so uncertain historical lettering cannot masquerade as an OCR
  regression;
- expected translation stress cases and a short reviewer note for known traps.

Two people, or one person plus a later independent check, must review a new annotated holdout scene.
Annotation disagreement is recorded and resolved before the scene becomes a gate. The benchmark may
use OCR-derived boxes to speed annotation, but it must never accept OCR output as its own truth.

## 5. Experimental loop

The work is deliberately iterative and evidence-led:

1. **Acquire and annotate.** Add a balanced licensed batch, validate provenance, and split it before
   looking at results. Preserve a small set of hand-built diagnostic scenes with exact known text and
   geometry for fast unit-level checks.
2. **Baseline.** Run the desktop and extension pipelines at fixed browser version, viewport, device
   scale and OCR language. Save OCR output, plate metadata, source screenshot, rendered replacement,
   translation-stress screenshots and the metric report.
3. **Inspect failures.** Triage by cause, not appearance: missed recognition, bad line/balloon grouping,
   coordinate mapping, background reconstruction, colour/contrast, font fitting, translation mutation,
   RTL behaviour or engine-specific difference. A failure report links the scene, image crop, metrics,
   output and exact configuration.
4. **Change one hypothesis.** Make the smallest scoped correction that addresses a demonstrated cause.
   Do not tune several thresholds or change both recognition and painting in the same unmeasured step.
5. **Re-run development and holdout.** Compare to the last accepted baseline by category and by edition.
   Keep the change only if it clears the targeted failure and meets the non-regression rule.
6. **Promote the lesson.** Add a deterministic test for the discovered invariant; add a minimized
   licensed regression scene if it was absent; document any new cross-edition constant or intentional
   difference in the parity contract.

The loop ends a cycle with a dated report, not with a visually plausible screenshot. Reports state the
corpus revision, OCR engines and language data versions, browser version, configuration, per-category
metrics, screenshots of every new failure, accepted/rejected hypotheses and the next smallest work item.

## 6. Edition parity and scope

| Edition | Required outcome |
|---|---|
| CLI (`doc-html-translate`) | Generates source-faithful, responsive image overlays; exposes enough diagnostic output for the corpus runner without changing normal user output. |
| GUI (`doc-html-ui`) | Inherits the desktop result and can reveal an honest visual-OCR failure/report path if the tactical design introduces one. |
| MSIX Store app | Runs the same local corpus/smoke path without relying on a writable install directory or external runtime state. |
| Browser extension | Produces the same annotated-scene evidence in the browser, including lazy image OCR, translation mutation and RTL cases. |
| Website / docs | Declined for initial implementation: no public quality claim moves until the benchmark has a stable, reproducible result. User documentation changes only if behaviour or fallback becomes visible. |

Shared OCR values and user-visible quality rules remain parity contracts. Exact OCR text may differ
between the native Tesseract command and Tesseract.js, so the gate compares their outcomes against the
same annotations rather than demanding byte-identical OCR output. If an edition cannot meet a safety
gate, it is not hidden behind aggregate parity; it is a failing edition with a named regression.

## 7. Architecture principles for the tactical pass

- Keep the original image intact as the base layer. Geometry is always expressed relative to natural
  image coordinates and must be observable after responsive scaling.
- Separate recognition, layout grouping, background/reconstruction decision, replacement drawing and
  post-translation fitting. Each stage needs an inspectable input and output so a screenshot failure
  does not become guesswork.
- Prefer text-shaped or tightly bounded masks to block-wide fills whenever a block contains artwork,
  gradient or a protected boundary. A background estimator must look locally, not assume the median
  of an entire paragraph is paper.
- Sample and preserve contrast from the local source region, then verify it after rendering. Colour
  adaptation is a quality aid, not permission to draw unreadable translated text.
- Treat real DOM translation as an event, not a one-time string. The fitted layout must re-check after
  browser translation, resizing, font loading and direction changes.
- Design the visual runner to make failures inspectable offline. It must not call paid translation
  services or upload user images; deterministic replacement strings are sufficient for geometry and
  overflow checks.
- Any new shared constant, rendering mode or fallback appears in the parity document and receives a
  mechanical drift guard. Do not copy a magic number into both editions and call that parity.

## 8. Done criteria

- [ ] A reproducible, licence-verified corpus contains at least 200 scenes and a stratified, immutable
      holdout of at least 30%, meeting every category minimum in §4.1.
- [ ] Every scored scene has independent transcript/geometry/protected-area annotations and a manifest
      entry with source, licence, attribution (where applicable), hash and split.
- [ ] One command produces side-by-side rendered evidence and machine-readable metrics for both
      editions at a pinned viewport/browser configuration; a missing dependency or corpus asset fails
      explicitly rather than silently reducing coverage.
- [ ] Baseline and every accepted iteration publish per-category recognition, grouping, placement,
      concealment, protected-area damage, overflow and performance results.
- [ ] On the final holdout, no plate clips after the supplied translation stress cases; no plate overlaps
      another annotated reading group; and no protected border/art damage remains unresolved.
- [ ] Every remaining non-perfect concealment is visible in the report, categorized and either fixed or
      explicitly accepted with a conservative behaviour. Aggregate scores never hide it.
- [ ] A change is accepted only with a holdout non-regression report and a new durable test/fixture for
      every newly discovered defect class.
- [ ] Both editions satisfy the same visible quality gates, required shared-invariant updates are made,
      and the Go and extension test suites pass.
- [ ] The engineering ledger records the delivered benchmark and each user-visible behaviour change.

## 9. Open research items

1. **Mask fidelity:** determine whether the available OCR word/line boxes can safely seed a sufficiently
   tight text mask, or whether a lightweight local binarization/segmentation pass is needed for text over
   artwork. Decide from the first annotated textured-background batch, not by intuition.
2. **Background reconstruction:** compare local colour sampling, directional interpolation and
   texture-aware reconstruction on the same protected-area annotations. Select the simplest method that
   passes the damage gate in both editions.
3. **Typography:** measure whether font-family/style classification materially improves the visual
   outcome versus a robust readable fallback. Do not add a font-recognition dependency unless the corpus
   shows that geometry and concealment are already sufficient.
4. **Annotation cost:** pilot enough scenes to establish how much manual polygon work is needed for a
   trustworthy gate. If full glyph masks are too costly, define a reviewed region-level proxy that still
   detects original-text bleed and artwork damage.
5. **Target thresholds:** establish numerical acceptance bounds only after the first stable baseline.
   They must be strict enough to catch a human-visible failure and practical enough to survive the two
   engine implementations; they are never retrofitted to make a preferred change pass.
