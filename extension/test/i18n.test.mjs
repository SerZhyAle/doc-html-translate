// The extension's half of the 13-language invariant. A key missing from a locale silently falls
// back to English at runtime and looks perfectly fine, so only a static check catches a language
// that was left behind.

import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";

const root = path.join(import.meta.dirname, "..");
const localesDir = path.join(root, "_locales");
const srcDir = path.join(root, "src");

// The UI language set, by _locales directory name. Chrome names the Chinese directory zh_CN.
const LOCALES = ["en", "ru", "uk", "de", "it", "es", "fr", "pt", "ar", "hi", "bn", "ur", "zh_CN"];

// Listing copy, not runtime strings: the long store description and the screenshot captions are
// pasted into the store dashboards per language and are maintained with the rest of the listing.
const LISTING_ONLY = new Set(["storeDescription", "shot1Caption", "shot2Caption", "shot3Caption"]);

function readLocale(dir) {
  return JSON.parse(fs.readFileSync(path.join(localesDir, dir, "messages.json"), "utf8"));
}

function runtimeKeys(messages) {
  return Object.keys(messages).filter((k) => !LISTING_ONLY.has(k));
}

test("every shipped locale directory exists", () => {
  const found = fs.readdirSync(localesDir, { withFileTypes: true })
    .filter((d) => d.isDirectory())
    .map((d) => d.name)
    .sort();
  assert.deepEqual(found, [...LOCALES].sort());
});

test("every locale carries the full runtime key set with non-empty messages", () => {
  const en = readLocale("en");
  const expected = runtimeKeys(en);
  assert.ok(expected.length > 40, `English has only ${expected.length} runtime keys`);

  for (const dir of LOCALES) {
    const messages = readLocale(dir);
    for (const key of expected) {
      assert.ok(key in messages, `${dir}: missing key ${key}`);
      assert.ok(messages[key].message.trim() !== "", `${dir}: empty message for ${key}`);
    }
    for (const key of runtimeKeys(messages)) {
      assert.ok(key in en, `${dir}: key ${key} does not exist in English`);
    }
  }
});

test("every data-i18n key used in markup is defined", () => {
  const en = readLocale("en");
  const attr = /data-i18n(?:-title|-ph|-aria)?="([A-Za-z0-9_]+)"/g;
  let seen = 0;
  for (const file of ["viewer.html", "options.html", "popup.html"]) {
    const html = fs.readFileSync(path.join(srcDir, file), "utf8");
    for (const m of html.matchAll(attr)) {
      seen++;
      assert.ok(m[1] in en, `${file}: data-i18n key ${m[1]} has no English message`);
    }
  }
  assert.ok(seen > 20, `only ${seen} tagged nodes found - the scan is wrong`);
});

// Every source file resolves its strings through i18n.js's t() - directly, or through a local
// msg() wrapper that forwards to it (popup.js, options.js, the OCR-language picker, the standalone
// image-OCR page). The wrapper call sites look identical, so both spellings are scanned: a key
// used only through a wrapper would otherwise ship missing from all thirteen files.
test("every t()/msg() key used in the sources is defined", () => {
  const en = readLocale("en");
  const call = /\b(?:t|msg)\(\s*"([A-Za-z0-9_]+)"/g;
  let seen = 0;
  for (const file of ["viewer.js", "options.js", "popup.js", "i18n.js", "ocr-lang-ui.js", "ocr.js"]) {
    const js = fs.readFileSync(path.join(srcDir, file), "utf8");
    for (const m of js.matchAll(call)) {
      seen++;
      assert.ok(m[1] in en, `${file}: t("${m[1]}") has no English message`);
    }
  }
  assert.ok(seen > 30, `only ${seen} message calls found - the scan is wrong`);
});

// ICON-RENDER rule 8: a landmark's accessible name is the meaning's name in the interface
// language. The TOC <nav> and the search panel are translated as one region each, so applyI18n
// must reach the element it is handed, not only what sits inside it.
test("applyI18n names the region it is given, not only its descendants", async () => {
  const { parseHTML } = await import("linkedom");
  const viewer = fs.readFileSync(path.join(srcDir, "viewer.html"), "utf8");
  const { document } = parseHTML(viewer);
  globalThis.document = document;
  globalThis.chrome = { i18n: { getMessage: (key) => `[${key}]`, getUILanguage: () => "ur" } };
  const { applyI18n } = await import("../src/i18n.js");

  const toc = document.getElementById("toc");
  assert.equal(toc.getAttribute("data-i18n-aria"), "ttToc", "the TOC landmark is tagged");
  applyI18n(toc);
  assert.equal(toc.getAttribute("aria-label"), "[ttToc]");
  assert.equal(toc.getAttribute("dir"), "rtl");

  const panel = document.getElementById("search-panel");
  applyI18n(panel);
  assert.equal(panel.getAttribute("aria-label"), "[vSearch]");
});

// docs/PARITY.md, reader chrome floor: a right-to-left interface mirrors the chrome through
// logical properties. The popup and the options page are chrome from edge to edge, so their
// styles name no physical side at all, and the popup's switch knob travels toward the reading end.
test("the popup and options styles mirror for right-to-left languages", () => {
  const physical = /\b(?:margin|padding|border)-(?:left|right)\b|(?:^|[\s;{"])(?:left|right)\s*:|float\s*:\s*(?:left|right)|text-align\s*:\s*(?:left|right)/;
  for (const file of ["popup.html", "options.html"]) {
    const html = fs.readFileSync(path.join(srcDir, file), "utf8");
    const styles = [
      ...[...html.matchAll(/<style>([\s\S]*?)<\/style>/g)].map((m) => m[1]),
      ...[...html.matchAll(/\sstyle="([^"]*)"/g)].map((m) => m[1]),
    ];
    assert.ok(styles.length > 0, `${file}: no styles found - the scan is wrong`);
    for (const css of styles) {
      for (const line of css.split("\n")) assert.doesNotMatch(line, physical, `${file}: ${line.trim()}`);
    }
  }
  const popup = fs.readFileSync(path.join(srcDir, "popup.html"), "utf8");
  assert.match(popup, /\[dir="rtl"\] input:checked \+ \.slider::before \{ transform: translateX\(-16px\); \}/);
});
