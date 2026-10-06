# doc-html-translate: OCR overlay loses lines of a paragraph and adds stray marks (found from FastMediaSorter Lite)

| | |
| --- | --- |
| Reported | 2026-10-04, from the FastMediaSorter Lite side (the "translate in the browser" button, `OCR-INVOCATION`) |
| Product / version | `doc-html-translate` 26.0930.1100 (CLI, `-version`); the page the viewer opened carried the nav label v26.0930.1107 |
| Engine | `C:\Program Files\Tesseract-OCR\tesseract.exe` 5.4.0.20240606, language `eng` (the viewer sends no `-ocr-lang` while its source language is `auto`, so the default applies) |
| Reproduced | Twice, byte-identical box lists, from the CLI with `-notranslate`, so the translation stage is not involved |
| Contracts touched | `OCR-OVERLAY` rules 5, 7, 12; `OCR-PIPELINE` sections 2.3-2.5 |

## Summary

On one real page (a two-panel picture: a photo on the left, a column of coloured dialogue lines on the right) the
overlay is wrong in four ways. The worst one drops two lines of a five-line paragraph: the plate covers three
lines, the translation stops mid-sentence, and the remaining two source lines stay on the page, readable and
untranslated, next to the translated text. Tesseract itself reads both lost lines at ~96 % confidence, so they are
lost inside the product's pipeline, between the engine and the plates.

## Reproduction

```
doc-html-translate.exe scene.jpg -ocr -notranslate -noopen -force -folder out
```

Scene: a 4384 x 3072 JPEG. **Not attached**: the content is unsuitable for a public issue. It is on the owner's
machine (`\\p7\_i\output\d\dm3ns3u-265747c7-57c9-43ea-80e4-3221f3a062f0.jpg`); a neutral synthetic repro is
suggested at the end. Layout, in percent of the image: the photo occupies x 0-48.7; the text column starts at
x ~49.1 and runs to the right edge; nine lines or paragraphs on a light background in four ink colours (green,
black, red, blue).

Result: 9 `ocr-box` spans in `out/scene/index.html`. Left edge `x`, top `y`, width `w` and min-height `h` are the
span's own style values, in %; `fs` is `font-size` in `cqw`.

| # | mode / conf | x | y | w | h | fs | What it is |
| - | - | - | - | - | - | - | - |
| 1 | fill 0.57 | 49.22 | 1.33 | 33.94 | 2.31 | 1.15 | one-line green heading |
| 2 | fill 0.48 | **46.62** | 7.68 | 50.02 | 8.85 | 1.15 | three-line black paragraph |
| 3 | mask 0.67 | 49.16 | 27.08 | 49.79 | **8.82** | 1.20 | **five-line black paragraph, covered as three lines** |
| 4 | mask 0.94 | 49.11 | 52.93 | 49.48 | 8.85 | 1.18 | three-line black paragraph |
| 5 | mask 0.86 | 49.11 | 65.82 | 49.79 | 8.85 | 1.18 | three-line blue paragraph |
| 6 | fill 0.22 | 49.13 | 78.81 | 41.01 | 2.31 | 1.15 | one-line black line |
| 7 | mask 1.00 | 49.09 | 84.31 | 49.68 | 15.33 | 1.15 | five-line blue paragraph (covered correctly) |
| 8 | fill 0.52 | 49.22 | 20.70 | 44.84 | 2.28 | 1.15 | one-line red line (**read correctly - good**) |
| 9 | mask **0.19** | 48.65 | 45.93 | 10.01 | 2.64 | **1.70** | two-word red line |

## Defects

### D1 (high): two lines of a paragraph are lost; their source text stays visible and untranslated

- Box 3 holds the first three lines of a five-line paragraph. Its text ends mid-sentence after the third line; its
  `mask` background has exactly three line rectangles; its height is 8.82 % where five lines need ~15 % (box 7, also five lines, is 15.33 %).
- Lines 4 and 5 (source y 1130-1300 px of 3072, x 2155-4320 px) are in **none** of the nine boxes.
- Tesseract reads them: `--psm 11` returns every word of both lines at 93-97 % confidence, `enjoy ... months!`,
  FastMediaSorter's own engine (Tesseract 5.2 through the .NET wrapper, page segmentation mode 3) also returns
  both as ordinary lines (confidence 0.94-0.96). So the loss happens after recognition.
