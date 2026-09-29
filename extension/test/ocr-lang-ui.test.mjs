// Behaviour tests for ocr-lang-ui.js - the OCR-language picker the popup and the options page
// both render. The ocr-lang.js dependency is stubbed at resolution (as vendor is in the popup
// tests), so a download can be made to fail and then succeed on retry. What matters here is the
// failure story: it used to vanish into the console while the button silently snapped back.

import { test } from "node:test";
import assert from "node:assert/strict";
import { register } from "node:module";
import { parseHTML } from "linkedom";

const STUB = `
export const LANGS = [
  { code: "eng", name: "English" },
  { code: "deu", name: "German" },
  { code: "rus", name: "Russian" },
];
export const getInstalledLangs = async () => globalThis.__stubInstalled || ["eng"];
export const downloadLang = (...a) => globalThis.__stubDownload(...a);
`;

const HOOKS = `
export async function resolve(specifier, context, next) {
  if (/(^|\\/)ocr-lang\\.js/.test(specifier)) {
    return { url: "data:text/javascript," + encodeURIComponent(${JSON.stringify(STUB)}), shortCircuit: true };
  }
  return next(specifier, context);
}
`;
register(`data:text/javascript,${encodeURIComponent(HOOKS)}`, import.meta.url);

let caseSeq = 0;

async function render({ ocrImages = true, installed = ["eng"] } = {}) {
  const { document, window } = parseHTML("<!DOCTYPE html><html><body></body></html>");
  globalThis.document = document;
  globalThis.__stubInstalled = installed;
  globalThis.chrome = {
    runtime: { getURL: (p) => `chrome-extension://test/${p}` },
    storage: {
      local: {
        get: async () => ({ options: { ocrImages } }),
        set: async () => {},
      },
    },
    i18n: { getMessage: () => "", getUILanguage: () => "en" },
  };
  const el = document.createElement("div");
  document.body.append(el);
  const { renderOcrLangs } = await import(`../src/ocr-lang-ui.js?case=${++caseSeq}`);
  await renderOcrLangs(el);
  return { el, window };
}

// the row holding lang's name, or null
function rowFor(el, name) {
  return [...el.querySelectorAll(".ocr-lang")].find((r) => r.textContent.includes(name)) || null;
}

test("installed languages are radios, the rest offer a Download button", async () => {
  const { el } = await render();
  assert.ok(rowFor(el, "English").querySelector("input[type=radio]"));
  const deu = rowFor(el, "German");
  assert.ok(!deu.querySelector("input"));
  assert.equal(deu.querySelector("button").textContent, "Download");
});

test("while OCR is off the list is a call-to-action, not rows", async () => {
  const { el } = await render({ ocrImages: false });
  assert.equal(el.querySelectorAll(".ocr-lang").length, 0);
  assert.match(el.querySelector(".hint").textContent, /Turn on to recognize text/);
});

test("a failed download is reported inline, next to its button, with a retry that works", async () => {
  const { el, window } = await render();
  let fail = true;
  globalThis.__stubDownload = async () => {
    if (fail) throw new Error("offline");
    globalThis.__stubInstalled = ["eng", "deu"];
  };
  const deu = rowFor(el, "German");
  const btn = deu.querySelector("button");
  btn.dispatchEvent(new window.Event("click"));
  await new Promise((r) => setTimeout(r, 5));

  const err = el.querySelector(".ocr-error");
  assert.ok(err, "the failure is visible on the page, not only in the console");
  assert.match(err.textContent, /Couldn't download German/);
  assert.equal(err.getAttribute("role"), "alert");
  assert.equal(btn.disabled, false, "the button is back, so the row itself stays usable");

  fail = false;
  err.querySelector("button").dispatchEvent(new window.Event("click"));
  await new Promise((r) => setTimeout(r, 5));
  assert.equal(el.querySelector(".ocr-error"), null, "a retry that succeeds clears the message");
  assert.ok(rowFor(el, "German").querySelector("input[type=radio]"), "German is now selectable");
});

test("a retry removes a stale failure message before starting over", async () => {
  const { el, window } = await render();
  globalThis.__stubDownload = async () => { throw new Error("offline"); };
  const deu = rowFor(el, "German");
  deu.querySelector("button").dispatchEvent(new window.Event("click"));
  await new Promise((r) => setTimeout(r, 5));
  assert.ok(el.querySelector(".ocr-error"));
  // Pressing Download again (the row's own button) must not stack a second message.
  rowFor(el, "German").querySelector("button").dispatchEvent(new window.Event("click"));
  await new Promise((r) => setTimeout(r, 5));
  assert.equal(el.querySelectorAll(".ocr-error").length, 1);
});
