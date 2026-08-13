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
//
// OCR_TYPE_SIZE_RATIO is the second half of that question, and it is the half pitch cannot answer. A
// page with separated regions - a poster, a form, a comic cover - hands the page-wide pitch estimate
// steps that belong to no single text, and a headline can then sit closer to the body than the body's
// own missing lines do. Measured on poster-display-type-on-flat-colour: the headline stands 168 px
// from the next recognized line while two body lines that a reader takes as one text stand 190 px
// apart, so no pitch bound separates them in the right place. Their type sizes do - the headline's
// ink is 281 px against the body's 155 px median (1.81x) - which is the distinction a reader uses.
//
// The value has to clear the spread a single text shows on its own and stay under the step between
// two texts. Both are measured on the corpus's hand-drawn line boxes, never on a recognizer's output:
// the widest within-group spread is samson-and-delilah-03-scroll, 19 lines of one caption whose ink
// runs 23-34 px, worst line 1.42x its own group's median; the narrowest across-group step is this
// poster's 1.86x (and synth-display-lettering's headline-to-footnote is 2.57x). 1.6 is the geometric
// middle of 1.42 and 1.81 - about 13% of margin on each side - rather than a round number chosen next
// to one of them.
//
// OCR_MAX_PLATE_COVERAGE and OCR_MIN_PLATE_LINE_FILL are the last pair, and together they answer a
// question none of the others can. Pitch and type size compare a line with its neighbours, so a page
// whose separated regions carry one type at one pitch - a form, a list, an application window -
// offers nothing in its typography to cut on, and the whole page arrives as a single plate. What
// separates that plate from a real one is that it is at once too big and too loose: it covers more
// of the picture than a text block plausibly can, and its own line boxes account for too little of
// the height it spans, because most of that height is the gaps between regions.
//
// Both conditions are required, and each is there because the other is not enough on its own. Size
// alone would release samson-and-delilah-03-scroll, a crop whose one hand-annotated caption covers
// 0.6087 of it. Looseness alone would release synth-uniform-paper, three lines of ordinary body text
// whose 0.6667 line fill is just the leading a paragraph has. Each bound is bracketed from opposite
// directions over the 46 lab scenes and the 13 hand-drawn annotations
// (DEV/research/ocr_plate_coverage_2026-08-13.md): coverage between the largest legitimate plate the
// recognizer produces (0.4004) and the defect (accounts.jpg at 0.6829); line fill between the defect
// (0.6608) and that scroll caption's hand-drawn lines (0.7921). Both are geometric middles.
//
// The area version of the fill - how much of a plate's box its lines cover - was measured on the
// same run and separates nothing (0.5891 for the defect against 0.4582 for a legitimate balloon), so
// the rule is stated on the vertical axis, which is the axis separated regions are separated on.
export const OCR_MIN_LINE_CONF = 50;
export const OCR_CLUSTER_PITCH_FACTOR = 1.2;
export const OCR_MAX_LEADING_RATIO = 3;
export const OCR_TYPE_SIZE_RATIO = 1.6;
export const OCR_MAX_PLATE_COVERAGE = 0.52;
export const OCR_MIN_PLATE_LINE_FILL = 0.72;

export const medianOf = (a) => (a.length ? a.slice().sort((p, q) => p - q)[a.length >> 1] : 0);

// sameTypeSize reports whether a line's ink height is close enough to a cluster's own to be part of
// the same text (see OCR_TYPE_SIZE_RATIO). The comparison is against the cluster's median rather than
// its previous line, so one odd box - a line of caps with no descenders, a line that is a single
// short word - cannot end a plate by itself.
//
// A missing height is not evidence of a break: a zero on either side keeps the line, because the
// clustering must never become stricter through an unmeasured quantity. Mirrors tesseract.go
// sameTypeSize (docs/PARITY.md).
export function sameTypeSize(h, clusterH) {
  if (h <= 0 || clusterH <= 0) return true;
  return Math.max(h, clusterH) <= Math.min(h, clusterH) * OCR_TYPE_SIZE_RATIO;
}

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

