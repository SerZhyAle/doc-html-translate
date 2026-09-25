// rtf.js - RTF reader. Ports internal/rtf (parse.go, codepage.go): a single forward pass with
// RTF's group state - a destination per group so font tables, metadata and pictures never reach
// the text, the \ucN fallback count, signed \uN values, \binN skipping, control symbols - and
// \'XX bytes decoded in the code page the document (\ansicpg) and its fonts (\fcharset) declare.
// The tables and the rules must match the Go side (docs/PARITY.md, "RTF text decoding";
// tests/parity_test.go pins the tables).

import { paragraphsToBook } from "./txt.js";

// Windows-1252 applies when the document names no code page, or one no browser can decode.
const DEFAULT_CODE_PAGE = 1252;

// Windows code page -> WHATWG label (TextDecoder). Mirrors codePageLabels.
const CODE_PAGE_LABELS = {
  866: "ibm866",
  874: "windows-874",
  932: "shift_jis",
  936: "gbk",
  949: "euc-kr",
  950: "big5",
  1250: "windows-1250",
  1251: "windows-1251",
  1252: "windows-1252",
  1253: "windows-1253",
  1254: "windows-1254",
  1255: "windows-1255",
  1256: "windows-1256",
  1257: "windows-1257",
  1258: "windows-1258",
  10000: "macintosh",
  10007: "x-mac-cyrillic",
  20866: "koi8-r",
  21866: "koi8-u",
  28592: "iso-8859-2",
  28595: "iso-8859-5",
  28597: "iso-8859-7",
  28605: "iso-8859-15",
  54936: "gb18030",
};

// \fcharsetN -> code page. ANSI (0) and DEFAULT (1) are absent on purpose: they mean "the
// document's \ansicpg". Mirrors charsetCodePages.
const CHARSET_CODE_PAGES = {
  77: 10000,
  128: 932,
  129: 949,
  134: 936,
  136: 950,
  161: 1253,
  162: 1254,
  163: 1258,
  177: 1255,
  178: 1256,
  186: 1257,
  204: 1251,
  222: 874,
  238: 1250,
};

// Groups that are never body text; every {\*\..} group is skipped as well. Mirrors
// skippedDestinations.
const SKIPPED_DESTINATIONS = new Set([
  "annotation",
  "atnauthor",
  "atnid",
  "bkmkend",
  "bkmkstart",
  "colorschememapping",
  "colortbl",
  "datastore",
  "docvar",
  "falt",
  "filetbl",
  "fldinst",
  "fontemb",
  "fontfile",
  "footer",
  "footerf",
  "footerl",
  "footerr",
  "ftncn",
  "ftnsep",
  "ftnsepc",
  "generator",
  "header",
  "headerf",
  "headerl",
  "headerr",
  "info",
  "latentstyles",
  "listoverridetable",
  "listpicture",
  "listtable",
  "nonshppict",
  "object",
  "objdata",
  "panose",
  "pgdsctbl",
  "pict",
  "private",
  "revtbl",
  "rsidtbl",
  "stylesheet",
  "tc",
  "template",
  "themedata",
  "userprops",
  "xe",
  "xmlnsdecl",
]);

// Control words that stand for one character. Mirrors symbolWords.
const SYMBOL_WORDS = {
  bullet: 0x2022,
  emdash: 0x2014,
  emspace: 0x2003,
  endash: 0x2013,
  enspace: 0x2002,
  ldblquote: 0x201c,
  lquote: 0x2018,
  ltrmark: 0x200e,
  qmspace: 0x2005,
  rdblquote: 0x201d,
  rquote: 0x2019,
  rtlmark: 0x200f,
  zwj: 0x200d,
  zwnj: 0x200c,
};

// Control words that end a paragraph. Mirrors breakWords.
const BREAK_WORDS = new Set(["line", "page", "par", "row", "sect"]);

const DEST_BODY = 0;
const DEST_SKIP = 1;
const DEST_FONT_TABLE = 2;

// A parameter is a signed 16-bit value in the spec; longer digit runs are garbage.
const MAX_PARAM_DIGITS = 10;

