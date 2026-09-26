// txt.js - plain-text reader. Splits text into paragraphs and paginates them into
// sections. Mirrors internal/txt/extract.go: normalize line endings; if the text
// has blank lines use them as paragraph separators (consecutive non-blank lines
// join with a space), otherwise treat each non-empty line as its own paragraph;
// 30 paragraphs per page.

const PARAS_PER_SECTION = 30;

// splitParagraphs turns raw text into paragraphs. Every separator the desktop edition's
// textutil.NormalizeLineSeparators knows is a line break here too (NEL, LS, PS, vertical tab, form
// feed), or the same file splits into different paragraphs - the shared fixture
// tests/testdata/txt_paragraph_cases.json holds both to it. Pure (no DOM) - unit-tested.
export function splitParagraphs(text) {
  const norm = String(text).replace(/\r\n?|[\u0085\u2028\u2029\v\f]/g, "\n");
  if (norm.includes("\n\n")) {
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
  return norm.split("\n").map((l) => l.trim()).filter((l) => l);
}

// paragraphsToBook chunks paragraphs into sections of PARAS_PER_SECTION and builds
// each as <p> nodes via textContent (no HTML injection). Shared by txt and rtf.
export function paragraphsToBook(paragraphs, idPrefix) {
  const sections = [];
  for (let i = 0; i < paragraphs.length; i += PARAS_PER_SECTION) {
    const chunk = paragraphs.slice(i, i + PARAS_PER_SECTION);
    const frag = document.createDocumentFragment();
    for (const p of chunk) {
      const node = document.createElement("p");
      node.textContent = p;
      frag.appendChild(node);
    }
    sections.push({ id: `${idPrefix}-${sections.length}`, label: "", frag });
  }
  let sampleText = "";
  for (const p of paragraphs) {
    if (sampleText.length >= 8000) break;
    sampleText += ` ${p}`;
  }
  return { title: "", lang: "", sampleText, sections, toc: [], revoke: () => {} };
}

// Relative frequency (percent) of each lowercase Russian letter. Mirrors ruLetterFreq in
// internal/txt/legacy.go - the two MUST stay identical, or the same file decodes to one code
// page here and another there (docs/PARITY.md).
const RU_LETTER_FREQ = {
  о: 10.98, е: 8.45, а: 8.01, и: 7.35, н: 6.7, т: 6.26,
  с: 5.47, р: 4.73, в: 4.54, л: 4.4, к: 3.49, м: 3.21,
  д: 2.98, п: 2.81, у: 2.62, я: 2.01, ы: 1.9, ь: 1.74,
  г: 1.7, з: 1.65, б: 1.59, ч: 1.44, й: 1.21, х: 0.97,
  ж: 0.94, ш: 0.73, ю: 0.64, ц: 0.48, щ: 0.36, э: 0.32,
  ф: 0.26, ъ: 0.04, ё: 0.04,
};

// The candidate code pages, most-likely-first, with their WHATWG TextDecoder labels. CP866 is
// "ibm866". Order and set mirror legacyCandidates in internal/txt/legacy.go.
const LEGACY_CANDIDATES = ["windows-1251", "koi8-r", "ibm866", "iso-8859-5"];

// Confidence floor: Russian letters as a share of all characters. Mirrors minCyrillicFraction
// in legacy.go. Measured: real cp1251 sits at 0.76, French Latin-1 mis-read as KOI8-R at 0.17.
const MIN_CYRILLIC_FRACTION = 0.3;

// cyrillicFit scores a decoded string: freqWeight (summed expected frequency of its Russian
// letters) picks the encoding - a wrong code page yields as many Cyrillic letters but the
// wrong, rarer ones - and fraction (Russian letters over all characters) is the confidence
// that the text is Russian at all. Mirrors cyrillicFit in legacy.go.
function cyrillicFit(s) {
  let weight = 0;
  let letters = 0;
  for (const ch of s) {
    const f = RU_LETTER_FREQ[ch.toLowerCase()];
    if (f !== undefined) {
      weight += f;
      letters += 1;
    }
  }
  const runes = [...s].length;
  return { weight, fraction: runes > 0 ? letters / runes : 0 };
}

// detectLegacy decodes non-UTF-8, BOM-less bytes as the most Russian-looking candidate code
// page, committing only when the result is confidently Cyrillic; otherwise null, so the caller
// falls back to the Western code page. Mirrors detectLegacy in internal/txt/legacy.go.
function detectLegacy(data) {
  let best = null;
  for (const label of LEGACY_CANDIDATES) {
    let decoded;
    try {
      decoded = new TextDecoder(label, { fatal: false }).decode(data);
    } catch {
      continue;
    }
    const { weight, fraction } = cyrillicFit(decoded);
    if (best === null || weight > best.weight) best = { weight, fraction, decoded };
  }
  if (best === null || best.fraction < MIN_CYRILLIC_FRACTION) return null;
  return best.decoded;
}

// The Western default code page: the last rung of the ladder. Mirrors westernLabel in legacy.go.
const WESTERN_LABEL = "windows-1252";

// The BOM-less UTF-16 check reads only the head of the file. Mirrors utf16SniffBytes.
const UTF16_SNIFF_BYTES = 4096;

// sniffUtf16 recognizes UTF-16 saved without a byte-order mark by where its NUL bytes sit: the
// high byte of every Latin letter, digit, space and punctuation mark is zero, so NULs pile up
// on one parity (odd offsets for little-endian) and real 8-bit text has none. The dominant
// parity needs at least 2 NULs covering 5% of the code units, the other at most a tenth as
// many. Returns the TextDecoder label, or null. Mirrors sniffUTF16 in internal/txt/decode.go.
export function sniffUtf16(b) {
  const n = Math.min(b.length, UTF16_SNIFF_BYTES) & ~1;
  const units = n / 2;
  if (units < 2) return null;
  let even = 0;
  let odd = 0;
  for (let i = 0; i < n; i += 2) {
    if (b[i] === 0) even++;
    if (b[i + 1] === 0) odd++;
  }
  const dominates = (hi, lo) => hi >= 2 && hi * 20 >= units && lo * 10 <= hi;
  if (dominates(odd, even)) return "utf-16le";
  if (dominates(even, odd)) return "utf-16be";
  return null;
}

// measureUtf8 counts UTF-8 damage with the WHATWG decoder algorithm - the one TextDecoder
// runs - so it agrees with the replacement characters the decode then emits: well-formed
// multi-byte sequences, bytes that become U+FFFD, the number of U+FFFD, and whether the data
// ends inside an otherwise well-formed sequence. Mirrors textutil.MeasureUTF8.
export function measureUtf8(b) {
  let multi = 0;
  let invalidBytes = 0;
  let errors = 0;
  let needed = 0;
  let seen = 0;
  let lower = 0x80;
  let upper = 0xbf;
  const bad = (size) => { errors++; invalidBytes += size; };
  for (let i = 0; i < b.length; ) {
    const c = b[i];
    if (needed === 0) {
      i++;
      if (c < 0x80) continue;
      if (c >= 0xc2 && c <= 0xdf) needed = 1;
      else if (c >= 0xe0 && c <= 0xef) {
        if (c === 0xe0) lower = 0xa0;
        else if (c === 0xed) upper = 0x9f;
        needed = 2;
      } else if (c >= 0xf0 && c <= 0xf4) {
        if (c === 0xf0) lower = 0x90;
        else if (c === 0xf4) upper = 0x8f;
        needed = 3;
      } else bad(1);
      continue;
    }
    if (c < lower || c > upper) {
      bad(seen + 1);
      needed = 0; seen = 0; lower = 0x80; upper = 0xbf;
      continue;
    }
    i++;
    lower = 0x80; upper = 0xbf;
    seen++;
    if (seen === needed) {
      multi++;
      needed = 0; seen = 0;
    }
  }
  const truncatedTail = needed > 0;
  if (truncatedTail) bad(seen + 1);
  return { multi, invalidBytes, errors, truncatedTail };
}

// acceptAsUtf8: bytes that are not strictly valid UTF-8 are still UTF-8 with a little damage
// when they hold at least one valid multi-byte sequence and the invalid bytes are under 1% of
// those sequences, or when the only invalidity is a sequence cut off by the end of the file
// (ticket 10, section 6.1). Mirrors acceptAsUTF8 in internal/txt/decode.go.
export function acceptAsUtf8(b) {
  const d = measureUtf8(b);
  if (d.errors === 0) return true;
  if (d.errors === 1 && d.truncatedTail) return true;
  return d.multi > 0 && d.invalidBytes * 100 < d.multi;
}

// decodeText turns a text file's bytes into a string, honouring the encoding those bytes
// declare. Mirrors internal/txt/extract.go decodeText - keep the two in step (docs/PARITY.md).
//
// The ladder: a BOM is authoritative; then BOM-less UTF-16 by its NUL pattern; then UTF-8,
// tolerating a little damage; then a Cyrillic legacy code page by detection; then
// windows-1252. Damage shows as U+FFFD, never dropped: one bad byte in a Russian UTF-8 book
// used to send the whole file to the legacy detector and turn it into mojibake.
//
// TextDecoder strips a leading BOM on its own (ignoreBOM defaults to false), for utf-8 and
// utf-16 alike, so the BOM branches only choose the decoder. Exported for the unit test.
export function decodeText(data) {
  const b = new Uint8Array(data);
  if (b.length >= 3 && b[0] === 0xef && b[1] === 0xbb && b[2] === 0xbf) return new TextDecoder("utf-8").decode(b);
  if (b.length >= 2) {
    if (b[0] === 0xff && b[1] === 0xfe) return new TextDecoder("utf-16le").decode(b);
    if (b[0] === 0xfe && b[1] === 0xff) return new TextDecoder("utf-16be").decode(b);
  }
  const utf16 = sniffUtf16(b);
  if (utf16) return new TextDecoder(utf16, { ignoreBOM: true }).decode(b);
  // A fatal decode is the fast path for the common, fully valid file.
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(b);
  } catch {
    // Not strictly valid: measure the damage below.
  }
  if (acceptAsUtf8(b)) return new TextDecoder("utf-8").decode(b);
  return detectLegacy(b) ?? new TextDecoder(WESTERN_LABEL).decode(b);
}

// parseText decodes the bytes and returns the render-ready book shape.
export async function parseText(data) {
  return paragraphsToBook(splitParagraphs(decodeText(data)), "txt-sec");
}
