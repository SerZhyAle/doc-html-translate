// comic.js - open a comic-book archive (CBZ / CBT) as a sequence of page images.
//
// A comic archive is a container of page images with no text layer at all - every
// word lives inside the artwork. So, like a standalone image, there is nothing to
// parse into semantic text: the viewer renders each page and OCRs it into
// translatable plates (forced on, independent of the "Use OCR for images" toggle),
// so the browser's built-in "Translate page" works on the speech bubbles.
//
// This mirrors the desktop app's internal/comic package - the natural page order
// and the page-entry filter are shared invariants that must match (see
// docs/PARITY.md). The desktop app also handles CBR (RAR) and CB7 (7z) by shelling
// out to 7-Zip; the browser has no RAR/7z decoder and cannot shell out, so those
// two are declined here with a clear "desktop app only" notice (detectContainer
// recognizes their signatures to give that message rather than a parse error).
//
// The pure pieces - naturalCompare(), isPageEntry(), the archive walkers - use no
// DOM and are unit-tested under node. Inflation is lazy: the walkers return one
// entry per page carrying a load() that inflates just that page on demand, so a
// large comic never inflates every page into memory at once.

// PAGE_EXTS are the image extensions that count as a comic page. TIFF is excluded
// (browsers cannot display it and it is vanishingly rare in comics). This set must
// match internal/comic pageExts on the Go side - see docs/PARITY.md.
export const PAGE_EXTS = ["png", "jpg", "jpeg", "gif", "webp", "bmp"];

const MIME_BY_EXT = {
  png: "image/png",
  jpg: "image/jpeg",
  jpeg: "image/jpeg",
  gif: "image/gif",
  webp: "image/webp",
  bmp: "image/bmp",
};

// isPageEntry reports whether an archive entry name is a comic page: a regular
// file with a page image extension, not a directory, not archive metadata
// (ComicInfo.xml), not OS cruft (Thumbs.db, .DS_Store, anything under __MACOSX/),
// and not a hidden dotfile. Must match internal/comic isPageEntry on the Go side,
// or the two editions disagree on page count and numbering for the same file.
export function isPageEntry(name) {
  name = String(name).replace(/\\/g, "/");
  if (name === "" || name.endsWith("/")) return false; // directory entry
  if (name.startsWith("__MACOSX/") || name.includes("/__MACOSX/")) return false;
  const slash = name.lastIndexOf("/");
  const base = slash >= 0 ? name.slice(slash + 1) : name;
  if (base === "" || base.startsWith(".")) return false; // hidden dotfile
  const lower = base.toLowerCase();
  if (lower === "thumbs.db" || lower === "comicinfo.xml") return false;
  return PAGE_EXTS.includes(extOf(base));
}

function extOf(name) {
  const dot = name.lastIndexOf(".");
  return dot >= 0 ? name.slice(dot + 1).toLowerCase() : "";
}

// naturalCompare orders two names numeric-aware: runs of ASCII digits compare by
// value, so "page2.jpg" sorts before "page10.jpg". Returns <0, 0 or >0 for use as
// an Array.sort comparator. Load-bearing for comics - page order *is* archive
// entry order by filename - so it must match internal/comic naturalLess on the Go
// side (equal value breaks toward the shorter raw run). See docs/PARITY.md.
export function naturalCompare(a, b) {
  const la = String(a).toLowerCase();
  const lb = String(b).toLowerCase();
  let i = 0;
  let j = 0;
  while (i < la.length && j < lb.length) {
    if (isDigit(la[i]) && isDigit(lb[j])) {
      const [si, ei] = digitRun(la, i);
      const [sj, ej] = digitRun(lb, j);
      const v = compareDigits(la.slice(si, ei), lb.slice(sj, ej));
      if (v !== 0) return v;
      // Equal numeric value: fewer leading zeros (shorter raw run) first.
      if (ei - i !== ej - j) return (ei - i) - (ej - j);
      i = ei;
      j = ej;
      continue;
    }
    if (la[i] !== lb[j]) return la[i] < lb[j] ? -1 : 1;
    i++;
    j++;
  }
  return (la.length - i) - (lb.length - j);
}

function isDigit(c) { return c >= "0" && c <= "9"; }

// digitRun returns [sigStart, end) for the digit run beginning at pos, with
// leading zeros stripped from sigStart (at least one digit is always kept).
function digitRun(s, pos) {
  let end = pos;
  while (end < s.length && isDigit(s[end])) end++;
  let sigStart = pos;
  while (sigStart < end - 1 && s[sigStart] === "0") sigStart++;
  return [sigStart, end];
}

function compareDigits(x, y) {
  if (x.length !== y.length) return x.length < y.length ? -1 : 1;
  if (x < y) return -1;
  if (x > y) return 1;
  return 0;
}

// detectContainer classifies the archive bytes by signature: "zip" (CBZ), "rar"
// (CBR), "7z" (CB7), or "tar" (CBT / unknown fallback - TAR has no leading magic).
export function detectContainer(u8) {
  if (u8.length >= 4 && u8[0] === 0x50 && u8[1] === 0x4b && u8[2] === 0x03 && u8[3] === 0x04) return "zip"; // PK\x03\x04
  if (u8.length >= 6 && u8[0] === 0x52 && u8[1] === 0x61 && u8[2] === 0x72 && u8[3] === 0x21) return "rar"; // Rar!
  if (u8.length >= 6 && u8[0] === 0x37 && u8[1] === 0x7a && u8[2] === 0xbc && u8[3] === 0xaf) return "7z";   // 7z\xBC\xAF
  return "tar";
}

