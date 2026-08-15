# What separates a merged plate from a real one, measured - and what does not

2026-08-13. Feeds [`DEV/plan/2026-08-13_ocr-sweep-plate-composition.md`](../plan/2026-08-13_ocr-sweep-plate-composition.md)
(P47), which asked for "a measure of a plate's own occupancy" and named two candidates without
measuring either. Both are measured here. One of them does not work, and saying so is the point of
the note.

## Method

Two independent sources, and neither is a recognizer grading itself:

- **Truth** - the 13 hand-drawn annotations in `DEV/ocrlab/annotations/*.json`, `origin: human`. Every
  reading group carries its own line boxes and its own bounds, so a group's coverage of the image and
  the share of its height its lines account for are both read straight off the ground truth.
- **The shipping recognizer** - `ocr.Recognize` (Tesseract 5, `eng`, the app's own staging and rescue
  ladder) over all 46 media files under `test_doc/ocrlab`, plus `test_doc/accounts.jpg`, the sweep's
  worst case. 96 plates in total.

Both were computed by a throwaway harness (`internal/ocr/zz_measure_test.go`, build-tagged `measure`,
deleted after the run) and a small reader over the annotation JSON. The raw table is reproduced below
rather than referenced, because the harness is gone.

## The measure P47 expected: how much of a plate's box its lines fill (by area)

**It does not separate anything.** Sorted by fill, the bottom of the recognizer's own output is:

| Fill | Coverage | Lines | Scene | Verdict |
|---:|---:|---:|---|---|
| 0.3621 | 0.0371 | 2 | `chicken-little-1961-political-cartoon` | legitimate |
| 0.4582 | 0.0529 | 3 | `synth-balloon-on-panel` | legitimate - the balloon the pitch rule exists for |
| 0.4732 | 0.1124 | 2 | `synth-display-lettering` | legitimate |
| 0.5185 | 0.2057 | 3 | `synth-uniform-paper` | legitimate |
| **0.5891** | **0.6829** | 6 | **`accounts.jpg`** | **the defect** |
| 0.6642 | 0.0295 | 2 | `synth-adjacent-balloons` | legitimate |

The defect sits above four scenes that must never be broken. There is no bound on area fill that
keeps `synth-balloon-on-panel` and drops `accounts.jpg`, and the reason is structural rather than
corpus-specific: area fill mixes the leading between lines with the ragged right edge of the lines
themselves, and a centred two-line balloon is ragged by construction.

## The measure that does: coverage of the image, and looseness on the vertical axis

Separated regions are separated **vertically**, so the axis has to be stated. Two quantities:

- **coverage** = the plate's box area over the image area;
- **line fill** = the sum of the plate's own line-box heights over the height of its box.

Neither works alone. Both together separate the corpus cleanly.

### Coverage alone would break a real scene

Largest plates the recognizer produced, and the largest annotated reading groups:

| Coverage | Source | Scene / group |
|---:|---|---|
| **0.6829** | recognizer | `accounts.jpg` - six list rows and their avatars in one plate |
| 0.6087 | truth | `samson-and-delilah-03-scroll` / `scroll-caption` - a crop that really is one caption |
| 0.4004 | recognizer | the same caption, as the recognizer boxes it |
| 0.2569 | truth | `samson-and-delilah-15-court-caption` / `band-caption` |
| 0.2141 | truth | `synth-uniform-paper` / `para` |
| 0.2091 | recognizer | `samson-and-delilah-15-court-caption` |

0.6087 against 0.6829 is a 12% window. Nothing defensible fits in it.

### Looseness alone would break a different one

Line fill over the annotated groups, lowest first: `synth-uniform-paper` **0.6667**,
`samson-and-delilah-03-scroll` 0.7921, `synth-two-columns` 0.7079, `samson-and-delilah-15` 0.7647,
`poster-display-type-on-flat-colour` 0.8460 (body) / 0.9221 (headline). `accounts.jpg` measures
**0.6608**. A bound that catches the defect takes `synth-uniform-paper` with it - three lines of
ordinary body text whose slack is just the leading a paragraph has.

### Together

The rule fires only on a plate that is **both** too big and too loose, which on this corpus is one
plate:

```
release when  coverage > 0.52  AND  lineFill < 0.72
```

