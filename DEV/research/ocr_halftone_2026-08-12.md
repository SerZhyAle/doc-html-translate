# Lettering on a halftone screen - measure-fix cycle, 2026-08-12

**Ticket:** [`2026-08-11_ocr-halftone-defeats-recognition`](../plan/2026-08-11_ocr-halftone-defeats-recognition.md)
**Question:** `synth-text-on-halftone` produces no plates at all, and the grey rescue ladder does not
recover it. What does, and does that fix anything a reader would ever meet?
**Runs:** a 59-image transform sweep (the whole lab corpus, the synthetic scenes, and 19 images
harvested for this cycle), then `temp/ocrlab/loop4` against the `loop3` baseline. Desktop edition,
tesseract 5.4.0.20240606, lang `eng`, `-split dev`.

> The ticket's main constraint was ordering: **annotate real screened scenes first, then measure,
> then choose.** That order was kept, and it changed the answer twice - once about which images are
> really screened, and once about where in the pipeline the fix can possibly help.

---

## 1. What the corpus actually held

The ticket assumed the lab corpus was "full of" screened material with text on it
(`test_doc/ocrlab/commons/`, "several visibly screened"). Measured rather than assumed, that is not
the case. A tile-wise autocorrelation of the high-pass residual (tooling: `temp/ocrscreen/scan`,
throwaway) scores every image for a periodic lattice, and reading the top of that list at 1:1 zoom
gives three distinct reasons the corpus has almost no usable scene:

| Why a screened-looking scene is not a *text on a screen* scene | Examples |
|---|---|
| The screen is in the artwork; the lettering sits on clean white balloons or plain paper | `cover-of-archie-and-me-no-1`, `cover-of-archie-s-pals-n-gals-no-25`, `daffy-tunes-…-panel-4` |
| The scan resolves no screen at all - the scanner already averaged it away | every `atomicwar*` page (939 px wide ≈ 140 DPI), `cover-of-timmy-the-timid-ghost` |
| It is not a halftone at all - wood engraving, crosshatching, or an LCD photographed | `1862-political-cartoon-lincoln-spins-the-news`, `louis-brandeis-…-1916`, the two `crowdstrike-outage-…` photos |

Exactly one existing scene carries legible text printed **on** a resolvable screen:
`le-petit-journal-balkan-crisis-1908` (a 1908 chromolithograph at 451 DPI), and its screened text is
the map's outline lettering.

So the corpus was extended with real material of the right kind rather than the fix being tuned to
the one synthetic scene. Nineteen candidates were harvested and scored; the winners are 1950s comic
pages whose caption panels are printed as a tint with the lettering laid straight over the dots.

**Three real screened scenes now carry reviewed annotations** (`DEV/ocrlab/annotations/`):

| Scene | Material | Screen | Tone |
|---|---|---|---|
| `samson-and-delilah-03-scroll` | comic caption scroll, 1950 | pitch 6 px (12 after staging) | light pink |
| `samson-and-delilah-15-court-caption` | comic caption band, same issue | pitch 6 px (12 after staging) | saturated red - tone between ink and paper |
| `le-petit-journal-1908-map-labels` | chromolithograph map lettering, 1908 | pitch 6 px (unchanged - the scan is far above the upscale floor) | cream, outline letterforms |

All three are crops, tied to their full pages by `derivedFrom` (the manifest's own mechanism for a
derived scene; `ocrlab add` grew a `-derived-from` flag so this did not need hand-edited JSON). Line
geometry was measured from the pixels by luminance threshold and projection, never from OCR, so the
lab's first rule holds: the engine is not scored against its own output.

## 2. What recovers the diagnostic scene

`synth-text-on-halftone` is a 50%-tone dot screen at pitch 6 under bold type. It is staged the way
every image is - estimated DPI 55, below `ocrUpscaleDPIFloor`, so it is upscaled 2x and the screen
the recognizer actually meets has **pitch 12**. Tuning against the file's own pitch would have been
tuning against the wrong picture.

