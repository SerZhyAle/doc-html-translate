# internal/appearance

`appearance.json` is the one description of the appearance the two editions share: the OCR
overlay unit (container, image, plate) and the reader theme palette (4 themes x 8 colour tokens).
It is the only place these values may be edited. Both editions derive their CSS from it; neither
edition's copy is written by hand.

## Consumers

- **Go - `internal/appearance`** (`appearance.go`). Embeds the file and builds the CSS at run time:
  `OverlayCSS` for `internal/ocr/overlay.go`, `PaletteCSS` for `internal/htmlgen/navbar.go`. The
  converted page still carries its styles inline, so offline output stays self-contained.
- **JS - `extension/scripts/gen-appearance.mjs`**. Reads the file at generation time, never at run
  time, and rewrites the marked regions of `extension/src/ocr-overlay.css` and
  `extension/src/viewer.css`. `npm run appearance` regenerates; `--check` fails when a region is
  stale, and the extension build runs that check before packaging.

## Making a change

Adding, removing or changing a declaration here is how a change reaches both editions: edit the
file, run `npm run appearance` in `extension/`, commit both. `tests/appearance_parity_test.go`
compares every role and theme on both sides against this file, declaration by declaration, and
fails on anything one side has and the other lacks.

## What is not in here

Names. Selectors, the OCR toggle class, the custom-property prefix and the theme attribute are
per-edition and supplied by each generator; the gate is blind to them. A difference between the
editions is legal only when `divergences` lists it with a reason - an entry with `role`,
`property` and `edition` lets that one declaration differ, and the gate fails on an entry that
nothing uses.

The plate's measured values (padding, radius, paper carrier) were each bracketed by a lab run
(`DEV/plan/16_2026-08-13_ocr-sweep-plate-composition.md`, `docs/PARITY.md` "OCR"); their `note`
fields say why, and they ship as CSS comments.
