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

async function bitmapToBlob(bitmap) {
  const canvas = new OffscreenCanvas(bitmap.width, bitmap.height);
  canvas.getContext("2d").drawImage(bitmap, 0, 0);
  return canvas.convertToBlob({ type: "image/png" });
}

async function imageObjToBlob(imgObj) {
  if (!imgObj) return null;
  if (typeof ImageBitmap !== "undefined" && imgObj instanceof ImageBitmap) {
    return bitmapToBlob(imgObj);
  }
  if (imgObj.bitmap && typeof ImageBitmap !== "undefined" && imgObj.bitmap instanceof ImageBitmap) {
    return bitmapToBlob(imgObj.bitmap);
  }
  if (imgObj.data && imgObj.width && imgObj.height) {
    const rgba = toRGBA(imgObj.data, imgObj.width, imgObj.height);
    if (!rgba) return null;
    const canvas = new OffscreenCanvas(imgObj.width, imgObj.height);
    canvas.getContext("2d").putImageData(new ImageData(rgba, imgObj.width, imgObj.height), 0, 0);
    return canvas.convertToBlob({ type: "image/png" });
  }
  return null;
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
  for (let i = 0; i < opList.fnArray.length; i++) {
    const fn = opList.fnArray[i];
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
      const blob = await imageObjToBlob(imgObj);
      if (blob) out.push({ blob, width, height });
    } catch {
      /* skip this image */
    }
  }
  return out;
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
