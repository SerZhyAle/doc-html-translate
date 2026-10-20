// ui-profile.js applies the saved interface settings to a page's root. The input profile is resolved once when the
// page opens (UI-SIZING 1.1 section 2 rule 1) and is never recomputed from an event.

import test from "node:test";
import assert from "node:assert/strict";
import { parseHTML } from "linkedom";
import { applyUiProfile, revealPage } from "../src/ui-profile.js";
import { UI_SIZES, TEXT_SIZES, INPUT_PROFILES, CHROME_THEMES, OPTIONS_PAGES, GROUP_IDS, normalizeUiState } from "../src/ui-settings.js";

const root = () => parseHTML("<!doctype html><html data-boot><body></body></html>").document.documentElement;

test("the saved size, text size, controls and theme become attributes of the root", () => {
  const r = root();
  applyUiProfile({ uiSize: "xlarge", textSize: "large", inputProfile: "touch", chromeTheme: "dark" }, r);
  assert.deepEqual([r.dataset.size, r.dataset.text, r.dataset.input, r.dataset.chromeTheme], ["xlarge", "large", "touch", "dark"]);
});

test("the system theme sets nothing, so the browser's own light or dark setting keeps deciding", () => {
  const r = root();
  applyUiProfile({ chromeTheme: "dark" }, r);
  applyUiProfile({ chromeTheme: "system" }, r);
  assert.equal(r.dataset.chromeTheme, undefined);
  assert.equal(r.getAttribute("data-chrome-theme"), null);
});

test("an unknown or missing value falls back to the ordinary default", () => {
  const r = root();
  applyUiProfile({ uiSize: "gigantic", textSize: 0, inputProfile: "stylus", chromeTheme: "neon" }, r);
  assert.deepEqual([r.dataset.size, r.dataset.text, r.dataset.chromeTheme], ["ordinary", "ordinary", undefined]);
  applyUiProfile(null, r);
  assert.equal(r.dataset.size, "ordinary");
});

test("automatic reads the surface once and keeps the answer", () => {
  const r = root();
  globalThis.matchMedia = (q) => ({ matches: q === "(pointer: coarse)" });
  try {
    applyUiProfile({ inputProfile: "auto" }, r);
    assert.equal(r.dataset.input, "touch");
    // The pointer changes under the page (a mouse is plugged in): the page does not follow it.
    globalThis.matchMedia = () => ({ matches: false });
    applyUiProfile({ inputProfile: "auto" }, r);
    assert.equal(r.dataset.input, "touch");
    applyUiProfile({ inputProfile: "pointer" }, r);
    assert.equal(r.dataset.input, "pointer", "an explicit choice is the only thing that changes it");
  } finally {
    delete globalThis.matchMedia;
  }
});

test("a surface with no pointer at all is a touch surface, and no media queries means a pointer", () => {
  const touch = root();
  globalThis.matchMedia = (q) => ({ matches: q === "(any-pointer: none)" });
  try { applyUiProfile({}, touch); } finally { delete globalThis.matchMedia; }
  assert.equal(touch.dataset.input, "touch");
  const plain = root();
  applyUiProfile({}, plain);
  assert.equal(plain.dataset.input, "pointer");
});

test("revealing the page removes the boot hold", () => {
  const r = root();
  assert.equal(r.hasAttribute("data-boot"), true);
  revealPage(r);
  assert.equal(r.hasAttribute("data-boot"), false);
});

test("the vocabulary lists are the ranges the stylesheet and the desktop launcher declare", () => {
  assert.deepEqual(UI_SIZES, ["compact", "ordinary", "large", "xlarge", "huge"]);
  assert.deepEqual(TEXT_SIZES, ["ordinary", "large", "xlarge", "huge"]);
  assert.deepEqual(INPUT_PROFILES, ["auto", "pointer", "touch"]);
  assert.deepEqual(CHROME_THEMES, ["system", "light", "dark"]);
  assert.ok(OPTIONS_PAGES.length <= 7 && OPTIONS_PAGES.at(-1) === "about", "seven pages at most, About last");
});

test("the remembered state falls back to a valid one", () => {
  assert.deepEqual(normalizeUiState(undefined), { page: "general", groups: {} });
  const state = normalizeUiState({
    page: "retired", groups: { aboutSupport: false, other: true, aboutRelated: "yes" },
    scroll: { about: { id: "grpX", off: 12.4 }, ocr: { id: 5, off: 1 }, nowhere: { id: "a", off: 1 } },
  });
  assert.equal(state.page, "general");
  assert.deepEqual(state.groups, { aboutSupport: false });
  assert.deepEqual(state.scroll, { about: { id: "grpX", off: 12 } });
  assert.ok(GROUP_IDS.includes("popupOcrLangs"));
});