| Constant | Must keep | Must release | Chosen | Margin |
|---|---:|---:|---:|---|
| `ocrMaxPlateCoverage` | 0.4004 (`scroll-caption`, as recognized) | 0.6829 (`accounts.jpg`) | **0.52** | ~30% each way |
| `ocrMinPlateLineFill` | 0.7921 (`scroll-caption`, hand-drawn) | 0.6608 (`accounts.jpg`) | **0.72** | ~9% each way |

Both are geometric middles of their brackets, the same construction `ocrTypeSizeRatio` used. The
line-fill bracket is the thin one, and it is thin because only one annotated group in the corpus is
large enough to be judged by it at all - stated here rather than hidden, and the first thing to
re-derive when the corpus grows.

**The response is to release, not to refuse.** The cluster becomes one plate per line, carrying that
line's own text and box. Every recognized word still reaches a plate - no scene loses text, which is
P47's own gate - and the released lines skip `isTranslatable`, because the assembled text already
passed it and re-testing the short rows individually would turn a composition fix into a recall
regression.

## What it does to the sweep's worst case

`test_doc\accounts.jpg`, converted with `-notranslate -noopen -force -ocr`:

| | Plates | Largest plate | Summed |
|---|---:|---:|---:|
| Sweep, 2026-08-13 build 26.0813.0245 | 1 | **80.6%** of the image | 80.6% |
| After the type-size rule (same day) | 2 | 68.3% | - |
| After this rule | **7** | **7.67%** | 44.6% |

Rendered, the page goes from a wall of grey words with the screenshot no longer visible behind it to
seven row plates, one per list row, with the window's own layout showing between them.

**One side effect, recorded rather than argued away:** the released rows are small boxes that each
contain a coloured avatar, so `blockColors` now samples six different inks (olive, purple, brown)
where the merged plate sampled one. The contrast floor still holds and every row is readable, but the
page is more colourful than the source. It is a consequence of judging colour on a smaller box, not
of the release itself.

## Concealment: the carrier question, settled against the corpus

P47 left open whether the opaque paper belongs on the plate box or on an inline span hugging the
string. The corpus had already answered by the numbers recorded in the ticket - **17% of the source
lettering left showing with the box carrier against 93% with the string carrier**, over 46 scenes,
with the lab's recorded bound at 0.28 - so the paper moved back to the box in both editions.

The cost is the reason the string carrier was tried and it is real: a block box is wider than centred
copy on its last line, so the plate paints paper beside that line (measured previously at 91 px
either side of a 984 px caption). Two things bound it now that did not before: the coverage rule
above stops a plate from being a page, and the type-size rule stops a headline from joining the body.

Verified by rendering, headless Edge at 1280x1800:

- `pdf-1page-scan_Lincoln-Proclamation-broadside` - the sweep's clearest "both layers legible at
  once" case - now reads as one layer. The blackletter lines the recognizer did not read stay
  visible, correctly, because no plate claims them.
- `pdf-1page-poster_Soldiers-Creed` - the creed plate conceals the creed. The plate's sampled paper
  is the panel's near-black, so it disappears into the poster.

## Two corrections to the sweep's reading

