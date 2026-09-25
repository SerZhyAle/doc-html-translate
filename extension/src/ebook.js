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

// Book-internal link schemes foliate emits: MOBI6 "filepos:<n>" and KF8 "kindle:pos:..".
// The shared sanitizer drops any scheme it does not know, so these are resolved to in-page
// targets first and carried through it on a data attribute.
const BOOK_LINK = /^(filepos|kindle):/i;
const TARGET_ATTR = "data-dht-target";

async function bookLinkTarget(href, book) {
  try {
    if (/^filepos:/i.test(href)) {
      // MOBI6 plants <a id="filepos<n>"> in the target section, so the exact spot is known;
      // the sanitizer namespaces that id as d<index>-.
      const [index, id] = book.splitTOCHref(href);
      return index >= 0 ? `d${index}-${id}` : "";
    }
    const r = await book.resolveHref(href);
    return r && r.index != null && r.index >= 0 ? `ebook-sec-${r.index}` : "";
  } catch {
    return "";
  }
}

// retargetBookLinks rewrites book-internal links in one section's HTML into data-dht-target
// markers; applyBookLinks turns them into "#.." hrefs after sanitizing.
export async function retargetBookLinks(html, book) {
  const doc = new DOMParser().parseFromString(html, "text/html");
  // A document-supplied marker would be trusted below as ours.
  for (const e of Array.from(doc.querySelectorAll(`[${TARGET_ATTR}]`))) e.removeAttribute(TARGET_ATTR);
  let changed = false;
  for (const a of Array.from(doc.querySelectorAll("a[href]"))) {
    const href = a.getAttribute("href").trim();
    if (!BOOK_LINK.test(href)) continue;
    const target = await bookLinkTarget(href, book);
    a.removeAttribute("href");
    if (target) a.setAttribute(TARGET_ATTR, target);
    changed = true;
  }
  return changed ? doc.documentElement.outerHTML : html;
}

export function applyBookLinks(frag) {
  for (const a of Array.from(frag.querySelectorAll(`[${TARGET_ATTR}]`))) {
    a.setAttribute("href", `#${a.getAttribute(TARGET_ATTR)}`);
    a.removeAttribute(TARGET_ATTR);
  }
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
  let remote = 0;
  const list = book.sections || [];
  for (let i = 0; i < list.length; i++) {
    const sec = list[i];
    if (!sec || typeof sec.load !== "function") continue; // skip non-linear placeholders
    let frag;
    let label = "";
    try {
      const url = await sec.load();
      const html = await retargetBookLinks(await (await fetch(url)).text(), book);
      URL.revokeObjectURL(url); // the section-doc blob is not needed after reading
      let r;
      ({ frag, label, remote: r } = sanitizeToFragment(html, i));
      applyBookLinks(frag);
      remote += r;
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

  return { title, lang, sampleText, sections, toc, remote, revoke };
}
