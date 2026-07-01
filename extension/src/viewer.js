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
import { overlayImage, makeBadge } from "./ocr-overlay.js";
import { extractPageImages, rasterizePage } from "./pdf-images.js";

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
    return name.replace(/\.(pdf|epub)$/i, "") || "Document";
  } catch {
    return "Document";
  }
}

// sniffType inspects the leading bytes to tell a PDF (%PDF) from an EPUB / ZIP
// (PK\x03\x04). The byte signature is authoritative regardless of the URL or
// filename, so a mislabelled file still routes to the right reader.
function sniffType(data) {
  const b = new Uint8Array(data, 0, Math.min(5, data.byteLength));
  if (b[0] === 0x25 && b[1] === 0x50 && b[2] === 0x44 && b[3] === 0x46) return "pdf";   // %PDF
  if (b[0] === 0x50 && b[1] === 0x4b && b[2] === 0x03 && b[3] === 0x04) return "epub";  // PK..
  return "unknown";
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
const DEFAULT_OPTIONS = { enabledByDefault: true, disabledHosts: [], sourceLang: "auto", theme: "light", ocrImages: false, ocrLang: "eng" };
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
    showNotice("Open a PDF or EPUB", [
      para("Pick a local PDF or EPUB to read here. Opening a PDF or EPUB link loads it here too - the extension is helpful like that."),
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
    data = await resp.arrayBuffer();
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
  await loadFromData(data, fileTitle(url));
}

// loadFromData routes already-fetched bytes (URL fetch or local file picker) to
// the right reader by sniffing the file signature: PDF -> PDF.js + reflow,
// EPUB/ZIP -> the EPUB reader. Anything else falls through to the PDF path, whose
// error handling reports an unreadable file clearly.
async function loadFromData(data, title) {
  teardownCurrent();
  if (sniffType(data) === "epub") {
    await loadEpubData(data, title);
    return;
  }
  await loadPdfData(data, title);
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
      await loadFromData(data, f.name.replace(/\.(pdf|epub)$/i, ""));
    } catch (err) {
      showNotice("Couldn't read the file", [para(err.message || String(err)), filePickerButton()]);
    }
  };
  input.click();
}

function filePickerButton(label = "Open a PDF or EPUB file") {
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
  revokeCurrent = book.revoke;
  renderEpubDocument(book, title);
}

function renderEpubDocument(book, fallbackTitle) {
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
