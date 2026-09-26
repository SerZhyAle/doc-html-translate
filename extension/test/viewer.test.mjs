// Behaviour tests for viewer.js - the reflow viewer page. The module has no exports: on import it
// reads `?file=` from its own URL, decides whether it may load it, fetches it and renders it into
// viewer.html. So each case builds a fresh viewer.html document, sets the page URL and the network,
// imports a fresh copy of the module and reads what the reader would see.
//
// viewer.js statically imports the vendored libraries (pdf.js, foliate, marked, tesseract), which
// `npm run vendor` fetches and which may be absent. None of them is on the paths tested here, so a
// module hook swaps every ../vendor/ import for an inert stub instead of depending on them.

import { test } from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { register } from "node:module";
import { parseHTML, DOMParser } from "linkedom";

const VENDOR_STUB = `
export const GlobalWorkerOptions = {};
export const PasswordResponses = {};
export function getDocument(src) {
  if (globalThis.__getDocument) return globalThis.__getDocument(src);
  throw new Error("pdf.js is stubbed in this test");
}
export class MOBI {}
export function unzlibSync() { throw new Error("fflate is stubbed in this test"); }
export const marked = { parse: () => "", use: () => {} };
export default {};
`;
const HOOKS = `
export async function resolve(specifier, context, next) {
  if (/(^|\\/)vendor\\//.test(specifier)) {
    return { url: "data:text/javascript," + encodeURIComponent(${JSON.stringify(VENDOR_STUB)}), shortCircuit: true };
  }
  return next(specifier, context);
}
`;
register(`data:text/javascript,${encodeURIComponent(HOOKS)}`, import.meta.url);

const root = path.join(import.meta.dirname, "..");
// The page's own markup, minus its module script: the test imports the module itself.
const VIEWER_HTML = fs.readFileSync(path.join(root, "src", "viewer.html"), "utf8")
  .replace(/<script[^>]*src="viewer\.js"[^>]*><\/script>/, "");

const recorded = [];
globalThis.chrome = {
  runtime: {
    getURL: (p) => `chrome-extension://test/${p}`,
    sendMessage: async () => {},
  },
  storage: {
    local: {
      get: async () => ({}),
      set: async (obj) => { recorded.push(obj); },
    },
  },
  tabs: { getCurrent: async () => ({ id: 1 }) },
  i18n: { getMessage: () => "", getUILanguage: () => "en" },
};
globalThis.requestAnimationFrame = (fn) => setTimeout(fn, 0);

// Two linkedom gaps the page setup runs into, filled with the browser's behaviour: a <select>'s
// `value` has no setter (applyPrefs assigns it), and a DocumentFragment's textContent is null
// (renderBook counts a section's characters from it, and would otherwise report every book as
// having no text).
{
  const { document } = parseHTML("<!doctype html><html><body><select></select></body></html>");
  const selectProto = Object.getPrototypeOf(document.querySelector("select"));
  const get = Object.getOwnPropertyDescriptor(selectProto, "value").get;
  Object.defineProperty(selectProto, "value", {
    configurable: true,
    get,
    set(v) {
      for (const o of this.querySelectorAll("option")) o.selected = o.getAttribute("value") === String(v);
    },
  });
  const fragProto = Object.getPrototypeOf(document.createDocumentFragment());
  Object.defineProperty(fragProto, "textContent", {
    configurable: true,
    get() { return [...this.childNodes].map((n) => n.textContent || "").join(""); },
  });
}

let caseSeq = 0;

// openViewer loads a fresh viewer for `search` with `network` standing in for fetch(), and resolves
// once the page has settled on a notice or on rendered sections. `framed` puts the page inside
// another window, as an embedding site would.
async function openViewer(search, network, { framed = false } = {}) {
  const { document, window } = parseHTML(VIEWER_HTML);
  window.top = framed ? {} : window;
  window.self = window;
  globalThis.document = document;
  globalThis.window = window;
  globalThis.location = { search, href: `chrome-extension://test/src/viewer.html${search}` };
  const fetched = [];
  globalThis.fetch = async (url) => {
    fetched.push(String(url));
    // The interface-language table is optional; a miss leaves the English fallbacks in place.
    if (String(url).startsWith("chrome-extension://")) return new Response("", { status: 404 });
    return network(String(url));
  };
  // A query string makes a new module instance, so every case runs main() against its own page.
  await import(`../src/viewer.js?case=${++caseSeq}`);
  const content = document.getElementById("content");
  for (let i = 0; i < 200 && !content.querySelector(".notice, section"); i++) {
    await new Promise((r) => setTimeout(r, 5));
  }
  const pageFetches = fetched.filter((u) => !u.startsWith("chrome-extension://"));
  return { document, content, pageFetches };
}

