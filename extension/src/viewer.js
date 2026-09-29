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
import { glyph, applyGlyphs } from "./glyphs.js";
import { loadEpub } from "./epub.js";
import { parseText } from "./txt.js";
import { parseRtf } from "./rtf.js";
import { parseHtml } from "./html.js";
import { parseMarkdown } from "./md.js";
import { parseFb2 } from "./fb2.js";
import { parseEbook, isMobiBytes } from "./ebook.js";
import { parseComic, DesktopOnlyError } from "./comic.js";
import { InputLimitError, checkTextInput, formatBytes } from "./limits.js";
import { overlayImage, makeBadge, ocrLangToHtmlLang, releaseOverlays } from "./ocr-overlay.js";
import { langLabel } from "./ocr-lang.js";
import { extractPageImages, rasterizePage } from "./pdf-images.js";
import { DEFAULT_OPTIONS } from "./defaults.js";
import { recordRun } from "./diagnostics.js";
import { buildExportHtml, exportImageEncoding, exportPlan, EXPORT_PREPARE_MAX_FILE_BYTES } from "./export-html.js";
import { restoreRemote, REMOTE_MARK } from "./url-policy.js";
import { parseFileParam } from "./site-host.js";
import { setupReaderSearch } from "./reader-search.js";
import {
  readerKey,
  currentPosition,
  resolvePositionTarget,
  mergePosition,
  removePosition,
  POSITIONS_KEY,
  SAVE_THROTTLE_MS,
} from "./reading-position.js";

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

