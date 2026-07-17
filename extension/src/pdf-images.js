// pdf-images.js - pull raster images out of a PDF page for OCR. The primary path walks
// the page operator list for image XObjects and decodes them to PNG blobs; when a page
// yields no usable image but is image-dominant (a scanned page), rasterizePage() renders
// the whole page as one fallback image. All failures are swallowed - a page that can't be
// mined for images simply contributes none.

import * as pdfjsLib from "../vendor/pdf.mjs";

const { OPS } = pdfjsLib;

// Resolve an image object from page.objs / page.commonObjs. get() takes a callback that
// fires once the object is ready; after getOperatorList() it normally already is. A
// timeout guards against a callback that never fires.
function getObj(store, name) {
  return new Promise((resolve) => {
    let done = false;
    const finish = (o) => { if (!done) { done = true; resolve(o || null); } };
    try { store.get(name, finish); } catch { finish(null); }
    setTimeout(() => finish(null), 3000);
  });
}

async function resolveImage(page, name) {
  let obj = await getObj(page.objs, name);
  if (!obj) obj = await getObj(page.commonObjs, name);
  return obj;
}

// Convert a pdf.js image object's raw bytes to RGBA. Returns null for packed 1-bpp or
// otherwise unrecognized layouts (those are skipped rather than mis-rendered).
function toRGBA(data, width, height) {
  const n = width * height;
  const out = new Uint8ClampedArray(n * 4);
  if (data.length >= n * 4) {
    out.set(data.subarray(0, n * 4));
  } else if (data.length >= n * 3) {
    for (let i = 0, j = 0; i < n; i++) {
      out[j++] = data[i * 3];
      out[j++] = data[i * 3 + 1];
      out[j++] = data[i * 3 + 2];
      out[j++] = 255;
    }
  } else if (data.length >= n) {
    for (let i = 0, j = 0; i < n; i++) {
      const v = data[i];
      out[j++] = v; out[j++] = v; out[j++] = v; out[j++] = 255;
    }
  } else {
    return null;
  }
  return out;
}

// Compose two PDF transform matrices [a b c d e f]; m2 is applied first
// (same semantics as pdf.js Util.transform and canvas ctx.transform).
export function composeTransform(m1, m2) {
  return [
    m1[0] * m2[0] + m1[2] * m2[1],
    m1[1] * m2[0] + m1[3] * m2[1],
    m1[0] * m2[2] + m1[2] * m2[3],
    m1[1] * m2[2] + m1[3] * m2[3],
    m1[0] * m2[4] + m1[2] * m2[5] + m1[4],
    m1[1] * m2[4] + m1[3] * m2[5] + m1[5],
  ];
}

// Flips that make extracted pixels match the image's rendered orientation.
// The standard placement matrix [w 0 0 h x y] has positive scales and renders
// the stored raster as-is; a negative scale means the PDF stores that axis
// mirrored and un-mirrors it at draw time — which raw XObject extraction
// bypasses. Rotated placements (dominant b/c terms) are left as stored.
export function paintFlips(ctm) {
  const [a, b, c, d] = ctm;
  if (Math.abs(b) + Math.abs(c) > Math.abs(a) + Math.abs(d)) return { flipX: false, flipY: false };
  return { flipX: a < 0, flipY: d < 0 };
}

function applyFlips(ctx, width, height, { flipX, flipY }) {
  ctx.translate(flipX ? width : 0, flipY ? height : 0);
  ctx.scale(flipX ? -1 : 1, flipY ? -1 : 1);
}

const NO_FLIPS = { flipX: false, flipY: false };

async function bitmapToBlob(bitmap, flips) {
  const canvas = new OffscreenCanvas(bitmap.width, bitmap.height);
  const ctx = canvas.getContext("2d");
  applyFlips(ctx, bitmap.width, bitmap.height, flips);
  ctx.drawImage(bitmap, 0, 0);
  return canvas.convertToBlob({ type: "image/png" });
}

