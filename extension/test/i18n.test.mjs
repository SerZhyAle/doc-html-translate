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
  const attr = /data-i18n(?:-title|-ph)?="([A-Za-z0-9_]+)"/g;
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

test("every t() key used in the sources is defined", () => {
  const en = readLocale("en");
  const call = /\bt\(\s*"([A-Za-z0-9_]+)"/g;
  for (const file of ["viewer.js", "options.js", "popup.js", "i18n.js"]) {
    const js = fs.readFileSync(path.join(srcDir, file), "utf8");
    for (const m of js.matchAll(call)) {
      assert.ok(m[1] in en, `${file}: t("${m[1]}") has no English message`);
    }
  }
});

// The standalone image-OCR page resolves its own messages through a local msg() helper rather
// than i18n.js, so the scan above never saw it and a key added only there would ship missing
// from all thirteen files.
test("every msg() key used in the standalone OCR page is defined", () => {
  const en = readLocale("en");
  const call = /\bmsg\(\s*"([A-Za-z0-9_]+)"/g;
  const js = fs.readFileSync(path.join(srcDir, "ocr.js"), "utf8");
  let seen = 0;
  for (const m of js.matchAll(call)) {
    seen++;
    assert.ok(m[1] in en, `ocr.js: msg("${m[1]}") has no English message`);
  }
  assert.ok(seen > 3, `only ${seen} msg() calls found - the scan is wrong`);
});
