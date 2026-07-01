// viewer.js - orchestrator for the reflow viewer.
//
// Flow: read ?file= -> load the PDF with PDF.js -> detect the source language and
// set <html lang> (so Chrome offers "Translate page") -> build the TOC from the
// outline -> reflow every page's text into clean <p>/<h2>/<h3> and stream it into
// the DOM. All *text* ends up in the DOM (never virtualized) so native translate
// sees the whole document, including content that was off-screen at load.

import * as pdfjsLib from "../vendor/pdf.mjs";
import { reflowPage } from "./reflow.js";
import { buildToc } from "./toc.js";
import { detectLang, normalizeLangTag } from "./lang.js";
import { loadEpub } from "./epub.js";
import { parseText } from "./txt.js";
import { parseRtf } from "./rtf.js";
import { parseHtml } from "./html.js";
import { parseMarkdown } from "./md.js";
import { parseFb2 } from "./fb2.js";
import { parseEbook, isMobiBytes } from "./ebook.js";
import { overlayImage, makeBadge, ocrLangToHtmlLang } from "./ocr-overlay.js";
import { extractPageImages, rasterizePage } from "./pdf-images.js";
import { DEFAULT_OPTIONS } from "./defaults.js";

pdfjsLib.GlobalWorkerOptions.workerSrc = chrome.runtime.getURL("vendor/pdf.worker.mjs");

const VENDOR = {
  cMapUrl: chrome.runtime.getURL("vendor/cmaps/"),
  standardFontDataUrl: chrome.runtime.getURL("vendor/standard_fonts/"),
};

const $ = (id) => document.getElementById(id);
const el = (tag, cls) => {
  const e = document.createElement(tag);
  if (cls) e.className = cls;
  return e;
};

// ---- URL / params ----------------------------------------------------------
// The interception rule substitutes the original URL after `file=` *without*
// URL-encoding, so it may itself contain `?`/`&` (its own query string). Take the
// whole raw tail after `file=` rather than URLSearchParams, which would split on
// the embedded `&`. If the tail isn't already an absolute URL, treat it as
// percent-encoded (the manual viewer.html?file=<encoded> entry point).
function parseFileParam(search) {
  const m = /[?&]file=(.*)$/s.exec(search);
  if (!m) return "";
  const raw = m[1];
  if (/^(https?|file):/i.test(raw)) return raw;
  try { return decodeURIComponent(raw); } catch { return raw; }
}
const fileUrl = parseFileParam(location.search);

// Only ever hand http/https/file URLs to fetch()/navigation. parseFileParam can
// return any opener-supplied string (the viewer is web-accessible), so reject
// javascript:/data:/blob: and anything else before it reaches a navigation API.
function isSafePdfUrl(url) {
  return /^(https?|file):/i.test(url);
}

function fileTitle(url) {
  try {
    const u = new URL(url);
    const name = decodeURIComponent(u.pathname.split("/").pop() || "");
    return name.replace(/\.(pdf|epub|txt|rtf|html?|md|fb2|mobi|azw3|png|jpe?g|gif|bmp|webp)$/i, "") || "Document";
  } catch {
    return "Document";
  }
}

// FORMAT_EXT maps a filename extension to the internal format id. Each format
// phase extends this map; magic-byte detection below takes priority when a
// signature exists.
const FORMAT_EXT = { pdf: "pdf", epub: "epub", txt: "txt", rtf: "rtf", htm: "html", html: "html", md: "md", markdown: "md", fb2: "fb2", mobi: "mobi", azw3: "mobi", png: "image", jpg: "image", jpeg: "image", gif: "image", bmp: "image", webp: "image" };

