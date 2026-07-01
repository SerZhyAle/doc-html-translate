// ebook.js - MOBI and AZW3 (KF8) reader, adapting the vendored foliate-js mobi.js
// to the extension's book shape. foliate replaces Calibre here: it parses MOBI and
// KF8 directly in the browser. mobi.js is self-contained and receives its unzlib
// (fflate, for KF8 fonts) via the constructor.
//
// Each foliate section's load() returns a blob: URL to fully-resolved HTML (images
// as blob: URLs, styles injected) for both MOBI6 and KF8, so we fetch that once,
// sanitize the body into a fragment, and collect the image blob: URLs for teardown.

import { MOBI } from "../vendor/foliate/mobi.js";
import { unzlibSync } from "../vendor/fflate.js";
import { sanitizeToFragment } from "./sanitize.js";
import { normalizeLangTag } from "./lang.js";

// isMobiBytes reports whether the buffer is a MOBI/KF8 (both carry the PDB
// type+creator "BOOKMOBI" at offset 60). Pure - safe to call from detectFormat.
export function isMobiBytes(data) {
  const b = new Uint8Array(data);
  if (b.byteLength < 68) return false;
  let magic = "";
  for (let i = 60; i < 68; i++) magic += String.fromCharCode(b[i]);
  return magic === "BOOKMOBI";
}

// mapToc maps foliate TOC entries to the extension's {title, anchor, children},
// resolving each href to the containing section index. resolveHref is sync for
// MOBI6 and async for KF8, so await covers both.
async function mapToc(items, book) {
  if (!Array.isArray(items)) return [];
  const out = [];
  for (const it of items) {
    let anchor = null;
    try {
      const r = await book.resolveHref(it.href);
      if (r && r.index != null) anchor = `ebook-sec-${r.index}`;
    } catch { /* unresolved href -> label-only entry */ }
    const children = await mapToc(it.subitems, book);
    if (anchor || children.length) out.push({ title: it.label || "", anchor, children });
  }
  return out;
}

// parseEbook opens MOBI/AZW3 bytes and returns the render-ready book shape.
export async function parseEbook(data) {
  const book = await new MOBI({ unzlib: unzlibSync }).open(new Blob([data]));

  const blobUrls = [];
  const sections = [];
  const list = book.sections || [];
  for (let i = 0; i < list.length; i++) {
    const sec = list[i];
    if (!sec || typeof sec.load !== "function") continue; // skip non-linear placeholders
    let frag;
    let label = "";
    try {
      const url = await sec.load();
      const html = await (await fetch(url)).text();
      URL.revokeObjectURL(url); // the section-doc blob is not needed after reading
      ({ frag, label } = sanitizeToFragment(html, i));
      frag.querySelectorAll("img[src^='blob:']").forEach((img) => blobUrls.push(img.getAttribute("src")));
    } catch {
      ({ frag, label } = sanitizeToFragment("", i));
    }
    sections.push({ id: `ebook-sec-${i}`, label, frag });
  }

  const md = book.metadata || {};
  const title = typeof md.title === "string" ? md.title : (md.title && md.title.value) || "";
  const lang = normalizeLangTag(Array.isArray(md.language) ? md.language[0] : md.language || "");
  const toc = await mapToc(book.toc, book);

  let sampleText = "";
  for (const s of sections) {
    if (sampleText.length >= 8000) break;
    sampleText += ` ${s.frag.textContent || ""}`;
  }

  const revoke = () => {
    for (const u of blobUrls) { try { URL.revokeObjectURL(u); } catch { /* ignore */ } }
    if (typeof book.destroy === "function") { try { book.destroy(); } catch { /* ignore */ } }
  };

  return { title, lang, sampleText, sections, toc, revoke };
}