- The first word of line 4 is preceded by a stray token at the left edge: `Vy,` at 2081,1106 (55 x 100 px,
  conf 23.8; `--psm 6` gives `Y/` at 2081,1110, 52 x 104, conf 33.9). It is 1.4x taller than its neighbouring
  words (71 px). It has letters, so the artefact-word rule (no letter, no digit, taller than the median) does not
  remove it. This is the likely trigger, but it is a lead, not a finding: it was not confirmed inside the product.
- Effect for the reader: the translation of the paragraph is cut off, and the two untouched source lines sit under
  the translated plate, so both languages are on screen at once. `OCR-OVERLAY` rule 7 (the plate conceals its
  source) is not met for that paragraph.
- Rule 12 asks that a silent decision leaves a record. Whichever stage dropped the two lines, the discard record
  should say which gate it was; if it says nothing, the loss is in clustering or merging and the record has a gap too.

### D2 (medium): stray marks inside recognized text

- Box 2 reads `... Now come ‘on Tyler ...`: a `‘` before the word that starts the second line. The picture has no such
  mark, and none of the plain Tesseract runs (`--psm 3`, `6`, `11`) produce a `‘` at that spot, so it is probably
  introduced by one of the product's own passes (a preprocessed or rescue pass, or the line merge) - to be checked.
- Box 9 reads `‘What, WHY?`, same kind of leading `‘`.
- Box 7 reads `my "“handsy"`: a doubled opening mark. Here Tesseract itself returns `"“handsy"` at conf 18.5
  (engine level), and box 1 starts with `“"KNOCK,KNOCK!"` (conf 31.6), also engine level. A stray quote glyph at low
  confidence glued to a real quote is the same class as the bare pipe token that `OCR-PIPELINE` amendment 1.5
  already treats as a misread.
- Effect: the marks are translated along with the text and show up in the target language.

### D3 (medium): a plate extends past its text column

- Box 2 starts at x = 46.62 %, while the other eight boxes of the column start at 49.09-49.22 %. The text of that
  paragraph starts at ~48.5 % (the line-start artefact `/eager` at 2126 px). The plate therefore spills about 2.5 %
  of the image width to the left, onto the photo.
- Likely the same cause as D1: an edge artefact is allowed to widen the plate box. `OCR-OVERLAY` rule 5 asks for
  boxes built from line boxes tightened to the words that survive the artefact rule.

### D4 (low): odd font size and a very low mode confidence on a two-word line

- Box 9: `font-size` 1.70cqw against 1.15-1.20cqw for all eight other boxes, though the line is the same size and weight
  as the red line in box 8 (1.15cqw). The line is `Word, WORD?`, a lowercase word followed by a capitalised one;
  the median over two word heights is not a robust type size. Hypothesis, not confirmed.
- Its mode confidence is 0.19 (box 6 is 0.22), yet the mode was applied. A mode decision at that confidence is
  closer to a guess than to a measurement.

### D5 (low): box order is emission order, not reading order

- Boxes 8 (y 20.7 %) and 9 (y 45.9 %) come last in the document, after box 7 (y 84.3 %). They are the lines the
  later pass added. In the DOM, that is the order Chrome's page translation, text search and a screen reader walk
  the text in, so a line from the upper half is read after the last paragraph. Painting order for overlapping
  plates is a separate, already documented rule; this is about text order.

## What is fine (so it is not "fixed" by accident)

- The red line in box 8 is read correctly (`... in this dress, and this is all you fault!`, conf 0.52) - better than
  the in-app engine of the viewer does on the same pixels. Whatever finds coloured text here works; keep it.
- Boxes 4, 5 and 7 are read, grouped and covered as whole paragraphs. Box 7 shows the five-line case works when no
  edge artefact is present.

## Suggested checks

1. Dump the rule 12 record for the scene (the `ocrlab` runner) and see which gate, if any, took lines 4-5 of box 3;
   look at the `Vy,` token at 2081,1106 specifically.
2. Find which pass introduces `‘` before `on` and before `What` (compare the box text after each pass).
3. Add a synthetic scene to the lab: a five-line paragraph whose fourth line starts with a tall, low-confidence
   token that contains a letter. If the loss reproduces, it is the case for the artefact rule to cover.

## Not verified

- No translated run was compared; `-notranslate` was used on purpose to keep the translator out of it.
- The two version labels (CLI 26.0930.1100, page footer 26.0930.1107) were not reconciled; the viewer may be
  launching a different copy of the executable than the one used here.
- The product's source was not read; every statement about its internals above is inferred from its output.
