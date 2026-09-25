// sanitize.js - turn an untrusted HTML string into a safe, id-namespaced DOM
// fragment for the reader.
//
// New format parsers (html.js, md.js, ebook.js) produce HTML that is merged into
// the viewer's single combined document. This mirrors the sanitize half of epub.js
// renderChapter (drop unsafe/irrelevant nodes, strip on*/style, namespace ids so
// chapters can share one document) without epub.js's zip/blob/anchor coupling.
// EPUB keeps its own renderChapter; this helper serves the other formats.

import { DROP_TAGS, scrubTree } from "./url-policy.js";

// sanitizeToFragment parses `html`, strips unsafe/irrelevant nodes and attributes,
// namespaces every id as `d<index>-<id>` (so multiple sections coexist in one
// document) with same-section "#x" links retargeted to match, and returns
// { frag, label, remote } where label is the first h1..h4 text and remote counts the
// elements whose remote URLs are parked (url-policy.js).
export function sanitizeToFragment(html, index) {
  const parsed = new DOMParser().parseFromString(html, "text/html");
  const body = parsed.body;
  const host = document.createElement("div");
  for (const child of Array.from(body ? body.childNodes : [])) {
    host.appendChild(document.importNode(child, true));
  }

  host.querySelectorAll(DROP_TAGS).forEach((e) => e.remove());

  const remote = scrubTree(host, index);

  const heading = host.querySelector("h1,h2,h3,h4");
  const label = heading ? heading.textContent.replace(/\s+/g, " ").trim().slice(0, 140) : "";

  const frag = document.createDocumentFragment();
  while (host.firstChild) frag.appendChild(host.firstChild);
  return { frag, label, remote };
}