// sourceFileName reduces a picker name or a document URL to the bare file name - the
// one component of the resume identity (reading-position.js). The full URL never
// enters storage: the key is a hash of the name, byte size, title and page count, and
// a URL can carry a path or a token its source would rather not see kept anywhere.
function sourceFileName(src) {
  const clean = String(src || "").split(/[?#]/)[0];
  const at = clean.lastIndexOf("/");
  let name = at >= 0 ? clean.slice(at + 1) : clean;
  try { name = decodeURIComponent(name); } catch { /* a broken escape keeps its raw text */ }
  return name;
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

// Release the previous document's resources before loading another file into the same tab: its
// book resources (blob: URLs, the MOBI reader), the pdf.js document and any load still in flight,
// the observers, the OCR overlays' fit listeners, and every blob: URL minted for its images.
let revokeCurrent = null;
let pdfTask = null; // the pdf.js loading task of the document being opened, until it settles
let readerSearch = null;
function teardownCurrent() {
  if (readerSearch) readerSearch.reset();
  clearRemoteNotice();
  hideExportDialog();
  stopExportPreparation();
  resetToolbar();
  resetReadingPosition();
  releaseOverlays($("content"));
  if (revokeCurrent) { try { revokeCurrent(); } catch { /* ignore */ } revokeCurrent = null; }
  // destroy() ends the document's worker-side state too; dropping the reference alone kept every
  // PDF opened in this tab alive in the pdf.js worker.
  if (pdfTask) { try { pdfTask.destroy(); } catch { /* ignore */ } pdfTask = null; }
  if (pdfDoc) { try { pdfDoc.destroy(); } catch { /* ignore */ } }
  if (ocrObserver) { ocrObserver.disconnect(); ocrObserver = null; }
  // Bumping the generation strands any chunk still rendering from the old document,
  // so it cannot insert its pages into the new one.
  docGen++;
  if (chunkObserver) { chunkObserver.disconnect(); chunkObserver = null; }
  if (pdfImageObserver) { pdfImageObserver.disconnect(); pdfImageObserver = null; }
  pdfImagesDeferred = 0;
  if (comicImageObserver) { comicImageObserver.disconnect(); comicImageObserver = null; }
  comicTotal = 0;
  comicPagesBytes = 0;
  chunkPending = null;
  pdfDoc = null;
  pdfTotal = 0;
  pdfRendered = 0;
  ocrTotal = 0;
  ocrDone = 0;
  ocrWithText = 0;
  ocrPending.length = 0;
  ocrStarted = new WeakSet();
  prepareBlocked = null;
  for (const url of pdfImageUrls) { try { URL.revokeObjectURL(url); } catch { /* ignore */ } }
  pdfImageUrls = [];
}

// resetToolbar puts every per-document control back to its page-load state, so nothing the last
// document decided - no TOC, OCR plates, a savable view, source bytes - carries over to the next
// one. Each loader then shows what its own document has.
function resetToolbar() {
  $("toc-tree").replaceChildren();
  $("btn-toc").classList.remove("hidden");
  $("grp-ocr").hidden = true;
  $("btn-save-html").classList.add("hidden");
  setOriginalDownload(null, "");
}

// beginLoad tears the current document down and returns the new load's token. Every load checks
// its token after each await (isCurrent): a slower, older load - a URL still downloading when the
// reader picks a local file - then stops and releases what it made instead of rendering over the
// newer document.
function beginLoad() {
  teardownCurrent();
  return docGen;
}
const isCurrent = (gen) => gen === docGen;

// ---- Preferences -----------------------------------------------------------
// DEFAULT_PREFS.size must match viewer.css's --reader-size fallback, which styles the document
// before this runs. A+/A- move it and persist; nothing is stored until the reader asks for a
// change, so this default reaches everyone who never expressed a preference.
// leading / width: the reading-comfort choices (ticket 59). null means "the shipped measure":
// body line-height 1.6 and the 46em column of viewer.css. The steps and the em values are the
// desktop reader's own (docs/PARITY.md "Reader comfort controls") and persist like the theme.
// dayTheme / nightTheme: the last theme picked from each family, so the night-mode toggle's
// round trip lands where it left - the same two slots the desktop reader keeps in localStorage.
// ocrLayer: whether the recognized-text plates are shown over the artwork. On by default -
// the plates are what makes a comic or a scan translatable - but a reader looking at the art
// wants them out of the way, so the choice persists like the theme does. Mirrors the app's
// dht_ocr toggle (docs/PARITY.md).
const DEFAULT_PREFS = { size: 28, family: "serif", theme: null, ocrLayer: true, leading: null, width: null, dayTheme: null, nightTheme: null };
const NIGHT_THEMES = new Set(["dark", "night"]);
let prefs = { ...DEFAULT_PREFS };
let options = { ...DEFAULT_OPTIONS };

const FAMILIES = {
  serif: 'Georgia, "Times New Roman", serif',
  sans: '"Segoe UI", system-ui, Arial, sans-serif',
  mono: '"Cascadia Code", "Consolas", monospace',
};

async function loadPrefs() {
  try {
    const got = await chrome.storage.local.get(["viewerPrefs", "options", POSITIONS_KEY]);
    prefs = { ...DEFAULT_PREFS, ...(got.viewerPrefs || {}) };
    options = { ...DEFAULT_OPTIONS, ...(got.options || {}) };
    positions = got[POSITIONS_KEY] && typeof got[POSITIONS_KEY] === "object" ? got[POSITIONS_KEY] : {};
  } catch { /* storage may be unavailable in odd contexts */ }
}
async function savePrefs() {
  try { await chrome.storage.local.set({ viewerPrefs: prefs }); } catch { /* ignore */ }
}
function applyPrefs() {
  document.documentElement.style.setProperty("--reader-size", `${prefs.size}px`);
  document.documentElement.style.setProperty("--reader-font", FAMILIES[prefs.family] || FAMILIES.serif);
  // Comfort choices apply only when the reader picked one; removing the property hands the
  // measure back to the shipped value in the var() fallback.
  if (prefs.leading) document.documentElement.style.setProperty("--reader-leading", prefs.leading);
  else document.documentElement.style.removeProperty("--reader-leading");
  if (prefs.width) document.documentElement.style.setProperty("--reader-width", prefs.width);
  else document.documentElement.style.removeProperty("--reader-width");
  const theme = prefs.theme || options.theme || "light";
  document.documentElement.setAttribute("data-theme", theme);
  $("sel-family").value = prefs.family;
  $("sel-leading").value = prefs.leading || "";
  $("sel-width").value = prefs.width || "";
  $("sel-theme").value = theme;
  $("btn-night").setAttribute("aria-pressed", NIGHT_THEMES.has(theme) ? "true" : "false");
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

// ---- Reading position (resume) ---------------------------------------------
// The viewer remembers where the reader stopped and offers a "Continue reading" action
// when the same document reopens (ticket 53). reading-position.js holds the rules
// (identity, anchor+offset shape, bounded storage); this block only wires them to the
// viewer's lifecycle. The identity binds once per load - name, byte size, title and
// page count as extracted, before any translation can rewrite the title - and the
// position is saved from the scroll handler, throttled, into chrome.storage.local.
// Nothing leaves the device, and the offer never moves the reader by itself.
let docKey = "";
let docSourceName = "";
let docSourceSize = 0;
let docPages = 0;
let positions = {};
let saveTimer = null;
let saveDirty = false;
let resumeBar = null;

// resetReadingPosition drops the previous document's resume state: its identity, any
// save still pending, and an unacted offer, so nothing carries into the next document.
function resetReadingPosition() {
  if (saveTimer) { clearTimeout(saveTimer); saveTimer = null; }
  saveDirty = false;
  clearResumeBar();
  docKey = "";
  docSourceName = "";
  docSourceSize = 0;
  docPages = 0;
}

// queuePositionSave coalesces scroll-driven saves: the position is read and written at
// most once per SAVE_THROTTLE_MS while the reader moves, and any pending write lands
// when the tab is hidden or closed (the two listeners at the bottom of wireToolbar).
function queuePositionSave() {
  if (!docKey) return;
  saveDirty = true;
  if (saveTimer) return;
  saveTimer = setTimeout(flushPositionSave, SAVE_THROTTLE_MS);
}

async function flushPositionSave() {
  saveTimer = null;
  if (!saveDirty || !docKey) return;
  saveDirty = false;
  const pos = currentPosition(document, window);
  if (!pos) return;
  positions = mergePosition(positions, docKey, pos, Date.now());
  try { await chrome.storage.local.set({ [POSITIONS_KEY]: positions }); } catch { /* ignore */ }
}

// maybeOfferResume shows the bar when this exact document was read before. A URL
// fragment is an explicit destination (a TOC entry, a footnote, a shared link) and
// wins - the rule the desktop reader also follows - and a saved place at the very top
// is not worth a bar. The offer sits in the flow above the document, so a reader who
// ignores it simply scrolls past.
function maybeOfferResume() {
  clearResumeBar();
  if (!docKey || location.hash) return;
  const pos = positions[docKey];
  if (!pos || !(pos.page >= 1 && pos.page <= docPages)) return;
  if (pos.page === 1 && !pos.frag && (pos.off || 0) + (pos.secoff || 0) < 40) return;
  const bar = el("div", "resume-notice");
  bar.setAttribute("role", "status");
  const line = el("span");
  line.textContent = t("vResumeAt", "Continue reading from page {1}?", pos.page);
  const go = el("button", "primary");
  go.type = "button";
  go.append(glyph("feature.continue-reading"), document.createTextNode(` ${t("vResumeContinue", "Continue reading")}`));
  go.addEventListener("click", () => { clearResumeBar(); resumeReading(pos); });
  const reset = el("button", "secondary");
  reset.type = "button";
  reset.textContent = t("vResumeReset", "Start over");
  reset.addEventListener("click", async () => {
    clearResumeBar();
    positions = removePosition(positions, docKey);
    try { await chrome.storage.local.set({ [POSITIONS_KEY]: positions }); } catch { /* ignore */ }
    window.scrollTo(0, 0);
  });
  bar.append(line, go, reset);
  $("content").before(bar);
  resumeBar = bar;
}

function clearResumeBar() {
  if (resumeBar) { resumeBar.remove(); resumeBar = null; }
}

// resumeReading jumps to the saved place. A PDF past the rendered edge renders forward
// first; the anchor is resolved afterwards, against the settled layout.
async function resumeReading(pos) {
  await ensurePageRendered(pos.page);
  const target = resolvePositionTarget(pos, document);
  if (!target) return;
  const viewport = Number(window.innerHeight) > 0 ? window.innerHeight : 0;
  window.scrollTo(0, Math.max(0, target.el.offsetTop + target.off - viewport / 3));
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
// ocrPending lists the registered images the shared observer has not processed yet, in
// register order; the HTML export's preparation drains it directly so pages the reader never
// scrolled to still get their plates before being saved (drainOcrQueue). ocrStarted guards
// against the same image entering recognition twice - the scroll observer and the drain can
// both reach for it.
const ocrPending = [];
let ocrStarted = new WeakSet();
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

// The document generation guards the counters: a picture still in the recognition queue when its
// document was replaced is skipped, and one that finishes anyway does not count toward the new
// document's progress. ocrStarted makes the function idempotent per image: the scroll observer
// and the export preparation can both reach for the same picture, and only the first wins.
async function ocrProcessImage(img) {
  if (ocrStarted.has(img)) return;
  ocrStarted.add(img);
  const gen = docGen;
  const wrapper = el("div", "ocr-pending");
  const badge = makeBadge("OCR..");
  img.replaceWith(wrapper);
  wrapper.append(img, badge);
  try {
    const container = await overlayImage(img, {
      lang: options.ocrLang || "eng",
      isCancelled: () => !isCurrent(gen),
      onProgress: (m) => {
        if (m && typeof m.progress === "number") badge.textContent = `OCR ${Math.round(m.progress * 100)}%`;
      },
    });
    if (!isCurrent(gen)) { releaseOverlays(container); return; }
    wrapper.replaceWith(container);
    if (!container.classList.contains("ocr-empty")) ocrWithText += 1;
    revealOcrToggle();
  } catch (err) {
    if (!isCurrent(gen)) return;
    console.warn("OCR failed for image", err);
    wrapper.replaceWith(img); // restore the plain image
  } finally {
    const at = ocrPending.indexOf(img);
    if (at >= 0) ocrPending.splice(at, 1);
    if (isCurrent(gen)) {
      ocrDone += 1;
      ocrUpdateStatus();
    }
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
    // A parked remote image has no src yet; it is registered when the reader allows it.
    if (ocrQueued.has(img) || img.hasAttribute(REMOTE_MARK)) continue;
    ocrQueued.add(img);
    ocrPending.push(img);
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
// then OCR'd by the shared lazy-OCR observer. comicTotal and comicRenderedCount() drive
// the export's completeness statement, mirroring the PDF counters. comicPagesBytes is
// the pages' inflated size as declared by the archive listing - known before any page
// is inflated, which is what lets the export judge a complete file's size up front
// (export-html.js).
let comicImageObserver = null;
let comicTotal = 0;
let comicPagesBytes = 0;

// comicRenderedCount reads what the reader has actually reached from the DOM: a comic
// page exists once its <img> is inside its section, so counting images is the honest
// rendered count - a page whose bytes failed to inflate leaves the reserved box and is
// not a page, however many the archive lists.
function comicRenderedCount() {
  return document.querySelectorAll('#content section[id^="comic-page-"] img').length;
}

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
// Returns the byte size of the images it appended - the export preparation adds these up
// against the file budget as it goes. 0 when the pass found nothing usable.
async function extractSectionImages(section) {
  const pageNum = Number(section.dataset.pdfPage);
  if (!pdfDoc || !pageNum) return 0;
  delete section.dataset.pdfPage;
  const gen = docGen;
  const pageChars = Number(section.dataset.pdfChars) || 0;
  let page = null;
  let added = 0;
  try {
    page = await pdfDoc.getPage(pageNum);
    if (gen !== docGen) return 0;
    added = await appendPdfImages(page, section, pageChars, gen);
  } catch {
    /* a page that will not yield its images just stays text-only */
  } finally {
    if (page) { try { page.cleanup(); } catch { /* ignore */ } }
    if (gen === docGen) section.querySelector(".pdf-page-pending")?.remove();
  }
  return added;
}

// Extract raster images from a PDF page (scanned pages fall back to a full-page raster),
// append them to the page section as <img>, and register them for lazy OCR. Must run
// before page.cleanup(). Returns the byte size of what was appended (0 when it found
// nothing usable) - see extractSectionImages.
async function appendPdfImages(page, section, pageChars, gen = docGen) {
  let imgs = [];
  let appended = 0;
  try {
    imgs = await extractPageImages(page);
    if (!imgs.length && pageChars < 20 && isCurrent(gen)) {
      const raster = await rasterizePage(page);
      if (raster) imgs = [raster];
    }
  } catch (err) {
    console.warn("PDF image extraction failed", err);
    return 0;
  }
  // Nothing minted yet: an old document's images are only Blobs, and go with this frame.
  if (!imgs.length || !isCurrent(gen)) return 0;
  for (const im of imgs) {
    const url = URL.createObjectURL(im.blob);
    pdfImageUrls.push(url);
    appended += im.blob.size;
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
  return appended;
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
  for (const n of bodyNodes) if (n) box.append(n);
  content.append(box);
  hideStatus();
}

function para(text) {
  const p = el("p");
  p.style.textIndent = "0";
  p.textContent = text;
  return p;
}

// originalButton is null when the document came from the file picker: there is no original URL
// to open, and showNotice skips it.
function originalButton(label = t("vBtnOpenOriginalPdf", "Open original PDF")) {
  if (!currentUrl) return null;
  const b = el("button");
  b.textContent = label;
  b.addEventListener("click", openOriginal);
  return b;
}

// ---- Open the untouched PDF (bypass interception for this tab) --------------
// The URL the current document was loaded from, or "" when it came from the file picker. Not
// fileUrl: that is only the page-load address, and after the reader picks a local file the
// "Original" button used to reopen the first document instead of the one on screen.
let currentUrl = "";

async function openOriginal() {
  const url = currentUrl;
  if (!url || !isSafePdfUrl(url)) return;
  try {
    const tab = await chrome.tabs.getCurrent();
    await chrome.runtime.sendMessage({ type: "open-original", url, tabId: tab && tab.id });
  } catch {
    // Last resort: navigate directly. Interception may re-catch it, but better
    // than a dead button.
    location.href = url;
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
//
// The click first asks what the file would contain. A fully rendered document saves at
// once; a chunk-rendered PDF or a scroll-inflated comic holds only the pages reached, so
// the export dialog says so before anything is written, offers a bounded "prepare all
// pages" pass (render + recognize the rest here in the viewer, then save complete) next
// to a clearly named partial export, and explains itself when the budget refuses a
// complete file. Rendering the remainder silently - the pre-ticket behavior was only to
// mutter "partial" after the fact - would either reimpose the freeze chunking exists to
// avoid or hand over a file the reader took for the whole book.
async function downloadHtml() {
  const content = $("content");
  if (!content || !content.children.length) return;
  if (readerSearch) readerSearch.reset(); // search marks are navigation, never exported annotations
  if (prepareCtx) return; // a preparation is already running - the status bar owns the moment

  const extent = exportExtent();
  const plan = prepareBlocked
    ? { action: "partial-only", reason: prepareBlocked }
    : exportPlan({ total: extent.total, rendered: extent.rendered, imageBytes: extent.imageBytes });
  if (plan.action === "direct") {
    await saveExportHtml();
    return;
  }
  showExportDialog(extent, plan);
}

// exportExtent reads the one thing the export decision needs: how many pages the document
// declares and how many the live view holds. Paged formats only - anything else is complete
// by construction (the whole DOM is what serializes).
function exportExtent() {
  if (comicTotal > 0) return { kind: "comic", total: comicTotal, rendered: comicRenderedCount(), imageBytes: comicPagesBytes };
  if (pdfDoc) return { kind: "pdf", total: pdfTotal, rendered: pdfRendered, imageBytes: 0 };
  return { kind: "static", total: 0, rendered: 0, imageBytes: 0 };
}

// ---- Export dialog ----------------------------------------------------------
// One card at a time, fixed under the toolbar so it is reachable from wherever in a long
// document the reader clicked Export. Every path out of it either saves or removes the
// card; a cancelled preparation reopens it with fresh counters rather than leaving the
// reader with a decision already made for them.
let exportDialog = null;

function hideExportDialog() {
  if (exportDialog) { exportDialog.remove(); exportDialog = null; }
}

function showExportDialog(extent, plan) {
  hideExportDialog();
  const card = el("div", "export-dialog");
  card.setAttribute("role", "dialog");
  card.setAttribute("aria-label", t("btnSaveHtml", "Export HTML"));

  const line = el("p", "export-line");
  // A PDF's rendered pages are the contiguous run 1..N; a comic's are whatever the reader
  // reached, holes included - so only the PDF may say "pages 1-N".
  line.textContent = extent.kind === "comic"
    ? t("vExportPartialLineComic", "Partial export: the file would hold {1} of {2} pages.", extent.rendered, extent.total)
    : t("vExportPartialLine", "Partial export: the document has {1} pages, and the file would hold pages 1-{2}.", extent.total, extent.rendered);
  card.append(line);

  const row = el("div", "export-actions");
  const close = () => hideExportDialog();
  const partial = el("button", "primary");
  partial.type = "button";
  partial.textContent = t("vExportPartialBtn", "Export partial ({1} pages)", extent.rendered);
  partial.addEventListener("click", async () => {
    close();
    await saveExportHtml();
  });
  row.append(partial);

  if (plan.action === "offer-prepare") {
    const hint = el("p", "export-hint");
    hint.textContent = t("vExportPrepareHint",
      "Prepares the remaining pages here in the viewer first, then saves a complete file. You can stop it at any time.");
    card.append(hint);
    const prepare = el("button", "primary");
    prepare.type = "button";
    prepare.textContent = t("vExportPrepareBtn", "Prepare all {1} pages", extent.total);
    prepare.addEventListener("click", () => {
      close();
      runExportPreparation(extent);
    });
    row.prepend(prepare);
  } else if (plan.reason) {
    const why = el("p", "export-hint");
    why.textContent = t("vExportImpossibleLine",
      "A complete export is not possible for this document: {1}.",
      t(plan.reason.key, plan.reason.fallback, ...plan.reason.args));
    card.append(why);
  }

  const cancel = el("button", "secondary");
  cancel.type = "button";
  cancel.textContent = t("vExportCancel", "Cancel");
  cancel.addEventListener("click", close);
  row.append(cancel);

  card.append(row);
  $("content").before(card);
  exportDialog = card;
  partial.focus();
}

// ---- Preparation ------------------------------------------------------------
// "Prepare all pages" materializes the rest of the document exactly the way scrolling to
// it would - chunk by chunk, one page image and one recognition at a time - so the export
// that follows is the whole book and the reader's tab never holds more than a chunk of
// new work at once. Three ways out, all leaving the reader usable and no file written:
// the reader stops it, a new document replaces this one (the docGen checks), or the
// accumulated image bytes pass the export budget (a resource-limit stop, whose reason is
// remembered so the dialog offers only the partial export afterwards).
let prepareCtx = null;      // in-flight preparation - the Stop button talks to it
let prepareBlocked = null;  // reason a complete export was refused for this document

function stopExportPreparation() {
  if (prepareCtx) prepareCtx.cancelled = true;
  prepareCtx = null;
  $("status-stop").hidden = true;
}

async function runExportPreparation(extent) {
  const ctx = { cancelled: false, gen: docGen, imageBytes: extent.imageBytes };
  prepareCtx = ctx;
  prepareBlocked = null;
  $("status").classList.remove("done");
  $("status-stop").hidden = false;
  $("status-stop").textContent = t("vExportStop", "Stop");
  try {
    const outcome = extent.kind === "comic"
      ? await prepareComicPages(ctx)
      : await preparePdfPages(ctx);
    if (ctx.cancelled || !isCurrent(ctx.gen)) {
      if (isCurrent(ctx.gen)) {
        // The dialog that reopens right after carries the exact counts; this line only has to
        // say what happened to the file - nothing was written. If the preparation had in fact
        // reached the last page before the stop landed, there is nothing left to decide: the
        // reader clicks Export again and takes the direct path.
        setStatus(t("vExportStopped", "Preparation stopped - nothing was saved"));
        setTimeout(hideStatus, 4000);
        const fresh = exportExtent();
        const plan = exportPlan({ total: fresh.total, rendered: fresh.rendered, imageBytes: fresh.imageBytes });
        if (plan.action !== "direct") showExportDialog(fresh, plan);
      }
      return;
    }
    if (outcome.limit) {
      prepareBlocked = outcome.limit;
      const fresh = exportExtent();
      setStatus(t("vExportImpossibleLine",
        "A complete export is not possible for this document: {1}.",
        t(outcome.limit.key, outcome.limit.fallback, ...outcome.limit.args)));
      setTimeout(hideStatus, 6000);
      showExportDialog(fresh, { action: "partial-only", reason: prepareBlocked });
      return;
    }
    await saveExportHtml();
  } finally {
    if (prepareCtx === ctx) prepareCtx = null;
    $("status-stop").hidden = true;
  }
}

// preparePdfPages renders the remaining chunks, then runs the deferred image pass and the
// recognition queue over the pages the reader never reached - the same work the scroll
// observers would have done, just without the scroll.
async function preparePdfPages(ctx) {
  while (pdfDoc && pdfRendered < pdfTotal) {
    if (ctx.cancelled || !isCurrent(ctx.gen)) return {};
    setStatus(t("vExportPreparing", "Preparing page {1} of {2}..", pdfRendered + 1, pdfTotal));
    setProgress(pdfRendered / pdfTotal);
    const before = pdfRendered;
    await renderChunk();
    if (pdfRendered === before) { // no forward progress - do not spin
      return { limit: prepareNoProgressReason() };
    }
    await yieldToUI();
  }
  const sections = [...document.querySelectorAll("#content section[data-pdf-page]")];
  for (const section of sections) {
    if (ctx.cancelled || !isCurrent(ctx.gen)) return {};
    if (!section.dataset.pdfPage) continue; // reached by the reader's own scrolling meanwhile
    setStatus(t("vExportPreparing", "Preparing page {1} of {2}..", Number(section.dataset.page), pdfTotal));
    ctx.imageBytes += await extractSectionImages(section) || 0;
    if (ctx.imageBytes > EXPORT_PREPARE_MAX_FILE_BYTES) {
      return { limit: prepareBytesReason(ctx.imageBytes) };
    }
    await drainOcrQueue(ctx);
    await yieldToUI();
  }
  await drainOcrQueue(ctx);
  return {};
}

// prepareComicPages inflates and recognizes every page still carrying its loader, in page
// order - the archive budget was checked before the offer, so no byte guard runs here.
async function prepareComicPages(ctx) {
  const sections = [...document.querySelectorAll('#content section[id^="comic-page-"]')];
  for (const section of sections) {
    if (ctx.cancelled || !isCurrent(ctx.gen)) return {};
    if (!comicLoaders.has(section)) continue; // reached by the reader's own scrolling meanwhile
    setStatus(t("vExportPreparing", "Preparing page {1} of {2}..", Number(section.dataset.page), comicTotal));
    await insertComicPage(section);
    await drainOcrQueue(ctx);
    await yieldToUI();
  }
  await drainOcrQueue(ctx);
  return {};
}

// drainOcrQueue recognizes the pages' images now instead of leaving them to the scroll
// observer the reader may never trigger again. One image at a time, cancellable between
// each; ocrProcessImage's own guard keeps this and the observer from ever doubling up.
async function drainOcrQueue(ctx) {
  while (ocrPending.length) {
    if (ctx.cancelled || !isCurrent(ctx.gen)) return;
    await ocrProcessImage(ocrPending[0]);
    if (ctx.cancelled || !isCurrent(ctx.gen)) return;
    await yieldToUI();
  }
}

function prepareBytesReason(bytes) {
  return {
    key: "vExportReasonBytes",
    fallback: "the complete file would be about {1}, more than a browser tab holds reliably",
    args: [formatBytes(bytes)],
  };
}

// A chunk that renders no pages means the pdfjs worker stopped handing them over - a
// concrete, if unusual, reason the declared page count cannot be reached.
function prepareNoProgressReason() {
  return {
    key: "vExportReasonStalled",
    fallback: "the remaining pages would not render",
    args: [],
  };
}

// The serialization half of the export, once the dialog has settled what the file will hold
// (or on the direct path, with no dialog at all). Decoding is awaited first: a freshly
// prepared - or freshly scrolled-to - image may still be inflating, and an image with no
// decoded size drops out of the export, which would make a "complete" file quietly miss the
// very pages preparation just rendered.
async function saveExportHtml() {
  const content = $("content");
  if (!content || !content.children.length) return;
  // Entry is gated in downloadHtml: a click while a preparation runs is ignored there, and
  // the preparation itself calls this when it finishes.

  setStatus(t("vStatusSaving", "Saving HTML.."));
  await Promise.all([...content.querySelectorAll("img")].map((img) => (
    img.decode ? img.decode().catch(() => {}) : Promise.resolve()
  )));

  const imgMap = await buildImageDataMap(content);
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
    cssVar("--reader-leading"),
    cssVar("--reader-width"),
  ].filter(Boolean).join(";");

  const doc = buildExportHtml({ title, theme, lang, styleVars, css, body: clone.outerHTML });
  const blob = new Blob([doc], { type: "text/html;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  triggerDownload(url, `${safeBase(title)}.html`);
  setTimeout(() => URL.revokeObjectURL(url), 10000);
  reportSaveCompleteness();
  if (blob.size >= EXPORT_WARN_BYTES) setTimeout(() => reportLargeSave(blob.size), 2500);
}

// Every image rides inside the saved file as a data: URI, so an image-heavy book saves as a file
// several times its source size. Past this a browser opens it slowly or not at all, and the
// reader is told so rather than left wondering why the file will not open.
const EXPORT_WARN_BYTES = 100 * 1024 * 1024;

function reportLargeSave(bytes) {
  $("status").classList.remove("done");
  setStatus(t("vSavedLarge", "Saved - the file is {1} MB because every image is inside it; some browsers open files this large slowly", Math.round(bytes / (1024 * 1024))));
  setTimeout(hideStatus, 6000);
}

// The final statement after a save, agreed with the artifact that just left: what the reader
// saved is what the live view held, so a partially rendered PDF or a comic with pages still
// un-inflated is labeled partial here exactly as the dialog labeled it before the save.
function reportSaveCompleteness() {
  const extent = exportExtent();
  $("status").classList.remove("done");
  if (extent.kind === "comic") {
    if (extent.rendered >= extent.total) setStatus(t("vSavedCompletePages", "Saved - complete, all {1} pages", extent.total));
    else setStatus(t("vSavedComic", "Partial export: {1} of {2} pages - prepare the remaining pages to add them", extent.rendered, extent.total));
  } else if (extent.kind === "pdf") {
    if (extent.rendered >= extent.total) setStatus(t("vSavedCompletePages", "Saved - complete, all {1} pages", extent.total));
    else setStatus(t("vSavedPdf", "Partial export: pages 1-{1} of {2} - prepare the remaining pages to add them", extent.rendered, extent.total));
  } else {
    setStatus(t("vSavedCompleteDoc", "Saved - the complete document"));
  }
  setTimeout(hideStatus, 5000);
}

function cssVar(name) {
  const v = document.documentElement.style.getPropertyValue(name).trim();
  return v ? `${name}:${v}` : "";
}

// Rasterize every live blob: image to a data: URI, keyed by src. Live images are already
// decoded, so naturalWidth/Height are valid (a fresh clone's would be 0). http(s)/data
// images are left untouched - they are already portable.
//
// Encoding is asynchronous and one image at a time: toDataURL encoded every picture of a comic
// synchronously in one task, freezing the tab and holding each canvas until the loop ended. Each
// canvas is emptied as soon as its image is encoded.
async function buildImageDataMap(root) {
  const map = new Map();
  for (const img of root.querySelectorAll("img")) {
    const src = img.getAttribute("src") || "";
    if (!src || map.has(src) || !/^blob:/i.test(src)) continue;
    const c = el("canvas");
    try {
      const w = img.naturalWidth, h = img.naturalHeight;
      if (!w || !h) { map.set(src, null); continue; }
      c.width = w;
      c.height = h;
      const ctx = c.getContext("2d");
      ctx.drawImage(img, 0, 0);
      const enc = exportImageEncoding(ctx, w, h);
      const blob = await new Promise((resolve) => c.toBlob(resolve, enc.type, enc.quality));
      map.set(src, blob ? await blobToDataUrl(blob) : null);
    } catch {
      map.set(src, null); // tainted or too large - dropped below
    } finally {
      c.width = 0;
      c.height = 0;
    }
  }
  return map;
}

function blobToDataUrl(blob) {
  return new Promise((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => resolve(String(r.result));
    r.onerror = () => reject(r.error);
    r.readAsDataURL(blob);
  });
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

// ---- TOC -------------------------------------------------------------------
function renderToc(entries) {
  const tree = $("toc-tree");
  tree.replaceChildren();
  const empty = !entries || entries.length === 0;
  $("btn-toc").classList.toggle("hidden", empty);
  if (empty) {
    $("toc").classList.add("hidden"); // an open panel would be left showing nothing
    $("btn-toc").setAttribute("aria-expanded", "false");
    return;
  }
  tree.append(buildTocList(entries));
  updateCurrentTocEntry();
}

function updateCurrentTocEntry() {
  const links = [...$("toc-tree").querySelectorAll("a[href^='#']")];
  let current = null;
  for (const link of links) {
    const id = link.getAttribute("href").slice(1);
    const target = document.getElementById(id) ||
      (id.startsWith("page-") ? document.querySelector(`#content section[data-page="${Number(id.slice(5))}"]`) : null);
    if (target && target.getBoundingClientRect().top <= 90) current = link;
  }
  if (!current) current = links.find((link) => document.getElementById(link.getAttribute("href").slice(1))) || null;
  for (const link of links) {
    if (link === current) link.setAttribute("aria-current", "location");
    else link.removeAttribute("aria-current");
  }
  for (let node = current?.parentElement; node && node !== $("toc"); node = node.parentElement) {
    if (node.classList?.contains("collapsed")) {
      node.classList.remove("collapsed");
      const toggle = node.querySelector(":scope > .toc-toggle");
      if (toggle) {
        toggle.replaceChildren(glyph("nav.collapse"));
        toggle.setAttribute("aria-expanded", "true");
        toggle.setAttribute("aria-label", t("ariaCollapse", "Collapse"));
      }
    }
  }
}

function setTocOpen(open, focusButton = false) {
  $("toc").classList.toggle("hidden", !open);
  $("btn-toc").setAttribute("aria-expanded", open ? "true" : "false");
  if (focusButton) $("btn-toc").focus();
}

function buildTocList(entries) {
  const ul = el("ul");
  for (const e of entries) {
    const li = el("li");
    const hasKids = e.children && e.children.length > 0;
    if (hasKids) {
      // The disclosure shows the action a click takes: nav.collapse while the branch is open,
      // nav.expand while it is folded - the same chevrons as the desktop GUI (docs/PARITY.md).
      const toggle = el("button", "toc-toggle");
      toggle.type = "button";
      const setToggle = () => {
        const folded = li.classList.contains("collapsed");
        toggle.replaceChildren(glyph(folded ? "nav.expand" : "nav.collapse"));
        toggle.setAttribute("aria-expanded", folded ? "false" : "true");
        toggle.setAttribute("aria-label", folded ? t("ariaExpand", "Expand") : t("ariaCollapse", "Collapse"));
      };
      setToggle();
      toggle.addEventListener("click", () => {
        li.classList.toggle("collapsed");
        setToggle();
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
        setTocOpen(false);
        updateCurrentTocEntry();
      });
      li.append(a);
    } else if (e.page != null) {
      const a = el("a");
      a.textContent = e.title || t("vPageN", "Page {1}", e.page);
      a.href = `#page-${e.page}`;
      a.addEventListener("click", (ev) => {
        ev.preventDefault();
        scrollToPage(e.page);
        setTocOpen(false);
        updateCurrentTocEntry();
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
  applyI18n(document.getElementById("search-panel"));
  document.getElementById("search-panel").lang = uiLang();
  applyGlyphs(document.getElementById("toolbar"));
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
  readerSearch = setupReaderSearch({
    root: $("content"),
    beforeWholeBook: async () => { if (pdfDoc) await ensurePageRendered(pdfTotal); },
    scopeLabel: (scope) => scope === "book"
      ? t("vSearchBook", "whole book") : t("vSearchPage", "this page"),
    translate: t,
  });

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

// isHtmlReplyForDocument: the server answered with HTML for a URL that names a non-HTML
// document. file:// replies carry no meaningful type, and a URL naming no known format may
// well be a page on purpose, so neither is judged here.
function isHtmlReplyForDocument(resp, url) {
  if (isFileUrl(url)) return false;
  const type = (resp.headers.get("content-type") || "").toLowerCase();
  if (!/^\s*(text\/html|application\/xhtml\+xml)/.test(type)) return false;
  const format = FORMAT_EXT[fileExt(url)];
  return !!format && format !== "html";
}

// loadUrl downloads a PDF by URL and renders it. On failure it offers the local
// file picker (which needs no host access and no file-URL toggle) as a fallback.
async function loadUrl(url) {
  const gen = beginLoad();
  currentUrl = url;
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
    // With the reader's cookies: a document behind a login opens here exactly when it opens in
    // a plain tab. The extension's host access already bypasses CORS, so this adds nothing a
    // page could use - the request is the viewer's own and its reply never reaches a page.
    const resp = await fetch(url, { credentials: "include" });
    if (!isCurrent(gen)) return;
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    if (isHtmlReplyForDocument(resp, url)) {
      // A login wall or an interstitial answers a document URL with a web page. Parsing it as
      // the document would report a corrupt file; the page itself is what the reader needs.
      showNotice(t("vHtmlReplyTitle", "The site sent a web page instead of the document"), [
        para(t("vHtmlReplyBody", "This usually means the document needs a sign-in or a confirmation click on the site. Open the original, finish that step, then convert the document again.")),
        originalButton(t("vBtnOpenOriginal", "Open original")),
        filePickerButton(),
      ]);
      return;
    }
    const blob = await resp.blob();
    if (!isCurrent(gen)) return;
    data = await blob.arrayBuffer();
    if (!isCurrent(gen)) return;
    setOriginalDownload(blob, filenameFromUrl(url)); // keep a downloadable copy (browser-backed)
  } catch (err) {
    if (!isCurrent(gen)) return;
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
  await loadFromData(data, fileTitle(url), url, gen);
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
  // The resume identity binds here, the one moment every loader has set both the
  // document's title and its page count - the desktop binds the same four inputs once,
  // before translation, for the same reason (docs/PARITY.md "Reading position").
  docPages = total > 0 ? total : 0;
  docKey = readerKey(sourceFileName(docSourceName), docSourceSize, document.title || "", docPages);
}

// The formats read whole, held against the text-input budget before parsing (limits.js).
const TEXT_FORMATS = new Set(["txt", "rtf", "html", "md", "fb2"]);

async function loadFromData(data, title, name, gen) {
  if (!isCurrent(gen)) return;
  docSourceName = name;
  docSourceSize = data.byteLength;
  const format = detectFormat(data, name);
  // The format id only - never the document's name, bytes or URL. See diagnostics.js.
  recordRun({ format });
  if (TEXT_FORMATS.has(format)) {
    try {
      checkTextInput(data.byteLength);
    } catch (err) {
      showLimitNotice(err);
      return;
    }
  }
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
    case "comic": await loadComicData(data, title, name); return;
    default: await loadPdfData(data, title);
  }
}

// loadBook is the generic intake the non-PDF/EPUB parsers reuse: set status, hide
// the PDF-only Original button, parse the bytes into the shared book shape, and
// render it. On a parse error it shows a notice with the file picker.
async function loadBook(data, title, parseFn, statusLabel) {
  const gen = docGen;
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
    if (!isCurrent(gen)) return;
    showNotice(t("vOpenFileFailTitle", "Couldn't open this file"), [
      para(t("vCorruptBody", "The file may be corrupt or not a supported document.")),
      para(err && err.message ? t("vDetails", "Details: {1}", err.message) : ""),
      filePickerButton(),
    ]);
    return;
  }
  if (!isCurrent(gen)) { if (book.revoke) book.revoke(); return; }
  renderBook(book, title);
}

// loadImageData renders a standalone image the user opened directly and OCRs it into
// translatable text plates (the same overlay unit the right-click image page and the
// in-document image OCR use). OCR runs unconditionally here, independent of the "Use OCR
// for images" toggle: opening a bare image is itself the explicit request to read its
// text, and without OCR there is nothing for the browser translator to work on.
async function loadImageData(data, title, mime) {
  const gen = docGen;
  $("doc-title").textContent = title;
  document.title = title;
  $("btn-original").classList.add("hidden"); // no "native viewer" concept for a picture
  $("status").classList.remove("done");
  setPageTotal(1);
  renderToc(null); // a picture has no table of contents
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
      isCancelled: () => !isCurrent(gen),
    });
    if (!isCurrent(gen)) { releaseOverlays(container); return; }
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
    if (!isCurrent(gen)) return;
    showNotice(t("vImageFailTitle", "Couldn't read this image"), [
      para(err && err.message ? t("vDetails", "Details: {1}", err.message) : t("vImageFailBody", "The image could not be processed.")),
      filePickerButton(),
    ]);
    return;
  }
  $("btn-save-html").classList.remove("hidden");
  setProgress(1);
  maybeOfferResume();
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
async function loadComicData(data, title, name) {
  const gen = docGen;
  $("doc-title").textContent = title;
  document.title = title;
  $("btn-original").classList.add("hidden"); // no "native viewer" concept for a comic
  $("status").classList.remove("done");
  setStatus(t("vStatusReadingComic", "Reading comic.."));
  setProgress(0.1);
  ensureOcrCss();

  let pages;
  try {
    pages = await parseComic(data, name);
  } catch (err) {
    if (!isCurrent(gen)) return;
    if (err instanceof InputLimitError) {
      showLimitNotice(err);
    } else if (err instanceof DesktopOnlyError) {
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
  if (!isCurrent(gen)) return;
  renderComic(pages);
}

// showLimitNotice refuses a file that is over the published input limits (limits.js), naming
// the limit in the reader's language rather than reporting a corrupt file.
function showLimitNotice(err) {
  showNotice(t("vLimitTitle", "This file is over the size limits"), [
    para(t(err.key, err.fallback, ...err.args)),
    filePickerButton(),
  ]);
}

// renderComic lays out one placeholder section per page up front (so the scrollbar
// reflects the whole book immediately) and defers each page's image inflation and OCR
// to scroll via the comic image observer. Mirrors the scanned-PDF path.
function renderComic(pages) {
  applyLang(ocrLangToHtmlLang(options.ocrLang || "eng"));
  comicTotal = pages.length;
  comicPagesBytes = pages.reduce((sum, pg) => sum + (pg.size || 0), 0);
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
  maybeOfferResume();
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
  img.src = url;
  section.append(img);
  section.querySelector(".comic-page-pending")?.remove();
  registerImagesForOcr(section, true); // comics force OCR on
}

// loadPdfData runs the PDF.js + reflow pipeline over PDF bytes.
async function loadPdfData(data, title) {
  const gen = docGen;
  $("doc-title").textContent = title;
  document.title = title;
  $("btn-original").classList.toggle("hidden", !currentUrl);
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
    pdfTask = task;
    pdf = await task.promise;
  } catch (err) {
    if (!isCurrent(gen)) return; // destroyed by the teardown of a newer load
    pdfTask = null;
    handleLoadError(err);
    return;
  }
  if (!isCurrent(gen)) { try { pdf.destroy(); } catch { /* ignore */ } return; }
  pdfTask = null;

  await renderDocument(pdf, title, gen);
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
    const gen = beginLoad();
    currentUrl = "";
    $("content").replaceChildren();
    $("status").classList.remove("done");
    setStatus(t("vStatusReadingFile", "Reading file.."));
    try {
      const data = await f.arrayBuffer();
      if (!isCurrent(gen)) return;
      setOriginalDownload(f, f.name); // the picked File is itself a downloadable Blob
      await loadFromData(data, f.name.replace(/\.(pdf|epub|txt|rtf|html?|md|fb2|mobi|azw3|png|jpe?g|gif|bmp|webp)$/i, ""), f.name, gen);
    } catch (err) {
      if (!isCurrent(gen)) return;
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

async function renderDocument(pdf, title, gen = docGen) {
  const total = pdf.numPages;
  setPageTotal(total);
  $("page-jump").max = String(total);

  // Sample early pages for language detection before rendering everything.
  setStatus(t("vStatusDetecting", "Detecting language.."));
  // Nothing of this document is referenced until pdfDoc is set below, so a superseded load
  // releases it at whichever await it notices - before it can touch the newer document's
  // <html lang>, TOC or banner.
  const drop = () => { try { pdf.destroy(); } catch { /* ignore */ } };
  const sampleText = await collectSample(pdf, Math.min(total, 5));
  if (!isCurrent(gen)) { drop(); return; }
  const lang = await pdfDocumentLang(pdf, sampleText);
  if (!isCurrent(gen)) { drop(); return; }
  applyLang(lang);

  // TOC from the outline.
  let toc = [];
  try { toc = await buildToc(pdf); } catch { toc = []; }
  if (!isCurrent(gen)) { drop(); return; }
  renderToc(toc);

  $("content").replaceChildren();

  pdfDoc = pdf;
  pdfTotal = total;
  pdfRendered = 0;
  pdfChars = 0;
  pdfPagesWithText = 0;

  await renderChunk();
  // A newer load's teardown has destroyed pdf by now and owns the counters warnIfNoText reads.
  if (!isCurrent(gen)) return;
  // Judge "scanned, image-only PDF" on the first chunk alone: up to PAGE_CHUNK pages
  // is a fair sample, and a scanned book is scanned throughout. Waiting for the whole
  // document would mean never showing the banner on the files that most need it.
  warnIfNoText();
  $("btn-save-html").classList.remove("hidden");
  maybeOfferResume();
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

// ---- Remote content --------------------------------------------------------
// A document's remote images and media are parked by the sanitizer (url-policy.js), so opening
// a book never tells its author, or a tracker they embedded, that it was opened. The reader
// gets one notice per document and decides: this document, or every document from now on
// (options.allowRemoteContent). The notice sits outside #content so it is never exported.
let remoteNotice = null;

function clearRemoteNotice() {
  if (remoteNotice) { remoteNotice.remove(); remoteNotice = null; }
}

function loadRemoteContent() {
  clearRemoteNotice();
  const content = $("content");
  restoreRemote(content);
  registerImagesForOcr(content); // the restored images now have something to recognize
}

async function allowRemoteAlways() {
  loadRemoteContent();
  options.allowRemoteContent = true;
  try {
    const got = await chrome.storage.local.get("options");
    await chrome.storage.local.set({ options: { ...DEFAULT_OPTIONS, ...(got.options || {}), allowRemoteContent: true } });
  } catch { /* the choice still holds for this document */ }
}

function offerRemoteContent(count) {
  clearRemoteNotice();
  const bar = el("div", "remote-notice");
  bar.setAttribute("role", "status");
  const text = el("span");
  text.textContent = t("vRemoteBlocked", "This document wants to load {1} item(s) from the internet. They are blocked, so its author cannot see that you opened it.", count);
  const once = el("button");
  once.type = "button";
  once.textContent = t("vRemoteLoad", "Load them");
  once.addEventListener("click", loadRemoteContent);
  const always = el("button");
  always.type = "button";
  always.textContent = t("vRemoteAlways", "Always load remote content");
  always.addEventListener("click", allowRemoteAlways);
  bar.append(text, once, always);
  $("content").before(bar);
  remoteNotice = bar;
}

// ---- EPUB ------------------------------------------------------------------
// EPUB content is already semantic XHTML, so there is no reflow step: epub.js
// returns chapters as sanitized DOM fragments (images -> blob: URLs, links ->
// in-page anchors) that we drop into the same #content / TOC / page UI the PDF
// path uses. Each spine document is one "page" for navigation.
async function loadEpubData(data, title) {
  const gen = docGen;
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
    if (!isCurrent(gen)) return;
    if (err instanceof InputLimitError) {
      showLimitNotice(err);
      return;
    }
    showNotice(t("vEpubFailTitle", "Couldn't open this EPUB"), [
      para(t("vEpubFailBody", "The file may be corrupt or not a valid EPUB.")),
      para(err && err.message ? t("vDetails", "Details: {1}", err.message) : ""),
      filePickerButton(),
    ]);
    return;
  }
  if (!isCurrent(gen)) { if (book.revoke) book.revoke(); return; }
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

  const remoteAllowed = options.allowRemoteContent === true;
  let totalChars = 0;
  book.sections.forEach((s, i) => {
    if (remoteAllowed) restoreRemote(s.frag);
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
  if (book.remote > 0 && !remoteAllowed) offerRemoteContent(book.remote);
  $("btn-save-html").classList.remove("hidden");
  setProgress(1);
  setStatus(total === 1
    ? t("vStatusDoneSectionsOne", "Done - 1 section")
    : t("vStatusDoneSections", "Done - {1} sections", total));
  maybeOfferResume();
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
    const orig = originalButton();
    if (orig) banner.append(orig);
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

// pdfDocumentLang only decides the language; the caller applies it once it knows the load is
// still current.
async function pdfDocumentLang(pdf, sampleText) {
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
  return lang;
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
    setTocOpen($("toc").classList.contains("hidden"));
  });
  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape" && !$("toc").classList.contains("hidden")) setTocOpen(false, true);
  });
  document.addEventListener("click", (event) => {
    if (window.matchMedia("(max-width: 700px)").matches && !$("toc").contains(event.target) && !$("btn-toc").contains(event.target)) setTocOpen(false);
  });
  let tocScrollQueued = false;
  window.addEventListener("scroll", () => {
    if (tocScrollQueued) return;
    tocScrollQueued = true;
    requestAnimationFrame(() => { tocScrollQueued = false; updateCurrentTocEntry(); });
  }, { passive: true });
  $("btn-font-inc").addEventListener("click", () => {
    prefs.size = Math.min(40, prefs.size + 1);
    applyPrefs(); savePrefs();
  });
  $("btn-font-dec").addEventListener("click", () => {
    prefs.size = Math.max(12, prefs.size - 1);
    applyPrefs(); savePrefs();
  });
  $("btn-font-reset").addEventListener("click", () => {
    prefs.size = DEFAULT_PREFS.size;
    applyPrefs(); savePrefs();
  });
  $("sel-family").addEventListener("change", (e) => {
    prefs.family = e.target.value; applyPrefs(); savePrefs();
  });
  $("sel-leading").addEventListener("change", (e) => {
    prefs.leading = e.target.value || null; applyPrefs(); savePrefs();
  });
  $("sel-width").addEventListener("change", (e) => {
    prefs.width = e.target.value || null; applyPrefs(); savePrefs();
  });
  // A theme picked from the select is remembered as its family's last word, so the
  // night-mode toggle can return the reader exactly where the round trip started.
  const rememberFamilyTheme = (theme) => {
    if (NIGHT_THEMES.has(theme)) prefs.nightTheme = theme;
    else prefs.dayTheme = theme;
  };
  $("sel-theme").addEventListener("change", (e) => {
    prefs.theme = e.target.value;
    rememberFamilyTheme(prefs.theme);
    applyPrefs(); savePrefs();
  });
  $("btn-night").addEventListener("click", () => {
    const night = NIGHT_THEMES.has(prefs.theme || options.theme || "light");
    prefs.theme = (night ? prefs.dayTheme : prefs.nightTheme) || (night ? "light" : "night");
    rememberFamilyTheme(prefs.theme);
    applyPrefs(); savePrefs();
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
  $("status-stop").addEventListener("click", () => {
    if (prepareCtx) prepareCtx.cancelled = true;
  });

  // Keep the page-jump box in sync with scroll position, and the reading position with
  // the reader.
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
      queuePositionSave();
    });
  }, { passive: true });
  // A throttled save still pending when the reader closes or hides the tab lands now,
  // not never - the last seconds of a session keep their place.
  window.addEventListener("pagehide", flushPositionSave);
  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState === "hidden") flushPositionSave();
  });
}

main();
