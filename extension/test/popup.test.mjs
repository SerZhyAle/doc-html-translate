// Behaviour tests for popup.js - the toolbar popup. Like the viewer, the module has no exports: on
// import it localizes popup.html, reads the active tab and wires the switches. Each case builds a
// fresh popup document, sets the active tab, imports a fresh copy of the module and reads what the
// reader would see. The vendored OCR engine is swapped for an inert stub, as in viewer.test.mjs.

import { test } from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { register } from "node:module";
import { parseHTML } from "linkedom";

const HOOKS = `
export async function resolve(specifier, context, next) {
  if (/(^|\\/)vendor\\//.test(specifier)) {
    return { url: "data:text/javascript," + encodeURIComponent("export default {};"), shortCircuit: true };
  }
  return next(specifier, context);
}
`;
register(`data:text/javascript,${encodeURIComponent(HOOKS)}`, import.meta.url);

const root = path.join(import.meta.dirname, "..");
const POPUP_HTML = fs.readFileSync(path.join(root, "src", "popup.html"), "utf8")
  .replace(/<script[^>]*src="popup\.js"[^>]*><\/script>/, "");

let caseSeq = 0;

// openPopup renders the popup for an active tab at tabUrl, with `options` in storage, and returns
// the page plus the options the popup wrote back.
async function openPopup(tabUrl, options) {
  const { document, window } = parseHTML(POPUP_HTML);
  globalThis.document = document;
  globalThis.window = window;
  const store = { options: { ...options } };
  globalThis.chrome = {
    runtime: {
      getURL: (p) => `chrome-extension://test/${p}`,
      getManifest: () => ({ version: "1.0" }),
      openOptionsPage: () => {},
    },
    storage: {
      local: {
        get: async (keys) => {
          const out = {};
          for (const k of [].concat(keys)) if (k in store) out[k] = store[k];
          return out;
        },
        set: async (obj) => { Object.assign(store, obj); },
      },
    },
    tabs: { query: async () => [{ url: tabUrl }], create: () => {} },
    i18n: { getMessage: () => "", getUILanguage: () => "en" },
  };
  globalThis.fetch = async () => new Response("", { status: 404 });
  await import(`../src/popup.js?case=${++caseSeq}`);
  for (let i = 0; i < 20; i++) await new Promise((r) => setTimeout(r, 5));
  return { document, window, store };
}

// B58: the label's i18n replaced its whole content, detaching the host span, so the popup never
// said which site its switch was about.
test("the popup names the site its switch acts on", async () => {
  const { document } = await openPopup("https://www.books.test/shelf/", { enabledByDefault: true, disabledHosts: [] });
  const host = document.getElementById("host");
  assert.ok(host, "the host element survives localization");
  assert.equal(host.textContent, "www.books.test");
  assert.ok(document.querySelector('label[for="site"]').contains(host), "shown inside the switch's label");
  assert.match(document.querySelector('label[for="site"]').textContent, /On this site/);

  const none = await openPopup("chrome://newtab/", { enabledByDefault: true, disabledHosts: [] });
  assert.equal(none.document.getElementById("host").textContent, "(not a website)");
  assert.equal(none.document.getElementById("site").disabled, true);
});

// B59: on the viewer's own tab the "site" was the extension's id, so switching it off did nothing.
test("on the viewer tab the switch acts on the shown document's site", async () => {
  const { document, window, store } = await openPopup(
    "chrome-extension://test/src/viewer.html?file=https://cdn.books.test/lib/a.pdf?dl=1&x=2",
    { enabledByDefault: true, disabledHosts: [] },
  );
  assert.equal(document.getElementById("host").textContent, "cdn.books.test");
  const site = document.getElementById("site");
  assert.equal(site.disabled, false);
  site.checked = false;
  site.dispatchEvent(new window.Event("change"));
  for (let i = 0; i < 10; i++) await new Promise((r) => setTimeout(r, 5));
  assert.deepEqual(store.options.disabledHosts, ["cdn.books.test"]);

  // A local document in the viewer is on no website.
  const local = await openPopup("chrome-extension://test/src/viewer.html?file=file:///C:/books/a.pdf", { enabledByDefault: true, disabledHosts: [] });
  assert.equal(local.document.getElementById("site").disabled, true);
});
