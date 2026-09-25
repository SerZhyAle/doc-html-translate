// epub.js - load an EPUB (a ZIP of XHTML) and turn it into one clean, reflowed
// HTML document the browser can translate in place.
//
// Unlike PDF, EPUB content is *already* semantic XHTML, so there is no reflow
// heuristic to run: the work is unzipping the archive, following the OPF spine in
// reading order, sanitizing each chapter, rewriting its images to blob: URLs and
// its internal links to in-page anchors, and reading the authored TOC. This mirrors
// the desktop app's internal/epub package (container -> OPF -> spine, EPUB3 nav /
// EPUB2 NCX table of contents, single-image SVG cover -> <img>), but renders into
// the DOM instead of writing files to disk.
//
// The pure pieces - unzip(), resolveBookPath() - use no DOM and are unit-tested under
// node; everything that parses XHTML uses DOMParser and runs only in the viewer.

import { normalizeLangTag } from "./lang.js";
import { EPUB_MAX_ENTRY_BYTES, InputLimitError, checkArchive, entryTooLarge, inflateRawCapped } from "./limits.js";

// ---- ZIP reader ------------------------------------------------------------
// A minimal, central-directory-driven reader. Sizes and the local-header offset
// come from the central directory (always authoritative), so data descriptors
// (streamed sizes in the local header) need no special handling. Deflate is
// inflated with the platform's native DecompressionStream (Chrome 80+ / node 18+),
// so we ship no compression dependency.
//
// The listing is held against the input limits (limits.js) before anything is
// inflated, and each inflation counts its bytes, so a zip bomb is refused rather
// than expanded into the tab's memory. The same limits apply on the desktop
// (internal/epub, docs/PARITY.md "Input limits").

const SIG_EOCD = 0x06054b50;
const SIG_CEN = 0x02014b50;

// findEOCD scans backwards for the End Of Central Directory signature. The record
// is 22 bytes plus an up-to-64KB trailing comment, so we look no further back.
function findEOCD(dv, len) {
  const min = Math.max(0, len - 22 - 0xffff);
  for (let i = len - 22; i >= min; i--) {
    if (dv.getUint32(i, true) === SIG_EOCD) return i;
  }
  return -1;
}

// unzip parses a ZIP archive into a Map of entry name -> bytes. Directory entries
// are skipped, and so is an entry over EPUB_MAX_ENTRY_BYTES, by name (the desktop
// edition skips it the same way). Throws on a non-ZIP buffer, ZIP64, an
// unsupported compression method, or a listing over the archive limits. Async
// because inflate is stream-based. Pure (no DOM) - unit-tested.
export async function unzip(arrayBuffer) {
  const u8 = new Uint8Array(arrayBuffer);
  const dv = new DataView(arrayBuffer);
  if (u8.length < 22) throw new Error("not a ZIP archive (too small)");

  const eocd = findEOCD(dv, u8.length);
  if (eocd < 0) throw new Error("not a ZIP archive (no end-of-central-directory record)");

  const count = dv.getUint16(eocd + 10, true);
  const cdOffset = dv.getUint32(eocd + 16, true);
  if (cdOffset === 0xffffffff) throw new Error("ZIP64 archives are not supported");

  const dec = new TextDecoder("utf-8");
  const listing = [];
  let total = 0;
  let p = cdOffset;
  for (let i = 0; i < count; i++) {
    if (p + 46 > u8.length || dv.getUint32(p, true) !== SIG_CEN) break;
    const nameLen = dv.getUint16(p + 28, true);
    const entry = {
      method: dv.getUint16(p + 10, true),
      compSize: dv.getUint32(p + 20, true),
      size: dv.getUint32(p + 24, true),
      localOff: dv.getUint32(p + 42, true),
      name: dec.decode(u8.subarray(p + 46, p + 46 + nameLen)),
    };
    p += 46 + nameLen + dv.getUint16(p + 30, true) + dv.getUint16(p + 32, true);
    if (entry.name.endsWith("/")) continue;
    if (entry.size > EPUB_MAX_ENTRY_BYTES) {
      console.warn(`EPUB: skipped ${entryTooLarge(entry.name, EPUB_MAX_ENTRY_BYTES).message}`);
      continue;
    }
    total += entry.size;
    listing.push(entry);
  }
  checkArchive(count, total);

  const files = new Map();
  for (const e of listing) {
    // Locate the entry's data via its local header (only its name/extra lengths
    // are needed; the sizes there may be zero for streamed entries).
    const lhNameLen = dv.getUint16(e.localOff + 26, true);
    const lhExtraLen = dv.getUint16(e.localOff + 28, true);
    const dataStart = e.localOff + 30 + lhNameLen + lhExtraLen;
    const comp = u8.subarray(dataStart, dataStart + e.compSize);
    try {
      if (e.method === 0) {
        if (comp.length > EPUB_MAX_ENTRY_BYTES) throw entryTooLarge(e.name, EPUB_MAX_ENTRY_BYTES);
        files.set(e.name, comp.slice());
      } else if (e.method === 8) {
        // Capped at the listing's size, which is what the total was checked against: an
        // entry that inflates past it is refused as the desktop's zip reader refuses it,
        // and never inflated further than its header admitted.
        files.set(e.name, await inflateRawCapped(comp, e.name, e.size));
      } else {
        throw new Error(`unsupported ZIP compression method ${e.method} for ${e.name}`);
      }
    } catch (err) {
      // Dropped by name, never kept cut short: a chapter cut mid-tag would read as a
      // complete book.
      if (!(err instanceof InputLimitError)) throw err;
      console.warn(`EPUB: skipped ${err.message}`);
    }
  }
  return files;
}