test("a text document from a URL is fetched and rendered as translatable sections", async () => {
  const body = "Chapter One\n\nIt was a bright cold day in April.\n\nThe clocks were striking thirteen.\n";
  const { document, content, pageFetches } = await openViewer(
    "?file=https%3A%2F%2Fbooks.test%2Flib%2F1984.txt",
    async () => new Response(body, { status: 200 }),
  );

  // The manual entry point percent-encodes the URL; the viewer has to decode it before fetching.
  assert.deepEqual(pageFetches, ["https://books.test/lib/1984.txt"]);
  // Asserted as booleans: a failing deepEqual on a linkedom node tries to print the whole tree.
  assert.ok(!content.querySelector(".notice"), content.innerHTML);
  const text = content.textContent;
  assert.match(text, /bright cold day in April/);
  assert.match(text, /striking thirteen/);
  // The title comes from the file name, without its extension.
  assert.equal(document.getElementById("doc-title").textContent, "1984");
  assert.equal(document.title, "1984");
  // Detected language on <html> is what makes the browser offer to translate the page.
  assert.equal(document.documentElement.lang, "en");
  // A non-PDF has no "original PDF" to fall back to, and a rendered book can be saved as HTML.
  assert.ok(document.getElementById("btn-original").classList.contains("hidden"));
  assert.ok(!document.getElementById("btn-save-html").classList.contains("hidden"));
  assert.match(document.getElementById("page-total").textContent, /^\/ \d+$/);
});

test("a download that fails shows the reason and offers the file picker", async () => {
  const { content } = await openViewer(
    "?file=https://books.test/missing.pdf",
    async () => new Response("gone", { status: 404 }),
  );

  const notice = content.querySelector(".notice");
  assert.ok(notice, "a failed download must end on a notice, not a blank page");
  assert.equal(notice.querySelector("h1").textContent, "Couldn't load this document");
  assert.match(notice.textContent, /Reason: HTTP 404/);
  assert.ok([...notice.querySelectorAll("button")].some((b) => b.textContent === "Open a document"));
  assert.ok(!content.querySelector("section"));
  // The failure is what a diagnostics report has to carry.
  assert.ok(recorded.some((r) => r.lastRun && r.lastRun.error === "Couldn't load this document"),
    JSON.stringify(recorded));
});

test("a non-document URL is refused before anything is fetched", async () => {
  let fetchedDoc = false;
  const { content, pageFetches } = await openViewer(
    "?file=javascript%3Aalert(1)",
    async () => { fetchedDoc = true; return new Response(""); },
  );

  assert.equal(fetchedDoc, false);
  assert.deepEqual(pageFetches, []);
  assert.equal(content.querySelector(".notice h1").textContent, "Unsupported URL");
});

test("a viewer loaded inside a frame refuses to run", async () => {
  let fetchedDoc = false;
  const { content, pageFetches } = await openViewer(
    "?file=https://books.test/a.txt",
    async () => { fetchedDoc = true; return new Response("text"); },
    { framed: true },
  );

  // A page that frames the viewer must not get it to fetch a URL with the extension's host access.
  assert.equal(content.querySelector(".notice h1").textContent, "Cannot run in a frame");
  assert.equal(fetchedDoc, false);
  assert.deepEqual(pageFetches, []);
});

test("a local file picked during a slow URL load is the document that stays shown", async () => {
  let release;
  const slow = new Promise((r) => { release = r; });
  const { document, window } = parseHTML(VIEWER_HTML);
  window.top = window;
  window.self = window;
  globalThis.document = document;
  globalThis.window = window;
  globalThis.location = { search: "?file=https://books.test/slow.txt", href: "chrome-extension://test/src/viewer.html" };
  globalThis.fetch = async (url) => {
    if (String(url).startsWith("chrome-extension://")) return new Response("", { status: 404 });
    await slow;
    return new Response("The remote book that arrived too late.\n", { status: 200 });
  };
  await import(`../src/viewer.js?case=${++caseSeq}`);
  const content = document.getElementById("content");
  // Let main() reach the pending download.
  for (let i = 0; i < 20; i++) await new Promise((r) => setTimeout(r, 5));

  // The reader picks a local file through the toolbar while the download is still pending.
  const input = document.getElementById("file-input");
  input.click = () => {};
  document.getElementById("btn-open").dispatchEvent(new window.Event("click"));
  const bytes = new TextEncoder().encode("The local book the reader chose.\n");
  Object.defineProperty(input, "files", { configurable: true, value: [{ name: "local.txt", arrayBuffer: async () => bytes.buffer }] });
  await input.onchange();
  assert.match(content.textContent, /local book the reader chose/);

  // The stale download finishes afterwards; it must neither replace nor join the local document.
  release();
  for (let i = 0; i < 20; i++) await new Promise((r) => setTimeout(r, 5));
  assert.match(content.textContent, /local book the reader chose/);
  assert.doesNotMatch(content.textContent, /arrived too late/);
  assert.equal(document.getElementById("doc-title").textContent, "local");
});

