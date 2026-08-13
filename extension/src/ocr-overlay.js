// ocr-overlay.js - the reusable OCR-overlay unit shared by the context-menu page and
// the PDF/EPUB viewer. Recognizes text in an image with a single shared Tesseract worker
// (one image at a time via a FIFO queue), groups results into block-level plates, and
// renders opaque plates carrying real HTML text over the source image so the browser's
// built-in "Translate page" can translate them in place. Progress is reported via
// onProgress({status, progress}) throughout.

import Tesseract from "../vendor/tesseract/tesseract.esm.min.js";
import { workerOptions } from "./ocr-lang.js";
import { clusterLines, medianOf, OCR_MIN_LINE_CONF } from "./ocr-cluster.js";
import { screenPitch, mergeScreenBlocks, OCR_SCREEN_SIGMA_DIVISOR } from "./ocr-screen.js";

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
// The line-to-plate grouping and its shared constants live in ocr-cluster.js, so they can be
// unit-tested without this module's Tesseract worker bundle (see docs/PARITY.md).

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

// Tesseract's `thresholding_method`: 0 is its own Otsu (the engine default), 1 is Leptonica's
// tiled Otsu, which thresholds locally rather than picking one cut-off for the whole picture.
// Shared invariant - see docs/PARITY.md and tesseract.go thresholdEngineDefault.
const THRESHOLD_ENGINE_DEFAULT = "0";
const THRESHOLD_LEPTONICA_OTSU = "1";

// GREY_RESCUE_PASSES are the retries for an image the ordinary pass could not read at all, in the
// order they are tried. Each hands Tesseract a greyscale copy; they differ in who decides where
// ink ends and paper begins.
//
// Why greyscale rescues a picture whose lettering is perfectly legible to a person: Tesseract's
// default thresholder runs Otsu on each RGB channel separately and a pixel counts as ink only
// where every channel agrees. On flat paper the three channels agree and this is harmless, but on
// saturated artwork - a brick-red comic panel behind a white balloon, a coloured poster - each
// channel splits the picture somewhere else, the channels disagree over the lettering, and the
// mask that reaches recognition has no text in it. The second pass then changes who thresholds: a
// single global cut-off cannot survive a background that varies across the image (on a sky
// gradient Otsu splits the gradient itself and a white caption comes out the same value as its
// background), while Leptonica's tiled Otsu decides locally.
//
// This is a ladder rather than a replacement because none of the three is best everywhere: the
// colour pass wins where lettering is separated by hue rather than brightness, and greyscale
// throws that away. Retrying only after the ordinary pass returned no plates at all keeps every
// image that works today unchanged. Mirrors tesseract.go greyRescuePasses (docs/PARITY.md).
const GREY_RESCUE_PASSES = [THRESHOLD_ENGINE_DEFAULT, THRESHOLD_LEPTONICA_OTSU];

// The line-confidence floor for a rescue pass, higher than OCR_MIN_LINE_CONF because a rescue is a
// second guess: the ordinary pass already looked at this image and found nothing, so the prior that
// there is text here at all is weaker, and a plate of invented words painted over artwork is worse
// for the reader than no overlay. Anchored on the same scale - OCR_MIN_LINE_CONF sits at the top of
// the band Tesseract hallucinates in, this sits at the bottom of the band real text occupies
// (~80-97). Measured over the scenes the ladder newly reads: genuine rescued lettering scored
// 93.1-97.0, Cyrillic posters read with English data (where the correct answer is no text) scored
// 50.8. Shared invariant - see docs/PARITY.md and tesseract.go ocrRescueLineConf.
const OCR_RESCUE_LINE_CONF = 80;

// Draw the image through a greyscale filter and return the result as a blob. The canvas stays
// RGBA, so unlike the desktop app's 8-bit PNG the three channels merely hold equal values - which
// reaches the same place, because per-channel Otsu on three identical channels is one decision.
// An extra CSS filter (the screen rung's blur) is appended when one is given. Best-effort: null on
// any failure, and the caller then keeps the empty colour result.
async function greyRendition(blob, extraFilter = "") {
  try {
    const { canvas, bmp } = await greyCanvas(blob, extraFilter);
    if (bmp.close) bmp.close();
    return canvas.convertToBlob ? await canvas.convertToBlob() : await new Promise((r) => canvas.toBlob(r));
  } catch {
    return null;
  }
}

// greyCanvas is the one place the greyscale draw happens, shared by the rendition the grey rungs
// hand to Tesseract and the pixel read the screen rung measures.
async function greyCanvas(blob, extraFilter = "") {
  const bmp = await createImageBitmap(blob);
  const { width: w, height: h } = bmp;
  const canvas = typeof OffscreenCanvas !== "undefined"
    ? new OffscreenCanvas(w, h)
    : Object.assign(document.createElement("canvas"), { width: w, height: h });
  const ctx = canvas.getContext("2d", { willReadFrequently: true });
  ctx.filter = extraFilter ? `grayscale(1) ${extraFilter}` : "grayscale(1)";
  ctx.drawImage(bmp, 0, 0);
  return { canvas, ctx, bmp, width: w, height: h };
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
      let blocks = clusterLines(collectLines(data, scale));
      if (!blocks.length) blocks = await greyRescue(worker, image, scale);
      else blocks = await screenSweep(worker, image, scale, blocks);
      await sampleColors(bitmap.source, blocks);
      return { blocks, width: bitmap.width, height: bitmap.height };
    } finally {
      currentProgress = null;
    }
  });
}

