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

  const text = await reportText("26.811.1600", {
    disabledHosts: ["secret-intranet.example.com"], allowedHosts: ["private-library.example.com"], siteMode: "allowlist",
  });
  assert.ok(!text.includes("http"), `report contains a URL:\n${text}`);
  assert.ok(!text.includes("secret-intranet.example.com"), `report names a disabled host:\n${text}`);
  assert.ok(!text.includes("private-library.example.com"), `report names an allowed host:\n${text}`);
  // The count is what a report needs; which sites someone reads is not diagnostic.
  assert.match(text, /disabled hosts: 1/);
  assert.match(text, /allowed hosts: 1/);
  assert.match(text, /site mode: allowlist/);
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

test("a new run forgets the previous run's error and page count", async () => {
  await recordRun({ format: "pdf" });
  await recordRun({ pages: 12 });
  await recordRun({ error: "Couldn't open this PDF" });
  await recordRun({ format: "epub" });
  const run = await readRun();
  assert.equal(run.format, "epub");
  assert.equal(run.error, "");
  assert.equal(run.pages, 0);
});

test("unawaited writes land in call order without overwriting each other", async () => {
  recordRun({ format: "txt" });
  recordRun({ pages: 3 });
  await recordRun({ error: "late" });
  const run = await readRun();
  assert.deepEqual([run.format, run.pages, run.error], ["txt", 3, "late"]);
});

test("reportText emits strictly count-only host metrics and expected fields only", async () => {
  await recordRun({ format: "epub", pages: 50, error: "" });
  const text = await reportText("26.811.1600", {
    theme: "light",
    disabledHosts: ["a.com", "b.com"],
    allowedHosts: ["c.com"],
  });
  const lines = text.trim().split("\n");
  const fieldKeys = lines.map((l) => l.split(":")[0].trim());
  const expectedKeys = [
    "edition", "version", "platform", "user agent", "interface language",
    "auto reflow", "site mode", "theme", "source language", "ocr", "ocr language",
    "disabled hosts", "allowed hosts", "last format", "last pages", "last error", "last run at",
  ];
  assert.deepEqual(fieldKeys, expectedKeys);
  // Disabled and allowed hosts must be numeric counts only
  const disabledLine = lines.find((l) => l.startsWith("disabled hosts:"));
  const allowedLine = lines.find((l) => l.startsWith("allowed hosts:"));
  assert.equal(disabledLine, "disabled hosts: 2");
  assert.equal(allowedLine, "allowed hosts: 1");
});
