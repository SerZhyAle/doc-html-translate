# The recognizer sometimes hands back a "line" that is two texts, and nothing downstream recovers

2026-09-12. Feeds
[`DEV/plan/done/2026-09-12_ocr-line-stitched-across-the-picture.md`](../plan/done/2026-09-12_ocr-line-stitched-across-the-picture.md).

Reported against the browser extension on `test_doc/1.png` - a photograph with nine comic speech
balloons laid over it, five on the left of the figure and four on the right. The overlay produced
three grey bars across the whole width of the picture, each carrying one speaker's words run into
the other's:

> "It's about your facial hair. I love everything about you, but that patchy mustache and beard.. it's
> just not working for me. **Oh, come on, Em. It's**"

## Where it comes from: not our clustering

The defect is in the recognizer's own line assembly, and it is identical in both editions. Raw
level-4 rows, PSM 3, `eng` - `tesseract 1.png tsv --psm 3` (desktop engine) and tesseract.js 7 in
Node (extension engine) return the same boxes on a 2048x2048 image:

| Line box | Width | What it is |
|---|---:|---|
| `x 181..1774  y 595..618` | 1593 | "just not working for me." + "Oh, come on, Em. It's" |
| `x 228..1215  y 920..938` | 987 | "stylish, and that facial hair just" + a stray "4" |
| `x 312..1944  y 940..967` | 1632 | "doesn't fit." + "But I feel more confident with it." |
| `x 219..1881  y 1348..1368` | 1662 | "us both happy. And honestly, a" + a stray "." |
| `x 207..1934  y 1366..1392` | 1727 | "clean-shaven look would suit you" + "been working on." |

Layout analysis walked across the photographed figure and joined text from the left column to text
from the right into one line box.

**No rule we have can recover from that**, and the three that look like they should each fail for a
different reason:

- The clustering's column test (`overlap * 10 >= narrower`) sees a *real* x-overlap, because the
  stitched box honestly spans both columns. It joins them.
- `OCR_MAX_PLATE_COVERAGE` (0.52) does not fire: the bar is wide but short. The first one measures
  1593x105 px on a 2048x2048 image - **0.04** of it.
- The pitch and type-size rules compare a line with its neighbours, and by this point the two texts
  are not neighbours; they are one line.

So the repair has to happen on the recognizer's output, before the clustering is handed it.

## The rule, and where its threshold comes from

A line is cut between two consecutive words whose boxes stand more than `OCR_MAX_WORD_GAP_RATIO`
times the line's **median word height** apart. Word height rather than the line box for the same
reason `inkHeight` uses it: the box is the union of the words, so one tall artefact sets it for the
whole line.

Measured over all 46 scenes of the lab corpus (`DEV/ocrlab/corpus.json`) plus `test_doc/1.png`, with
tesseract.js 7 at PSM 3 and `eng` - **307 multi-word lines, of which 199 clear `OCR_MIN_LINE_CONF`
(50)**. Only those 199 can reach a plate, so only those 199 can be damaged or repaired; the other
108 are dropped by the floor either way. For each line: the largest gap between two consecutive word
boxes, over that line's median word height.

**The largest gap inside a line that really is one line:**

| Ratio | Conf | Scene | Line |
|---:|---:|---|---|
| **2.57x** | 71 | `join-the-ranks-of-the-red-army-russian-propaganda-poster-1920` | "Proletarjusze wazystkich krajow taczcie sie" |
| 1.88x | 90 | `ludwig-hohlwein-tyskland-1936-iv-vinter-olympiade` | "ORGANISATIONS-KOMITEEN FOR DEN IV. VINTER-OLYMPIADE 1936" |
| 1.69x | 81 | `le-petit-journal-balkan-crisis-1908` | "LE REVEIL DE LA QUESTION D'ORIENT" |

The 2.57x is a slogan across the foot of a poster measured at 18 px over a 7 px median word height,
so it is also the noisiest number in the table - at that scale two pixels of letter-spacing move the
ratio by 0.3. It is used as the bound anyway, because bracketing against the *second*-largest would
be choosing the measurement that suits the answer.

**The narrowest cross-region stitch above it:**

| Ratio | Conf | Scene | Line |
|---:|---:|---|---|
| **4.80x** | 86 | `samson-and-delilah-15` | "STONE! LEPHANTS, THEN BRING" (two balloons) |
| 12.4-17.6x | 96 | `synth-two-columns` | all three of its lines - the corpus's one known merge |
| 19.9x | 81 | `le-petit-journal-balkan-crisis-1908` | a masthead's three separated items |
| 38.6x | 93 | `chicken-little-1961-political-cartoon` | "Mon., Jan. 9, 1961" + "ST.LOUIS POST-DISPATCH" |
| 36-100x | 77-95 | `test_doc/1.png` | the five lines in the table above |

**`OCR_MAX_WORD_GAP_RATIO = 3.5`** - the geometric middle of 2.57 and 4.80, ~36% of margin each way.

### What the corpus says afterwards

`npm run ocrlab -- --scene synth-two-columns`, the extension edition, against the scene's own
hand-drawn ground truth (two reading groups, three lines each):

