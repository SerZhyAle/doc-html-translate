// reflow.js - paragraph/heading reflow heuristics, ported from the desktop app's
// internal/pdf/extract.go (rowsToText + classifyBlock + isLigaturesArtifact).
//
// The Go app has two text sources: pdftotext "-layout" (string with leading
// spaces) and the pure-Go reader's GetTextByRow (positioned words). PDF.js
// getTextContent() yields positioned items (transform + width), so we port the
// *positioned* path (rowsToText): paragraphs by Y-gap and first-word X-indent,
// headings by centering / ALL-CAPS / shortness. Because we have real coordinates
// and font sizes, centering and headings are computed from geometry rather than
// pdftotext's leading-space proxy, and a conservative font-size boost is added.
//
// Everything here is pure (plain objects in, plain objects out) so it can be unit
// tested under `node --test` without a browser. See test/reflow.test.mjs.

// Points; typical first-line paragraph indent is 18-36pt, so 8pt cleanly separates
// an indented paragraph opener from a flush continuation line. Mirrors Go's indentThreshold.
const INDENT_THRESHOLD = 8;
// A vertical gap beyond 1.5x the median line spacing marks a paragraph/section break.
const PARA_GAP_FACTOR = 1.5;
// Relative font-height change between adjacent rows that forces a block break
// (isolates a heading set in a different size from the body it precedes).
const FONT_BREAK_RATIO = 0.25;

// classifyBlock assigns an HTML tag from text content and layout, mirroring
// internal/pdf/extract.go:classifyBlock. `centered` and `fontRatio` (this block's
// font height / the page body mode height) replace pdftotext's leadingSpaces proxy.
// `notWide` (the block does not reach the right text margin) gates the font-size
// triggers so a long, full-width body line in a slightly larger size can't be
// promoted to a heading - the Go classifier has no size trigger at all, so this
// keeps the JS port from over-tagging.
export function classifyBlock(text, { centered = false, fontRatio = 1, notWide = false } = {}) {
  const words = text.trim().split(/\s+/).filter(Boolean);
  if (words.length === 0) return "p";

  const isAllCaps = text.toUpperCase() === text && /[A-ZА-ЯЁ]/u.test(text);
  const isShort = words.length <= 8;
  const big = fontRatio >= 1.25 && notWide;
  const veryBig = fontRatio >= 1.6 && notWide;

  if ((isAllCaps || centered || veryBig) && isShort) return "h2";
  if ((centered || big) && words.length <= 14) return "h3";
  return "p";
}

// isLigaturesArtifact flags ligature-garbage rows (e.g. "if lf if if if if")
// produced when the font map can't be decoded - avg word length < 3 over >= 4
// words. Mirrors internal/pdf/extract.go:isLigaturesArtifact.
export function isLigaturesArtifact(s) {
  const words = s.trim().split(/\s+/).filter(Boolean);
  if (words.length < 4) return false;
  const total = words.reduce((n, w) => n + w.length, 0);
  return total / words.length < 3.0;
}

// extractRows groups PDF.js text items into visual rows (top-to-bottom,
// left-to-right) and reconstructs word spacing from x-gaps. Each row carries the
// geometry the paragraph/heading heuristics need.
//
// items: array of PDF.js TextItem-likes: { str, transform:[a,b,c,d,e,f], width, height, hasEOL }.
// transform[4]/[5] are the x/y of the text origin in PDF user space (y up).
export function extractRows(items) {
  const recs = [];
  for (const it of items || []) {
    if (!it || typeof it.str !== "string" || !Array.isArray(it.transform)) continue;
    const t = it.transform;
    const h = Math.hypot(t[1], t[3]) || it.height || 0; // glyph run height (font size)
    recs.push({
      str: it.str,
      x: t[4],
      y: t[5],
      h,
      w: typeof it.width === "number" ? it.width : 0,
    });
  }
  if (recs.length === 0) return [];

  // Reading order: top of the page first (PDF y increases upward), then left to right.
  recs.sort((a, b) => b.y - a.y || a.x - b.x);

  // Bucket into rows by baseline proximity (half the local font height).
  const rows = [];
  let cur = null;
  for (const r of recs) {
    const tol = Math.max(2, (r.h || 10) * 0.5);
    if (cur && Math.abs(cur.y - r.y) <= tol) {
      cur.items.push(r);
    } else {
      cur = { y: r.y, items: [r] };
      rows.push(cur);
    }
  }

  const out = [];
  for (const row of rows) {
    row.items.sort((a, b) => a.x - b.x);
    let text = "";
    let prevEndX = null;
    for (const r of row.items) {
      if (r.str === "") continue;
      if (text === "") {
        text = r.str;
      } else {
        const gap = prevEndX != null ? r.x - prevEndX : 0;
        const needSpace =
          !/\s$/.test(text) && !/^\s/.test(r.str) && gap > (r.h ? r.h * 0.15 : 1);
        text += (needSpace ? " " : "") + r.str;
      }
      prevEndX = r.x + r.w;
    }
    text = text.replace(/\s+/g, " ").trim();
    if (text === "") continue;

    const firstX = row.items[0].x;
    const last = row.items[row.items.length - 1];
    const rightX = last.x + last.w;
    const h = row.items.reduce((m, r) => Math.max(m, r.h || 0), 0);
    out.push({ text, y: row.y, firstX, rightX, h });
  }
  return out;
}

