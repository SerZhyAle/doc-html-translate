// sanitize.js - turn an untrusted HTML string into a safe, id-namespaced DOM
// fragment for the reader.
//
// New format parsers (html.js, md.js, ebook.js) produce HTML that is merged into
// the viewer's single combined document. This mirrors the sanitize half of epub.js
// renderChapter (drop unsafe/irrelevant nodes, strip on*/style, namespace ids so
// chapters can share one document) without epub.js's zip/blob/anchor coupling.
// EPUB keeps its own renderChapter; this helper serves the other formats.

// Same set epub.js drops: nothing that should reach the reader's DOM.
const DROP_TAGS = "script,style,link,base,meta,title,noscript,iframe,object,embed,form";

// sanitizeToFragment parses `html`, strips unsafe/irrelevant nodes and attributes,
// namespaces every id as `d<index>-<id>` (so multiple sections coexist in one
// document), and returns { frag, label } where label is the first h1..h4 text.
export function sanitizeToFragment(html, index) {
  const parsed = new DOMParser().parseFromString(html, "text/html");
  const body = parsed.body;
  const host = document.createElement("div");
  for (const child of Array.from(body ? body.childNodes : [])) {
    host.appendChild(document.importNode(child, true));
  }

  host.querySelectorAll(DROP_TAGS).forEach((e) => e.remove());

  // Strip event handlers and author inline styles, and namespace every id so the
  // combined single-page document has no cross-section id collisions.
  host.querySelectorAll("*").forEach((e) => {
    for (const attr of Array.from(e.attributes)) {
      const n = attr.name.toLowerCase();
      if (n.startsWith("on") || n === "style") e.removeAttribute(attr.name);
    }
    if (e.id) e.id = `d${index}-${e.id}`;
  });

  const heading = host.querySelector("h1,h2,h3,h4");
  const label = heading ? heading.textContent.replace(/\s+/g, " ").trim().slice(0, 140) : "";

  const frag = document.createDocumentFragment();
  while (host.firstChild) frag.appendChild(host.firstChild);
  return { frag, label };
}
