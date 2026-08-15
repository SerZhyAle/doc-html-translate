# The rescue floor, re-measured over the corpus it now has - 2026-08-15

**Ticket:** [`2026-08-13_ocr-rescue-floor-drops-genuine-lettering`](../plan/2026-08-13_ocr-rescue-floor-drops-genuine-lettering.md)
**Question:** `ocrRescueLineConf` is 80, set between two points from one cycle (genuine rescued
lettering 93.1-97.0, hallucinated lettering 50.8). The corpus has grown since. What does the
distribution look like now, and what floor does it actually support?
**Scope:** the 46 lab scenes and the 13 annotated ones. No rendering: the question is what the
recognizer returned and what the floor did with it.

## The instrument had to be built first, and that is half the finding

The floor was **unmeasurable** before this cycle. Nothing in the app or the lab recorded a line the
floor rejected, so a scene where a correctly read word was thrown away looked exactly like a scene
where the recognizer found nothing - and the only way to re-derive the number was to guess again.

So the first change is the record: `Result.Dropped` / `droppedLines`, written through the same
`keepLine` predicate the gate itself asks, into the `DOCHT_OCR_DIAG` sidecar - **including for a
page that produced no plates at all**, which is precisely the case that used to write nothing.
Everything below is read off that record.

## 1. The old band's gap is not empty any more

Every rejected line on the 13 annotated scenes, converted with the language the annotation declares
(so the recognizer *can* read the lettering) and classified against the annotation's own transcript
- a line is genuine when at least half its 3+ character tokens appear in the transcript:

| conf | floor | verdict | scene | text |
|---:|---:|---|---|---|
| 73.9 | 80 | not in the transcript | `poster-display-type-on-flat-colour` | `ОБ ЗЛОМ` (a misread of `ОБ ЭТОМ`) |
| 69.2 | 80 | **genuine** | `poster-display-type-on-flat-colour` | `ЗАЧЕМ` |
| 42.8 | 50 | **genuine** | `samson-and-delilah-03-scroll` | `HEARTLESS, ALLTHE` |
| 42.4 | 50 | not in the transcript | `samson-and-delilah-03-scroll` | `VERTH OF --=— 2` |
| 41.2 | 80 | **genuine** | `poster-display-type-on-flat-colour` | `ВЗРОСЛЫЕ` |
| 32.8 | 50 | **genuine** | `samson-and-delilah-03-scroll` | `FHE SMALL TOWN OF _` |
| 26.5 | 80 | no letters | `synth-caption-on-gradient` | `—` |
| 8.4 | 50 | not in the transcript | `samson-and-delilah-03-scroll` | `VELA!  §` |

**Genuine runs 32.8-69.2. Not-genuine runs 8.4-73.9. They overlap.** The highest non-genuine line
sits *above* the highest genuine one. So there is no value of a single confidence floor that admits
the poster's first word and rejects the misread beside it, and moving the number would trade one
scene's loss for another's. That answers the ticket's first bullet: the band was re-measured, and
what it says is that the band does not exist.

`le-petit-journal-1908-map-labels` is annotated `fr` and no French data is installed on this
machine, so it contributes nothing here. Stated rather than quietly dropped.

## 2. What does separate them is length, and it is not close

The same record over **all 46 scenes**, converted with the lab's default `eng` - which is also the
app's default, and therefore the condition under which invention actually happens. The rescue floor
rejected **175 lines**. Sorted by confidence, the ones a lower floor would admit first are:

```
77.9  "\"            70.8  "e\\)"        69.0  "Cor"        66.5  "in"
77.4  "XO" ET"       69.2  "i"           68.1  "4"          64.5  "yar"
```

Every one is debris of one to six characters, on three scenes: the two Polish-Soviet propaganda
posters (Cyrillic read with English data) and `louis-brandeis-nomination-political-cartoon`.

Requiring an unbroken run of **4 letters** leaves 9 of those 175:

| conf | scene | text | what it is |
|---:|---|---|---|
| 58.3 | `polish-soviet-propaganda-poster-18y` | `KPECTbAHHH!` | **genuine** - КРЕСТЬЯНИН!, transliterated |
| 36.1 | `polish-soviet-propaganda-poster-18y` | `allie` | invention |
| 35.7 | `polish-soviet-propaganda-poster-18y` | `floAbCKUA NOME` | genuine - ПОЛЬСКИЙ ПОМЕЩИК, transliterated |
| 33.5 | `louis-brandeis-nomination-political-cartoon` | `aati` | invention |
| 29.7 | `polish-soviet-propaganda-poster-19y` | `gona` | invention |
| 24.0 | `polish-soviet-propaganda-poster-18y` | `veer 4 Tae Cramay I` | garbled |
| 21.8 | `polish-soviet-propaganda-poster-18y` | `PoccHicKAS COUMANKCTHYECKAA ..` | genuine - the poster's banner |
| 6.9 | `polish-soviet-propaganda-poster-18y` | `TIpomerapuu Bce.. °TPAH, ..` | genuine - Пролетарии всех стран.. |
| 4.4 | `louis-brandeis-nomination-political-cartoon` | `appomtment!` | genuine - a misread of "appointment!" |

