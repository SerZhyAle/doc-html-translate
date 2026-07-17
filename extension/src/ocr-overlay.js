// ocr-overlay.js - the reusable OCR-overlay unit shared by the context-menu page and
// the PDF/EPUB viewer. Recognizes text in an image with a single shared Tesseract worker
// (one image at a time via a FIFO queue), groups results into block-level plates, and
// renders opaque plates carrying real HTML text over the source image so the browser's
// built-in "Translate page" can translate them in place. Progress is reported via
// onProgress({status, progress}) throughout.

import Tesseract from "../vendor/tesseract/tesseract.esm.min.js";
import { workerOptions } from "./ocr-lang.js";
import { isTranslatable } from "./ocr-text.js";

const { createWorker } = Tesseract;

// ---- Shared worker (lazy, one per language) --------------------------------
// A single worker recognizes one image at a time. It is created lazily and reused;
// switching language terminates and recreates it (a document normally uses one OCR
// language, so this is rare). The fixed logger delegates to whatever recognize() call
// currently owns the worker, so progress routes to the right onProgress.
let workerPromise = null;
let workerLang = null;
let currentProgress = null;

// Page-segmentation mode for the recognition worker. tesseract.js defaults tessedit_pageseg_mode to
// PSM 6 (SINGLE_BLOCK), which reads a whole illustrated/scanned page as ONE text block - so on a page
// with a figure and scattered text (a speech bubble, a caption) it folds scene edges into the
// recognized text (stray "< =", "|", digits) and mis-merges regions. Pin PSM 3 (AUTO) so layout
// analysis isolates real text regions, matching the desktop app's tesseract CLI (whose own default is
// PSM 3). Shared invariant - see docs/PARITY.md and tesseract.go ocrPageSegMode.
const OCR_PSM = "3";

async function getWorker(lang) {
  if (workerPromise && workerLang === lang) return workerPromise;
  if (workerPromise) {
    try { (await workerPromise).terminate(); } catch { /* ignore */ }
    workerPromise = null;
  }
  workerLang = lang;
  const options = workerOptions(lang);
  options.logger = (m) => { if (currentProgress) currentProgress(m); };
  workerPromise = createWorker(lang, 1, options).then(async (w) => {
    // Best-effort worker params (keep the worker if a call fails). Set PSM first and on its own so
    // silencing the logs below can never revert it.
    // - PSM 3 to match the desktop CLI (see OCR_PSM).
    try { await w.setParameters({ tessedit_pageseg_mode: OCR_PSM }); } catch { /* keep default */ }
    // - Route Tesseract's engine chatter ("Estimating resolution as N", "Detected N diacritics",
    //   "Invalid resolution 0 dpi") to the null device. It is printed via the C++ tprintf, which
    //   tesseract.js forwards to the worker console - and Chrome's extension console flags those as
    //   errors. /dev/null exists in the Emscripten FS; real failures still reject the promise. The
    //   desktop CLI needs no equivalent - it captures stderr and discards it on success.
    try { await w.setParameters({ debug_file: "/dev/null" }); } catch { /* leave engine logs on */ }
    return w;
  });
  return workerPromise;
}

// ---- FIFO queue (single-flight) --------------------------------------------
// Chain every recognize() through one tail promise so only one image is processed at a
// time, whatever the caller does. A failed task never breaks the chain.
let queueTail = Promise.resolve();
function enqueue(task) {
  const run = queueTail.then(task, task);
  queueTail = run.then(() => {}, () => {});
  return run;
}

// ---- Image source normalization (cross-origin safe) ------------------------
// Accept a URL string, a Blob, or an <img>. For URLs (and an <img>'s src) fetch the
// bytes via the extension's host access instead of reading a possibly-tainted canvas.
function isSafeImageUrl(url) {
  return /^(https?:|file:|blob:|data:)/i.test(url);
}

async function fetchToBlob(url) {
  if (!isSafeImageUrl(url)) throw new Error("Unsupported image URL scheme");
  const resp = await fetch(url);
  if (!resp.ok) throw new Error(`Image fetch failed: ${resp.status}`);
  return resp.blob();
}

