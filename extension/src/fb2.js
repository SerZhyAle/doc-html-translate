// fb2.js - FictionBook 2 reader. Ports internal/fb2/extract.go (walk <body> ->
// <section> recursively, collecting <title>/<epigraph>/<p> text) and additionally
// inlines embedded <binary> images as data: URLs and builds a nested TOC from the
// section titles. Pure XML via DOMParser; no external library.

import { normalizeLangTag } from "./lang.js";

const XLINK = "http://www.w3.org/1999/xlink";

function textOf(el) {
  return el ? el.textContent.replace(/\s+/g, " ").trim() : "";
}

function childByLocal(el, local) {
  for (const c of el.children) if (c.localName === local) return c;
  return null;
}

// titleText joins a section's <title>'s <p> lines into one label.
function titleText(secEl) {
  const t = childByLocal(secEl, "title");
  if (!t) return "";
  return Array.from(t.children).filter((c) => c.localName === "p").map(textOf).join(" ").trim();
}

// parseFb2 decodes UTF-8 bytes and returns the render-ready book shape.
export async function parseFb2(data) {
  const doc = new DOMParser().parseFromString(new TextDecoder("utf-8").decode(data), "application/xml");
  if (doc.getElementsByTagName("parsererror").length) throw new Error("invalid FB2 XML");

  // Embedded images: <binary id content-type>base64</binary> -> data: URL by id.
  const binaries = new Map();
  for (const b of Array.from(doc.getElementsByTagNameNS("*", "binary"))) {
    const id = b.getAttribute("id");
    if (id) {
      const ct = b.getAttribute("content-type") || "image/jpeg";
      binaries.set(id, `data:${ct};base64,${(b.textContent || "").replace(/\s+/g, "")}`);
    }
  }

  const titleInfo = doc.getElementsByTagNameNS("*", "title-info")[0] || null;
  const title = titleInfo ? textOf(childByLocal(titleInfo, "book-title")) : "";
  const lang = titleInfo ? normalizeLangTag(textOf(childByLocal(titleInfo, "lang"))) : "";

  const imgFor = (imageEl) => {
    let href =
      imageEl.getAttributeNS(XLINK, "href") ||
      imageEl.getAttribute("l:href") ||
      imageEl.getAttribute("xlink:href") ||
      imageEl.getAttribute("href") || "";
    href = href.replace(/^#/, "");
    const url = binaries.get(href);
    if (!url) return null;
    const img = document.createElement("img");
    img.src = url;
    return img;
  };

  let counter = 0;
  // renderSection builds a fragment for one <section> and returns its TOC entry.
  const renderSection = (secEl, depth) => {
    const id = `fb2-sec-${counter++}`;
    const label = titleText(secEl);
    const frag = document.createDocumentFragment();
    const heading = document.createElement(depth <= 0 ? "h2" : depth === 1 ? "h3" : "h4");
    heading.id = id;
    heading.textContent = label;
    frag.appendChild(heading);
    const children = [];
    for (const child of Array.from(secEl.children)) {
      const ln = child.localName;
      if (ln === "title") continue;
      if (ln === "p") {
        const p = document.createElement("p");
        p.textContent = textOf(child);
        frag.appendChild(p);
      } else if (ln === "image") {
        const img = imgFor(child);
        if (img) frag.appendChild(img);
      } else if (ln === "epigraph") {
        for (const ep of Array.from(child.children)) {
          if (ep.localName === "p") {
            const p = document.createElement("p");
            p.textContent = textOf(ep);
            frag.appendChild(p);
          }
        }
      } else if (ln === "section") {
        const sub = renderSection(child, depth + 1);
        frag.appendChild(sub.frag);
        children.push(sub.entry);
      }
    }
    return { frag, entry: { title: label, anchor: id, children } };
  };

  const sections = [];
  const toc = [];
  for (const body of Array.from(doc.getElementsByTagNameNS("*", "body"))) {
    for (const sec of Array.from(body.children)) {
      if (sec.localName !== "section") continue;
      const r = renderSection(sec, 0);
      sections.push({ id: `fb2-page-${sections.length}`, label: r.entry.title, frag: r.frag });
      toc.push(r.entry);
    }
  }

  // Fallback: a book with no <section> structure - gather every <p> into one page.
  if (!sections.length) {
    const frag = document.createDocumentFragment();
    for (const p of Array.from(doc.getElementsByTagNameNS("*", "p"))) {
      const node = document.createElement("p");
      node.textContent = textOf(p);
      if (node.textContent) frag.appendChild(node);
    }
    sections.push({ id: "fb2-page-0", label: "", frag });
  }

  let sampleText = "";
  for (const s of sections) {
    if (sampleText.length >= 8000) break;
    sampleText += ` ${s.frag.textContent || ""}`;
  }
  return { title, lang, sampleText, sections, toc, revoke: () => {} };
}
