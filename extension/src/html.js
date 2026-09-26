// html.js - local HTML reader. Mirrors internal/htmlconv/extract.go intent: read
// the <title>, take the <body> inner HTML, and render it as one section. The body
// is sanitized (drop script/style/on*/inline styles) since it is merged into the
// viewer's combined document.

import { sanitizeToFragment } from "./sanitize.js";
import { normalizeLangTag } from "./lang.js";
import { decodeHtml } from "./charset.js";

// declaredLang is the language the page declares, asked where internal/htmlconv rootAttrs asks:
// <html> first, then <body> - only the body's content is kept, so its declaration is lifted -
// each by lang and then xml:lang, a blank value counting as none. The desktop edition copies the
// tag as found; here it is normalized, because the viewer compares it with its own detection.
function declaredLang(doc) {
  for (const el of [doc.documentElement, doc.body]) {
    if (!el) continue;
    for (const key of ["lang", "xml:lang"]) {
      const v = (el.getAttribute(key) || "").trim();
      if (v) return normalizeLangTag(v);
    }
  }
  return "";
}

// parseHtml decodes the bytes by their declared or detected encoding (charset.js decodeHtml)
// and returns the render-ready book shape.
export async function parseHtml(data) {
  const source = decodeHtml(data);
  const doc = new DOMParser().parseFromString(source, "text/html");
  const titleEl = doc.querySelector("title");
  const title = titleEl ? titleEl.textContent.trim() : "";
  const lang = declaredLang(doc);
  const bodyHtml = doc.body ? doc.body.innerHTML : source;
  const { frag, label, remote } = sanitizeToFragment(bodyHtml, 0);
  const sampleText = (frag.textContent || "").slice(0, 8000);
  return {
    title,
    lang,
    sampleText,
    sections: [{ id: "html-sec-0", label, frag }],
    toc: [],
    remote,
    revoke: () => {},
  };
}
