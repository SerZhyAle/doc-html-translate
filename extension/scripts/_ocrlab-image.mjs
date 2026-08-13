// _ocrlab-image.mjs - the lab's screenshot-to-image-space step, the counterpart of CropToImage in
// tools/ocrlab/runner/browser.go.
//
// Why the runner assembles a shot itself instead of asking Chrome for one finished picture. Every
// pixel measurement compares the rendered page with the source image coordinate for coordinate, so
// a captured stress shot has to be in the source's own pixels. The obvious way to get there is to
// hand Page.captureScreenshot a clip scaled up to the natural size and let the browser render at
// that resolution - and that is what this runner used to do. It cannot work: the reply travels back
// as one base64 websocket message, and a message past roughly four megabytes is dropped and takes
// the connection with it. Measured on 2026-08-12 (see
// DEV/plan/2026-08-12_extension-crashes-the-tab-on-a-detailed-scan.md): replies up to 4 121 988
// bytes arrived, larger ones killed the socket, and Chrome itself went on running throughout.
//
// So the page is captured in horizontal bands small enough that no single reply can approach that,
// and the bands are stitched and resampled here. It also puts the two editions on the same footing:
// the Go runner captures at the device scale and resamples locally, so both now measure a resampled
// render rather than one edition measuring a natively re-rastered one.

import { createCanvas, loadImage } from "@napi-rs/canvas";

// CaptureBandPx is the largest area, in captured pixels, the runner will ask for in one reply.
// Derived rather than chosen: a page of pure noise - the worst case a PNG can present - encodes to
// about 2.0 MB of base64 per megapixel, measured over the same probe, so 1.5 Mpx worst-cases to
// about 3.0 MB against a limit that took 4.12 MB. Ordinary pages are far below that; the margin is
// there for the ones that are not.
export const CaptureBandPx = 1_500_000;

// bandsFor splits a region of `width` x `height` captured pixels into horizontal bands, each within
// CaptureBandPx. Boundaries are computed from the whole, never accumulated, so a band's rounding
// cannot drift into the next one and shift the bottom of the image against the source it is about
// to be compared with.
export function bandsFor(width, height) {
  const w = Math.max(1, Math.round(width));
  const h = Math.max(1, Math.round(height));
  const count = Math.max(1, Math.ceil((w * h) / CaptureBandPx));
  const out = [];
  for (let i = 0; i < count; i++) {
    const y0 = Math.round((i * h) / count);
    const y1 = Math.round(((i + 1) * h) / count);
    if (y1 > y0) out.push({ y0, y1 });
  }
  return out;
}

// assembleToNatural stitches the captured bands back into one picture and returns it as a PNG of
// exactly naturalWidth x naturalHeight, which is the space every metric indexes. Each part carries
// the band it was captured for, so a part is placed by where it belongs rather than by its own size.
export async function assembleToNatural(parts, width, height, naturalWidth, naturalHeight) {
  if (!(naturalWidth > 0) || !(naturalHeight > 0)) {
    throw new Error(`cannot map a screenshot into a ${naturalWidth}x${naturalHeight} image space`);
  }
  if (!parts.length) throw new Error("no captured bands to assemble");
  const w = Math.max(1, Math.round(width));
  const h = Math.max(1, Math.round(height));

  const stitched = createCanvas(w, h);
  const sctx = stitched.getContext("2d");
  for (const part of parts) {
    const img = await loadImage(part.png);
    if (img.width < 2 || img.height < 2) {
      throw new Error(`a captured band is ${img.width}x${img.height}, too small to have caught the page`);
    }
    sctx.drawImage(img, 0, 0, img.width, img.height, 0, part.y0, w, part.y1 - part.y0);
  }

  const canvas = createCanvas(naturalWidth, naturalHeight);
  const ctx = canvas.getContext("2d");
  ctx.imageSmoothingEnabled = true;
  ctx.imageSmoothingQuality = "high";
  ctx.drawImage(stitched, 0, 0, w, h, 0, 0, naturalWidth, naturalHeight);
  return canvas.toBuffer("image/png");
}