// Walk GREY_RESCUE_PASSES over a greyscale copy and return the first pass that finds any plates,
// then try the halftone-screen rung for a picture no thresholder can see through. The thresholder
// is restored to the engine default afterwards so the shared worker is left as the ordinary pass
// expects to find it. Returns [] when nothing reads, which is the honest answer.
async function greyRescue(worker, image, scale) {
  const grey = await greyRendition(image);
  if (!grey) return [];
  try {
    for (const method of GREY_RESCUE_PASSES) {
      try {
        await worker.setParameters({ thresholding_method: method });
        const { data } = await worker.recognize(grey, {}, { blocks: true });
        const blocks = clusterLines(collectLines(data, scale), OCR_RESCUE_LINE_CONF);
        if (blocks.length) return blocks;
      } catch { /* try the next rung */ }
    }
  } finally {
    try { await worker.setParameters({ thresholding_method: THRESHOLD_ENGINE_DEFAULT }); } catch { /* next recognize resets it */ }
  }
  return screenRescue(worker, image, scale);
}

// The ladder's last rung: measure the halftone screen the picture is printed with and, if there is
// one, hand Tesseract a copy low-passed just wide enough to dissolve it.
//
// Why this is a rung and not a filter applied to every image: the same blur that recovers text
// printed on a screen merges neighbouring words on clean lettering, and on real screened material
// the ordinary passes usually read the text unaided. Firing only after every other rung returned
// nothing keeps that cost where there is nothing left to lose. The sigma comes from the screen's
// own measured period rather than a fixed number, because a screen's pitch depends on the press and
// on the scan resolution. Mirrors tesseract.go screenRescue (docs/PARITY.md); CSS `blur(Npx)` is a
// Gaussian whose standard deviation is N, which is the same kernel the desktop app builds.
async function screenRescue(worker, image, scale) {
  const pitch = await measureScreenPitch(image);
  if (!pitch) return [];
  const blurred = await greyRendition(image, `blur(${pitch / OCR_SCREEN_SIGMA_DIVISOR}px)`);
  if (!blurred) return [];
  try {
    const { data } = await worker.recognize(blurred, {}, { blocks: true });
    return clusterLines(collectLines(data, scale), OCR_RESCUE_LINE_CONF);
  } catch {
    return [];
  }
}

// The screen pass for a page that already read. The ladder above cannot reach it: the ladder fires
// only for an image that produced no plates at all, and on a real comic page the dialogue on clean
// white balloons reads fine while the caption printed as a tint does not - so the page is never
// "unread", the rung never runs, and the caption stays untranslated with no explanation.
//
// So the pass is additive rather than a replacement. It has to be: the screen pass wins on screened
// material and loses badly where there is no screen, so keeping every plate the ordinary pass
// produced and merging only what does not overlap one is what takes the gain without the loss. The
// trigger is the detector restricted to the area no plate covers, which is far cheaper than the
// second recognition it decides against. Every failure returns the input untouched - the page was
// already good enough to show. Mirrors tesseract.go screenSweep (docs/PARITY.md).
//
// The one coordinate difference between the editions: collectLines has already divided these blocks
// by `scale` while the prepared image the detector reads has not been downscaled, so the covered
// rectangles are multiplied back up. It follows from where each edition puts its downscale - the Go
// app does it after the sweep, so there it needs nothing.
async function screenSweep(worker, image, scale, kept) {
  const covered = kept.map(({ bbox: b }) => (scale === 1 ? b : {
    x0: Math.round(b.x0 * scale), y0: Math.round(b.y0 * scale),
    x1: Math.round(b.x1 * scale), y1: Math.round(b.y1 * scale),
  }));
  const pitch = await measureScreenPitch(image, covered);
  if (!pitch) return kept;
  const blurred = await greyRendition(image, `blur(${pitch / OCR_SCREEN_SIGMA_DIVISOR}px)`);
  if (!blurred) return kept;
  try {
    const { data } = await worker.recognize(blurred, {}, { blocks: true });
    return mergeScreenBlocks(kept, clusterLines(collectLines(data, scale), OCR_RESCUE_LINE_CONF));
  } catch {
    return kept;
  }
}

// Read the prepared image's luminance back off a canvas and measure its screen, optionally only
// outside the rectangles already plated. The draw already goes through `grayscale(1)`, so the three
// channels hold the same value and one of them is the luminance the detector wants. Best-effort:
// 0 (no screen) on any failure, which skips the rung.
async function measureScreenPitch(blob, covered = []) {
  try {
    const { ctx, bmp, width, height } = await greyCanvas(blob);
    const { data } = ctx.getImageData(0, 0, width, height);
    if (bmp.close) bmp.close();
    const grey = new Uint8Array(width * height);
    for (let i = 0; i < grey.length; i++) grey[i] = data[i * 4];
    return screenPitch(grey, width, height, covered);
  } catch {
    return 0;
  }
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