async function toBitmap(src) {
  if (src instanceof Blob) {
    const bmp = await createImageBitmap(src);
    return { source: src, width: bmp.width, height: bmp.height };
  }
  if (typeof HTMLImageElement !== "undefined" && src instanceof HTMLImageElement) {
    return toBitmap(await fetchToBlob(src.currentSrc || src.src));
  }
  if (typeof src === "string") {
    return toBitmap(await fetchToBlob(src));
  }
  throw new Error("Unsupported image source");
}

// ---- Recognition -----------------------------------------------------------
// Overlay grouping constants, shared verbatim with the desktop app (see docs/PARITY.md and
// internal/ocr/tesseract.go ocrMinLineConf / ocrClusterGapFactor). OCR_MIN_LINE_CONF drops a
// line whose mean word confidence is below it: real text scores ~80-97, while the "text"
// Tesseract hallucinates out of a drawing scores ~0-50, so the gate removes noise that would
// otherwise become an opaque plate covering the figure (and whose oversized boxes inflate the
// font). OCR_CLUSTER_GAP_FACTOR then grows one plate while the vertical gap to the next line
// stays within this many median line heights; a bigger gap - a figure, a section break, a new
// column - starts a new plate.
const OCR_MIN_LINE_CONF = 50;
const OCR_CLUSTER_GAP_FACTOR = 1.2;

// OCR resolution constants, shared with the desktop app's tesseract.go (docs/PARITY.md). We do not
// gate on raw pixel count - a page scan is over 1000 px tall even at a poor ~100 DPI, so a pixel
// threshold either upscales everything (4x the OCR cost on a clean render that gains nothing) or
// nothing (the low-res scan that needs it most). Instead we estimate DPI from the long side against
// an assumed page height and act on that: below OCR_UPSCALE_DPI_FLOOR the image is enlarged
// OCR_UPSCALE_FACTOR-fold before recognition (coordinates divided back after), and in every case the
// resolution is declared to Tesseract (clamped to >= OCR_MIN_DECLARED_DPI). Measured: a ~90-DPI
// newsprint scan gains hugely from the upscale, while a ~150-DPI scan only needs the DPI declared -
// the upscale over-segments it for no benefit.
const OCR_UPSCALE_FACTOR = 2;
const OCR_ASSUMED_PAGE_INCHES = 11; // assumed long-side page size (US Letter) for the DPI estimate
const OCR_UPSCALE_DPI_FLOOR = 120; // estimated DPI below which an image is upscaled before OCR
const OCR_MIN_DECLARED_DPI = 70; // never declare a DPI below this (Tesseract ignores sub-70 anyway)

// estimateDpi approximates an image's resolution from its long side, treating it as one
// OCR_ASSUMED_PAGE_INCHES-tall page. 0 when unknown. Crude, but enough to tell a low-res scan that
// needs enlarging from a mid-res one that only needs its DPI declared.
function estimateDpi(longSidePx) {
  if (!longSidePx || longSidePx <= 0) return 0;
  return Math.round(longSidePx / OCR_ASSUMED_PAGE_INCHES);
}

function clampDeclaredDpi(d) {
  return d > 0 && d < OCR_MIN_DECLARED_DPI ? OCR_MIN_DECLARED_DPI : d;
}

// ---- Adaptive plate colours ------------------------------------------------
// Sample the source image so each plate borrows the block's background ("paper") and text
// ("ink") colours - the overlay then blends into the document instead of showing a white
// patch. bg is the median colour over the whole block (text is the minority, so the median
// lands on paper). ink is the mean of the pixels that stand out from bg within the FIRST
// line only (real text lives there, not figures lower in a merged block - this is "the
// colour of the original's first letter"), with a near-black/near-white fallback that
// guarantees contrast. Best-effort: any failure leaves the CSS defaults. Mirrors the
// desktop app's overlay.go blockColors (see docs/PARITY.md - keep the two in sync).
const luma = (r, g, b) => (299 * r + 587 * g + 114 * b) / 1000;
const medianOf = (a) => (a.length ? a.slice().sort((p, q) => p - q)[a.length >> 1] : 0);

