// Unit tests for the pure transform helpers in pdf-images.js. Run: npm test.
// The canvas/blob paths need a browser; only the CTM math is covered here.

import { test } from "node:test";
import assert from "node:assert/strict";

// pdf-images.js imports pdf.mjs (for the OPS enum), and from pdfjs 6 that module
// constructs a DOMMatrix at top level - a browser global Node does not have, so the
// import throws before any test runs. The extension itself is unaffected: every
// context that loads pdf.mjs (the viewer page, the pdfjs worker) has DOMMatrix. Stub
// the one constructor pdf.mjs touches at load time, then import dynamically so the
// stub is in place first (static imports are hoisted).
globalThis.DOMMatrix ??= class DOMMatrix {
  constructor() {
    this.a = 1; this.b = 0; this.c = 0; this.d = 1; this.e = 0; this.f = 0;
  }
};

const { composeTransform, paintFlips } = await import("../src/pdf-images.js");

const IDENTITY = [1, 0, 0, 1, 0, 0];

test("composeTransform: identity is neutral", () => {
  const m = [2, 0, 0, 3, 5, 7];
  assert.deepEqual(composeTransform(IDENTITY, m), m);
  assert.deepEqual(composeTransform(m, IDENTITY), m);
});

test("composeTransform: scales multiply through nesting", () => {
  const outer = [0.24, 0, 0, 0.24, 0, 0];
  const inner = [100, 0, 0, -200, 10, 20];
  const m = composeTransform(outer, inner);
  assert.equal(m[0], 24);
  assert.equal(m[3], -48);
});

test("paintFlips: standard placement matrix needs no flip", () => {
  assert.deepEqual(paintFlips([461.96, 0, 0, 725.94, 0, 66]), { flipX: false, flipY: false });
});

test("paintFlips: negative y-scale needs a vertical flip (High plains twister cover)", () => {
  // Actual page-1 image CTM from the bug report: raster stored bottom-up,
  // un-mirrored at draw time via the negative y-scale.
  assert.deepEqual(paintFlips([1385.88, 0, 0, -2177.81, 553.699, 2859.64]), { flipX: false, flipY: true });
});

test("paintFlips: negative both axes = 180 degree placement", () => {
  assert.deepEqual(paintFlips([-100, 0, 0, -200, 0, 0]), { flipX: true, flipY: true });
});

test("paintFlips: rotated placement is left as stored", () => {
  assert.deepEqual(paintFlips([0, 100, -200, 0, 0, 0]), { flipX: false, flipY: false });
});
