import test from "node:test";
import assert from "node:assert/strict";
import { DEFAULT_OPTIONS } from "../src/defaults.js";
import { makeSettingsFile, validateSettingsFile, changedSettings, SETTINGS_VERSION } from "../src/settings-transfer.js";
import { DEFAULT_UI_STATE } from "../src/ui-settings.js";

test("settings file round-trips both site lists and the interface language", () => {
  const source = { ...DEFAULT_OPTIONS, siteMode: "allowlist", disabledHosts: ["off.example"], allowedHosts: ["on.example"], ocrLang: "jpn_vert" };
  const file = makeSettingsFile(source, "uk");
  const restored = validateSettingsFile(JSON.parse(JSON.stringify(file)));
  assert.deepEqual(restored.options, source);
  assert.equal(restored.uiLang, "uk");
  assert.deepEqual(changedSettings(file, restored), []);
});

test("the interface settings and the remembered page and groups round-trip", () => {
  const source = { ...DEFAULT_OPTIONS, uiSize: "huge", textSize: "xlarge", inputProfile: "touch", chromeTheme: "dark" };
  const ui = { page: "about", groups: { aboutSupport: false, aboutResources: true } };
  const file = makeSettingsFile(source, "ar", ui);
  assert.equal(file.version, SETTINGS_VERSION);
  const restored = validateSettingsFile(JSON.parse(JSON.stringify(file)));
  assert.deepEqual(restored.options, source);
  assert.deepEqual(restored.ui, ui);
  assert.deepEqual(changedSettings(file, restored), []);
  assert.deepEqual(changedSettings(makeSettingsFile(DEFAULT_OPTIONS, "", DEFAULT_UI_STATE), restored).sort(), ["chromeTheme", "inputProfile", "textSize", "uiLang", "uiSize", "ui"].sort());
});

test("page viewports stay out of the file: they belong to one device", () => {
  const file = makeSettingsFile(DEFAULT_OPTIONS, "", { page: "data", groups: {}, scroll: { data: { id: "x", off: 10 } } });
  assert.deepEqual(file.ui, { page: "data", groups: {} });
  assert.ok(!JSON.stringify(file).includes("scroll"));
});

test("a version 1 file still imports: what it lacks takes its default", () => {
  const v1 = makeSettingsFile(DEFAULT_OPTIONS, "ru");
  v1.version = 1;
  delete v1.ui;
  for (const key of ["uiSize", "textSize", "inputProfile", "chromeTheme"]) delete v1.options[key];
  const restored = validateSettingsFile(v1);
  assert.deepEqual(restored.options, DEFAULT_OPTIONS);
  assert.equal(restored.uiLang, "ru");
  assert.deepEqual(restored.ui, DEFAULT_UI_STATE);
});

test("an interface value this build does not know falls back instead of refusing the file", () => {
  const file = makeSettingsFile(DEFAULT_OPTIONS);
  file.options.uiSize = "gigantic";
  file.options.chromeTheme = 7;
  file.ui = { page: "retired-page", groups: { aboutSupport: "yes", unknownGroup: true, aboutRelated: true } };
  const restored = validateSettingsFile(file);
  assert.equal(restored.options.uiSize, DEFAULT_OPTIONS.uiSize);
  assert.equal(restored.options.chromeTheme, DEFAULT_OPTIONS.chromeTheme);
  assert.equal(restored.ui.page, DEFAULT_UI_STATE.page);
  assert.deepEqual(restored.ui.groups, { aboutRelated: true });
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
    (f) => { f.version = 3; },
    (f) => { f.options.theme = "injected"; },
    (f) => { f.options.allowedHosts = ["https://site.example/path"]; },
    (f) => { delete f.options.ocrImages; },
    (f) => { f.options.surprise = true; },
    (f) => { f.readingPositions = { secret: "text" }; },
    (f) => { f.ui = "page"; },
  ]) {
    const candidate = structuredClone(file);
    mutation(candidate);
    assert.throws(() => validateSettingsFile(candidate));
  }
});