// ---- Path helpers (pure) ---------------------------------------------------

// resolveBookPath turns a name the book supplies (container full-path, manifest
// href, link, image or TOC target) into an archive entry path, or null when the
// name must be dropped. It is the port of the desktop resolveBookPath
// (internal/epub/resolve.go), and both run the shared fixture
// tests/testdata/epub_href_cases.json - see ../../docs/PARITY.md, "EPUB href
// resolution". The rule, in order:
//   1. cut ?query and #fragment off the raw value, before decoding, so an encoded
//      %23 stays part of the name;
//   2. percent-decode once (a malformed escape keeps the raw text);
//   3. treat "\" as "/";
//   4. refuse a colon, a leading "//", control characters, a segment ending in a
//      dot or space, and Windows device names - the desktop edition writes these
//      to disk, and one rule keeps the two editions reading the same chapters;
//   5. a leading "/" is archive-root-relative, anything else resolves against
//      baseDir;
//   6. the result must name something strictly inside the archive.
// Pure (no DOM) - unit-tested.
export function resolveBookPath(baseDir, href) {
  let p = String(href ?? "");
  const cut = p.search(/[?#]/);
  if (cut >= 0) p = p.slice(0, cut);
  p = decodeHref(p).replace(/\\/g, "/");
  if (!p || p.startsWith("//")) return null;
  if (/[\u0000-\u001f\u007f:]/.test(p)) return null;
  for (const seg of p.split("/")) {
    if (seg === "" || seg === "." || seg === "..") continue;
    if (/[. ]$/.test(seg) || isWindowsDeviceName(seg)) return null;
  }
  const stack = !p.startsWith("/") && baseDir ? baseDir.split("/").filter((s) => s && s !== ".") : [];
  for (const part of p.split("/")) {
    if (part === "" || part === ".") continue;
    if (part === "..") {
      if (stack.length === 0) return null;
      stack.pop();
    } else {
      stack.push(part);
    }
  }
  return stack.length ? stack.join("/") : null;
}

// isWindowsDeviceName: CON, NUL, COM1.. with or without an extension.
function isWindowsDeviceName(seg) {
  const stem = seg.split(".")[0].replace(/ +$/, "").toUpperCase();
  return /^(CON|PRN|AUX|NUL|CONIN\$|CONOUT\$|COM[1-9]|LPT[1-9])$/.test(stem);
}

function dirOf(path) {
  const i = path.lastIndexOf("/");
  return i < 0 ? "" : path.slice(0, i);
}

function splitFrag(href) {
  const i = href.indexOf("#");
  return i >= 0 ? [href.slice(0, i), href.slice(i + 1)] : [href, ""];
}

function decodeHref(s) {
  try { return decodeURIComponent(s); } catch { return s; }
}

function hasToken(value, token) {
  return String(value || "").split(/\s+/).includes(token);
}

function decodeText(bytes) {
  return bytes ? new TextDecoder("utf-8").decode(bytes) : "";
}

const MIME_BY_EXT = {
  jpg: "image/jpeg", jpeg: "image/jpeg", png: "image/png", gif: "image/gif",
  webp: "image/webp", svg: "image/svg+xml", bmp: "image/bmp", tif: "image/tiff",
  tiff: "image/tiff", avif: "image/avif",
};
function guessMime(path) {
  const ext = path.slice(path.lastIndexOf(".") + 1).toLowerCase();
  return MIME_BY_EXT[ext] || "application/octet-stream";
}

// ---- OPF / container parsing (DOM) -----------------------------------------

// parseContainer returns the OPF package path from META-INF/container.xml.
function parseContainer(xml) {
  const doc = new DOMParser().parseFromString(xml, "application/xml");
  const rootfiles = Array.from(doc.getElementsByTagNameNS("*", "rootfile"));
  for (const rf of rootfiles) {
    const fp = rf.getAttribute("full-path");
    const mt = rf.getAttribute("media-type");
    if (fp && (mt === "application/oebps-package+xml" || fp.toLowerCase().endsWith(".opf"))) return fp;
  }
  if (rootfiles[0] && rootfiles[0].getAttribute("full-path")) return rootfiles[0].getAttribute("full-path");
  throw new Error("no rootfile in container.xml");
}

// parseOpf reads the package document: title, language, manifest, spine order.
// getElementsByTagNameNS("*", local) matches regardless of the OPF/DC namespace
// prefix the author used, mirroring the Go xml.Unmarshal behaviour.
function parseOpf(xml) {
  const doc = new DOMParser().parseFromString(xml, "application/xml");
  const titleEl = doc.getElementsByTagNameNS("*", "title")[0];
  const langEl = doc.getElementsByTagNameNS("*", "language")[0];

  const manifest = Array.from(doc.getElementsByTagNameNS("*", "item"))
    .map((it) => ({
      id: it.getAttribute("id") || "",
      href: it.getAttribute("href") || "",
      mediaType: it.getAttribute("media-type") || "",
      properties: it.getAttribute("properties") || "",
    }))
    .filter((it) => it.href);

  const spineEl = doc.getElementsByTagNameNS("*", "spine")[0];
  const spine = [];
  let spineTocId = "";
  if (spineEl) {
    spineTocId = spineEl.getAttribute("toc") || "";
    for (const ir of Array.from(spineEl.getElementsByTagNameNS("*", "itemref"))) {
      const idref = ir.getAttribute("idref");
      if (idref) spine.push(idref);
    }
  }

  return {
    title: titleEl ? titleEl.textContent.trim() : "",
    lang: langEl ? langEl.textContent.trim() : "",
    manifest,
    spine,
    spineTocId,
  };
}

// ---- TOC (nav.xhtml / toc.ncx -> in-page anchors) --------------------------

function directChild(el, localName) {
  for (const c of el.children) if (c.localName === localName) return c;
  return null;
}
function directChildren(el, localName) {
  return Array.from(el.children).filter((c) => c.localName === localName);
}

// firstAnchorBeforeList finds the first <a> under li that is not inside a nested
// list (which belongs to child entries, not this li's own label).
function firstAnchorBeforeList(li) {
  let found = null;
  const walk = (n) => {
    for (const c of n.children) {
      if (found) return;
      const tag = c.localName;
      if (tag === "ol" || tag === "ul") continue;
      if (tag === "a") { found = c; return; }
      walk(c);
    }
  };
  walk(li);
  return found;
}

function parseNavList(ol) {
  const out = [];
  for (const li of directChildren(ol, "li")) {
    let title = "", href = "";
    const a = firstAnchorBeforeList(li);
    if (a) {
      title = a.textContent.replace(/\s+/g, " ").trim();
      href = a.getAttribute("href") || "";
    } else {
      const span = directChild(li, "span");
      if (span) title = span.textContent.replace(/\s+/g, " ").trim();
    }
    const nested = directChild(li, "ol");
    const children = nested ? parseNavList(nested) : [];
    if (title || href || children.length) out.push({ title, href, children });
  }
  return out;
}

// parseNavToc reads an EPUB3 navigation document, preferring the <nav
// epub:type="toc"> (then role="doc-toc", then the first nav). Lenient HTML parse,
// matching the desktop app's use of x/net/html for nav docs.
export function parseNavToc(html) {
  const doc = new DOMParser().parseFromString(html, "text/html");
  const navs = Array.from(doc.querySelectorAll("nav"));
  const nav =
    navs.find((n) => hasToken(n.getAttribute("epub:type"), "toc")) ||
    navs.find((n) => hasToken(n.getAttribute("role"), "doc-toc")) ||
    navs[0];
  if (!nav) return [];
  const ol = nav.querySelector("ol");
  return ol ? parseNavList(ol) : [];
}

// parseNcxToc reads an EPUB2 NCX navMap into raw {title, href, children} entries.
export function parseNcxToc(xml) {
  const doc = new DOMParser().parseFromString(xml, "application/xml");
  const navMap = doc.getElementsByTagNameNS("*", "navMap")[0];
  if (!navMap) return [];
  const points = (parent) =>
    directChildren(parent, "navPoint").map((np) => {
      const navLabel = directChild(np, "navLabel");
      const textEl = navLabel ? directChild(navLabel, "text") : null;
      const content = directChild(np, "content");
      return {
        title: textEl ? textEl.textContent.replace(/\s+/g, " ").trim() : "",
        href: content ? content.getAttribute("src") || "" : "",
        children: points(np),
      };
    });
  return points(navMap);
}

// isExternalHref: any scheme with "://" (http, ftp, ..) or a mailto:/tel:/data: URI.
// Shared by the TOC resolver and the chapter-link rewriter. The desktop app's
// ExternalHref (internal/epub links.go) counts every scheme as external and keeps only
// http/https/mailto clickable; here a TOC entry that is external is dropped anyway, and a
// javascript:/vbscript: value fails resolveBookPath (colon) so it never becomes a link -
// see ../../docs/PARITY.md, "EPUB TOC parsing".
function isExternalHref(href) {
  const s = String(href || "").toLowerCase();
  return s.includes("://") || /^(mailto|tel|data):/.test(s);
}

// resolveTocAnchor maps a raw nav/NCX href to an in-page element id, or null when
// it points outside the rendered spine. Internal hrefs resolve relative to the TOC
// document's directory; a #fragment becomes "d<index>-<fragment>" (the namespaced
// id the chapter's element carries), a bare file becomes the chapter wrapper id.
function resolveTocAnchor(href, tocDir, pathToIndex) {
  href = String(href || "").trim();
  if (!href || isExternalHref(href)) return null;
  const [file, frag] = splitFrag(href);
  if (!file) return null;
  const idx = pathToIndex.get(resolveBookPath(tocDir, file));
  if (idx == null) return null;
  return frag ? `d${idx}-${frag}` : `epub-sec-${idx}`;
}

function resolveTocEntries(raw, tocDir, pathToIndex) {
  const out = [];
  for (const e of raw) {
    const children = resolveTocEntries(e.children, tocDir, pathToIndex);
    const anchor = resolveTocAnchor(e.href, tocDir, pathToIndex);
    if (!anchor && children.length === 0) continue;
    out.push({ title: e.title, anchor, children });
  }
  return out;
}

function buildEpubToc(pkg, files, opfDir, pathToIndex) {
  const navItem = pkg.manifest.find((it) => hasToken(it.properties, "nav"));
  if (navItem) {
    const navPath = navItem.path;
    const html = decodeText(files.get(navPath));
    if (html) {
      const resolved = resolveTocEntries(parseNavToc(html), dirOf(navPath), pathToIndex);
      if (resolved.length) return resolved;
    }
  }
  let ncxItem = pkg.spineTocId ? pkg.manifest.find((it) => it.id === pkg.spineTocId) : null;
  if (!ncxItem) ncxItem = pkg.manifest.find((it) => it.mediaType === "application/x-dtbncx+xml");
  if (ncxItem) {
    const ncxPath = ncxItem.path;
    const xml = decodeText(files.get(ncxPath));
    if (xml) {
      const resolved = resolveTocEntries(parseNcxToc(xml), dirOf(ncxPath), pathToIndex);
      if (resolved.length) return resolved;
    }
  }
  return [];
}

// ---- Chapter sanitize + rewrite (DOM) --------------------------------------

const DROP_TAGS = "script,style,link,base,meta,title,noscript,iframe,object,embed,form";

// rewriteImg points a relative <img src> at a blob: URL of the in-archive image,
// or removes the image when the target is missing. http(s)/data sources are left
// alone. srcset (which would carry now-broken relative refs) is dropped.
export function rewriteImg(img, docDir, blobFor) {
  const src = img.getAttribute("src");
  if (!src) { img.remove(); return; }
  img.removeAttribute("srcset");
  if (/^(https?|data):/i.test(src)) return;
  const target = resolveBookPath(docDir, src);
  const url = target && blobFor(target);
  if (url) img.setAttribute("src", url);
  else img.remove();
}

// convertSvgImage turns an SVG <image> into a plain <img>. When the <svg> wraps a
// single image (the Calibre/Kindlegen cover idiom) the whole <svg> is replaced, so
// the injected img CSS controls sizing and the browser applies EXIF orientation -
// mirroring the desktop normalizeCoverImages.
export function convertSvgImage(im, docDir, blobFor) {
  const href = im.getAttribute("href") || im.getAttribute("xlink:href") ||
    im.getAttributeNS("http://www.w3.org/1999/xlink", "href");
  const img = document.createElement("img");
  img.alt = im.getAttribute("alt") || "";
  if (href && !/^(https?|data):/i.test(href)) {
    const target = resolveBookPath(docDir, href);
    const url = target && blobFor(target);
    if (url) img.setAttribute("src", url);
  } else if (href) {
    img.setAttribute("src", href);
  }
  const svg = im.closest("svg");
  if (svg && svg.querySelectorAll("image").length === 1) svg.replaceWith(img);
  else im.replaceWith(img);
}

// rewriteAnchor turns a chapter's links into in-page navigation. External links
// open in a new tab; a same-document #fragment and cross-chapter file links are
// remapped onto the namespaced ids the combined document uses; links outside the
// rendered spine lose their href (kept as inert text).
export function rewriteAnchor(a, index, docDir, pathToIndex) {
  const nm = a.getAttribute("name");
  if (nm && !a.id) a.id = nm;
  const href = a.getAttribute("href");
  if (!href) return;
  if (isExternalHref(href)) {
    a.target = "_blank";
    a.rel = "noopener noreferrer";
    return;
  }
  if (href.startsWith("#")) {
    a.setAttribute("href", `#d${index}-${href.slice(1)}`);
    return;
  }
  const [file, frag] = splitFrag(href);
  const idx = pathToIndex.get(resolveBookPath(docDir, file));
  if (idx == null) { a.removeAttribute("href"); return; }
  a.setAttribute("href", frag ? `#d${idx}-${frag}` : `#epub-sec-${idx}`);
}

// renderChapter parses one spine XHTML document, strips unsafe/irrelevant nodes,
// rewrites images and links, namespaces ids so chapters can share one document,
// and returns a fragment plus a heading-derived label. The fragment's text is read
// by the caller (for language detection) before it is appended.
export function renderChapter(xhtml, index, docDir, pathToIndex, blobFor) {
  const parsed = new DOMParser().parseFromString(xhtml, "text/html");
  const body = parsed.body;
  const host = document.createElement("div");
  for (const child of Array.from(body ? body.childNodes : [])) {
    host.appendChild(document.importNode(child, true));
  }

  // A fragment link can target an id placed on <body> (or <html>); those elements
  // are not imported, so re-expose their ids as marker anchors at the top so links
  // like "#intro" -> "#d<index>-intro" still resolve.
  const rootIds = [];
  for (const root of [parsed.documentElement, body]) {
    const rid = root && root.getAttribute && root.getAttribute("id");
    if (rid) rootIds.push(rid);
  }

  host.querySelectorAll(DROP_TAGS).forEach((e) => e.remove());
  host.querySelectorAll("img").forEach((img) => rewriteImg(img, docDir, blobFor));
  host.querySelectorAll("image").forEach((im) => convertSvgImage(im, docDir, blobFor));
  host.querySelectorAll("a").forEach((a) => rewriteAnchor(a, index, docDir, pathToIndex));

  // Strip event handlers and author inline styles, and namespace every id so the
  // combined single-page document has no cross-chapter id collisions.
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
  for (const rid of rootIds) {
    const marker = document.createElement("span");
    marker.id = `d${index}-${rid}`;
    frag.appendChild(marker);
  }
  while (host.firstChild) frag.appendChild(host.firstChild);
  return { frag, label };
}

// ---- Public entry point ----------------------------------------------------

// loadEpub parses EPUB bytes into a render-ready model:
//   { title, lang, sampleText, sections:[{id,label,frag}], toc, revoke }
// `revoke` releases the image blob: URLs; call it before loading another file.
export async function loadEpub(arrayBuffer) {
  const files = await unzip(arrayBuffer);

  const containerXml = decodeText(files.get("META-INF/container.xml"));
  if (!containerXml) throw new Error("not an EPUB (missing META-INF/container.xml)");
  const opfPath = resolveBookPath("", parseContainer(containerXml));
  if (!opfPath) throw new Error("EPUB package document path points outside the book");
  const opfXml = decodeText(files.get(opfPath));
  if (!opfXml) throw new Error("EPUB package document not found");
  const opfDir = dirOf(opfPath);
  const pkg = parseOpf(opfXml);

  // Resolve every manifest href once, through the same gate as the desktop app:
  // an item that is malformed or points outside the book is dropped, and its
  // spine entries with it.
  pkg.manifest = pkg.manifest.filter((it) => {
    it.path = resolveBookPath(opfDir, it.href);
    if (!it.path) console.warn(`EPUB: skipped manifest item ${it.href}`);
    return it.path != null;
  });

  // Keep only spine entries whose file exists; one missing chapter costs that chapter.
  const byId = new Map(pkg.manifest.map((it) => [it.id, it]));
  const mediaByPath = new Map(pkg.manifest.map((it) => [it.path, it.mediaType]));
  const spine = [];
  for (const idref of pkg.spine) {
    const it = byId.get(idref);
    if (!it) continue;
    if (files.has(it.path)) spine.push(it.path);
    else console.warn(`EPUB: skipped spine item ${it.href}: not in the book`);
  }
  if (spine.length === 0) throw new Error("EPUB has no readable content in its spine");

  const pathToIndex = new Map();
  spine.forEach((zp, i) => pathToIndex.set(zp, i));

  // Lazily mint (and cache) one blob: URL per referenced in-archive image.
  const blobUrls = new Map();
  const blobFor = (zipPath) => {
    if (blobUrls.has(zipPath)) return blobUrls.get(zipPath);
    const bytes = files.get(zipPath);
    if (!bytes) return null;
    const type = mediaByPath.get(zipPath) || guessMime(zipPath);
    const url = URL.createObjectURL(new Blob([bytes], { type }));
    blobUrls.set(zipPath, url);
    return url;
  };

  const sections = [];
  let sampleText = "";
  for (let i = 0; i < spine.length; i++) {
    const zp = spine[i];
    const { frag, label } = renderChapter(decodeText(files.get(zp)), i, dirOf(zp), pathToIndex, blobFor);
    if (sampleText.length < 8000) sampleText += " " + (frag.textContent || "");
    sections.push({ id: `epub-sec-${i}`, label, frag });
  }

  const toc = buildEpubToc(pkg, files, opfDir, pathToIndex);

  return {
    title: pkg.title,
    lang: normalizeLangTag(pkg.lang),
    sampleText,
    sections,
    toc,
    revoke: () => { for (const url of blobUrls.values()) URL.revokeObjectURL(url); blobUrls.clear(); },
  };
}