| | Plates | Geometry |
|---|---:|---|
| Before (`temp/ocrlab/ext-0818`) | **1** | `[49,60 578,145]`, 529 px wide on a 720 px image - it crosses the gutter |
| After | **2** | `[49,60 237,147]` and `[397,60 578,145]` - one per column, texts matching both transcripts verbatim |

That is the `merges` hard gate in `DEV/ocrlab/thresholds.json` - fixed at 0 by the strategic spec and
standing at 1 since the baseline - going to 0, with no split traded for it.

### The band that is deliberately left alone

Between 1.87x and 2.57x the two populations **overlap**, and no ratio separates them:

| Ratio | Conf | Scene | Line | Verdict |
|---:|---:|---|---|---|
| 2.57x | 71 | `join-the-ranks-of-the-red-army` | the slogan above | one line |
| 2.48x | 86 | `samson-and-delilah-15` | "WE CANNOT CONTINUE a WE HAVE NO OTHER. \|" | stitch |
| 2.12x | 90 | `samson-and-delilah-15` | "GIVE ME A THOUSAND MEN TO) THOUSAND MEN," | stitch |
| 1.88x | 90 | `ludwig-hohlwein-tyskland-1936` | the committee line above | one line |
| 1.87x | 87 | `samson-and-delilah-15` | "FEARED! I MUST] HIS BRIDE!" | stitch |

Comic balloons drawn side by side sit that close together, and a geometric rule must not pretend to
know. Separating them needs evidence from the pixels *between* the two words - a balloon outline, a
change of ground - which is Phase 07 Step 07.3's boundary test, not a ratio. The consequence is
stated rather than hidden: **this rule leaves the adjacent-balloon stitches on
`samson-and-delilah-15` and `samson-and-delilah-03` merged**, and the nearest one it declines is
3.46x ("HEAD FOR THE YOU, HE WIkL | ISAMSON, FOR THE L055 | LAST").

## What the rule changes over the corpus

At 3.5x, **36 of the 199 kept lines are cut, on 9 of the 46 scenes**. Every one of them was checked
by eye against its scene, and every one is a genuine cross-region stitch: comic balloons
(`samson-and-delilah-03/15`, `cover-of-archie-and-me-no-1`), newspaper mastheads and datelines
(`le-petit-journal-balkan-crisis-1908`, `chicken-little-1961`), a shop window and a wall of
handwritten notices photographed at an angle (`crowdstrike-outage-at-*`,
`handwritten-notices-regarding-the-baltimorelink-*`), and all three lines of `synth-two-columns` -
**the one merge `DEV/ocrlab/thresholds.json` names in its grouping baseline**. No line that a reader
would call one line is cut.

## Cutting alone makes it worse, and that is the second half of the fix

`clusterLines` walks its input once and closes the open plate the moment a line does not belong to
it, so it needs a column's lines to arrive together. That is exactly what the engine stops providing
once a stitched line is cut: the runs interleave left, right, left, right down the page, and a stray
noise token lands between two lines of the same balloon.

So the runs of a page that was cut are regrouped into columns (x-overlap, the clustering's own test)
and handed over column by column, top to bottom. A page nothing was cut on keeps the engine's own
order, untouched.

Measured on `test_doc/1.png` in the extension edition - nine balloons, five left and four right:

| | Plates | The left balloon at y838-967 |
|---|---:|---|
| Before | 9 | one plate - a 1593 px bar across the figure, carrying both speakers |
| Split only | 12 | **three** fragments - "..just", "doesn't fit.", and a noise plate |
| Split + `orderColumns` | **10** | one plate, boxed to its own lettering |

Ten plates over nine balloons: the only extra is the tail "better.", which its balloon's own line
pitch separates. The desktop edition produces the same ten, with the same text.

Two things were got wrong on the way to that, and both are worth recording because both were
invisible until the corpus was run:

- **The scope is the page, not the recognizer paragraph.** Regrouping each paragraph on its own was
  the first attempt, and it left `synth-two-columns` at four plates over two columns: the engine puts
  the third row of *both* columns into a second paragraph, so the runs still interleaved across the
  paragraph boundary. `clusterLines` deliberately merges across those boundaries - the engine invents
  them mid-column - so the reordering has to as well.
- **A line the confidence floor will drop must not form a column.** On `test_doc/1.png` the engine
  also returns an empty `1982x1864 px` "line" across the whole picture. It reaches no plate, but as a
  column it overlaps both real columns and chains them into one, which sorts the page straight back
  into the interleaving. Columns are therefore formed only from lines that clear `OCR_MIN_LINE_CONF`,
  and the rest are parked at the end where only the discard record reads them. The desktop edition
  fragmented into **19 plates** until this was fixed; the extension edition did not, because
  tesseract.js does not return that line - which is exactly why both editions had to be run.

## Reproducing

The measurement harness was a throwaway under the gitignored `temp/` and is not committed. To
rebuild it: run tesseract.js (the extension's own pinned version and its `vendor/tesseract/lang`)
over every `file` in `DEV/ocrlab/corpus.json` with `tessedit_pageseg_mode: "3"`, walk
`blocks -> paragraphs -> lines -> words`, and for each line with two or more words record
`max(words[i].bbox.x0 - words[i-1].bbox.x1) / median(word heights)` together with the line's
confidence. The desktop engine's equivalent is the level-4/5 rows of
`tesseract <scene> stdout --psm 3 -l eng -c tessedit_create_tsv=1`.
