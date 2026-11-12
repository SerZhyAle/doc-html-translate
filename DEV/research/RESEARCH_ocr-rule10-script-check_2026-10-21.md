# Rule 10 on the desktop: what the script check says over the corpus, and what an assumed `eng` does to Cyrillic

2026-10-21. Phase B (R10) of [`DEV/plan/done/E0121_ocr-owed-evidence-and-rule-10-reverification.md`](../plan/done/E0121_ocr-owed-evidence-and-rule-10-reverification.md).
Reads `OCR-OVERLAY` rule 10 and amendment 1.3 A against the desktop edition. Follows
[`ocr_plate_coverage_2026-08-13.md`](ocr_plate_coverage_2026-08-13.md) section 4 (the 6.4 floor). No threshold and
no decision of the script check was changed; one behaviour-neutral record was added.

## Question

Amendment 1.3 A lets a product draw plates on a language the reader did not choose only when it records evidence
that the page was read in a script that language can read. The desktop edition's evidence is the script check
(`resolveScript`, floor 6.4). Three things were open (ticket 120, `RESEARCH_ocr-evidence-mapping_2026-10-20.md`):

1. a silent or under-floor verdict, and a confirming one, left no record;
2. nobody had tabulated the verdicts since the 2026-08-13 measurement;
3. nobody knew what an assumed `eng` does to a Cyrillic page when the check says nothing, nor whether the rule 12
   discard record (the contract's second route, used by Lite and Android) separates that from an honest `eng` read.

## Answer

- **The record is in** (section "What was added"): the run log and the `DOCHT_OCR_DIAG` line now say where the
  language came from and what the check made of it, silent verdicts included.
- **The floor holds on 52 images.** 0 wrong corrections and 2 right ones; the highest wrong non-Latin verdict is still
  Arabic at 5.00, under 6.4, and the weaker right one is still 8.15.
- **The check misses 3 of the 5 Cyrillic images** (two silent, one at 2.92), and on all three an assumed `eng`
  draws **no plates at all**: the recognition floor refuses everything the English data reads from them. No
  debris was produced on a Cyrillic image the check missed. The two Cyrillic images that do get debris plates
  under `eng` (5 plates and 1 plate) are exactly the two the check catches.
- **The discard record does not separate them on this corpus.** The share of read letters that were refused is
  0.77-1.00 for Cyrillic under `eng` and 0.00-0.98 (median 0.56) for the other images that draw plates; with n=2 Cyrillic images
  that draw plates there is no threshold to derive (rule 13). Adopting that route here would need thresholds
  marked "chosen, not derived".
- **The route per edition is the owner's decision** (ticket 121 phase B). Evidence for it is in "Routes".

## What was added

`internal/ocr/script.go` (`LanguageRecord`, `newLanguageRecord`), `overlay.go` (`OverlayResult.Language`, stamped on
every recognition), `diag.go` (five optional fields on the image's line), `internal/pipeline/ocrstep.go` (one log
line). The outcome is derived from `resolveScript`'s own answer, so it cannot disagree with the decision:

| `scriptCheck` | Meaning |
|---|---|
| `chosen` | `-ocr-lang` was given; the check does not run |
| `no-verdict` | the check ran and said nothing (no osd data, too little text, a failed pass) |
| `under-floor` | a verdict under 6.4, not acted on |
| `agrees` | a verdict at or over 6.4 naming a script the language reads |
| `latin-ignored` | a Latin verdict over 6.4 while the assumed language reads another script, not acted on |
| `corrected` / `stopped` | the existing corrective and stopping cases |

The line carries `lang`, `langSource` (`chosen` / `assumed`), `scriptCheck`, and `script` / `scriptConf` when there
was a verdict; a line with no record stays byte-identical (the literal both editions pin, `goDiagLine`). One CLI
run each, `DOCHT_OCR_DIAG` set, 2026-10-21:

| Input | Run log line | Diagnostics line |
|---|---|---|
| `image.png`, no `-ocr-lang` | `language rus + eng assumed (not chosen with -ocr-lang); script check: Cyrillic at 8.24 - the language was corrected` | `lang=rus+eng langSource=assumed scriptCheck=corrected script=Cyrillic scriptConf=8.24` |
| `image.png`, `-ocr-lang eng` | `language eng chosen with -ocr-lang; script check: not run ..` | `langSource=chosen scriptCheck=chosen` |
| `1.png`, no `-ocr-lang` | `language eng assumed ..; script check: no verdict (no osd data, or too little text on the first image)` | `langSource=assumed scriptCheck=no-verdict` |

## The verdicts over the corpus

`DetectScript` over the 47 lab scenes plus `1.png`, `image.png`, `accounts.jpg`, `img-png_Nyoka-comic-page.png` and
a Russian marriage-certificate scan (52 images), `eng` assumed, the shipped staging (no upscale):

| Outcome | Images |
|---|---:|
| no verdict | 20 |
| under the floor | 27 |
| at or over 6.4, agrees | 3 (Latin 26.67, 23.33, 10.00) |
| at or over 6.4, acted on | 2 (Cyrillic 8.24 `image.png`, 8.15 `polish-soviet-propaganda-poster-18y`) |

The wrong verdicts are the ones the 2026-08-13 note found, reproduced: the English Archie cover as Cyrillic at
3.81, the 1887 English cartoon as Arabic at 5.00, the Polish poster as Latin at 3.97. Nothing new stands between 5.00
and 8.15, so the floor's bracket is unchanged at n=52 (it was 46 scenes).

Of the five images that are genuinely Cyrillic, the check catches two and misses three:

| Image | Verdict | Under assumed `eng`: plates | Under `rus`: plates |
|---|---|---:|---:|
| `image.png` (Russian UI) | Cyrillic 8.24 | **5**, debris (`Katanoru-nonyyarenn`) | 9, Russian |
| `polish-soviet-propaganda-poster-18y` | Cyrillic 8.15 | **1**, debris (`HF BBIBATb`) | 4, Russian |
| `a-propaganda-poster-from-the-soviet-union` | silent | 0 | 1, Russian |
| `poster-display-type-on-flat-colour` | silent | 0 | 2, Russian |
| Russian marriage-certificate scan | Cyrillic 2.92, under floor | 0 | 3 |

## What the discard record looks like

Per image, `Recognize` with an assumed `eng`: the share of the letters the engine read that the pipeline refused
(the confidence floor and the translatability gate - the record of rule 12).

| Set | Images | Share refused |
|---|---:|---|
| Cyrillic under `eng` | 5 (3 with no plates, 2 with) | 1.00, 1.00, 1.00, 0.96, 0.77 |
| everything else under `eng`, with plates | 40 | 0.00 to 0.98, median 0.56 |

Among the 40, 3 refuse 0.95 or more (`diaz-political-cartoon`, the Brandeis cartoon, the signed manuscript: one small
plate each) and 7 refuse 0.77 or more. A guard at 0.95 would catch the one Cyrillic image with debris plates
(`polish-soviet-18y`, 9 letters) and refuse 3 honest plates; at 0.77 it would also catch `image.png` and refuse 7.
The three Cyrillic images with no plates need no guard. With two positives there is nothing to bracket.

## Routes (for the owner)

- **Desktop.** The check already catches both Cyrillic images that would draw debris, and misses three on which
  `eng` draws nothing. The record closes the gap amendment 1.3 A names ("a confirming verdict is not recorded"). What is
  left is only that a silent or under-floor verdict is not evidence in the contract's words, so the product still
  proceeds without it: the exception stays a dated one until the catalog takes proposal P2 ("record, do not
  refuse" is acceptable) or the product refuses silent verdicts (which would refuse 20 of the 52 images here, among
  them nearly every single-line English image). The discard-record route adds nothing the check does not already
  do on this corpus.
- **Extension.** It has no check. The discard-record route is the only evidence route that needs no 10 MB OSD pack,
  but nothing here derives its thresholds, and the extension engine's behaviour on Cyrillic was not measured
  (tesseract.js was only run for the word-gap note). Record-only has nothing to record. A dated exception, or a
  measurement of tesseract.js on the five Cyrillic images first.

## What this does not say

- Desktop engine only, `eng` data from the system Tesseract (5.4.0.20240606); tesseract.js on Cyrillic is unmeasured.
- Five Cyrillic images, of which one is a UI screenshot and three are scans or posters with little text; no Greek,
  Arabic, Hebrew, Han or Hangul page is in the sample.
- The verdicts are `DetectScript` on the book's first image only, as `OverlayBook` does; a multi-page book whose
  first page is a cover is a different population.
- The refusal share counts letters of the lines the engine returned; it is one candidate signature, not the only
  one (kept confidence or letter run could separate better), and none was derived.

## Reproducing

`temp/ocr-r10/` (gitignored): `osd.jsonl`, `sig-eng.jsonl`, `sig-rus,rus_eng.jsonl`, `sig.mjs`. The harness was a
throwaway test in `internal/ocr` (`DetectScript` and `Recognize` per image, `DOCHT_R10_PROBE=<dir>`), deleted after
the run.