// bootViewer loads a fresh viewer for `search` without waiting for it to settle, for the cases
// that interleave a second document with a first one still loading.
async function bootViewer(search, fetchDoc) {
  const { document, window } = parseHTML(VIEWER_HTML);
  window.top = window;
  window.self = window;
  globalThis.document = document;
  globalThis.window = window;
  globalThis.DOMParser = DOMParser;
  globalThis.location = { search, href: `chrome-extension://test/src/viewer.html${search}` };
  globalThis.fetch = async (url) => {
    if (String(url).startsWith("chrome-extension://")) return new Response("", { status: 404 });
    return fetchDoc(String(url));
  };
  await import(`../src/viewer.js?case=${++caseSeq}`);
  for (let i = 0; i < 20; i++) await new Promise((r) => setTimeout(r, 5));
  return { document, window, content: document.getElementById("content") };
}

// pickLocalFile opens `text` as a local file named `name` through the toolbar's file picker.
async function pickLocalFile(document, window, name, text) {
  const input = document.getElementById("file-input");
  input.click = () => {};
  document.getElementById("btn-open").dispatchEvent(new window.Event("click"));
  const bytes = new TextEncoder().encode(text);
  Object.defineProperty(input, "files", { configurable: true, value: [{ name, arrayBuffer: async () => bytes.buffer }] });
  await input.onchange();
}

// B39: renderDocument checked the load token once, after the TOC; a PDF superseded while its
// metadata was still being read went on to set its own language on the newer document.
test("a PDF superseded while its language is being read leaves the newer document's language alone", async () => {
  let releaseMeta;
  const meta = new Promise((r) => { releaseMeta = r; });
  let destroyed = false;
  const page = {
    getTextContent: async () => ({ items: [{ str: "Ein Satz auf Deutsch" }] }),
    getViewport: () => ({ width: 600, height: 800 }),
    cleanup() {},
  };
  const pdf = {
    numPages: 1,
    getPage: async () => page,
    getMetadata: () => meta,
    getOutline: async () => [],
    destroy() { destroyed = true; },
  };
  globalThis.__getDocument = () => ({ promise: Promise.resolve(pdf), destroy() {} });
  try {
    const pdfBytes = new TextEncoder().encode("%PDF-1.7\n");
    const { document, window, content } = await bootViewer(
      "?file=https://books.test/slow.pdf",
      async () => new Response(pdfBytes, { status: 200 }),
    );
    // The PDF load now waits on its metadata; the reader opens an English text meanwhile.
    await pickLocalFile(document, window, "local.txt", "It was a bright cold day in April, and the clocks were striking thirteen.\n");
    assert.equal(document.documentElement.lang, "en");

    releaseMeta({ info: { Language: "de" } });
    for (let i = 0; i < 20; i++) await new Promise((r) => setTimeout(r, 5));
    assert.equal(document.documentElement.lang, "en", "the stale PDF must not relabel the text's language");
    assert.equal(document.querySelector('meta[http-equiv="content-language"]').content, "en");
    assert.ok(destroyed, "the superseded PDF is released");
    assert.match(content.textContent, /bright cold day/);
  } finally {
    delete globalThis.__getDocument;
  }
});

// B41: the TOC button was hidden for a document without a TOC and never shown again, and the
// OCR layer control, once revealed, stayed on screen for every later document.
test("toolbar state is reset for each document", async () => {
  const fb2 = `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0"><body>
<section><title><p>Chapter One</p></title><p>It was a bright cold day in April.</p></section>
<section><title><p>Chapter Two</p></title><p>The clocks were striking thirteen.</p></section>
</body></FictionBook>`;
  const { document, window } = await bootViewer("", async () => new Response("", { status: 404 }));
  const tocBtn = document.getElementById("btn-toc");
  const ocrGroup = document.getElementById("grp-ocr");

  await pickLocalFile(document, window, "plain.txt", "Just a line of prose without chapters.\n");
  assert.ok(tocBtn.classList.contains("hidden"), "a document without a TOC hides the button");
  // The previous document's plates revealed the OCR layer control.
  ocrGroup.hidden = false;

  await pickLocalFile(document, window, "book.fb2", fb2);
  assert.match(document.getElementById("content").textContent, /Chapter Two/);
  assert.ok(!tocBtn.classList.contains("hidden"), "a document with a TOC shows the button again");
  assert.ok(document.querySelectorAll("#toc-tree a").length >= 2, "and its entries");
  assert.equal(ocrGroup.hidden, true, "no plates in this document, so no OCR layer control");

  await pickLocalFile(document, window, "plain.txt", "Another line of prose.\n");
  assert.ok(tocBtn.classList.contains("hidden"));
  assert.equal(document.querySelectorAll("#toc-tree a").length, 0, "no stale entries from the last book");
});