async function imageObjToBlob(imgObj, flips = NO_FLIPS) {
  if (!imgObj) return null;
  if (typeof ImageBitmap !== "undefined" && imgObj instanceof ImageBitmap) {
    return bitmapToBlob(imgObj, flips);
  }
  if (imgObj.bitmap && typeof ImageBitmap !== "undefined" && imgObj.bitmap instanceof ImageBitmap) {
    return bitmapToBlob(imgObj.bitmap, flips);
  }
  if (imgObj.data && imgObj.width && imgObj.height) {
    const rgba = toRGBA(imgObj.data, imgObj.width, imgObj.height);
    if (!rgba) return null;
    const canvas = new OffscreenCanvas(imgObj.width, imgObj.height);
    const ctx = canvas.getContext("2d");
    const imageData = new ImageData(rgba, imgObj.width, imgObj.height);
    if (flips.flipX || flips.flipY) {
      // putImageData ignores the canvas transform — stage on a second canvas.
      const staged = new OffscreenCanvas(imgObj.width, imgObj.height);
      staged.getContext("2d").putImageData(imageData, 0, 0);
      applyFlips(ctx, imgObj.width, imgObj.height, flips);
      ctx.drawImage(staged, 0, 0);
    } else {
      ctx.putImageData(imageData, 0, 0);
    }
    return canvas.convertToBlob({ type: "image/png" });
  }
  return null;
}

// ASPECT_RATIO_TOLERANCE mirrors the desktop app (internal/pdf/extract.go): two rasters
// whose aspect ratios match within 1% are the same picture at a different resolution.
const ASPECT_RATIO_TOLERANCE = 0.01;

// sameShapeRaster reports whether two rasters are the same picture at a different scale:
// both dimensions positive and their aspect ratios equal within tolerance. A uniform
// scale preserves the aspect ratio, so equal ratios is the signal for "a resized copy".
export function sameShapeRaster(a, b) {
  if (!a.width || !a.height || !b.width || !b.height) return false;
  const ra = a.width / a.height;
  const rb = b.width / b.height;
  return Math.abs(ra - rb) <= ASPECT_RATIO_TOLERANCE * Math.max(ra, rb);
}

// dedupeSameShape collapses proportional-scale duplicates - the same scanned page
// embedded twice at two resolutions - keeping the largest, so a page is not shown twice.
// Differently-shaped images (a composed page: an illustration beside a figure) are all
// kept. Mirrors selectPageImages in the desktop app (internal/pdf/extract.go); the app
// also drops /Thumb previews, which never reach here because a thumbnail is a page-dict
// entry that the content stream never paints.
export function dedupeSameShape(imgs) {
  const kept = [];
  for (const im of imgs) {
    let merged = false;
    for (let k = 0; k < kept.length; k++) {
      if (sameShapeRaster(kept[k], im)) {
        if (im.width * im.height > kept[k].width * kept[k].height) kept[k] = im;
        merged = true;
        break;
      }
    }
    if (!merged) kept.push(im);
  }
  return kept;
}

// Extract embedded raster images from a page as { blob, width, height }. Images smaller
// than minSize on a side are skipped (icons, bullets, rules).
export async function extractPageImages(page, { minSize = 64 } = {}) {
  const out = [];
  let opList;
  try {
    opList = await page.getOperatorList();
  } catch {
    return out;
  }
  const seen = new Set();
  // Track the CTM so images placed with a mirroring matrix can be normalized.
  let ctm = [1, 0, 0, 1, 0, 0];
  const ctmStack = [];
  for (let i = 0; i < opList.fnArray.length; i++) {
    const fn = opList.fnArray[i];
    if (fn === OPS.save) { ctmStack.push(ctm); continue; }
    if (fn === OPS.restore) { if (ctmStack.length) ctm = ctmStack.pop(); continue; }
    if (fn === OPS.transform) { ctm = composeTransform(ctm, opList.argsArray[i]); continue; }
    if (fn !== OPS.paintImageXObject && fn !== OPS.paintInlineImageXObject) continue;
    try {
      const args = opList.argsArray[i];
      let imgObj;
      if (fn === OPS.paintInlineImageXObject) {
        imgObj = args[0];
      } else {
        const name = args[0];
        if (seen.has(name)) continue;
        seen.add(name);
        imgObj = await resolveImage(page, name);
      }
      const width = imgObj && imgObj.width;
      const height = imgObj && imgObj.height;
      if (!width || !height || width < minSize || height < minSize) continue;
      const blob = await imageObjToBlob(imgObj, paintFlips(ctm));
      if (blob) out.push({ blob, width, height });
    } catch {
      /* skip this image */
    }
  }
  return dedupeSameShape(out);
}

// Render the whole page as one image - the fallback for scanned pages that are a single
// full-page image with no separately-extractable XObject.
export async function rasterizePage(page, { scale = 2 } = {}) {
  try {
    const viewport = page.getViewport({ scale });
    const canvas = new OffscreenCanvas(Math.ceil(viewport.width), Math.ceil(viewport.height));
    const ctx = canvas.getContext("2d");
    await page.render({ canvasContext: ctx, viewport }).promise;
    const blob = await canvas.convertToBlob({ type: "image/png" });
    return { blob, width: canvas.width, height: canvas.height };
  } catch {
    return null;
  }
}
