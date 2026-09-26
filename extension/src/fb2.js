// fb2.js - FictionBook 2 reader. Ports internal/fb2 (walk <body> -> <section> recursively,
// keeping every prose element: paragraphs, verse stanzas, subtitles, epigraph and poem
// authors, citations, table cells) and additionally inlines embedded <binary> images as data:
// URLs and builds a nested TOC from the section titles. Pure XML via DOMParser; no external
// library.

import { normalizeLangTag } from "./lang.js";

const XLINK = "http://www.w3.org/1999/xlink";

// The XML declaration must open the document, so only its head is searched. Mirrors
// declPrescanBytes and xmlDeclEncodingRe in internal/fb2/content.go.
const DECL_PRESCAN_BYTES = 1024;
const XML_DECL_ENCODING = /^[\t\n\f\r ]*<\?xml[\t\n\f\r ][^>]*?\bencoding[\t\n\f\r ]*=[\t\n\f\r ]*["']([A-Za-z0-9._:-]+)["']/;

// decodeFb2 turns FB2 bytes into a string: a BOM wins, then the encoding the XML declaration
// names (a WHATWG label, which TextDecoder takes directly), then UTF-8. A Russian FB2 declared
// windows-1251 used to be decoded as UTF-8 and came out as replacement characters. An unknown
// label, or a UTF-16 label with no BOM (an ASCII declaration rules UTF-16 out), reads as UTF-8.
// Mirrors decodingReader in internal/fb2/content.go. Exported for the unit test.
export function decodeFb2(data) {
  const b = new Uint8Array(data);
  if (b.length >= 3 && b[0] === 0xef && b[1] === 0xbb && b[2] === 0xbf) return new TextDecoder("utf-8").decode(b);
  if (b.length >= 2) {
    if (b[0] === 0xff && b[1] === 0xfe) return new TextDecoder("utf-16le").decode(b);
    if (b[0] === 0xfe && b[1] === 0xff) return new TextDecoder("utf-16be").decode(b);
  }
  let head = "";
  for (let i = 0; i < Math.min(b.length, DECL_PRESCAN_BYTES); i++) head += String.fromCharCode(b[i]);
  const m = XML_DECL_ENCODING.exec(head);
  if (m) {
    try {
      const d = new TextDecoder(m[1]);
      if (d.encoding !== "utf-8" && !d.encoding.startsWith("utf-16")) return d.decode(b);
    } catch {
      // Not a label the browser knows: fall through to UTF-8.
    }
  }
  return new TextDecoder("utf-8").decode(b);
}

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

// inlineImages lists the <image> elements nested inside a prose element, in document order.
function inlineImages(el) {
  const found = [];
  for (const c of Array.from(el.children)) {
    if (c.localName === "image") found.push(c);
    else found.push(...inlineImages(c));
  }
  return found;
}

// The paragraph class per prose element; the same classes internal/fb2 writes.
const PROSE_CLASS = { p: "", subtitle: "subtitle", "text-author": "text-author", td: "", th: "" };

// Elements whose children are rendered in place: they group prose but carry none themselves.
const CONTAINERS = new Set(["epigraph", "cite", "poem", "annotation", "table", "tr", "title"]);

// parseFb2 decodes the bytes and returns the render-ready book shape.
export async function parseFb2(data) {
  const doc = new DOMParser().parseFromString(decodeFb2(data), "application/xml");
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

  // imgFor renders an <image>: the picture, or - when no <binary> carries it - the visible
  // note internal/fb2 writes in its place, so a missing picture is never a silent gap.
  const imgFor = (imageEl) => {
    let href =
      imageEl.getAttributeNS(XLINK, "href") ||
      imageEl.getAttribute("l:href") ||
      imageEl.getAttribute("xlink:href") ||
      imageEl.getAttribute("href") || "";
    href = href.replace(/^#/, "");
    if (!href) return null;
    const url = binaries.get(href);
    if (!url) {
      const p = document.createElement("p");
      const em = document.createElement("em");
      em.textContent = `[image not found: ${href}]`;
      p.appendChild(em);
      return p;
    }
    const img = document.createElement("img");
    img.src = url;
    return img;
  };

  const addImages = (el, frag) => {
    for (const im of inlineImages(el)) {
      const img = imgFor(im);
      if (img) frag.appendChild(img);
    }
  };

  const addPara = (frag, text, cls) => {
    if (!text) return;
    const p = document.createElement("p");
    if (cls) p.className = cls;
    p.textContent = text;
    frag.appendChild(p);
  };

  // A stanza is one paragraph with its verse lines separated by <br>.
  const addStanza = (frag, lines) => {
    if (!lines.length) return;
    const p = document.createElement("p");
    p.className = "stanza";
    lines.forEach((line, i) => {
      if (i) p.appendChild(document.createElement("br"));
      p.appendChild(document.createTextNode(line));
    });
    frag.appendChild(p);
  };

  // renderBlock renders one non-section body element, mirroring the element set parseFB2
  // keeps in internal/fb2/content.go.
  const renderBlock = (el, frag) => {
    const ln = el.localName;
    if (ln in PROSE_CLASS) {
      addPara(frag, textOf(el), PROSE_CLASS[ln]);
      addImages(el, frag);
    } else if (ln === "image") {
      const img = imgFor(el);
      if (img) frag.appendChild(img);
    } else if (ln === "stanza") {
      // A stanza's own <title> and <subtitle> come before its verses, as internal/fb2 emits them.
      const verses = [];
      for (const c of Array.from(el.children)) {
        if (c.localName === "v") verses.push(c);
        else renderBlock(c, frag);
      }
      addStanza(frag, verses.map(textOf).filter((t) => t));
      for (const v of verses) addImages(v, frag);
    } else if (ln === "v") {
      addStanza(frag, [textOf(el)].filter((t) => t));
      addImages(el, frag);
    } else if (CONTAINERS.has(ln)) {
      for (const child of Array.from(el.children)) renderBlock(child, frag);
    }
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
      if (ln === "section") {
        const sub = renderSection(child, depth + 1);
        frag.appendChild(sub.frag);
        children.push(sub.entry);
      } else {
        renderBlock(child, frag);
      }
    }
    return { frag, entry: { title: label, anchor: id, children } };
  };

  const sections = [];
  const toc = [];
  // The cover lives in <description>, ahead of the body, so it opens the book - the first image
  // of any <coverpage>, as internal/fb2 picks it.
  for (const cp of Array.from(doc.getElementsByTagNameNS("*", "coverpage"))) {
    const im = cp.getElementsByTagNameNS("*", "image")[0];
    const cover = im && imgFor(im);
    if (cover) {
      const frag = document.createDocumentFragment();
      frag.appendChild(cover);
      sections.push({ id: `fb2-page-${sections.length}`, label: "", frag });
      break;
    }
  }
  for (const body of Array.from(doc.getElementsByTagNameNS("*", "body"))) {
    // A body's own title and epigraph come before its sections; they get a page of their own
    // rather than being dropped.
    let lead = null;
    const flushLead = () => {
      if (lead && lead.childNodes.length) sections.push({ id: `fb2-page-${sections.length}`, label: "", frag: lead });
      lead = null;
    };
    for (const child of Array.from(body.children)) {
      if (child.localName !== "section") {
        if (!lead) lead = document.createDocumentFragment();
        renderBlock(child, lead);
        continue;
      }
      flushLead();
      const r = renderSection(child, 0);
      sections.push({ id: `fb2-page-${sections.length}`, label: r.entry.title, frag: r.frag });
      toc.push(r.entry);
    }
    flushLead();
  }

  // Fallback: a book with no <body> structure - gather every <p> into one page.
  if (!sections.length) {
    const frag = document.createDocumentFragment();
    for (const p of Array.from(doc.getElementsByTagNameNS("*", "p"))) addPara(frag, textOf(p), "");
    sections.push({ id: "fb2-page-0", label: "", frag });
  }

  let sampleText = "";
  for (const s of sections) {
    if (sampleText.length >= 8000) break;
    sampleText += ` ${s.frag.textContent || ""}`;
  }
  return { title, lang, sampleText, sections, toc, revoke: () => {} };
}
