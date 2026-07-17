// txt.js - plain-text reader. Splits text into paragraphs and paginates them into
// sections. Mirrors internal/txt/extract.go: normalize line endings; if the text
// has blank lines use them as paragraph separators (consecutive non-blank lines
// join with a space), otherwise treat each non-empty line as its own paragraph;
// 30 paragraphs per page.

const PARAS_PER_SECTION = 30;

// splitParagraphs turns raw text into paragraphs. Pure (no DOM) - unit-tested.
export function splitParagraphs(text) {
  const norm = String(text).replace(/\r\n?/g, "\n");
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
// leaves the bytes as UTF-8. Mirrors detectLegacy in internal/txt/legacy.go.
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

// decodeText turns a text file's bytes into a string, honouring the encoding those bytes
// declare. Mirrors internal/txt/extract.go decodeText - keep the two in step (docs/PARITY.md).
//
// Decoding everything as UTF-8 turned a Notepad "Unicode" save into mojibake: UTF-16 is two
// bytes per character, so the decoder saw a NUL between every letter and a Cyrillic file came
// out as replacement characters. A pre-Unicode Cyrillic code page (cp1251, koi8-r, cp866,
// iso-8859-5) came out equally wrong. Order: BOM first, then valid UTF-8 as-is, then a legacy
// code page by detection, else UTF-8.
//
// One place the two editions genuinely differ: TextDecoder strips a leading BOM on its own
// (ignoreBOM defaults to false), for utf-8 and utf-16 alike, so this side never had the Go
// side's UTF-8-BOM leak. Exported for the unit test.
export function decodeText(data) {
  const b = new Uint8Array(data);
  if (b.length >= 2) {
    if (b[0] === 0xff && b[1] === 0xfe) return new TextDecoder("utf-16le").decode(data);
    if (b[0] === 0xfe && b[1] === 0xff) return new TextDecoder("utf-16be").decode(data);
  }
  // A fatal utf-8 decode throws on the first invalid sequence, which is the cheap
  // equivalent of Go's utf8.Valid gate: if it succeeds, the bytes were UTF-8.
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(data);
  } catch {
    return detectLegacy(data) ?? new TextDecoder("utf-8").decode(data);
  }
}

// parseText decodes the bytes and returns the render-ready book shape.
export async function parseText(data) {
  return paragraphsToBook(splitParagraphs(decodeText(data)), "txt-sec");
}