function percentile(sorted, p) {
  if (sorted.length === 0) return 0;
  const idx = Math.min(sorted.length - 1, Math.floor(sorted.length * p));
  return sorted[idx];
}

// reflowPage turns one page's getTextContent() + viewport into tagged blocks
// ([{ tag:"p"|"h2"|"h3", text }]). This is the JS analogue of one pass of
// rowsToText + classifyBlock over a page.
export function reflowPage(textContent, viewport) {
  const rawRows = extractRows(textContent && textContent.items);
  const rows = rawRows.filter((r) => !isLigaturesArtifact(r.text));
  if (rows.length === 0) return [];

  const pageWidth = (viewport && (viewport.width || viewport.viewBox?.[2])) || 612;

  // Median line spacing (Go: median Y-gap).
  const gaps = [];
  for (let i = 1; i < rows.length; i++) {
    const g = Math.abs(rows[i].y - rows[i - 1].y);
    if (g > 0) gaps.push(g);
  }
  gaps.sort((a, b) => a - b);
  const medianGap = gaps.length ? gaps[Math.floor(gaps.length / 2)] : 12;

  // Body left/right margins. Go uses the 25th percentile of first-word X for the
  // left margin so a page that is ~40% indented openers still finds the true
  // margin; we add a 75th-percentile right edge to detect symmetric (centered) insets.
  const lefts = rows.map((r) => r.firstX).sort((a, b) => a - b);
  const rights = rows.map((r) => r.rightX).sort((a, b) => a - b);
  const leftMargin = percentile(lefts, 0.25);
  const rightMargin = percentile(rights, 0.75);
  const colCenter = (leftMargin + rightMargin) / 2;

  // Body font height for the heading size-boost. Use the *mode* (most common
  // rounded height), not the median: on title/caption/size-mixed pages the median
  // can sit below the true body size and inflate every line's ratio.
  const freq = new Map();
  let bodyHeight = 0;
  let bodyHeightCount = -1;
  for (const r of rows) {
    if (!(r.h > 0)) continue;
    const key = Math.round(r.h);
    const c = (freq.get(key) || 0) + 1;
    freq.set(key, c);
    if (c > bodyHeightCount) { bodyHeightCount = c; bodyHeight = key; }
  }

  // A line is centered when it is inset from BOTH margins by a comparable amount
  // and its midpoint sits near the column centre. Requiring a real right inset (and
  // a left inset well beyond a paragraph first-line indent) stops an indented body
  // opener - which is also a single-line block - from being mistaken for a heading.
  const isCentered = (row) => {
    const leftGap = row.firstX - leftMargin;
    const rightInset = rightMargin - row.rightX;
    const blockCenter = (row.firstX + row.rightX) / 2;
    return (
      leftGap > pageWidth * 0.08 &&
      leftGap > INDENT_THRESHOLD * 2 &&
      rightInset > pageWidth * 0.04 &&
      Math.abs(leftGap - rightInset) < pageWidth * 0.12 &&
      Math.abs(blockCenter - colCenter) < pageWidth * 0.06
    );
  };

  // Group rows into paragraphs (Go: rowsToText). Break on a big vertical gap, a
  // first-line indent, a font-size change, or a change in centering. Centering is
  // precomputed so consecutive centered rows (a wrapped heading) stay in one block
  // and the indent test is suppressed for them (their large X is centering, not a
  // paragraph indent).
  const rowCentered = rows.map(isCentered);
  const paragraphs = [];
  let curRows = [];
  for (let i = 0; i < rows.length; i++) {
    const r = rows[i];
    if (i > 0) {
      const prev = rows[i - 1];
      const yBreak = Math.abs(r.y - prev.y) > medianGap * PARA_GAP_FACTOR;
      const xBreak = !rowCentered[i] && r.firstX > leftMargin + INDENT_THRESHOLD;
      const maxH = Math.max(r.h, prev.h) || 1;
      const fontBreak = Math.abs(r.h - prev.h) / maxH > FONT_BREAK_RATIO;
      const centeredBreak = rowCentered[i] !== rowCentered[i - 1];
      if (yBreak || xBreak || fontBreak || centeredBreak) {
        if (curRows.length) paragraphs.push(curRows);
        curRows = [];
      }
    }
    curRows.push(r);
  }
  if (curRows.length) paragraphs.push(curRows);

  const blocks = [];
  for (const para of paragraphs) {
    const text = para.map((r) => r.text).join(" ").replace(/\s+/g, " ").trim();
    if (text === "") continue;
    const wordCount = text.split(/\s+/).filter(Boolean).length;
    const maxH = para.reduce((m, r) => Math.max(m, r.h || 0), 0);
    const fontRatio = bodyHeight > 0 ? maxH / bodyHeight : 1;
    // Headings may wrap onto a couple of lines (the Go pdftotext path keeps such a
    // heading as one block), so allow centering for short multi-line blocks whose
    // every row is individually centered - not just single-line blocks.
    const shortBlock = para.length <= 3 && wordCount <= 14;
    const centered = shortBlock && para.every(isCentered);
    // notWide: the block's farthest right edge stops short of the right margin.
    const blockRight = para.reduce((m, r) => Math.max(m, r.rightX), 0);
    const notWide = rightMargin - blockRight > pageWidth * 0.04;
    blocks.push({ tag: classifyBlock(text, { centered, fontRatio, notWide }), text });
  }
  return blocks;
}
