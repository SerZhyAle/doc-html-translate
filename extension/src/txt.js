// txt.js - plain-text reader. Splits text into paragraphs and paginates them into
// sections. Mirrors internal/txt/extract.go: normalize line endings; if the text
// has blank lines use them as paragraph separators (consecutive non-blank lines
// join with a space), otherwise treat each non-empty line as its own paragraph;
// 30 paragraphs per page.

const PARAS_PER_SECTION = 30;

// splitParagraphs turns raw text into paragraphs. Pure (no DOM) - unit-tested.
export function splitParagraphs(text) {
  const norm = String(text).replace(/\r\n?/g, "\n");
  if (norm.includes("\n\n")) {
    const paras = [];
    let cur = "";
    for (const line of norm.split("\n")) {
      const t = line.trim();
      if (t === "") {
        if (cur) { paras.push(cur); cur = ""; }
      } else {
        cur = cur ? `${cur} ${t}` : t;
      }
    }
    if (cur) paras.push(cur);
    return paras;
  }
  return norm.split("\n").map((l) => l.trim()).filter((l) => l);
}

// paragraphsToBook chunks paragraphs into sections of PARAS_PER_SECTION and builds
// each as <p> nodes via textContent (no HTML injection). Shared by txt and rtf.
export function paragraphsToBook(paragraphs, idPrefix) {
  const sections = [];
  for (let i = 0; i < paragraphs.length; i += PARAS_PER_SECTION) {
    const chunk = paragraphs.slice(i, i + PARAS_PER_SECTION);
    const frag = document.createDocumentFragment();
    for (const p of chunk) {
      const node = document.createElement("p");
      node.textContent = p;
      frag.appendChild(node);
    }
    sections.push({ id: `${idPrefix}-${sections.length}`, label: "", frag });
  }
  let sampleText = "";
  for (const p of paragraphs) {
    if (sampleText.length >= 8000) break;
    sampleText += ` ${p}`;
  }
  return { title: "", lang: "", sampleText, sections, toc: [], revoke: () => {} };
}

// parseText decodes UTF-8 bytes and returns the render-ready book shape.
export async function parseText(data) {
  const text = new TextDecoder("utf-8").decode(data);
  return paragraphsToBook(splitParagraphs(text), "txt-sec");
}
