// html.js - local HTML reader. Mirrors internal/htmlconv/extract.go intent: read
// the <title>, take the <body> inner HTML, and render it as one section. The body
// is sanitized (drop script/style/on*/inline styles) since it is merged into the
// viewer's combined document.

import { sanitizeToFragment } from "./sanitize.js";
import { normalizeLangTag } from "./lang.js";

// parseHtml decodes UTF-8 bytes and returns the render-ready book shape.
export async function parseHtml(data) {
  const source = new TextDecoder("utf-8").decode(data);
  const doc = new DOMParser().parseFromString(source, "text/html");
  const titleEl = doc.querySelector("title");
  const title = titleEl ? titleEl.textContent.trim() : "";
  const lang = normalizeLangTag(doc.documentElement ? doc.documentElement.getAttribute("lang") || "" : "");
  const bodyHtml = doc.body ? doc.body.innerHTML : source;
  const { frag, label } = sanitizeToFragment(bodyHtml, 0);
  const sampleText = (frag.textContent || "").slice(0, 8000);
  return {
    title,
    lang,
    sampleText,
    sections: [{ id: "html-sec-0", label, frag }],
    toc: [],
    revoke: () => {},
  };
}
