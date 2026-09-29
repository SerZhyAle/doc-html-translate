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
import { readerKey } from "../src/reading-position.js";

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
// storedData stands in for chrome.storage.local: a test can seed it (a saved reading
// position, ticket 53) before the viewer runs, and everything the viewer writes lands in
// it, so a later module instance reads back what an earlier one saved - one shared
// profile store, which is exactly the scope the resume feature lives in.
const storedData = {};
globalThis.chrome = {
  runtime: {
    getURL: (p) => `chrome-extension://test/${p}`,
    sendMessage: async () => {},
  },
  storage: {
    local: {
      get: async (keys) => {
        const out = {};
        for (const k of (Array.isArray(keys) ? keys : [keys])) {
          if (k in storedData) out[k] = storedData[k];
        }
        return out;
      },
      set: async (obj) => { Object.assign(storedData, obj); recorded.push(obj); },
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
    get() {
      // linkedom's own getter cannot read a selected option, so the setter records the
      // assigned value and this getter returns it.
      if (this.__value !== undefined) return this.__value;
      return get.call(this);
    },
    set(v) {
      this.__value = String(v);
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
  window.matchMedia = window.matchMedia || (() => ({ matches: false }));
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
// that interleave a second document with a first one still loading. `hash` puts a fragment on
// the viewer URL - an explicit destination (ticket 53's direct-link exception).
async function bootViewer(search, fetchDoc, { hash = "" } = {}) {
  const { document, window } = parseHTML(VIEWER_HTML);
  window.top = window;
  window.self = window;
  window.matchMedia = window.matchMedia || (() => ({ matches: false }));
  globalThis.document = document;
  globalThis.window = window;
  globalThis.DOMParser = DOMParser;
  globalThis.location = { search, href: `chrome-extension://test/src/viewer.html${search}`, hash };
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

// ---- Ticket 55: the export states what it holds ------------------------------
// Every lazy surface the export touches - comic page inflation, page-image OCR, the PDF
// chunk sentinel - rides IntersectionObservers that a real reading scroll would fire.
// linkedom has none, so this stub stands in; whether it fires on observe() is decided by
// ioAutoFire, because the comic tests want "the reader is looking at everything" while the
// PDF test wants chunk two to wait for its preparation.
class StubIO {
  constructor(cb) { this.cb = cb; }
  observe(target) { if (ioAutoFire) this.cb([{ isIntersecting: true, target }], this); }
  unobserve() {}
  disconnect() {}
}
let ioAutoFire = true;
globalThis.IntersectionObserver = globalThis.IntersectionObserver || StubIO;

// The saved file leaves through an object URL; node's own createObjectURL is replaced so
// the tests can see every download (comic page images included - they go through it too).
const savedExports = [];
URL.createObjectURL = (blob) => { savedExports.push(blob); return `blob:export-${savedExports.length}`; };
URL.revokeObjectURL = () => {};

// letOcrSettle waits out the forced comic-page recognition attempts (they fail fast under
// the vendored stub, but land as status ticks a moment after their image is restored), so a
// test's own status assertion is not overwritten mid-flight.
async function letOcrSettle(content) {
  await waitFor(() => !content.querySelector(".ocr-pending"));
  await new Promise((r) => setTimeout(r, 300));
}

// cbzBytes builds a minimal stored CBZ. comic.js's reader only walks the central directory,
// so a blank checksum is fine; a page with broken: true carries a compression method the
// reader refuses, which is how a page fails to inflate in the wild.
function cbzBytes(pages) {
  const locals = [];
  const central = [];
  let offset = 0;
  for (const p of pages) {
    const data = Buffer.from(p.data || "PG", "utf8");
    const method = p.broken ? 5 : 0;
    const lh = Buffer.alloc(30);
    lh.writeUInt32LE(0x04034b50, 0);
    lh.writeUInt16LE(20, 4);
    lh.writeUInt16LE(method, 8);
    lh.writeUInt32LE(data.length, 18);
    lh.writeUInt32LE(data.length, 22);
    lh.writeUInt16LE(p.name.length, 26);
    locals.push(lh, Buffer.from(p.name, "utf8"), data);
    const ch = Buffer.alloc(46);
    ch.writeUInt32LE(0x02014b50, 0);
    ch.writeUInt16LE(20, 4);
    ch.writeUInt16LE(20, 6);
    ch.writeUInt16LE(method, 10);
    ch.writeUInt32LE(data.length, 20);
    ch.writeUInt32LE(data.length, 24);
    ch.writeUInt16LE(p.name.length, 28);
    ch.writeUInt32LE(offset, 42);
    central.push(ch, Buffer.from(p.name, "utf8"));
    offset += 30 + p.name.length + data.length;
  }
  const cd = Buffer.concat(central);
  const eocd = Buffer.alloc(22);
  eocd.writeUInt32LE(0x06054b50, 0);
  eocd.writeUInt16LE(pages.length, 8);
  eocd.writeUInt16LE(pages.length, 10);
  eocd.writeUInt32LE(cd.length, 12);
  eocd.writeUInt32LE(offset, 16);
  return Buffer.concat([...locals, cd, eocd]);
}

// pickComic hands the archive to the toolbar's file picker the way a reader would.
async function pickComic(document, window, name, buf) {
  const input = document.getElementById("file-input");
  input.click = () => {};
  document.getElementById("btn-open").dispatchEvent(new window.Event("click"));
  const ab = buf.buffer.slice(buf.byteOffset, buf.byteOffset + buf.byteLength);
  Object.defineProperty(input, "files", { configurable: true, value: [{ name, arrayBuffer: async () => ab }] });
  await input.onchange();
}

async function waitFor(fn, ms = 5000) {
  for (let i = 0; i < Math.ceil(ms / 5); i++) {
    const v = fn();
    if (v) return v;
    await new Promise((r) => setTimeout(r, 5));
  }
  return null;
}

const comicImages = (content) => content.querySelectorAll('section[id^="comic-page-"] img').length;

test("a fully inflated comic exports at once through the short direct path", async () => {
  const { document, window, content } = await bootViewer("", async () => new Response("", { status: 404 }));
  await pickComic(document, window, "book.cbz", cbzBytes([
    { name: "page1.jpg", data: "ONE" },
    { name: "page2.jpg", data: "TWO" },
  ]));
  assert.ok(await waitFor(() => comicImages(content) === 2), "both pages inflate");
  await letOcrSettle(content);

  const before = savedExports.length;
  document.getElementById("btn-save-html").dispatchEvent(new window.Event("click"));
  // No dialog: every page is on screen, so the export goes straight to the file.
  assert.equal(document.querySelector(".export-dialog"), null);
  const blob = await waitFor(() => savedExports.length > before && savedExports[savedExports.length - 1]);
  assert.ok(blob, "the export downloaded");
  assert.match(document.getElementById("status-text").textContent, /Saved - complete, all 2 pages/);
});

test("a comic with a page that will not inflate is labeled partial before and after the export", async () => {
  const { document, window, content } = await bootViewer("", async () => new Response("", { status: 404 }));
  await pickComic(document, window, "book.cbz", cbzBytes([
    { name: "page1.jpg", data: "ONE" },
    { name: "page2.jpg", data: "TWO", broken: true },
    { name: "page3.jpg", data: "THREE" },
  ]));
  assert.ok(await waitFor(() => comicImages(content) === 2), "the sound pages inflate, the broken one leaves its box");
  await letOcrSettle(content);

  document.getElementById("btn-save-html").dispatchEvent(new window.Event("click"));
  const dialog = await waitFor(() => document.querySelector(".export-dialog"));
  assert.ok(dialog, "a partial view asks before writing anything");
  assert.match(dialog.textContent, /Partial export: the file would hold 2 of 3 pages/);
  const buttons = [...dialog.querySelectorAll("button")].map((b) => b.textContent);
  assert.ok(buttons.some((t) => t === "Prepare all 3 pages"), "preparation is offered within the budget");
  assert.ok(buttons.some((t) => t === "Export partial (2 pages)"), "the partial export keeps its explicit name");
  assert.ok(buttons.includes("Cancel"));

  // Cancel decides nothing: the dialog goes, no file is written.
  const before = savedExports.length;
  [...dialog.querySelectorAll("button")].find((b) => b.textContent === "Cancel").dispatchEvent(new window.Event("click"));
  assert.equal(document.querySelector(".export-dialog"), null);
  assert.equal(savedExports.length, before);

  // The named partial export writes exactly what is on screen and says so.
  document.getElementById("btn-save-html").dispatchEvent(new window.Event("click"));
  const again = await waitFor(() => document.querySelector(".export-dialog"));
  [...again.querySelectorAll("button")].find((b) => b.textContent === "Export partial (2 pages)").dispatchEvent(new window.Event("click"));
  await waitFor(() => savedExports.length > before);
  assert.match(document.getElementById("status-text").textContent,
    /Partial export: 2 of 3 pages - prepare the remaining pages to add them/);
});

// makePdfStub answers a PDF of `total` text pages; page `stallAt` blocks on `gate` so a
// test can hold a preparation mid-chunk and stop it there. `language` puts a /Lang value
// in the metadata and `text` replaces the page's text layer, so a test can stage the
// declaration-vs-text conflicts of ticket 76.
function makePdfStub(total, { stallAt = 0, gate, language = "", text = "Ordinary prose on a text page, plenty of it." } = {}) {
  const page = {
    getTextContent: async () => ({ items: [{ str: text }] }),
    getViewport: () => ({ width: 600, height: 800 }),
    cleanup() {},
  };
  return {
    numPages: total,
    getPage: async (n) => {
      if (stallAt && n >= stallAt) await gate;
      return page;
    },
    getMetadata: async () => ({ info: language ? { Language: language } : {} }),
    getOutline: async () => [],
    destroy() {},
  };
}

// Ticket 76: a PDF whose /Lang is its authoring template's "en-GB" over Simplified Chinese
// text must not label the page English - the contradicted declaration is dropped and the
// script heuristic decides.
test("a PDF whose /Lang contradicts its text is labelled by the text", async () => {
  globalThis.__getDocument = () => ({
    promise: Promise.resolve(makePdfStub(1, { language: "en-GB", text: "消除贫穷和饥饿，确保所有人享有尊严。大会通过了一项方案，并呼吁各国政府采取具体措施。" })),
    destroy() {},
  });
  try {
    const { document } = await bootViewer(
      "?file=https://books.test/zh-un.pdf",
      async () => new Response(new TextEncoder().encode("%PDF-1.7\n").buffer, { status: 200 }),
    );
    assert.ok(await waitFor(() => document.documentElement.lang === "zh"), "the Han text decides the language");
    assert.equal(document.querySelector('meta[http-equiv="content-language"]').content, "zh");
  } finally {
    delete globalThis.__getDocument;
  }
});

// Ticket 76: an image-only scan has no text layer and no /Lang, so the page must state no
// language at all - not the viewer's static lang="en" standing in as a false declaration.
test("a scanned PDF with no text layer and no /Lang states no language", async () => {
  globalThis.__getDocument = () => ({
    promise: Promise.resolve(makePdfStub(1, { text: "" })),
    destroy() {},
  });
  try {
    const { document, content } = await bootViewer(
      "?file=https://books.test/scan.pdf",
      async () => new Response(new TextEncoder().encode("%PDF-1.7\n").buffer, { status: 200 }),
    );
    assert.ok(await waitFor(() => content.querySelector("section")), "the page renders");
    assert.equal(document.documentElement.hasAttribute("lang"), false, "no lang is claimed");
    assert.equal(document.querySelector('meta[http-equiv="content-language"]'), null);
  } finally {
    delete globalThis.__getDocument;
  }
});

test("a chunk-rendered PDF offers to prepare the rest, and preparing saves a complete file", async () => {
  ioAutoFire = false; // chunk two must wait for the preparation, not for the stubbed scroll
  try {
    const pdf = makePdfStub(150);
    globalThis.__getDocument = () => ({ promise: Promise.resolve(pdf), destroy() {} });
    const { document, window, content } = await bootViewer(
      "?file=https://books.test/big.pdf",
      async () => new Response(new TextEncoder().encode("%PDF-1.7\n").buffer, { status: 200 }),
    );
    try {
      // reportRenderIdle's line is the honest "chunk landed" signal - the sections stream
      // into the DOM a macrotask before pdfRendered catches up, so counting them races.
      assert.ok(await waitFor(() => /Pages 1-100 of 150/.test(document.getElementById("status-text").textContent)), "the first chunk renders");
      assert.equal(document.getElementById("page-total").textContent, "/ 150");

      document.getElementById("btn-save-html").dispatchEvent(new window.Event("click"));
      const dialog = await waitFor(() => document.querySelector(".export-dialog"));
      assert.ok(dialog, "100 of 150 pages is partial, and the reader is told before any file");
      assert.match(dialog.textContent, /pages 1-100/);
      assert.match(dialog.textContent, /150/);

      // Prepare: the remaining chunks render right here, then the complete file saves itself.
      [...dialog.querySelectorAll("button")].find((b) => b.textContent === "Prepare all 150 pages")
        .dispatchEvent(new window.Event("click"));
      const done = await waitFor(() => /Saved - complete, all 150 pages/.test(document.getElementById("status-text").textContent), 20000);
      assert.ok(done, "the preparation ends in a complete save");
      assert.equal(content.querySelectorAll("section").length, 150, "every declared page is in the artifact");
    } finally {
      delete globalThis.__getDocument;
    }
  } finally {
    ioAutoFire = true;
  }
});

test("a stopped preparation writes nothing, labels no file, and leaves the reader usable", async () => {
  ioAutoFire = false;
  try {
    let release;
    const gate = new Promise((r) => { release = r; });
    const pdf = makePdfStub(150, { stallAt: 130, gate });
    globalThis.__getDocument = () => ({ promise: Promise.resolve(pdf), destroy() {} });
    const { document, window, content } = await bootViewer(
      "?file=https://books.test/stall.pdf",
      async () => new Response(new TextEncoder().encode("%PDF-1.7\n").buffer, { status: 200 }),
    );
    try {
      assert.ok(await waitFor(() => /Pages 1-100 of 150/.test(document.getElementById("status-text").textContent)));
      document.getElementById("btn-save-html").dispatchEvent(new window.Event("click"));
      const dialog = await waitFor(() => document.querySelector(".export-dialog"));
      [...dialog.querySelectorAll("button")].find((b) => b.textContent === "Prepare all 150 pages")
        .dispatchEvent(new window.Event("click"));

      // Hold the chunk mid-render and stop the preparation there, then let the pages through.
      // The chunk in flight always finishes - only the gaps between pages are interruptible -
      // so this lands after it completes; the outcome is still "stopped, nothing written".
      assert.ok(await waitFor(() => !document.getElementById("status-stop").hidden), "the status bar carries a Stop control");
      const before = savedExports.length;
      document.getElementById("status-stop").dispatchEvent(new window.Event("click"));
      release();
      for (let i = 0; i < 10; i++) {
        await new Promise((r) => setTimeout(r, 100));
        console.log("POLL", i, "sections:", content.querySelectorAll("section").length,
          "status:", JSON.stringify(document.getElementById("status-text").textContent),
          "stopHidden:", document.getElementById("status-stop").hidden);
      }
      const stopped = await waitFor(() => /Preparation stopped - nothing was saved/.test(document.getElementById("status-text").textContent), 15000);
      assert.ok(stopped, "the stop is named, and no completeness is claimed");
      assert.equal(savedExports.length, before, "no file was written by the stopped preparation");
      assert.equal(content.querySelectorAll("section").length, 150, "the reader keeps what rendered");
      assert.equal(document.querySelector(".export-dialog"), null, "a fully rendered document needs no partial decision");

      // Exporting again takes the direct path - the stop lost nothing.
      document.getElementById("btn-save-html").dispatchEvent(new window.Event("click"));
      const saved = await waitFor(() => /Saved - complete, all 150 pages/.test(document.getElementById("status-text").textContent), 15000);
      assert.ok(saved, "the follow-up export goes straight to the file and says complete");
    } finally {
      delete globalThis.__getDocument;
    }
  } finally {
    ioAutoFire = true;
  }
});

// ---- Ticket 53: resume reading ------------------------------------------------
// A one-paragraph text renders exactly one section (txt.js PARAS_PER_SECTION = 30), so the
// resume key of a picked file is computable here: name, byte size, title (the file name
// without its extension) and the section count - the four inputs the viewer binds in
// setPageTotal, the same four the desktop's ReaderKey hashes.
const BOOK_TEXT = "It was a bright cold day in April, and the clocks were striking thirteen.\n";

function seedPosition(name, pos) {
  const bytes = new TextEncoder().encode(BOOK_TEXT);
  const key = readerKey(name, bytes.length, name.replace(/\.txt$/, ""), 1);
  storedData.readingPositions = {
    ...(storedData.readingPositions || {}),
    [key]: { page: 1, frag: "", off: 800, secoff: 800, at: 1, ...pos },
  };
  return key;
}

const resumeButtons = (bar) => [...bar.querySelectorAll("button")].map((b) => b.textContent.trim());

test("scrolling a document saves a bounded reading position locally", async () => {
  const { document, window } = await bootViewer("", async () => new Response("", { status: 404 }));
  window.innerHeight = 900;
  await pickLocalFile(document, window, "book.txt", BOOK_TEXT);
  assert.match(document.getElementById("content").textContent, /bright cold day/);
  assert.equal(document.querySelector(".resume-notice"), null, "a fresh document offers nothing");
  assert.equal(storedData.readingPositions, undefined, "nothing is written before the reader moves");

  window.scrollTo = () => {};
  document.dispatchEvent(new window.Event("scroll"));
  const stored = await waitFor(() => {
    const map = storedData.readingPositions;
    return map ? map[Object.keys(map)[0]] : null;
  });
  assert.ok(stored, "the throttled save landed in storage");
  assert.equal(stored.page, 1);
  assert.equal(stored.frag, "", "a text page has no inner anchors, the section offset carries it");
  // linkedom lays nothing out, so every offset reads 0 here; the anchor/offset math itself is
  // asserted with real numbers in reading-position.test.mjs.
  assert.equal(typeof stored.secoff, "number");
});

test("reopening the same document offers Continue reading, which jumps to the saved place", async () => {
  seedPosition("book.txt");
  const { document, window } = await bootViewer("", async () => new Response("", { status: 404 }));
  await pickLocalFile(document, window, "book.txt", BOOK_TEXT);
  const bar = await waitFor(() => document.querySelector(".resume-notice"));
  assert.ok(bar, "the saved place is offered on a plain reopen");
  assert.match(bar.textContent, /Continue reading from page 1\?/);
  const buttons = resumeButtons(bar);
  assert.ok(buttons.includes("Continue reading"));
  assert.ok(buttons.includes("Start over"));

  const jumps = [];
  window.scrollTo = (x, y) => jumps.push(y);
  [...bar.querySelectorAll("button")].find((b) => b.textContent.trim() === "Continue reading")
    .dispatchEvent(new window.Event("click"));
  assert.ok(await waitFor(() => jumps.length > 0), "Continue scrolls to the saved place");
  assert.equal(document.querySelector(".resume-notice"), null, "the offer is spent once acted on");
});

test("Start over clears the saved position for that document only", async () => {
  const key = seedPosition("book.txt");
  seedPosition("other.txt", { page: 1, frag: "", off: 5, secoff: 5 });
  const { document, window } = await bootViewer("", async () => new Response("", { status: 404 }));
  await pickLocalFile(document, window, "book.txt", BOOK_TEXT);
  const bar = await waitFor(() => document.querySelector(".resume-notice"));
  window.scrollTo = () => {};
  [...bar.querySelectorAll("button")].find((b) => b.textContent.trim() === "Start over")
    .dispatchEvent(new window.Event("click"));
  assert.ok(await waitFor(() => !document.querySelector(".resume-notice")));
  const cleared = await waitFor(() => storedData.readingPositions[key] === undefined);
  assert.ok(cleared, "the entry is gone from storage");
  assert.ok(storedData.readingPositions[Object.keys(storedData.readingPositions).find((k) => k !== key)],
    "another document's position is untouched");
});

test("a direct link to a section wins over the saved position", async () => {
  seedPosition("book.txt");
  const { document, window } = await bootViewer("", async () => new Response("", { status: 404 }), { hash: "#txt-0" });
  await pickLocalFile(document, window, "book.txt", BOOK_TEXT);
  for (let i = 0; i < 20; i++) await new Promise((r) => setTimeout(r, 5));
  assert.equal(document.querySelector(".resume-notice"), null, "an explicit destination gets no offer");
});

test("an unrelated file with the same title does not reuse the position", async () => {
  seedPosition("book.txt");
  const { document, window } = await bootViewer("", async () => new Response("", { status: 404 }));
  // Same displayed title ("book"), different bytes: the size differs, so the identity differs.
  await pickLocalFile(document, window, "book.txt", "A different little book entirely, of a different byte size.\n");
  for (let i = 0; i < 20; i++) await new Promise((r) => setTimeout(r, 5));
  assert.equal(document.querySelector(".resume-notice"), null, "no offer for a document this key never saw");
});

test("a saved place past the rendered edge is still offered for a chunked PDF", async () => {
  ioAutoFire = false; // chunk two must wait for the reader's Continue, not for the stubbed scroll
  try {
    // The URL document is 9 bytes ("%PDF-1.7\n"), titled "long", 150 pages.
    seedPositionKey(readerKey("long.pdf", 9, "long", 150), { page: 120, frag: "", off: 400, secoff: 400 });
    globalThis.__getDocument = () => ({ promise: Promise.resolve(makePdfStub(150)), destroy() {} });
    const { document, window } = await bootViewer(
      "?file=https://books.test/long.pdf",
      async () => new Response(new TextEncoder().encode("%PDF-1.7\n").buffer, { status: 200 }),
    );
    try {
      assert.ok(await waitFor(() => /Pages 1-100 of 150/.test(document.getElementById("status-text").textContent)),
        "the first chunk renders");
      const bar = document.querySelector(".resume-notice");
      assert.ok(bar, "page 120 is offered before it is rendered");
      assert.match(bar.textContent, /page 120\?/);

      const jumps = [];
      window.scrollTo = (x, y) => jumps.push(y);
      [...bar.querySelectorAll("button")].find((b) => b.textContent.trim() === "Continue reading")
        .dispatchEvent(new window.Event("click"));
      assert.ok(await waitFor(() => jumps.length > 0, 20000), "the jump renders forward first");
      assert.equal(document.querySelectorAll("#content section").length, 150, "the jump rendered the rest");
    } finally {
      delete globalThis.__getDocument;
    }
  } finally {
    ioAutoFire = true;
  }
});

function seedPositionKey(key, pos) {
  storedData.readingPositions = { [key]: { at: 1, ...pos } };
}

test("reading-comfort choices persist and a reopen applies them (ticket 59)", async () => {
  const body = "Chapter One\n\nIt was a bright cold day in April.\n\nThe clocks were striking thirteen.\n";
  const first = await openViewer(
    "?file=https%3A%2F%2Fbooks.test%2Flib%2Fcomfort.txt",
    async () => new Response(body, { status: 200 }),
  );
  const w1 = globalThis.window;
  // Pick a line spacing, a column width and a night theme, and nudge the size off default.
  const leading = first.document.getElementById("sel-leading");
  leading.value = "1.9";
  leading.dispatchEvent(new w1.Event("change"));
  const width = first.document.getElementById("sel-width");
  width.value = "64em";
  width.dispatchEvent(new w1.Event("change"));
  first.document.getElementById("btn-night").dispatchEvent(new w1.Event("click"));
  first.document.getElementById("btn-font-inc").dispatchEvent(new w1.Event("click"));
  assert.ok(await waitFor(() => storedData.viewerPrefs && storedData.viewerPrefs.leading === "1.9"),
    "the spacing choice is saved");
  assert.equal(storedData.viewerPrefs.width, "64em");
  assert.equal(storedData.viewerPrefs.theme, "night", "the toggle switches to the night family");
  assert.equal(storedData.viewerPrefs.nightTheme, "night", "the family's last theme is remembered");
  assert.equal(storedData.viewerPrefs.size, 29);
  assert.equal(first.document.documentElement.getAttribute("data-theme"), "night");

  // A fresh module instance is the close-and-reopen: same profile store, choices applied.
  const second = await openViewer(
    "?file=https%3A%2F%2Fbooks.test%2Flib%2Fcomfort2.txt",
    async () => new Response(body, { status: 200 }),
  );
  assert.equal(second.document.getElementById("sel-leading").value, "1.9");
  assert.equal(second.document.getElementById("sel-width").value, "64em");
  assert.equal(second.document.documentElement.getAttribute("data-theme"), "night");
  assert.equal(second.document.getElementById("btn-night").getAttribute("aria-pressed"), "true",
    "pressed names the night family the reader is in");

  // The reset returns the shipped size; Default hands the measures back to the shipped ones.
  const w2 = globalThis.window;
  second.document.getElementById("btn-font-reset").dispatchEvent(new w2.Event("click"));
  assert.ok(await waitFor(() => storedData.viewerPrefs.size === 28), "the size reset lands in storage");
  const width2 = second.document.getElementById("sel-width");
  width2.value = "";
  width2.dispatchEvent(new w2.Event("change"));
  assert.ok(await waitFor(() => storedData.viewerPrefs.width === null), "Default clears the width choice");
});
