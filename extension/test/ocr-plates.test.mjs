// Unit tests for ocr-plates.js: the arithmetic that turns a recognized block into plate geometry
// (plateSpecs) and the runtime re-fit (fitPlate). Both are the extension's copy of rules the
// desktop app holds in overlay.go percentStyle and ocrScript; tests/parity_test.go pins the numbers
// across editions, these pin what the numbers do.
import { test } from "node:test";
import assert from "node:assert/strict";
import "./_dom.mjs";
import {
  FONT_FIT, FONT_GROW_CAP, fitPlate, pictureBox, plateSpecs, renderPlates, transformRotates,
} from "../src/ocr-plates.js";

test("plateSpecs places a block in percent of the image and sizes its font in cqw", () => {
  const specs = plateSpecs({
    width: 1000, height: 500,
    blocks: [
      { text: "Hello", bbox: { x0: 100, y0: 50, x1: 600, y1: 150 }, lineHeight: 40, colors: { bg: "rgb(1,2,3)", ink: "rgb(4,5,6)" } },
      { text: "", bbox: { x0: 0, y0: 0, x1: 10, y1: 10 }, lineHeight: 10 }, // nothing to show
      null,
    ],
  });
  assert.equal(specs.length, 1, "a block without text gets no plate");
  const s = specs[0];
  assert.equal(s.left, "10%");
  assert.equal(s.top, "10%");
  assert.equal(s.width, "50%");
  // min-height rather than height, so the plate may grow past its region instead of clipping.
  assert.equal(s.minHeight, "20%");
  assert.equal(s.fontSize, `${(4 * FONT_FIT).toFixed(2)}cqw`, "font = line height / width x FONT_FIT, in cqw");
  assert.equal(s.bg, "rgb(1,2,3)");
  assert.equal(s.ink, "rgb(4,5,6)");
});

test("plateSpecs without colours leaves the CSS default, and without a size draws nothing", () => {
  const block = { text: "x", bbox: { x0: 0, y0: 0, x1: 10, y1: 10 }, lineHeight: 10 };
  const [s] = plateSpecs({ width: 100, height: 100, blocks: [block] });
  assert.equal(s.bg, "");
  assert.equal(s.ink, "");
  assert.deepEqual(plateSpecs({ width: 0, height: 100, blocks: [block] }), []);
  assert.deepEqual(plateSpecs({ width: 100, height: 0, blocks: [block] }), []);
});

test("renderPlates puts paper and ink on the plate box", () => {
  const container = document.createElement("div");
  renderPlates(container, plateSpecs({
    width: 100, height: 100,
    blocks: [{ text: "Hi", bbox: { x0: 0, y0: 0, x1: 50, y1: 20 }, lineHeight: 10, colors: { bg: "red", ink: "blue" } }],
  }));
  const plate = container.querySelector(".ocr-plate");
  assert.equal(plate.textContent, "Hi");
  assert.equal(plate.style.background, "red");
  assert.equal(plate.style.color, "blue");
});

// fakePlate is a plate whose text needs `perCqw` pixels of height per cqw of font. Its box is the
// inline min-height unless fitPlate released it to height:auto, in which case it is as tall as its
// text - which is what "released and grows" means in a browser.
function fakePlate({ base, minHeight, perCqw }) {
  const style = { fontSize: base > 0 ? `${base}cqw` : "", height: "" };
  const plate = {
    dataset: {},
    style,
    minHeight,
    get scrollHeight() {
      const m = /([0-9.]+)cqw/.exec(style.fontSize);
      return m ? parseFloat(m[1]) * perCqw : 0;
    },
    get clientHeight() {
      return style.height === "auto" ? this.scrollHeight : parseFloat(style.height) || 0;
    },
  };
  return plate;
}

function withComputedStyle(fn) {
  const prev = globalThis.getComputedStyle;
  globalThis.getComputedStyle = (el) => ({ minHeight: `${el.minHeight}px` });
  try { fn(); } finally { globalThis.getComputedStyle = prev; }
}
const cqw = (plate) => parseFloat(/([0-9.]+)cqw/.exec(plate.style.fontSize)[1]);

