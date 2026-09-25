// Resource-lifecycle guards for the OCR overlay fit release (the page raster cap is pinned in
// pdf-images.test.mjs). See DEV/plan/done/18_2026-09-24_bugfix-extension-lifecycle-leaks.md.

import "./_dom.mjs";

import { test } from "node:test";
import assert from "node:assert/strict";

import { scheduleFit, releaseOverlays, buildOverlay } from "../src/ocr-plates.js";

// fakeWindowListeners stands a window in whose listeners can be counted, keyed by type + handler.
function fakeWindowListeners() {
  const added = new Map();
  globalThis.window = {
    addEventListener: (t, fn) => added.set(`${t}`, fn),
    removeEventListener: (t, fn) => { if (added.get(`${t}`) === fn) added.delete(`${t}`); },
  };
  return added;
}

test("an overlay's fit listeners are released with its document", async () => {
  const added = fakeWindowListeners();
  try {
    const host = document.createElement("div");
    document.body.append(host);
    const overlay = buildOverlay({ imageSrc: "blob:x", blocks: [], width: 10, height: 10 });
    host.append(overlay);
    assert.equal(added.size, 2, "load + resize while the overlay is live");
    releaseOverlays(host);
    assert.equal(added.size, 0);
    host.remove();
  } finally {
    delete globalThis.window;
  }
});

test("a fit stops itself once its container has been placed and then detached", async () => {
  const added = fakeWindowListeners();
  try {
    const box = document.createElement("div");
    document.body.append(box);
    scheduleFit(box);
    await new Promise((r) => setTimeout(r, 5)); // first fit: placed
    box.remove();
    added.get("resize")(); // the next resize finds it detached
    await new Promise((r) => setTimeout(r, 5));
    assert.equal(added.size, 0);
  } finally {
    delete globalThis.window;
  }
});