function fileExt(name) {
  const clean = String(name || "").split(/[?#]/)[0];
  const dot = clean.lastIndexOf(".");
  return dot >= 0 ? clean.slice(dot + 1).toLowerCase() : "";
}

// detectFormat classifies bytes + source name into a format id. Byte signatures
// are authoritative (a mislabelled file still routes correctly); the filename
// extension is the fallback for formats without a reliable magic number.
function detectFormat(data, name) {
  const b = new Uint8Array(data, 0, Math.min(12, data.byteLength));
  if (b[0] === 0x25 && b[1] === 0x50 && b[2] === 0x44 && b[3] === 0x46) return "pdf";   // %PDF
  if (b[0] === 0x50 && b[1] === 0x4b && b[2] === 0x03 && b[3] === 0x04) return "epub";  // PK..
  if (b[0] === 0x7b && b[1] === 0x5c && b[2] === 0x72 && b[3] === 0x74 && b[4] === 0x66) return "rtf"; // {\rtf
  if (isMobiBytes(data)) return "mobi"; // MOBI / AZW3 (PDB "BOOKMOBI" at offset 60)
  if (imageMime(data, "")) return "image"; // PNG / JPEG / GIF / BMP / WebP by signature
  return FORMAT_EXT[fileExt(name)] || "unknown";
}

// imageMime returns the MIME type for a raster image the user opened directly, by byte
// signature first (authoritative) then the filename extension. Returns "" when the bytes
// and name are not a recognized image, which is also how detectFormat tells images apart.
function imageMime(data, name) {
  const b = new Uint8Array(data, 0, Math.min(12, data.byteLength));
  if (b[0] === 0x89 && b[1] === 0x50 && b[2] === 0x4e && b[3] === 0x47) return "image/png";  // .PNG
  if (b[0] === 0xff && b[1] === 0xd8 && b[2] === 0xff) return "image/jpeg";                  // JPEG SOI
  if (b[0] === 0x47 && b[1] === 0x49 && b[2] === 0x46 && b[3] === 0x38) return "image/gif";  // GIF8
  if (b[0] === 0x42 && b[1] === 0x4d) return "image/bmp";                                    // BM
  if (b[0] === 0x52 && b[1] === 0x49 && b[2] === 0x46 && b[3] === 0x46 &&
      b[8] === 0x57 && b[9] === 0x45 && b[10] === 0x42 && b[11] === 0x50) return "image/webp"; // RIFF..WEBP
  const byExt = { png: "image/png", jpg: "image/jpeg", jpeg: "image/jpeg", gif: "image/gif", bmp: "image/bmp", webp: "image/webp" };
  return byExt[fileExt(name)] || "";
}

// Release the previous document's resources (EPUB image blob: URLs) before
// loading another file into the same tab.
let revokeCurrent = null;
function teardownCurrent() {
  if (revokeCurrent) { try { revokeCurrent(); } catch { /* ignore */ } revokeCurrent = null; }
  if (ocrObserver) { ocrObserver.disconnect(); ocrObserver = null; }
  ocrTotal = 0;
  ocrDone = 0;
  for (const url of pdfImageUrls) { try { URL.revokeObjectURL(url); } catch { /* ignore */ } }
  pdfImageUrls = [];
}

// ---- Preferences -----------------------------------------------------------
const DEFAULT_PREFS = { size: 19, family: "serif", theme: null };
let prefs = { ...DEFAULT_PREFS };
let options = { ...DEFAULT_OPTIONS };

const FAMILIES = {
  serif: 'Georgia, "Times New Roman", serif',
  sans: '"Segoe UI", system-ui, Arial, sans-serif',
  mono: '"Cascadia Code", "Consolas", monospace',
};

async function loadPrefs() {
  try {
    const got = await chrome.storage.local.get(["viewerPrefs", "options"]);
    prefs = { ...DEFAULT_PREFS, ...(got.viewerPrefs || {}) };
    options = { ...DEFAULT_OPTIONS, ...(got.options || {}) };
  } catch { /* storage may be unavailable in odd contexts */ }
}
async function savePrefs() {
  try { await chrome.storage.local.set({ viewerPrefs: prefs }); } catch { /* ignore */ }
}
function applyPrefs() {
  document.documentElement.style.setProperty("--reader-size", `${prefs.size}px`);
  document.documentElement.style.setProperty("--reader-font", FAMILIES[prefs.family] || FAMILIES.serif);
  const theme = prefs.theme || options.theme || "light";
  document.documentElement.setAttribute("data-theme", theme);
  $("sel-family").value = prefs.family;
  $("sel-theme").value = theme;
}

// ---- Status / progress -----------------------------------------------------
function setStatus(text) { $("status-text").textContent = text; }
function setProgress(frac) { $("progress-bar").style.width = `${Math.round(frac * 100)}%`; }
function hideStatus() { $("status").classList.add("done"); }

// ---- Lazy image OCR --------------------------------------------------------
// When options.ocrImages is on, document images are OCR'd only as they scroll into
// view (a single shared worker processes them one at a time, so an image-heavy book
// never blocks reading). Recognized text is overlaid as opaque translatable plates.
// The status bar shows an overall "OCR: done/total" counter while any are pending.
let ocrObserver = null;
let ocrTotal = 0;
let ocrDone = 0;
const ocrQueued = new WeakSet();
let pdfImageUrls = []; // object URLs for PDF-extracted images, revoked on teardown

function ensureOcrCss() {
  if (document.getElementById("ocr-overlay-css")) return;
  const link = el("link");
  link.id = "ocr-overlay-css";
  link.rel = "stylesheet";
  link.href = chrome.runtime.getURL("src/ocr-overlay.css");
  document.head.append(link);
}

function ocrUpdateStatus() {
  if (ocrTotal === 0) return;
  $("status").classList.remove("done");
  setStatus(`OCR: ${ocrDone}/${ocrTotal}`);
  setProgress(ocrDone / ocrTotal);
  if (ocrDone >= ocrTotal) setTimeout(hideStatus, 1000);
}

function getOcrObserver() {
  if (ocrObserver) return ocrObserver;
  ocrObserver = new IntersectionObserver((entries, obs) => {
    for (const e of entries) {
      if (!e.isIntersecting) continue;
      obs.unobserve(e.target);
      ocrProcessImage(e.target);
    }
  }, { rootMargin: "300px" });
  return ocrObserver;
}

async function ocrProcessImage(img) {
  const wrapper = el("div", "ocr-pending");
  const badge = makeBadge("OCR..");
  img.replaceWith(wrapper);
  wrapper.append(img, badge);
  try {
    const container = await overlayImage(img, {
      lang: options.ocrLang || "eng",
      onProgress: (m) => {
        if (m && typeof m.progress === "number") badge.textContent = `OCR ${Math.round(m.progress * 100)}%`;
      },
    });
    wrapper.replaceWith(container);
  } catch (err) {
    console.warn("OCR failed for image", err);
    wrapper.replaceWith(img); // restore the plain image
  } finally {
    ocrDone += 1;
    ocrUpdateStatus();
  }
}

// Register every <img> under `root` for lazy OCR. No-op when OCR is off.
function registerImagesForOcr(root) {
  if (!options.ocrImages) return;
  const imgs = root.querySelectorAll("img");
  if (!imgs.length) return;
  ensureOcrCss();
  const obs = getOcrObserver();
  for (const img of imgs) {
    if (ocrQueued.has(img)) continue;
    ocrQueued.add(img);
    ocrTotal += 1;
    obs.observe(img);
  }
  ocrUpdateStatus();
}

// Extract raster images from a PDF page (scanned pages fall back to a full-page raster),
// append them to the page section as <img>, and register them for lazy OCR. Must run
// before page.cleanup(). No-op unless it finds usable images.
async function appendPdfImages(page, section, pageChars) {
  let imgs = [];
  try {
    imgs = await extractPageImages(page);
    if (!imgs.length && pageChars < 20) {
      const raster = await rasterizePage(page);
      if (raster) imgs = [raster];
    }
  } catch (err) {
    console.warn("PDF image extraction failed", err);
    return;
  }
  if (!imgs.length) return;
  for (const im of imgs) {
    const url = URL.createObjectURL(im.blob);
    pdfImageUrls.push(url);
    const imgEl = el("img");
    imgEl.src = url;
    section.append(imgEl);
  }
  registerImagesForOcr(section);
}

// ---- Notices / fallbacks ---------------------------------------------------
function showNotice(titleText, bodyNodes) {
  $("btn-save-html").classList.add("hidden"); // nothing valid to save as HTML
  const content = $("content");
  content.replaceChildren();
  const box = el("div", "notice");
  const h = el("h1");
  h.textContent = titleText;
  box.append(h);
  for (const n of bodyNodes) box.append(n);
  content.append(box);
  hideStatus();
}

function para(text) {
  const p = el("p");
  p.style.textIndent = "0";
  p.textContent = text;
  return p;
}

function originalButton(label = "Open original PDF") {
  const b = el("button");
  b.textContent = label;
  b.addEventListener("click", openOriginal);
  return b;
}

// ---- Open the untouched PDF (bypass interception for this tab) --------------
async function openOriginal() {
  if (!isSafePdfUrl(fileUrl)) return;
  try {
    const tab = await chrome.tabs.getCurrent();
    await chrome.runtime.sendMessage({ type: "open-original", url: fileUrl, tabId: tab && tab.id });
  } catch {
    // Last resort: navigate directly. Interception may re-catch it, but better
    // than a dead button.
    location.href = fileUrl;
  }
}

// ---- Download / save -------------------------------------------------------
// Two toolbar actions. "File" downloads the untouched source bytes. "HTML" saves the
// current on-screen view as a self-contained .html - and because it serializes the
// *live* #content, if the reader translated the page with Chrome's built-in translator
// (which rewrites the DOM in place), the saved file carries that translation. There is
// no API to trigger that translation from here: the user does it, we capture the result.
let originalBlob = null;   // source bytes as a Blob/File (browser-backed, no extra copy)
let originalName = "document";

function setOriginalDownload(blob, name) {
  originalBlob = blob || null;
  originalName = safeBase(name);
  $("btn-save-src").classList.toggle("hidden", !originalBlob);
}

function triggerDownload(href, filename) {
  const a = el("a");
  a.href = href;
  a.download = filename || "download";
  a.rel = "noopener";
  a.style.display = "none";
  document.body.append(a);
  a.click();
  a.remove();
}

// Strip characters a filesystem rejects and cap the length; never returns "".
function safeBase(name) {
  const clean = String(name || "").replace(/[\\/:*?"<>|\r\n\t]+/g, "_").replace(/\s+/g, " ").trim();
  return clean.slice(0, 120) || "document";
}

function filenameFromUrl(url) {
  try {
    return decodeURIComponent(new URL(url).pathname.split("/").pop() || "");
  } catch {
    return "";
  }
}

// Download the exact bytes the viewer opened. Works for URL-loaded documents and for
// files chosen via the picker (we keep the File/Blob, so there is no re-fetch).
function downloadOriginal() {
  if (!originalBlob) return;
  const url = URL.createObjectURL(originalBlob);
  triggerDownload(url, originalName);
  setTimeout(() => URL.revokeObjectURL(url), 10000);
}

// Save the current view as a standalone HTML file: captures translated text when the
// page has been translated in place, inlines blob: images as data URIs, and unwraps the
// translator's <font> wrappers so the result is portable and clean.
async function downloadHtml() {
  const content = $("content");
  if (!content || !content.children.length) return;

  const imgMap = buildImageDataMap(content); // read decoded live images once
  const clone = content.cloneNode(true);
  unwrapTranslateFonts(clone);
  applyImageDataMap(clone, imgMap);

  const title = document.title || "document";
  const theme = document.documentElement.getAttribute("data-theme") || "light";
  const lang = document.documentElement.lang || "";
  const css = await collectExportCss(clone);
  const styleVars = [
    cssVar("--reader-size"),
    cssVar("--reader-font"),
  ].filter(Boolean).join(";");

  const doc = buildExportHtml({ title, theme, lang, styleVars, css, body: clone.outerHTML });
  const blob = new Blob([doc], { type: "text/html;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  triggerDownload(url, `${safeBase(title)}.html`);
  setTimeout(() => URL.revokeObjectURL(url), 10000);
}

function cssVar(name) {
  const v = document.documentElement.style.getPropertyValue(name).trim();
  return v ? `${name}:${v}` : "";
}

// Rasterize every live blob: image to a data: URI, keyed by src. Live images are already
// decoded, so naturalWidth/Height are valid (a fresh clone's would be 0). http(s)/data
// images are left untouched - they are already portable.
function buildImageDataMap(root) {
  const map = new Map();
  for (const img of root.querySelectorAll("img")) {
    const src = img.getAttribute("src") || "";
    if (!src || map.has(src) || !/^blob:/i.test(src)) continue;
    try {
      const w = img.naturalWidth, h = img.naturalHeight;
      if (!w || !h) { map.set(src, null); continue; }
      const c = el("canvas");
      c.width = w;
      c.height = h;
      c.getContext("2d").drawImage(img, 0, 0);
      map.set(src, c.toDataURL("image/jpeg", 0.85));
    } catch {
      map.set(src, null); // tainted or too large - dropped below
    }
  }
  return map;
}

function applyImageDataMap(clone, map) {
  for (const img of clone.querySelectorAll("img")) {
    const src = img.getAttribute("src") || "";
    if (map.has(src)) {
      const data = map.get(src);
      if (data) { img.setAttribute("src", data); img.removeAttribute("srcset"); }
      else img.remove();
    } else if (/^blob:/i.test(src)) {
      img.remove(); // an unmapped blob would be a dead link outside this tab
    }
  }
}

// Chrome's built-in translator wraps translated runs in <font> tags. Unwrap them so the
// export keeps the translated text without the translator's scaffolding.
function unwrapTranslateFonts(root) {
  for (const font of root.querySelectorAll("font")) {
    const parent = font.parentNode;
    if (!parent) continue;
    while (font.firstChild) parent.insertBefore(font.firstChild, font);
    parent.removeChild(font);
  }
}

async function fetchText(url) {
  try {
    const r = await fetch(url);
    return r.ok ? await r.text() : "";
  } catch {
    return "";
  }
}

// Inline the same stylesheets the viewer uses so the saved file reads identically.
async function collectExportCss(clone) {
  const parts = [await fetchText(chrome.runtime.getURL("src/viewer.css"))];
  if (clone.querySelector(".ocr-overlay")) {
    parts.push(await fetchText(chrome.runtime.getURL("src/ocr-overlay.css")));
  }
  return parts.filter(Boolean).join("\n\n");
}

function escapeHtml(s) {
  return String(s).replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));
}

function buildExportHtml({ title, theme, lang, styleVars, css, body }) {
  const attrs = `lang="${escapeHtml(lang)}" data-theme="${escapeHtml(theme)}"` +
    (styleVars ? ` style="${escapeHtml(styleVars)}"` : "");
  return `<!DOCTYPE html>
<html ${attrs}>
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>${escapeHtml(title)}</title>
<style>
${css}
</style>
</head>
<body>
${body}
</body>
</html>
`;
}

// ---- TOC -------------------------------------------------------------------
function renderToc(entries) {
  const tree = $("toc-tree");
  tree.replaceChildren();
  if (!entries || entries.length === 0) {
    $("btn-toc").classList.add("hidden");
    return;
  }
  tree.append(buildTocList(entries));
}

function buildTocList(entries) {
  const ul = el("ul");
  for (const e of entries) {
    const li = el("li");
    const hasKids = e.children && e.children.length > 0;
    if (hasKids) {
      const toggle = el("span", "toc-toggle");
      toggle.textContent = "▾"; // down triangle
      toggle.addEventListener("click", () => {
        li.classList.toggle("collapsed");
        toggle.textContent = li.classList.contains("collapsed") ? "▸" : "▾";
      });
      li.append(toggle);
    }
    if (e.anchor != null) {
      // EPUB entry: scroll to an element id in the combined document.
      const a = el("a");
      a.textContent = e.title || "Section";
      a.href = `#${e.anchor}`;
      a.addEventListener("click", (ev) => {
        ev.preventDefault();
        scrollToAnchor(e.anchor);
      });
      li.append(a);
    } else if (e.page != null) {
      const a = el("a");
      a.textContent = e.title || `Page ${e.page}`;
      a.href = `#page-${e.page}`;
      a.addEventListener("click", (ev) => {
        ev.preventDefault();
        scrollToPage(e.page);
      });
      li.append(a);
    } else {
      const span = el("span");
      span.textContent = e.title;
      li.append(span);
    }
    if (hasKids) li.append(buildTocList(e.children));
    ul.append(li);
  }
  return ul;
}

function scrollToPage(n) {
  // Works for both PDF (id="page-N") and EPUB (id="epub-sec-i") sections, which
  // both carry data-page as the 1-based navigation index.
  const sec = document.querySelector(`#content section[data-page="${n}"]`);
  if (sec) sec.scrollIntoView({ behavior: "smooth", block: "start" });
}

function scrollToAnchor(id) {
  const target = document.getElementById(id);
  if (target) target.scrollIntoView({ behavior: "smooth", block: "start" });
}

// ---- Page rendering --------------------------------------------------------
function renderBlocks(section, blocks) {
  let firstP = true;
  for (const b of blocks) {
    const node = el(b.tag);
    if (b.tag === "p" && firstP) { node.classList.add("first"); firstP = false; }
    node.textContent = b.text;
    section.append(node);
  }
}

// Yield to the event loop so the UI stays responsive and rendered pages paint
// progressively for large PDFs.
const yieldToUI = () => new Promise((r) => setTimeout(r, 0));

// ---- Main ------------------------------------------------------------------
async function main() {
  await loadPrefs();
  applyPrefs();
  wireToolbar();

  // The viewer fetches arbitrary URLs with the extension's host access, so it must
  // only run as a top-level page. Refuse to run framed to close an SSRF-style
  // vector where a page iframes the viewer pointed at a URL of its choosing.
  if (window.top !== window.self) {
    showNotice("Cannot run in a frame", [para("Open this document in a top-level tab.")]);
    return;
  }

  if (!fileUrl) {
    showNotice("Open a document", [
      para("Pick a local document to read here (PDF, EPUB, MOBI, AZW3, FB2, RTF, TXT, Markdown, HTML) - or an image (PNG, JPEG, GIF, BMP, WebP) to OCR into translatable text. Opening a document link loads it here too - the extension is helpful like that."),
      filePickerButton(),
    ]);
    return;
  }

  if (!isSafePdfUrl(fileUrl)) {
    showNotice("Unsupported URL", [
      para("Only http(s) and local file documents can be opened from a URL."),
      filePickerButton(),
    ]);
    return;
  }

  await loadUrl(fileUrl);
}

function isFileUrl(url) {
  return /^file:/i.test(url);
}

// loadUrl downloads a PDF by URL and renders it. On failure it offers the local
// file picker (which needs no host access and no file-URL toggle) as a fallback.
async function loadUrl(url) {
  setStatus("Downloading document..");
  let data;
  try {
    const resp = await fetch(url);
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    const blob = await resp.blob();
    setOriginalDownload(blob, filenameFromUrl(url)); // keep a downloadable copy (browser-backed)
    data = await blob.arrayBuffer();
  } catch (err) {
    showNotice("Couldn't load this document", [
      para("The file could not be downloaded by the extension."),
      para(isFileUrl(url)
        ? 'For local files, use "Open a file" below, or enable "Allow access to file URLs" for this extension in chrome://extensions.'
        : `Reason: ${err.message}`),
      filePickerButton(),
      originalButton(),
    ]);
    return;
  }
  await loadFromData(data, fileTitle(url), url);
}

// loadFromData routes already-fetched bytes (URL fetch or local file picker) to
// the right reader via detectFormat (byte signature first, then the source name's
// extension). Unknown types fall through to the PDF path, whose error handling
// reports an unreadable file clearly.
async function loadFromData(data, title, name) {
  teardownCurrent();
  switch (detectFormat(data, name)) {
    case "epub": await loadEpubData(data, title); return;
    case "pdf": await loadPdfData(data, title); return;
    case "txt": await loadBook(data, title, parseText, "Reading text.."); return;
    case "rtf": await loadBook(data, title, parseRtf, "Reading RTF.."); return;
    case "html": await loadBook(data, title, parseHtml, "Reading HTML.."); return;
    case "md": await loadBook(data, title, parseMarkdown, "Reading Markdown.."); return;
    case "fb2": await loadBook(data, title, parseFb2, "Reading FB2.."); return;
    case "mobi": await loadBook(data, title, parseEbook, "Reading e-book.."); return;
    case "image": await loadImageData(data, title, imageMime(data, name)); return;
    default: await loadPdfData(data, title);
  }
}

// loadBook is the generic intake the non-PDF/EPUB parsers reuse: set status, hide
// the PDF-only Original button, parse the bytes into the shared book shape, and
// render it. On a parse error it shows a notice with the file picker.
async function loadBook(data, title, parseFn, statusLabel) {
  $("doc-title").textContent = title;
  document.title = title;
  $("btn-original").classList.add("hidden");
  $("status").classList.remove("done");
  setStatus(statusLabel);
  setProgress(0.1);
  let book;
  try {
    book = await parseFn(data);
  } catch (err) {
    showNotice("Couldn't open this file", [
      para("The file may be corrupt or not a supported document."),
      para(err && err.message ? `Details: ${err.message}` : ""),
      filePickerButton(),
    ]);
    return;
  }
  renderBook(book, title);
}

// loadImageData renders a standalone image the user opened directly and OCRs it into
// translatable text plates (the same overlay unit the right-click image page and the
// in-document image OCR use). OCR runs unconditionally here, independent of the "Use OCR
// for images" toggle: opening a bare image is itself the explicit request to read its
// text, and without OCR there is nothing for the browser translator to work on.
async function loadImageData(data, title, mime) {
  $("doc-title").textContent = title;
  document.title = title;
  $("btn-original").classList.add("hidden"); // no "native viewer" concept for a picture
  $("status").classList.remove("done");
  $("page-total").textContent = "/ 1";
  ensureOcrCss();

  const content = $("content");
  content.replaceChildren();
  const url = URL.createObjectURL(new Blob([data], mime ? { type: mime } : undefined));
  pdfImageUrls.push(url); // revoked on the next teardownCurrent()

  const lang = options.ocrLang || "eng";
  setStatus("Recognizing text..");
  setProgress(0.1);
  try {
    const container = await overlayImage(url, {
      lang,
      onProgress: (m) => { if (m && typeof m.progress === "number") setProgress(m.progress); },
    });
    content.append(container);
    applyLang(ocrLangToHtmlLang(lang));
    if (container.classList.contains("ocr-empty")) {
      container.append(makeBadge("No text found"));
      setStatus("No text found");
    } else {
      setStatus('Done - use the browser\'s "Translate page"');
    }
  } catch (err) {
    showNotice("Couldn't read this image", [
      para(err && err.message ? `Details: ${err.message}` : "The image could not be processed."),
      filePickerButton(),
    ]);
    return;
  }
  $("btn-save-html").classList.remove("hidden");
  setProgress(1);
  setTimeout(hideStatus, 1400);
}

// loadPdfData runs the PDF.js + reflow pipeline over PDF bytes.
async function loadPdfData(data, title) {
  $("doc-title").textContent = title;
  document.title = title;
  $("btn-original").classList.remove("hidden");
  $("status").classList.remove("done");
  setProgress(0.05);

  let pdf;
  try {
    const task = pdfjsLib.getDocument({
      data,
      isEvalSupported: false,
      cMapUrl: VENDOR.cMapUrl,
      cMapPacked: true,
      standardFontDataUrl: VENDOR.standardFontDataUrl,
    });
    task.onProgress = (p) => {
      if (p && p.total) setProgress(Math.min(0.15, (p.loaded / p.total) * 0.15));
    };
    task.onPassword = (updatePassword, reason) => askPassword(updatePassword, reason);
    pdf = await task.promise;
  } catch (err) {
    handleLoadError(err);
    return;
  }

  await renderDocument(pdf, title);
}

// openFilePicker reads a user-chosen local PDF via the OS file dialog. This needs
// no host permission and no "Allow access to file URLs" toggle, and - because the
// file path never appears in any URL - it is immune to URL-based content blockers.
function openFilePicker() {
  const input = $("file-input");
  input.value = "";
  input.onchange = async () => {
    const f = input.files && input.files[0];
    if (!f) return;
    $("content").replaceChildren();
    $("status").classList.remove("done");
    setStatus("Reading file..");
    try {
      const data = await f.arrayBuffer();
      setOriginalDownload(f, f.name); // the picked File is itself a downloadable Blob
      await loadFromData(data, f.name.replace(/\.(pdf|epub|txt|rtf|html?|md|fb2|mobi|azw3|png|jpe?g|gif|bmp|webp)$/i, ""), f.name);
    } catch (err) {
      showNotice("Couldn't read the file", [para(err.message || String(err)), filePickerButton()]);
    }
  };
  input.click();
}

function filePickerButton(label = "Open a document") {
  const b = el("button");
  b.textContent = label;
  b.addEventListener("click", openFilePicker);
  return b;
}

function handleLoadError(err) {
  const name = err && err.name;
  if (name === "PasswordException") {
    showNotice("Password required", [
      para("This PDF is password-protected and the password was not provided."),
      originalButton(),
    ]);
    return;
  }
  showNotice("Couldn't open this PDF", [
    para("The file may be corrupt, truncated, or in an unsupported format."),
    para(err && err.message ? `Details: ${err.message}` : ""),
    originalButton(),
  ]);
}

// Password prompt: PDF.js calls back with updatePassword(pw); we render an inline
// form and resolve it from the input.
function askPassword(updatePassword, reason) {
  const need = reason === pdfjsLib.PasswordResponses?.INCORRECT_PASSWORD;
  const input = el("input");
  input.type = "password";
  input.placeholder = "PDF password";
  const submit = el("button");
  submit.textContent = "Unlock";
  const row = el("div", "pw-row");
  row.append(input, submit);
  const body = [
    para(need ? "Incorrect password - try again." : "This PDF is protected. Enter its password to read it."),
    row,
  ];
  showNotice("Password required", body);
  $("status").classList.remove("done");
  const go = () => { if (input.value) updatePassword(input.value); setStatus("Unlocking.."); };
  submit.addEventListener("click", go);
  input.addEventListener("keydown", (e) => { if (e.key === "Enter") go(); });
  input.focus();
}

async function renderDocument(pdf, title) {
  const total = pdf.numPages;
  $("page-total").textContent = `/ ${total}`;
  $("page-jump").max = String(total);

  // Sample early pages for language detection before rendering everything.
  setStatus("Detecting language..");
  const sampleText = await collectSample(pdf, Math.min(total, 5));
  await setDocumentLang(pdf, sampleText);

  // TOC from the outline.
  let toc = [];
  try { toc = await buildToc(pdf); } catch { toc = []; }
  renderToc(toc);

  const content = $("content");
  content.replaceChildren();

  let totalChars = 0;
  let pagesWithText = 0;
  for (let n = 1; n <= total; n++) {
    let blocks = [];
    let page = null;
    try {
      page = await pdf.getPage(n);
      const viewport = page.getViewport({ scale: 1 });
      const tc = await page.getTextContent();
      blocks = reflowPage(tc, viewport);
    } catch {
      blocks = [];
    }

    const section = el("section");
    section.id = `page-${n}`;
    section.dataset.page = String(n);
    if (n > 1) content.append(el("hr", "page-sep"));
    const label = el("div", "page-label");
    label.textContent = `Page ${n}`;
    section.append(label);
    renderBlocks(section, blocks);

    let pageChars = 0;
    for (const b of blocks) pageChars += b.text.length;

    // Image OCR must run before cleanup (it needs the live page). No-op when off.
    if (page && options.ocrImages) await appendPdfImages(page, section, pageChars);
    if (page) { try { page.cleanup(); } catch { /* ignore */ } }

    content.append(section);

    totalChars += pageChars;
    if (pageChars >= 20) pagesWithText++;

    setProgress(0.15 + 0.85 * (n / total));
    setStatus(`Rendering page ${n} / ${total}`);
    if (n % 4 === 0) await yieldToUI();
  }

  finishDocument(total, totalChars, pagesWithText);
}

// ---- EPUB ------------------------------------------------------------------
// EPUB content is already semantic XHTML, so there is no reflow step: epub.js
// returns chapters as sanitized DOM fragments (images -> blob: URLs, links ->
// in-page anchors) that we drop into the same #content / TOC / page UI the PDF
// path uses. Each spine document is one "page" for navigation.
async function loadEpubData(data, title) {
  $("doc-title").textContent = title;
  document.title = title;
  // No "native viewer" exists for EPUB; hide the PDF-only Original button.
  $("btn-original").classList.add("hidden");
  $("status").classList.remove("done");
  setStatus("Reading EPUB..");
  setProgress(0.1);

  let book;
  try {
    book = await loadEpub(data);
  } catch (err) {
    showNotice("Couldn't open this EPUB", [
      para("The file may be corrupt or not a valid EPUB."),
      para(err && err.message ? `Details: ${err.message}` : ""),
      filePickerButton(),
    ]);
    return;
  }
  renderBook(book, title);
}

function renderBook(book, fallbackTitle) {
  if (book.revoke) revokeCurrent = book.revoke;
  const title = book.title || fallbackTitle;
  $("doc-title").textContent = title;
  document.title = title;

  applyLang(book.lang || detectLang(book.sampleText));

  const total = book.sections.length;
  $("page-total").textContent = `/ ${total}`;
  $("page-jump").max = String(total);

  renderToc(book.toc);

  const content = $("content");
  content.replaceChildren();

  let totalChars = 0;
  book.sections.forEach((s, i) => {
    const n = i + 1;
    const section = el("section");
    section.id = s.id;
    section.dataset.page = String(n);
    if (i > 0) content.append(el("hr", "page-sep"));
    const label = el("div", "page-label");
    label.textContent = s.label || `Section ${n}`;
    section.append(label);
    totalChars += (s.frag.textContent || "").length; // read before append empties it
    section.append(s.frag);
    content.append(section);
    registerImagesForOcr(section); // lazy image OCR (no-op when options.ocrImages is off)
    setProgress(0.1 + 0.9 * (n / total));
  });

  if (totalChars === 0) {
    const banner = el("div", "notice");
    const h = el("h1");
    h.textContent = "Little or no text found";
    banner.append(
      h,
      para("This EPUB has no extractable text (it may be image-only). Native page-translate needs actual text, not a pretty picture of it."),
      filePickerButton(),
    );
    content.prepend(banner);
  }
  $("btn-save-html").classList.remove("hidden");
  setProgress(1);
  setStatus(`Done - ${total} ${total === 1 ? "section" : "sections"}`);
  setTimeout(hideStatus, 1200);
}

function finishDocument(total, totalChars, pagesWithText) {
  // Scanned / image-only heuristic: a large majority of pages have (almost) no
  // extractable text. Using per-page density rather than an absolute character
  // floor avoids mislabeling a legitimately short one-page document.
  const mostlyEmpty = total > 1 && pagesWithText / total < 0.3;
  if (totalChars === 0 || mostlyEmpty) {
    const content = $("content");
    const banner = el("div", "notice");
    const h = el("h1");
    h.textContent = "Little or no text found";
    banner.append(
      h,
      para("This looks like a scanned or image-only PDF, so there is little text to translate - the browser translates words, not pixels."),
      para("To translate scanned pages, run them through OCR first (e.g. the doc-html-translate desktop app), then reopen the result."),
      originalButton(),
    );
    content.prepend(banner);
  }
  $("btn-save-html").classList.remove("hidden");
  setProgress(1);
  setStatus(`Done - ${total} pages`);
  setTimeout(hideStatus, 1200);
}

async function collectSample(pdf, pages) {
  let text = "";
  for (let n = 1; n <= pages && text.length < 8000; n++) {
    try {
      const page = await pdf.getPage(n);
      const tc = await page.getTextContent();
      text += tc.items.map((i) => (typeof i.str === "string" ? i.str : "")).join(" ") + " ";
      page.cleanup();
    } catch { /* skip */ }
  }
  return text;
}

async function setDocumentLang(pdf, sampleText) {
  // Priority: explicit options hint -> PDF /Lang metadata -> text heuristic.
  let lang = "";
  if (options.sourceLang && options.sourceLang !== "auto") {
    lang = normalizeLangTag(options.sourceLang);
  }
  if (!lang) {
    try {
      const meta = await pdf.getMetadata();
      const raw = (meta && meta.info && (meta.info.Language || meta.info.Lang)) || "";
      lang = normalizeLangTag(raw);
    } catch { /* ignore */ }
  }
  if (!lang) lang = detectLang(sampleText);
  applyLang(lang);
}

// applyLang sets <html lang> and a content-language meta so Chrome offers
// "Translate page" with the right source language. Shared by the PDF and EPUB paths.
function applyLang(lang) {
  if (!lang) return;
  document.documentElement.lang = lang;
  let metaTag = document.querySelector('meta[http-equiv="content-language"]');
  if (!metaTag) {
    metaTag = el("meta");
    metaTag.httpEquiv = "content-language";
    document.head.append(metaTag);
  }
  metaTag.content = lang;
}

// ---- Toolbar ---------------------------------------------------------------
function wireToolbar() {
  $("btn-toc").addEventListener("click", () => {
    $("toc").classList.toggle("hidden");
  });
  $("btn-font-inc").addEventListener("click", () => {
    prefs.size = Math.min(40, prefs.size + 1);
    applyPrefs(); savePrefs();
  });
  $("btn-font-dec").addEventListener("click", () => {
    prefs.size = Math.max(12, prefs.size - 1);
    applyPrefs(); savePrefs();
  });
  $("sel-family").addEventListener("change", (e) => {
    prefs.family = e.target.value; applyPrefs(); savePrefs();
  });
  $("sel-theme").addEventListener("change", (e) => {
    prefs.theme = e.target.value; applyPrefs(); savePrefs();
  });
  $("page-jump").addEventListener("change", (e) => {
    const n = parseInt(e.target.value, 10);
    if (n >= 1) scrollToPage(n);
  });
  $("btn-open").addEventListener("click", openFilePicker);
  $("btn-original").addEventListener("click", openOriginal);
  $("btn-save-src").addEventListener("click", downloadOriginal);
  $("btn-save-html").addEventListener("click", downloadHtml);

  // Keep the page-jump box in sync with scroll position.
  let ticking = false;
  document.addEventListener("scroll", () => {
    if (ticking) return;
    ticking = true;
    requestAnimationFrame(() => {
      ticking = false;
      const sections = document.querySelectorAll("#content section");
      const mid = window.scrollY + window.innerHeight / 3;
      for (const s of sections) {
        if (s.offsetTop <= mid) $("page-jump").value = s.dataset.page;
        else break;
      }
    });
  }, { passive: true });
}

main();