test("fitPlate grows short text up to the cap and no further", () => {
  withComputedStyle(() => {
    // Text at the base size fills a tenth of the box: growth is limited by the cap, not the box.
    const p = fakePlate({ base: 4, minHeight: 400, perCqw: 10 });
    fitPlate(p);
    assert.ok(Math.abs(cqw(p) - 4 * FONT_GROW_CAP) < 1e-9, `font ${cqw(p)}, want the cap ${4 * FONT_GROW_CAP}`);
    assert.equal(p.style.height, "400px", "a plate that fits keeps its source region height");
  });
});

test("fitPlate grows only while the text still fits, and stops one step before it overflows", () => {
  withComputedStyle(() => {
    // A 41 px box with the 1 px slack holds 4.2 cqw: the first grow step (4 -> 4.3, 43 px)
    // overflows and is undone.
    const p = fakePlate({ base: 4, minHeight: 41, perCqw: 10 });
    fitPlate(p);
    assert.equal(cqw(p), 4);
    assert.ok(p.scrollHeight <= p.clientHeight + 1);
  });
});

test("fitPlate shrinks long text until it fits, above the floor", () => {
  withComputedStyle(() => {
    const p = fakePlate({ base: 10, minHeight: 70, perCqw: 10 });
    fitPlate(p);
    assert.ok(cqw(p) <= 7.1, `font ${cqw(p)} still overflows a box that holds 7`);
    assert.ok(cqw(p) >= 10 * 0.5, "the shrink never passes the floor");
    assert.equal(p.style.height, "70px", "text that fits after shrinking keeps the box");
  });
});

test("fitPlate releases the box rather than clip text that does not fit at the floor", () => {
  withComputedStyle(() => {
    // Even at half the base the text needs 50 px of a 20 px box.
    const p = fakePlate({ base: 10, minHeight: 20, perCqw: 10 });
    fitPlate(p);
    assert.equal(p.style.height, "auto", "OCR-OVERLAY rule 9: released, never clipped");
    assert.ok(cqw(p) <= 5 && cqw(p) > 4, `font ${cqw(p)}, want it stopped at the floor`);
    assert.ok(p.scrollHeight <= p.clientHeight + 1, "nothing is hidden");
  });
});

test("fitPlate re-fits from the compile-time size, not from its own last answer", () => {
  withComputedStyle(() => {
    const p = fakePlate({ base: 10, minHeight: 70, perCqw: 10 });
    fitPlate(p);
    const first = cqw(p);
    // The translator swapped in shorter text: the next fit starts over from the stored base.
    p.minHeight = 400;
    fitPlate(p);
    assert.equal(p.dataset.ocrCqw, "10");
    assert.ok(cqw(p) > first, "a shorter translation gets its size back");
    assert.ok(Math.abs(cqw(p) - 10 * FONT_GROW_CAP) < 1e-9);
  });
});

test("fitPlate leaves a plate with no cqw size alone apart from its box", () => {
  withComputedStyle(() => {
    const p = fakePlate({ base: 0, minHeight: 30, perCqw: 10 });
    fitPlate(p);
    assert.equal(p.dataset.ocrCqw, "0");
    assert.equal(p.style.fontSize, "");
    assert.equal(p.style.height, "30px");
  });
});

// ---- pictureBox: where a live page's picture is actually drawn --------------------------------
const noBox = {
  borderLeftWidth: "0px", borderRightWidth: "0px", borderTopWidth: "0px", borderBottomWidth: "0px",
  paddingLeft: "0px", paddingRight: "0px", paddingTop: "0px", paddingBottom: "0px",
  objectFit: "fill", objectPosition: "50% 50%",
};
const at = (left, top, width, height) => ({ left, top, width, height });
const place = (over) => pictureBox({
  rect: at(10, 20, 400, 200), naturalWidth: 400, naturalHeight: 200,
  offsetWidth: 400, offsetHeight: 200, rotated: false, ...over, style: { ...noBox, ...(over.style || {}) },
});

