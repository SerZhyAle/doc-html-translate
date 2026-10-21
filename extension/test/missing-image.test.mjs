// A document picture that cannot be shown draws content.image with its alt text (ICON-EXTERNAL
// section 13 F; ticket 122). The DOM comes from linkedom via ./_dom.mjs, imported first.
//
// linkedom does not model the capture phase (its dispatchEvent walks target -> parents only), so the
// listener's wiring is asserted on a recording document and its effect by calling the recorded
// handler with a real <img>. That the engine really delivers a non-bubbling error event to a
// capture listener on the document is measured in Chromium by tools/uicheck (reader scenario).

import "./_dom.mjs";
import { el } from "./_dom.mjs";

import { test } from "node:test";
import assert from "node:assert/strict";

import { GLYPHS } from "../src/glyphs.js";
import {
  MISSING_IMG_CLASS,
  installMissingImageFallback,
  missingImageFor,
  missingImagePlaceholder,
  showMissingImage,
} from "../src/missing-image.js";

const pathOf = (node) => node.querySelector("svg path").getAttribute("d");

test("missingImagePlaceholder: the content.image glyph, then the alt text", () => {
  const box = missingImagePlaceholder("A missing figure");
  assert.equal(box.className, MISSING_IMG_CLASS);
  assert.equal(pathOf(box), GLYPHS["content.image"]);
  assert.equal(box.querySelector("svg").getAttribute("aria-hidden"), "true", "the glyph is decorative: the alt text names it");
  assert.equal(box.textContent, "A missing figure");
});

test("missingImagePlaceholder: no alt, or an empty one, is the glyph alone", () => {
  for (const alt of [undefined, null, "", "   "]) {
    const box = missingImagePlaceholder(alt);
    assert.equal(pathOf(box), GLYPHS["content.image"]);
    assert.equal(box.textContent, "", `alt ${JSON.stringify(alt)}`);
    assert.equal(box.querySelectorAll("span").length, 0);
  }
});

test("missingImagePlaceholder: the alt is text, never markup", () => {
  const box = missingImagePlaceholder(`<b onclick="x()">bold</b>`);
  assert.equal(box.querySelector("b"), null);
  assert.equal(box.textContent, `<b onclick="x()">bold</b>`);
});

test("showMissingImage: swaps the picture in place and leaves its src alone", () => {
  const div = el(`<div><p>before</p><img src="gone.png" alt="Figure 1"><p>after</p></div>`, "div");
  const img = div.querySelector("img");
  const box = showMissingImage(img);
  assert.equal(div.querySelector("img"), null, "the broken picture is gone");
  assert.deepEqual([...div.children].map((c) => c.localName + (c.className ? "." + c.className : "")), ["p", "span.missing-img", "p"]);
  assert.equal(box.textContent, "Figure 1");
  assert.equal(img.getAttribute("src"), "gone.png", "the failed picture is not retargeted, so nothing is requested again");
  assert.equal(missingImageFor(img), box);
});

test("showMissingImage: a picture outside any tree is left as it is", () => {
  const img = document.createElement("img");
  img.setAttribute("alt", "x");
  assert.equal(showMissingImage(img), null);
  assert.equal(missingImageFor(img), null);
});

test("installMissingImageFallback: one capture-phase error listener on the document", () => {
  const calls = [];
  installMissingImageFallback({ addEventListener: (...args) => calls.push(args) });
  assert.equal(calls.length, 1);
  const [type, handler, capture] = calls[0];
  assert.equal(type, "error");
  assert.equal(typeof handler, "function");
  assert.equal(capture, true, "an error event does not bubble: only a capture listener sees every picture");
});

test("the installed listener replaces only the picture whose load failed", () => {
  const div = el(`<div><img id="ok" src="blob:fine" alt="Fine"><img id="bad" src="gone.png" alt="Broken"></div>`, "div");
  let handler;
  installMissingImageFallback({ addEventListener: (type, fn) => { handler = fn; } });

  // a valid picture raises no error event, so the handler never sees it
  handler({ target: div.querySelector("#bad") });
  assert.ok(div.querySelector("#ok"), "the valid picture is untouched");
  assert.equal(div.querySelector("#ok").getAttribute("src"), "blob:fine");
  assert.equal(div.querySelector("#bad"), null);
  assert.equal(div.querySelector(".missing-img").textContent, "Broken");
});

test("the installed listener ignores errors that are not a picture's", () => {
  const div = el(`<div><script src="x.js"></script><span>kept</span></div>`, "div");
  let handler;
  installMissingImageFallback({ addEventListener: (type, fn) => { handler = fn; } });
  handler({ target: div.querySelector("script") });
  handler({ target: { localName: "window" } });
  handler({ target: null });
  assert.equal(div.querySelector(".missing-img"), null);
  assert.ok(div.querySelector("script") && div.querySelector("span"));
});
