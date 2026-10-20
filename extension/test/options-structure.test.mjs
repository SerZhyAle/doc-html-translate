// The options page's structure against SETTINGS-NAVIGATION 2.0, BOOLEAN-OPTION 2.0 and COLLAPSIBLE-GROUP 1.1:
// what a static read of the markup decides. Geometry, focus and interaction are measured in a real browser by
// tools/uicheck (node tools/uicheck/run.mjs --only extension); this file keeps the wiring honest in `npm test`.

import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { parseHTML } from "linkedom";
import { OPTIONS_PAGES, GROUP_IDS } from "../src/ui-settings.js";
import { GLYPHS } from "../src/glyphs.js";

const src = path.join(import.meta.dirname, "..", "src");
const load = (file) => parseHTML(fs.readFileSync(path.join(src, file), "utf8")).document;
const options = load("options.html");
const popup = load("popup.html");

test("the rail lists the pages in order, About last, each with a glyph and a name", () => {
  const tabs = [...options.querySelectorAll('[role="tablist"] [role="tab"]')];
  assert.deepEqual(tabs.map((t) => t.getAttribute("data-page")), OPTIONS_PAGES);
  assert.ok(tabs.length <= 7);
  for (const tab of tabs) {
    const page = tab.getAttribute("data-page");
    assert.equal(tab.id, `tab-${page}`);
    assert.equal(tab.getAttribute("aria-controls"), `page-${page}`);
    assert.ok(tab.querySelector("[data-glyph]"), `${page}: a glyph`);
    assert.ok(tab.querySelector("[data-i18n]"), `${page}: a localized name`);
    assert.equal(tab.getAttribute("tabindex"), page === OPTIONS_PAGES[0] ? "0" : "-1", `${page}: one tab stop`);
    assert.equal(tab.getAttribute("aria-selected"), page === OPTIONS_PAGES[0] ? "true" : "false");
  }
  assert.equal(options.querySelector("#railList").getAttribute("aria-orientation"), "vertical");
});

test("every page is in the document, named by its tab, and only the first is shown", () => {
  for (const page of OPTIONS_PAGES) {
    const el = options.getElementById(`page-${page}`);
    assert.ok(el, page);
    assert.equal(el.getAttribute("role"), "tabpanel");
    assert.equal(el.getAttribute("aria-labelledby"), `tab-${page}`);
    assert.equal(el.hasAttribute("hidden"), page !== OPTIONS_PAGES[0]);
    assert.ok(el.querySelector(".page-head h2"), `${page}: the shared title treatment`);
    assert.ok(el.querySelector(".page-head [data-i18n^='purpose']"), `${page}: one purpose line`);
  }
  assert.equal(options.querySelectorAll(".page").length, OPTIONS_PAGES.length);
});

test("the first page holds the language and the themes; About holds the whole set", () => {
  const general = options.getElementById("page-general");
  for (const id of ["ui-lang", "chrome-theme", "theme", "lang", "enabled"]) assert.ok(general.querySelector(`#${id}`), id);
  const about = options.getElementById("page-about");
  for (const id of ["about-ver", "about-links", "about-updates", "about-support", "copy-diag", "save-diag", "res-table", "rel-table"]) {
    assert.ok(about.querySelector(`#${id}`), `About lacks #${id}`);
  }
});

test("groups appear only on About and the popup, with stable ids and a summary", () => {
  const groups = [...options.querySelectorAll("details")];
  assert.deepEqual(groups.map((g) => g.id), ["aboutSupport", "aboutResources", "aboutRelated"]);
  for (const g of groups) {
    assert.ok(g.classList.contains("group"));
    assert.ok(GROUP_IDS.includes(g.id), `${g.id} is not a remembered group id`);
    assert.ok(g.querySelector(":scope > summary"));
    assert.ok(g.closest("#page-about"), "single-topic pages show their controls directly");
  }
  assert.equal(options.getElementById("aboutSupport").hasAttribute("open"), true, "Support is open by default");
  assert.equal(options.getElementById("aboutResources").hasAttribute("open"), false);
  const popupGroups = [...popup.querySelectorAll("details")];
  assert.deepEqual(popupGroups.map((g) => g.id), ["popupOcrLangs"]);
  assert.ok(GROUP_IDS.includes("popupOcrLangs"));
});

test("every boolean option is a checkbox row: a heading label, a description, help outside the label", () => {
  for (const [name, doc] of [["options", options], ["popup", popup]]) {
    const boxes = [...doc.querySelectorAll("input.opt-box")];
    assert.ok(boxes.length >= 3, `${name}: ${boxes.length} options`);
    for (const box of boxes) {
      assert.equal(box.getAttribute("type"), "checkbox");
      const label = doc.querySelector(`label.opt-name[for="${box.id}"]`);
      assert.ok(label, `${name}: #${box.id} has no heading label`);
      const described = box.getAttribute("aria-describedby");
      if (described) assert.ok(doc.getElementById(described), `${name}: #${box.id} describes a missing #${described}`);
      for (const wrap of box.closest(".opt").querySelectorAll(".opt-help-wrap")) {
        assert.equal(wrap.closest("label"), null, "help stays outside the activation label");
        assert.ok(wrap.getAttribute("data-opt-help") && wrap.getAttribute("data-opt-name"));
        assert.equal(wrap.querySelector("button.opt-help").getAttribute("type"), "button");
      }
    }
  }
  assert.deepEqual([...options.querySelectorAll("input.opt-box")].map((b) => b.id), ["enabled", "ocr-images", "allow-remote"]);
  assert.deepEqual([...popup.querySelectorAll("input.opt-box")].map((b) => b.id), ["ocr-images", "global", "site"]);
});

test("every control that is a command is a typed button, and the popup's Settings is one", () => {
  for (const doc of [options, popup]) {
    for (const b of doc.querySelectorAll("button")) assert.ok(b.getAttribute("type"), `an untyped button: ${b.outerHTML.slice(0, 80)}`);
  }
  assert.equal(popup.getElementById("opts").tagName.toLowerCase(), "button");
  assert.equal(popup.querySelector('a[href="#"]'), null, "a command is never a dead link");
  // The primary rank is single on the task panel.
  assert.equal(popup.querySelectorAll("button.primary").length, 1);
});

test("outside links keep link semantics and the external marker", () => {
  for (const a of popup.querySelectorAll('a[href^="http"], a[href^="mailto"]')) {
    assert.ok(a.querySelector('[data-glyph="nav.open-external"]'), `${a.getAttribute("href")} lacks the marker`);
    assert.ok(a.classList.contains("ext"));
  }
});

test("every glyph the pages name is a vocabulary glyph of the extension's table", () => {
  for (const doc of [options, popup]) {
    for (const el of doc.querySelectorAll("[data-glyph]")) assert.ok(GLYPHS[el.getAttribute("data-glyph")], el.getAttribute("data-glyph"));
  }
});

test("each page loads the boot script first and its module last, with no inline script or style", () => {
  for (const file of ["options.html", "popup.html", "ocr.html"]) {
    const html = fs.readFileSync(path.join(src, file), "utf8");
    assert.match(html, /<html[^>]*\bdata-boot\b/, `${file}: held until ready`);
    assert.ok(html.indexOf("ui-boot.js") < html.indexOf("</head>"), `${file}: ui-boot.js runs before the first paint`);
    assert.match(html, /<link rel="stylesheet" href="ui-sizing\.css">[\s\S]*<link rel="stylesheet" href="ui-components\.css">/);
    assert.doesNotMatch(html, /<script(?![^>]*\bsrc=)/, `${file}: an inline script`);
  }
});