test("pictureBox takes border and padding off the element's box", () => {
  const got = place({
    rect: at(0, 0, 430, 230), offsetWidth: 430, offsetHeight: 230,
    style: { borderLeftWidth: "5px", borderRightWidth: "5px", borderTopWidth: "5px", borderBottomWidth: "5px",
      paddingLeft: "10px", paddingRight: "10px", paddingTop: "10px", paddingBottom: "10px" },
  });
  assert.deepEqual(got.box, at(15, 15, 400, 200));
  assert.deepEqual(got.clip, got.box);
});

test("pictureBox letterboxes under contain and keeps the whole picture", () => {
  const got = place({ rect: at(0, 0, 300, 300), offsetWidth: 300, offsetHeight: 300, style: { objectFit: "contain" } });
  assert.deepEqual(got.box, at(0, 75, 300, 150));
  assert.deepEqual(got.clip, got.box);
});

test("pictureBox crops under cover, honouring object-position", () => {
  const got = place({
    rect: at(0, 0, 150, 150), offsetWidth: 150, offsetHeight: 150,
    style: { objectFit: "cover", objectPosition: "0% 50%" },
  });
  assert.deepEqual(got.box, at(0, 0, 300, 150), "the picture is scaled to fill and anchored left");
  assert.deepEqual(got.clip, at(0, 0, 150, 150), "only the element's box of it is shown");
});

test("pictureBox handles none and scale-down", () => {
  assert.deepEqual(place({ rect: at(0, 0, 200, 200), offsetWidth: 200, offsetHeight: 200, style: { objectFit: "none" } }).box,
    at(-100, 0, 400, 200));
  assert.deepEqual(place({ rect: at(0, 0, 800, 800), offsetWidth: 800, offsetHeight: 800, style: { objectFit: "scale-down" } }).box,
    at(200, 300, 400, 200), "scale-down never enlarges");
});

test("pictureBox follows a scaled ancestor", () => {
  // Laid out at 400x200, drawn at half size by a transformed container.
  const got = place({ rect: at(0, 0, 200, 100), style: { paddingLeft: "20px", paddingRight: "20px" }, offsetWidth: 440 });
  const k = 200 / 440;
  for (const [key, want] of Object.entries(at(20 * k, 0, 400 * k, 100))) {
    assert.ok(Math.abs(got.box[key] - want) < 1e-9, `${key}: ${got.box[key]} want ${want}`);
  }
});

test("pictureBox refuses what it cannot place", () => {
  assert.equal(place({ rotated: true }), null, "a rotated picture is cleared, not drawn skewed");
  assert.equal(place({ style: { objectFit: "cover", objectPosition: "calc(10% + 5px) 50%" } }), null);
  assert.equal(place({ style: { objectFit: "cover", objectPosition: "right 10px bottom 5px" } }), null);
  assert.equal(place({ rect: at(0, 0, 0, 0) }), null);
});

test("transformRotates tells turning, skewing and mirroring from moving and scaling", () => {
  for (const [t, r, want] of [
    ["none", "none", false],
    ["matrix(1, 0, 0, 1, 30, 40)", "none", false],
    ["matrix(0.5, 0, 0, 0.5, 0, 0)", "none", false],
    ["matrix(0.984808, 0.173648, -0.173648, 0.984808, 0, 0)", "none", true], // rotate(10deg)
    ["matrix(1, 0, 0.176327, 1, 0, 0)", "none", true], // skewX(10deg)
    ["matrix(-1, 0, 0, 1, 0, 0)", "none", true], // mirrored
    ["none", "10deg", true],
    ["none", "0deg", false],
    ["matrix3d(1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 5, 6, 0, 1)", "none", false],
    ["matrix3d(1, 0, 0, 0, 0, 0.5, 0.866, 0, 0, -0.866, 0.5, 0, 0, 0, 0, 1)", "none", true], // rotateX
    ["perspective(10px)", "none", true],
  ]) {
    assert.equal(transformRotates(t, r), want, `${t} / ${r}`);
  }
  assert.equal(transformRotates("none", "none", "0.5"), false);
  assert.equal(transformRotates("none", "none", "2 0.5"), false);
  assert.equal(transformRotates("none", "none", "-1 1"), true, "the scale property mirrors too");
});
