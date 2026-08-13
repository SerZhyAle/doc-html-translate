// The diagnostics report is the browser edition's whole answer to "send the author some
// evidence". It has to carry enough to act on and nothing the user did not agree to hand over,
// so both halves are pinned here: the fields that must be present, and the ones that must not.

import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";

const root = path.join(import.meta.dirname, "..");
const LOCALES = ["en", "ru", "uk", "de", "it", "es", "fr", "pt", "ar", "hi", "bn", "ur", "zh_CN"];

// diagnostics.js reads chrome.storage and navigator; stub both before importing it.
const store = {};
globalThis.chrome = {
  storage: {
    local: {
      get: async (key) => (key in store ? { [key]: store[key] } : {}),
      set: async (obj) => Object.assign(store, obj),
    },
  },
  i18n: { getUILanguage: () => "en" },
};
// Node ships its own read-only `navigator`, so the browser's has to be defined over it.
Object.defineProperty(globalThis, "navigator", {
  value: { platform: "Win32", userAgent: "Mozilla/5.0 Test" },
  configurable: true,
});

const { recordRun, readRun, reportText } = await import("../src/diagnostics.js");

test("reportText names the version and the last format", async () => {
  await recordRun({ format: "epub", pages: 42 });

  const text = await reportText("26.811.1600", { theme: "dark", disabledHosts: ["example.com", "example.org"] });
  assert.match(text, /version: 26\.811\.1600/);
  assert.match(text, /last format: epub/);
  assert.match(text, /last pages: 42/);
  assert.match(text, /theme: dark/);
});

test("the report carries no URL and no host name", async () => {
  await recordRun({ format: "pdf", pages: 3 });

  const text = await reportText("26.811.1600", { disabledHosts: ["secret-intranet.example.com"] });
  assert.ok(!text.includes("http"), `report contains a URL:\n${text}`);
  assert.ok(!text.includes("secret-intranet.example.com"), `report names a disabled host:\n${text}`);
  // The count is what a report needs; which sites someone reads is not diagnostic.
  assert.match(text, /disabled hosts: 1/);
});

test("recordRun keeps no document identity and truncates the error", async () => {
  await recordRun({ format: "fb2", pages: 7, error: "x".repeat(500) });

  const run = await readRun();
  assert.deepEqual(Object.keys(run).sort(), ["at", "error", "format", "pages"]);
  assert.equal(run.error.length, 200);
});

test("the About block's keys exist in every locale", () => {
  for (const dir of LOCALES) {
    const messages = JSON.parse(fs.readFileSync(path.join(root, "_locales", dir, "messages.json"), "utf8"));
    for (const key of ["optAbout", "btnCopyDiag", "optCopied", "diagHint"]) {
      assert.ok(key in messages, `${dir}: missing key ${key}`);
      assert.ok(messages[key].message.trim() !== "", `${dir}: empty message for ${key}`);
    }
  }
});