// releaseOversized turns a cluster that is both too big and too loose (see OCR_MAX_PLATE_COVERAGE
// and OCR_MIN_PLATE_LINE_FILL) into one plate per line, and returns null when the cluster passes
// either test - the ordinary case, and the case on every scene of the lab corpus.
//
// Releasing rather than refusing is the whole point: every recognized word still reaches a plate, so
// no scene loses text, while the area the overlay paints drops from the rectangle spanning the
// regions to the lines themselves. The released plates skip isTranslatable deliberately - the
// cluster's assembled text has already passed it, and re-testing each line separately would drop the
// short ones and turn a composition fix into a recall regression. A single-line cluster is never
// released: its box is its line. Mirrors tesseract.go releaseOversized (docs/PARITY.md).
export function releaseOversized(cur, imgW, imgH) {
  if (!(imgW > 0 && imgH > 0) || cur.lines.length < 2 || cur.texts.length !== cur.lines.length) return null;
  const boxH = cur.y1 - cur.y0;
  const box = (cur.x1 - cur.x0) * boxH;
  if (box <= 0 || boxH <= 0 || box <= imgW * imgH * OCR_MAX_PLATE_COVERAGE) return null;
  let ink = 0;
  for (const l of cur.lines) ink += l.y1 - l.y0;
  if (ink >= boxH * OCR_MIN_PLATE_LINE_FILL) return null;
  const out = [];
  for (let i = 0; i < cur.lines.length; i++) {
    const text = (cur.texts[i] || "").trim();
    if (!text) continue;
    const l = cur.lines[i];
    out.push({
      text,
      bbox: { x0: l.x0, y0: l.y0, x1: l.x1, y1: l.y1 },
      lineHeight: l.y1 - l.y0,
      lines: [{ x0: l.x0, y0: l.y0, x1: l.x1, y1: l.y1 }],
    });
  }
  return out.length ? out : null;
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
//
// imgW/imgH are the page's own dimensions, used only by the coverage rule (see
// OCR_MAX_PLATE_COVERAGE): a finished cluster that covers more of the picture than a text block
// plausibly can, and packs its own lines into too little of the height it spans, is released into
// those lines. Zero on either means the page size is unknown and the rule does not run.
export function clusterLines(lines, minConf = OCR_MIN_LINE_CONF, imgW = 0, imgH = 0) {
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
      const released = releaseOversized(cur, imgW, imgH);
      if (released) blocks.push(...released);
      else {
        blocks.push({
          text,
          bbox: { x0: cur.x0, y0: cur.y0, x1: cur.x1, y1: cur.y1 },
          lineHeight: medianOf(cur.heights) || (cur.y1 - cur.y0),
          // The block's own line boxes, in reading order. Mirrors Block.Lines in the desktop app
          // (docs/PARITY.md); the coverage rule reads them, and so does the lab's geometry.
          lines: cur.lines.slice(),
        });
      }
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
      const sameSize = sameTypeSize(y1 - y0, medianOf(cur.heights));
      // same column (share x-extent), same type size, and vertically adjacent (the line keeps the
      // page's pitch; a small negative gap tolerates overlapping boxes, a big one means a new
      // column/section).
      if (adjacent && sameSize && gap >= -medianH && overlap * 10 >= narrower) {
        cur.x0 = Math.min(cur.x0, x0); cur.y0 = Math.min(cur.y0, y0);
        cur.x1 = Math.max(cur.x1, x1); cur.y1 = Math.max(cur.y1, y1);
        cur.lastY0 = y0;
        cur.texts.push(l.text); cur.heights.push(y1 - y0);
        cur.lines.push({ x0, y0, x1, y1 });
        continue;
      }
      flush();
    }
    cur = { x0, y0, x1, y1, lastY0: y0, texts: [l.text], heights: [y1 - y0], lines: [{ x0, y0, x1, y1 }] };
  }
  flush();
  return blocks;
}

// resultStrength measures how much a recognition attempt actually found: the number of words it
// placed on the image. The count is over the plates a pass produced, so the confidence floor that
// pass ran under is already applied - a word here is a word the reader would see. Word count rather
// than plated area, because area is a statement about image size as much as about what was read.
// Mirrors tesseract.go resultStrength (docs/PARITY.md).
export function resultStrength(blocks) {
  let n = 0;
  for (const b of blocks || []) n += (b.text || "").split(/\s+/).filter(Boolean).length;
  return n;
}

// strictlyBetter is the only rule by which a retry may replace a result already in hand. It is
// deliberately not symmetric: a retry is a second look at the same pixels, so two attempts that
// find the same amount are one piece of evidence and not two, and a tie keeps the incumbent.
// Mirrors tesseract.go strictlyBetter (docs/PARITY.md).
export function strictlyBetter(candidate, current) {
  return resultStrength(candidate) > resultStrength(current);
}