function pixelsIn(ctx, x0, y0, w, h) {
  const data = ctx.getImageData(x0, y0, w, h).data;
  const n = w * h, step = Math.max(1, Math.floor(n / 6000));
  const rs = [], gs = [], bs = [];
  for (let i = 0; i < n; i += step) {
    const o = i * 4;
    if (data[o + 3] < 128) continue;
    rs.push(data[o]); gs.push(data[o + 1]); bs.push(data[o + 2]);
  }
  return { rs, gs, bs };
}

function blockColors(ctx, bbox, lineHeight) {
  const W = ctx.canvas.width, H = ctx.canvas.height;
  const x0 = Math.max(0, Math.floor(bbox.x0)), y0 = Math.max(0, Math.floor(bbox.y0));
  const w = Math.max(1, Math.min(W - x0, Math.ceil(bbox.x1 - bbox.x0)));
  const h = Math.max(1, Math.min(H - y0, Math.ceil(bbox.y1 - bbox.y0)));
  if (w < 2 || h < 2) return null;
  const all = pixelsIn(ctx, x0, y0, w, h);
  if (!all.rs.length) return null;
  const bg = [medianOf(all.rs), medianOf(all.gs), medianOf(all.bs)];
  const lh = Math.max(1, Math.min(h, Math.round((lineHeight || h) * 1.3)));
  const first = pixelsIn(ctx, x0, y0, w, lh);
  let ir = 0, ig = 0, ib = 0, c = 0;
  for (let i = 0; i < first.rs.length; i++) {
    if (Math.abs(first.rs[i] - bg[0]) + Math.abs(first.gs[i] - bg[1]) + Math.abs(first.bs[i] - bg[2]) > 90) {
      ir += first.rs[i]; ig += first.gs[i]; ib += first.bs[i]; c++;
    }
  }
  const fallback = () => (luma(...bg) > 140 ? [17, 17, 17] : [240, 240, 240]);
  let ink = c >= Math.max(6, first.rs.length * 0.015) ? [Math.round(ir / c), Math.round(ig / c), Math.round(ib / c)] : fallback();
  if (Math.abs(luma(...ink) - luma(...bg)) < 55) ink = fallback();
  return { bg: `rgb(${bg[0]},${bg[1]},${bg[2]})`, ink: `rgb(${ink[0]},${ink[1]},${ink[2]})` };
}

// Draw the (untainted) source blob to a canvas and attach { bg, ink } to each block.
async function sampleColors(blob, blocks) {
  try {
    const bmp = await createImageBitmap(blob);
    const cv = typeof OffscreenCanvas !== "undefined"
      ? new OffscreenCanvas(bmp.width, bmp.height)
      : Object.assign(document.createElement("canvas"), { width: bmp.width, height: bmp.height });
    const ctx = cv.getContext("2d", { willReadFrequently: true });
    ctx.drawImage(bmp, 0, 0);
    for (const b of blocks) b.colors = blockColors(ctx, b.bbox, b.lineHeight);
    if (bmp.close) bmp.close();
  } catch { /* best-effort: plates keep the default white/dark CSS */ }
}

