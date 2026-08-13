// Tests for the halftone-screen detector (ocr-screen.js). Mirrors internal/ocr/screen_test.go -
// the two implementations are hand-ported, so they are given the same input and must reach the
// same pitch (see docs/PARITY.md).

import { test } from "node:test";
import assert from "node:assert/strict";
import {
  screenPitch,
  tileStep,
  mergeScreenBlocks,
  coveredFraction,
  OCR_SCREEN_TILE,
  OCR_SCREEN_MAX_TILES,
  OCR_SCREEN_MIN_PITCH,
  OCR_SCREEN_SIGMA_DIVISOR,
  OCR_SCREEN_MERGE_MAX_OVERLAP,
} from "../src/ocr-screen.js";

// halftoneField draws a dot lattice of the given pitch over a light ground - the thing a press
// lays down to print a tone, and the thing the screen rung has to recognize. Same construction as
// the Go test's halftoneField, so the two detectors are judged on the same picture.
function halftoneField(w, h, pitch) {
  const grey = new Uint8Array(w * h).fill(240);
  const r = pitch / 3;
  for (let cy = 0; cy < h; cy += pitch) {
    for (let cx = 0; cx < w; cx += pitch) {
      for (let y = cy - pitch; y <= cy + pitch; y++) {
        for (let x = cx - pitch; x <= cx + pitch; x++) {
          if (x < 0 || y < 0 || x >= w || y >= h) continue;
          if (Math.hypot(x - cx, y - cy) <= r) grey[y * w + x] = 90;
        }
      }
    }
  }
  return grey;
}

// The sigma the rung filters with is derived from this number, so reading the period wrong is the
// same as choosing the wrong kernel. Several pitches, because a screen's period depends on the
// press and on the scan resolution - a detector that only works at one pitch is a fix for one pitch.
test("screenPitch finds the lattice at every pitch measured", () => {
  for (const pitch of [4, 6, 8, 12]) {
    assert.equal(screenPitch(halftoneField(320, 320, pitch), 320, 320), pitch,
      `pitch-${pitch} screen`);
  }
});

// The rung costs a recognition pass, and on a picture with no lattice that pass is spent for
// nothing. Flat paper has no residual to measure at all; noise has residual everywhere but no
// period the tiles agree on.
test("screenPitch ignores what is not a screen", () => {
  assert.equal(screenPitch(new Uint8Array(320 * 320).fill(236), 320, 320), 0, "flat paper");

  // A fixed sequence rather than Math.random, so a failure is reproducible.
  const noise = new Uint8Array(320 * 320);
  let seed = 12345;
  for (let i = 0; i < noise.length; i++) {
    seed = (seed * 1103515245 + 12345) & 0x7fffffff;
    noise[i] = seed & 0xff;
  }
  assert.equal(screenPitch(noise, 320, 320), 0, "white noise");
});

// A two-pixel "period" is JPEG blocking or the scanner's own grain, and blurring for it would
// soften an image that has no screen at all.
test("screenPitch rejects periods outside the bounds", () => {
  assert.equal(screenPitch(halftoneField(320, 320, 2), 320, 320), 0);
  assert.ok(OCR_SCREEN_MIN_PITCH >= 3, "a period below 3 px is noise, not a press screen");
});

// The additive sweep's trigger, mirroring the Go test of the same shape. Not "does this picture
// carry a screen" but "is there screened area the reader has no plate over" - the third case is the
// one the feature exists for, a screened caption standing beside a balloon the ordinary pass read,
// and a trigger that fails it does nothing at all and does it silently.
test("screenPitch asks the narrower question when it is given plates", () => {
  const pitch = 8;
  // A screen over the right half of the picture, flat paper over the left.
  const grey = halftoneField(320, 320, pitch);
  for (let y = 0; y < 320; y++) {
    for (let x = 0; x < 160; x++) grey[y * 320 + x] = 240;
  }

  assert.equal(screenPitch(grey, 320, 320), pitch, "no plates");
  assert.equal(screenPitch(grey, 320, 320, [{ x0: 128, y0: 0, x1: 320, y1: 320 }]), 0,
    "a plate over the screened half leaves nothing to sweep for");
  assert.equal(screenPitch(grey, 320, 320, [{ x0: 0, y0: 0, x1: 128, y1: 320 }]), pitch,
    "a plate over the plain half leaves the screened caption beside it unserved");
});

// The additive sweep's other half, mirroring the Go test of the same shape. The sweep is only safe
// if the page it adds to comes through untouched, and only useful if what it adds is not a second
// plate over lettering that already has one - which is visible damage, worse for the reader than the
// caption staying untranslated.
//
// The straddling case is the one the design turns on: a rule that took each existing plate on its
// own would see two halves under the bound and let a fully covered candidate through.
test("mergeScreenBlocks drops what is already plated and keeps every plate it was given", () => {
  const block = (text, x0, y0, x1, y1) => ({ text, bbox: { x0, y0, x1, y1 }, lineHeight: 10 });
  const left = block("already read", 200, 0, 335, 100);
  const right = block("also already read", 465, 0, 600, 100);
  const kept = [left, right];

  const clear = block("a screened caption", 0, 200, 100, 260);
  const duplicate = block("already read", 210, 10, 330, 90);
  // Each existing plate takes 35 columns of this one's 200, under the bound on its own; together
  // they take 70, over it.
  const straddling = block("already read twice", 300, 0, 500, 100);

  const got = mergeScreenBlocks(kept, [clear, duplicate, straddling]);
  assert.equal(got[0], left, "the ordinary pass's plates come through unchanged and in order");
  assert.equal(got[1], right);
  assert.deepEqual(got.slice(2), [clear], "only the plate clear of every existing one is added");

  for (const k of kept) {
    assert.ok(coveredFraction(straddling.bbox, [k.bbox]) <= OCR_SCREEN_MERGE_MAX_OVERLAP,
      `${k.text} alone must not put the straddling candidate over the bound`);
  }
  assert.ok(coveredFraction(straddling.bbox, kept.map((k) => k.bbox)) > OCR_SCREEN_MERGE_MAX_OVERLAP,
    "the two together must");
});

// The work has to be bounded, or the rung's cost would grow with the page instead of staying the
// price of one measurement: a magazine page scanned at 600 DPI is 60 times the area of a panel.
test("tileStep caps the sampled tiles whatever the image size", () => {
  assert.equal(tileStep(OCR_SCREEN_TILE * 4, OCR_SCREEN_TILE * 4), OCR_SCREEN_TILE,
    "a small image is walked tile by tile");
  const big = OCR_SCREEN_TILE * 100;
  const step = tileStep(big, big);
  const sampled = Math.floor(big / step) ** 2;
  assert.ok(sampled <= OCR_SCREEN_MAX_TILES, `sampled ${sampled} tiles, cap is ${OCR_SCREEN_MAX_TILES}`);
  assert.ok(step >= OCR_SCREEN_TILE, "tiles must not overlap");
});

// The kernel the rung blurs with is pitch / this. The extension applies it as CSS blur(Npx), whose
// length is the Gaussian's standard deviation - the same kernel the desktop app builds - so the
// divisor is the whole of the shared parameter and must not drift from screen.go.
test("the sigma divisor is the shared invariant", () => {
  assert.equal(OCR_SCREEN_SIGMA_DIVISOR, 4);
});
