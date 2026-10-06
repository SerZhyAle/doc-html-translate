// Tests for the concealment-mode decision (ocr-conceal.js). Mirrors internal/ocr/conceal_test.go:
// the constructed pictures below are the Go test's, so the two editions are judged on the same
// pixels; tests/parity_test.go TestParityOCRConcealment pins the constants and expressions.
import { test } from "node:test";
import assert from "node:assert/strict";
import {
  conceal, decideMode, measureRing, plateBackground, MODE_FILL, MODE_MASK, MODE_RECONSTRUCT,
} from "../src/ocr-conceal.js";

// The colour-sampling constants ocr-overlay.js hands over as RING.
const RING = { deviation: 90, padDivisor: 3, minPad: 2, minSamples: 40 };

// raster is an RGBA picture with a sampler shaped like ocr-overlay.js pixelsIn: the same sub-sampling
// step over the rectangle, alpha under 128 skipped.
function raster(w, h, bg) {
  const data = new Uint8ClampedArray(w * h * 4);
  const set = (x, y, c) => { const o = (y * w + x) * 4; data[o] = c[0]; data[o + 1] = c[1]; data[o + 2] = c[2]; data[o + 3] = 255; };
  for (let y = 0; y < h; y++) for (let x = 0; x < w; x++) set(x, y, bg);
  const sample = (x0, y0, sw, sh) => {
    const n = sw * sh, step = Math.max(1, Math.floor(n / 6000));
    const rs = [], gs = [], bs = [];
    for (let i = 0; i < n; i += step) {
      const x = x0 + (i % sw), y = y0 + Math.floor(i / sw);
      const o = (y * w + x) * 4;
      if (data[o + 3] < 128) continue;
      rs.push(data[o]); gs.push(data[o + 1]); bs.push(data[o + 2]);
    }
    return { rs, gs, bs };
  };
  return { w, h, set, sample };
}

const block = (x0, y0, x1, y1, lineHeight, lines) => ({ bbox: { x0, y0, x1, y1 }, lineHeight, lines: lines || [{ x0, y0, x1, y1 }] });

test("flat paper keeps the fill", () => {
  const img = raster(300, 120, [250, 249, 246]);
  const r = measureRing(img.sample, block(40, 40, 260, 70, 30).bbox, 30, img.w, img.h, RING);
  assert.deepEqual(decideMode(r), { mode: MODE_FILL, conf: 1 });
});

test("a ramp across the block is reconstructed along it", () => {
  const img = raster(300, 200, [0, 0, 0]);
  for (let y = 0; y < 200; y++) {
    const t = y / 200;
    for (let x = 0; x < 300; x++) img.set(x, y, [Math.floor(70 + 160 * t), Math.floor(130 + 70 * t), Math.floor(200 - 50 * t)]);
  }
  const b = block(40, 60, 260, 140, 30);
  const r = measureRing(img.sample, b.bbox, 30, img.w, img.h, RING);
  assert.equal(decideMode(r).mode, MODE_RECONSTRUCT);
  const bg = plateBackground(MODE_RECONSTRUCT, r, b, "rgb(1,2,3)", img.w);
  assert.match(bg, /^linear-gradient\(to bottom,rgb\(\d+,\d+,\d+\),rgb\(\d+,\d+,\d+\)\)$/);
});

test("halftone round the block is a mask", () => {
  const img = raster(300, 150, [245, 240, 225]);
  for (let y = 0; y < 150; y++) for (let x = 0; x < 300; x++) if (x % 6 < 3 && y % 6 < 3) img.set(x, y, [120, 110, 100]);
  const r = measureRing(img.sample, block(40, 50, 260, 80, 24).bbox, 24, img.w, img.h, RING);
  assert.equal(decideMode(r).mode, MODE_MASK);
});

// Same picture as conceal_test.go TestAnEdgeBesideTheBlockIsNotAGradient.
test("an edge beside the block is not a gradient", () => {
  const img = raster(400, 120, [250, 250, 245]);
  for (let y = 0; y < 120; y++) for (let x = 330; x < 345; x++) img.set(x, y, [10, 10, 10]);
  const r = measureRing(img.sample, block(40, 40, 328, 70, 30).bbox, 30, img.w, img.h, RING);
  assert.equal(decideMode(r).mode, MODE_FILL);
});

// Same picture as conceal_test.go TestAThinRingIsAMaskAtZero.
test("a block with no ring to judge is a mask at 0", () => {
  const img = raster(60, 20, [255, 255, 255]);
  const r = measureRing(img.sample, block(0, 0, 60, 20, 20).bbox, 20, img.w, img.h, RING);
  assert.deepEqual(decideMode(r), { mode: MODE_MASK, conf: 0 });
});

// Same block as conceal_test.go TestMaskPaintsLessThanTheBlock.
test("a mask paints one padded stripe per line, in cqw from the plate's corner", () => {
  const b = block(100, 100, 700, 190, 30, [{ x0: 100, y0: 100, x1: 700, y1: 130 }, { x0: 100, y0: 160, x1: 400, y1: 190 }]);
  const bg = plateBackground(MODE_MASK, { minPad: 2 }, b, "rgb(9,9,9)", 1000);
  assert.equal(bg,
    "linear-gradient(rgb(9,9,9),rgb(9,9,9)) -0.500cqw -0.500cqw/61.000cqw 4.000cqw no-repeat," +
    "linear-gradient(rgb(9,9,9),rgb(9,9,9)) -0.500cqw 5.500cqw/31.000cqw 4.000cqw no-repeat");
  assert.equal(plateBackground(MODE_MASK, { minPad: 2 }, b, "", 1000), "", "without a sampled paper the plate keeps the CSS default");
  assert.equal(plateBackground(MODE_FILL, { minPad: 2 }, b, "rgb(9,9,9)", 1000), "", "the fill is the caller's paper");
});

test("conceal returns the mode, its confidence and the background together", () => {
  const img = raster(300, 150, [245, 240, 225]);
  for (let y = 0; y < 150; y++) for (let x = 0; x < 300; x++) if (x % 6 < 3 && y % 6 < 3) img.set(x, y, [120, 110, 100]);
  const c = conceal(img.sample, block(40, 50, 260, 80, 24), img.w, img.h, RING, "rgb(245,240,225)");
  assert.equal(c.mode, MODE_MASK);
  assert.ok(c.conf > 0 && c.conf <= 1);
  assert.match(c.background, /cqw no-repeat$/);
});

test("mask padding survives the plate boundary and stays inside the image", () => {
  const img = raster(300, 150, [245, 240, 225]);
  const b = block(0, 0, 300, 150, 24, [{ x0: 0, y0: 0, x1: 300, y1: 150 }]);
  const edge = conceal(img.sample, b, img.w, img.h, RING, "rgb(245,240,225)");
  assert.equal(edge.mode, MODE_MASK);
  assert.deepEqual(edge.bounds, b.bbox, "image edges clamp the mask padding");
  for (let y = 0; y < 150; y++) for (let x = 0; x < 300; x++) if (x % 6 < 3 && y % 6 < 3) img.set(x, y, [120, 110, 100]);
  const c = conceal(img.sample, block(40, 50, 260, 80, 24, [{ x0: 40, y0: 50, x1: 260, y1: 80 }]), img.w, img.h, RING, "rgb(245,240,225)");
  assert.deepEqual(c.bounds, { x0: 36, y0: 46, x1: 264, y1: 84 });
  assert.match(c.background, /0\.000cqw 0\.000cqw/, "source padding now starts inside the background box");
});
