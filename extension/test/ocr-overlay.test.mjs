// Unit tests for collectLines - ocr-overlay.js's read of the recognizer's answer, the counterpart
// of tesseract.go tsvLines with parseTSV's unread half. The module imports the vendored Tesseract
// bundle at load, which `npm run vendor` fetches and which may be absent, and nothing here
// recognizes anything - so a module hook swaps that import for an inert stub, as the viewer tests
// do, and the parse runs on hand-built recognizer output.

import { test } from "node:test";
import assert from "node:assert/strict";
import { register } from "node:module";
import { clusterLines, OCR_MIN_LINE_CONF } from "../src/ocr-cluster.js";
import { coveredFraction, OCR_SCREEN_MERGE_MAX_OVERLAP } from "../src/ocr-screen.js";

const HOOKS = `
export async function resolve(specifier, context, next) {
  if (/(^|\\/)vendor\\//.test(specifier)) {
    return { url: "data:text/javascript," + encodeURIComponent("export default {}"), shortCircuit: true };
  }
  return next(specifier, context);
}
`;
register(`data:text/javascript,${encodeURIComponent(HOOKS)}`, import.meta.url);

const { collectLines, ocrLangToHtmlLang } = await import("../src/ocr-overlay.js?under-test=collectLines");

test("Japanese OCR choices declare Japanese page language", () => {
  assert.equal(ocrLangToHtmlLang("jpn"), "ja");
  assert.equal(ocrLangToHtmlLang("jpn_vert"), "ja");
});

// The word a recognizer returns: text, its own box, a confidence. Word heights of 30 px
// throughout, so the split's ratio rule cuts a gap above 3.5 x 30 = 105 px.
const word = (text, x0, x1, y0) => ({ text, confidence: 95, bbox: { x0, y0, x1, y1: y0 + 30 } });

const line = (words) => ({
  text: words.map((w) => w.text).join(" "),
  confidence: 95,
  bbox: {
    x0: words[0].bbox.x0, y0: words[0].bbox.y0,
    x1: words[words.length - 1].bbox.x1, y1: words[words.length - 1].bbox.y1,
  },
  words,
});

// The caption the layout analysis isolated and recognition read nothing from - the grey sweep's
// evidence, held as tesseract.go holds Result.unread.
const CAPTION = { x0: 120, y0: 400, x1: 420, y1: 460 };
const captionLine = () => ({ text: "", confidence: 0, bbox: { ...CAPTION }, words: [] });

// A page whose two columns the layout analysis stitched into one line box, twice - the words'
// gaps are ten word heights, far over the split's ratio - and a tinted caption under them.
// Input order is the engine's: each cut emits left then right, so the runs interleave and
// orderColumns regroups the page into columns.
const splitPage = () => ({
  blocks: [{
    paragraphs: [{
      lines: [
        line([word("SAYS", 100, 200, 100), word("MOUSE", 500, 640, 100)]),
        line([word("AND", 100, 190, 160), word("FOX", 520, 640, 160)]),
        captionLine(),
      ],
    }],
  }],
});

test("a page with a split line keeps the grey sweep's evidence (ticket 82)", () => {
  const lines = collectLines(splitPage());
  // The regroup ran: each column's lines now arrive together. If this order ever changes to the
  // input's, the test is reading the no-split path and proving nothing.
  assert.deepEqual(lines.map((l) => l.text), ["SAYS", "AND", "MOUSE", "FOX", ""]);
  // The caption's box rides on the array that came back - greySweep reads its trigger there.
  assert.deepEqual(lines.unread, [CAPTION]);
  // And the sweep fires on it: a marked region stands where the plates clusterLines accepts
  // leave it uncovered (ocr-overlay.js greySweep, tesseract.go unreadOutside).
  const covered = clusterLines(lines, OCR_MIN_LINE_CONF, 800, 600, []).map((b) => b.bbox);
  assert.equal(covered.length, 2);
  assert.ok(
    lines.unread.some((r) => coveredFraction(r, covered) <= OCR_SCREEN_MERGE_MAX_OVERLAP),
    "a marked region the plates leave uncovered - greySweep must fire",
  );
});

test("a page without a split carries the evidence on the array it returns", () => {
  const page = {
    blocks: [{
      paragraphs: [{
        lines: [
          line([word("SAYS", 100, 200, 100), word("MOUSE", 230, 370, 100)]),
          captionLine(),
        ],
      }],
    }],
  };
  const lines = collectLines(page);
  // The gap is one word height, no cut, no regroup - the engine's own order comes back, with the
  // caption still on it. The regression ticket 82 fixes was specific to the split path.
  assert.deepEqual(lines.map((l) => l.text), ["SAYS MOUSE", ""]);
  assert.deepEqual(lines.unread, [CAPTION]);
});
