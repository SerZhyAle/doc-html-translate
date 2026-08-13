// ocr-cluster.js - the hand-port of the desktop app's internal/ocr/tesseract.go clusterLines:
// turn the recognizer's text lines into the block-level plates the overlay renders. It lives in
// its own module, like ocr-text.js, so it can be unit-tested without loading the Tesseract worker
// bundle - the grouping arithmetic is the part that drifts, so it is the part that gets tests.
//
// Keep this file and tesseract.go in sync (docs/PARITY.md).

import { isTranslatable } from "./ocr-text.js";

// Overlay grouping constants, shared verbatim with the desktop app (see docs/PARITY.md and
// internal/ocr/tesseract.go ocrMinLineConf / ocrClusterPitchFactor / ocrMaxLeadingRatio).
//
// OCR_MIN_LINE_CONF drops a line whose mean word confidence is below it: real text scores ~80-97,
// while the "text" Tesseract hallucinates out of a drawing scores ~0-50, so the gate removes noise
// that would otherwise become an opaque plate covering the figure (and whose oversized boxes
// inflate the font).
//
// OCR_CLUSTER_PITCH_FACTOR then grows one plate while the next line keeps the running line pitch -
// the distance from one line's top to the next line's top - within this many reference pitches; a
// bigger step (a figure, a section break, a new column) starts a new plate.
//
// The factor multiplies the pitch and not the height of the recognized ink box, because the ink box
// is not the line. All-caps lettering with no descenders boxes far shorter than the line it came
// from, and the difference is not marginal: measured on the lab scene synth-balloon-on-panel
// (2026-08-11), a balloon drawn with 36 px leading recognizes as 14-17 px ink boxes with 19-22 px
// between them, so a gap read against 1.2 x 14 px broke one balloon into three plates - three
// sentence fragments handed to a translator that has no sentence to work with. The same balloon's
// pitch is a flat 36 px, well inside 1.2 x 36. Pitch is also what keeps the opposite failure from
// being traded in: on synth-adjacent-balloons the pitch inside a balloon is 26 px and the step
// across to the next balloon is 42 px, so the two stay two plates.
//
// OCR_MAX_LEADING_RATIO bounds what may count as a line pitch when the reference is estimated: a
// step of more than this many median ink heights is a section break, not leading. Typography sets
// the scale - all-caps ink runs about 0.7 em and loose leading about 2 em, so real leading tops out
// near 2.9 ink heights - and the bound is what stops one big jump on a two-line page (a heading and
// a body paragraph far below it) from becoming the page's "typical" pitch and merging them.
export const OCR_MIN_LINE_CONF = 50;
export const OCR_CLUSTER_PITCH_FACTOR = 1.2;
export const OCR_MAX_LEADING_RATIO = 3;

export const medianOf = (a) => (a.length ? a.slice().sort((p, q) => p - q)[a.length >> 1] : 0);

// medianLinePitch estimates the page's line pitch from the tops of successive kept lines, or
// returns 0 when the page offers none.
//
// The unit is the image, not the cluster: a cluster's own pitch is unavailable exactly when the
// decision is hardest - joining its second line, where there is no pitch yet - and that is the very
// join that split the balloon. Taking the median over the page instead makes the estimate available
// from the first decision, and the median is what keeps a page carrying two type sizes honest: the
// steps between the sizes are outliers around the body text's pitch, not the middle of it.
//
// A pair contributes only when it could plausibly be one text's leading: same column (the test the
// clustering itself uses), moving forward, and no further than OCR_MAX_LEADING_RATIO ink heights.
// Both bounds are there to keep a step that is really a section break out of the estimate - without
// them a page holding one heading and one distant paragraph would make that single jump the median
// and merge the two. Mirrors tesseract.go medianLinePitch (docs/PARITY.md).
export function medianLinePitch(kept, medianH) {
  const maxPitch = medianH * OCR_MAX_LEADING_RATIO;
  const pitches = [];
  for (let i = 1; i < kept.length; i++) {
    const prev = kept[i - 1].bbox, cur = kept[i].bbox;
    const dy = cur.y0 - prev.y0;
    if (dy <= 0 || dy > maxPitch) continue;
    const overlap = Math.min(cur.x1, prev.x1) - Math.max(cur.x0, prev.x0);
    const narrower = Math.min(cur.x1 - cur.x0, prev.x1 - prev.x0);
    if (overlap * 10 < narrower) continue;
    pitches.push(dy);
  }
  return pitches.length ? medianOf(pitches) : 0;
}

// Drop low-confidence noise lines, then group the survivors (in reading order) into one plate
// per run of vertically-adjacent, horizontally-overlapping lines. A plate's box is the union of
// its line boxes and its font tracks the median line height, so a plate covers a coherent text
// column without spanning imagery or the gaps between columns/sections. Mirrors the desktop
// app's tesseract.go clusterLines - keep the two in sync (docs/PARITY.md).
//
// Vertical adjacency is judged on the line pitch (see OCR_CLUSTER_PITCH_FACTOR). A page whose lines
// yield no measurable pitch at all - one line, or nothing but lines that share no column - has no
// reference to compare against, and falls back to the ink-box gap this test used before.
export function clusterLines(lines, minConf = OCR_MIN_LINE_CONF) {
  const kept = lines.filter((l) => l.text && l.conf >= minConf);
  if (!kept.length) return [];
  const medianH = medianOf(kept.map((l) => l.bbox.y1 - l.bbox.y0)) || 1;
  const refPitch = medianLinePitch(kept, medianH);
  const pitchMax = refPitch * OCR_CLUSTER_PITCH_FACTOR;
  const gapMax = medianH * OCR_CLUSTER_PITCH_FACTOR;

  const blocks = [];
  let cur = null;
  const flush = () => {
    if (!cur) return;
    const text = cur.texts.join(" ").trim();
    if (isTranslatable(text)) {
      blocks.push({
        text,
        bbox: { x0: cur.x0, y0: cur.y0, x1: cur.x1, y1: cur.y1 },
        lineHeight: medianOf(cur.heights) || (cur.y1 - cur.y0),
      });
    }
    cur = null;
  };
  for (const l of kept) {
    const { x0, y0, x1, y1 } = l.bbox;
    if (cur) {
      const gap = y0 - cur.y1;
      const pitch = y0 - cur.lastY0;
      const overlap = Math.min(x1, cur.x1) - Math.max(x0, cur.x0);
      const narrower = Math.min(x1 - x0, cur.x1 - cur.x0);
      const adjacent = refPitch ? pitch <= pitchMax : gap <= gapMax;
      // same column (share x-extent) and vertically adjacent (the line keeps the page's pitch; a
      // small negative gap tolerates overlapping boxes, a big one means a new column/section).
      if (adjacent && gap >= -medianH && overlap * 10 >= narrower) {
        cur.x0 = Math.min(cur.x0, x0); cur.y0 = Math.min(cur.y0, y0);
        cur.x1 = Math.max(cur.x1, x1); cur.y1 = Math.max(cur.y1, y1);
        cur.lastY0 = y0;
        cur.texts.push(l.text); cur.heights.push(y1 - y0);
        continue;
      }
      flush();
    }
    cur = { x0, y0, x1, y1, lastY0: y0, texts: [l.text], heights: [y1 - y0] };
  }
  flush();
  return blocks;
}
