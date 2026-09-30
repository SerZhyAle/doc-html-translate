// Unit tests for the pure transform helpers in pdf-images.js. Run: npm test.
// The canvas/blob paths are covered against a stubbed canvas at the bottom.

import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

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

const { composeTransform, paintFlips, sameShapeRaster, dedupeSameShape, needsPageComposite, rasterScale, RASTER_MAX_PIXELS, RASTER_MAX_SIDE } = await import("../src/pdf-images.js");
const mrcFixture = JSON.parse(readFileSync(new URL("../../tests/testdata/pdf_mrc_pair.json", import.meta.url)));

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

test("sameShapeRaster: same scan at two resolutions matches (Plague proclamation)", () => {
  // The measured duplicate: one page embedded as 1455x2065 and 4363x6193 (exactly 3x).
  assert.equal(sameShapeRaster({ width: 1455, height: 2065 }, { width: 4363, height: 6193 }), true);
});

test("sameShapeRaster: different shapes do not match", () => {
  assert.equal(sameShapeRaster({ width: 800, height: 600 }, { width: 600, height: 800 }), false);
});

test("sameShapeRaster: zero dimensions never match", () => {
  assert.equal(sameShapeRaster({ width: 0, height: 0 }, { width: 100, height: 100 }), false);
});

test("dedupeSameShape: keeps the largest of a proportional-scale group", () => {
  const small = { blob: "a", width: 1455, height: 2065 };
  const big = { blob: "b", width: 4363, height: 6193 };
  const kept = dedupeSameShape([small, big]);
  assert.equal(kept.length, 1);
  assert.equal(kept[0].blob, "b"); // the larger raster wins regardless of order
  assert.deepEqual(dedupeSameShape([big, small]).map((i) => i.blob), ["b"]);
});

test("dedupeSameShape: a composed page of differently-shaped images keeps all", () => {
  const portrait = { blob: "p", width: 600, height: 900 };
  const landscape = { blob: "l", width: 900, height: 600 };
  const square = { blob: "s", width: 500, height: 500 };
  assert.equal(dedupeSameShape([portrait, landscape, square]).length, 3);
});

test("the shared MRC pair is rendered as a page composite", async () => {
  assert.equal(needsPageComposite([mrcFixture.background, mrcFixture.foreground]), true);
  assert.equal(needsPageComposite([{ width: 24, height: 24 }, { width: 96, height: 96 }]), false);
  const page = stubPage({
    background: { ...mrcFixture.background, data: new Uint8Array(mrcFixture.background.width * mrcFixture.background.height * 3) },
    foreground: { ...mrcFixture.foreground, data: new Uint8Array(mrcFixture.foreground.width * mrcFixture.foreground.height * 4) },
  });
  let rendered = false;
  page.getViewport = ({ scale }) => ({ width: mrcFixture.background.width * scale, height: mrcFixture.background.height * scale });
  page.render = () => { rendered = true; return { promise: Promise.resolve() }; };
  const images = await extractPageImages(page);
  assert.equal(rendered, true);
  assert.deepEqual(images.map(({ width, height }) => [width, height]), [[mrcFixture.foreground.width, mrcFixture.foreground.height]]);
});

test("the page raster keeps ordinary pages at full scale and caps outsized ones", () => {
  // A4 in PDF points at scale 2 is well under the cap.
  assert.equal(rasterScale(595, 842, 2), 2);
  // A0 poster: the pixel cap decides.
  const a0 = rasterScale(2384, 3370, 2);
  assert.ok(a0 < 2 && Math.round(2384 * a0) * Math.round(3370 * a0) <= RASTER_MAX_PIXELS * 1.001);
  // A long strip: the side cap decides.
  const strip = rasterScale(500, 40000, 2);
  assert.ok(40000 * strip <= RASTER_MAX_SIDE + 1e-6);
  assert.equal(rasterScale(0, 0, 2), 2);
});

// ---- extractPageImages over a stub page -------------------------------------------
// The canvas is stubbed down to what the draw path touches, so the test can see which
// image objects become blobs. VideoFrame stands in for the object pdf.js hands over for a
// JPEG it decoded through ImageDecoder.
class StubVideoFrame {
  constructor(w, h) { this.displayWidth = w; this.displayHeight = h; }
}
class StubCanvas {
  constructor(w, h) { this.width = w; this.height = h; this.drawn = []; }
  getContext() {
    const c = this;
    return { translate() {}, scale() {}, drawImage(src) { c.drawn.push(src); }, putImageData() {} };
  }
  convertToBlob() { return Promise.resolve({ size: this.width * this.height, drawn: this.drawn }); }
}
globalThis.VideoFrame ??= StubVideoFrame;
globalThis.OffscreenCanvas ??= StubCanvas;
globalThis.ImageData ??= class ImageData { constructor(d, w, h) { this.data = d; this.width = w; this.height = h; } };

const { OPS } = await import("../vendor/pdf.mjs");
const { extractPageImages, ocrWorthy, OCR_MIN_SIDE } = await import("../src/pdf-images.js");

function stubPage(objs) {
  const names = Object.keys(objs);
  return {
    getOperatorList: async () => ({
      fnArray: names.map(() => OPS.paintImageXObject),
      argsArray: names.map((n) => [n, objs[n].width, objs[n].height]),
    }),
    objs: { get: (name, cb) => cb(objs[name]) },
    commonObjs: { get: (_name, cb) => cb(null) },
  };
}

test("extractPageImages: a JPEG pdf.js decoded to a VideoFrame becomes an image (ticket 77)", async () => {
  // The measured shape: data null, bitmap a VideoFrame. Only an ImageBitmap used to count, so
  // every such picture was dropped without a word.
  const frame = new globalThis.VideoFrame(500, 166);
  const imgs = await extractPageImages(stubPage({ img_p1_1: { data: null, width: 500, height: 166, bitmap: frame } }));
  assert.equal(imgs.length, 1);
  assert.equal(imgs[0].width, 500);
  assert.equal(imgs[0].blob.drawn[0], frame);
});

test("extractPageImages: icon-sized rasters are kept, as the desktop keeps them", async () => {
  const imgs = await extractPageImages(stubPage({
    photo: { data: new Uint8Array(250 * 333 * 3), width: 250, height: 333 },
    icon: { data: new Uint8Array(20 * 27 * 3), width: 20, height: 27 },
  }));
  assert.deepEqual(imgs.map((i) => `${i.width}x${i.height}`), ["250x333", "20x27"]);
});

test("ocrWorthy: only a picture of real size is queued for recognition", () => {
  assert.equal(ocrWorthy({ width: 250, height: 333 }), true);
  assert.equal(ocrWorthy({ width: 20, height: 27 }), false);
  assert.equal(ocrWorthy({ width: 767, height: OCR_MIN_SIDE - 1 }), false);
  assert.equal(ocrWorthy({ width: OCR_MIN_SIDE, height: OCR_MIN_SIDE }), true);
});