// DesktopOnlyError signals a CBR/CB7 the browser cannot open; the viewer turns it
// into a "use the desktop app" notice rather than a generic parse failure.
export class DesktopOnlyError extends Error {}

// parseComic reads a comic archive and returns its page list in natural order.
// Each page is { name, mime, load } where load() inflates just that page's bytes
// on demand (Uint8Array). Throws DesktopOnlyError for CBR/CB7 and a plain Error
// for a container with no page images.
export async function parseComic(arrayBuffer) {
  const u8 = new Uint8Array(arrayBuffer);
  const kind = detectContainer(u8);
  if (kind === "rar" || kind === "7z") {
    throw new DesktopOnlyError(
      `${kind === "rar" ? "CBR (RAR)" : "CB7 (7z)"} comics need the doc-html-translate desktop app - ` +
      "the browser has no RAR/7z decoder. CBZ and CBT comics open here.",
    );
  }

  const entries = kind === "zip" ? await zipEntries(arrayBuffer) : tarEntries(u8);
  const pages = entries.filter((e) => isPageEntry(e.name));
  if (pages.length === 0) {
    throw new Error("no page images found in this comic archive");
  }
  pages.sort((a, b) => naturalCompare(a.name, b.name));
  return pages.map((e) => ({ name: e.name, mime: MIME_BY_EXT[extOf(e.name)] || "image/jpeg", load: e.load }));
}

// ---- ZIP reader (lazy) -----------------------------------------------------
// A central-directory-driven reader that returns a load() per entry instead of
// inflating everything up front (the memory-safe half of the EPUB unzip). Deflate
// is inflated with the platform's native DecompressionStream, so no dependency
// ships. The compressed archive stays in memory once; a page is inflated only when
// its load() is called (on scroll).

const SIG_EOCD = 0x06054b50;
const SIG_CEN = 0x02014b50;

async function inflateRaw(bytes) {
  const stream = new Blob([bytes]).stream().pipeThrough(new DecompressionStream("deflate-raw"));
  return new Uint8Array(await new Response(stream).arrayBuffer());
}

function findEOCD(dv, len) {
  const min = Math.max(0, len - 22 - 0xffff);
  for (let i = len - 22; i >= min; i--) {
    if (dv.getUint32(i, true) === SIG_EOCD) return i;
  }
  return -1;
}

async function zipEntries(arrayBuffer) {
  const u8 = new Uint8Array(arrayBuffer);
  const dv = new DataView(arrayBuffer);
  if (u8.length < 22) throw new Error("not a ZIP archive (too small)");
  const eocd = findEOCD(dv, u8.length);
  if (eocd < 0) throw new Error("not a ZIP archive (no end-of-central-directory record)");
  const count = dv.getUint16(eocd + 10, true);
  const cdOffset = dv.getUint32(eocd + 16, true);
  if (cdOffset === 0xffffffff) throw new Error("ZIP64 archives are not supported");

  const dec = new TextDecoder("utf-8");
  const out = [];
  let p = cdOffset;
  for (let i = 0; i < count; i++) {
    if (p + 46 > u8.length || dv.getUint32(p, true) !== SIG_CEN) break;
    const method = dv.getUint16(p + 10, true);
    const compSize = dv.getUint32(p + 20, true);
    const nameLen = dv.getUint16(p + 28, true);
    const extraLen = dv.getUint16(p + 30, true);
    const commentLen = dv.getUint16(p + 32, true);
    const localOff = dv.getUint32(p + 42, true);
    const name = dec.decode(u8.subarray(p + 46, p + 46 + nameLen));
    p += 46 + nameLen + extraLen + commentLen;
    if (name.endsWith("/")) continue;

    const lhNameLen = dv.getUint16(localOff + 26, true);
    const lhExtraLen = dv.getUint16(localOff + 28, true);
    const dataStart = localOff + 30 + lhNameLen + lhExtraLen;
    const comp = u8.subarray(dataStart, dataStart + compSize);
    out.push({
      name,
      load: async () => {
        if (method === 0) return comp.slice();
        if (method === 8) return inflateRaw(comp);
        throw new Error(`unsupported ZIP compression method ${method} for ${name}`);
      },
    });
  }
  return out;
}

// ---- TAR reader (lazy) -----------------------------------------------------
// TAR is a flat sequence of 512-byte header blocks each followed by the file's
// bytes padded to 512. No compression, so an entry's bytes are just a subarray of
// the buffer - load() slices lazily. Only regular files are returned.

function tarEntries(u8) {
  const out = [];
  const dec = new TextDecoder("utf-8");
  let off = 0;
  while (off + 512 <= u8.length) {
    const block = u8.subarray(off, off + 512);
    // Two consecutive zero blocks mark the end of the archive.
    if (block.every((b) => b === 0)) break;
    let name = dec.decode(block.subarray(0, 100)).replace(/\0.*$/, "");
    const size = parseOctal(block.subarray(124, 136));
    const typeflag = block[156];
    // Prefix field (GNU/USTAR long paths under 155 bytes).
    const prefix = dec.decode(block.subarray(345, 500)).replace(/\0.*$/, "");
    if (prefix) name = `${prefix}/${name}`;
    const dataStart = off + 512;
    off = dataStart + Math.ceil(size / 512) * 512;
    // typeflag '0' or '\0' is a regular file; skip directories and other types.
    if (typeflag === 0x30 || typeflag === 0x00) {
      const start = dataStart;
      const end = dataStart + size;
      out.push({ name, load: async () => u8.slice(start, end) });
    }
  }
  return out;
}

function parseOctal(bytes) {
  let s = "";
  for (const b of bytes) {
    if (b === 0 || b === 0x20) continue;
    s += String.fromCharCode(b);
  }
  return s ? parseInt(s, 8) : 0;
}