const isLetter = (c) => (c >= 0x61 && c <= 0x7a) || (c >= 0x41 && c <= 0x5a);
const isDigit = (c) => c >= 0x30 && c <= 0x39;

function hexVal(c) {
  if (c >= 0x30 && c <= 0x39) return c - 0x30;
  if (c >= 0x61 && c <= 0x66) return c - 0x61 + 10;
  if (c >= 0x41 && c <= 0x46) return c - 0x41 + 10;
  return -1;
}

// stripRtf turns RTF bytes into plain text; paragraph breaks come out as blank lines. Pure
// (TextDecoder only, no DOM) - unit-tested.
export function stripRtf(bytes) {
  const input = bytes instanceof Uint8Array ? bytes : new Uint8Array(bytes);
  const out = [];
  let st = { dest: DEST_BODY, uc: 1, font: 0 };
  const stack = [];
  let pos = 0;
  let ansiCP = DEFAULT_CODE_PAGE;
  let defFont = 0;
  const fontCP = new Map();
  let definingFont = 0;
  const decoders = new Map();
  let pending = [];
  let pendingCP = 0;
  let skip = 0;
  let high = 0;

  // One decoder per code page per document - never one per byte.
  const decoderFor = (cp) => {
    let d = decoders.get(cp);
    if (!d) {
      try {
        d = new TextDecoder(CODE_PAGE_LABELS[cp] || CODE_PAGE_LABELS[DEFAULT_CODE_PAGE]);
      } catch {
        d = new TextDecoder(CODE_PAGE_LABELS[DEFAULT_CODE_PAGE]);
      }
      decoders.set(cp, d);
    }
    return d;
  };
  const resolveHigh = () => {
    if (high) { out.push("\u{FFFD}"); high = 0; }
  };
  const flush = () => {
    if (!pending.length) return;
    resolveHigh();
    out.push(decoderFor(pendingCP).decode(Uint8Array.from(pending)));
    pending = [];
  };
  const codePage = () => (fontCP.has(st.font) ? fontCP.get(st.font) : ansiCP);
  // Consecutive code-page bytes are decoded together so a double-byte code page sees both
  // halves of a character.
  const addByte = (b) => {
    const cp = codePage();
    if (pending.length && cp !== pendingCP) flush();
    pending.push(b);
    pendingCP = cp;
  };
  const emit = (s) => {
    if (st.dest !== DEST_BODY) return;
    flush();
    resolveHigh();
    out.push(s);
  };

  // \uN is a signed 16-bit number; characters beyond the BMP arrive as a surrogate pair.
  const unicode = (n) => {
    if (st.dest !== DEST_BODY) return;
    if (n < 0) n += 0x10000;
    flush();
    if (n >= 0xd800 && n <= 0xdbff) {
      resolveHigh();
      high = n;
      return;
    }
    if (n >= 0xdc00 && n <= 0xdfff) {
      if (high) {
        out.push(String.fromCodePoint(0x10000 + ((high - 0xd800) << 10) + (n - 0xdc00)));
        high = 0;
      } else {
        out.push("\u{FFFD}");
      }
      return;
    }
    resolveHigh();
    if (n > 0 && n <= 0x10ffff) out.push(String.fromCodePoint(n));
  };

  const readParam = () => {
    let neg = false;
    if (pos + 1 < input.length && input[pos] === 0x2d && isDigit(input[pos + 1])) {
      neg = true;
      pos++;
    }
    const start = pos;
    let n = 0;
    while (pos < input.length && isDigit(input[pos])) {
      if (pos - start < MAX_PARAM_DIGITS) n = n * 10 + (input[pos] - 0x30);
      pos++;
    }
    if (pos === start) return null;
    return neg ? -n : n;
  };

  const word = (w, param) => {
    const has = param !== null;
    if (st.dest === DEST_FONT_TABLE) {
      if (w === "f") { definingFont = param ?? 0; return; }
      if (w === "fcharset") {
        if (CHARSET_CODE_PAGES[param] !== undefined) fontCP.set(definingFont, CHARSET_CODE_PAGES[param]);
        return;
      }
      if (w === "cpg") { fontCP.set(definingFont, param ?? 0); return; }
    }
    if (w === "fonttbl") st.dest = DEST_FONT_TABLE;
    else if (SKIPPED_DESTINATIONS.has(w)) st.dest = DEST_SKIP;
    else if (w === "ansicpg" && has) ansiCP = param;
    else if (w === "mac") ansiCP = 10000;
    else if (w === "deff" && has) { defFont = param; st.font = param; }
    else if (w === "f" && has) st.font = param;
    else if (w === "plain") st.font = defFont;
    else if (w === "uc" && has && param >= 0) st.uc = param;
    else if (w === "u" && has) { unicode(param); skip = st.uc; }
    else if (BREAK_WORDS.has(w)) emit("\n\n");
    else if (w === "tab" || w === "cell" || w === "nestcell") emit("\t");
    else if (SYMBOL_WORDS[w] !== undefined) emit(String.fromCodePoint(SYMBOL_WORDS[w]));
  };

  const symbol = (c) => {
    if (c === 0x27) { // \'XX
      if (pos + 2 > input.length) { pos = input.length; return; }
      const hi = hexVal(input[pos]);
      const lo = hexVal(input[pos + 1]);
      pos += 2;
      if (hi < 0 || lo < 0) return;
      if (skip > 0) { skip--; return; }
      if (st.dest === DEST_BODY) addByte((hi << 4) | lo);
      return;
    }
    if (skip > 0) { skip--; return; }
    switch (c) {
      case 0x2a: st.dest = DEST_SKIP; break; // \*
      case 0x7e: emit("\u{A0}"); break; // \~
      case 0x5f: emit("\u{2011}"); break; // \_
      case 0x7b: case 0x7d: case 0x5c: emit(String.fromCharCode(c)); break;
      case 0x0d: case 0x0a: emit("\n\n"); break;
      default: break; // \- optional hyphen, \| and \: are invisible
    }
  };

  const control = () => {
    if (pos >= input.length) return;
    const c = input[pos];
    if (!isLetter(c)) {
      pos++;
      symbol(c);
      return;
    }
    const start = pos;
    while (pos < input.length && isLetter(input[pos])) pos++;
    let w = "";
    for (let i = start; i < pos; i++) w += String.fromCharCode(input[i]);
    const param = readParam();
    if (pos < input.length && input[pos] === 0x20) pos++;
    if (w === "bin") {
      // Raw binary may contain braces and backslashes: skip it by count.
      flush();
      if (param !== null && param > 0) pos = Math.min(pos + param, input.length);
      return;
    }
    if (skip > 0) { skip--; return; }
    word(w, param);
  };

  while (pos < input.length) {
    const c = input[pos++];
    if (c === 0x7b) {
      flush();
      skip = 0;
      stack.push({ ...st });
    } else if (c === 0x7d) {
      flush();
      skip = 0;
      if (stack.length) st = stack.pop();
    } else if (c === 0x5c) {
      control();
    } else if (c === 0x0d || c === 0x0a) {
      // Line breaks in RTF source are formatting of the file, not of the text.
    } else if (skip > 0) {
      skip--;
    } else if (st.dest === DEST_BODY) {
      if (c >= 0x80) addByte(c);
      else {
        flush();
        resolveHigh();
        out.push(String.fromCharCode(c));
      }
    }
  }
  flush();
  resolveHigh();
  return out.join("");
}

// splitRtfParagraphs groups consecutive non-blank lines into paragraphs, matching
// the Go rtf splitParagraphs (blank-line separated, joined with a space).
function splitRtfParagraphs(text) {
  const norm = String(text).replace(/\r\n?/g, "\n");
  const paras = [];
  let cur = "";
  for (const line of norm.split("\n")) {
    const t = line.trim();
    if (t === "") {
      if (cur) { paras.push(cur); cur = ""; }
    } else {
      cur = cur ? `${cur} ${t}` : t;
    }
  }
  if (cur) paras.push(cur);
  return paras;
}

// parseRtf strips the RTF and returns the render-ready book shape.
export async function parseRtf(data) {
  const text = stripRtf(data);
  return paragraphsToBook(splitRtfParagraphs(text), "rtf-sec");
}
