// viewer.js - orchestrator for the reflow viewer.
//
// Flow: read ?file= -> load the PDF with PDF.js -> detect the source language and
// set <html lang> (so Chrome offers "Translate page") -> build the TOC from the
// outline -> reflow page text into clean <p>/<h2>/<h3> and insert it into the DOM.
// Text is never hidden or unloaded once rendered, so native translate sees every
// page the reader has reached, including whatever is off-screen. PDFs longer than
// PAGE_CHUNK pages are rendered forward in chunks as the reader approaches the edge
// (see "Lazy page rendering"); shorter ones land in a single pass, as before.

import * as pdfjsLib from "../vendor/pdf.mjs";
import { reflowPage } from "./reflow.js";
import { buildToc } from "./toc.js";
import { detectLang, normalizeLangTag } from "./lang.js";
import { t, initI18n, applyI18n, loadMessages, uiLang } from "./i18n.js";
import { loadEpub } from "./epub.js";
import { parseText } from "./txt.js";
import { parseRtf } from "./rtf.js";
import { parseHtml } from "./html.js";
import { parseMarkdown } from "./md.js";
import { parseFb2 } from "./fb2.js";
import { parseEbook, isMobiBytes } from "./ebook.js";
import { parseComic, DesktopOnlyError } from "./comic.js";
import { overlayImage, makeBadge, ocrLangToHtmlLang } from "./ocr-overlay.js";
import { langLabel } from "./ocr-lang.js";
import { extractPageImages, rasterizePage } from "./pdf-images.js";
import { DEFAULT_OPTIONS } from "./defaults.js";
import { recordRun } from "./diagnostics.js";

pdfjsLib.GlobalWorkerOptions.workerSrc = chrome.runtime.getURL("vendor/pdf.worker.mjs");

// From pdfjs 5 the JBIG2 / JPEG2000 decoders and the QCMS colour engine are WASM
// modules fetched from wasmUrl at runtime, and ICC profiles come from iccUrl. Scanned
// PDFs are JBIG2/JPX, so these are not optional extras - without them the viewer fails
// on exactly the documents it exists for. Vendored by build.mjs.
const VENDOR = {
  cMapUrl: chrome.runtime.getURL("vendor/cmaps/"),
  standardFontDataUrl: chrome.runtime.getURL("vendor/standard_fonts/"),
  wasmUrl: chrome.runtime.getURL("vendor/wasm/"),
  iccUrl: chrome.runtime.getURL("vendor/iccs/"),
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
const FORMAT_EXT = { pdf: "pdf", epub: "epub", txt: "txt", rtf: "rtf", htm: "html", html: "html", md: "md", markdown: "md", fb2: "fb2", mobi: "mobi", azw3: "mobi", png: "image", jpg: "image", jpeg: "image", gif: "image", bmp: "image", webp: "image", cbz: "comic", cbr: "comic", cb7: "comic", cbt: "comic" };

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
  // Both EPUB and CBZ are ZIP (PK..). The signature alone cannot tell them apart, so the
  // filename extension breaks the tie: a .cbz is a comic, anything else ZIP is an EPUB.
  // This keeps the EPUB hot path (a .epub, or a ZIP with no comic extension) unchanged.
  if (b[0] === 0x50 && b[1] === 0x4b && b[2] === 0x03 && b[3] === 0x04) {
    return FORMAT_EXT[fileExt(name)] === "comic" ? "comic" : "epub"; // PK..
  }
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
  // Bumping the generation strands any chunk still rendering from the old document,
  // so it cannot insert its pages into the new one.
  docGen++;
  if (chunkObserver) { chunkObserver.disconnect(); chunkObserver = null; }
  if (pdfImageObserver) { pdfImageObserver.disconnect(); pdfImageObserver = null; }
  pdfImagesDeferred = 0;
  if (comicImageObserver) { comicImageObserver.disconnect(); comicImageObserver = null; }
  comicTotal = 0;
  comicRendered = 0;
  chunkPending = null;
  pdfDoc = null;
  pdfTotal = 0;
  pdfRendered = 0;
  ocrTotal = 0;
  ocrDone = 0;
  ocrWithText = 0;
  for (const url of pdfImageUrls) { try { URL.revokeObjectURL(url); } catch { /* ignore */ } }
  pdfImageUrls = [];
}

// ---- Preferences -----------------------------------------------------------
// size must match viewer.css's --reader-size fallback, which styles the document before
// this runs. A+/A- move it and persist; nothing is stored until the reader asks for a
// change, so this default reaches everyone who never expressed a preference.
// ocrLayer: whether the recognized-text plates are shown over the artwork. On by default -
// the plates are what makes a comic or a scan translatable - but a reader looking at the art
// wants them out of the way, so the choice persists like the theme does. Mirrors the app's
// dht_ocr toggle (docs/PARITY.md).
const DEFAULT_PREFS = { size: 28, family: "serif", theme: null, ocrLayer: true };
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
  const ocrOn = prefs.ocrLayer !== false;
  document.documentElement.classList.toggle("ocr-layer-off", !ocrOn);
  $("btn-ocr").setAttribute("aria-pressed", ocrOn ? "true" : "false");
}