Confident words (the app's own gate, at the rescue floor of 80) over the staged image:

| Transform | engine thresholder | tiled Otsu |
|---|---|---|
| grey (rungs 1-2 today) | 0 | 0 |
| median 3x3 / 5x5 | 0 | 0 |
| downsample 2x / 3x / 4x | 0 / 0 / 0 | 0 / 0 / 1 |
| box blur r=1..3 | 0 | 0 (r=3: 2 broken) |
| box blur r=4 | **2 - full transcript** | 0 |
| Gaussian sigma 2.00-2.75 | 0 | 0 |
| Gaussian sigma **3.00-3.50** | **2 - full transcript** | 2 |
| Gaussian sigma 3.75-4.50 | 0 | 0 |

Two things fall out of this table:

- **A resolution change does not substitute for the kernel** - the ticket's second open question,
  answered. Averaging k x k blocks only removes a screen when k reaches the screen's own period, and
  by then the lettering has gone with it.
- **The working window is narrow** - sigma/pitch between 0.25 and 0.29 here - which is exactly why
  the kernel must be derived from the measured pitch rather than fixed.

## 3. Is a screen cheaply detectable? Yes, without a frequency transform

The ticket's first open question. A dot lattice repeats, so after a 3x3 high-pass the residual
correlates with itself at the screen's period; tile-wise autocorrelation finds it at O(n) with a
bounded tile count, and no FFT. Over the 59 images the separation is wide: the diagnostic scene
scores 1.00 of textured tiles agreeing at peak 0.78, while the strongest ordinary artwork reaches
0.86 at peak 0.44 and most sit under 0.25.

The shipped detector therefore does not gate on a strength threshold invented from one scene. It
requires only a period inside `[3, 24]` px that at least a quarter of the textured tiles agree on -
enough to reject noise, gradients and JPEG blocking - and its real safety comes from *where it runs*:
after every other rung has returned nothing, where a wasted pass costs one recognition on an image
that produced nothing anyway.

## 4. The number that was chosen, and what corroborates it

`ocrScreenSigmaDivisor = 4`: sigma = measured pitch / 4. Not fitted to the diagnostic scene - it is
at or next to the optimum on every screened image measured, at two pitches, synthetic and real:

| Scene | Pitch (staged) | Best sigma measured | pitch/4 |
|---|---|---|---|
| `synth-text-on-halftone` | 12 | 3.0-3.5 (nothing outside) | 3.0 |
| `samson-and-delilah-03-scroll` | 12 | 2.4-3.0 (56 / 51 words vs 46 for grey) | 3.0 |
| `samson-and-delilah-03` (whole page) | 6 | 1.5 (90 words vs 76 for grey) | 1.5 |
| `le-petit-journal-…-1908` (whole page) | 6 | 1.5 (81 words vs 55 for grey) | 1.5 |

## 5. The finding that matters most: where this can help, and where it cannot

A corpus-wide sweep ran every image twice - the ordinary grey rendition against the pitch-driven
low-pass - and compared confident-word counts (`temp/ocrscreen/lean/sweep.txt`).

**Of 59 images, exactly one goes from nothing to something: the synthetic scene.** No real image in
the corpus is both unreadable and recoverable by screen removal.

That is not a failure of the transform. It is a fact about where the screen defeats recognition:

- On real screened material the ordinary passes **already read the text**. The scroll crop yields 46
  confident words, the caption band reads its line, a whole screened comic page yields 76-96. The
  rescue position - "the image produced no plates at all" - is never reached, so a rung there can
  never help them.
- The one real scene that *does* reach the rescue position, `le-petit-journal-1908-map-labels`,
  reads nothing before the pass and nothing after it. Its obstacle is not the screen but the
  letterforms: outline-drawn place names following the drapery and the map's perspective. Screen
  removal is the wrong tool and the rung correctly changes nothing.
- Where screen removal genuinely pays on real material, it pays on images that **already produce
  plates**: `le-petit-journal` 55 -> 81 confident words (+47%), `samson-and-delilah-03` 76 -> 90
  (+18%), and the caption band completes its sentence (`IN THE COURT OF KING` -> `IN THE COURT OF
  KING TARENT.`). Reaching that gain needs a different position in the pipeline - a second pass
  whose new, non-overlapping plates are merged into the first pass's result - with its own
  regression surface (plate composition changes on every screened page) and its own cost (a second
  full recognition on every image carrying a screen). That is
  [`2026-08-12_ocr-screen-pass-for-pages-that-already-read`](../plan/2026-08-12_ocr-screen-pass-for-pages-that-already-read.md),
  not this ticket.

The same sweep is also the case *against* applying the low-pass more widely: as a replacement it is
plainly worse - `cover-of-archie-s-pals-n-gals` 16 -> 0 confident words, `cover-of-timmy-the-timid-ghost`
18 -> 7, `join-the-ranks-of-the-red-army` 71 -> 49.

## 6. Results

`synth-text-on-halftone`, through the shipped path (`ocr.Recognize`, not a reimplementation):

```
1 plate(s)   600x300
  (61,115)-(378,137) h=22  "MEANWHILE, ELSEWHERE"
```

The annotation says the line sits at (60,110)-(378,139) and reads `MEANWHILE, ELSEWHERE` - the
transcript is exact and the box is within 5 px on every edge.

Everything else is unchanged by construction and confirmed by the run: the rung is unreachable for
any image that produced a plate, so no scene that worked can move.

The baseline is `temp/ocrlab/pitch1` - the run made for the grouping ticket that landed the same
day - **not** `loop3`, which predates it and would credit this change with that one's gains.

**Plates, scene by scene: 5 gained, 0 lost, 40 unchanged.** Four of the five gains are the new
corpus scenes, which were not in the baseline at all; the only existing scene that moved is
`synth-text-on-halftone`, 0 -> 1.

| Metric (dev split) | pitch1 | loop4 | read honestly |
|---|---:|---:|---|
| Scored scenes | 8 | **11** | the three new annotated screened scenes joined |
| Mean recall / precision | 0.750 | 0.727 | `le-petit-journal-1908-map-labels` entered at recall 0 - it reads nothing, before and after |
| Mean IoU | 0.820 | 0.776 | same cause; `worstIou` is **unchanged** at 0.349 |
| Mean CER | 0.149 | **0.146** | |
| Mean covered (concealment) | 0.777 | **0.800** | |
| Worst residual / halo | 0.124 / 0.093 | 0.332 / 0.527 | both are `synth-text-on-halftone`, which had no plate to measure before |
| Min contrast | 86 | 86 | |
| Merges / splits / protected px / cross-group / drift | 1 / 0 / 0 / 6 / 0 | 1 / 0 / 0 / 6 / 0 | **every safety dimension unchanged** |
| Clipped | 0 | 3 | all three are `samson-and-delilah-03-scroll` under the long-text stress cases - a new scene |
| Failing scenes | 2 | 3 | the third is `le-petit-journal-1908-map-labels`, new and failing for a reason this pass cannot fix |

Not one of the eight previously scored scenes changed on any dimension. `synth-text-on-halftone`
enters the average at recall 1.00 and IoU 0.76.

### Cost

Measured as an A/B on the same machine, the rung switched off and on (`ocr.Recognize` end to end,
milliseconds):

| Scene | rung off | rung on | delta | what ran |
|---|---:|---:|---:|---|
| `samson-and-delilah-03` - reads, 7 plates | 605 | 610 | +5 | nothing: the rung is unreachable |
| `1862-…-lincoln-spins-the-news` - 0 plates, no screen | 861 | 863 | +2 | the measurement only |
| `himalaya-…-cropped` - 0 plates, no screen | 1676 | 1694 | +18 | the measurement only |
| `polish-soviet-propaganda-poster-18y` - 0 plates, no screen | 1494 | 1521 | +27 | the measurement only |
| `handwritten-sticker-art…` - 0 plates, 12 megapixels | 4986 | 4983 | -3 | the measurement only - and the tile cap is why 12 megapixels costs what a panel does |
| `le-petit-journal-1908-map-labels` - 0 plates, screen found | 1192 | 1635 | +443 | measurement, blur, one more recognition - and still nothing to show for it |
| `synth-text-on-halftone` - 0 plates, screen found | 405 | 546 | +141 | the same, and it gains its plate |

So: nothing at all on an image that reads, tens of milliseconds on an unreadable image with no
screen, and one extra recognition - here 140-440 ms - on an unreadable image that has one.

(The whole-run OCR totals in `summary.json` are **not** a usable cost signal for this change and are
deliberately not quoted: the corpus grew by five scenes between the two runs, and part of loop4 ran
while the Go test suite had the machine.)

## 7. Code

| Change | Where |
|---|---|
| Screen detector, pitch-driven Gaussian | [`internal/ocr/screen.go`](../../internal/ocr/screen.go) |
| The rung, and the ladder that now ends with it | [`internal/ocr/tesseract.go`](../../internal/ocr/tesseract.go) `greyRescue` / `screenRescue` |
| Extension port | [`extension/src/ocr-screen.js`](../../extension/src/ocr-screen.js), [`ocr-overlay.js`](../../extension/src/ocr-overlay.js) `screenRescue` / `measureScreenPitch` |
| Shared-invariant record | [`docs/PARITY.md`](../../docs/PARITY.md) "Halftone screen rung" |
| Drift guard | [`tests/parity_test.go`](../../tests/parity_test.go) `TestParityOCRScreenRung` |
| Unit tests | [`internal/ocr/screen_test.go`](../../internal/ocr/screen_test.go), [`extension/test/ocr-screen.test.mjs`](../../extension/test/ocr-screen.test.mjs) |
| Corpus: three annotated real screened scenes, two parent pages, `-derived-from` | `DEV/ocrlab/corpus.json`, `DEV/ocrlab/annotations/`, [`tools/ocrlab/cmd_add.go`](../../tools/ocrlab/cmd_add.go) |

The measurement tooling lives under `temp/ocrscreen/` and is gitignored on purpose: like the
`temp/ocrdiag/` harness before it, it reads the lab's own scenes to answer one question, so it is
cheaper to re-derive from the tables above than to maintain.
