// JPEG XL is decoded locally by a packaged libjxl WASM decoder. PNG gives the
// reader, OCR and saved HTML the same pixels independently of browser codecs.
import { InputLimitError, entryTooLarge } from "./limits.js";

const SIGNATURE = [0, 0, 0, 12, 74, 88, 76, 32, 13, 10, 135, 10];
export function isJxl(data) {
  const b = bytesOf(data);
  return (b[0] === 255 && b[1] === 10) || SIGNATURE.every((v, i) => b[i] === v);
}
function bytesOf(data) {
  return data instanceof Uint8Array ? data : new Uint8Array(data);
}

// Read only the SizeHeader before WASM allocates a raster. Field encodings and
// aspect ratios follow libjxl SizeHeader::VisitFields (headers.cc).
export function jxlSize(data) {
  const b = bytesOf(data);
  let prefix = b;
  if (SIGNATURE.every((v, i) => b[i] === v)) {
    const chunks = [];
    let have = 0;
    for (let pos = 12; pos + 8 <= b.length && have < 16;) {
      const dv = new DataView(b.buffer, b.byteOffset + pos, b.length - pos);
      let size = dv.getUint32(0), start = pos + 8;
      const type = String.fromCharCode(...b.subarray(pos + 4, pos + 8));
      if (size === 1) {
        if (pos + 16 > b.length) throw new Error("Truncated JPEG XL box");
        size = dv.getUint32(8) * 2 ** 32 + dv.getUint32(12);
        start += 8;
      } else if (size === 0) size = b.length - pos;
      if (!Number.isSafeInteger(size) || size < start - pos || size > b.length - pos) throw new Error("Invalid JPEG XL box");
      if (type === "jxlc" || type === "jxlp") {
        if (type === "jxlp") start += 4;
        if (start > pos + size) throw new Error("Truncated JPEG XL part");
        const part = b.subarray(start, Math.min(pos + size, start + 16 - have));
        chunks.push(part); have += part.length;
      }
      pos += size;
    }
    prefix = new Uint8Array(have);
    let offset = 0;
    for (const part of chunks) { prefix.set(part, offset); offset += part.length; }
  }
  if (prefix[0] !== 255 || prefix[1] !== 10) throw new Error("Invalid JPEG XL signature");
  let bit = 16;
  const read = (n) => {
    if (bit + n > prefix.length * 8) throw new Error("Truncated JPEG XL size header");
    let v = 0;
    for (let i = 0; i < n; i++, bit++) v += ((prefix[bit >> 3] >> (bit & 7)) & 1) * 2 ** i;
    return v;
  };
  const small = read(1);
  const side = () => small ? (read(5) + 1) * 8 : read([9, 13, 18, 30][read(2)]) + 1;
  const height = side(), ratio = read(3);
  const ratios = [[0, 1], [1, 1], [12, 10], [4, 3], [3, 2], [16, 9], [5, 4], [2, 1]];
  const width = ratio ? Math.floor(height * ratios[ratio][0] / ratios[ratio][1]) : side();
  if (!width || width > 32768 || height > 32768 || width * height > 100_000_000) {
    throw new InputLimitError("vLimitPixels", "The image is {1} x {2} pixels, above the limit of {3} megapixels and {4} pixels per side", width, height, 100, 32768);
  }
  return { width, height };
}

export async function decodeJxl(data) {
  const b = bytesOf(data);
  if (b.byteLength > 200 * 1024 * 1024) throw entryTooLarge("JPEG XL", 200 * 1024 * 1024);
  jxlSize(b);
  const { default: decode } = await import("../vendor/jxl/decode.js");
  return decode(b);
}

export async function imageBlob(data, mime = "") {
  if (mime !== "image/jxl" && !isJxl(data)) return new Blob([data], mime ? { type: mime } : undefined);
  const pixels = await decodeJxl(data);
  const canvas = new OffscreenCanvas(pixels.width, pixels.height);
  canvas.getContext("2d").putImageData(pixels, 0, 0);
  return canvas.convertToBlob({ type: "image/png" });
}

// Normalize embedded local/data pictures before attaching a parsed book. Remote
// pictures retain the reader's existing permission decision and fetch policy.
export async function normalizeJxlImages(book) {
  const created = [];
  const revoke = book.revoke;
  book.revoke = () => { for (const u of created) URL.revokeObjectURL(u); revoke?.(); };
  try {
    for (const section of book.sections) {
      for (const img of section.frag.querySelectorAll("img[src]")) {
        const src = img.getAttribute("src");
        if (!src.startsWith("blob:") && !/^data:image\/jxl[;,]/i.test(src)) continue;
        const blob = await (await fetch(src)).blob();
        if (blob.type !== "image/jxl" && !isJxl(await blob.slice(0, 12).arrayBuffer())) continue;
        const png = await imageBlob(await blob.arrayBuffer(), "image/jxl");
        const url = URL.createObjectURL(png);
        created.push(url);
        img.setAttribute("src", url);
      }
    }
  } catch (err) { book.revoke(); throw err; }
}