// Flatten the recognized hierarchy (blocks -> paragraphs -> lines -> words) to a flat list of
// text lines, each with its bbox, concatenated word text and mean word confidence. Falls back
// to the paragraph, then the block, when a level exposes no finer children.
function collectLines(data, scale = 1) {
  const out = [];
  const push = (u) => {
    if (!u || !u.bbox) return;
    const words = u.words || [];
    const text = (words.length ? words.map((w) => w.text).join(" ") : (u.text || ""))
      .replace(/\s+/g, " ").trim();
    const conf = words.length
      ? words.reduce((s, w) => s + (w.confidence || 0), 0) / words.length
      : (typeof u.confidence === "number" ? u.confidence : 0);
    const b = u.bbox;
    const bbox = scale === 1 ? b : {
      x0: Math.round(b.x0 / scale), y0: Math.round(b.y0 / scale),
      x1: Math.round(b.x1 / scale), y1: Math.round(b.y1 / scale),
    };
    out.push({ bbox, text, conf });
  };
  for (const b of data.blocks || []) {
    const paras = (b.paragraphs && b.paragraphs.length) ? b.paragraphs : [b];
    for (const p of paras) {
      const lines = (p.lines && p.lines.length) ? p.lines : [p];
      for (const l of lines) push(l);
    }
  }
  return out;
}

// Drop low-confidence noise lines, then group the survivors (in reading order) into one plate
// per run of vertically-adjacent, horizontally-overlapping lines. A plate's box is the union of
// its line boxes and its font tracks the median line height, so a plate covers a coherent text
// column without spanning imagery or the gaps between columns/sections. Mirrors the desktop
// app's tesseract.go clusterLines - keep the two in sync (docs/PARITY.md).
function clusterLines(lines) {
  const kept = lines.filter((l) => l.text && l.conf >= OCR_MIN_LINE_CONF);
  if (!kept.length) return [];
  const medianH = medianOf(kept.map((l) => l.bbox.y1 - l.bbox.y0)) || 1;
  const gapMax = medianH * OCR_CLUSTER_GAP_FACTOR;

  const blocks = [];
  let cur = null;
  const flush = () => {
    if (!cur) return;
    const text = cur.texts.join(" ").trim();
    if (isTranslatable(text)) {
      blocks.push({
        text,
        bbox: { x0: cur.x0, y0: cur.y0, x1: cur.x1, y1: cur.y1 },
        lineHeight: medianOf(cur.heights) || (cur.y1 - cur.y0),
      });
    }
    cur = null;
  };
  for (const l of kept) {
    const { x0, y0, x1, y1 } = l.bbox;
    if (cur) {
      const gap = y0 - cur.y1;
      const overlap = Math.min(x1, cur.x1) - Math.max(x0, cur.x0);
      const narrower = Math.min(x1 - x0, cur.x1 - cur.x0);
      // same column (share x-extent) and vertically adjacent (small forward gap; a small
      // negative gap tolerates overlapping boxes, a big one means a new column/section).
      if (gap <= gapMax && gap >= -medianH && overlap * 10 >= narrower) {
        cur.x0 = Math.min(cur.x0, x0); cur.y0 = Math.min(cur.y0, y0);
        cur.x1 = Math.max(cur.x1, x1); cur.y1 = Math.max(cur.y1, y1);
        cur.texts.push(l.text); cur.heights.push(y1 - y0);
        continue;
      }
      flush();
    }
    cur = { x0, y0, x1, y1, texts: [l.text], heights: [y1 - y0] };
  }
  flush();
  return blocks;
}

// Decide how to feed the image to Tesseract: estimate its DPI, upscale genuinely low-res images
// (estimated DPI below the floor) by OCR_UPSCALE_FACTOR so recognition is legible, and compute the
// DPI to declare (0 = none). The caller divides recognized coordinates by the returned scale to
// return to the original space, and declares the DPI so layout analysis separates regions. Best-
// effort: any failure returns the original blob with scale 1. Mirrors the desktop app's
// tesseract.go prepareForOCR (docs/PARITY.md).
async function upscaleForOcr(blob, width, height) {
  const dpi = clampDeclaredDpi(estimateDpi(Math.max(width, height)));
  if (estimateDpi(Math.max(width, height)) >= OCR_UPSCALE_DPI_FLOOR || !width || !height) {
    return { image: blob, scale: 1, dpi };
  }
  try {
    const bmp = await createImageBitmap(blob);
    const w = bmp.width * OCR_UPSCALE_FACTOR, h = bmp.height * OCR_UPSCALE_FACTOR;
    const cv = typeof OffscreenCanvas !== "undefined"
      ? new OffscreenCanvas(w, h)
      : Object.assign(document.createElement("canvas"), { width: w, height: h });
    const ctx = cv.getContext("2d");
    ctx.imageSmoothingEnabled = true;
    ctx.imageSmoothingQuality = "high";
    ctx.drawImage(bmp, 0, 0, w, h);
    if (bmp.close) bmp.close();
    const out = cv.convertToBlob ? await cv.convertToBlob() : await new Promise((r) => cv.toBlob(r));
    const upDpi = clampDeclaredDpi(estimateDpi(Math.max(width, height)) * OCR_UPSCALE_FACTOR);
    return out ? { image: out, scale: OCR_UPSCALE_FACTOR, dpi: upDpi } : { image: blob, scale: 1, dpi };
  } catch {
    return { image: blob, scale: 1, dpi };
  }
}

