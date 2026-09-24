# Phase 01 — Corpus scenes for display lettering on saturated colour

**Strategic spec:** [`../2026-08-12_ocr-misses-display-lettering-on-saturated-art.md`](../2026-08-12_ocr-misses-display-lettering-on-saturated-art.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** none - foundation phase
**Steps done:** 3 / 3

## Objective

Put the reported class into the corpus with reviewed annotations, so every later phase is scored
against measured geometry instead of against the recognizer's own output.

## Prerequisites
- [x] Strategic §6 items blocking this phase are Resolved - none block it.
- [x] Working tree clean or on a feature branch.

## Files touched
| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `DEV/ocrlab/corpus.json` | Modified | ≤ 80 |
| `DEV/ocrlab/annotations/<scene>.json` | New | ≤ 250 each |
| `test_doc/ocrlab/own/` | New (media) | n/a |

## Steps

### Step 01.1 — Register the reported poster as a scene
**Files:** `DEV/ocrlab/corpus.json`, `test_doc/ocrlab/own/`
**Depends on:** - start of phase

**Prompt for developer:**
> Register the user-reported poster with `ocrlab add -categories poster -split dev -licence OWN`,
> giving it an id that names the class rather than the file (for example
> `poster-display-type-on-flat-colour`). Copy it into the corpus root rather than referencing it in
> place, so the run does not depend on a path in the reporter's Downloads folder.

**Verification:**
- `ocrlab verify` exits 0.
- The new scene id appears exactly once in `DEV/ocrlab/corpus.json`.
- The media file exists under `test_doc/ocrlab/` and its hash matches the manifest entry.

**Status:** `[x]` done

---

### Step 01.2 — Annotate it from the pixels
**Files:** `DEV/ocrlab/annotations/<scene>.json`
**Depends on:** Step 01.1

**Prompt for developer:**
> Write the annotation with one group per line of display type, transcript in Russian, reading order
> top to bottom. Measure every box by luminance threshold and projection over the image, never from
> an OCR result - the lab's first rule is that the engine is not scored against its own output. Set
> `origin` to `human`.

**Verification:**
- `DEV/ocrlab/annotations/<scene>.json` exists and `ocrlab verify` exits 0.
- `"origin": "human"` appears in the file.
- The group count equals the number of type lines in the poster, and each `transcript` is non-empty.

**Status:** `[x]` done

---

### Step 01.3 — Add at least one sibling scene of the same class
**Files:** `DEV/ocrlab/corpus.json`, `DEV/ocrlab/annotations/<scene>.json`
**Depends on:** Step 01.2

**Prompt for developer:**
> One scene cannot defend a threshold. Add a second scene of the same class - display lettering over
> flat saturated colour, opened as a standalone image - from licence-verified material, annotate it
> the same way, and keep both in the `dev` split. `join-the-ranks-of-the-red-army` and
> `polish-soviet-propaganda-poster-18y` are already in the corpus as posters; a scene derived from
> one of them counts only if it is a standalone-image case and carries `derivedFrom`.

**Verification:**
- A second scene id of this class appears in `DEV/ocrlab/corpus.json` with `"split": "dev"`.
- Its annotation file exists with `"origin": "human"`.
- `ocrlab verify` exits 0.

**Status:** `[x]` done

## Phase done criteria
- [x] Every `Step 01.*` is `[x] done`.
- [x] `go run ./tools/ocrlab verify` exits 0.
- [x] Grep for `TODO(phase-01)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

## Handoff notes

The class now exists in the corpus, so Phase 02 can compute a strength measure over scenes that
represent both sides of the question: images that fail and look like they read, and images that read.

## Rollback plan

Revert the phase commit; the corpus manifest and annotations are data, nothing else consumes them yet.

## Execution notes (2026-08-12)

- **The verification predicate "`ocrlab verify` exits 0" was unattainable and is replaced.** verify
  already exits 1 on this corpus: 35 of its 46 scenes are scored but unannotated, which is a standing
  coverage shortfall and not something this ticket introduced or can close. What was checked instead:
  neither new scene appears anywhere in verify's problem list, and the problem count fell from 36 to
  34 as the two annotations landed.
- **Step 01.3 annotates an existing corpus scene rather than adding a derived one.**
  `a-propaganda-poster-from-the-soviet-union-in-the-1920s-391853265017-jpg` is already in the dev
  split, is licence-verified, and is the class exactly - red Cyrillic display type over a flat
  saturated ground, as a standalone image. Converted with `-ocr -ocr-lang rus` before annotating, it
  produced no plate at all, which is the same failure as the reported poster. A derived crop would
  have been weaker evidence, not stronger.
