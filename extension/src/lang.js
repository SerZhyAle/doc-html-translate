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

// detectLang returns a BCP-47 primary language subtag for a text sample, or "" when the
// sample carries nothing to speak from: an image-only scan states no language, and an
// invented "en" would be exactly the false label a wrong <html lang> produces (ticket 76).
export function detectLang(text) {
  if (!text || !text.trim()) return "";
  const sample = text.slice(0, SAMPLE_CHARS);
  const { script } = dominantScript(sample);
  switch (script) {
    case "cyrillic":
      // Ukrainian-only letters distinguish uk from ru. і is not one of them: pre-1918
      // Russian orthography used it constantly, so one і does not make Cyrillic Ukrainian.
      return /[їєґЇЄҐ]/u.test(sample) ? "uk" : "ru";
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
    default: return "";
  }
}

// How far into the text the heuristics read. detectLang and declarationContradicted look
// at the same first SAMPLE_CHARS characters, and internal/textutil reads the same window.
const SAMPLE_CHARS = 8000;

// The script(s) each known primary subtag is written in, over the blocks dominantScript
// counts. A subtag absent from the table (bn, ta, ka..) says nothing provable, so the
// guard below never drops it. internal/textutil DeclarationContradicted keeps the same
// table on the desktop.
const DECLARED_SCRIPTS = {
  en: ["latin"], fr: ["latin"], de: ["latin"], es: ["latin"], it: ["latin"],
  pt: ["latin"], nl: ["latin"],
  ru: ["cyrillic"], uk: ["cyrillic"], bg: ["cyrillic"], sr: ["cyrillic"],
  mk: ["cyrillic"], be: ["cyrillic"],
  el: ["greek"],
  ar: ["arabic"], ur: ["arabic"], fa: ["arabic"], ps: ["arabic"],
  he: ["hebrew"], yi: ["hebrew"],
  zh: ["han"],
  ja: ["han", "hiragana", "katakana"],
  ko: ["hangul", "han"],
  hi: ["devanagari"], mr: ["devanagari"], ne: ["devanagari"],
  th: ["thai"],
};

// declarationContradicted says whether a text sample proves a declared language wrong
// (ticket 76): the sample's dominant script is known and is not a script the language is
// written in - the authoring template's "en-GB" over a Han sample. Without a sample, or
// for a language the table cannot speak for, nothing is provable and the declaration
// stands. Shared with the desktop: internal/textutil DeclarationContradicted must decide
// the same cases (tests/testdata/pdf_lang_cases.json).
export function declarationContradicted(tag, text) {
  const expected = DECLARED_SCRIPTS[normalizeLangTag(tag).split("-")[0]];
  if (!expected || !text || !text.trim()) return false;
  const { script, letters } = dominantScript(text.slice(0, SAMPLE_CHARS));
  if (letters === 0 || script === "unknown") return false;
  return !expected.includes(script);
}

// normalizeLangTag keeps only a sane BCP-47 primary subtag (and optional region)
// from a PDF /Lang value like "en-US" or "EN". The lookahead stops a longer word from
// passing as a tag ("russian" is not "rus", "zh-Hans" is "zh"); internal/textutil
// NormalizeLangTag applies the same rule on the desktop.
export function normalizeLangTag(tag) {
  if (typeof tag !== "string") return "";
  const m = tag.trim().match(/^([A-Za-z]{2,3})(?:[-_]([A-Za-z]{2}))?(?![A-Za-z0-9])/);
  if (!m) return "";
  return m[2] ? `${m[1].toLowerCase()}-${m[2].toUpperCase()}` : m[1].toLowerCase();
}
