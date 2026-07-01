// md.js - Markdown reader. Converts Markdown to HTML with the vendored `marked`,
// sanitizes it, and splits at H1/H2 boundaries into sections (mirrors the heading
// split in internal/md/extract.go splitBySections) so each heading is a TOC entry.

import { marked } from "../vendor/marked.esm.js";
import { sanitizeToFragment } from "./sanitize.js";

// parseMarkdown decodes UTF-8 bytes and returns the render-ready book shape.
export async function parseMarkdown(data) {
  const text = new TextDecoder("utf-8").decode(data);
  const { frag } = sanitizeToFragment(marked.parse(text), 0);

  const sections = [];
  const toc = [];
  let cur = null;
  const openSection = (label) => {
    cur = { id: `md-sec-${sections.length}`, label: label || "", frag: document.createDocumentFragment() };
    sections.push(cur);
    return cur.id;
  };

  // Snapshot the child list before moving nodes into per-section fragments.
  for (const node of Array.from(frag.childNodes)) {
    const isHeading = node.nodeType === 1 && (node.tagName === "H1" || node.tagName === "H2");
    if (isHeading || !cur) {
      const label = isHeading ? node.textContent.replace(/\s+/g, " ").trim() : "";
      const id = openSection(label);
      if (isHeading) toc.push({ title: label, anchor: id, children: [] });
    }
    cur.frag.appendChild(node);
  }
  if (!sections.length) openSection("");

  let sampleText = "";
  for (const s of sections) {
    if (sampleText.length >= 8000) break;
    sampleText += ` ${s.frag.textContent || ""}`;
  }
  return { title: "", lang: "", sampleText, sections, toc, revoke: () => {} };
}
