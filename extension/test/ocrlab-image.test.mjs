// The lab's screenshot-to-image-space step. What matters is that a page captured in bands, at
// whatever resolution the viewport happened to use, comes back as one picture in exactly the
// source's pixel space - because every pixel metric indexes the rendered image with the source's
// own coordinates, and a band placed a few pixels off would be measured as concealment that moved.

import test from "node:test";
import assert from "node:assert/strict";
import { createCanvas, loadImage } from "@napi-rs/canvas";
import { assembleToNatural, bandsFor, CaptureBandPx } from "../scripts/_ocrlab-image.mjs";

function solid(width, height, colour) {
  const c = createCanvas(width, height);
  const x = c.getContext("2d");
  x.fillStyle = colour;
  x.fillRect(0, 0, width, height);
  return c.toBuffer("image/png");
}

// A capture stand-in: the left half black, the right half white, so a resample can be checked for
// position as well as for size.
function halfBlack(width, height) {
  const c = createCanvas(width, height);
  const x = c.getContext("2d");
  x.fillStyle = "#fff";
  x.fillRect(0, 0, width, height);
  x.fillStyle = "#000";
  x.fillRect(0, 0, Math.round(width / 2), height);
  return c.toBuffer("image/png");
}

const readAt = async (png, w, h) => {
  const c = createCanvas(w, h);
  const ctx = c.getContext("2d");
  ctx.drawImage(await loadImage(png), 0, 0);
  return (x, y) => ctx.getImageData(x, y, 1, 1).data[0];
};

test("a small region is one band", () => {
  assert.deepEqual(bandsFor(800, 600), [{ y0: 0, y1: 600 }]);
});

test("bands cover the region exactly, with no gap and no overlap", () => {
  const h = 4000;
  const bands = bandsFor(1280, h);
  assert.ok(bands.length > 1, "a 5.1 Mpx region should not be asked for in one reply");
  assert.equal(bands[0].y0, 0);
  assert.equal(bands[bands.length - 1].y1, h);
  for (let i = 1; i < bands.length; i++) assert.equal(bands[i].y0, bands[i - 1].y1);
});

test("no band exceeds the measured reply ceiling", () => {
  for (const [w, h] of [[1280, 4000], [1213, 943], [2000, 9000], [400, 20000]]) {
    for (const b of bandsFor(w, h)) {
      assert.ok(Math.round(w) * (b.y1 - b.y0) <= CaptureBandPx,
        `band ${b.y0}-${b.y1} of ${w}x${h} is over the ceiling`);
    }
  }
});

test("a capture is returned in the source image's own pixel space", async () => {
  const png = await assembleToNatural(
    [{ png: halfBlack(320, 200), y0: 0, y1: 200 }], 320, 200, 1800, 1400);
  const img = await loadImage(png);
  assert.equal(img.width, 1800);
  assert.equal(img.height, 1400);
});

test("the resample keeps what is where", async () => {
  const png = await assembleToNatural(
    [{ png: halfBlack(320, 200), y0: 0, y1: 200 }], 320, 200, 800, 600);
  const at = await readAt(png, 800, 600);
  assert.ok(at(100, 300) < 40, "the left half should still be dark");
  assert.ok(at(700, 300) > 215, "the right half should still be light");
});

test("bands are stitched where they belong, not where they arrived", async () => {
  // Three bands of a 300-tall page, deliberately handed over out of order and at a capture
  // resolution of their own: black, white, black.
  const parts = [
    { png: solid(100, 100, "#000"), y0: 200, y1: 300 },
    { png: solid(100, 100, "#000"), y0: 0, y1: 100 },
    { png: solid(100, 100, "#fff"), y0: 100, y1: 200 },
  ];
  const png = await assembleToNatural(parts, 100, 300, 100, 300);
  const at = await readAt(png, 100, 300);
  assert.ok(at(50, 50) < 20, "the top band is black");
  assert.ok(at(50, 150) > 235, "the middle band is white");
  assert.ok(at(50, 250) < 20, "the bottom band is black");
});

test("an image space with no size is refused rather than silently mismeasured", async () => {
  await assert.rejects(
    () => assembleToNatural([{ png: halfBlack(320, 200), y0: 0, y1: 200 }], 320, 200, 0, 1400),
    /image space/);
});

test("a band that caught nothing is refused", async () => {
  await assert.rejects(
    () => assembleToNatural([{ png: solid(1, 1, "#000"), y0: 0, y1: 1 }], 1, 1, 800, 600),
    /too small/);
});