// revealOcrToggle shows the OCR layer control once the document actually has plates on it.
// Called after an overlay lands, because plates arrive lazily as images scroll into view -
// checking once at load would hide the control on every book.
function revealOcrToggle() {
  if (document.querySelector(".ocr-overlay")) $("grp-ocr").hidden = false;
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
let ocrWithText = 0; // recognized images that actually carried a plate - see ocrUpdateStatus
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

// docExtent says where the reader is in the document as a whole. Any other counter is
// about some slice of it, so this is what stops a number like "3/5" from being read as a
// statement about the book. Empty for formats with no page dimension of their own.
function docExtent() {
  if (!pdfTotal) return "";
  if (pdfRendered >= pdfTotal) return t("vExtentAll", "{1} pages", pdfTotal);
  return t("vExtentPartial", "pages 1-{1} of {2}", pdfRendered, pdfTotal);
}

// A page or two with nothing on them is ordinary - art panels carry no dialogue. A run of
// them with not one recognized word is the signature of the wrong recognition language, which
// is easy to hit here: OCR reads English by default while this viewer's readers mostly do not.
// The queue grows as the reader scrolls, so there is no end-of-document moment to report at
// (the desktop app has one, and says it there); this many empties in a row is the closest
// honest substitute, and it is low enough to reach on the first screen of a comic.
const OCR_EMPTY_RUN_HINT = 3;

function ocrUpdateStatus() {
  if (ocrTotal === 0) return;
  $("status").classList.remove("done");
  // The denominator is images found so far, not the document's - extraction is deferred,
  // so it climbs as the reader scrolls. Naming the unit and saying where we are in the
  // book keeps "OCR: 3/5" from looking like a claim that the book holds five of anything.
  const where = docExtent();
  setStatus(where
    ? t("vOcrStatusWhere", "OCR: {1}/{2} images - {3}", ocrDone, ocrTotal, where)
    : t("vOcrStatus", "OCR: {1}/{2} images", ocrDone, ocrTotal));
  setProgress(ocrDone / ocrTotal);
  if (ocrDone < ocrTotal) return;
  if (ocrWithText === 0 && ocrDone >= OCR_EMPTY_RUN_HINT) {
    const lang = options.ocrLang || "eng";
    setStatus(t("ocrNoTextLang", "No text found using {1} - if this page is in another language, pick it in the extension popup.", langLabel(lang)));
    setTimeout(hideStatus, 6000); // a sentence to read and act on, not a progress tick
    return;
  }
  setTimeout(hideStatus, 1000);
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
    if (!container.classList.contains("ocr-empty")) ocrWithText += 1;
    revealOcrToggle();
  } catch (err) {
    console.warn("OCR failed for image", err);
    wrapper.replaceWith(img); // restore the plain image
  } finally {
    ocrDone += 1;
    ocrUpdateStatus();
  }
}

// Register every <img> under `root` for lazy OCR. No-op when OCR is off, unless
// `force` is set - comics force OCR on regardless of the "Use OCR for images"
// toggle, because opening a comic is itself the request to read its bubbles (the
// same rationale as a standalone image).
function registerImagesForOcr(root, force = false) {
  if (!options.ocrImages && !force) return;
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

// ---- Deferred PDF image extraction -----------------------------------------
// Pulling the raster out of a page is expensive: pdf.js decodes the image, we draw it to
// a canvas and re-encode it, and a page with no image of its own gets rasterized whole.
// Doing that inline for every page made a scanned book take minutes to render - the very
// wait chunking exists to remove - while the *recognition* of those same images was
// already deferred to scroll. That split was incoherent: eager extraction bought nothing,
// because the plate that carries the readable text only ever arrived on scroll anyway.
//
// So extraction now rides the same trigger as OCR. This costs nothing in translation
// coverage (the plates were always going to appear late), and it means a page's raster is
// decoded only if the reader actually reaches it.
let pdfImageObserver = null;
let pdfImagesDeferred = 0;

// ---- Comic page state ------------------------------------------------------
// A comic archive renders like a scanned PDF: placeholder sections up front, each
// page's image inflated and inserted only as it scrolls near view (comicImageObserver),
// then OCR'd by the shared lazy-OCR observer. comicTotal/comicRendered drive the
// partial-save warning, mirroring the PDF counters.
let comicImageObserver = null;
let comicTotal = 0;
let comicRendered = 0;

function getPdfImageObserver() {
  if (pdfImageObserver) return pdfImageObserver;
  pdfImageObserver = new IntersectionObserver((entries, obs) => {
    for (const e of entries) {
      if (!e.isIntersecting) continue;
      obs.unobserve(e.target);
      extractSectionImages(e.target);
    }
    // A wide margin so a page's raster is decoded well before it is looked at: the work
    // is slow enough to be visible, and the reader is heading this way.
  }, { rootMargin: "1500px" });
  return pdfImageObserver;
}

// deferPageImages marks a rendered section as "images still to come" and reserves their
// space. No-op when OCR is off, which is also the only time page images are extracted.
function deferPageImages(section, pageNum, pageChars, width, height) {
  if (!options.ocrImages) return;
  section.dataset.pdfPage = String(pageNum);
  section.dataset.pdfChars = String(pageChars);
  pdfImagesDeferred++;
  // Reserve the page's own shape only for a page with no text. That is exactly when
  // appendPdfImages rasterizes the whole page, so the pending raster is known to fill the
  // column and the reserved box is the right one - the scanned-book case, where getting
  // it wrong would mean the document growing by a page-height under the reader's eyes on
  // every scroll. A text page's figures are small and unpredictable, and its text already
  // gives the section a height, so a guess there would be wrong in both directions.
  if (pageChars < 20 && width > 0 && height > 0) {
    const box = el("div", "pdf-page-pending");
    box.style.aspectRatio = `${width} / ${height}`;
    section.append(box);
  }
  getPdfImageObserver().observe(section);
}

// extractSectionImages runs the image pass for one section, re-opening its page (the
// render loop released it) and dropping the reserved box once the real images land.
async function extractSectionImages(section) {
  const pageNum = Number(section.dataset.pdfPage);
  if (!pdfDoc || !pageNum) return;
  delete section.dataset.pdfPage;
  const gen = docGen;
  const pageChars = Number(section.dataset.pdfChars) || 0;
  let page = null;
  try {
    page = await pdfDoc.getPage(pageNum);
    if (gen !== docGen) return;
    await appendPdfImages(page, section, pageChars);
  } catch {
    /* a page that will not yield its images just stays text-only */
  } finally {
    if (page) { try { page.cleanup(); } catch { /* ignore */ } }
    if (gen === docGen) section.querySelector(".pdf-page-pending")?.remove();
  }
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
    // Publish the intrinsic size so the browser reserves the box from the aspect ratio
    // before the blob decodes. Without it a blob: image is zero-height until decoded, so
    // the page would collapse and snap back the moment it loaded - which the reserved
    // box above exists to prevent.
    if (im.width > 0 && im.height > 0) {
      imgEl.width = im.width;
      imgEl.height = im.height;
    }
    imgEl.src = url;
    section.append(imgEl);
  }
  registerImagesForOcr(section);
}

