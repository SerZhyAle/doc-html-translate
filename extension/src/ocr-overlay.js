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

async function getWorker(lang) {
  if (workerPromise && workerLang === lang) return workerPromise;
  if (workerPromise) {
    try { (await workerPromise).terminate(); } catch { /* ignore */ }
    workerPromise = null;
  }
  workerLang = lang;
  const options = workerOptions(lang);
  options.logger = (m) => { if (currentProgress) currentProgress(m); };
  workerPromise = createWorker(lang, 1, options);
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
// The lines of a recognized block. Tesseract.js v5 nests lines under
// block.paragraphs[].lines[] (the block itself has no `.lines`); older shapes exposed
// block.lines directly. We gather whichever exists so the plate font-size tracks the real
// median line height (mirroring the desktop app's per-line sizing). Without this the code
// falls back to the whole-block height, producing an absurd cqw font-size and a plate that
// swallows the page.
function blockLines(b) {
  if (Array.isArray(b.lines) && b.lines.length) return b.lines;
  if (Array.isArray(b.paragraphs)) return b.paragraphs.flatMap((p) => p.lines || []);
  return [];
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

// Returns block-level results with bounding boxes (pixel coords) plus the image's
// natural dimensions, so callers can position plates in percent.
export async function recognize(imageSource, { lang = "eng", onProgress } = {}) {
  return enqueue(async () => {
    const bitmap = await toBitmap(imageSource);
    const worker = await getWorker(lang);
    currentProgress = onProgress || null;
    try {
      const { data } = await worker.recognize(bitmap.source, {}, { blocks: true });
      const blocks = (data.blocks || [])
        .map((b) => {
          const text = (b.text || "").trim();
          const lines = blockLines(b);
          let lineHeight = b.bbox.y1 - b.bbox.y0;
          if (lines.length) {
            const hs = lines.map((l) => l.bbox.y1 - l.bbox.y0).sort((a, z) => a - z);
            lineHeight = hs[Math.floor(hs.length / 2)] || lineHeight;
          }
          return { text, bbox: b.bbox, lineHeight };
        })
        .filter((b) => isTranslatable(b.text));
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
// the desktop app's overlay.go fontFitFactor (see docs/PARITY.md).
const FONT_FIT = 0.85;

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
  return container;
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