- **`First-Earthman` cover.** The sweep recorded "a plate whose text is the foot-of-page fine print
  renders over the `PLANET COMICS` logo". Re-rendered, the plate's text - `el ADVENTURES ON
  OTHER\WORLDS—THE UNIVERSE (OF THE FUTURE` - is the cover's **own top banner line**, and the plate
  sits on that banner. It does overlap the top of the logo, so the observation that something is
  wrong stands, but it is a plate box a little too tall for its line, not a plate carrying text from
  the other end of the page. Coverage 0.2774, well inside the bound; untouched by this work and still
  open.
- **`Soldiers-Creed`.** The twelve creed lines are one text and one plate is the right answer for
  them; what the sweep saw was the plate failing to conceal (the source lines showed around the
  recognized string) plus the lines the recognizer missed. The first half is fixed by the carrier
  move; the second is recall and is not this ticket's.

## Regression check

Plate counts before (sweep, 2026-08-13) and after, same inputs, `-ocr-lang eng`:

| Case | Plates before | Plates after | Max before | Max after |
|---|---:|---:|---:|---:|
| `img-jpeg_Nyoka-comic-page` | 17 | 20 | 11.5% | 11.49% |
| `comic-scan-tiny_First-Earthman` (6 pp) | 47 | 56 | 27.7% | 27.74% |
| `Hersley agreement` (4 pp) | 54 | 56 | 9.3% | 9.29% |
| `pdf-1page-scan_Lincoln-broadside` | 17 | 18 | 3.9% | 3.86% |

No case lost a plate. The small gains are the other in-flight OCR work of the same day (the rescue
ladder's strongest-rung rule), not this change.

## The OCR language rule, and why its floor is where it is

P47's third defect - an English recognizer pointed at a Russian screenshot produces transliterated
debris - is answered by letting the document's own script correct a language **nobody chose**
(`internal/ocr/script.go`). The detector is Tesseract's own `--psm 0` pass, and the measurement that
matters is how badly it performs, because that is what sets the floor. Over the same 46 scenes:

| Verdict | Confidence | Scene | Correct? |
|---|---:|---|---|
| Latin | 26.67 | `commons-right-to-read-poster-1970` | yes |
| Latin | 23.33 | `synth-rtl-layout` | yes |
| Latin | 10.00 | `synth-uniform-paper` | yes |
| **Cyrillic** | **8.24** | `image.png` (Russian UI screenshot) | **yes** |
| **Cyrillic** | **8.15** | `polish-soviet-propaganda-poster-18y` | **yes** |
| Latin | 5.71 | `accounts.jpg` | yes |
| **Arabic** | **5.00** | `john-bull-vs-grover-cleveland` (English, 1887) | **no** |
| Latin | 4.78 | `le-petit-journal-balkan-crisis-1908` | yes |
| **Cyrillic** | **3.81** | `cover-of-archie-s-pals-n-gals-no-25` (English) | **no** |
| Cyrillic | 3.66 | `cover-of-reggie-no-18` (English) | no |
| Cyrillic | 3.11 | `samson-and-delilah-03` (English) | no |
| Cyrillic | 2.73 | `atomicwar0201` (English) | no |
| Japanese | 0.73 | `crowdstrike-outage-at-whitcoulls` (English) | no |

19 of the 48 files produced no verdict at all ("too few characters") - and two of those are the
corpus's other Cyrillic scenes, `a-propaganda-poster-from-the-soviet-union` and
`poster-display-type-on-flat-colour`, so silence is the detector's commonest answer on exactly the
material the rule is for. The detector is wrong more often than it is right, so the floor is
bracketed by **the worst wrong answer (5.00) and the weaker right one (8.15)**:
`ocrScriptConfidenceFloor = 6.4`, the geometric middle, ~28% each way, with no scene in the corpus
between them.

Two more things keep a wrong verdict cheap:

- a language the reader typed is never second-guessed - the rule runs only when `-ocr-lang` is empty
  and the language came from `-src`;
- where data for the detected script is installed the language becomes `<script>+<default>`
  (`rus+eng`), not a replacement, so Tesseract still reads the original.

Where nothing installed can read the script, the book gets **no plates** and a line naming the
script, the download that fixes it and the flag that overrides it. Measured end to end on
`test_doc/image.png` with only `eng` installed:

```
OCR overlay: the pages are written in the Cyrillic script, which the eng (English) data cannot
read, so no text plates were added - install the data with -ocr-download rus, or pass -ocr-lang to
choose for yourself
OCR overlay: 0 image(s) overlaid
```

Against the sweep's `Katanoru-nonyyarenn | Npocmorp | Bugeo u Kavecteo` laid over a readable
interface, that is the criterion met: correct plates, or none.

Detection runs **once per book**, on its first image, not once per page: a book is one document in
one language, and the pass costs a whole extra Tesseract process (0.43 s measured), which on a
480-page comic would be four minutes for a question that has one answer.

## The lab run

`go run ./tools/ocrlab run -split dev` over the 46 dev scenes, 13 of them annotated, Edge
151.0.4129.78, Tesseract 5.4.0. Run `p47b` in `temp/ocrlab/`; the two reference runs are `p46b` (the
state this work started from - the string carrier) and `base06-desktop` (the run `thresholds.json`
was derived from, 11 annotated scenes because two were annotated later).

| | base06 (11 scenes) | p46b (13) | p47b (13) |
|---|---:|---:|---:|
| worst residual ink | 0.2707 | **0.9996** | **0.2705** |
| worst halo | 0.3503 | 1.0000 | 0.6532 |
| mean recall / IoU | 0.7273 / 0.7756 | 0.6154 / 0.7756 | 0.6154 / 0.7756 |
| worst IoU | 0.3489 | 0.3489 | 0.3489 |
| merges / splits | 1 / 0 | 1 / 0 | 1 / 0 |
| cross-group / clipped | 6 / 0 | 6 / 0 | 6 / 0 |
| worst drift (px) | 0 | 0 | 0 |
| scenes with a hard failure | 2 | 4 | 4 |
| total OCR (ms) | 12 380 | 35 396 | 34 503 |

Self-diagnosis over all 46 scenes, which needs no ground truth: original still visible under the
plates **mean 19%, worst 55%** - against the 17%/56% recorded when the box carrier was last measured,
and against P46's 93%/100% with the string carrier.

**Gate:** the concealment rows go from FAIL (0.9996) to **PASS (0.2705, limit 0.28)**, on all three
of overall / comic / texture. Every other row is exactly what `p46b` scored, and the rows still red
are red for reasons that predate this work:

- `merges 1` and `crossGroup 6` are red in `base06-desktop` too - `synth-two-columns`, the known open
  grouping defect the hard gate fixes at 0 by fiat rather than by measurement;
- `recall 0.6154 < 0.72`, `review 4 > 2` - the corpus grew from 11 annotated scenes to 13 after the
  thresholds were derived, and both new scenes (`a-propaganda-poster-from-the-soviet-union`,
  `poster-display-type-on-flat-colour`) score 0. 8 of 13 against 8 of 11;
- `cost 34 503 ms > 18 500` - the grey rescue ladder now runs every rung instead of stopping at the
  first non-empty one (2026-08-13, P46). It was already 35 396 ms in `p46b`.

### The padding is load-bearing, and the first run found it

The first run (`p47`) scored **0.2841** and failed the concealment gate by 0.0041. The plate rects on
the worst scene were **byte-identical** to `base06-desktop`'s - same four rectangles, same sampled
paper - so the difference was not geometry. It was the plate's own lettering: the residual metric
asks whether the *rendered* page still has ink where the source had it, so a plate's replacement text
counts against it wherever the two coincide, and the padding decides how large the runtime fit grows
that text. Restoring the paper's original inset (`0.05em 0.15em` -> `0.08em 0.28em`) took one plate's
font from 60.0 px back to 56.4 px and the scene from 0.2841 to **0.2705**, a hair under
`base06-desktop`'s own 0.2707. Both editions carry the value and `TestParityOCRFontFit` pins it.

Two lessons worth keeping: a residual-ink measurement cannot separate "the source shows through" from
"the replacement happens to land on the same pixels", so at 0.27 the two are the same order of
magnitude; and a padding value that reads as cosmetic can move a gated number.

### One number went up, and it is the metric rather than the plate

`worstHalo` is 0.6532 against `base06-desktop`'s 0.3503, all of it from one scene:
`synth-text-on-halftone`, 0.0091 -> 0.6532. The plate rect is identical; what changed is the sampled
pair, and it changed **before** this work, in P46's ring-test and ink-median pass:

| Run | paper | ink |
|---|---|---|
| `base06-desktop` | rgb(120, 110, 102) | rgb(240, 240, 240) |
| `p47b` | rgb(244, 239, 224) | rgb(120, 110, 102) |

The scene's own generator settles which is right: `sceneTextOnHalftone` paints cream
rgb(245, 240, 225) under a rgb(120, 110, 100) dot screen and draws the caption in **black**. So the
current pair is correct and `base06-desktop`'s was inverted - it rendered light text on the screen's
own grey. Halo measures ink in a ring just inside the group's bounds, so a correctly light plate on a
screened ground scores worse than an inverted dark one that blends into the screen. The number is the
metric penalising the right answer, not a plate getting worse; halo carries no gate, and the reason
is recorded here rather than left as an unexplained jump.

## What is not measured here

- **The three scenes this ticket is named for are still not in the corpus**, so nothing here scores
  them. See the ticket: the entry is blocked on the two fields only a person may fill.
- `accounts.jpg` is a private account list. Its geometry is preserved in the unit-test fixtures
  because that is what the rule is measured on; its row texts are not, and it is deliberately kept
  out of the versioned corpus.
- The **extension edition** was not run (`npm run ocrlab`). Its share of this work is the grouping
  rule and the carrier, both pinned by parity tests against the Go source; its own evidence is a
  separate run. **Run 2026-08-15**, `temp/ocrlab/ext-0815` against this run:
  [`ocrlab/2026-08-15__extension-parity-run.md`](ocrlab/2026-08-15__extension-parity-run.md).
  Concealment in the extension is equal or better on every scene that plates; it carries two defects
  this edition does not; and the comparison surfaced a scorer defect that flatters the desktop
  numbers here on the three scenes where recognition found nothing.
