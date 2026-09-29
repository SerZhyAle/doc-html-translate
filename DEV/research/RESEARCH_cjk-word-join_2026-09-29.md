# CJK word join: where a recognized CJK line gets its spaces back, and which ones are real - 2026-09-29

**Ticket:** [`75_2026-09-29_cjk-ocr-plates-split-by-spaces`](../plan/75_2026-09-29_cjk-ocr-plates-split-by-spaces.md)
**Question:** both editions build a line's text by joining the recognizer's words with a space
(OCR-PIPELINE 2.3). Japanese and Chinese are written without spaces, and Korean spaces words, not
syllables, but Tesseract cuts CJK lettering into single characters or short runs. Which spaces
should a line keep, and what does the corpus say about Korean, where some of the spaces are real?
**Answer:** a pair of words that meets on Han, kana or CJK punctuation joins with no space; a pair
that involves Hangul joins only when the gap between the two boxes is under **0.33** of the
median word height; every other pair keeps its space. Plate lines join the same way on Han, kana
and CJK punctuation, while a Hangul line break keeps its space. Landed as `OCR-PIPELINE` amendment
1.8 in both editions.

## Method

The join rule reads only a line's words: their text and their boxes. So it can be measured from the
words, without running the rest of the pipeline once per candidate rule. A throwaway harness in
`internal/ocr` (deleted after the run) read each page the way the ordinary pass reads it
(`prepareForOCR`, `tesseractArgs` at PSM 3, `tsvLines`), kept the lines that clear `ocrMinLineConf`,
and rebuilt each line's text under every candidate rule.

- **Pages:** the 11 pages of the episode 6 CBZ plus the hi-res page 3, per language:
  `ja-cbz-peppercarrot-e06` + `ja-comic-peppercarrot-p03` (jpn), the zh pair (chi_sim), the ko pair
  (kor). Pepper&Carrot by David Revoy, CC BY 4.0.
- **Truth:** the author's own lettering lines from the lang-pack SVGs, as the draft expectations in
  `DEV/doccorpus/expect/` carry them (both editions' output never becomes truth). Lines of one
  character are skipped: 270 ja, 155 zh and 235 ko lines remain.
- **Metric:** the exact-substring rate - how many lettering lines appear verbatim in the pages'
  rebuilt line texts. `letteringRecall` (the corpus report's character recall) ignores spaces by
  construction and cannot see this defect at all.
- **Pair labels (Korean):** each pair of consecutive words meeting on Hangul is labelled from the
  truth - `space` if `a b` occurs in the lettering and `ab` does not, `none` for the reverse, else
  unlabelled. 480 of 539 pairs are labelled (185 `space`, 295 `none`). The label is noisy for
  one-syllable tokens that occur in many words, so it is used for the error curve only; the
  exact-substring rate is the decision metric.

## Results

Exact lettering lines matched, per rule:

| Rule | ja /270 | zh /155 | ko /235 |
| --- | --- | --- | --- |
| every space kept (before) | 1 | 0 | 19 |
| Han + kana joined | 46 | 21 | 19 |
| Han + kana + Hangul joined outright | 46 | 21 | 13 |
| Han + kana joined, Hangul by gap < 0.33 x median | 46 | 21 | 46 |
| the same + CJK punctuation joined like Han (**chosen**) | **52** | **25** | **46** |
| the same + ASCII punctuation after a CJK character joined | 47 | 25 | 47 |

Korean, by the Hangul gap bound (fraction of the line's median word height), with CJK punctuation
out of the picture:

| Bound | 0.20 | 0.22 | 0.24 | 0.26 | 0.28 | 0.30 | 0.32 | 0.34 | 0.36 | 0.38 | 0.40 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| exact lines /235 | 42 | 43 | 42 | 43 | 43 | 46 | 47 | 47 | 48 | 44 | 42 |
| wrongly joined pairs | 1 | 1 | 3 | 4 | 5 | 12 | 15 | 18 | 22 | 33 | 46 |
| wrongly spaced pairs | 96 | 78 | 66 | 61 | 46 | 38 | 32 | 27 | 22 | 19 | 16 |

The labelled gaps separate cleanly below 0.28 (249 `none` against 5 `space`), and the two
populations overlap from there to about 0.8, which is why no bound brings the error count to zero. The
exact-substring rate holds a plateau of 46-48 from 0.30 to 0.36 and falls on both sides (43 at 0.28,
44 at 0.38). **0.33** is the geometric middle of the plateau's two ends. At 0.33 itself the rebuilt
text matches 46 lines.

- Joining every Hangul pair is worse than keeping every space (13 against 19): the ticket's premise
  that "dropping them all is as wrong as keeping them all" holds, measured.
- CJK punctuation (the CJK Symbols and Punctuation block, the katakana middle dot and prolonged
  sound mark, the halfwidth and fullwidth forms) is Common script, so a rule stated on scripts alone
  leaves `市长 ， 我` and `ケ ー キ`. Joining it like Han lifts ja from 46 to 52 and zh from 21 to 25.
- ASCII punctuation after a CJK character is **not** joined: that drops ja from 52 to 47 (the
  Japanese lettering sets `!` and `?` after a space as often as not), for one extra Korean line.
- Plate lines (not measured by this metric, which compares line by line) join on the same Han /
  kana / CJK punctuation rule: CJK text wraps with no space at the break, and Korean lettering
  breaks its lines between words - every multi-line Korean balloon of the lang-pack does.

## What else reads the text, and the regression check

A full `Recognize` snapshot of 44 images - the 36 CJK pages above, `ja-rakuten-tokyo-1902`,
`zh-sanmimzhuyi-cartoon`, `ko-spring-cartoon-1933`, the en / fr / ru page 3 and the two images at
the top of `test_doc/` - was taken on the unchanged code and after the change.

The first after-snapshot was **not** spacing-only on one page: `ja_P02` returned a different
reading (`魔法業` for `魔法薬`, every box a pixel off, one plate fewer). The page's ordinary pass
plates nothing, so the rescue ladder picks a rung by `resultStrength`, and that counted the plate
text's space-separated fields - with CJK joined, a Japanese plate counted as one word per line and
another rung won. The comparator now counts the recognizer's own words (`Block.tokens`, the
extension's block `tokens`), which is exactly what the fields counted before the join. After that
fix: **8 images identical byte for byte** (every Latin and Cyrillic image, and the CJK images with no
joinable pair), **36 identical except for whitespace** (same plates, same boxes, same characters),
**0 other differences**. Both runs are deterministic (three repeats of the after state and two of
the before state on `ja_P02` gave identical output).

A second reader of the line text surfaced only in the extension edition, in the corpus run through
headless Chrome: on the same `ja_P02` its plates still changed after the comparator fix (121 plates
over the CBZ against 118 on a clean HEAD worktree, `魔法業` against `魔法薬`). The anchored rescue
admission's letter-run condition (amendment 1.6, "a run of at least 4 letters") read the line's
text, and a joined Japanese line is one long run where the spaced one was many short ones, so lines
the admission had refused were admitted. The run is now the longest run of any one recognized word
(`(*ocrLine).letterRun`, `lineLetterRun` / the line's `letterRun`), which is what the spaced text's
run was; the extension then matched HEAD exactly, 118 plates, 0 differences ignoring whitespace.