**Between 36.1 and 58.3 there is nothing at all.** That is the empty band the old pair of points was
supposed to have and no longer did. Above it: one line, genuine. Below it: genuine and invented
interleave, so nothing there is separable by any rule measured here.

## 3. The rule the evidence supported, and why it does not ship

A rescued line kept when its mean confidence clears `ocrRescueLineConf` (80), **or** when it carries
a run of **4** letters and clears **47** - the middle of the measured empty band, arithmetic because a
confidence is a percentage. 4 letters and not 3: at 3 the band closes, since `Cor` (69.0) and `yar`
(64.5) are debris that clears it. Corpus-wide under `eng` the pair admits exactly **one** of the 175
rejected lines, `KPECTbAHHH!` at 58.3, which is real lettering. On paper that is a clean result.

It was implemented in both editions and run over the dev split against the baseline
`temp/ocrlab/20260815-190756`. **The corpus rejected it.**

| | baseline | with the word rule |
|---|---:|---:|
| mean recall / precision | 0.6154 | 0.6154 |
| mean CER | 0.1459 | **0.2236** |
| mean IoU | 0.7756 | **0.7388** |
| mean covered | 0.6765 | 0.7086 |
| worst residual ink | 0.9992 | 0.9148 |
| clipped | 0 | **5** |
| cross-group | 6 | **10** |
| protected-area damage | 0 | 0 |
| merges / splits | 1 / 0 | 1 / 0 |

Every moved number is **one scene**: `poster-display-type-on-flat-colour`, which under the lab's
default `eng` produced no plates before and now produces one. Its text is `TPAXATBCR: 4 y` - the
poster's own headline transliterated by an English recognizer - in a box 782x310 px, and the rendered
page shows it covering the poster from under `ЗАЧЕМ` down across `ОБ ЭТОМ`. That is the failure
mode the rescue floor exists to prevent, arriving on the one input class where the app's default
language is wrong.

Two things about that row are worth keeping straight:

- **The `clipped` count is not a clipping defect.** The plate overshoots its box by exactly 5 px in
  every stress case, one pixel past `ClipSlackPx` (4, itself measured as layout-rounding residue),
  and the rendered screenshot shows the full translated string. Widening the slack to make this pass
  would be moving a number until one scene passes, which the lab's own rules forbid, so the count is
  reported as it stands and attributed here instead.
- **The regression is the plate, not the arithmetic.** Recall and precision do not move, because the
  new plate matches no annotated group either way.

So `ocrRescueLineConf` stays at **80** and the gap stays open. What ships from this cycle is the
instrument and the measurement.

**The revert is verified, not assumed.** `temp/ocrlab/revert-check`, same dev split: identical to
the baseline `temp/ocrlab/20260815-190756` on every scored field - recall, precision, CER, IoU,
covered, residual, halo, contrast, merges, splits, protected-area damage, clipped, cross-group and
drift. The only number that moved is total OCR wall-clock (40 438 ms to 42 656 ms), which is machine
noise.

## 4. What the poster costs either way, measured

Read with `rus` - the language the annotation declares - the rule does what the ticket asked:

- the headline group reads **`ЗАЧЕМ ТРАХАТЬСЯ:`**, exactly the annotation's transcript. Before,
  `ЗАЧЕМ` was discarded at 69.2, the plate reached IoU 0.455 against the group, under the lab's 0.5
  detection floor, and the group scored a miss.
- **the body becomes two plates where it was one.** The rule also recovers `ОБ ЗЛОМ` (73.9), and PSM
  11 returns lines "in no particular order" - that row comes back at y=1694 in a list whose next entry
  is y=1509. `clusterLines` walks the engine's order by design (`docs/PARITY.md`: reading order is
  entirely the engine's), so the out-of-order row ends the running cluster. No word is lost and both
  plates stay inside the annotated body group.
- `ВЗРОСЛЫЕ` at 41.2 is genuine and stays rejected: it sits below the band, where genuine and
  invented interleave.

The scene therefore improves under the right language and regresses under the default one, and the
default is what a reader who never opens the language setting gets. That asymmetry is the finding.

## 5. What this does not settle

- **The extension edition is not measured here.** Its lab runner reads plates back out of the DOM and
  does not persist the discard record, so closing that needs an `evidence.Scene` field and a
  `SchemaVersion` bump on both sides. The record itself is ported with mirrored tests; what is
  missing is the ability to *check* the distribution on tesseract.js, whose confidences differ.
- **The next attempt needs a third axis.** Confidence does not separate the populations and length
  alone lets a wrong-language transliteration through. The candidates the ticket already named -
  agreement between rungs, the line's fit to the page's own type sizes - are untested, and so is the
  most direct one: not running the rescue ladder at all when the script check says the language is
  wrong, which is where the debris on this poster actually comes from.
- **Three-letter words are collateral of any length rule.** No scene in the corpus loses one today.
