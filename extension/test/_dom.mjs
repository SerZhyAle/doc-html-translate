// Shared DOM shim for the extension's DOM-path unit tests.
//
// The chapter sanitize/image/link/TOC helpers in epub.js and sanitize.js run
// `new DOMParser()` and touch a global `document` (in the browser, the viewer's
// page). node has no DOM, so these paths were previously only covered by manual
// acceptance gates. linkedom is a small, pure-JS DOM used here (dev-only) to drive
// them under `node --test`. It is NOT bundled into the extension.
//
// Underscore-prefixed so the `test/*.test.mjs` runner glob ignores this file.
//
// Caveat: linkedom's DOMParser does not synthesize a missing <html>/<body> the way
// a browser's HTML parser does, so tests feed full documents. Use `fragHtml` to read
// a DocumentFragment's markup - linkedom returns null for `fragment.textContent`.

import { DOMParser, parseHTML } from "linkedom";

const { document } = parseHTML("<!doctype html><html><body></body></html>");
globalThis.document = document;
globalThis.DOMParser = DOMParser;

// fragHtml serializes a DocumentFragment by moving its children into a detached
// <div> and reading innerHTML (linkedom fragments have no innerHTML of their own).
export function fragHtml(frag) {
  const div = document.createElement("div");
  div.appendChild(frag.cloneNode ? frag.cloneNode(true) : frag);
  return div.innerHTML;
}

// fragText returns the text content of a DocumentFragment (worked around the same
// way - linkedom returns null for fragment.textContent directly).
export function fragText(frag) {
  const div = document.createElement("div");
  div.appendChild(frag.cloneNode ? frag.cloneNode(true) : frag);
  return div.textContent || "";
}

// el builds a fixture inside the global document and returns the first element
// matching `selector` - a convenience for the element-level rewriters (rewriteImg /
// convertSvgImage / rewriteAnchor). Nodes live in the global document so that
// convertSvgImage's `document.createElement("img")` + replaceWith stays a
// same-document operation, exactly as it is in the viewer after renderChapter has
// imported the chapter body.
export function el(html, selector) {
  const div = document.createElement("div");
  div.innerHTML = html;
  return div.querySelector(selector);
}