// Returns block-level results with bounding boxes (pixel coords) plus the image's
// natural dimensions, so callers can position plates in percent.
export async function recognize(imageSource, { lang = "eng", onProgress } = {}) {
  return enqueue(async () => {
    const bitmap = await toBitmap(imageSource);
    const worker = await getWorker(lang);
    currentProgress = onProgress || null;
    try {
      const { image, scale, dpi } = await upscaleForOcr(bitmap.source, bitmap.width, bitmap.height);
      // Declare the resolution so layout analysis separates regions Tesseract otherwise merges
      // (adjacent balloons read as one plate). Best-effort: keep going if the param won't set.
      if (dpi > 0) { try { await worker.setParameters({ user_defined_dpi: String(dpi) }); } catch { /* leave it to guess */ } }
      const { data } = await worker.recognize(image, {}, { blocks: true });
      const blocks = clusterLines(collectLines(data, scale));
      await sampleColors(bitmap.source, blocks);
      return { blocks, width: bitmap.width, height: bitmap.height };
    } finally {
      currentProgress = null;
    }
  });
}

// ---- Overlay rendering -----------------------------------------------------
// Shrinks the plate font below the block's raw line height so the recognized text fits
// inside the block box (the opaque box - sized by min-height - is what covers the source,
// independent of the font). Without it a tall title block wraps to more lines than the
// source and the plate grows past its region, colliding with the next plate. Shared with
// the desktop app's overlay.go fontFitFactor (see docs/PARITY.md). 0.92 keeps plate text close to
// the source size while leaving headroom for longer translations and word-wrap slack.
const FONT_FIT = 0.92;

// A container the size of the image (via aspect-ratio) with the image as a base layer
// and one opaque plate per recognized block, positioned/sized in percent so it survives
// responsive scaling. Font-size is expressed in container-width units (cqw) derived from
// the block's median line height, scaled by FONT_FIT so the text fits its block. Plates
// grow downward (min-height) so longer post-translation text wraps instead of clipping.
export function buildOverlay({ imageSrc, imageEl, blocks, width, height }) {
  const container = document.createElement("div");
  container.className = "ocr-overlay";
  if (width && height) container.style.aspectRatio = `${width} / ${height}`;

  const img = imageEl || document.createElement("img");
  if (!imageEl) img.src = imageSrc;
  img.classList.add("ocr-overlay-img");
  container.append(img);

  for (const b of blocks) {
    if (!b.text) continue;
    const { x0, y0, x1, y1 } = b.bbox;
    const plate = document.createElement("div");
    plate.className = "ocr-plate";
    plate.style.left = `${(x0 / width) * 100}%`;
    plate.style.top = `${(y0 / height) * 100}%`;
    plate.style.width = `${((x1 - x0) / width) * 100}%`;
    plate.style.minHeight = `${((y1 - y0) / height) * 100}%`;
    plate.style.fontSize = `${((b.lineHeight / width) * 100 * FONT_FIT).toFixed(2)}cqw`;
    if (b.colors) {
      plate.style.background = b.colors.bg;
      plate.style.color = b.colors.ink;
      plate.style.boxShadow = "none"; // colour matches the image; drop the delineating border
    }
    plate.textContent = b.text;
    container.append(plate);
  }
  scheduleFit(container);
  return container;
}

