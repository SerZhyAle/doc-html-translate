import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { register } from "node:module";
import { parseHTML } from "linkedom";

const overlayStub = `
export const ocrLangToHtmlLang = (code) => ({ jpn: "ja", jpn_vert: "ja", eng: "en" })[code] || "en";
export const makeBadge = () => document.createElement("span");
export const overlayImage = async (...args) => globalThis.__overlayImage(...args);
`;
const langStub = `export const langLabel = (code) => code;`;
const hooks = `
export async function resolve(specifier, context, next) {
  if (/(^|\\/)ocr-overlay\\.js/.test(specifier))
    return { url: "data:text/javascript," + encodeURIComponent(${JSON.stringify(overlayStub)}), shortCircuit: true };
  if (/(^|\\/)ocr-lang\\.js/.test(specifier))
    return { url: "data:text/javascript," + encodeURIComponent(${JSON.stringify(langStub)}), shortCircuit: true };
  return next(specifier, context);
}
`;
register(`data:text/javascript,${encodeURIComponent(hooks)}`, import.meta.url);

test("image OCR declares the chosen Japanese source language before recognition", async () => {
  const html = readFileSync(new URL("../src/ocr.html", import.meta.url), "utf8");
  const { document, window } = parseHTML(html);
  assert.equal(document.documentElement.hasAttribute("lang"), false,
    "the initial page must not declare English while OCR is pending");

  globalThis.document = document;
  globalThis.window = window;
  globalThis.location = { search: "?src=https%3A%2F%2Fexample.test%2Fimage.png" };
  globalThis.chrome = {
    storage: { local: { get: async () => ({ options: { ocrLang: "jpn" } }) } },
    i18n: { getMessage: () => "" },
  };
  let called = false;
  globalThis.__overlayImage = async (_src, options) => {
    called = true;
    assert.equal(options.lang, "jpn");
    assert.equal(document.documentElement.lang, "ja");
    assert.equal(document.querySelector('meta[http-equiv="content-language"]').content, "ja");
    const result = document.createElement("div");
    return result;
  };

  await import("../src/ocr.js");
  assert.equal(called, true);
  assert.equal(document.documentElement.lang, "ja");
});
