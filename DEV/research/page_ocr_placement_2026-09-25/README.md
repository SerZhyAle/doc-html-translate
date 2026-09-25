# Plates on a live page sit over the picture as drawn; with JS off nothing clips, the box grows

2026-09-25. Feeds [`DEV/plan/21_2026-09-23_contract-ocr-pipeline-sync.md`](../../plan/21_2026-09-23_contract-ocr-pipeline-sync.md)
(Direction A items 5 and 6). Chromium 1194 (Playwright 1.56), headless, device scale 1.

## 1. Page-OCR placement (OCR-OVERLAY rule 4)

`placement.mjs` serves `extension/` over HTTP, loads `page.html` with a stand-in `chrome.*`, injects
the real `src/page-agent.js`, and hands it one plate per picture through its own message listener -
the path the broker uses. Every picture is a 400x200 bitmap, left half red, right half blue; the plate
is green and covers the left half (`left:0; width:50%; min-height:100%`). Counted in a full-page
screenshot, per picture, before and after:

- `redAfter` - red still showing: text the plate should have covered;
- `greenOutside` - green where there was no red before: a plate outside its text;
- `blueLost` - blue now hidden: a plate over the wrong part of the picture.

| picture | before (HEAD `8e15132`) | after |
|---|---|---|
| plain | clean | clean |
| `padding` + `border` | green outside 13592, blue lost 1200 | clean |
| `object-fit: contain` (letterbox) | green outside 22500 | clean |
| `object-fit: cover`, `object-position: 0% 50%` | red left 11250 (half the text uncovered) | clean |
| `object-fit: cover`, `object-position: -40px 30%` | red left 1800 | clean |
| ancestor `transform: scale(0.5)` + padding | green outside 3200 | clean |
| `transform: rotate(10deg)` | green outside 17980, blue lost 848 | no layer (red 39794 left, nothing drawn) |
| ancestor `rotate: 5deg` | green outside 9314, blue lost 391 | no layer |
| `scale: -1 1` (the individual property) | not in the HEAD run | no layer |
| `transform: scaleX(-1)` | plate on the other half: blue lost 40000, red 40000 left | no layer |

"Clean" is `redAfter 0, greenOutside 0, blueLost 0`. The rotated and mirrored rows are the rule's
"clears the overlay instead of drawing it in the wrong place": the text stays uncovered and no plate
lands anywhere else.

Run: `PW_ROOT=$(npm root -g) node placement.mjs <repo>/extension <this dir>/page.html`.

## 2. JS disabled: the "clipped" claim of `OCR-PIPELINE` §3.4

The catalog says the CSS `overflow:hidden` "is the degraded fallback" when the fit script does not
run. Checked against what the desktop app writes: `jsoff_page.go.txt` (a throwaway `main`, rename to
`.go` under `temp/` to run) builds a page through `ocr.OverlayFile` with the stand-in engine
`jsoff_tesseract.sh` - one 200x20 px line box holding a 17-word sentence - and `jsoff.mjs` opens it
with JS off and on:

| | font | box height | scrollHeight = clientHeight | clipped |
|---|---|---|---|---|
| JS off | `4.6cqw` (compile-time) | 244 px (source region 39 px) | yes | no |
| JS on | `2.08cqw` (shrunk to the floor, released) | 56 px, `height:auto` | yes | no |

Nothing clips in either case. The plate sets only `min-height`, so `overflow:hidden` has no height to
cut against: with JS off the box simply grows at the unfitted size, six times its source region, over
whatever lies below. The prose is wrong in the reassuring direction - rule 9 (nothing clipped) holds -
but the degraded page is a plate spilling over the art, not a clipped one. The catalog correction
belongs to ticket 21 step B1 (local).
