// lang.js - guess the source language of the extracted text so the viewer can set
// <html lang>. Chrome only offers "Translate page" when the page language differs
// from the UI language, and a correct source language makes the translation
// accurate, so this drives the whole free-translate flow (spec sec 4).
//
// Strategy: non-Latin scripts are identified by Unicode block (cheap and reliable);
// Latin text is disambiguated by a small stop-word vote. Pure function, testable.

// Count characters by script over a sample. Returns the dominant non-Latin script
// code or "latin"/"unknown".
function dominantScript(text) {
  const counts = {
    latin: 0, cyrillic: 0, greek: 0, arabic: 0, hebrew: 0,
    han: 0, hiragana: 0, katakana: 0, hangul: 0, devanagari: 0, thai: 0,
  };
  let letters = 0;
  for (const ch of text) {
    const c = ch.codePointAt(0);
    if (c >= 0x41 && c <= 0x7a && /[A-Za-z]/.test(ch)) counts.latin++;
    else if (c >= 0x0400 && c <= 0x04ff) counts.cyrillic++;
    else if (c >= 0x0370 && c <= 0x03ff) counts.greek++;
    else if (c >= 0x0600 && c <= 0x06ff) counts.arabic++;
    else if (c >= 0x0590 && c <= 0x05ff) counts.hebrew++;
    else if (c >= 0x4e00 && c <= 0x9fff) counts.han++;
    else if (c >= 0x3040 && c <= 0x309f) counts.hiragana++;
    else if (c >= 0x30a0 && c <= 0x30ff) counts.katakana++;
    else if (c >= 0xac00 && c <= 0xd7a3) counts.hangul++;
    else if (c >= 0x0900 && c <= 0x097f) counts.devanagari++;
    else if (c >= 0x0e00 && c <= 0x0e7f) counts.thai++;
    else continue;
    letters++;
  }
  if (letters === 0) return { script: "unknown", counts, letters };
  let script = "latin";
  let max = -1;
  for (const [k, v] of Object.entries(counts)) {
    if (v > max) { max = v; script = k; }
  }
  return { script, counts, letters };
}

const LATIN_STOPWORDS = {
  en: ["the", "and", "of", "to", "in", "is", "that", "for", "with", "was", "it", "as", "this"],
  fr: ["le", "la", "les", "des", "et", "une", "que", "dans", "pour", "est", "qui", "pas", "plus"],
  de: ["der", "die", "und", "das", "ist", "den", "von", "mit", "nicht", "ein", "auch", "auf", "eine"],
  es: ["el", "la", "los", "las", "que", "de", "una", "para", "con", "por", "como", "más", "pero"],
  it: ["il", "la", "che", "di", "una", "per", "con", "del", "non", "come", "sono", "anche", "delle"],
  pt: ["de", "que", "os", "as", "uma", "para", "com", "não", "por", "mais", "como", "dos", "uma"],
  nl: ["de", "het", "een", "van", "en", "dat", "die", "niet", "met", "voor", "aan", "op", "te"],
};

function voteLatin(text) {
  const words = text.toLowerCase().match(/[a-zà-ÿ]+/giu) || [];
  if (words.length === 0) return "en";
  const sample = new Map();
  for (const w of words) sample.set(w, (sample.get(w) || 0) + 1);
  let best = "en";
  let bestScore = -1;
  for (const [lang, stops] of Object.entries(LATIN_STOPWORDS)) {
    let score = 0;
    for (const s of stops) score += sample.get(s) || 0;
    if (score > bestScore) { bestScore = score; best = lang; }
  }
  return best;
}

// detectLang returns a BCP-47 primary language subtag for a text sample.
export function detectLang(text) {
  if (!text || !text.trim()) return "en";
  const sample = text.slice(0, 8000);
  const { script } = dominantScript(sample);
  switch (script) {
    case "cyrillic":
      // Ukrainian-specific letters distinguish uk from ru.
      return /[іїєґІЇЄҐ]/u.test(sample) ? "uk" : "ru";
    case "greek": return "el";
    case "arabic": return "ar";
    case "hebrew": return "he";
    case "hiragana":
    case "katakana": return "ja";
    case "hangul": return "ko";
    case "han": return "zh";
    case "devanagari": return "hi";
    case "thai": return "th";
    case "latin": return voteLatin(sample);
    default: return "en";
  }
}

// normalizeLangTag keeps only a sane BCP-47 primary subtag (and optional region)
// from a PDF /Lang value like "en-US" or "EN".
export function normalizeLangTag(tag) {
  if (typeof tag !== "string") return "";
  const m = tag.trim().match(/^([A-Za-z]{2,3})(?:[-_]([A-Za-z]{2}))?/);
  if (!m) return "";
  return m[2] ? `${m[1].toLowerCase()}-${m[2].toUpperCase()}` : m[1].toLowerCase();
}
