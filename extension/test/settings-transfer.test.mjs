import test from "node:test";
import assert from "node:assert/strict";
import { DEFAULT_OPTIONS } from "../src/defaults.js";
import { makeSettingsFile, validateSettingsFile, changedSettings } from "../src/settings-transfer.js";

test("settings file round-trips both site lists and the interface language", () => {
  const source = { ...DEFAULT_OPTIONS, siteMode: "allowlist", disabledHosts: ["off.example"], allowedHosts: ["on.example"], ocrLang: "jpn_vert" };
  const file = makeSettingsFile(source, "uk");
  const restored = validateSettingsFile(JSON.parse(JSON.stringify(file)));
  assert.deepEqual(restored.options, source);
  assert.equal(restored.uiLang, "uk");
  assert.deepEqual(changedSettings(file, restored), []);
});

test("export uses an allowlist, excluding browsing and document data", () => {
  const file = makeSettingsFile({ ...DEFAULT_OPTIONS, readingPositions: { secret: "document text" }, lastRun: { url: "https://secret.example" } });
  const json = JSON.stringify(file);
  assert.ok(!json.includes("document text"));
  assert.ok(!json.includes("secret.example"));
  assert.ok(!json.includes("readingPositions"));
  assert.ok(!json.includes("lastRun"));
});

test("foreign, corrupt and incomplete files fail before import", () => {
  const file = makeSettingsFile(DEFAULT_OPTIONS);
  for (const mutation of [
    (f) => { f.format = "another-extension"; },
    (f) => { f.version = 2; },
    (f) => { f.options.theme = "injected"; },
    (f) => { f.options.allowedHosts = ["https://site.example/path"]; },
    (f) => { delete f.options.ocrImages; },
    (f) => { f.readingPositions = { secret: "text" }; },
  ]) {
    const candidate = structuredClone(file);
    mutation(candidate);
    assert.throws(() => validateSettingsFile(candidate));
  }
});