// fitPlate fits one plate's text to its box: it shrinks the cqw font down to a floor, and if the
// text still overflows there it lets the box grow so nothing is ever clipped. The source region
// height (the inline min-height) is the target the font is fitted to. Mirrors the desktop app's
// ocrScript fit() (see docs/PARITY.md and overlay.go).
function fitPlate(b) {
  if (!b.dataset.ocrCqw) {
    const m = /([0-9.]+)cqw/.exec(b.style.fontSize || "");
    b.dataset.ocrCqw = m ? m[1] : "0";
  }
  const base = parseFloat(b.dataset.ocrCqw);
  b.style.height = "";
  const target = parseFloat(getComputedStyle(b).minHeight) || 0;
  if (target > 0) b.style.height = target + "px";
  if (base > 0) {
    let s = base; const floor = base * 0.5; let g = 0;
    b.style.fontSize = s + "cqw";
    while (b.scrollHeight > b.clientHeight + 1 && s > floor && g < 40) {
      s -= Math.max(0.3, s * 0.08); g++;
      b.style.fontSize = s + "cqw";
    }
  }
  if (b.scrollHeight > b.clientHeight + 1) b.style.height = "auto";
}

// scheduleFit fits every plate in a container once it is laid out in the DOM (the caller appends it
// synchronously, so a setTimeout(0) sees it placed), and re-fits when the page translator swaps a
// plate's text or the container resizes - the compile-time font size is computed from the source
// geometry and cannot know the translated length. Mirrors the desktop app's ocrScript scheduling.
function scheduleFit(container) {
  const fitAll = () => { container.querySelectorAll(".ocr-plate").forEach(fitPlate); };
  let t;
  const go = () => { clearTimeout(t); t = setTimeout(fitAll, 0); };
  go();
  if (typeof window !== "undefined") {
    window.addEventListener("load", go);
    window.addEventListener("resize", go);
  }
  if (typeof MutationObserver !== "undefined") {
    new MutationObserver((muts) => {
      for (const mu of muts) {
        let n = mu.target;
        while (n && n !== container) {
          if (n.classList && n.classList.contains("ocr-plate")) { go(); return; }
          n = n.parentNode;
        }
      }
    }).observe(container, { childList: true, characterData: true, subtree: true });
  }
  if (typeof ResizeObserver !== "undefined") {
    try { new ResizeObserver(go).observe(container); } catch { /* ignore */ }
  }
}

// A progress badge (styled by .ocr-badge) callers overlay on a pending image.
export function makeBadge(text) {
  const badge = document.createElement("div");
  badge.className = "ocr-badge";
  badge.textContent = text;
  return badge;
}

// ---- One-call convenience + language tag -----------------------------------
// Recognize then build the overlay, resolving to the container. Pass an <img> to reuse
// (and move) that element; pass a URL/Blob to create a fresh <img>.
export async function overlayImage(source, { lang = "eng", onProgress } = {}) {
  const isEl = typeof HTMLImageElement !== "undefined" && source instanceof HTMLImageElement;
  const { blocks, width, height } = await recognize(source, { lang, onProgress });
  const container = buildOverlay(
    isEl
      ? { imageEl: source, blocks, width, height }
      : { imageSrc: source, blocks, width, height },
  );
  if (!blocks.length) container.classList.add("ocr-empty");
  return container;
}

// Map a Tesseract language code to a BCP-47 tag for <html lang> (drives the browser's
// translate offer).
const HTML_LANG = {
  eng: "en", rus: "ru", ukr: "uk", jpn: "ja", jpn_vert: "ja",
  deu: "de", fra: "fr", spa: "es", ita: "it", por: "pt", pol: "pl", chi_sim: "zh", kor: "ko",
};
export function ocrLangToHtmlLang(code) {
  return HTML_LANG[code] || "en";
}
