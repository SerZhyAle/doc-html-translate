// charset.js - decode an EPUB chapter or an HTML file to text by the encoding it declares, the
// way the desktop app does: internal/epub/charset.go decodeToUTF8 for chapters, and
// internal/htmlconv parseDocument (golang.org/x/net/html/charset DetermineEncoding) for HTML
// input. Both used to be read as UTF-8 whatever they declared, so a windows-1251 chapter came out
// as mojibake here while the desktop edition read it (docs/PARITY.md, "EPUB and HTML content
// fidelity"). Labels are WHATWG labels on both sides, which TextDecoder takes directly.
//
// Pure (no DOM) - unit-tested under node against the shared fixture
// tests/testdata/content_fidelity_cases.json.

// Both Go readers look for a declaration in the first 1024 bytes only, as the HTML encoding
// sniffing prescan does.
const PRESCAN_BYTES = 1024;

const XML_DECL_ENCODING = /^[\t\n\f\r ]*<\?xml[\t\n\f\r ][^>]*?\bencoding[\t\n\f\r ]*=[\t\n\f\r ]*["']([A-Za-z0-9._:-]+)["']/;

function bomEncoding(b) {
  if (b.length >= 3 && b[0] === 0xef && b[1] === 0xbb && b[2] === 0xbf) return "utf-8";
  if (b.length >= 2 && b[0] === 0xff && b[1] === 0xfe) return "utf-16le";
  if (b.length >= 2 && b[0] === 0xfe && b[1] === 0xff) return "utf-16be";
  return "";
}

// The head as one character per byte: declarations are ASCII, whatever the document is in.
function latin1Head(b) {
  let s = "";
  for (let i = 0; i < Math.min(b.length, PRESCAN_BYTES); i++) s += String.fromCharCode(b[i]);
  return s;
}

// metaCharset returns the charset a <meta charset> or a Content-Type http-equiv <meta> declares
// in head, or "". Mirrors metaCharset in internal/epub/charset.go.
export function metaCharset(head) {
  for (const m of head.matchAll(/<meta\b([^>]*)>/gi)) {
    const attrs = new Map();
    for (const a of m[1].matchAll(/([^\s=/>]+)\s*=\s*("[^"]*"|'[^']*'|[^\s>]+)/g)) {
      const key = a[1].toLowerCase();
      if (!attrs.has(key)) attrs.set(key, a[2].replace(/^["']|["']$/g, ""));
    }
    if (attrs.has("charset")) return attrs.get("charset").trim();
    if ((attrs.get("http-equiv") || "").trim().toLowerCase() === "content-type") {
      const content = attrs.get("content") || "";
      const i = content.toLowerCase().indexOf("charset=");
      if (i >= 0) return content.slice(i + "charset=".length).trim().replace(/^["']+|["']+$/g, "").split(/[; \t]/)[0];
    }
  }
  return "";
}

// canonical resolves a WHATWG label to its encoding name, or "" for a label TextDecoder does not
// know.
function canonical(label) {
  try {
    return new TextDecoder(label).encoding;
  } catch {
    return "";
  }
}

function decode(bytes, encoding) {
  return new TextDecoder(encoding).decode(bytes);
}

function isUtf8(bytes) {
  try {
    new TextDecoder("utf-8", { fatal: true }).decode(bytes);
    return true;
  } catch {
    return false;
  }
}

// decodeChapter reads an EPUB content document: a BOM wins, then the XML declaration, then a
// <meta> charset; with none the document is UTF-8, the XML default. A label read from ASCII bytes
// cannot mean UTF-16, so such a label - like an unknown one - falls back to UTF-8.
export function decodeChapter(data) {
  if (!data) return "";
  const b = data instanceof Uint8Array ? data : new Uint8Array(data);
  const bom = bomEncoding(b);
  if (bom) return decode(b, bom);
  const head = latin1Head(b);
  const m = XML_DECL_ENCODING.exec(head);
  const enc = canonical(m ? m[1] : metaCharset(head));
  if (!enc || enc === "utf-8" || enc.startsWith("utf-16")) return decode(b, "utf-8");
  return decode(b, enc);
}

// decodeHtml reads an HTML file: a BOM wins, then a <meta> charset (a UTF-16 label read from
// ASCII means UTF-8, as the HTML prescan rules), then detection - bytes that are valid UTF-8 are
// UTF-8, anything else windows-1252, the web's fallback. windows-1252 is also a common mislabel,
// so a page declared windows-1252 (or Latin-1) whose bytes are valid UTF-8 reads as UTF-8.
export function decodeHtml(data) {
  const b = data instanceof Uint8Array ? data : new Uint8Array(data);
  const bom = bomEncoding(b);
  if (bom) return decode(b, bom);
  let enc = canonical(metaCharset(latin1Head(b)));
  if (enc.startsWith("utf-16")) enc = "utf-8";
  if (!enc || enc === "windows-1252") {
    if (isUtf8(b)) return decode(b, "utf-8");
    // Detection looks at the head alone: high bytes there that read as UTF-8 make the page
    // UTF-8 even when a later byte is broken. A multi-byte character at the head's end is cut
    // off first, as DetermineEncoding cuts it, since the window may have split it.
    if (!enc) {
      const head = b.subarray(0, PRESCAN_BYTES);
      let end = head.length;
      for (let i = head.length - 1; i >= 0 && i > head.length - 4; i--) {
        if (head[i] < 0x80) break;
        if ((head[i] & 0xc0) !== 0x80) {
          end = i;
          break;
        }
      }
      const prefix = head.subarray(0, end);
      enc = prefix.some((x) => x >= 0x80) && isUtf8(prefix) ? "utf-8" : "windows-1252";
    }
  }
  return decode(b, enc);
}