// ---- Notices / fallbacks ---------------------------------------------------
// Every failure the viewer shows the user comes through here, so this is the one place the
// last error has to be recorded for a diagnostics report.
function showNotice(titleText, bodyNodes) {
  recordRun({ error: titleText });
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

function originalButton(label = t("vBtnOpenOriginalPdf", "Open original PDF")) {
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
  reportPartialSave();
}

// A chunk-rendered PDF holds only the pages the reader has reached, so the export is
// as long as the read. Say so instead of handing over a quietly truncated book.
// Rendering the remainder here would cost exactly the long freeze the chunking exists
// to avoid, and would bury untranslated pages under the ones Chrome already
// translated - so the honest partial file wins over the confusing complete one.
function reportPartialSave() {
  // A comic inflates its pages on scroll, so the export holds only the pages reached.
  if (comicTotal > 0) {
    if (comicRendered >= comicTotal) return;
    $("status").classList.remove("done");
    setStatus(t("vSavedComic", "Saved {1} of {2} pages - scroll further and save again to include more", comicRendered, comicTotal));
    setTimeout(hideStatus, 5000);
    return;
  }
  if (!pdfDoc || pdfRendered >= pdfTotal) return;
  $("status").classList.remove("done");
  setStatus(t("vSavedPdf", "Saved pages 1-{1} of {2} - scroll further and save again to include more", pdfRendered, pdfTotal));
  setTimeout(hideStatus, 5000);
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
      a.textContent = e.title || t("vTocSection", "Section");
      a.href = `#${e.anchor}`;
      a.addEventListener("click", (ev) => {
        ev.preventDefault();
        scrollToAnchor(e.anchor);
      });
      li.append(a);
    } else if (e.page != null) {
      const a = el("a");
      a.textContent = e.title || t("vPageN", "Page {1}", e.page);
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

async function scrollToPage(n) {
  // Works for both PDF (id="page-N") and EPUB (id="epub-sec-i") sections, which
  // both carry data-page as the 1-based navigation index. A PDF target past the
  // rendered edge is rendered on the way there.
  await ensurePageRendered(n);
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
// applyViewerChromeI18n translates the toolbar and the table-of-contents panel only. It must not
// touch <html lang> or the reflowed content: that attribute carries the *document's* language and
// is what makes Chrome offer "Translate page" - the whole point of this extension.
function applyViewerChromeI18n() {
  applyI18n(document.getElementById("toolbar"));
  applyI18n(document.getElementById("toc"));
  document.title = t("viewerTitle", document.title);
}

async function main() {
  // The chrome speaks the interface language; the document keeps its own <html lang>, which is
  // what makes the browser offer to translate it. applyI18n only touches the toolbar and the
  // panels, never the rendered document.
  await initI18n();
  await loadMessages(uiLang());
  applyViewerChromeI18n();

  await loadPrefs();
  applyPrefs();
  wireToolbar();

  // The viewer fetches arbitrary URLs with the extension's host access, so it must
  // only run as a top-level page. Refuse to run framed to close an SSRF-style
  // vector where a page iframes the viewer pointed at a URL of its choosing.
  if (window.top !== window.self) {
    showNotice(t("vFrameTitle", "Cannot run in a frame"), [para(t("vFrameBody", "Open this document in a top-level tab."))]);
    return;
  }

  if (!fileUrl) {
    showNotice(t("vOpenDocTitle", "Open a document"), [
      para(t("vOpenDocBody", "Pick a local document to read here (PDF, EPUB, MOBI, AZW3, FB2, RTF, TXT, Markdown, HTML) - or an image (PNG, JPEG, GIF, BMP, WebP) to OCR into translatable text. Opening a document link loads it here too - the extension is helpful like that.")),
      filePickerButton(),
    ]);
    return;
  }

  if (!isSafePdfUrl(fileUrl)) {
    showNotice(t("vUnsupportedUrlTitle", "Unsupported URL"), [
      para(t("vUnsupportedUrlBody", "Only http(s) and local file documents can be opened from a URL.")),
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
  // file URLs with a host (UNC, \\server\share) are unreachable for extensions:
  // file-scheme match patterns only cover empty-host URLs, so fetch() can never
  // be permitted. Fail fast with a targeted hint instead of a doomed download.
  if (/^file:\/\/[^/]/i.test(url)) {
    showNotice(t("vUncTitle", "Network paths are not supported"), [
      para(t("vUncBody", "Extensions cannot read network file paths (\\\\server\\share). Map the share to a drive letter and open it as a local file, or pick the file below.")),
      filePickerButton(),
      originalButton(t("vBtnBuiltinViewer", "Open in built-in viewer")),
    ]);
    return;
  }
  setStatus(t("vStatusDownloading", "Downloading document.."));
  let data;
  try {
    const resp = await fetch(url);
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    const blob = await resp.blob();
    setOriginalDownload(blob, filenameFromUrl(url)); // keep a downloadable copy (browser-backed)
    data = await blob.arrayBuffer();
  } catch (err) {
    showNotice(t("vLoadFailTitle", "Couldn't load this document"), [
      para(t("vLoadFailBody", "The file could not be downloaded by the extension.")),
      para(isFileUrl(url)
        ? t("vLoadFailFile", 'For local files, use "Open a file" below, or enable "Allow access to file URLs" for this extension in chrome://extensions.')
        : t("vReason", "Reason: {1}", err.message)),
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
// setPageTotal is the one place the page count reaches the chrome, so a reader added later
// cannot quietly leave the diagnostics record without one.
function setPageTotal(total) {
  $("page-total").textContent = `/ ${total}`;
  recordRun({ pages: total });
}

async function loadFromData(data, title, name) {
  teardownCurrent();
  const format = detectFormat(data, name);
  // The format id only - never the document's name, bytes or URL. See diagnostics.js.
  recordRun({ format });
  switch (format) {
    case "epub": await loadEpubData(data, title); return;
    case "pdf": await loadPdfData(data, title); return;
    case "txt": await loadBook(data, title, parseText, t("vStatusReadingText", "Reading text..")); return;
    case "rtf": await loadBook(data, title, parseRtf, t("vStatusReadingRtf", "Reading RTF..")); return;
    case "html": await loadBook(data, title, parseHtml, t("vStatusReadingHtml", "Reading HTML..")); return;
    case "md": await loadBook(data, title, parseMarkdown, t("vStatusReadingMd", "Reading Markdown..")); return;
    case "fb2": await loadBook(data, title, parseFb2, t("vStatusReadingFb2", "Reading FB2..")); return;
    case "mobi": await loadBook(data, title, parseEbook, t("vStatusReadingEbook", "Reading e-book..")); return;
    case "image": await loadImageData(data, title, imageMime(data, name)); return;
    case "comic": await loadComicData(data, title); return;
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
    showNotice(t("vOpenFileFailTitle", "Couldn't open this file"), [
      para(t("vCorruptBody", "The file may be corrupt or not a supported document.")),
      para(err && err.message ? t("vDetails", "Details: {1}", err.message) : ""),
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
  setPageTotal(1);
  ensureOcrCss();

  const content = $("content");
  content.replaceChildren();
  const url = URL.createObjectURL(new Blob([data], mime ? { type: mime } : undefined));
  pdfImageUrls.push(url); // revoked on the next teardownCurrent()

  const lang = options.ocrLang || "eng";
  setStatus(t("ocrProgress", "Recognizing text.."));
  setProgress(0.1);
  try {
    const container = await overlayImage(url, {
      lang,
      onProgress: (m) => { if (m && typeof m.progress === "number") setProgress(m.progress); },
    });
    content.append(container);
    applyLang(ocrLangToHtmlLang(lang));
    if (container.classList.contains("ocr-empty")) {
      // The badge stays short because it sits on the picture; the status line carries the
      // part that matters - which language was read. "No text found" is true about the data
      // that was loaded and reads as a verdict on the image, which is the wrong lesson when
      // the real answer is that an English recognizer was pointed at a Russian page.
      container.append(makeBadge(t("ocrNoText", "No text found")));
      setStatus(t("ocrNoTextLang", "No text found using {1} - if this page is in another language, pick it in the extension popup.", langLabel(lang)));
    } else {
      setStatus(t("ocrDone", 'Done - use the browser\'s "Translate page"'));
    }
  } catch (err) {
    showNotice(t("vImageFailTitle", "Couldn't read this image"), [
      para(err && err.message ? t("vDetails", "Details: {1}", err.message) : t("vImageFailBody", "The image could not be processed.")),
      filePickerButton(),
    ]);
    return;
  }
  $("btn-save-html").classList.remove("hidden");
  setProgress(1);
  setTimeout(hideStatus, 1400);
}

// comicLoaders maps a placeholder <section> to its lazy page loader until the page
// scrolls near view. A WeakMap so a section removed on teardown is collectable.
const comicLoaders = new WeakMap();

// loadComicData opens a comic archive (CBZ / CBT) and renders it page by page with
// forced OCR: each page image is inflated and inserted only as it nears the viewport,
// then OCR'd into translatable plates so the browser's "Translate page" reaches the
// speech bubbles. CBR/CB7 (RAR/7z) have no in-browser decoder and are declined with a
// notice pointing at the desktop app.
async function loadComicData(data, title) {
  $("doc-title").textContent = title;
  document.title = title;
  $("btn-original").classList.add("hidden"); // no "native viewer" concept for a comic
  $("status").classList.remove("done");
  setStatus(t("vStatusReadingComic", "Reading comic.."));
  setProgress(0.1);
  ensureOcrCss();

  let pages;
  try {
    pages = await parseComic(data);
  } catch (err) {
    if (err instanceof DesktopOnlyError) {
      showNotice(t("vComicAppTitle", "This comic needs the desktop app"), [
        para(err.message),
        para(t("vComicAppBody", "Get the free doc-html-translate app at https://serzhyale.github.io/doc-html-translate/ - it opens CBR and CB7 (with 7-Zip installed).")),
        filePickerButton(),
      ]);
    } else {
      showNotice(t("vComicFailTitle", "Couldn't open this comic"), [
        para(err && err.message ? t("vDetails", "Details: {1}", err.message) : t("vComicFailBody", "The archive may be corrupt or hold no page images.")),
        filePickerButton(),
      ]);
    }
    return;
  }
  renderComic(pages);
}

// renderComic lays out one placeholder section per page up front (so the scrollbar
// reflects the whole book immediately) and defers each page's image inflation and OCR
// to scroll via the comic image observer. Mirrors the scanned-PDF path.
function renderComic(pages) {
  applyLang(ocrLangToHtmlLang(options.ocrLang || "eng"));
  comicTotal = pages.length;
  comicRendered = 0;
  setPageTotal(comicTotal);
  $("page-jump").max = String(comicTotal);
  renderToc(null); // comics carry no authored table of contents

  const content = $("content");
  content.replaceChildren();
  const obs = getComicImageObserver();
  pages.forEach((pg, i) => {
    const n = i + 1;
    const section = el("section");
    section.id = `comic-page-${n}`;
    section.dataset.page = String(n);
    if (i > 0) content.append(el("hr", "page-sep"));
    // Reserve a page-shaped box so layout does not jump much when the real image lands;
    // a default portrait ratio is close enough for the moments before it decodes.
    const box = el("div", "comic-page-pending");
    box.style.aspectRatio = "2 / 3";
    section.append(box);
    comicLoaders.set(section, pg);
    content.append(section);
    obs.observe(section);
  });

  $("btn-save-html").classList.remove("hidden");
  setProgress(1);
  setStatus(comicTotal === 1
    ? t("vComicReadyOne", "Ready - 1 page, text is recognized as you scroll")
    : t("vComicReady", "Ready - {1} pages, text is recognized as you scroll", comicTotal));
  setTimeout(hideStatus, 1800);
}

function getComicImageObserver() {
  if (comicImageObserver) return comicImageObserver;
  comicImageObserver = new IntersectionObserver((entries, obs) => {
    for (const e of entries) {
      if (!e.isIntersecting) continue;
      obs.unobserve(e.target);
      insertComicPage(e.target);
    }
    // A wide margin so a page's image is inflated before it is looked at.
  }, { rootMargin: "1500px" });
  return comicImageObserver;
}

// insertComicPage inflates one page's bytes, inserts it as an <img>, drops the reserved
// box, and registers it for forced OCR. Guarded by docGen so a page still inflating from
// a torn-down comic cannot insert itself into the next document.
async function insertComicPage(section) {
  const pg = comicLoaders.get(section);
  if (!pg) return;
  comicLoaders.delete(section);
  const gen = docGen;
  let bytes;
  try {
    bytes = await pg.load();
  } catch (err) {
    console.warn("comic page load failed", err);
    return; // leave the reserved box; a page that will not inflate just stays blank
  }
  if (gen !== docGen) return;
  const url = URL.createObjectURL(new Blob([bytes], { type: pg.mime }));
  pdfImageUrls.push(url); // revoked on the next teardownCurrent()
  const img = el("img");
  img.addEventListener("load", () => { comicRendered++; }, { once: true });
  img.src = url;
  section.append(img);
  section.querySelector(".comic-page-pending")?.remove();
  registerImagesForOcr(section, true); // comics force OCR on
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
      wasmUrl: VENDOR.wasmUrl,
      iccUrl: VENDOR.iccUrl,
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
    setStatus(t("vStatusReadingFile", "Reading file.."));
    try {
      const data = await f.arrayBuffer();
      setOriginalDownload(f, f.name); // the picked File is itself a downloadable Blob
      await loadFromData(data, f.name.replace(/\.(pdf|epub|txt|rtf|html?|md|fb2|mobi|azw3|png|jpe?g|gif|bmp|webp)$/i, ""), f.name);
    } catch (err) {
      showNotice(t("vReadFileFailTitle", "Couldn't read the file"), [para(err.message || String(err)), filePickerButton()]);
    }
  };
  input.click();
}

function filePickerButton(label = t("vOpenDocTitle", "Open a document")) {
  const b = el("button");
  b.textContent = label;
  b.addEventListener("click", openFilePicker);
  return b;
}

function handleLoadError(err) {
  const name = err && err.name;
  if (name === "PasswordException") {
    showNotice(t("vPwdTitle", "Password required"), [
      para(t("vPwdMissing", "This PDF is password-protected and the password was not provided.")),
      originalButton(),
    ]);
    return;
  }
  showNotice(t("vPdfFailTitle", "Couldn't open this PDF"), [
    para(t("vPdfFailBody", "The file may be corrupt, truncated, or in an unsupported format.")),
    para(err && err.message ? t("vDetails", "Details: {1}", err.message) : ""),
    originalButton(),
  ]);
}

// Password prompt: PDF.js calls back with updatePassword(pw); we render an inline
// form and resolve it from the input.
function askPassword(updatePassword, reason) {
  const need = reason === pdfjsLib.PasswordResponses?.INCORRECT_PASSWORD;
  const input = el("input");
  input.type = "password";
  input.placeholder = t("vPwdPlaceholder", "PDF password");
  const submit = el("button");
  submit.textContent = t("vBtnUnlock", "Unlock");
  const row = el("div", "pw-row");
  row.append(input, submit);
  const body = [
    para(need
      ? t("vPwdIncorrect", "Incorrect password - try again.")
      : t("vPwdPrompt", "This PDF is protected. Enter its password to read it.")),
    row,
  ];
  showNotice(t("vPwdTitle", "Password required"), body);
  $("status").classList.remove("done");
  const go = () => { if (input.value) updatePassword(input.value); setStatus(t("vStatusUnlocking", "Unlocking..")); };
  submit.addEventListener("click", go);
  input.addEventListener("keydown", (e) => { if (e.key === "Enter") go(); });
  input.focus();
}

// ---- Lazy page rendering ---------------------------------------------------
// Long PDFs are rendered forward in chunks rather than in one pass. Reflowing a
// thousand pages takes minutes during which the tab looks hung - and worse, Chrome's
// translator snapshots the DOM, ships the text off and patches it back, so a render
// loop appending pages underneath pulls the rug out from under it: "Translate page"
// during a long render collapses.
//
// The window is deliberately large, because native translate only ever covers what was
// in the DOM at the moment it ran. Pages appended after that arrive untranslated, and
// the reader has to toggle translate off and on to pick them up. So every chunk
// boundary costs the reader an interruption, and the fix is fewer, bigger, faster
// chunks - not smaller ones. PAGE_CHUNK is the balance: big enough that boundaries are
// rare, small enough that the first pages are readable in seconds.
const PAGE_CHUNK = 100; // pages per chunk
const CHUNK_LEAD = 5;   // build the next chunk once this page before the edge is reached
// Pages in flight to the pdfjs worker at once. Fetching a page is a round trip to that
// worker, so awaiting them one at a time left both sides idle in turn - the main thread
// waiting on the worker, the worker waiting while the main thread built DOM. Keeping a
// few requests outstanding overlaps the two and bounds how many live pages are held.
const PAGE_LOOKAHEAD = 8;

let pdfDoc = null;    // live PDF.js document, held open for later chunks
let pdfTotal = 0;
let pdfRendered = 0;  // pages in the DOM; always the contiguous run 1..pdfRendered
let pdfChars = 0;
let pdfPagesWithText = 0;
let chunkPending = null;  // in-flight chunk - concurrent callers await this one
let chunkObserver = null; // watches the lead page of the rendered run
let docGen = 0;           // bumped per loaded document; strands chunks from the old one

async function renderDocument(pdf, title) {
  const total = pdf.numPages;
  setPageTotal(total);
  $("page-jump").max = String(total);

  // Sample early pages for language detection before rendering everything.
  setStatus(t("vStatusDetecting", "Detecting language.."));
  const sampleText = await collectSample(pdf, Math.min(total, 5));
  await setDocumentLang(pdf, sampleText);

  // TOC from the outline.
  let toc = [];
  try { toc = await buildToc(pdf); } catch { toc = []; }
  renderToc(toc);

  $("content").replaceChildren();

  pdfDoc = pdf;
  pdfTotal = total;
  pdfRendered = 0;
  pdfChars = 0;
  pdfPagesWithText = 0;

  await renderChunk();
  // Judge "scanned, image-only PDF" on the first chunk alone: up to PAGE_CHUNK pages
  // is a fair sample, and a scanned book is scanned throughout. Waiting for the whole
  // document would mean never showing the banner on the files that most need it.
  warnIfNoText();
  $("btn-save-html").classList.remove("hidden");
}

// renderChunk renders the next PAGE_CHUNK pages. Callers that race (the scroll
// sentinel, a page jump, a TOC click) share the in-flight promise instead of
// pushing a second chunk into the same range.
function renderChunk() {
  if (chunkPending) return chunkPending;
  if (!pdfDoc || pdfRendered >= pdfTotal) return Promise.resolve();
  const gen = docGen;
  const from = pdfRendered + 1;
  const to = Math.min(pdfTotal, pdfRendered + PAGE_CHUNK);
  const p = renderPages(from, to).finally(() => {
    if (gen !== docGen) return; // stranded by a new document, which owns the state now
    chunkPending = null;
    armChunkSentinel();
  });
  chunkPending = p;
  return p;
}

// fetchPage asks the pdfjs worker for one page and reflows its text. The live page is
// handed back too: the OCR image pass needs it, and it must be cleaned up afterwards.
// A page that fails to load resolves to no blocks rather than rejecting, so one bad
// page cannot take the chunk (or the lookahead window) down with it.
function fetchPage(n) {
  return pdfDoc.getPage(n).then(async (page) => {
    const viewport = page.getViewport({ scale: 1 });
    const tc = await page.getTextContent();
    // The page's own dimensions are kept for the deferred image pass, which reserves a
    // box of the right shape before the raster exists.
    return { page, blocks: reflowPage(tc, viewport), width: viewport.width, height: viewport.height };
  }).catch(() => ({ page: null, blocks: [], width: 0, height: 0 }));
}

// renderPages reflows [from..to] and inserts the result, streaming or in one shot.
//
// The first chunk streams, in batches, into an empty document: the reader watches pages
// arrive instead of staring at a blank screen, which matters most exactly when a page is
// slowest to build (image extraction with OCR on). Later chunks land mid-read, below the
// reader, where watching them arrive is worth nothing and a hundred separate layout
// passes are worth less than nothing - so they are built off-DOM and inserted once.
//
// This does not make them translated. Chrome's translator only covers what was in the
// DOM when it ran; pages appended afterwards stay in the source language until the
// reader toggles translate off and on. That is why PAGE_CHUNK is large and this function
// is worth keeping fast - each boundary is an interruption, so the goal is to have few
// of them, not to smooth them over.
async function renderPages(from, to) {
  const gen = docGen;
  const stream = from === 1;
  const content = $("content");
  // Appending a fragment moves its children out, leaving it empty and reusable, so
  // the same fragment serves both as the streaming batch and the one-shot buffer.
  const frag = document.createDocumentFragment();
  $("status").classList.remove("done");

  // Keep PAGE_LOOKAHEAD fetches outstanding, consumed strictly in page order. Refilling
  // right after taking one means the worker is reflowing the next pages while this one's
  // DOM is being built, instead of the two taking turns.
  const inflight = new Map();
  let nextFetch = from;
  const pump = () => {
    while (nextFetch <= to && inflight.size < PAGE_LOOKAHEAD) {
      inflight.set(nextFetch, fetchPage(nextFetch));
      nextFetch++;
    }
  };
  pump();

  for (let n = from; n <= to; n++) {
    if (gen !== docGen) return; // another document was opened - drop this chunk
    const { page, blocks, width, height } = await inflight.get(n);
    inflight.delete(n);
    pump();

    const section = el("section");
    section.id = `page-${n}`;
    section.dataset.page = String(n);
    if (n > 1) frag.append(el("hr", "page-sep"));
    const label = el("div", "page-label");
    label.textContent = t("vPageN", "Page {1}", n);
    section.append(label);
    renderBlocks(section, blocks);

    let pageChars = 0;
    for (const b of blocks) pageChars += b.text.length;

    // Images are pulled out later, on the same scroll trigger that drives OCR. No-op
    // when OCR is off - nothing extracts page images then anyway.
    deferPageImages(section, n, pageChars, width, height);
    if (page) { try { page.cleanup(); } catch { /* ignore */ } }

    frag.append(section);

    pdfChars += pageChars;
    if (pageChars >= 20) pdfPagesWithText++;

    setProgress(0.15 + 0.85 * (n / pdfTotal));
    setStatus(t("vStatusRendering", "Rendering page {1} / {2}", n, pdfTotal));
    if (n % 4 === 0) {
      if (stream) content.append(frag);
      await yieldToUI();
    }
  }

  if (gen !== docGen) return;
  content.append(frag); // the streaming remainder, or the whole chunk
  pdfRendered = to;
  reportRenderIdle();
}

// armChunkSentinel watches the page CHUNK_LEAD before the rendered edge: reaching it
// means the reader is close enough that the next chunk should already be building.
function armChunkSentinel() {
  if (chunkObserver) { chunkObserver.disconnect(); chunkObserver = null; }
  if (!pdfDoc || pdfRendered >= pdfTotal) return;
  const lead = Math.max(1, pdfRendered - CHUNK_LEAD);
  const sec = document.querySelector(`#content section[data-page="${lead}"]`);
  if (!sec) return;
  chunkObserver = new IntersectionObserver((entries) => {
    if (entries.some((e) => e.isIntersecting)) renderChunk();
  }, { rootMargin: "200px" });
  chunkObserver.observe(sec);
}

// ensurePageRendered renders forward until page n exists. Rendered pages are one
// contiguous run from page 1, so a jump past the edge fills in everything between
// rather than leaving a hole - which is what keeps sections in document order for
// both the translator and the HTML export. No-op for non-PDF documents.
async function ensurePageRendered(n) {
  while (pdfDoc && pdfRendered < Math.min(n, pdfTotal)) {
    const before = pdfRendered;
    await renderChunk();
    if (pdfRendered === before) return; // no forward progress - do not spin
  }
}

// reportRenderIdle reports where the rendered edge is once a chunk lands.
function reportRenderIdle() {
  setProgress(pdfRendered / pdfTotal);
  setStatus(pdfRendered >= pdfTotal
    ? t("vStatusDonePages", "Done - {1} pages", pdfTotal)
    : t("vStatusPagesSoFar", "Pages 1-{1} of {2} - keep scrolling to load more", pdfRendered, pdfTotal));
  setTimeout(hideStatus, 1400);
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
  setStatus(t("vStatusReadingEpub", "Reading EPUB.."));
  setProgress(0.1);

  let book;
  try {
    book = await loadEpub(data);
  } catch (err) {
    showNotice(t("vEpubFailTitle", "Couldn't open this EPUB"), [
      para(t("vEpubFailBody", "The file may be corrupt or not a valid EPUB.")),
      para(err && err.message ? t("vDetails", "Details: {1}", err.message) : ""),
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
  setPageTotal(total);
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
    label.textContent = s.label || t("vSectionN", "Section {1}", n);
    section.append(label);
    totalChars += (s.frag.textContent || "").length; // read before append empties it
    section.append(s.frag);
    content.append(section);
    registerImagesForOcr(section); // lazy image OCR (no-op when options.ocrImages is off)
    setProgress(0.1 + 0.9 * (n / total));
  });

  // Same as the PDF path: when OCR is on and images were queued for recognition, they become
  // translatable plates, so suppress the "no extractable text" banner.
  const ocrCovering = options.ocrImages && ocrTotal > 0;
  if (totalChars === 0 && !ocrCovering) {
    const banner = el("div", "notice");
    const h = el("h1");
    h.textContent = t("vLittleTextTitle", "Little or no text found");
    banner.append(
      h,
      para(t("vLittleTextEpub", "This EPUB has no extractable text (it may be image-only). Native page-translate needs actual text, not a pretty picture of it.")),
      filePickerButton(),
    );
    content.prepend(banner);
  }
  $("btn-save-html").classList.remove("hidden");
  setProgress(1);
  setStatus(total === 1
    ? t("vStatusDoneSectionsOne", "Done - 1 section")
    : t("vStatusDoneSections", "Done - {1} sections", total));
  setTimeout(hideStatus, 1200);
}

function warnIfNoText() {
  // Scanned / image-only heuristic: a large majority of pages have (almost) no
  // extractable text. Using per-page density rather than an absolute character
  // floor avoids mislabeling a legitimately short one-page document. Judged over the
  // pages rendered so far (the first chunk), not the whole file.
  const totalChars = pdfChars;
  const mostlyEmpty = pdfRendered > 1 && pdfPagesWithText / pdfRendered < 0.3;
  // When "Use OCR for images" is on the page images become translatable plates, so the
  // "little or no text" banner - which judges only the PDF text layer - would contradict what
  // the viewer is doing. This asks whether pages are queued for extraction, not whether images
  // have been found yet (ocrTotal): extraction is deferred to scroll now, so nothing has been
  // pulled out at banner time. A queued page always ends up with a plate anyway - a page with
  // no text is exactly the case appendPdfImages rasterizes whole.
  const ocrCovering = options.ocrImages && pdfImagesDeferred > 0;
  if ((totalChars === 0 || mostlyEmpty) && !ocrCovering) {
    const content = $("content");
    const banner = el("div", "notice");
    const h = el("h1");
    h.textContent = t("vLittleTextTitle", "Little or no text found");
    banner.append(
      h,
      para(t("vLittleTextPdf", "This looks like a scanned or image-only PDF, so there is little text to translate - the browser translates words, not pixels.")),
    );
    // OCR is off here (or found no images); when it is on we skip the whole banner above. Still,
    // if the reader has OCR off, suggest turning to a real OCR pass rather than a dead end.
    if (!options.ocrImages) {
      banner.append(
        para(t("vLittleTextOcrHint", "To translate scanned pages, run them through OCR first (turn on \"Use OCR for images\", or use the doc-html-translate desktop app), then reopen the result.")),
      );
    }
    banner.append(originalButton());
    content.prepend(banner);
  }
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
  $("btn-ocr").addEventListener("click", () => {
    prefs.ocrLayer = prefs.ocrLayer === false; applyPrefs(); savePrefs();
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
